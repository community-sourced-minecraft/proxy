package hosting

import (
	"encoding/json"
	"os"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/kv"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/storage"
	"github.com/rs/zerolog/log"
)

func initKV(strg storage.Storage) (kv.Client, error) {
	logging := getEnvBoolWithDefault("KV_LOGGING", false)
	backend := getEnvWithDefault("KV_BACKEND", "json")
	backendOptions := os.Getenv("KV_BACKEND_OPTIONS")

	var kvC kv.Client
	var err error

	switch backend {
	case "nats":
		log.Info().Msg("Using NATS as KV backend")

		opts := kv.NATSOptions{}
		if err := json.Unmarshal([]byte(backendOptions), &opts); err != nil {
			return nil, err
		}

		js, err := connectToJetStream(opts.URL)
		if err != nil {
			return nil, err
		}

		kvC = kv.NewNATSClient(js)

	case "json":
		log.Info().Msg("Using JSON as KV backend")

		kvC, err = kv.NewJSONClient(strg, "")

	default:
		log.Fatal().Msgf("unknown KV backend: %s", backend)
	}

	if err != nil {
		return nil, err
	}

	if logging {
		log.Info().Msg("Enabling logging for KV")

		kvC = kv.WithLogger(kvC)
	}

	return kvC, nil
}

func (n *Hosting) KV() kv.Client {
	return n.kv
}
