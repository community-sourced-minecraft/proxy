package messaging

import (
	"context"
)

type Messager interface {
	Subscribe(topic string, handler func(msg Message)) error
	Publish(ctx context.Context, topic string, message []byte) error
}

type Message interface {
	String() string

	Context() context.Context
	Topic() string
	Data() []byte
	Respond(message []byte) error
	Nak() error
	Ack() error
}
