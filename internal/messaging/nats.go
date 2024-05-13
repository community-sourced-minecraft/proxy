package messaging

import (
	"context"
	"fmt"
	"strings"

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
		handler(&NATSMessage{
			m:   msg,
			ctx: context.Background(),
		})
	})

	return err
}

func (n *NATSMessager) Publish(_ctx context.Context, topic string, message []byte) error {
	return n.nc.Publish(topic, message)
}

var _ Message = &NATSMessage{}

type NATSMessage struct {
	m   *nats.Msg
	ctx context.Context
}

func (m NATSMessage) Context() context.Context {
	return m.ctx
}

func (m NATSMessage) Topic() string {
	return m.m.Subject
}

func (m NATSMessage) Data() []byte {
	return m.m.Data
}

func (m NATSMessage) String() string {
	sb := strings.Builder{}

	sb.WriteString(m.m.Subject)
	sb.WriteString(" ")
	if m.m.Reply != "" {
		sb.WriteString(" (<- ")
		sb.WriteString(m.m.Reply)
		sb.WriteString(") ")
	}
	sb.WriteString("-> ")
	sb.WriteString(fmt.Sprint(len(m.m.Data)))
	sb.WriteString(" bytes")

	return sb.String()
}

func (m *NATSMessage) Respond(message []byte) error {
	return m.m.Respond(message)
}

func (m *NATSMessage) Nak() error {
	return m.m.Nak()
}

func (m *NATSMessage) Ack() error {
	return m.m.Ack()
}
