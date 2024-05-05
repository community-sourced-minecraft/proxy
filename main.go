package main

import (
	"context"
	"log"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/bossbar"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/core"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/fallback"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/motd"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/permissions"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/resourcepack"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/tab"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/whitelist"

	"go.minekube.com/gate/cmd/gate"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type PluginCreator = func(h *hosting.Hosting) (proxy.Plugin, error)

func main() {
	h, err := hosting.Init()
	if err != nil {
		log.Fatal(err)
	}

	perms, err := permissions.NewKVPermissions(context.Background(), h)
	if err != nil {
		log.Fatal(err)
	}

	var plugins = []PluginCreator{
		func(h *hosting.Hosting) (proxy.Plugin, error) {
			return core.New(h)
		},
		func(h *hosting.Hosting) (proxy.Plugin, error) {
			return fallback.New(h)
		},
		func(_ *hosting.Hosting) (proxy.Plugin, error) {
			return permissions.New(perms)
		},
		func(_ *hosting.Hosting) (proxy.Plugin, error) {
			return whitelist.New(h, perms)
		},
		motd.New,
	}

	proxy.Plugins = append(proxy.Plugins,
		tab.Plugin,
		bossbar.Plugin,
		resourcepack.Plugin,
	)

	for _, create := range plugins {
		p, err := create(h)
		if err != nil {
			log.Fatal(err)
		}
		proxy.Plugins = append(proxy.Plugins, p)
	}

	gate.Execute()
}
