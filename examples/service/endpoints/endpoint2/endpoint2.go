package endpoint2

import (
	"github.com/nats-io/nats.go/micro"
	"github.com/telemac/natsservice"
	"github.com/telemac/natsservice/examples/service/pkg/counter"
)

var _ natsservice.Endpointer = (*Endpoint2)(nil)

type Endpoint2 struct {
	natsservice.Endpoint
	Common *counter.CommonCounter
}

func New(common *counter.CommonCounter) *Endpoint2 {
	return &Endpoint2{Common: common}
}

func (e *Endpoint2) Config() *natsservice.EndpointConfig {
	return &natsservice.EndpointConfig{
		Name: "endpoint2",
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

func (e *Endpoint2) Handle(request micro.Request) {
	log := e.Service().Config().Logger
	e.Common.Counter++
	log.Info("endpoint handler",
		"service", e.Service().Config().Name,
		"endpoint", e.Config().Name,
		"version", e.Service().Config().Version,
		"counter", e.Common.Counter)
	request.RespondJSON(e.Common.Counter)
}
