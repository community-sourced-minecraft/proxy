package whitelist

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"slices"
	"sync"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting/kv"
)

var _ Whitelist = &NATSWhitelist{}

type NATSWhitelist struct {
	Enabled     bool     `json:"enabled"`
	Whitelisted []string `json:"whitelisted"`
	m           sync.RWMutex
	h           *hosting.Hosting
	kv          kv.Bucket
}

func NewNATSWhitelist(ctx context.Context, h *hosting.Hosting) (*NATSWhitelist, error) {
	kv, err := h.KV().Bucket(ctx, h.Info.KVNetworkKey()+"_whitelist")
	if err != nil {
		return nil, err
	}

	w := &NATSWhitelist{
		Enabled:     false,
		Whitelisted: make([]string, 0),
		h:           h,
		kv:          kv,
	}

	watcher, err := kv.WatchAll(context.Background())
	if err != nil {
		return nil, err
	}

	go func() {
		for key := range watcher.Changes() {
			if key == nil {
				continue
			}

			switch key.Key {
			case "enabled":
				log.Printf("Enabled key changed: %s", key.Value)

				w.m.Lock()

				if err := json.Unmarshal(key.Value, &w.Enabled); err != nil {
					w.m.Unlock()
					log.Printf("Failed to unmarshal enabled key: %v", err)
				}

				w.m.Unlock()

			case "whitelisted":
				log.Printf("Whitelisted key changed: %s", key.Value)

				w.m.Lock()

				if err := json.Unmarshal(key.Value, &w.Whitelisted); err != nil {
					w.m.Unlock()
					log.Printf("Failed to unmarshal whitelisted key: %v", err)
				}

				w.m.Unlock()
			}
		}
	}()

	return w, nil
}

func (w *NATSWhitelist) Reload() error {
	w.m.Lock()
	defer w.m.Unlock()

	if err := hosting.GetKeyFromKV(context.Background(), w.kv, "enabled", &w.Enabled); errors.Is(errors.Unwrap(err), kv.ErrKeyNotFound) {
		w.Enabled = false
	} else {
		return err
	}

	if err := hosting.GetKeyFromKV(context.Background(), w.kv, "whitelisted", &w.Whitelisted); errors.Is(errors.Unwrap(err), kv.ErrKeyNotFound) {
		w.Whitelisted = make([]string, 0)
	} else if err != nil {
		return err
	}

	return nil
}

func (w *NATSWhitelist) saveEnabled() error {
	w.m.Lock()
	defer w.m.Unlock()

	if err := hosting.SetKeyToKV(context.Background(), w.kv, "enabled", w.Enabled); err != nil {
		return err
	}

	return nil
}

func (w *NATSWhitelist) saveWhitelisted() error {
	w.m.Lock()
	defer w.m.Unlock()

	if err := hosting.SetKeyToKV(context.Background(), w.kv, "whitelisted", w.Whitelisted); err != nil {
		return err
	}

	return nil
}

func (w *NATSWhitelist) IsEnabled() bool {
	w.m.RLock()
	defer w.m.RUnlock()

	return w.Enabled
}

func (w *NATSWhitelist) Enable() error {
	w.m.Lock()
	w.Enabled = true
	w.m.Unlock()

	return w.saveEnabled()
}

func (w *NATSWhitelist) Disable() error {
	w.m.Lock()
	w.Enabled = false
	w.m.Unlock()

	return w.saveEnabled()
}

func (w *NATSWhitelist) Add(uuid string) error {
	w.m.Lock()
	w.Whitelisted = append(w.Whitelisted, uuid)
	w.m.Unlock()

	return w.saveWhitelisted()
}

func (w *NATSWhitelist) Remove(uuid string) error {
	w.m.Lock()
	w.Whitelisted = slices.DeleteFunc(w.Whitelisted, func(s string) bool {
		return s == uuid
	})
	w.m.Unlock()

	return w.saveWhitelisted()
}

func (w *NATSWhitelist) Contains(uuid string) bool {
	w.m.RLock()
	defer w.m.RUnlock()

	return slices.Contains(w.Whitelisted, uuid)
}

func (w *NATSWhitelist) AllWhitelisted() []string {
	w.m.RLock()
	defer w.m.RUnlock()

	copy := make([]string, 0)
	copy = append(copy, w.Whitelisted...)

	return copy
}
