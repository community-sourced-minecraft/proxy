package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting/rpc"
	"github.com/nats-io/nats.go"
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

	{
		errorReqRes, err := json.Marshal(&rpc.TransferPlayerResponse{Status: rpc.StatusError})
		if err != nil {
			log.Printf("Failed to marshal transfer player response: %v", err)
			return err
		}

		errorRes, err := json.Marshal(&rpc.Response{Type: rpc.TypeTransferPlayer, Data: string(errorReqRes)})
		if err != nil {
			log.Printf("Failed to marshal response: %v", err)
			return err
		}

		sub, err := p.h.NATS().Subscribe(p.h.Info.RPCBaseSubject()+".transfers", func(msg *nats.Msg) {
			log.Printf("Received raw request on transfers queue: %s", string(msg.Data))

			payload := &rpc.Request{}
			if err := json.Unmarshal(msg.Data, payload); err != nil {
				log.Printf("Failed to unmarshal payload: %v", err)
				return
			}

			if payload.Type != rpc.TypeTransferPlayer {
				log.Printf("Invalid payload type: %s", payload.Type)
				msg.Nak()
				return
			}

			req := &rpc.TransferPlayerRequest{}
			if err := json.Unmarshal([]byte(payload.Data), req); err != nil {
				log.Printf("Failed to unmarshal transfer player request: %v", err)
				msg.Nak()
				return
			}
			log.Printf("Transfer player request: %v", req)

			player := p.proxy.Player(req.UUID)
			if player == nil {
				log.Printf("Player %s not found", req.UUID)
				msg.Nak()
				return
			}

			var newServer proxy.RegisteredServer
			for _, s := range p.proxy.Servers() {
				sName := s.ServerInfo().Name()

				if sName == req.Destination {
					newServer = s
					break
				}

				if strings.HasPrefix(sName, req.Destination+"-") {
					newServer = s
					break
				}
			}
			if newServer == nil {
				log.Printf("Server %s not found", req.Destination)
				msg.Nak()
				return
			}

			c, err := player.CreateConnectionRequest(newServer).Connect(context.Background())
			if err != nil {
				log.Printf("Failed to connect player %s to server %s: %v", req.UUID, req.Destination, err)

				if err := msg.Respond(errorRes); err != nil {
					log.Printf("Failed to respond to transfer player request: %v", err)
				}

				return
			}

			if c.Status() == proxy.AlreadyConnectedConnectionStatus {
				log.Printf("Player %s is already connected to server %s", req.UUID, req.Destination)
				msg.Ack()
				return
			} else if c.Status() != proxy.SuccessConnectionStatus {
				log.Printf("Failed to connect player %s to server %s: %v: %v", req.UUID, req.Destination, c.Status(), c.Reason())

				if err := msg.Respond(errorRes); err != nil {
					log.Printf("Failed to respond to transfer player request: %v", err)
				}

				return
			}

			reqRes, err := json.Marshal(&rpc.TransferPlayerResponse{Status: rpc.StatusOk})
			if err != nil {
				log.Printf("Failed to marshal transfer player response: %v", err)

				if err := msg.Respond(errorRes); err != nil {
					log.Printf("Failed to respond to transfer player request: %v", err)
				}

				return
			}

			res, err := json.Marshal(&rpc.Response{Type: payload.Type, Data: string(reqRes)})
			if err != nil {
				log.Printf("Failed to marshal response: %v", err)

				if err := msg.Respond(errorRes); err != nil {
					log.Printf("Failed to respond to transfer player request: %v", err)
				}

				return
			}

			if err := msg.Respond(res); err != nil {
				log.Printf("Failed to respond to transfer player request: %v", err)
			}

			log.Printf("Player %s transferred to server %s", req.UUID, req.Destination)
		})
		if err != nil {
			log.Printf("Failed to subscribe to transfers: %v", err)
			return err
		}

		log.Printf("Subscribed to %v", sub.Subject)
	}

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
