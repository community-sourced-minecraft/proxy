package fallback

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/robinbraemer/event"
	"go.minekube.com/common/minecraft/color"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type FallbackPlugin struct {
	prx         *proxy.Proxy
	h           *hosting.Hosting
	rnd         *rand.Rand
	instancesKV jetstream.KeyValue
}

func New(h *hosting.Hosting) (proxy.Plugin, error) {
	return proxy.Plugin{
		Name: "Fallback",
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

			p := &FallbackPlugin{prx: prx, h: h, rnd: rnd, instancesKV: instancesKV}

			return p.Init(ctx)
		},
	}, nil
}

func (p *FallbackPlugin) Init(ctx context.Context) error {
	event.Subscribe(p.prx.Event(), 0, p.onServerDisconnect)

	return nil
}

func (p *FallbackPlugin) onServerDisconnect(e *proxy.KickedFromServerEvent) {
	fmt.Println("Kicked from server!")
	servers, err := p.GetServersOfGamemode(e.Player().Context(), "lobby")
	if err != nil {
		log.Printf("Failed to get servers of gamemode lobby: %v", err)
		// Fallback to default
		e.Player().CreateConnectionRequest(p.prx.Server("lobby-0"))
		return
	}

	availableServers := len(servers)
	if availableServers == 0 {
		log.Printf("No servers available for player %s", e.Player().ID())
		return
	}

	server := servers[p.rnd.Intn(availableServers)]
	log.Printf("Chose server %s for player %s", server.ServerInfo().Name(), e.Player().ID())

	e.SetResult(&proxy.RedirectPlayerKickResult{
		Server: server,
		Message: &Text{
			Extra: []Component{
				&Text{Content: "Redirected to fallback server!", S: Style{Color: color.Gray}},
			},
		},
	})

	// e.Player().CreateConnectionRequest(server)
	e.Player().SendActionBar(&Text{
		Content: "Connecting to the fallback server.",
		S:       Style{Color: color.Gray},
	})
}

func (p *FallbackPlugin) GetServersOfGamemode(ctx context.Context, gamemode string) ([]proxy.RegisteredServer, error) {
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
