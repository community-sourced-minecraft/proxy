package hosting

import (
	"encoding/json"
	"sync"

	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/hosting/rpc"
	"github.com/Community-Sourced-Minecraft/Gate-Proxy/internal/messaging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func initMessaging() (messaging.Messager, error) {
	logging := getEnvBoolWithDefault("MESSAGING_LOGGING", false)
	backend := getEnvWithDefault("MESSAGING_BACKEND", "nats")
	backendOptions := getEnvWithDefault("MESSAGING_BACKEND_OPTIONS", "{\"url\":\"nats://127.0.0.1:4222\"}")

	var msgC messaging.Messager

	switch backend {
	case "nats":
		log.Info().Msg("Using NATS as messaging backend")

		opts := messaging.NATSOptions{}
		if err := json.Unmarshal([]byte(backendOptions), &opts); err != nil {
			return nil, err
		}

		nc, err := connectToNATS(opts.URL)
		if err != nil {
			return nil, err
		}

		msgC = messaging.NewNATS(nc)

	default:
		log.Fatal().Msgf("unknown messaging backend: %s", backend)
	}

	if logging {
		msgC = messaging.WithLogger(msgC)
	}

	return msgC, nil
}

func (n *Hosting) Messaging() messaging.Messager {
	return n.msg
}

type EventHandler = func(msg messaging.Message, req *rpc.Request) error

type EventBus struct {
	handlers map[rpc.Type]EventHandler
	m        sync.Mutex
	l        zerolog.Logger
}

func (b *EventBus) Register(t rpc.Type, h EventHandler) func() {
	b.m.Lock()
	b.handlers[t] = h
	b.m.Unlock()

	return func() {
		b.m.Lock()
		delete(b.handlers, t)
		b.m.Unlock()
	}
}

func (b *EventBus) Handle(msg messaging.Message) {
	l := b.l.With().Bytes("data", msg.Data()).Logger()
	l.Trace().Msgf("Received raw request")

	payload := &rpc.Request{}
	if err := json.Unmarshal(msg.Data(), payload); err != nil {
		l.Error().Err(err).Msg("Failed to unmarshal request")
		return
	}

	b.m.Lock()
	handler, exists := b.handlers[payload.Type]
	b.m.Unlock()

	if !exists {
		l.Trace().Msgf("Ignoring request of type %s", payload.Type)

		if err := msg.Nak(); err != nil {
			l.Error().Err(err).Msg("Failed to nack request")
		}

		return
	}

	if err := handler(msg, payload); err != nil {
		l.Error().Err(err).Msg("Failed to handle request")

		if err := msg.Nak(); err != nil {
			l.Error().Err(err).Msg("Failed to nack request")
		}

		return
	}
}

func NewEventBus(messager messaging.Messager, subject string) (*EventBus, error) {
	b := &EventBus{
		handlers: make(map[rpc.Type]EventHandler),
		l:        log.With().Str("subject", subject).Logger(),
	}

	if err := messager.Subscribe(subject, b.Handle); err != nil {
		return nil, err
	}

	return b, nil
}

func (h *Hosting) NetworkEventBus() *EventBus {
	return h.nwEventBus
}

func (h *Hosting) PodEventBus() *EventBus {
	return h.podEventBus
}
