// package repo is a repository where the data models can be stored to
// and loaded from.
package repo

import "io"
import "context"
import "net/url"
import "encoding/json"
import "log/slog"

import nsrv "github.com/nats-io/nats-server/v2/server"
import nats "github.com/nats-io/nats.go"
import "github.com/nats-io/nats.go/jetstream"

import "github.com/qrochet/qrochet/pkg/model"

type Context = context.Context

type Repository struct {
	*nsrv.Server
	*nats.Conn
	jetstream.JetStream
	User    *UserMapper
	Session *BasicMapper[model.Session]
	Craft   *CraftMapper
	Image   *UploadMapper
}

func Open(nurl string) (r *Repository, err error) {
	u, err := url.Parse(nurl)
	if err != nil {
		return nil, err
	}

	r = &Repository{}

	if u.Scheme == "nats+builtin" {
		opts := nsrv.Options{
			JetStream: true,
			StoreDir:  u.Path,
		}
		r.Server, err = nsrv.NewServer(&opts)
		if err != nil {
			return nil, err
		}
		r.Conn, err = nats.Connect("", nats.InProcessServer(r.Server))
		if err != nil {
			return nil, err
		}
	} else {
		r.Conn, err = nats.Connect(nurl)
		if err != nil {
			return nil, err
		}
	}

	r.JetStream, err = jetstream.New(r.Conn)
	if err != nil {
		return nil, err
	}

	err = r.Setup(context.TODO())
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Repository) Setup(ctx context.Context) error {
	var err error
	r.User, err = NewUserMapper(ctx, r, "user")
	if err != nil {
		return err
	}
	r.Session, err = NewBasicMapper[model.Session](ctx, r, "session")
	if err != nil {
		return err
	}
	r.Craft, err = NewCraftMapper(ctx, r, "craft")
	if err != nil {
		return err
	}

	r.Image, err = NewUploadMapper(ctx, r, "image")
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) Close() {
	r.Conn.Close()
	r.Server.Shutdown()
}

type BasicMapper[T any] struct {
	Name string
	*Repository
	jetstream.KeyValue
}

var MapperPrefix = "qro-"

