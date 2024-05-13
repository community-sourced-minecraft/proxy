package kv

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func testKV(ctx context.Context, t *testing.T, k Client) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	testKVCRUD(ctx, t, k)
	testKVDoubleAccess(ctx, t, k)
}

func testKVCRUD(ctx context.Context, t *testing.T, k Client) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	t.Run("CRUD", func(t *testing.T) {
		b, err := k.Bucket(ctx, "crud")
		if err != nil {
			t.Fatal(err)
		}

		if b.Name() != "crud" {
			t.Fatalf("expected bucket name to be 'crud', got '%s'", b.Name())
		}

		if err := b.Set(ctx, "test", []byte("test")); err != nil {
			t.Fatal(err)
		}

		v, err := b.Get(ctx, "test")
		if err != nil {
			t.Fatal(err)
		}

		if string(v) != "test" {
			t.Fatalf("expected value to be 'test', got '%s'", string(v))
		}

		keys, err := b.ListKeys(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if len(keys) != 1 {
			t.Fatalf("expected 1 key, got %d", len(keys))
		}

		if keys[0] != "test" {
			t.Fatalf("expected key to be 'test', got '%s'", keys[0])
		}

		if err := b.Delete(ctx, "test"); err != nil {
			t.Fatal(err)
		}
	})
}

func testKVDoubleAccess(ctx context.Context, t *testing.T, k Client) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	t.Run("Accessing the same bucket twice", func(t *testing.T) {
		bucket1, err := k.Bucket(ctx, "double-access")
		if err != nil {
			t.Fatal(err)
		}

		if bucket1.Name() != "double-access" {
			t.Fatalf("expected bucket name to be 'double-access', got '%s'", bucket1.Name())
		}

		if err := bucket1.Set(ctx, "test", []byte("test")); err != nil {
			t.Fatal(err)
		}

		bucket1_, err := k.Bucket(ctx, "double-access")
		if err != nil {
			t.Fatal(err)
		}

		v, err := bucket1_.Get(ctx, "test")
		if err != nil {
			t.Fatal(err)
		}

		if string(v) != "test" {
			t.Fatalf("expected value to be 'test', got '%s'", string(v))
		}
	})
}

func testKVWatch(ctx context.Context, t *testing.T, k Client) {
	testKVWatchWatch(ctx, t, k)
	testKVWatchReplay(ctx, t, k)
}

func testKVWatchWatch(ctx context.Context, t *testing.T, k Client) {
	t.Run("Watch", func(t *testing.T) {
		b, err := k.Bucket(ctx, "watch")
		if err != nil {
			t.Fatal(err)
		}

		watcher, err := b.WatchAll(ctx)
		if err != nil {
			t.Fatal(err)
		}

		tests := []*Value{
			nil,
			{Key: "test", Value: []byte("test"), Operation: Put},
			{Key: "test", Value: nil, Operation: Delete},
		}

		t.Run("Watcher", func(t *testing.T) {
			t.Parallel()

			for _, test := range tests {
				t.Logf("expected: %v", test)

				msg := <-watcher.Changes()
				t.Logf("got:      %v", msg)

				if msg == nil {
					if test != nil {
						t.Fatalf("expected key to be '%s', got nil", test.Key)
					}
				} else {
					if test == nil {
						t.Fatalf("expected key to be nil, got '%s'", msg.Key)
					}

					if msg.Key != test.Key {
						t.Fatalf("expected key to be '%s', got '%s'", test.Key, msg.Key)
					}

					if string(msg.Value) != string(test.Value) {
						t.Fatalf("expected value to be '%s', got '%s'", string(test.Value), string(msg.Value))
					}

					if msg.Operation != test.Operation {
						t.Fatalf("expected operation to be %d, got %d", test.Operation, msg.Operation)
					}
				}
			}
		})

		t.Run("Modifier", func(t *testing.T) {
			t.Parallel()

			if err := b.Set(ctx, "test", []byte("test")); err != nil {
				t.Fatal(err)
			}

			v, err := b.Get(ctx, "test")
			if err != nil {
				t.Fatal(err)
			}

			if string(v) != "test" {
				t.Fatalf("expected value to be 'test', got '%s'", string(v))
			}

			keys, err := b.ListKeys(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if len(keys) != 1 {
				t.Fatalf("expected 1 key, got %d", len(keys))
			}

			if keys[0] != "test" {
				t.Fatalf("expected key to be 'test', got '%s'", keys[0])
			}

			if err := b.Delete(ctx, "test"); err != nil {
				t.Fatal(err)
			}
		})
	})
}

func testKVWatchReplay(ctx context.Context, t *testing.T, k Client) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	t.Run("Initial data replay", func(t *testing.T) {
		b, err := k.Bucket(ctx, "watch-replay")
		if err != nil {
			t.Fatal(err)
		}

		if err := b.Set(ctx, "test", []byte("test")); err != nil {
			t.Fatal(err)
		}

		watcher, err := b.WatchAll(ctx)
		if err != nil {
			t.Fatal(err)
		}

		tests := []*Value{
			{Key: "test", Value: []byte("test"), Operation: Put},
			nil,
			{Key: "test", Value: []byte("test2"), Operation: Put},
			{Key: "test", Value: nil, Operation: Delete},
		}

		t.Run("Watcher", func(t *testing.T) {
			t.Parallel()

			for _, test := range tests {
				t.Logf("expected: %v", test)

				msg := <-watcher.Changes()
				t.Logf("got:      %v", msg)

				if msg == nil {
					if test != nil {
						t.Fatalf("expected key to be '%s', got nil", test.Key)
					}
				} else {
					if msg.Key != test.Key {
						t.Fatalf("expected key to be '%s', got '%s'", test.Key, msg.Key)
					}

					if string(msg.Value) != string(test.Value) {
						t.Fatalf("expected value to be '%s', got '%s'", string(test.Value), string(msg.Value))
					}

					if msg.Operation != test.Operation {
						t.Fatalf("expected operation to be %d, got %d", test.Operation, msg.Operation)
					}
				}
			}

			watcher.Unwatch()
		})

		t.Run("Modifier", func(t *testing.T) {
			t.Parallel()

			if err := b.Set(ctx, "test", []byte("test2")); err != nil {
				t.Fatal(err)
			}

			if err := b.Delete(ctx, "test"); err != nil {
				t.Fatal(err)
			}

			if err := b.Set(ctx, "test", []byte("test3")); err != nil {
				t.Fatal(err)
			}
		})
	})
}
