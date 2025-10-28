package add

import (
	"encoding/json"

	"github.com/nats-io/nats.go/micro"
	"github.com/telemac/natsservice"
)

var _ natsservice.Endpointer = (*AddEndpoint)(nil)

type AddRequest struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type AddResponse struct {
	Result float64 `json:"result"`
}

type AddEndpoint struct {
	natsservice.Endpoint
}

func New() *AddEndpoint {
	return &AddEndpoint{}
}

func (e *AddEndpoint) Config() *natsservice.EndpointConfig {
	serviceName := e.Service().Config().Name
	return &natsservice.EndpointConfig{
		Name: "add",
		Metadata: map[string]string{
			"service": e.Service().Config().Name,
			"version": "1.0.0",
			"author":  "telemac",
		},
		QueueGroup: serviceName + ".add",
		Subject:    "",
		UserData:   nil,
	}
}

func (e *AddEndpoint) Handle(request micro.Request) {
	log := e.Service().Logger().With(
		"endpoint", e.Config().Name,
	)

	var req AddRequest
	err := json.Unmarshal(request.Data(), &req)
	if err != nil {
		log.Error("failed to unmarshal request", "error", err)
		request.Error("400", "invalid request format", nil)
		return
	}

	result := req.A + req.B

	log.Info("add operation",
		"service", e.Service().Config().Name,
		"endpoint", e.Config().Name,
		"version", e.Service().Config().Version,
		"a", req.A,
		"b", req.B,
		"result", result)

	response := AddResponse{Result: result}
	request.RespondJSON(response)
}