func NewBasicMapper[T any](ctx Context, r *Repository, name string) (*BasicMapper[T], error) {
	var err error

	bm := &BasicMapper[T]{
		Repository: r,
		Name:       MapperPrefix + name,
	}
	bm.KeyValue, err = bm.Repository.JetStream.KeyValue(ctx, bm.Name)
	if err != nil {
		if err == jetstream.ErrBucketNotFound {
			kvc := jetstream.KeyValueConfig{Bucket: bm.Name}
			bm.KeyValue, err = bm.Repository.JetStream.CreateKeyValue(ctx, kvc)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return bm, nil
}

func (b *BasicMapper[T]) Get(ctx Context, key string) (T, error) {
	var obj T
	entry, err := b.KeyValue.Get(ctx, key)
	if err != nil {
		return obj, err
	}

	err = json.Unmarshal(entry.Value(), &obj)
	if err != nil {
		return obj, err
	}

	return obj, nil
}

func (b *BasicMapper[T]) Put(ctx Context, key string, obj T) (T, error) {
	var zero T

	buf, err := json.Marshal(obj)
	if err != nil {
		return zero, err
	}

	_, err = b.KeyValue.Put(ctx, key, buf)
	if err != nil {
		return zero, err
	}
	return obj, nil
}

func (b *BasicMapper[T]) Purge(ctx Context, key string) error {
	return b.KeyValue.Purge(ctx, key)
}

func (b *BasicMapper[T]) Keys(ctx Context, keys ...string) (chan (string), error) {
	lister, err := b.KeyValue.ListKeysFiltered(ctx, keys...)
	if err != nil {
		return nil, err
	}
	ch := make(chan (string))
	go func() {
		for res := range lister.Keys() {
			ch <- res
		}
		close(ch)
	}()

	return ch, nil
}

func (b *BasicMapper[T]) Watch(ctx Context, keys ...string) (chan (T), error) {
	watcher, err := b.KeyValue.WatchFiltered(ctx, keys,
		jetstream.UpdatesOnly(), jetstream.IgnoreDeletes())
	if err != nil {
		return nil, err
	}

	ch := make(chan (T))
	go func() {
		for res := range watcher.Updates() {
			var obj T
			if res == nil {
				watcher.Stop()
				break
			}
			err = json.Unmarshal(res.Value(), &obj)
			if err != nil {
				watcher.Stop()
				break
			}
			ch <- obj
		}
		close(ch)
	}()

	return ch, nil
}

func (b *BasicMapper[T]) All(ctx Context, keys ...string) (chan (T), error) {
	watcher, err := b.KeyValue.WatchFiltered(ctx, keys, jetstream.IgnoreDeletes())
	if err != nil {
		slog.Error("BasicMapper.All", "err", err)
		return nil, err
	}
	slog.Debug("BasicMapper.All", "keys", keys)

	ch := make(chan (T))
	go func() {
		for res := range watcher.Updates() {

			slog.Debug("BasicMapper.All Updates", "res", res)

			var obj T
			if res == nil {
				watcher.Stop()
				break
			}
			err = json.Unmarshal(res.Value(), &obj)
			if err != nil {
				watcher.Stop()
				break
			}
			ch <- obj
		}
		slog.Debug("BasicMapper.All closed")
		close(ch)
	}()

	return ch, nil
}

// Watches the mapper and returns the first value where the matcher returns true.
// Return nil if not found, or an error on error.
// As this is a simple linear scan it isn't very performant yet, but it will do as
// a first implementation.
func (b *BasicMapper[T]) GetFirstMatch(ctx Context, matcher func(t *T) bool) (*T, error) {
	watcher, err := b.KeyValue.WatchAll(ctx, jetstream.IgnoreDeletes())
	if err != nil {
		return nil, err
	}
	defer watcher.Stop()

	for res := range watcher.Updates() {
		var obj T
		if res == nil {
			return nil, nil
		}

		err = json.Unmarshal(res.Value(), &obj)
		if err != nil {
			return nil, err
		}

		if matcher(&obj) {
			return &obj, nil
		}
	}

	return nil, nil
}

type UploadMapper struct {
	Name string
	*Repository
	jetstream.ObjectStore
}

func NewUploadMapper(ctx Context, r *Repository, name string) (*UploadMapper, error) {
	var err error

	bm := &UploadMapper{
		Repository: r,
		Name:       MapperPrefix + name,
	}
	bm.ObjectStore, err = bm.Repository.JetStream.ObjectStore(ctx, bm.Name)
	if err != nil {
		if err == jetstream.ErrBucketNotFound {
			kvc := jetstream.ObjectStoreConfig{Bucket: bm.Name}
			bm.ObjectStore, err = bm.Repository.JetStream.CreateObjectStore(ctx, kvc)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return bm, nil
}

func infoToUpload(info *jetstream.ObjectInfo, rd io.ReadCloser) (*model.Upload, error) {
	var up model.Upload

	up.ID = model.Reference(info.Name)
	up.Detail = info.Description
	up.UserID = info.Metadata["user_id"]
	up.Title = info.Metadata["title"]

	up.ReadCloser = rd
	return &up, nil
}

func entryToUpload(entry jetstream.ObjectResult) (*model.Upload, error) {
	var up model.Upload
	info, err := entry.Info()
	if err != nil || info == nil {
		entry.Close()
		return nil, err
	}

	up.ReadCloser = entry
	up.ID = model.Reference(info.Name)
	up.Detail = info.Description
	up.UserID = info.Metadata["user_id"]
	up.Title = info.Metadata["title"]

	return &up, nil
}

func (b *UploadMapper) Get(ctx Context, key string) (*model.Upload, error) {
	entry, err := b.ObjectStore.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return entryToUpload(entry)
}

func (b *UploadMapper) Put(ctx Context, up *model.Upload) (*model.Upload, error) {
	var err error

	info := jetstream.ObjectMeta{}
	info.Name = string(up.ID)
	info.Description = up.Detail
	info.Metadata = map[string]string{
		"user_id": up.UserID,
		"title":   up.Title,
	}

	_, err = b.ObjectStore.Put(ctx, info, up.ReadCloser)
	if err != nil {
		return nil, err
	}
	return up, nil
}

func (b *UploadMapper) Delete(ctx Context, key string) error {
	return b.ObjectStore.Delete(ctx, key)
}

func (b *UploadMapper) List(ctx Context, userId string) (chan (string), error) {
	lister, err := b.ObjectStore.List(ctx)
	if err != nil {
		return nil, err
	}
	ch := make(chan (string))
	go func() {
		for _, info := range lister {
			if userId != "" && userId != info.Metadata["user_id"] {
				continue
			}
			ch <- info.Name
		}
		close(ch)
	}()

	return ch, nil
}

func (b *UploadMapper) Watch(ctx Context) (chan (*model.Upload), error) {
	watcher, err := b.ObjectStore.Watch(ctx,
		jetstream.UpdatesOnly(), jetstream.IgnoreDeletes())
	if err != nil {
		return nil, err
	}

	ch := make(chan (*model.Upload))
	go func() {
		for info := range watcher.Updates() {
			up, err := infoToUpload(info, nil)
			if err != nil {
				watcher.Stop()
				break
			}
			ch <- up
		}
		close(ch)
	}()

	return ch, nil
}

// CraftMapper is a mapper for cafts.
type CraftMapper struct {
	// Inherit from BasicMapper
	*BasicMapper[model.Craft]
}

func NewCraftMapper(ctx Context, r *Repository, name string) (*CraftMapper, error) {
	var err error
	res := &CraftMapper{}
	res.BasicMapper, err = NewBasicMapper[model.Craft](ctx, r, name)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *CraftMapper) Put(ctx Context, key string, craft model.Craft) (model.Craft, error) {
	key = craft.UserID + "." + key
	return c.BasicMapper.Put(ctx, key, craft)
}

func (c *CraftMapper) GetForUserID(ctx Context, key string, UserID string) (model.Craft, error) {
	key = UserID + "." + key
	return c.BasicMapper.Get(ctx, key)
}

func (c *CraftMapper) AllForUserID(ctx Context, UserID string) (chan model.Craft, error) {
	key := UserID + ".>"
	return c.BasicMapper.All(ctx, key)
}

// UserMapper is a mapper for cafts.
type UserMapper struct {
	// Inherit from BasicMapper
	*BasicMapper[model.User]
}

func NewUserMapper(ctx Context, r *Repository, name string) (*UserMapper, error) {
	var err error
	res := &UserMapper{}
	res.BasicMapper, err = NewBasicMapper[model.User](ctx, r, name)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *UserMapper) GetByEmail(ctx Context, email string) (*model.User, error) {
	return c.BasicMapper.GetFirstMatch(ctx, func(u *model.User) bool { return u.Email == email })
}
