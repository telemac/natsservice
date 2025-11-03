package endpoint_can_panic

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
			"service":     e.Service().Config().Name,
			"description": "recovers from panic",
			"version":     "1.0.0",
			"author":      "telemac",
		},
	}
}

func (e *Endpoint2) Handle(request micro.Request) {
	// RecoverPanic protects the service from crashes by catching any panic
	// and returning a proper error response to the client
	defer natsservice.RecoverPanic(e, request)
	log := e.Service().Logger().With("endpoint", e.Config().Name)

	e.Common.Increment()
	if e.Common.Counter() > 5 {
		// This panic will be caught by RecoverPanic above
		panic("counter overflow")
	}
	log.Info("endpoint handler",
		"service", e.Service().Config().Name,
		"endpoint", e.Config().Name,
		"version", e.Service().Config().Version,
		"counter", e.Common.Counter())
	request.RespondJSON(e.Common.Counter())
}
