package permissions

import (
	"context"
	"fmt"
	"log"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util/uuid"
	"go.minekube.com/brigodier"
	"go.minekube.com/common/minecraft/color"
	"go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type Permissions interface {
	Reload(ctx context.Context) error

	UserPermissions(UUID string) ([]string, bool)
	UserGroups(UUID string) ([]string, bool)
	UserHasPermission(UUID, permission string) bool
	UserAddPermission(ctx context.Context, UUID, permission string) error
	UserRemovePermission(ctx context.Context, UUID, permission string) error

	GroupNames() []string
	GetGroup(group string) (PermissionGroup, bool)
	GroupHasPermission(group, permission string) bool
	GroupAddPermission(ctx context.Context, group, permission string) error
	GroupRemovePermission(ctx context.Context, group, permission string) error
}

type PermissionsPlugin struct {
	prx         *proxy.Proxy
	permissions Permissions
}

func NewPlugin(prx *proxy.Proxy, permissions Permissions) (*PermissionsPlugin, error) {
	return &PermissionsPlugin{
		prx:         prx,
		permissions: permissions,
	}, nil
}

func (p *PermissionsPlugin) Init() error {
	if err := p.Reload(); err != nil {
		return err
	}

	p.prx.Command().Register(p.command())

	return nil
}

func (p *PermissionsPlugin) Reload() error {
	if err := p.permissions.Reload(context.Background()); err != nil {
		return err
	}

	return nil
}

func New(permissions Permissions) (proxy.Plugin, error) {
	return proxy.Plugin{
		Name: "Permissions",
		Init: func(ctx context.Context, prx *proxy.Proxy) error {
			plugin, err := NewPlugin(prx, permissions)
			if err != nil {
				return err
			}

			return plugin.Init()
		},
	}, nil
}

func (p *PermissionsPlugin) command() brigodier.LiteralNodeBuilder {
	return brigodier.Literal("permissions").
		Then(brigodier.
			Literal("user").
			Then(brigodier.
				Argument("name", brigodier.String).
				Suggests(command.SuggestFunc(func(c *command.Context, b *brigodier.SuggestionsBuilder) *brigodier.Suggestions {
					for _, user := range p.prx.Players() {
						// TODO: resolve UUID to username
						b.Suggest(user.Username())
					}
					return b.Build()
				})).
				Then(brigodier.Literal("info").
					Executes(p.InfoCommand(PermissionTypeUser))).
				Then(brigodier.Literal("remove").Then(brigodier.Argument("permission", brigodier.String).Executes(p.removeCommand(PermissionTypeUser)))).
				Then(brigodier.Literal("add").Then(brigodier.Argument("permission", brigodier.String).Executes(p.addCommand(PermissionTypeUser)))),
			),
		).
		Then(brigodier.
			Literal("group").
			Then(brigodier.
				Argument("name", brigodier.String).
				Then(brigodier.
					Literal("info").
					Executes(p.InfoCommand(PermissionTypeGroup)),
				).
				Then(brigodier.Literal("remove").Then(brigodier.Argument("permission", brigodier.String).Executes(p.removeCommand(PermissionTypeGroup)))).
				Then(brigodier.Literal("add").Then(brigodier.Argument("permission", brigodier.String).Executes(p.addCommand(PermissionTypeGroup)))),
			).
			Executes(p.helpCommand())).
		Then(brigodier.
			Literal("reload").
			Executes(p.reloadCommand())).
		Executes(p.helpCommand())
}

type PermissionListType string

const (
	PermissionTypeUser  PermissionListType = "User"
	PermissionTypeGroup PermissionListType = "Group"
)

