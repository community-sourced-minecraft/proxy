package hosting

import (
	"context"
	"encoding/json"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pkg/errors"
)

func connectToNATS() (*nats.Conn, jetstream.JetStream, error) {
	natsUrl := os.Getenv("NATS_URL")

	nc, err := nats.Connect(natsUrl)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to connect to NATS")
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create JetStream")
	}

	return nc, js, nil
}

func GetKeyFromKV(ctx context.Context, kv jetstream.KeyValue, key string, obj any) error {
	val, err := kv.Get(ctx, key)
	if err != nil {
		return errors.Wrap(err, "failed to get key-value")
	}

	if err := json.Unmarshal(val.Value(), &obj); err != nil {
		return errors.Wrap(err, "failed to unmarshal key-value")
	}

	return nil
}

func SetKeyToKV(ctx context.Context, kv jetstream.KeyValue, key string, obj any) error {
	val, err := json.Marshal(obj)
	if err != nil {
		return errors.Wrap(err, "failed to marshal key-value")
	}

	if _, err := kv.Put(ctx, key, val); err != nil {
		return errors.Wrap(err, "failed to set key-value")
	}

	return nil
}
