package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting/rpc"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/kv"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/messaging"
	"github.com/robinbraemer/event"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.minekube.com/brigodier"
	"go.minekube.com/common/minecraft/color"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

type CorePlugin struct {
	prx         *proxy.Proxy
	h           *hosting.Hosting
	mgr         *hosting.InstanceManager
	instancesKV kv.Bucket
	l           zerolog.Logger
}

func New(h *hosting.Hosting) (proxy.Plugin, error) {
	return proxy.Plugin{
		Name: "Core",
		Init: func(ctx context.Context, prx *proxy.Proxy) error {
			instancesKV, err := h.KV().Bucket(ctx, h.Info.KVInstancesKey())
			if err != nil {
				return err
			}

			l := log.With().Str("plugin", "core").Logger()

			mgr, err := h.InstanceManager(ctx, prx)
			if err != nil {
				return err
			}

			p := &CorePlugin{prx: prx, h: h, instancesKV: instancesKV, mgr: mgr, l: l}

			return p.Init(ctx)
		},
	}, nil
}

func (p *CorePlugin) Init(ctx context.Context) error {
	go func() {
		watcher, err := p.instancesKV.WatchAll(ctx)
		if err != nil {
			p.l.Fatal().Err(err).Msg("Failed to watch all instances")
		}

		for key := range watcher.Changes() {
			if key == nil {
				continue
			}

			podName := key.Key

			switch key.Operation {
			case kv.Put:
				info := hosting.InstanceInfo{}
				if err := json.Unmarshal(key.Value, &info); err != nil {
					p.l.Error().Err(err).Msgf("Failed to unmarshal pod info for %s", podName)
					continue
				}

				p.l.Info().Msgf("Parsed pod info for %s: %+v", podName, info)

				if err := p.mgr.Register(ctx, podName, info); err != nil {
					p.l.Error().Err(err).Msgf("Failed to register server %s", podName)
				}

			case kv.Delete:
				if err := p.mgr.Unregister(ctx, podName); err != nil {
					p.l.Error().Err(err).Msgf("Failed to unregister server %s", podName)
				}
			}
		}
	}()

	{
		handleTransferPlayerRequest, err := p.handleTransferPlayerRequest()
		if err != nil {
			return err
		}

		p.h.NetworkEventBus().Register(rpc.TypeTransferPlayer, func(msg messaging.Message, req *rpc.Request) error {
			return handleTransferPlayerRequest(msg, req)
		})

		p.h.PodEventBus().Register(rpc.TypeUpgradeInstance, p.upgradeInstance)
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
	server, err := p.mgr.GetRandomServerOfGamemode(e.Player().Context(), "lobby")
	if errors.Is(err, hosting.ErrNoServersAvailable) {
		p.l.Warn().Msgf("No servers available for player %s", e.Player().ID())
		return
	} else if err != nil {
		p.l.Error().Err(err).Msg("Failed to get servers of gamemode lobby")
		// Fallback to default
		e.SetInitialServer(p.prx.Server("lobby-0"))
		return
	}

	p.l.Trace().Msgf("Chose server %s for player %s", server.ServerInfo().Name(), e.Player().ID())

	e.SetInitialServer(server)
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

func (p *CorePlugin) handleTransferPlayerRequest() (hosting.EventHandler, error) {
	errorReqRes, err := json.Marshal(&rpc.TransferPlayerResponse{Status: rpc.StatusError})
	if err != nil {
		p.l.Error().Err(err).Msg("Failed to marshal error transfer player response")
		return nil, err
	}

	errorRes, err := json.Marshal(&rpc.Response{Type: rpc.TypeTransferPlayer, Data: string(errorReqRes)})
	if err != nil {
		p.l.Error().Err(err).Msg("Failed to marshal error response")
		return nil, err
	}

	return func(msg messaging.Message, payload *rpc.Request) error {
		req := &rpc.TransferPlayerRequest{}
		if err := json.Unmarshal([]byte(payload.Data), req); err != nil {
			p.l.Error().Err(err).Msg("Failed to unmarshal transfer player request")

			return err
		}

		l := p.l.With().Str("player", req.UUID.String()).Str("destination", req.Destination).Logger()

		player := p.prx.Player(req.UUID)
		if player == nil {
			l.Trace().Msg("Player not found on this proxy")
			return fmt.Errorf("player %s not found on this proxy", req.UUID)
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
			p.l.Error().Msgf("Server %s not found", req.Destination)
			return fmt.Errorf("server %s not found", req.Destination)
		}

		c, err := player.CreateConnectionRequest(newServer).Connect(msg.Context())
		if err != nil {
			p.l.Error().Err(err).Msgf("Failed to connect player %s to server %s", req.UUID, req.Destination)

			if err := msg.Respond(errorRes); err != nil {
				p.l.Error().Err(err).Msg("Failed to respond to transfer player request: %v")
			}

			return err
		}

		if c.Status() == proxy.AlreadyConnectedConnectionStatus {
			p.l.Info().Msgf("Player %s already connected to server %s", req.UUID, req.Destination)

			if err := msg.Ack(); err != nil {
				p.l.Error().Err(err).Msg("Failed to ack transfer player request")
			}

			return nil
		} else if c.Status() != proxy.SuccessConnectionStatus {
			p.l.Printf("Failed to connect player %s to server %s: %v: %v", req.UUID, req.Destination, c.Status(), c.Reason())

			if err := msg.Respond(errorRes); err != nil {
				p.l.Error().Err(err).Msg("Failed to respond to transfer player request: %v")
			}

			return nil
		}

		reqRes, err := json.Marshal(&rpc.TransferPlayerResponse{Status: rpc.StatusOk})
		if err != nil {
			p.l.Error().Err(err).Msg("Failed to marshal transfer player response")

			if err := msg.Respond(errorRes); err != nil {
				p.l.Error().Err(err).Msg("Failed to respond to transfer player request: %v")
			}

			return nil
		}

		res, err := json.Marshal(&rpc.Response{Type: payload.Type, Data: string(reqRes)})
		if err != nil {
			p.l.Error().Err(err).Msg("Failed to marshal response")

			if err := msg.Respond(errorRes); err != nil {
				p.l.Error().Err(err).Msg("Failed to respond to transfer player request: %v")
			}

			return nil
		}

		if err := msg.Respond(res); err != nil {
			l.Error().Err(err).Msg("Failed to respond to transfer player request: %v")
		}

		l.Info().Msgf("Player %s transferred to server %s", req.UUID, req.Destination)

		return nil
	}, nil
}

func (p *CorePlugin) upgradeInstance(msg messaging.Message, req *rpc.Request) error {
	p.prx.Shutdown(&Text{
		Content: "Proxy is upgrading...",
		S:       Style{Color: color.Red},
	})

	return nil
}
