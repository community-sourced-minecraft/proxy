package messaging

import (
	"context"
	"log"
)

var _ Messager = &Logged{}

type Logged struct {
	m Messager
}

func WithLogger(m Messager) *Logged {
	return &Logged{m: m}
}

func (l *Logged) Subscribe(topic string, handler func(Message)) error {
	log.Printf("Subscribe %s", topic)

	return l.m.Subscribe(topic, func(m Message) {
		log.Printf("Received %s", m)

		handler(&LoggedMessage{m: m})
	})
}

func (l *Logged) Publish(ctx context.Context, topic string, message []byte) error {
	if err := l.m.Publish(ctx, topic, message); err != nil {
		log.Printf("Failed to publish %s/%s: %v", topic, message, err)
		return err
	}

	log.Printf("Published %s/%s", topic, message)

	return nil
}

var _ Message = &LoggedMessage{}

type LoggedMessage struct {
	m Message
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
	if err := m.m.Respond(message); err != nil {
		log.Printf("DBG: Failed to respond to %s: %v", message, err)
		return err
	}

	log.Printf("DBG: Responded to %s", message)

	return nil
}

func (m LoggedMessage) Nak() error {
	if err := m.m.Nak(); err != nil {
		log.Printf("DBG: Failed to Nak: %v", err)
		return err
	}

	log.Printf("DBG: Nak")

	return nil
}

func (m LoggedMessage) Ack() error {
	if err := m.m.Ack(); err != nil {
		log.Printf("DBG: Failed to Ack: %v", err)
		return err
	}

	log.Printf("DBG: Ack")

	return nil
}
