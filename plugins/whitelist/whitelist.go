package whitelist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util/uuid"
	"github.com/robinbraemer/event"
	"go.minekube.com/brigodier"
	"go.minekube.com/common/minecraft/color"
	"go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

// TODO: Refactor to custom plugin struct
var whitelist WhitelistFile
var Plugin = proxy.Plugin{
	Name: "Whitelist",
	Init: func(ctx context.Context, p *proxy.Proxy) error {
		if err := ReloadWhitelist(); err != nil {
			return err
		}

		event.Subscribe(p.Event(), 0, func(e *proxy.ServerPostConnectEvent) {
			uuid := e.Player().GameProfile().ID
			if !slices.Contains(whitelist.Whitelisted, strings.Replace(uuid.String(), "-", "", -1)) && whitelist.Enabled {
				e.Player().Disconnect(&component.Text{
					Content: "You are not whitelisted!",
					S:       component.Style{Color: color.Red},
				})
			}
		})

		p.Command().Register(whitelistCommand())

		return nil
	},
}

func whitelistCommand() brigodier.LiteralNodeBuilder {
	whitelistAddCommand := command.Command(func(c *command.Context) error {
		uuid, err := uuid.UsernameToUUID(c.Arguments["user"].Result.(string))
		if err != nil {
			return UsageWhitelist().Run(c.CommandContext)
		}

		res, err := os.OpenFile("whitelist.json", os.O_RDWR|os.O_TRUNC, 0755)
		if err != nil {
			return c.SendMessage(&component.Text{
				Content: "Error while reading whitelist.json",
			})
		}
		defer res.Close()

		if slices.Contains(whitelist.Whitelisted, uuid) {
			return c.SendMessage(&component.Text{
				Content: c.Arguments["user"].Result.(string) + " is already on whitelist!",
				S:       component.Style{Color: color.Red},
			})
		}

		whitelist.Whitelisted = append(whitelist.Whitelisted, uuid)

		if err := json.NewEncoder(res).Encode(whitelist); err != nil {
			return err
		}

		if err := ReloadWhitelist(); err != nil {
			return err
		}

		return c.SendMessage(&component.Text{Content: "Added " + c.Arguments["user"].Result.(string) + " to whitelist!", S: component.Style{Color: color.Green}})
	})

	whitelistRemoveCommand := command.Command(func(c *command.Context) error {
		uuid, err := uuid.UsernameToUUID(c.Arguments["user"].Result.(string))
		if err != nil {
			return UsageWhitelist().Run(c.CommandContext)
		}

		res, err := os.OpenFile("whitelist.json", os.O_RDWR|os.O_TRUNC, 0755)
		if err != nil {
			return c.SendMessage(&component.Text{
				Content: "Error while reading whitelist.json",
			})
		}
		defer res.Close()

		// TODO: Anna will fix that shit

		var newWhitelisted []string

		for id := range whitelist.Whitelisted {
			if whitelist.Whitelisted[id] != uuid {
				newWhitelisted = append(newWhitelisted, whitelist.Whitelisted[id])
			}
		}

		whitelist.Whitelisted = newWhitelisted

		if err := json.NewEncoder(res).Encode(whitelist); err != nil {
			return err
		}

		if err := ReloadWhitelist(); err != nil {
			return err
		}

		return c.SendMessage(&component.Text{Content: "Removed " + c.Arguments["user"].Result.(string) + " from whitelist!", S: component.Style{Color: color.Green}})
	})

	EnableWhitelistCommand := command.Command(func(c *command.Context) error {
		if whitelist.Enabled {
			return c.SendMessage(&component.Text{
				Content: "Whitelist already on",
				S:       component.Style{Color: color.Red},
			})
		}
		res, err := os.OpenFile("whitelist.json", os.O_RDWR|os.O_TRUNC, 0755)
		if err != nil {
			return c.SendMessage(&component.Text{
				Content: "Error while reading whitelist.json",
			})
		}
		defer res.Close()

		whitelist.Enabled = true

		if err := json.NewEncoder(res).Encode(whitelist); err != nil {
			return err
		}

		if err := ReloadWhitelist(); err != nil {
			return err
		}

		return c.SendMessage(&component.Text{Content: "Enabled whitelist!", S: component.Style{Color: color.Green}})
	})

	DisableWhitelistCommand := command.Command(func(c *command.Context) error {
		if !whitelist.Enabled {
			return c.SendMessage(&component.Text{
				Content: "Whitelist already off",
				S:       component.Style{Color: color.Red},
			})
		}
		res, err := os.OpenFile("whitelist.json", os.O_RDWR|os.O_TRUNC, 0755)
		if err != nil {
			return c.SendMessage(&component.Text{
				Content: "Error while reading whitelist.json",
			})
		}
		defer res.Close()

		whitelist.Enabled = false

		if err := json.NewEncoder(res).Encode(whitelist); err != nil {
			return err
		}

		if err := ReloadWhitelist(); err != nil {
			return err
		}

		return c.SendMessage(&component.Text{Content: "Disabled whitelist!", S: component.Style{Color: color.Green}})
	})

	ListWhitelistCommand := command.Command(func(c *command.Context) error {
		var users bytes.Buffer
		for _, i := range whitelist.Whitelisted {
			str, _ := uuid.UUIDtoUsername(i)
			if users.Len() == 0 {
				users.WriteString(str)
			} else {
				users.WriteString(", " + str)
			}
		}

		return c.SendMessage(&component.Text{Content: fmt.Sprintf("Whitelisted users (%d): %s", len(whitelist.Whitelisted), users.String()), S: component.Style{Color: color.Green}})
	})

	return brigodier.Literal("whitelist").
		Then(brigodier.Literal("enable").Executes(EnableWhitelistCommand)).
		Then(brigodier.Literal("disable").Executes(DisableWhitelistCommand)).
		Then(brigodier.Literal("list").Executes(ListWhitelistCommand)).
		Then(brigodier.Literal("reload").Executes(ReloadWhitelistCommand())).
		Then(brigodier.Literal("add").Executes(UsageWhitelist()).Then(brigodier.Argument("user", brigodier.String).Executes(whitelistAddCommand))).
		Then(brigodier.Literal("remove").Executes(UsageWhitelist()).Then(brigodier.Argument("user", brigodier.String).Executes(whitelistRemoveCommand))).Executes(UsageWhitelist())
}

func UsageWhitelist() brigodier.Command {
	return command.Command(func(c *command.Context) error {
		return c.SendMessage(&component.Text{
			Content: "Usage: /whitelist <add/remove/enable/disable> <user>",
			S:       component.Style{Color: color.Red},
		})
	})
}

func ReloadWhitelistCommand() brigodier.Command {
	return command.Command(func(c *command.Context) error {
		if err := ReloadWhitelist(); err != nil {
			return err
		}

		return c.SendMessage(&component.Text{
			Content: "Reloaded command successfully!",
			S:       component.Style{Color: color.Green},
		})
	})
}

func ReloadWhitelist() error {
	res, err := os.OpenFile("whitelist.json", os.O_RDONLY, 0755)
	if err != nil {
		return err
	}
	defer res.Close()

	if err := json.NewDecoder(res).Decode(&whitelist); err != nil {
		return err
	}

	return nil
}

type WhitelistFile struct {
	Enabled     bool     `json:"enabled"`
	Whitelisted []string `json:"whitelisted"`
}
