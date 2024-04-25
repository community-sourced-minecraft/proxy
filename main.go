package main

import (
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/core"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/motd"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/tab"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/whitelist"
	"go.minekube.com/gate/cmd/gate"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

func main() {
	proxy.Plugins = append(proxy.Plugins,
		tab.Plugin,
		motd.Plugin,
		core.Plugin,
		whitelist.Plugin,
	)

	gate.Execute()
}
