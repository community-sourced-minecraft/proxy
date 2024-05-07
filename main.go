package main

import (
	"context"
	"os"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/bossbar"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/core"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/fallback"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/motd"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/permissions"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/resourcepack"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/tab"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/plugins/whitelist"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"go.minekube.com/gate/cmd/gate"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type PluginCreator = func(h *hosting.Hosting) (proxy.Plugin, error)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("LOG_FORMAT") != "json" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	if os.Getenv("LOG_LEVEL") != "" {
		lvl, err := zerolog.ParseLevel(os.Getenv("LOG_LEVEL"))
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to parse log level")
		}
		log.Logger = log.Level(lvl)
	}

	h, err := hosting.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize hosting")
	}

	perms, err := permissions.NewKVPermissions(context.Background(), h)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize permissions")
	}

	var plugins = []PluginCreator{
		core.New,
		fallback.New,
		func(_ *hosting.Hosting) (proxy.Plugin, error) {
			return permissions.New(perms)
		},
		func(h *hosting.Hosting) (proxy.Plugin, error) {
			return whitelist.New(h, perms)
		},
		motd.New,
		tab.New,
		bossbar.New,
		resourcepack.New,
	}

	for _, create := range plugins {
		p, err := create(h)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create plugin")
		}
		proxy.Plugins = append(proxy.Plugins, p)
	}

	gate.Execute()
}
