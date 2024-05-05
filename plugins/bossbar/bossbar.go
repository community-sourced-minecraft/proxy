package bossbar

import (
	"context"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util"
	"github.com/robinbraemer/event"
	"go.minekube.com/common/minecraft/color"
	"go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/edition/java/bossbar"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

func New(_ *hosting.Hosting) (proxy.Plugin, error) {
	return proxy.Plugin{
		Name: "Bossbar",
		Init: func(ctx context.Context, proxy *proxy.Proxy) error {
			event.Subscribe(proxy.Event(), 0, bossbarDisplay())

			return nil
		},
	}, nil
}

func bossbarDisplay() func(*proxy.ServerConnectedEvent) {
	return func(ple *proxy.ServerConnectedEvent) {
		bossbar.New(&component.Text{
			Extra: []component.Component{
				&component.Text{
					Content: util.Latinize("not representative of the final product."),
					S:       component.Style{Color: color.HexInt(0xffffff)},
				},
			},
		}, bossbar.MinProgress, bossbar.WhiteColor, bossbar.ProgressOverlay).AddViewer(ple.Player())
	}
}
