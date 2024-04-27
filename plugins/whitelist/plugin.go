package whitelist

import (
	"context"
	"fmt"
	"strings"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util/uuid"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/permissions"
	"github.com/robinbraemer/event"
	"go.minekube.com/brigodier"
	"go.minekube.com/common/minecraft/color"
	"go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type WhitelistPlugin struct {
	whitelist   *Whitelist
	permissions *permissions.FPermission
}

func NewPlugin(permissions *permissions.FPermission) (*WhitelistPlugin, error) {
	whitelist, err := ReadWhitelist("whitelist.json")
	if err != nil {
		return nil, err
	}

	return &WhitelistPlugin{
		whitelist:   whitelist,
		permissions: permissions,
	}, nil
}

func (p *WhitelistPlugin) Reload() error {
	if err := p.whitelist.Reload(); err != nil {
		return err
	}

	return nil
}

func (p *WhitelistPlugin) Init(prx *proxy.Proxy) error {
	if err := p.Reload(); err != nil {
		return err
	}

	event.Subscribe(prx.Event(), 0, p.onPostConnectEvent)
	prx.Command().Register(p.command())

	return nil
}

func (p *WhitelistPlugin) onPostConnectEvent(e *proxy.ServerPostConnectEvent) {
	uuid := e.Player().GameProfile().ID

	if !p.whitelist.Contains(strings.Replace(uuid.String(), "-", "", -1)) && p.whitelist.Enabled() {
		e.Player().Disconnect(&component.Text{
			Content: "You are not whitelisted!",
			S:       component.Style{Color: color.Red},
		})
	}
}

func New(permissions *permissions.FPermission) (proxy.Plugin, error) {
	return proxy.Plugin{
		Name: "Whitelist",
		Init: func(ctx context.Context, px *proxy.Proxy) error {
			plugin, err := NewPlugin(permissions)
			if err != nil {
				return err
			}

			return plugin.Init(px)
		},
	}, nil
}

func (p *WhitelistPlugin) command() brigodier.LiteralNodeBuilder {
	return brigodier.Literal("whitelist").
		Then(brigodier.
			Literal("status").
			Executes(p.statusCommand())).
		Then(brigodier.
			Literal("help").
			Executes(p.UsageWhitelist())).
		Then(brigodier.
			Literal("enable").
			Executes(p.enableCommand())).
		Then(brigodier.
			Literal("disable").
			Executes(p.disableCommand())).
		Then(brigodier.
			Literal("list").
			Executes(p.listCommand())).
		Then(brigodier.
			Literal("reload").
			Executes(p.reloadCommand())).
		Then(brigodier.
			Literal("add").
			Executes(p.UsageWhitelist()).
			Then(brigodier.
				Argument("user", brigodier.String).
				Executes(p.addCommand())),
		).
		Then(brigodier.
			Literal("remove").
			Executes(p.UsageWhitelist()).
			Then(brigodier.
				Argument("user", brigodier.String).
				Executes(p.removeCommand()))).
		Executes(p.statusCommand())
}

func (p *WhitelistPlugin) UsageWhitelist() brigodier.Command {
	usage := component.Text{Content: "Usage: /whitelist <add/remove/enable/disable> <user>", S: component.Style{Color: color.Red}}

	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "whitelist.add") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}
		return c.SendMessage(&usage)
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

func (p *WhitelistPlugin) addCommand() brigodier.Command {
	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "whitelist.add") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}

		username := c.Arguments["user"].Result.(string)
		uuid, err := uuid.UsernameToUUID(username)

		if err != nil {
			return p.UsageWhitelist().Run(c.CommandContext)
		}

		if p.whitelist.Contains(uuid) {
			return c.SendMessage(&component.Text{
				Content: username + " is already on whitelist!",
				S:       component.Style{Color: color.Red},
			})
		}

		if err := p.whitelist.Add(uuid); err != nil {
			return err
		}

		return c.SendMessage(&component.Text{Content: "Added " + username + " to whitelist!", S: component.Style{Color: color.Green}})
	})
}

func (p *WhitelistPlugin) removeCommand() brigodier.Command {
	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "whitelist.add") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}
		username := c.Arguments["user"].Result.(string)
		uuid, err := uuid.UsernameToUUID(username)

		if err != nil {
			return p.UsageWhitelist().Run(c.CommandContext)
		}

		if !p.whitelist.Contains(uuid) {
			return c.SendMessage(&component.Text{
				Content: username + " is not on whitelist!",
				S:       component.Style{Color: color.Red},
			})
		}

		if err := p.whitelist.Remove(uuid); err != nil {
			return err
		}

		return c.SendMessage(&component.Text{Content: "Removed " + username + " from whitelist!", S: component.Style{Color: color.Green}})
	})
}

func (p *WhitelistPlugin) reloadCommand() brigodier.Command {
	reloaded := component.Text{Content: "Reloaded command successfully!", S: component.Style{Color: color.Green}}

	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "whitelist.add") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}

		if err := p.whitelist.Reload(); err != nil {
			return err
		}

		return c.SendMessage(&reloaded)
	})
}

func (p *WhitelistPlugin) listCommand() brigodier.Command {
	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "whitelist.add") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}
		users := strings.Builder{}

		for i, id := range p.whitelist.AllWhitelisted() {
			str, err := uuid.UUIDtoUsername(id) // sorry mojank
			if err != nil {
				str = id
			}

			if i != 0 {
				users.WriteString(", ")
			}

			users.WriteString(str)
		}

		return c.SendMessage(&component.Text{
			Content: fmt.Sprintf("Whitelisted users (%d): %s", len(p.whitelist.AllWhitelisted()), users.String()),
			S:       component.Style{Color: color.Green},
		})
	})
}

func (p *WhitelistPlugin) enableCommand() brigodier.Command {
	alreadyEnabled := component.Text{Content: "Whitelist is already on", S: component.Style{Color: color.Red}}
	enabled := component.Text{Content: "Enabled whitelist!", S: component.Style{Color: color.Green}}

	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "whitelist.add") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}
		if p.whitelist.Enabled() {
			return c.SendMessage(&alreadyEnabled)
		}

		if err := p.whitelist.Enable(); err != nil {
			return err
		}

		return c.SendMessage(&enabled)
	})
}

func (p *WhitelistPlugin) disableCommand() brigodier.Command {
	alreadyDisabled := component.Text{Content: "Whitelist is already off", S: component.Style{Color: color.Red}}
	disabled := component.Text{Content: "Disabled whitelist!", S: component.Style{Color: color.Green}}

	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "whitelist.add") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}
		if !p.whitelist.Enabled() {
			return c.SendMessage(&alreadyDisabled)
		}

		if err := p.whitelist.Disable(); err != nil {
			return err
		}

		return c.SendMessage(&disabled)
	})
}

func (p *WhitelistPlugin) statusCommand() brigodier.Command {
	base := component.Text{Content: "Whitelist is ", S: component.Style{Color: color.White}}
	enabled := component.Text{Content: "enabled", S: component.Style{Color: color.Green}}
	disabled := component.Text{Content: "disabled", S: component.Style{Color: color.Red}}

	return command.Command(func(c *command.Context) error {
		if !p.permissions.UserHasPermission(c.Source.(proxy.Player).ID().String(), "whitelist.add") {
			return PermissionMissingCommand().Run(c.CommandContext)
		}
		var state component.Text
		if p.whitelist.Enabled() {
			state = enabled
		} else {
			state = disabled
		}

		return c.SendMessage(util.Join(&base, &state))
	})
}
