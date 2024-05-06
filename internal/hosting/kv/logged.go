package kv

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
		l: log.With().Str("bucket", name).Logger(),
	}, nil
}

var _ Bucket = &LoggedBucket{}

type LoggedBucket struct {
	b Bucket
	l zerolog.Logger
}

func (b *LoggedBucket) Name() string {
	return b.b.Name()
}

func (b *LoggedBucket) Get(ctx context.Context, key string) ([]byte, error) {
	l := b.l.With().Str("key", key).Logger()

	v, err := b.b.Get(ctx, key)
	if err != nil {
		l.Debug().Err(err).Msg("Get")
		return nil, err
	}

	l.Debug().Bytes("data", v).Msg("Get")

	return v, nil
}

func (b *LoggedBucket) Set(ctx context.Context, key string, value []byte) error {
	l := b.l.With().Str("key", key).Bytes("value", value).Logger()

	if err := b.b.Set(ctx, key, value); err != nil {
		l.Debug().Err(err).Msg("Set")
		return err
	}

	l.Debug().Msg("Set")

	return nil
}

func (b *LoggedBucket) Delete(ctx context.Context, key string) error {
	l := b.l.With().Str("key", key).Logger()

	if err := b.b.Delete(ctx, key); err != nil {
		l.Debug().Err(err).Msg("Delete")
		return err
	}

	l.Debug().Msg("Delete")

	return nil
}

func (b *LoggedBucket) WatchAll(ctx context.Context) (Watcher, error) {
	w, err := b.b.WatchAll(ctx)
	if err != nil {
		b.l.Debug().Err(err).Msg("WatchAll")
		return nil, err
	}

	b.l.Debug().Msg("WatchAll")

	return &LoggedWatcher{
		w: w,
		l: b.l,
	}, nil
}

func (b *LoggedBucket) Unwatch(w Watcher) {
	b.b.Unwatch(w)

	b.l.Debug().Msg("Unwatch")
}

func (b *LoggedBucket) ListKeys(ctx context.Context) ([]string, error) {
	return b.b.ListKeys(ctx)
}

var _ Watcher = &LoggedWatcher{}

type LoggedWatcher struct {
	w Watcher
	l zerolog.Logger
}

func (w *LoggedWatcher) Changes() <-chan *Value {
	return w.w.Changes()
}

func (w *LoggedWatcher) Unwatch() {
	w.w.Unwatch()

	w.l.Debug().Msg("Unwatch")
}
