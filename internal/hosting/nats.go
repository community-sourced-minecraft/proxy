package hosting

import (
	"context"
	"encoding/json"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting/kv"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pkg/errors"
)

func connectToNATS(url string) (*nats.Conn, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to NATS")
	}

	return nc, nil
}

func connectToJetStream(url string) (jetstream.JetStream, error) {
	nc, err := connectToNATS(url)
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create JetStream")
	}

	return js, nil
}

func GetKeyFromKV(ctx context.Context, kv kv.Bucket, key string, obj any) error {
	val, err := kv.Get(ctx, key)
	if err != nil {
		return errors.Wrap(err, "failed to get key-value")
	}

	if err := json.Unmarshal(val, &obj); err != nil {
		return errors.Wrap(err, "failed to unmarshal key-value")
	}

	return nil
}

func SetKeyToKV(ctx context.Context, kv kv.Bucket, key string, obj any) error {
	val, err := json.Marshal(obj)
	if err != nil {
		return errors.Wrap(err, "failed to marshal key-value")
	}

	if err := kv.Set(ctx, key, val); err != nil {
		return errors.Wrap(err, "failed to set key-value")
	}

	return nil
}
