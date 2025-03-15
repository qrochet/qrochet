// package repo is a repository where the data models can be stored to
// and loaded from.
package repo

import "context"
import "net/url"
import "encoding/json"

import nsrv "github.com/nats-io/nats-server/v2/server"
import nats "github.com/nats-io/nats.go"
import "github.com/nats-io/nats.go/jetstream"

import "github.com/qrochet/qrochet/pkg/model"

type Context = context.Context

type Repository struct {
	*nsrv.Server
	*nats.Conn
	jetstream.JetStream
	User    *BasicMapper[model.User]
	Session *BasicMapper[model.Session]
	Craft   *BasicMapper[model.Craft]
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
	r.User, err = NewBasicMapper[model.User](ctx, r, "user")
	if err != nil {
		return err
	}
	r.Session, err = NewBasicMapper[model.Session](ctx, r, "session")
	if err != nil {
		return err
	}
	r.Craft, err = NewBasicMapper[model.Craft](ctx, r, "craft")
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
		}
		close(ch)
	}()

	return ch, nil
}

func (b *BasicMapper[T]) All(ctx Context, keys ...string) (chan (T), error) {
	watcher, err := b.KeyValue.WatchFiltered(ctx, keys, jetstream.IgnoreDeletes())
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
		}
		close(ch)
	}()

	return ch, nil
}
