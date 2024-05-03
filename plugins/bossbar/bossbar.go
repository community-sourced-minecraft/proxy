package bossbar

import (
	"context"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/lib/util"
	"github.com/robinbraemer/event"
	"go.minekube.com/common/minecraft/color"
	"go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/edition/java/bossbar"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

var Plugin = proxy.Plugin{
	Name: "Bossbar",
	Init: func(ctx context.Context, proxy *proxy.Proxy) error {
		event.Subscribe(proxy.Event(), 0, bossbarDisplay())

		return nil
	},
}

func bossbarDisplay() func(*proxy.PostLoginEvent) {
	return func(ple *proxy.PostLoginEvent) {
		ple.Player().SendResourcePack(proxy.ResourcePackInfo{
			URL:         "https://s3.devminer.xyz/csmc/csmc.zip",
			ShouldForce: true,
			Prompt: &component.Text{
				Content: util.Latinize("you are required to use this texturepack in csmc.dev"),
				S:       component.Style{Color: color.Yellow, Bold: component.True},
			},
		})
		bossbar.New(&component.Text{
			Extra: []component.Component{
				// &component.Translation{Key: "space.-50"},
				&component.Text{
					Content: util.Latinize("not representative of the final product."),
					S:       component.Style{Color: color.HexInt(0xffffff)},
				},
				// &component.Translation{Key: "space.-170"},
				// &component.Translation{Key: "newlayer"},
			},
		}, bossbar.MinProgress, bossbar.WhiteColor, bossbar.ProgressOverlay).AddViewer(ple.Player())

		// bossbar.New(&component.Text{
		// 	Extra: []component.Component{
		// 		&component.Text{
		// 			Content: util.Latinize("not representative of the final product."),
		// 			S:       component.Style{Color: color.HexInt(0xffffff)},
		// 		},
		// 	},
		// }, bossbar.MinProgress, bossbar.WhiteColor, bossbar.ProgressOverlay).AddViewer(ple.Player())

		// bossbar.New(&component.Text{
		// 	Content: "",
		// })
	}
}
