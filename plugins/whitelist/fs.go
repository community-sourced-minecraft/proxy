package whitelist

import (
	"encoding/json"
	"os"
	"slices"
	"sync"
)

var _ Whitelist = &FSWhitelist{}

type FSWhitelist struct {
	Enabled     bool     `json:"enabled"`
	Whitelisted []string `json:"whitelisted"`
	fileName    string
	m           sync.RWMutex
}

func NewFSWhitelist(fileName string) (*FSWhitelist, error) {
	w := &FSWhitelist{
		fileName:    fileName,
		Enabled:     false,
		Whitelisted: make([]string, 0),
	}

	if err := w.Reload(); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *FSWhitelist) Reload() error {
	fd, err := os.OpenFile(w.fileName, os.O_RDONLY, 0755)
	if err != nil {
		if os.IsNotExist(err) {
			// Save the default whitelist file
			return w.save()
		}

		return err
	}
	defer fd.Close()

	w.m.Lock()
	defer w.m.Unlock()
	if err := json.NewDecoder(fd).Decode(&w); err != nil {
		return err
	}

	return nil
}

func (w *FSWhitelist) save() error {
	fd, err := os.OpenFile(w.fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fd.Close()

	w.m.Lock()
	defer w.m.Unlock()
	if err := json.NewEncoder(fd).Encode(w); err != nil {
		return err
	}

	return nil
}

func (w *FSWhitelist) IsEnabled() bool {
	w.m.RLock()
	defer w.m.RUnlock()

	return w.Enabled
}

func (w *FSWhitelist) Enable() error {
	w.m.Lock()
	w.Enabled = true
	w.m.Unlock()

	return w.save()
}

func (w *FSWhitelist) Disable() error {
	w.m.Lock()
	w.Enabled = false
	w.m.Unlock()

	return w.save()
}

func (w *FSWhitelist) Add(uuid string) error {
	w.m.Lock()
	w.Whitelisted = append(w.Whitelisted, uuid)
	w.m.Unlock()

	return w.save()
}

func (w *FSWhitelist) Remove(uuid string) error {
	w.m.Lock()
	w.Whitelisted = slices.DeleteFunc(w.Whitelisted, func(s string) bool {
		return s == uuid
	})
	w.m.Unlock()

	return w.save()
}

func (w *FSWhitelist) Contains(uuid string) bool {
	w.m.RLock()
	defer w.m.RUnlock()

	return slices.Contains(w.Whitelisted, uuid)
}

func (w *FSWhitelist) AllWhitelisted() []string {
	w.m.RLock()
	defer w.m.RUnlock()

	copy := make([]string, 0)
	copy = append(copy, w.Whitelisted...)

	return copy
}
