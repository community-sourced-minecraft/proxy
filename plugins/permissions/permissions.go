package permissions

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util/uuid"
)

type PermissionFile struct {
	Users  map[string]PermissionUser  `json:"users"`
	Groups map[string]PermissionGroup `json:"groups"`
}

type FPermission struct {
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

func ReadFile(file string) (*FPermission, error) {
	w := &FPermission{
		name: file,
		file: PermissionFile{
			Users:  make(map[string]PermissionUser),
			Groups: make(map[string]PermissionGroup),
		},
	}

	if err := w.Reload(); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *FPermission) Reload() error {
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

func (w *FPermission) Save() error {

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

func (p *FPermission) GetGroups() []string {
	return util.MapKeys(p.file.Groups)
}

func (p *FPermission) GetUsers() []string {
	return util.MapKeys(p.file.Users)
}

func (p *FPermission) GroupPermissions(name string) ([]string, bool) {
	group, ok := p.file.Groups[name]
	if !ok {
		return make([]string, 0), false
	}

	return group.Permissions, true
}

func (p *FPermission) UserPermissions(name string) ([]string, bool) {
	user, ok := p.file.Users[name]
	if !ok {
		return make([]string, 0), false
	}

	return user.Permissions, true
}

func (p *FPermission) UserGroups(name string) ([]string, bool) {
	user, ok := p.file.Users[name]
	if !ok {
		return make([]string, 0), false
	}

	return user.Groups, true
}

func (p *FPermission) GroupHasPermission(name string, permission string) bool {
	perms, exists := p.GroupPermissions(name)
	if !exists {
		log.Printf("WARN: Group %s does not exist", name)
		return false
	}

	for _, _permission := range perms {
		if _permission == permission {
			return true
		}

		if strings.Split(_permission, ".")[0] == strings.Split(permission, ".")[0] && strings.Split(_permission, ".")[1] == "*" {
			return true
		}
	}

	return false
}

func (p *FPermission) UserHasPermission(player string, permission string) bool {
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
