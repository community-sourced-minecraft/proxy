package core

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/robinbraemer/event"
	"go.minekube.com/brigodier"
	"go.minekube.com/common/minecraft/color"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type CorePlugin struct {
	proxy *proxy.Proxy
	h     *hosting.Hosting
}

func New(h *hosting.Hosting) (proxy.Plugin, error) {
	return proxy.Plugin{
		Name: "Core",
		Init: func(ctx context.Context, prx *proxy.Proxy) error {
			p := &CorePlugin{proxy: prx, h: h}

			return p.Init(ctx)
		},
	}, nil
}

func (p *CorePlugin) registerPodByName(gamemodeName, podName string) error {
	ip, err := net.ResolveTCPAddr("tcp4", podName+"."+gamemodeName+"."+p.h.Info.PodNamespace+".svc.cluster.local:25565")
	if err != nil {
		return err
	}

	if s := p.proxy.Server(podName); s != nil {
		if p.proxy.Unregister(s.ServerInfo()) {
			log.Printf("Unregistered server %s", podName)
		}
	}

	_, err = p.proxy.Register(proxy.NewServerInfo(podName, ip))

	return err
}

func (p *CorePlugin) Init(ctx context.Context) error {
	gamemodesKVBucket := "csmc_" + p.h.Info.PodNamespace + "_" + p.h.Info.Network + "_gamemodes"
	log.Printf("Connecting to %s", gamemodesKVBucket)
	gamemodesKV, err := p.h.JetStream().KeyValue(ctx, gamemodesKVBucket)
	if err != nil {
		return err
	}
	log.Printf("Connected to %s", gamemodesKVBucket)

	go func() {
		watcher, err := gamemodesKV.WatchAll(ctx)
		if err != nil {
			log.Fatal(err)
		}

		for key := range watcher.Updates() {
			if key == nil {
				log.Println("Replayed keys for all gamemodes")
				continue
			}

			gamemodeName := key.Key()

			log.Printf("Gamemode %s added", gamemodeName)
			gamemodeInstancesKV, err := p.h.JetStream().KeyValue(ctx, "csmc_"+p.h.Info.PodNamespace+"_"+p.h.Info.Network+"_gamemode_"+gamemodeName+"_instances")
			if err != nil {
				log.Fatal(err)
			}

			go func() {
				watcher, err := gamemodeInstancesKV.WatchAll(ctx)
				if err != nil {
					log.Fatal(err)
				}

				for key := range watcher.Updates() {
					if key == nil {
						log.Printf("Replayed keys for all instances of gamemode %s", gamemodeName)
						continue
					}

					podName := key.Key()

					log.Printf("Pod %s added to gamemode %s", podName, gamemodeName)

					if err := p.registerPodByName(gamemodeName, podName); err != nil {
						log.Fatal(err)
					}
				}
			}()
		}
	}()

	p.proxy.Command().Register(brigodier.Literal("ping").
		Executes(command.Command(func(c *command.Context) error {
			player, ok := c.Source.(proxy.Player)
			if !ok {
				return c.Source.SendMessage(&Text{Content: "Pong!"})
			}

			return player.SendMessage(&Text{
				Content: fmt.Sprintf("Pong! Your ping is %s", player.Ping()),
				S:       Style{Color: color.Green},
			})
		})),
	)

	event.Subscribe(p.proxy.Event(), 0, p.onServerSwitch)
	event.Subscribe(p.proxy.Event(), 0, p.onChooseServer)

	return nil
}

func (p *CorePlugin) onChooseServer(e *proxy.PlayerChooseInitialServerEvent) {
	// TODO: Get initial server from NATS
	e.SetInitialServer(p.proxy.Server("lobby-0"))
}

func (p *CorePlugin) onServerSwitch(e *proxy.ServerPostConnectEvent) {
	s := e.Player().CurrentServer()
	if s == nil {
		return
	}

	_ = e.Player().SendMessage(&Text{
		S: Style{Color: color.Aqua},
		Extra: []Component{
			&Text{Content: "You connected to "},
			&Text{Content: s.Server().ServerInfo().Name(), S: Style{Color: color.Yellow}},
			&Text{Content: "."},
		},
	})
}
