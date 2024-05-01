package motd

import (
	"context"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util"
	"github.com/robinbraemer/event"
	"go.minekube.com/common/minecraft/color"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type Plugin struct {
	h *hosting.Hosting
}

func New(h *hosting.Hosting) (proxy.Plugin, error) {
	return proxy.Plugin{
		Name: "MOTD",
		Init: func(ctx context.Context, proxy *proxy.Proxy) error {
			plugin := &Plugin{h: h}

			return plugin.Init(proxy)
		},
	}, nil
}

func (p *Plugin) Init(prx *proxy.Proxy) error {
	event.Subscribe(prx.Event(), 0, p.onPingEvent())

	return nil
}

func (p *Plugin) onPingEvent() func(e *proxy.PingEvent) {
	return func(e *proxy.PingEvent) {
		ping := e.Ping()
		ping.Description = &Text{
			Extra: []Component{
				&Text{Content: "  ᴄѕᴍᴄ ", S: Style{Color: color.Green, Bold: True}},
				&Text{Content: "-", S: Style{Color: color.Gray, Bold: True}},
				&Text{Content: " " + util.Latinize("open beta") + "\n", S: Style{Color: color.Yellow, Bold: True}},
				&Text{Content: "  ɪɴᴅᴇᴠ ᴠᴇʀѕɪᴏɴ - ", S: Style{Color: color.LightPurple, Bold: True}},
				&Text{Content: util.Latinize(p.h.Info.PodName), S: Style{Color: color.LightPurple, Bold: True}},
			},
		}
		ping.Players.Max = ping.Players.Online + 1
	}
}
