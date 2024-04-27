package whitelist

import (
	"encoding/json"
	"os"
	"slices"
	"sync"
)

type WhitelistFile struct {
	Enabled     bool     `json:"enabled"`
	Whitelisted []string `json:"whitelisted"`
}

type Whitelist struct {
	name string
	file WhitelistFile
	m    sync.RWMutex
}

func ReadWhitelist(file string) (*Whitelist, error) {
	w := &Whitelist{
		name: file,
		file: WhitelistFile{
			Enabled:     false,
			Whitelisted: make([]string, 0),
		},
	}

	if err := w.Reload(); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *Whitelist) Reload() error {
	fd, err := os.OpenFile(w.name, os.O_RDONLY, 0755)
	if err != nil {
		if os.IsNotExist(err) {
			// Save the default whitelist file
			return w.Save()
		}

		return err
	}
	defer fd.Close()

	w.m.Lock()
	defer w.m.Unlock()
	if err := json.NewDecoder(fd).Decode(&w.file); err != nil {
		return err
	}

	return nil
}

func (w *Whitelist) Save() error {
	fd, err := os.OpenFile(w.name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()

	w.m.Lock()
	defer w.m.Unlock()
	if err := json.NewEncoder(fd).Encode(w.file); err != nil {
		return err
	}

	return nil
}

func (w *Whitelist) Enabled() bool {
	w.m.RLock()
	defer w.m.RUnlock()

	return w.file.Enabled
}

func (w *Whitelist) Enable() error {
	w.m.Lock()
	w.file.Enabled = true
	w.m.Unlock()

	return w.Save()
}

func (w *Whitelist) Disable() error {
	w.m.Lock()
	w.file.Enabled = false
	w.m.Unlock()

	return w.Save()
}

func (w *Whitelist) Add(uuid string) error {
	w.m.Lock()
	w.file.Whitelisted = append(w.file.Whitelisted, uuid)
	w.m.Unlock()

	return w.Save()
}

func (w *Whitelist) Remove(uuid string) error {
	w.m.Lock()
	w.file.Whitelisted = slices.DeleteFunc(w.file.Whitelisted, func(s string) bool {
		return s == uuid
	})
	w.m.Unlock()

	return w.Save()
}

func (w *Whitelist) Contains(uuid string) bool {
	w.m.RLock()
	defer w.m.RUnlock()

	return slices.Contains(w.file.Whitelisted, uuid)
}

func (w *Whitelist) AllWhitelisted() []string {
	w.m.RLock()
	defer w.m.RUnlock()

	copy := make([]string, 0)
	copy = append(copy, w.file.Whitelisted...)

	return copy
}
