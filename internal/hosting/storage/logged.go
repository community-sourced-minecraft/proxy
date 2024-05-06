package storage

import (
	"context"
	"io"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var _ Storage = &Logged{}

type Logged struct {
	s Storage
}

func WithLogger(s Storage) *Logged {
	return &Logged{s: s}
}

func (c *Logged) Read(ctx context.Context, key string) ([]byte, error) {
	l := log.With().Str("key", key).Logger()

	v, err := c.s.Read(ctx, key)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read")
		return nil, err
	}

	l.Trace().Bytes("data", v).Msg("Read key")

	return v, nil
}

func (c *Logged) ReadStreaming(ctx context.Context, key string) (io.ReadCloser, error) {
	l := log.With().Str("key", key).Logger()

	v, err := c.s.ReadStreaming(ctx, key)
	if err != nil {
		l.Error().Err(err).Msg("Failed to read streaming")
		return nil, err
	}

	l.Trace().Msg("Read streaming")

	return &loggedReadCloser{
		ReadCloser: v,
		l:          l,
	}, nil
}

type loggedReadCloser struct {
	io.ReadCloser
	l zerolog.Logger
}

func (c *loggedReadCloser) Read(p []byte) (n int, err error) {
	n, err = c.ReadCloser.Read(p)
	if err != nil {
		c.l.Error().Err(err).Msg("Failed to read streaming")
		return
	}

	c.l.Trace().Bytes("data", p[:n]).Msg("Read streaming")

	return
}
func (c *loggedReadCloser) Close() error {
	err := c.ReadCloser.Close()
	if err != nil {
		c.l.Error().Err(err).Msg("Failed to close streaming reader")
		return err
	}

	c.l.Trace().Msg("Close streaming reader")

	return nil
}

func (c *Logged) Save(ctx context.Context, key string, content []byte) error {
	l := log.With().Str("key", key).Bytes("content", content).Logger()

	if err := c.s.Save(ctx, key, content); err != nil {
		l.Error().Err(err).Msg("Failed to save")
		return err
	}

	l.Trace().Msg("Saved")

	return nil
}

func (c *Logged) SaveStreaming(ctx context.Context, key string) (io.WriteCloser, error) {
	l := log.With().Str("key", key).Logger()

	writer, err := c.s.SaveStreaming(ctx, key)
	if err != nil {
		l.Error().Err(err).Msg("Failed to save streaming")
		return nil, err
	}

	l.Trace().Msg("Save streaming")

	return &loggedWriteCloser{
		WriteCloser: writer,
		l:           l,
	}, nil
}

func (c *Logged) Delete(ctx context.Context, key string) error {
	l := log.With().Str("key", key).Logger()

	if err := c.s.Delete(ctx, key); err != nil {
		l.Error().Err(err).Msg("Failed to delete")
		return err
	}

	l.Trace().Msg("Deleted")

	return nil
}

type loggedWriteCloser struct {
	io.WriteCloser
	l zerolog.Logger
}

func (c *loggedWriteCloser) Write(p []byte) (n int, err error) {
	l := log.With().Bytes("data", p).Logger()

	n, err = c.WriteCloser.Write(p)
	if err != nil {
		l.Error().Err(err).Msg("Failed to write streaming")
		return
	}

	l.Trace().Msg("Write streaming")

	return
}

func (c *loggedWriteCloser) Close() error {
	if err := c.WriteCloser.Close(); err != nil {
		c.l.Error().Err(err).Msg("Failed to close streaming writer")
		return err
	}

	c.l.Trace().Msg("Close streaming writer")

	return nil
}
