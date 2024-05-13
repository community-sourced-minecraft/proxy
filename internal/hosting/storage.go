package hosting

import (
	"encoding/json"
	"os"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/storage"
	"github.com/rs/zerolog/log"
)

func initStorage() (storage.Storage, error) {
	logging := getEnvBoolWithDefault("STORAGE_LOGGING", false)
	backend := getEnvWithDefault("STORAGE_BACKEND", "memory")
	backendOptions := os.Getenv("STORAGE_BACKEND_OPTIONS")

	var storageC storage.Storage
	switch backend {
	case "memory":
		log.Info().Msg("Using memory as storage backend")

		storageC = storage.NewMemory()

	case "fs":
		log.Info().Msg("Using FS as storage backend")

		opts := storage.FSOptions{}
		if err := json.Unmarshal([]byte(backendOptions), &opts); err != nil {
			return nil, err
		}

		storageC = storage.NewFS(opts)

	default:
		log.Fatal().Msgf("unknown storage backend: %s", backend)
	}

	if logging {
		storageC = storage.WithLogger(storageC)
	}

	return storageC, nil
}

func (n *Hosting) Storage() storage.Storage {
	return n.strg
}
