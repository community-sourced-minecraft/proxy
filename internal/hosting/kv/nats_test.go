package kv

import (
	"context"
	"os"
	"testing"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	testPort = 60782
)

func TestNATSKV(t *testing.T) {
	ctx := context.Background()

	s := runServerOnPort(testPort)
	s.Start()
	defer s.Shutdown()

	js := testConnectToNATS(s.ClientURL())
	k := NewNATSClient(js)

	testKV(ctx, t, k)
}

func TestNATSKVWatch(t *testing.T) {
	ctx := context.Background()

	s := runServerOnPort(testPort)
	s.Start()
	defer s.Shutdown()

	js := testConnectToNATS(s.ClientURL())
	k := WithLogger(NewNATSClient(js))

	testKVWatch(ctx, t, k)
}

func TestNATSKVResumability(t *testing.T) {
	ctx := context.Background()

	s := runServerOnPort(testPort)
	s.Start()
	defer s.Shutdown()

	{
		js := testConnectToNATS(s.ClientURL())
		k := NewNATSClient(js)

		bucket1, err := k.Bucket(ctx, "test")
		if err != nil {
			t.Fatal(err)
		}

		if err := bucket1.Set(ctx, "test", []byte("test")); err != nil {
			t.Fatal(err)
		}
	}

	{
		js := testConnectToNATS(s.ClientURL())
		k := NewNATSClient(js)

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

func runServerOnPort(port int) *natsserver.Server {
	tmp, err := os.MkdirTemp("", "nats")
	if err != nil {
		panic(err)
	}

	opts := natsserver.Options{
		Port:      port,
		JetStream: true,
		StoreDir:  tmp,
	}

	s, err := natsserver.NewServer(&opts)
	if err != nil {
		panic(err)
	}

	return s
}

func testConnectToNATS(uri string) jetstream.JetStream {
	nc, err := nats.Connect(uri)
	if err != nil {
		panic(err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		panic(err)
	}

	return js
}
