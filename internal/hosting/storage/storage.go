package storage

import (
	"context"
	"errors"
	"io"
)

type Storage interface {
	Read(ctx context.Context, key string) ([]byte, error)
	ReadStreaming(ctx context.Context, key string) (io.ReadCloser, error)
	Save(ctx context.Context, key string, content []byte) error
	SaveStreaming(ctx context.Context, key string) (io.WriteCloser, error)
	Delete(ctx context.Context, key string) error
}

var (
	ErrKeyNotFound = errors.New("key not found")
)
