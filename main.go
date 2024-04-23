package main

import (
	"context"
	"fmt"

	"github.com/robinbraemer/event"
	"go.minekube.com/brigodier"
	"go.minekube.com/common/minecraft/color"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/cmd/gate"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

func main() {
	proxy.Plugins = append(proxy.Plugins, proxy.Plugin{
		Name: "SimpleProxy",
		Init: func(ctx context.Context, proxy *proxy.Proxy) error {
			return newSimpleProxy(proxy).init()
		},
	})

	gate.Execute()
}

type SimpleProxy struct {
	*proxy.Proxy
}

func newSimpleProxy(proxy *proxy.Proxy) *SimpleProxy {
	return &SimpleProxy{
		Proxy: proxy,
	}
}

func (p *SimpleProxy) init() error {
	p.registerCommands()
	p.registerSubscribers()

	return nil
}

func (p *SimpleProxy) registerCommands() {
	p.Command().Register(brigodier.Literal("ping").
		Executes(command.Command(func(c *command.Context) error {
			player, ok := c.Source.(proxy.Player)
			if !ok {
				return c.Source.SendMessage(&Text{Content: "Pong!"})
			}

			return player.SendMessage(&Text{
				Content: fmt.Sprintf("Pong! Your ping is %s", player.Ping()),
				S:       Style{Color: color.Green},
			})
		})),
	)
}

func (p *SimpleProxy) registerSubscribers() {
	event.Subscribe(p.Event(), 0, p.onServerSwitch)
	event.Subscribe(p.Event(), 0, pingHandler())
}

func (p *SimpleProxy) onServerSwitch(e *proxy.ServerPostConnectEvent) {
	newServer := e.Player().CurrentServer()
	if newServer == nil {
		return
	}

	_ = e.Player().SendMessage(&Text{
		S: Style{Color: color.Aqua},
		Extra: []Component{
			&Text{
				Content: "\nWelcome to the Gate Sample proxy!\n\n",
				S:       Style{Color: color.Green, Bold: True},
			},
			&Text{Content: "You connected to "},
			&Text{Content: newServer.Server().ServerInfo().Name(), S: Style{Color: color.Yellow}},
			&Text{Content: "."},
			&Text{
				S: Style{
					ClickEvent: SuggestCommand("/broadcast Gate is awesome!"),
					HoverEvent: ShowText(&Text{Content: "/broadcast Gate is awesome!"}),
				},
				Content: "\n\nClick me to run ",
				Extra: []Component{&Text{
					Content: "/broadcast Gate is awesome!",
					S:       Style{Color: color.White, Bold: True, Italic: True},
				}},
			},
			&Text{
				Content: "\n\nClick me to run sample /title command!",
				S: Style{
					HoverEvent: ShowText(&Text{Content: "/title <title> [subtitle]"}),
					ClickEvent: SuggestCommand(`/title "&eGate greets" &2&o` + e.Player().Username()),
				},
			},
			&Text{Content: "\n\nMore sample commands you can try: "},
			&Text{
				Content: "/ping",
				S:       Style{Color: color.Yellow},
			},
		},
	})
}

func pingHandler() func(p *proxy.PingEvent) {
	motd := &Text{Content: "Simple Proxy!\nJoin and test me."}
	return func(e *proxy.PingEvent) {
		p := e.Ping()
		p.Description = motd
		p.Players.Max = p.Players.Online + 1
	}
}
