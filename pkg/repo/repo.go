// package repo is a repository where the data models can be stored to
// and loaded from.
package repo

import "context"
import "encoding/json"

import nsrv "github.com/nats-io/nats-server/v2/server"
import nats "github.com/nats-io/nats.go"
import "github.com/nats-io/nats.go/jetstream"

type Context = context.Context

type Repository struct {
	*nsrv.Server
	*nats.Conn
	jetstream.JetStream
}

func Open(sd string) (r *Repository, err error) {
	opts := nsrv.Options{
		JetStream: true,
		StoreDir:  sd,
	}
	r = &Repository{}
	r.Server, err = nsrv.NewServer(&opts)
	if err != nil {
		return nil, err
	}
	r.Conn, err = nats.Connect("", nats.InProcessServer(r.Server))
	if err != nil {
		return nil, err
	}

	r.JetStream, err = jetstream.New(r.Conn)
	if err != nil {
		return nil, err
	}
	return r, nil
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

func NewBasicMapper[T any](ctx Context, r *Repository, name string) (*BasicMapper[T], error) {
	var err error

	bm := &BasicMapper[T]{
		Repository: r,
		Name:       name,
	}
	bm.KeyValue, err = bm.Repository.JetStream.KeyValue(ctx, bm.Name)
	if err != nil {
		return nil, err
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
	return nil
}

func (b *BasicMapper[T]) Watch(ctx Context, pattern string) (chan (T), error) {
	return nil, nil
}

func (b *BasicMapper[T]) All(ctx Context, pattern string) (chan (T), error) {
	return nil, nil
}

func (b *BasicMapper[T]) Keys(ctx Context, pattern string) (chan (string), error) {
	return nil, nil
}
