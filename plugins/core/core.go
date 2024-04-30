package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting/rpc"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/robinbraemer/event"
	"go.minekube.com/brigodier"
	"go.minekube.com/common/minecraft/color"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type CorePlugin struct {
	prx         *proxy.Proxy
	h           *hosting.Hosting
	rnd         *rand.Rand
	instancesKV jetstream.KeyValue
}

func New(h *hosting.Hosting) (proxy.Plugin, error) {
	return proxy.Plugin{
		Name: "Core",
		Init: func(ctx context.Context, prx *proxy.Proxy) error {
			rnd := rand.New(rand.NewSource(time.Now().Unix()))

			instancesKV, err := h.JetStream().CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: h.Info.KVInstancesKey()})
			if errors.Is(err, jetstream.ErrBucketExists) {
				instancesKV, err = h.JetStream().KeyValue(ctx, h.Info.KVInstancesKey())
				if err != nil {
					return err
				}
			} else if err != nil {
				return err
			}

			p := &CorePlugin{prx: prx, h: h, rnd: rnd, instancesKV: instancesKV}

			return p.Init(ctx)
		},
	}, nil
}

func (p *CorePlugin) registerPodByName(podName string, info hosting.InstanceInfo) error {
	ip, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", info.Address, info.Port))
	if err != nil {
		return err
	}

	if s := p.prx.Server(podName); s != nil {
		if p.prx.Unregister(s.ServerInfo()) {
			log.Printf("Unregistered server %s", podName)
		}
	}

	_, err = p.prx.Register(proxy.NewServerInfo(podName, ip))

	return err
}

func (p *CorePlugin) Init(ctx context.Context) error {
	go func() {
		watcher, err := p.instancesKV.WatchAll(ctx)
		if err != nil {
			log.Fatal(err)
		}

		for key := range watcher.Updates() {
			if key == nil {
				log.Println("Replayed keys for all instances")
				continue
			}

			podName := key.Key()

			switch key.Operation() {
			case jetstream.KeyValuePut:
				info := hosting.InstanceInfo{}
				if err := json.Unmarshal(key.Value(), &info); err != nil {
					log.Printf("Failed to unmarshal instance info: %v", err)
					continue
				}

				log.Printf("Parsed pod info for %s: %+v", podName, info)

				if err := p.registerPodByName(podName, info); err != nil {
					log.Fatal(err)
				}

			case jetstream.KeyValueDelete:
				log.Printf("Deleted pod info for %s", podName)

				if s := p.prx.Server(podName); s != nil {
					if p.prx.Unregister(s.ServerInfo()) {
						log.Printf("Unregistered server %s", podName)
					}
				}

				continue
			}
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

		sub, err := p.h.NATS().Subscribe(p.h.Info.RPCNetworkSubject(), func(msg *nats.Msg) {
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

			player := p.prx.Player(req.UUID)
			if player == nil {
				log.Printf("Player %s not found", req.UUID)
				msg.Nak()
				return
			}

			var newServer proxy.RegisteredServer
			for _, s := range p.prx.Servers() {
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

	p.prx.Command().Register(brigodier.Literal("ping").
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

	event.Subscribe(p.prx.Event(), 0, p.onServerSwitch)
	event.Subscribe(p.prx.Event(), 0, p.onChooseServer)

	return nil
}

func (p *CorePlugin) onChooseServer(e *proxy.PlayerChooseInitialServerEvent) {
	servers, err := p.getServersOfGamemode(e.Player().Context(), "lobby")
	if err != nil {
		log.Printf("Failed to get servers of gamemode lobby: %v", err)
		// Fallback to default
		e.SetInitialServer(p.prx.Server("lobby-0"))
		return
	}

	e.SetInitialServer(servers[p.rnd.Intn(len(servers))])
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
			&Text{Content: " through "},
			&Text{Content: p.h.Info.PodName, S: Style{Color: color.Yellow}},
			&Text{Content: "."},
		},
	})
}

func (p *CorePlugin) getServersOfGamemode(ctx context.Context, gamemode string) ([]proxy.RegisteredServer, error) {
	list, err := p.instancesKV.ListKeys(ctx)
	if err != nil {
		return nil, err
	}

	var servers []proxy.RegisteredServer
	for key := range list.Keys() {
		v, err := p.instancesKV.Get(ctx, key)
		if err != nil {
			return nil, err
		}

		info := hosting.InstanceInfo{}
		if err := json.Unmarshal(v.Value(), &info); err != nil {
			log.Printf("Failed to unmarshal instance info: %v", err)
			continue
		}

		if info.Gamemode != gamemode {
			continue
		}

		s := p.prx.Server(key)
		if s == nil {
			log.Printf("Server %s not found in registry", key)
			continue
		}

		servers = append(servers, s)
	}

	return servers, nil
}
