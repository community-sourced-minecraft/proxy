package permissions

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"go.minekube.com/brigodier"
	"go.minekube.com/common/minecraft/color"
	"go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type PermissionsPlugin struct {
	permissions *FPermission
}

func NewPlugin() (*PermissionsPlugin, error) {
	_permission, err := ReadFile("permissions.json")
	if err != nil {
		return nil, err
	}

	return &PermissionsPlugin{
		permissions: _permission,
	}, nil
}

func (p *PermissionsPlugin) Reload() error {
	if err := p.permissions.Reload(); err != nil {
		return err
	}

	return nil
}

func (p *PermissionsPlugin) Init(prx *proxy.Proxy) error {
	if err := p.Reload(); err != nil {
		return err
	}

	prx.Command().Register(p.command())

	return nil
}

var Plugin = proxy.Plugin{
	Name: "Permissions",
	Init: func(ctx context.Context, px *proxy.Proxy) error {
		plugin, err := NewPlugin()
		if err != nil {
			return err
		}

		return plugin.Init(px)
	},
}

func (p *PermissionsPlugin) command() brigodier.LiteralNodeBuilder {
	return brigodier.Literal("permissions").
		Then(brigodier.
			Literal("user")).
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

		case Group:
			if !slices.Contains(p.permissions.GetGroupsAsString(), name) {
				return c.SendMessage(&component.Text{
					Content: "This group doesn't exist",
					S:       component.Style{Color: color.Red},
				})
			}

			permissions := strings.Builder{}
			permissionList := p.permissions.GroupPermissions(name)

			for i, permission := range permissionList {
				if i != 0 {
					permissions.WriteString(", ")
				}

				permissions.WriteString(permission)
			}

			return c.SendMessage(&component.Text{
				Extra: []component.Component{
					&component.Text{Content: name + "'s Permissions (" + fmt.Sprint(len(permissionList)) + ")", S: component.Style{Color: color.Blue}},
					&component.Text{Content: permissions.String(), S: component.Style{Color: color.Green}},
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
				&component.Text{Content: ">", S: component.Style{Color: color.Blue, Bold: component.False}},
				&component.Text{Content: "/permissions user\n", S: component.Style{Color: color.LightPurple, Bold: component.False}},
				&component.Text{Content: ">", S: component.Style{Color: color.Blue, Bold: component.False}},
				&component.Text{Content: "/permissions group\n", S: component.Style{Color: color.LightPurple, Bold: component.False}},
				&component.Text{Content: ">", S: component.Style{Color: color.Blue, Bold: component.False}},
				&component.Text{Content: "/permissions reload\n", S: component.Style{Color: color.LightPurple, Bold: component.False}},
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
