package storage

import (
	"context"
	"io"
	"log"
)

var _ Storage = &Logged{}

type Logged struct {
	s Storage
}

func WithLogger(s Storage) *Logged {
	return &Logged{s: s}
}

func (c *Logged) Read(ctx context.Context, key string) ([]byte, error) {
	v, err := c.s.Read(ctx, key)
	if err != nil {
		log.Printf("ERROR: Read key %s%v", key, err)
		return nil, err
	}

	log.Printf("INFO: Read key %s: %s", key, string(v))
	return v, nil
}

func (c *Logged) ReadStreaming(ctx context.Context, key string) (io.ReadCloser, error) {
	v, err := c.s.ReadStreaming(ctx, key)
	if err != nil {
		log.Printf("ERROR: ReadStreaming key %s: %v", key, err)
		return nil, err
	}

	return &loggedReadCloser{
		ReadCloser: v,
		key:        key,
	}, nil
}

type loggedReadCloser struct {
	io.ReadCloser
	key string
}

func (c *loggedReadCloser) Read(p []byte) (n int, err error) {
	n, err = c.ReadCloser.Read(p)
	if err != nil {
		log.Printf("ERROR: Read key %s: %v", c.key, err)
		return
	}

	log.Printf("INFO: Read key %s: %s", c.key, string(p))
	return
}
func (c *loggedReadCloser) Close() error {
	err := c.ReadCloser.Close()
	if err != nil {
		log.Printf("ERROR: Close key %s: %v", c.key, err)
		return err
	}

	log.Printf("INFO: Close key %s", c.key)
	return nil
}

func (c *Logged) Save(ctx context.Context, key string, content []byte) error {
	if err := c.s.Save(ctx, key, content); err != nil {
		log.Printf("ERROR: Save key %s: %v", key, err)
		return err
	}

	log.Printf("INFO: Save key %s: %s", key, string(content))
	return nil
}

func (c *Logged) SaveStreaming(ctx context.Context, key string) (io.WriteCloser, error) {
	writer, err := c.s.SaveStreaming(ctx, key)
	if err != nil {
		log.Printf("ERROR: SaveStreaming key %s: %v", key, err)
		return nil, err
	}

	return &loggedWriteCloser{
		WriteCloser: writer,
		key:         key,
	}, nil
}

func (c *Logged) Delete(ctx context.Context, key string) error {
	if err := c.s.Delete(ctx, key); err != nil {
		log.Printf("ERROR: Delete key %s: %v", key, err)
		return err
	}

	log.Printf("INFO: Delete key %s", key)

	return nil
}

type loggedWriteCloser struct {
	io.WriteCloser
	key string
}

func (l *loggedWriteCloser) Write(p []byte) (n int, err error) {
	n, err = l.WriteCloser.Write(p)
	if err != nil {
		log.Printf("ERROR: Write key %s: %v", l.key, err)
		return
	}

	log.Printf("INFO: Write key %s: %s", l.key, string(p))
	return
}

func (l *loggedWriteCloser) Close() error {
	err := l.WriteCloser.Close()
	if err != nil {
		log.Printf("ERROR: Close key %s: %v", l.key, err)
		return err
	}

	log.Printf("INFO: Close key %s", l.key)
	return nil
}
