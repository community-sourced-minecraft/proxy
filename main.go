package main

import (
	"log"
	"os"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/core"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/motd"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/permissions"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/tab"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/whitelist"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.minekube.com/gate/cmd/gate"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type PluginCreator = func() (proxy.Plugin, error)

func main() {
	permissionsFile, err := permissions.ReadFile("permissions.json")
	if err != nil {
		log.Fatal(err)
	}

	nc, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		log.Fatal(err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal(err)
	}

	var plugins = []PluginCreator{
		func() (proxy.Plugin, error) {
			return core.New(nc, js)
		},
		func() (proxy.Plugin, error) {
			return permissions.New(permissionsFile)
		},
		func() (proxy.Plugin, error) {
			return whitelist.New(permissionsFile)
		},
	}

	proxy.Plugins = append(proxy.Plugins,
		tab.Plugin,
		motd.Plugin,
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
