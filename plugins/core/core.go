package core

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/robinbraemer/event"
	"go.minekube.com/brigodier"
	"go.minekube.com/common/minecraft/color"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type PodInfo struct {
	Network      string
	PodName      string
	PodNamespace string
}

func (p PodInfo) DebugString() string {
	return fmt.Sprintf("PodInfo{Network: %s, PodName: %s, PodNamespace: %s}", p.Network, p.PodName, p.PodNamespace)
}

type CorePlugin struct {
	*proxy.Proxy
	NATS      *nats.Conn
	JetStream jetstream.JetStream
	Info      PodInfo
}

func New(nc *nats.Conn, js jetstream.JetStream) (proxy.Plugin, error) {
	info := PodInfo{
		Network:      os.Getenv("CSMC_NETWORK"),
		PodName:      os.Getenv("POD_NAME"),
		PodNamespace: os.Getenv("POD_NAMESPACE"),
	}

	return proxy.Plugin{
		Name: "Core",
		Init: func(ctx context.Context, prx *proxy.Proxy) error {
			p := &CorePlugin{Proxy: prx, NATS: nc, JetStream: js, Info: info}

			return p.Init(ctx)
		},
	}, nil
}

func (p *CorePlugin) registerPodByName(gamemodeName, podName string) error {
	ip, err := net.ResolveTCPAddr("tcp4", podName+"."+gamemodeName+"."+p.Info.PodNamespace+".svc.cluster.local:25565")
	if err != nil {
		return err
	}

	_, err = p.Register(proxy.NewServerInfo(podName, ip))

	return err
}

func (p *CorePlugin) Init(ctx context.Context) error {
	gamemodesKVBucket := "csmc_" + p.Info.PodNamespace + "_" + p.Info.Network + "_gamemodes"
	log.Printf("Connecting to %s", gamemodesKVBucket)
	gamemodesKV, err := p.JetStream.KeyValue(ctx, gamemodesKVBucket)
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
			gamemodeInstancesKV, err := p.JetStream.KeyValue(ctx, "csmc_"+p.Info.PodNamespace+"_"+p.Info.Network+"_gamemode_"+gamemodeName+"_instances")
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

	p.Command().Register(brigodier.Literal("ping").
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

	event.Subscribe(p.Event(), 0, p.onServerSwitch)
	event.Subscribe(p.Event(), 0, p.onChooseServer)

	return nil
}

func (p *CorePlugin) onChooseServer(e *proxy.PlayerChooseInitialServerEvent) {
	// TODO: Get initial server from NATS
	e.SetInitialServer(p.Server("lobby-0"))
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
