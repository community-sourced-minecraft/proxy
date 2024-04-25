package motd

import (
	"context"

	"github.com/robinbraemer/event"
	"go.minekube.com/common/minecraft/color"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

var Plugin = proxy.Plugin{
	Name: "MOTD",
	Init: func(ctx context.Context, proxy *proxy.Proxy) error {
		event.Subscribe(proxy.Event(), 0, onPingEvent())

		return nil
	},
}

func onPingEvent() func(e *proxy.PingEvent) {

	return func(e *proxy.PingEvent) {
		p := e.Ping()
		p.Description = &Text{
			Extra: []Component{
				&Text{Content: "  ᴄѕᴍᴄ ", S: Style{Color: color.Green, Bold: True}},
				&Text{Content: "-", S: Style{Color: color.Gray, Bold: True}},
				&Text{Content: " ᴄʟᴏѕᴇᴅ ʙᴇᴛᴀ\n", S: Style{Color: color.Red, Bold: True}},
				&Text{Content: "  ɪɴᴅᴇᴠ ᴠᴇʀѕɪᴏɴ", S: Style{Color: color.LightPurple, Bold: True}},
			},
		}
		p.Players.Max = p.Players.Online + 1
	}

}
