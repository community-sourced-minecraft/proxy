package kv

import (
	"context"
	"errors"
	"log"
	"slices"
	"sync"

	"github.com/nats-io/nats.go/jetstream"
)

var _ Client = &NATSClient{}

type NATSOptions struct {
	URL string `json:"url"`
}

type NATSClient struct {
	js jetstream.JetStream
}

func NewNATSClient(js jetstream.JetStream) *NATSClient {
	return &NATSClient{js: js}
}

func (n *NATSClient) Bucket(ctx context.Context, name string) (Bucket, error) {
	kv, err := n.js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: name})
	if errors.Is(err, jetstream.ErrBucketExists) {
		kv, err = n.js.KeyValue(ctx, name)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &NATSBucket{
		name:     name,
		kv:       kv,
		watchers: make([]*NATSWatcher, 0),
	}, nil
}

var _ Bucket = &NATSBucket{}

type NATSBucket struct {
	name     string
	kv       jetstream.KeyValue
	watchers []*NATSWatcher
	m        sync.RWMutex
}

func (b *NATSBucket) Name() string {
	return b.name
}

func (b *NATSBucket) Get(ctx context.Context, key string) ([]byte, error) {
	k, err := b.kv.Get(ctx, key)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return nil, ErrKeyNotFound
	} else if err != nil {
		return nil, err
	}

	return k.Value(), nil
}

func (b *NATSBucket) Set(ctx context.Context, key string, value []byte) error {
	_, err := b.kv.Put(ctx, key, value)
	if err != nil {
		return err
	}

	return nil
}

func (b *NATSBucket) Delete(ctx context.Context, key string) error {
	if err := b.kv.Delete(ctx, key); err != nil {
		return err
	}

	return nil
}

func (b *NATSBucket) ListKeys(ctx context.Context) ([]string, error) {
	keys := make([]string, 0)
	lister, err := b.kv.ListKeys(ctx)
	if err != nil {
		return nil, err
	}

	for k := range lister.Keys() {
		keys = append(keys, k)
	}

	return keys, nil
}

func (b *NATSBucket) WatchAll(ctx context.Context) (Watcher, error) {
	watcher, err := b.kv.WatchAll(ctx)
	if err != nil {
		return nil, err
	}

	w := &NATSWatcher{
		bucket:  b,
		w:       watcher,
		changes: make(chan *Value),
	}

	go func() {
		for msg := range watcher.Updates() {
			if msg == nil {
				w.m.Lock()
				log.Printf("sending nil change")
				w.changes <- nil
				w.m.Unlock()
				continue
			}

			var op Operation
			switch msg.Operation() {
			case jetstream.KeyValueDelete:
				op = Delete
			case jetstream.KeyValuePut:
				op = Put
			case jetstream.KeyValuePurge:
				continue
			}

			w.m.Lock()
			log.Printf("sending %s change for %s", op, msg.Key())
			w.changes <- &Value{
				Key:       msg.Key(),
				Value:     msg.Value(),
				Operation: op,
			}
			w.m.Unlock()
		}

		log.Printf("watcher stopped")
	}()

	b.m.Lock()
	b.watchers = append(b.watchers, w)
	b.m.Unlock()

	return w, nil
}

func (b *NATSBucket) Unwatch(w Watcher) {
	w_, ok := w.(*NATSWatcher)
	if !ok {
		return
	}

	b.m.Lock()
	b.watchers = slices.DeleteFunc(b.watchers, func(w2 *NATSWatcher) bool {
		return w_.w == w2.w
	})

	_ = w_.w.Stop()
	b.m.Unlock()
}

var _ Watcher = &NATSWatcher{}

type NATSWatcher struct {
	bucket  *NATSBucket
	w       jetstream.KeyWatcher
	changes chan *Value
	m       sync.Mutex
}

func (w *NATSWatcher) Changes() <-chan *Value {
	return w.changes
}

func (w *NATSWatcher) Unwatch() {
	w.m.Lock()
	defer w.m.Unlock()

	close(w.changes)
	w.bucket.Unwatch(w)
}
