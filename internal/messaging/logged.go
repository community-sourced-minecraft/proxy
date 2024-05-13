package messaging

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var _ Messager = &Logged{}

type Logged struct {
	m Messager
}

func WithLogger(m Messager) *Logged {
	return &Logged{m: m}
}

func (m *Logged) Subscribe(topic string, handler func(Message)) error {
	l := log.With().Str("topic", topic).Logger()

	l.Trace().Msg("Subscribe")

	return m.m.Subscribe(topic, func(m Message) {
		wm := newLoggedMessage(m)
		wm.l.Trace().Msg("Received")

		handler(wm)
	})
}

func (m *Logged) Publish(ctx context.Context, topic string, message []byte) error {
	l := log.With().Str("topic", topic).Bytes("message", message).Logger()

	if err := m.m.Publish(ctx, topic, message); err != nil {
		l.Trace().Err(err).Msg("Failed to publish")
		return err
	}

	l.Trace().Msg("Published")

	return nil
}

var _ Message = &LoggedMessage{}

type LoggedMessage struct {
	m Message
	l zerolog.Logger
}

func newLoggedMessage(m Message) *LoggedMessage {
	return &LoggedMessage{
		m: m,
		l: log.With().Str("msg", m.String()).Logger(),
	}
}

func (m LoggedMessage) Context() context.Context {
	return m.m.Context()
}

func (m LoggedMessage) Topic() string {
	return m.m.Topic()
}

func (m LoggedMessage) Data() []byte {
	return m.m.Data()
}

func (m LoggedMessage) String() string {
	return m.m.String()
}

func (m LoggedMessage) Respond(message []byte) error {
	l := m.l.With().Bytes("response", message).Logger()

	if err := m.m.Respond(message); err != nil {
		l.Error().Err(err).Msg("Failed to respond")
		return err
	}

	l.Trace().Msg("Responded")

	return nil
}

func (m LoggedMessage) Nak() error {
	if err := m.m.Nak(); err != nil {
		m.l.Error().Err(err).Msg("Failed to Nak")
		return err
	}

	m.l.Trace().Msg("Nak")

	return nil
}

func (m LoggedMessage) Ack() error {
	if err := m.m.Ack(); err != nil {
		m.l.Error().Err(err).Msg("Failed to Ack")
		return err
	}

	m.l.Trace().Msg("Ack")

	return nil
}
