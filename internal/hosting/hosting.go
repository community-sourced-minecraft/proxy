package hosting

import (
	"encoding/json"
	"log"
	"os"
	"strconv"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting/kv"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting/messaging"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting/storage"
)

type Hosting struct {
	strg storage.Storage
	kv   kv.Client
	msg  messaging.Messager
	Info *PodInfo
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

	return &Hosting{
		strg: storageC,
		kv:   kvC,
		msg:  msgC,
		Info: ParsePodInfo(),
	}, nil
}

func (n *Hosting) Storage() storage.Storage {
	return n.strg
}

func (n *Hosting) KV() kv.Client {
	return n.kv
}

func (n *Hosting) Messaging() messaging.Messager {
	return n.msg
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
		log.Fatal(err)
	}

	return v
}

func initStorage() (storage.Storage, error) {
	logging := getEnvBoolWithDefault("STORAGE_LOGGING", false)
	backend := getEnvWithDefault("STORAGE_BACKEND", "memory")
	backendOptions := os.Getenv("STORAGE_BACKEND_OPTIONS")

	var storageC storage.Storage
	switch backend {
	case "memory":
		log.Println("Using memory as storage backend")

		storageC = storage.NewMemory()

	case "fs":
		log.Println("Using FS as storage backend")

		opts := storage.FSOptions{}
		if err := json.Unmarshal([]byte(backendOptions), &opts); err != nil {
			return nil, err
		}

		storageC = storage.NewFS(opts)

	default:
		log.Fatalf("unknown storage backend: %s", backend)
	}

	if logging {
		storageC = storage.WithLogger(storageC)
	}

	return storageC, nil
}

func initKV(strg storage.Storage) (kv.Client, error) {
	logging := getEnvBoolWithDefault("KV_LOGGING", false)
	backend := getEnvWithDefault("KV_BACKEND", "json")
	backendOptions := os.Getenv("KV_BACKEND_OPTIONS")

	var kvC kv.Client
	var err error

	switch backend {
	case "nats":
		log.Println("Using NATS as KV backend")

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
		log.Println("Using JSON as KV backend")

		kvC, err = kv.NewJSONClient(strg, "")
	}

	if err != nil {
		return nil, err
	}

	if logging {
		log.Println("Enabling logging for KV")

		kvC = kv.WithLogger(kvC)
	}

	return kvC, nil
}

func initMessaging() (messaging.Messager, error) {
	logging := getEnvBoolWithDefault("MESSAGING_LOGGING", false)
	backend := getEnvWithDefault("MESSAGING_BACKEND", "nats")
	backendOptions := getEnvWithDefault("MESSAGING_BACKEND_OPTIONS", "{\"url\":\"nats://127.0.0.1:4222\"}")

	var msgC messaging.Messager
	var err error

	switch backend {
	case "nats":
		log.Println("Using NATS as messaging backend")

		opts := messaging.NATSOptions{}
		if err := json.Unmarshal([]byte(backendOptions), &opts); err != nil {
			return nil, err
		}

		nc, err := connectToNATS(opts.URL)
		if err != nil {
			return nil, err
		}

		msgC = messaging.NewNATS(nc)

	default:
		log.Fatalf("unknown messaging backend: %s", backend)
	}

	if err != nil {
		return nil, err
	}

	if logging {
		msgC = messaging.WithLogger(msgC)
	}

	return msgC, nil
}
