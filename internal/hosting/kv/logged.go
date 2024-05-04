package kv

import (
	"context"
	"log"
)

var _ Client = &Logged{}

type Logged struct {
	c Client
}

func WithLogger(c Client) *Logged {
	return &Logged{c: c}
}

func (l *Logged) Bucket(ctx context.Context, name string) (Bucket, error) {
	b, err := l.c.Bucket(ctx, name)
	if err != nil {
		return nil, err
	}

	return &LoggedBucket{
		b: b,
	}, nil
}

var _ Bucket = &LoggedBucket{}

type LoggedBucket struct {
	b Bucket
}

func (b *LoggedBucket) Name() string {
	return b.b.Name()
}

func (b *LoggedBucket) Get(ctx context.Context, key string) ([]byte, error) {
	return b.b.Get(ctx, key)
}

func (b *LoggedBucket) Set(ctx context.Context, key string, value []byte) error {
	log.Printf("Set %s/%s", b.Name(), key)
	return b.b.Set(ctx, key, value)
}

func (b *LoggedBucket) Delete(ctx context.Context, key string) error {
	log.Printf("Delete %s/%s", b.Name(), key)
	return b.b.Delete(ctx, key)
}

func (b *LoggedBucket) WatchAll(ctx context.Context) (Watcher, error) {
	w, err := b.b.WatchAll(ctx)
	if err != nil {
		log.Printf("WatchAll %s: %v", b.Name(), err)
		return nil, err
	}

	log.Printf("WatchAll %s", b.Name())

	return &LoggedWatcher{
		w: w,
	}, nil
}

func (b *LoggedBucket) Unwatch(w Watcher) {
	b.b.Unwatch(w)

	log.Printf("Unwatch %s", b.Name())
}

func (b *LoggedBucket) ListKeys(ctx context.Context) ([]string, error) {
	return b.b.ListKeys(ctx)
}

var _ Watcher = &LoggedWatcher{}

type LoggedWatcher struct {
	w Watcher
}

func (w *LoggedWatcher) Changes() <-chan *Value {
	return w.w.Changes()
}

func (w *LoggedWatcher) Unwatch() {
	log.Printf("Unwatch")

	w.w.Unwatch()
}