func (p *PermissionsPlugin) InfoCommand(_type PermissionListType) brigodier.Command {
	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "permissions.info") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}
		name := c.String("name")

		switch _type {
		case PermissionTypeUser:
			UUID, err := uuid.UsernameToUUID(name)
			if err != nil {
				return c.SendMessage(&component.Text{
					Content: "Error while connecting to Mojang Servers! (maybe they are off)",
					S:       component.Style{Color: color.Red},
				})
			}
			UUID = uuid.Normalize(UUID)

			groupsMsg := []component.Component{
				&component.Text{Content: "\n >", S: component.Style{Color: color.Yellow}},
				&component.Text{Content: " default", S: component.Style{Color: color.White}},
			}
			permissionMsg := []component.Component{
				&component.Text{Content: " >", S: component.Style{Color: color.Yellow}},
				&component.Text{Content: " " + name + " doesn't have any permissions set.", S: component.Style{Color: color.White}},
			}

			groups, ok := p.permissions.UserGroups(UUID)

			if ok {
				for _, group := range groups {
					groupsMsg = append(groupsMsg, &component.Text{Content: "\n > ", S: component.Style{Color: color.Yellow}}, &component.Text{Content: group, S: component.Style{Color: color.White}})
				}
			}

			permissions, ok := p.permissions.UserPermissions(UUID)

			if len(permissionMsg) != 0 && ok {
				permissionMsg = []component.Component{&component.Text{Content: "\nPermissions: ", S: component.Style{Color: color.Yellow}}}

				for _, permission := range permissions {
					permissionMsg = append(permissionMsg, &component.Text{Content: "\n > ", S: component.Style{Color: color.Yellow}}, &component.Text{Content: permission, S: component.Style{Color: color.White}})
				}
			}

			return c.SendMessage(&component.Text{
				Extra: []component.Component{
					&component.Text{Content: "\n"},
					&component.Text{Content: "User Info: ", S: component.Style{Color: color.Yellow}},
					&component.Text{Content: name + "\n", S: component.Style{Color: color.White}},
					&component.Text{Content: "UUID: ", S: component.Style{Color: color.Yellow}},
					&component.Text{Content: UUID + "\n", S: component.Style{Color: color.White}},
					&component.Text{Content: "Groups: ", S: component.Style{Color: color.Yellow}},
					&component.Text{Extra: groupsMsg},
					&component.Text{Extra: permissionMsg},
				},
			})
		case PermissionTypeGroup:
			group, exists := p.permissions.GetGroup(name)
			if !exists {
				log.Printf("WARN: Group %s doesn't exist", name)
				return c.SendMessage(&component.Text{
					// TODO: Change this message
					S: component.Style{Color: color.Red},
					Extra: []component.Component{
						&component.Text{Content: "ᴘᴇʀᴍѕ ", S: component.Style{Color: color.Green, Bold: component.True}},
						&component.Text{Content: "This group doesn't exist!", S: component.Style{Color: color.Red}},
					},
				})
			}

			permissionMsg := []component.Component{
				&component.Text{Content: " >", S: component.Style{Color: color.Yellow}},
				&component.Text{Content: " " + name + " doesn't have any permissions set.", S: component.Style{Color: color.White}},
			}

			if len(permissionMsg) != 0 {
				permissionMsg = []component.Component{&component.Text{Content: "\nPermissions: ", S: component.Style{Color: color.Yellow}}}

				for _, permission := range group.Permissions {
					permissionMsg = append(permissionMsg, &component.Text{Content: "\n > ", S: component.Style{Color: color.Yellow}}, &component.Text{Content: permission, S: component.Style{Color: color.White}})
				}
			}

			return c.SendMessage(&component.Text{
				Extra: []component.Component{
					&component.Text{Content: "\n"},
					&component.Text{Content: "Group Info: ", S: component.Style{Color: color.Yellow}},
					&component.Text{Content: name, S: component.Style{Color: color.White}},
					&component.Text{Content: "\nWeight: ", S: component.Style{Color: color.Yellow}},
					&component.Text{Content: fmt.Sprint(group.Weight), S: component.Style{Color: color.White}},
					&component.Text{Content: "\nPrefix: ", S: component.Style{Color: color.Yellow}},
					&component.Text{Content: group.Prefix, S: component.Style{Color: color.White}},
					&component.Text{Extra: permissionMsg},
				},
			})
		}

		return nil
	})
}

func (p *PermissionsPlugin) helpCommand() brigodier.Command {
	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "permissions.help") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}
		return c.SendMessage(&component.Text{
			Extra: []component.Component{
				&component.Text{Content: "ᴘᴇʀᴍѕ ", S: component.Style{Color: color.Green, Bold: component.True}},
				&component.Text{Content: "Running", S: component.Style{Color: color.Green, Bold: component.False}},
				&component.Text{Content: " Permissions v0.2.1-BETA\n", S: component.Style{Color: color.LightPurple, Bold: component.False}},
				&component.Text{Content: "> ", S: component.Style{Color: color.Blue, Bold: component.False}},
				&component.Text{Content: "/permissions user\n", S: component.Style{Color: color.LightPurple, Bold: component.False}},
				&component.Text{Content: "> ", S: component.Style{Color: color.Blue, Bold: component.False}},
				&component.Text{Content: "/permissions group\n", S: component.Style{Color: color.LightPurple, Bold: component.False}},
				&component.Text{Content: "> ", S: component.Style{Color: color.Blue, Bold: component.False}},
				&component.Text{Content: "/permissions reload", S: component.Style{Color: color.LightPurple, Bold: component.False}},
			},
		})
	})
}

