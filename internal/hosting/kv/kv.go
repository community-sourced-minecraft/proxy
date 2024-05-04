package kv

import "context"

type Client interface {
	Bucket(ctx context.Context, name string) (Bucket, error)
}

type Bucket interface {
	Name() string
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
	WatchAll(ctx context.Context) (Watcher, error)
	Unwatch(w Watcher)
	ListKeys(ctx context.Context) ([]string, error)
}

type Watcher interface {
	Changes() <-chan *Value
	Unwatch()
}

type Value struct {
	Key       string
	Value     []byte
	Operation Operation
}

type Operation int

const (
	Put Operation = iota
	Delete
)

func (o Operation) String() string {
	switch o {
	case Put:
		return "Put"
	case Delete:
		return "Delete"
	default:
		return "Unknown"
	}
}
