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

		handler(m)
	})
}

func (l *Logged) Publish(ctx context.Context, topic string, message []byte) error {
	log.Printf("Publish %s/%s", topic, message)

	return l.m.Publish(ctx, topic, message)
}

func (l *Logged) Nak(msg Message) error {
	log.Printf("Nak %s", msg)

	return l.m.Nak(msg)
}

func (l *Logged) Ack(msg Message) error {
	log.Printf("Ack %s", msg)

	return l.m.Ack(msg)
}