func (p *PermissionsPlugin) addCommand(_type PermissionListType) brigodier.Command {
	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "permissions.add") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}

		permission := c.Arguments["permission"].Result.(string)
		name := c.Arguments["name"].Result.(string)

		errorMsg := &component.Text{
			Extra: []component.Component{
				&component.Text{Content: "ᴘᴇʀᴍѕ ", S: component.Style{Color: color.Green, Bold: component.True}},
				&component.Text{Content: "Permission ", S: component.Style{Color: color.Red}},
				&component.Text{Content: permission, S: component.Style{Color: color.LightPurple}},
				&component.Text{Content: " is already set for ", S: component.Style{Color: color.Red}},
				&component.Text{Content: name, S: component.Style{Color: color.LightPurple}},
			},
		}

		switch _type {
		case PermissionTypeUser:
			UUID, err := uuid.UsernameToUUID(name)
			if err != nil {
				return err
			}

			UUID = uuid.Normalize(UUID)
			res := p.permissions.UserHasPermission(UUID, permission)
			if res {
				return c.SendMessage(errorMsg)
			}

			p.permissions.UserAddPermission(c.Context, UUID, permission)
		case PermissionTypeGroup:
			res := p.permissions.GroupHasPermission(name, permission)
			if res {
				return c.SendMessage(errorMsg)
			}

			p.permissions.GroupAddPermission(c.Context, name, permission)
		}

		return c.SendMessage(&component.Text{
			Extra: []component.Component{
				&component.Text{Content: "ᴘᴇʀᴍѕ ", S: component.Style{Color: color.Green, Bold: component.True}},
				&component.Text{Content: "Set ", S: component.Style{Color: color.Green}},
				&component.Text{Content: permission, S: component.Style{Color: color.LightPurple}},
				&component.Text{Content: " for ", S: component.Style{Color: color.Green}},
				&component.Text{Content: name, S: component.Style{Color: color.LightPurple}},
			},
		})
	})
}

func (p *PermissionsPlugin) removeCommand(_type PermissionListType) brigodier.Command {
	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "permissions.remove") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}

		permission := c.Arguments["permission"].Result.(string)
		name := c.Arguments["name"].Result.(string)

		errorMsg := &component.Text{
			Extra: []component.Component{
				&component.Text{Content: "ᴘᴇʀᴍѕ ", S: component.Style{Color: color.Green, Bold: component.True}},
				&component.Text{Content: "Permission ", S: component.Style{Color: color.Red}},
				&component.Text{Content: permission, S: component.Style{Color: color.LightPurple}},
				&component.Text{Content: " doesn't exists for ", S: component.Style{Color: color.Red}},
				&component.Text{Content: name, S: component.Style{Color: color.LightPurple}},
			},
		}

		switch _type {
		case PermissionTypeUser:
			UUID, err := uuid.UsernameToUUID(name)
			if err != nil {
				return err
			}

			UUID = uuid.Normalize(UUID)
			res := p.permissions.UserHasPermission(UUID, permission)
			if !res {
				return c.SendMessage(errorMsg)
			}

			p.permissions.UserRemovePermission(c.Context, UUID, permission)
		case PermissionTypeGroup:
			res := p.permissions.GroupHasPermission(name, permission)
			if !res {
				return c.SendMessage(errorMsg)
			}

			p.permissions.GroupRemovePermission(c.Context, name, permission)
		}

		return c.SendMessage(&component.Text{
			Extra: []component.Component{
				&component.Text{Content: "ᴘᴇʀᴍѕ ", S: component.Style{Color: color.Green, Bold: component.True}},
				&component.Text{Content: "Removed ", S: component.Style{Color: color.Green}},
				&component.Text{Content: permission, S: component.Style{Color: color.LightPurple}},
				&component.Text{Content: " for ", S: component.Style{Color: color.Green}},
				&component.Text{Content: name, S: component.Style{Color: color.LightPurple}},
			},
		})
	})
}

func (p *PermissionsPlugin) reloadCommand() brigodier.Command {
	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "permissions.reload") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}
		if err := p.permissions.Reload(c.Context); err != nil {
			return err
		}

		return c.SendMessage(&component.Text{
			Extra: []component.Component{
				&component.Text{Content: "ᴘᴇʀᴍѕ ", S: component.Style{Color: color.Green, Bold: component.True}},
				&component.Text{Content: "Reloaded permissions successfully!", S: component.Style{Color: color.Green}},
			},
		})
	})
}

func PermissionMissingCommand() brigodier.Command {
	usage := component.Text{
		Content: "You don't have the permission to do that!",
		S:       component.Style{Color: color.Red},
	}

	return command.Command(func(c *command.Context) error {
		return c.SendMessage(&usage)
	})
}
