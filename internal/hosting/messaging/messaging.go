package messaging

import (
	"context"
	"strings"
)

type Messager interface {
	Subscribe(topic string, handler func(msg Message)) error
	Publish(ctx context.Context, topic string, message []byte) error
	Ack(msg Message) error
	Nak(msg Message) error
}

type Message struct {
	m       Messager
	Context context.Context
	Topic   string
	Data    []byte
	ReplyTo *string
}

func (m Message) String() string {
	sb := strings.Builder{}

	sb.WriteString(m.Topic)
	sb.WriteString(" ")
	if m.ReplyTo != nil {
		sb.WriteString(" (<- ")
		sb.WriteString(*m.ReplyTo)
		sb.WriteString(") ")
	}
	sb.WriteString("-> ")
	sb.Write(m.Data)

	return sb.String()
}

func (m Message) Respond(message []byte) error {
	if m.ReplyTo == nil {
		return nil
	}

	return m.m.Publish(m.Context, *m.ReplyTo, message)
}

func (m Message) Nak() error {
	return m.m.Nak(m)
}

func (m Message) Ack() error {
	return m.m.Ack(m)
}
