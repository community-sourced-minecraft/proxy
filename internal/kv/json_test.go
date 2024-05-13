package kv

import (
	"context"
	"testing"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/storage"
)

func TestJSONKV(t *testing.T) {
	ctx := context.Background()

	k, err := NewJSONClient(storage.WithLogger(storage.NewMemory()), "test.json")
	if err != nil {
		t.Fatal(err)
	}

	testKV(ctx, t, k)
}

func TestJSONKVWatch(t *testing.T) {
	ctx := context.Background()

	var k Client

	k, err := NewJSONClient(storage.NewMemory(), "test.json")
	if err != nil {
		t.Fatal(err)
	}

	k = WithLogger(k)

	testKVWatch(ctx, t, k)
}

func TestJSONKVResumability(t *testing.T) {
	ctx := context.Background()

	store := storage.WithLogger(storage.NewMemory())

	{
		k, err := NewJSONClient(store, "test.json")
		if err != nil {
			t.Fatal(err)
		}

		bucket1, err := k.Bucket(ctx, "test")
		if err != nil {
			t.Fatal(err)
		}

		if err := bucket1.Set(ctx, "test", []byte("test")); err != nil {
			t.Fatal(err)
		}
	}

	{
		k, err := NewJSONClient(store, "test.json")
		if err != nil {
			t.Fatal(err)
		}

		bucket1, err := k.Bucket(ctx, "test")
		if err != nil {
			t.Fatal(err)
		}

		v, err := bucket1.Get(ctx, "test")
		if err != nil {
			t.Fatal(err)
		}

		if string(v) != "test" {
			t.Fatalf("expected value to be 'test', got '%s'", string(v))
		}
	}
}
