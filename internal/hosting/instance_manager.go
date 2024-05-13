package hosting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/kv"
	"github.com/rs/zerolog/log"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

var (
	ErrNoServersAvailable = errors.New("no servers available")
)

type InstanceManager struct {
	prx         *proxy.Proxy
	instancesKV kv.Bucket
	rnd         *rand.Rand
}

func (h *Hosting) InstanceManager(ctx context.Context, prx *proxy.Proxy) (*InstanceManager, error) {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))

	instancesKV, err := h.KV().Bucket(ctx, h.Info.KVInstancesKey())
	if err != nil {
		return nil, err
	}

	return &InstanceManager{
		prx:         prx,
		instancesKV: instancesKV,
		rnd:         rnd,
	}, nil
}

func (m *InstanceManager) Register(ctx context.Context, name string, info InstanceInfo) error {
	ip, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", info.Address, info.Port))
	if err != nil {
		return err
	}

	if err := m.Unregister(ctx, name); err != nil {
		return err
	}

	_, err = m.prx.Register(proxy.NewServerInfo(name, ip))

	return err
}

func (m *InstanceManager) Unregister(ctx context.Context, name string) error {
	s := m.prx.Server(name)
	if s == nil {
		return nil
	}

	if m.prx.Unregister(s.ServerInfo()) {
		log.Info().Msgf("Unregistered server %s", name)
	}

	return nil
}

func (m *InstanceManager) GetServersOfGamemode(ctx context.Context, gamemode string) ([]proxy.RegisteredServer, error) {
	keys, err := m.instancesKV.ListKeys(ctx)
	if err != nil {
		return nil, err
	}

	var servers []proxy.RegisteredServer
	for _, key := range keys {
		v, err := m.instancesKV.Get(ctx, key)
		if err != nil {
			return nil, err
		}

		info := InstanceInfo{}
		if err := json.Unmarshal(v, &info); err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal instance info")
			continue
		}

		if info.Gamemode != gamemode {
			continue
		}

		s := m.prx.Server(key)
		if s == nil {
			log.Warn().Msgf("Server %s not found in registry", key)
			continue
		}

		servers = append(servers, s)
	}

	return servers, nil
}

func (m *InstanceManager) GetRandomServerOfGamemode(ctx context.Context, gamemode string) (proxy.RegisteredServer, error) {
	servers, err := m.GetServersOfGamemode(ctx, gamemode)
	if err != nil {
		return nil, err
	}

	if len(servers) == 0 {
		return nil, ErrNoServersAvailable
	}

	return servers[m.rnd.Intn(len(servers))], nil
}
