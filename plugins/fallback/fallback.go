package fallback

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/robinbraemer/event"
	"go.minekube.com/common/minecraft/color"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type FallbackPlugin struct {
	prx *proxy.Proxy
	h   *hosting.Hosting
	mgr *hosting.InstanceManager
}

func New(h *hosting.Hosting) (proxy.Plugin, error) {
	return proxy.Plugin{
		Name: "Fallback",
		Init: func(ctx context.Context, prx *proxy.Proxy) error {
			mgr, err := h.InstanceManager(ctx, prx)
			if err != nil {
				return err
			}

			p := &FallbackPlugin{prx: prx, h: h, mgr: mgr}

			return p.Init(ctx)
		},
	}, nil
}

func (p *FallbackPlugin) Init(ctx context.Context) error {
	event.Subscribe(p.prx.Event(), 0, p.onServerDisconnect)

	return nil
}

func (p *FallbackPlugin) onServerDisconnect(e *proxy.KickedFromServerEvent) {
	fmt.Println("Kicked from server!")

	server, err := p.mgr.GetRandomServerOfGamemode(e.Player().Context(), "lobby")
	if errors.Is(err, hosting.ErrNoServersAvailable) {
		log.Printf("No servers available for player %s", e.Player().ID())
		return
	} else if err != nil {
		log.Printf("Failed to get random server of gamemode lobby: %v", err)
		// Fallback to default
		e.Player().CreateConnectionRequest(p.prx.Server("lobby-0"))
		return
	}

	log.Printf("Chose server %s for player %s", server.ServerInfo().Name(), e.Player().ID())

	e.SetResult(&proxy.RedirectPlayerKickResult{
		Server: server,
		Message: &Text{
			Extra: []Component{
				&Text{Content: "Redirected to fallback server!", S: Style{Color: color.Gray}},
			},
		},
	})

	// e.Player().CreateConnectionRequest(server)
	e.Player().SendActionBar(&Text{
		Content: "Connecting to the fallback server.",
		S:       Style{Color: color.Gray},
	})
}
