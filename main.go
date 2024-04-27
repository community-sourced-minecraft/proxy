package main

import (
	"log"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/core"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/motd"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/permissions"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/tab"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/whitelist"
	"go.minekube.com/gate/cmd/gate"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type PluginCreator = func() (proxy.Plugin, error)

var plugins = []PluginCreator{
	permissions.New,
}

func main() {
	proxy.Plugins = append(proxy.Plugins,
		tab.Plugin,
		motd.Plugin,
		core.Plugin,
		whitelist.Plugin,
	)

	for _, create := range plugins {
		p, err := create()
		if err != nil {
			log.Fatal(err)
		}
		proxy.Plugins = append(proxy.Plugins, p)
	}

	gate.Execute()
}
