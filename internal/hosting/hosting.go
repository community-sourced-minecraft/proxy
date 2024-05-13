package hosting

import (
	"os"
	"strconv"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/kv"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/messaging"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/storage"
	"github.com/rs/zerolog/log"
)

type Hosting struct {
	strg        storage.Storage
	kv          kv.Client
	msg         messaging.Messager
	Info        *PodInfo
	nwEventBus  *EventBus
	podEventBus *EventBus
}

func Init() (*Hosting, error) {
	storageC, err := initStorage()
	if err != nil {
		return nil, err
	}

	kvC, err := initKV(storageC)
	if err != nil {
		return nil, err
	}

	msgC, err := initMessaging()
	if err != nil {
		return nil, err
	}

	info := ParsePodInfo()

	nwEventBus, err := NewEventBus(msgC, info.RPCNetworkSubject())
	if err != nil {
		return nil, err
	}

	podEventBus, err := NewEventBus(msgC, info.RPCPodSubject())
	if err != nil {
		return nil, err
	}

	h := &Hosting{
		strg:        storageC,
		kv:          kvC,
		msg:         msgC,
		Info:        info,
		nwEventBus:  nwEventBus,
		podEventBus: podEventBus,
	}

	return h, nil
}

func getEnvWithDefault(key, def string) string {
	v, exists := os.LookupEnv(key)
	if !exists {
		return def
	}

	return v
}

func getEnvBoolWithDefault(key string, def bool) bool {
	raw, exists := os.LookupEnv(key)
	if !exists {
		return def
	}

	v, err := strconv.ParseBool(raw)
	if err != nil {
		log.Fatal().Err(err).Str("key", key).Msg("Failed to parse boolean from env")
	}

	return v
}
