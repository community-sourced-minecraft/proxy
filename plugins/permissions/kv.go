package permissions

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting/kv"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util/uuid"
)

type PermissionUser struct {
	Groups      []string `json:"groups"`
	Permissions []string `json:"permissions"`
}

type PermissionGroup struct {
	Prefix      string   `json:"prefix"`
	Weight      uint8    `json:"weight"`
	Permissions []string `json:"permissions"`
}

type Permissions struct {
	Users  map[string]PermissionUser
	Groups map[string]PermissionGroup
	m      sync.RWMutex
	h      *hosting.Hosting
	kv     kv.Bucket
}

func NewKVPermissions(ctx context.Context, h *hosting.Hosting) (*Permissions, error) {
	kv, err := h.KV().Bucket(ctx, h.Info.KVNetworkKey()+"_permissions")
	if err != nil {
		return nil, err
	}

	w := &Permissions{
		Users:  make(map[string]PermissionUser),
		Groups: make(map[string]PermissionGroup),
		h:      h,
		kv:     kv,
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
			case "users":
				log.Printf("Users key changed: %s", key.Value)

				w.m.Lock()

				if err := json.Unmarshal(key.Value, &w.Users); err != nil {
					w.m.Unlock()
					log.Printf("Failed to unmarshal users key: %v", err)
				}

				w.m.Unlock()

			case "groups":
				log.Printf("Groups key changed: %s", key.Value)

				w.m.Lock()

				if err := json.Unmarshal(key.Value, &w.Groups); err != nil {
					w.m.Unlock()
					log.Printf("Failed to unmarshal groups key: %v", err)
				}

				w.m.Unlock()
			}
		}
	}()

	return w, nil
}

func (w *Permissions) Reload(ctx context.Context) error {
	w.m.Lock()
	defer w.m.Unlock()

	if err := hosting.GetKeyFromKV(ctx, w.kv, "users", &w.Users); errors.Is(errors.Unwrap(err), kv.ErrKeyNotFound) {
		w.Users = make(map[string]PermissionUser)
	} else if err != nil {
		return err
	}

	if err := hosting.GetKeyFromKV(ctx, w.kv, "groups", &w.Groups); errors.Is(errors.Unwrap(err), kv.ErrKeyNotFound) {
		w.Groups = make(map[string]PermissionGroup)
	} else if err != nil {
		return err
	}

	return nil
}

func (w *Permissions) saveUsers(ctx context.Context) error {
	w.m.Lock()
	defer w.m.Unlock()

	return hosting.SetKeyToKV(ctx, w.kv, "users", w.Users)
}

func (w *Permissions) saveGroups(ctx context.Context) error {
	w.m.Lock()
	defer w.m.Unlock()

	return hosting.SetKeyToKV(ctx, w.kv, "groups", w.Groups)
}

func (p *Permissions) GroupNames() []string {
	return util.MapKeys(p.Groups)
}

func (p *Permissions) GetUsers() []string {
	return util.MapKeys(p.Users)
}

func (p *Permissions) GetGroup(name string) (PermissionGroup, bool) {
	group, exists := p.Groups[name]
	return group, exists
}

func (p *Permissions) UserPermissions(name string) ([]string, bool) {
	user, ok := p.Users[name]
	if !ok {
		return make([]string, 0), false
	}

	return user.Permissions, true
}

func (p *Permissions) UserGroups(name string) ([]string, bool) {
	user, ok := p.Users[name]
	if !ok {
		return make([]string, 0), false
	}

	return user.Groups, true
}

func (p *Permissions) GroupHasPermission(name string, permission string) bool {
	group, exists := p.GetGroup(name)
	if !exists {
		log.Printf("WARN: Group %s does not exist", name)
		return false
	}

	for _, _permission := range group.Permissions {
		if _permission == permission {
			return true
		}

		if strings.Split(_permission, ".")[0] == strings.Split(permission, ".")[0] && strings.Split(_permission, ".")[1] == "*" {
			return true
		}
	}

	return false
}

func (p *Permissions) UserHasPermission(player string, permission string) bool {
	player = uuid.Normalize(player)

	user, ok := p.Users[player]
	if !ok {
		log.Printf("DBG: User %s does not exist", player)
		return false
	}

	for _, userPermission := range user.Permissions {
		if userPermission == permission {
			return true
		}

		if strings.Split(userPermission, ".")[0] == strings.Split(permission, ".")[0] && strings.Split(userPermission, ".")[1] == "*" {
			return true
		}
	}

	for _, userGroup := range user.Groups {
		if p.GroupHasPermission(userGroup, permission) {
			return true
		}
	}

	return false
}

func (p *Permissions) UserAddPermission(ctx context.Context, UUID string, permission string) error {
	p.m.Lock()
	UUID = uuid.Normalize(UUID)
	user := p.Users[UUID]
	user.Permissions = append(user.Permissions, permission)
	p.Users[UUID] = user
	p.m.Unlock()

	return p.saveUsers(ctx)
}

func (p *Permissions) GroupAddPermission(ctx context.Context, name string, permission string) error {
	p.m.Lock()
	group := p.Groups[name]
	group.Permissions = append(group.Permissions, permission)
	p.Groups[name] = group
	p.m.Unlock()

	return p.saveGroups(ctx)
}

func (p *Permissions) UserRemovePermission(ctx context.Context, UUID string, permission string) error {
	p.m.Lock()
	UUID = uuid.Normalize(UUID)
	user := p.Users[UUID]

	user.Permissions = slices.DeleteFunc(user.Permissions, func(s string) bool {
		return s == permission
	})

	p.Users[UUID] = user
	p.m.Unlock()

	return p.saveUsers(ctx)
}

func (p *Permissions) GroupRemovePermission(ctx context.Context, name string, permission string) error {
	p.m.Lock()
	group := p.Groups[name]

	group.Permissions = slices.DeleteFunc(group.Permissions, func(s string) bool {
		return s == permission
	})

	p.Groups[name] = group
	p.m.Unlock()

	return p.saveGroups(ctx)
}
