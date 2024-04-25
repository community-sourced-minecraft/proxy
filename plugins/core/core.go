package core

import (
	"context"
	"fmt"

	"github.com/robinbraemer/event"
	"go.minekube.com/brigodier"
	"go.minekube.com/common/minecraft/color"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

var Plugin = proxy.Plugin{
	Name: "Core",
	Init: func(ctx context.Context, proxy *proxy.Proxy) error {
		csmcProxy(proxy).init()
		return nil
	},
}

type CSMCProxy struct {
	*proxy.Proxy
}

func csmcProxy(proxy *proxy.Proxy) *CSMCProxy {
	return &CSMCProxy{Proxy: proxy}
}

func (p *CSMCProxy) init() error {
	// host := os.Getenv("GAME_SERVER_SERVICE_HOST")
	// p.Register(proxy.NewServerInfo("lobby", net.TCPAddrFromAddrPort(netip.MustParseAddrPort(host+":25565"))))

	// TODO: Connect to NATS
	// TODO: Listen to NATS events and register + unregister servers

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

	event.Subscribe(p.Event(), 0, p.onServerSwitch)
	event.Subscribe(p.Event(), 0, p.onChooseServer)

	return nil
}

func (p *CSMCProxy) onChooseServer(e *proxy.PlayerChooseInitialServerEvent) {
	// e.SetInitialServer(p.Server("lobby"))
}

func (p *CSMCProxy) onServerSwitch(e *proxy.ServerPostConnectEvent) {
	s := e.Player().CurrentServer()
	if s == nil {
		return
	}

	_ = e.Player().SendMessage(&Text{
		S: Style{Color: color.Aqua},
		Extra: []Component{
			&Text{Content: "You connected to "},
			&Text{Content: s.Server().ServerInfo().Name(), S: Style{Color: color.Yellow}},
			&Text{Content: "."},
		},
	})
}
