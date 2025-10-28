package endpoint1

import (
	"github.com/nats-io/nats.go/micro"
	"github.com/telemac/natsservice"
	"github.com/telemac/natsservice/examples/service/pkg/counter"
)

var _ natsservice.Endpointer = (*Endpoint1)(nil)

type Endpoint1 struct {
	natsservice.Endpoint
	Common *counter.CommonCounter
}

func New(common *counter.CommonCounter) *Endpoint1 {
	return &Endpoint1{Common: common}
}

func (e *Endpoint1) Config() *natsservice.EndpointConfig {
	return &natsservice.EndpointConfig{
		Name: "endpoint1",
		Metadata: map[string]string{
			"service": e.Service().Config().Name,
			"version": "1.0.0",
			"author":  "telemac",
		},
		QueueGroup: "queue",
		Subject:    "",
		UserData:   12345,
	}
}

func (e *Endpoint1) Handle(request micro.Request) {
	log := e.Service().Logger()
	e.Common.Counter++
	log.Info("endpoint handler",
		"service", e.Service().Config().Name,
		"endpoint", e.Config().Name,
		"version", e.Service().Config().Version,
		"counter", e.Common.Counter)
	request.RespondJSON(e.Common.Counter)
}
