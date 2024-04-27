package permissions

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util/uuid"
	"go.minekube.com/brigodier"
	"go.minekube.com/common/minecraft/color"
	"go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type PermissionsPlugin struct {
	prx         *proxy.Proxy
	permissions *FPermission
}

func NewPlugin(prx *proxy.Proxy, permissions *FPermission) (*PermissionsPlugin, error) {
	return &PermissionsPlugin{
		prx:         prx,
		permissions: permissions,
	}, nil
}

func (p *PermissionsPlugin) Reload() error {
	if err := p.permissions.Reload(); err != nil {
		return err
	}

	return nil
}

func (p *PermissionsPlugin) Init() error {
	if err := p.Reload(); err != nil {
		return err
	}

	p.prx.Command().Register(p.command())

	return nil
}

func New(permissions *FPermission) (proxy.Plugin, error) {
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
				Then(brigodier.Literal("list").
					Executes(p.ListPermission(User))),
			),
		).
		Then(brigodier.
			Literal("group").
			Then(brigodier.
				Argument("name", brigodier.String).
				Then(brigodier.
					Literal("list").
					Executes(p.ListPermission(Group)),
				),
			).
			Executes(UsageCommand())).
		Then(brigodier.
			Literal("reload").
			Executes(p.reloadCommand())).
		Executes(p.helpCommand())
}

type PermissionListType string

const (
	User  PermissionListType = "User"
	Group PermissionListType = "Group"
)

func (p *PermissionsPlugin) ListPermission(_type PermissionListType) brigodier.Command {
	return command.Command(func(c *command.Context) error {
		name := c.String("name")

		switch _type {
		case User:
			UUID, err := uuid.UsernameToUUID(name)
			if err != nil {
				return c.SendMessage(&component.Text{
					Content: "Error while connecting to Mojang Servers! (maybe they are off)",
					S:       component.Style{Color: color.Red},
				})
			}
			UUID = uuid.Normalize(UUID)

			permissionList, exists := p.permissions.UserPermissions(UUID)
			if !exists {
				log.Printf("WARN: User %s doesn't exist", name)
				return c.SendMessage(&component.Text{
					// TODO: Change this message
					Content: "This user doesn't have any permissions set",
					S:       component.Style{Color: color.Red},
				})
			}

			permissions := strings.Join(permissionList, ", ")

			return c.SendMessage(&component.Text{
				Extra: []component.Component{
					&component.Text{Content: name + "'s Permissions (" + fmt.Sprint(len(permissionList)) + "): ", S: component.Style{Color: color.Blue}},
					&component.Text{Content: permissions, S: component.Style{Color: color.Green}},
				},
			})
		case Group:
			if !slices.Contains(p.permissions.GetGroups(), name) {
				return c.SendMessage(&component.Text{
					Content: "This group doesn't exist!",
					S:       component.Style{Color: color.Red},
				})
			}

			permissionList, exists := p.permissions.GroupPermissions(name)
			if !exists {
				log.Printf("WARN: Group %s doesn't exist", name)
				return c.SendMessage(&component.Text{
					// TODO: Change this message
					Content: "This group doesn't exist, idk what happened",
					S:       component.Style{Color: color.Red},
				})
			}

			permissions := strings.Join(permissionList, ", ")

			return c.SendMessage(&component.Text{
				Extra: []component.Component{
					&component.Text{Content: name + "'s Permissions (" + fmt.Sprint(len(permissionList)) + "): ", S: component.Style{Color: color.Blue}},
					&component.Text{Content: permissions, S: component.Style{Color: color.Green}},
				},
			})
		}

		return nil
	})
}

func (p *PermissionsPlugin) helpCommand() brigodier.Command {
	return command.Command(func(c *command.Context) error {
		return c.SendMessage(&component.Text{
			Extra: []component.Component{
				&component.Text{Content: "ᴘᴇʀᴍѕ ", S: component.Style{Color: color.Green, Bold: component.True}},
				&component.Text{Content: "Running", S: component.Style{Color: color.Green, Bold: component.False}},
				&component.Text{Content: " Permissions v0.1.1-BETA\n", S: component.Style{Color: color.LightPurple, Bold: component.False}},
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

func UsageCommand() brigodier.Command {
	usage := component.Text{Content: "Usage: /whitelist <add/remove/enable/disable> <user>", S: component.Style{Color: color.Red}}

	return command.Command(func(c *command.Context) error {
		return c.SendMessage(&usage)
	})
}

func (p *PermissionsPlugin) reloadCommand() brigodier.Command {
	reloaded := component.Text{Content: "Reloaded permissions successfully!", S: component.Style{Color: color.Green}}

	return command.Command(func(c *command.Context) error {
		if err := p.permissions.Reload(); err != nil {
			return err
		}

		return c.SendMessage(&reloaded)
	})
}
