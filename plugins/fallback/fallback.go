package fallback

import (
	"context"
	"errors"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/robinbraemer/event"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.minekube.com/common/minecraft/color"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type FallbackPlugin struct {
	prx *proxy.Proxy
	h   *hosting.Hosting
	mgr *hosting.InstanceManager
	l   zerolog.Logger
}

func New(h *hosting.Hosting) (proxy.Plugin, error) {
	return proxy.Plugin{
		Name: "Fallback",
		Init: func(ctx context.Context, prx *proxy.Proxy) error {
			mgr, err := h.InstanceManager(ctx, prx)
			if err != nil {
				return err
			}

			p := &FallbackPlugin{prx: prx, h: h, mgr: mgr, l: log.With().Str("plugin", "fallback").Logger()}

			return p.Init(ctx)
		},
	}, nil
}

func (p *FallbackPlugin) Init(ctx context.Context) error {
	event.Subscribe(p.prx.Event(), 0, p.onServerDisconnect)

	return nil
}

func (p *FallbackPlugin) onServerDisconnect(e *proxy.KickedFromServerEvent) {
	l := p.l.With().Str("player", e.Player().ID().String()).Logger()

	// TODO: Figure out if the player got disconnected because of a kick or a server shutdown

	p.l.Println("Got kicked from server")

	server, err := p.mgr.GetRandomServerOfGamemode(e.Player().Context(), "lobby")
	if errors.Is(err, hosting.ErrNoServersAvailable) {
		l.Warn().Msg("No servers available")
		return
	} else if err != nil {
		l.Error().Err(err).Msgf("Failed to get random server of gamemode lobby")
		// Fallback to default
		e.Player().CreateConnectionRequest(p.prx.Server("lobby-0"))
		return
	}

	l.Info().Msgf("Chose server %s", server.ServerInfo().Name())

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
