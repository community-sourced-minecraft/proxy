package permissions

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

type PermissionFile struct {
	Users  []PermissionUser  `json:"users"`
	Groups []PermissionGroup `json:"groups"`
}

type FPermission struct {
	name string
	file PermissionFile
	m    sync.RWMutex
}

type PermissionUser struct {
	uuid        string
	group       []PermissionGroup
	permissions []string
}

type PermissionGroup struct {
	name        string
	prefix      string
	weight      uint8
	permissions []string
}

func ReadFile(file string) (*FPermission, error) {
	w := &FPermission{
		name: file,
		file: PermissionFile{
			Users:  make([]PermissionUser, 0),
			Groups: make([]PermissionGroup, 0),
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

func (p *FPermission) GetGroups() []PermissionGroup {
	return p.file.Groups
}

func (p *FPermission) GetGroupsAsString() []string {
	list := make([]string, 0)

	for _, group := range p.GetGroups() {
		list = append(list, group.name)
	}

	return list
}

func (p *FPermission) GroupPermissions(name string) []string {
	list := make([]string, 0)

	for _, group := range p.GetGroups() {
		if group.name == name {
			list = append(list, group.permissions...)
		}
	}

	return list
}

func (p *FPermission) GroupHasPermission(name string, permission string) bool {
	result := false

	for _, _permission := range p.GroupPermissions(name) {
		if _permission == permission {
			result = true
		}

		if strings.Split(_permission, ".")[0] == strings.Split(permission, ".")[0] && strings.Split(_permission, ".")[1] == "*" {
			result = true
		}
	}

	return result
}

func (p *FPermission) UserHasPermission(uuid string, permission string) bool {
	result := false

	for _, user := range p.file.Users {
		for _, userGroup := range user.group {
			if p.GroupHasPermission(userGroup.name, permission) {
				result = true
			}
		}

		for _, userPermission := range user.permissions {
			if userPermission == permission {
				result = true
			}

			if strings.Split(userPermission, ".")[0] == strings.Split(permission, ".")[0] && strings.Split(userPermission, ".")[1] == "*" {
				result = true
			}
		}
	}

	return result
}
