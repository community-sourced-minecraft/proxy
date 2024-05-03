package tab

import (
	"context"

	"github.com/robinbraemer/event"
	"go.minekube.com/common/minecraft/color"
	c "go.minekube.com/common/minecraft/component"
	"go.minekube.com/common/minecraft/key"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

var Plugin = proxy.Plugin{
	Name: "Tablist",
	Init: func(ctx context.Context, proxy *proxy.Proxy) error {
		event.Subscribe(proxy.Event(), 0, onPostLogin)

		return nil
	},
}

func onPostLogin(e *proxy.ServerPostConnectEvent) {
	serverName := "LOADING"
	if e.Player().CurrentServer() != nil {
		serverName = e.Player().CurrentServer().Server().ServerInfo().Name()
	}

	header := &c.Text{
		S: c.Style{Bold: c.True},
		Extra: []c.Component{
			&c.Text{
				Content: "0",
				S:       c.Style{Font: key.New("csmc", "default")},
			},
			&c.Text{Content: "\n\n\n\n\n\n\n"},
			// &c.Text{Content: "\nᴡᴇʟᴄᴏᴍᴇ ᴛᴏ ", S: c.Style{Color: *&color.HexInt(0x926dd1).Named().RGB}},
			// mini.Gradient("ᴄѕᴍᴄ\n", c.Style{Bold: c.True}, *color.HexInt(0x7b52e3), *color.HexInt(0xcd52e3)),
			&c.Text{Content: serverName + "\n", S: c.Style{Color: color.DarkGray}},
		},
	}

	footer := &c.Text{
		Content: "\n  github.com/community-sourced-minecraft  \n",
		S:       c.Style{Color: color.Yellow},
	}

	// Most Gate methods are thread-safe and can be called from any goroutine.
	// We could also handle errors gracefully, like the tab list could not be sent
	// to the player because they disconnected, but we can often ignore them for simplicity.
	_ = e.Player().TabList().SetHeaderFooter(header, footer)
}
