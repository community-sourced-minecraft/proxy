package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

var _ Storage = &FS{}

type FSOptions struct {
	Folder string `json:"folder"`
}

type FS struct {
	folder string
}

func NewFS(opts FSOptions) *FS {
	return &FS{folder: opts.Folder}
}

func (f *FS) Read(ctx context.Context, key string) ([]byte, error) {
	fd, err := f.ReadStreaming(ctx, key)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	buf, err := io.ReadAll(fd)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (f *FS) ReadStreaming(ctx context.Context, key string) (io.ReadCloser, error) {
	fd, err := os.Open(filepath.Join(f.folder, key))
	if os.IsNotExist(err) {
		return nil, ErrKeyNotFound
	} else if err != nil {
		return nil, err
	}

	return fd, nil
}

func (f *FS) Save(ctx context.Context, key string, content []byte) error {
	fd, err := f.SaveStreaming(ctx, key)
	if err != nil {
		return err
	}
	defer fd.Close()

	if _, err := fd.Write(content); err != nil {
		return err
	}

	return nil
}

func (f *FS) SaveStreaming(ctx context.Context, key string) (io.WriteCloser, error) {
	fd, err := os.Create(filepath.Join(f.folder, key))
	if err != nil {
		return nil, err
	}

	return fd, nil
}

func (f *FS) Delete(ctx context.Context, key string) error {
	if err := os.Remove(filepath.Join(f.folder, key)); err != nil {
		return err
	}

	return nil
}
