package hosting

import (
	"log"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pkg/errors"
)

type NATS struct {
	nc *nats.Conn
	js jetstream.JetStream
}

func Init() (*NATS, error) {
	natsUrl := os.Getenv("NATS_URL")

	nc, err := nats.Connect(natsUrl)
	if err != nil {
		log.Fatal(err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create JetStream")
	}

	return &NATS{
		nc: nc,
		js: js,
	}, nil
}

func (n *NATS) JetStream() jetstream.JetStream {
	return n.js
}

func (n *NATS) NATS() *nats.Conn {
	return n.nc
}
