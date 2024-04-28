package permissions

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util/uuid"
)

type PermissionFile struct {
	Users  map[string]PermissionUser  `json:"users"`
	Groups map[string]PermissionGroup `json:"groups"`
}

var _ Permissions = &FSPermissions{}

type FSPermissions struct {
	name string
	file PermissionFile
	m    sync.RWMutex
}

type PermissionUser struct {
	Groups      []string `json:"groups"`
	Permissions []string `json:"permissions"`
}

type PermissionGroup struct {
	Prefix      string   `json:"prefix"`
	Weight      uint8    `json:"weight"`
	Permissions []string `json:"permissions"`
}

func ReadFile(file string) (*FSPermissions, error) {
	w := &FSPermissions{
		name: file,
		file: PermissionFile{
			Users:  make(map[string]PermissionUser),
			Groups: make(map[string]PermissionGroup),
		},
	}

	if err := w.Reload(context.Background()); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *FSPermissions) Reload(ctx context.Context) error {
	fd, err := os.OpenFile(w.name, os.O_RDONLY, 0755)
	if err != nil {
		if os.IsNotExist(err) {
			// Save the default permissions file
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

func (w *FSPermissions) Save() error {
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

func (p *FSPermissions) GetGroup(name string) (PermissionGroup, bool) {
	group, exists := p.file.Groups[name]
	return group, exists
}

func (p *FSPermissions) GroupNames() []string {
	return util.MapKeys(p.file.Groups)
}

func (p *FSPermissions) GetUsers() []string {
	return util.MapKeys(p.file.Users)
}

func (p *FSPermissions) UserPermissions(name string) ([]string, bool) {
	user, ok := p.file.Users[name]
	if !ok {
		return make([]string, 0), false
	}

	return user.Permissions, true
}

func (p *FSPermissions) UserGroups(name string) ([]string, bool) {
	user, ok := p.file.Users[name]
	if !ok {
		return make([]string, 0), false
	}

	return user.Groups, true
}

func (p *FSPermissions) GroupHasPermission(name string, permission string) bool {
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

func (p *FSPermissions) UserHasPermission(player string, permission string) bool {
	player = uuid.Normalize(player)

	user, ok := p.file.Users[player]
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

func (p *FSPermissions) UserAddPermission(_ctx context.Context, UUID string, permission string) error {
	p.m.Lock()
	UUID = uuid.Normalize(UUID)
	user := p.file.Users[UUID]
	user.Permissions = append(user.Permissions, permission)
	p.file.Users[UUID] = user
	p.m.Unlock()
	return p.Save()
}

func (p *FSPermissions) GroupAddPermission(_ctx context.Context, name string, permission string) error {
	p.m.Lock()
	group := p.file.Groups[name]
	group.Permissions = append(group.Permissions, permission)
	p.file.Groups[name] = group
	p.m.Unlock()

	return p.Save()
}

func (p *FSPermissions) UserRemovePermission(_ctx context.Context, UUID string, permission string) error {
	p.m.Lock()
	UUID = uuid.Normalize(UUID)
	user := p.file.Users[UUID]

	user.Permissions = slices.DeleteFunc(user.Permissions, func(s string) bool {
		return s == permission
	})

	p.file.Users[UUID] = user
	p.m.Unlock()

	return p.Save()
}

func (p *FSPermissions) GroupRemovePermission(_ctx context.Context, name string, permission string) error {
	p.m.Lock()
	group := p.file.Groups[name]

	group.Permissions = slices.DeleteFunc(group.Permissions, func(s string) bool {
		return s == permission
	})

	p.file.Groups[name] = group
	p.m.Unlock()

	return p.Save()
}
