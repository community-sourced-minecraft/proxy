package messaging

import (
	"context"

	"github.com/nats-io/nats.go"
)

var _ Messager = &NATSMessager{}

type NATSOptions struct {
	URL string `json:"url"`
}

type NATSMessager struct {
	nc *nats.Conn
}

func NewNATS(nc *nats.Conn) *NATSMessager {
	return &NATSMessager{nc: nc}
}

func (n *NATSMessager) Subscribe(topic string, handler func(Message)) error {
	_, err := n.nc.Subscribe(topic, func(msg *nats.Msg) {
		handler(Message{
			m:     n,
			Topic: topic,
			Data:  msg.Data,
		})
	})

	return err
}

func (n *NATSMessager) Publish(_ctx context.Context, topic string, message []byte) error {
	return n.nc.Publish(topic, message)
}

func (n *NATSMessager) Ack(msg Message) error {
	return msg.Nak()
}

func (n *NATSMessager) Nak(msg Message) error {
	return msg.Ack()
}
