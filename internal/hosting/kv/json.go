package kv

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"slices"
	"sync"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting/storage"
)

var _ Client = &JSONClient{}

type JSONClient struct {
	fileName string
	store    storage.Storage
	buckets  map[string]*JSONBucket
	m        sync.RWMutex
}

func NewJSONClient(store storage.Storage, fileName string) (*JSONClient, error) {
	c := &JSONClient{
		fileName: fileName,
		store:    store,
		buckets:  make(map[string]*JSONBucket),
	}

	if err := c.init(context.Background()); err != nil {
		return nil, err
	}

	return c, nil
}

func (j *JSONClient) init(ctx context.Context) error {
	reader, err := j.store.ReadStreaming(ctx, j.fileName)
	if err != nil {
		if errors.Is(err, storage.ErrKeyNotFound) {
			return j.save(ctx)
		}

		return err
	}

	j.m.Lock()
	defer j.m.Unlock()

	j.buckets = make(map[string]*JSONBucket)

	if err := json.NewDecoder(reader).Decode(&j.buckets); err != nil {
		return err
	}

	for _, b := range j.buckets {
		b.save = func(ctx context.Context) error { return j.save(ctx) }
		b.watchers = make([]*JSONWatcher, 0)
	}

	return nil
}

func (j *JSONClient) save(ctx context.Context) error {
	fd, err := j.store.SaveStreaming(ctx, j.fileName)
	if err != nil {
		return err
	}
	defer fd.Close()

	j.m.Lock()
	defer j.m.Unlock()

	if err := json.NewEncoder(fd).Encode(j.buckets); err != nil {
		return err
	}

	return nil
}

func (j *JSONClient) Bucket(ctx context.Context, name string) (Bucket, error) {
	j.m.RLock()
	b, exists := j.buckets[name]
	j.m.RUnlock()

	if !exists {
		b = &JSONBucket{
			BucketName: name,
			Data:       make(map[string][]byte),
			save:       func(ctx context.Context) error { return j.save(ctx) },
			watchers:   make([]*JSONWatcher, 0),
		}
		j.m.Lock()
		j.buckets[name] = b
		j.m.Unlock()
	}

	if err := j.save(ctx); err != nil {
		return nil, err
	}

	return b, nil
}

var _ Bucket = &JSONBucket{}

type JSONBucket struct {
	BucketName string            `json:"name"`
	Data       map[string][]byte `json:"data"`
	watchers   []*JSONWatcher
	m          sync.RWMutex
	save       func(ctx context.Context) error
}

func (b *JSONBucket) Name() string {
	return b.BucketName
}

func (b *JSONBucket) Get(ctx context.Context, key string) ([]byte, error) {
	b.m.RLock()
	v, exists := b.Data[key]
	b.m.RUnlock()

	if !exists {
		return nil, ErrKeyNotFound
	}

	return v, nil
}

func (b *JSONBucket) Set(ctx context.Context, key string, value []byte) error {
	b.m.Lock()
	b.Data[key] = value

	for _, w := range b.watchers {
		log.Printf("sending put change for %s", key)
		w.m.Lock()
		w.changes <- &Value{Key: key, Value: value, Operation: Put}
		w.m.Unlock()
	}

	b.m.Unlock()

	return b.save(ctx)
}

func (b *JSONBucket) Delete(ctx context.Context, key string) error {
	b.m.Lock()
	if _, exists := b.Data[key]; !exists {
		b.m.Unlock()
		return ErrKeyNotFound
	}

	delete(b.Data, key)

	for _, w := range b.watchers {
		log.Printf("sending delete change for %s", key)
		w.m.Lock()
		w.changes <- &Value{Key: key, Operation: Delete}
		w.m.Unlock()
	}

	b.m.Unlock()

	return b.save(ctx)
}

func (b *JSONBucket) ListKeys(ctx context.Context) ([]string, error) {
	b.m.RLock()
	defer b.m.RUnlock()

	keys := make([]string, 0, len(b.Data))
	for k := range b.Data {
		keys = append(keys, k)
	}

	return keys, nil
}

func (b *JSONBucket) WatchAll(ctx context.Context) (Watcher, error) {
	b.m.Lock()
	w := &JSONWatcher{
		bucket:  b,
		changes: make(chan *Value, len(b.Data)+1),
	}

	w.m.Lock()
	b.watchers = append(b.watchers, w)

	log.Printf("replaying %d changes", len(b.Data))
	for k, v := range b.Data {
		log.Printf("sending put change for %s", k)
		v_ := &Value{Key: k, Value: v, Operation: Put}

		w.changes <- v_
	}

	w.changes <- nil

	w.m.Unlock()

	b.m.Unlock()

	return w, nil
}

func (b *JSONBucket) Unwatch(w Watcher) {
	w_, ok := w.(*JSONWatcher)
	if !ok {
		return
	}

	b.m.Lock()
	defer b.m.Unlock()

	b.watchers = slices.DeleteFunc(b.watchers, func(w2 *JSONWatcher) bool {
		return w_.changes == w2.changes
	})
}

var ErrKeyNotFound = errors.New("key not found")

var _ Watcher = &JSONWatcher{}

type JSONWatcher struct {
	bucket  *JSONBucket
	changes chan *Value
	m       sync.Mutex
}

func (j *JSONWatcher) Changes() <-chan *Value {
	return j.changes
}

func (j *JSONWatcher) Unwatch() {
	j.m.Lock()
	defer j.m.Unlock()

	close(j.changes)
	j.bucket.Unwatch(j)
}
