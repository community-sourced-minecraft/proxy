package hosting

import (
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Hosting struct {
	nc   *nats.Conn
	js   jetstream.JetStream
	Info *PodInfo
}

func Init() (*Hosting, error) {
	nc, js, err := connectToNATS()
	if err != nil {
		return nil, err
	}

	return &Hosting{
		nc:   nc,
		js:   js,
		Info: ParsePodInfo(),
	}, nil
}

func (n *Hosting) JetStream() jetstream.JetStream {
	return n.js
}

func (n *Hosting) NATS() *nats.Conn {
	return n.nc
}
