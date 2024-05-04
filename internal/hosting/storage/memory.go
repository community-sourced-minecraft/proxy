package storage

import (
	"bytes"
	"context"
	"io"
)

var _ Storage = &Memory{}

type Memory struct {
	data map[string][]byte
}

func NewMemory() *Memory {
	return &Memory{
		data: make(map[string][]byte),
	}
}

func (m *Memory) Read(ctx context.Context, key string) ([]byte, error) {
	v, exists := m.data[key]
	if !exists {
		return nil, ErrKeyNotFound
	}

	return v, nil
}

func (m *Memory) ReadStreaming(ctx context.Context, key string) (io.ReadCloser, error) {
	v, err := m.Read(ctx, key)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(v)

	return io.NopCloser(buf), nil
}

func (m *Memory) Save(ctx context.Context, key string, content []byte) error {
	m.data[key] = content
	return nil
}

func (m *Memory) SaveStreaming(ctx context.Context, key string) (io.WriteCloser, error) {
	buf := &bytes.Buffer{}

	return &writeCloser{
		Buffer: buf,
		key:    key,
		m:      m,
	}, nil
}

func (m *Memory) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

type writeCloser struct {
	*bytes.Buffer
	key string
	m   *Memory
}

func (w *writeCloser) Close() error {
	w.m.data[w.key] = w.Bytes()
	return nil
}
