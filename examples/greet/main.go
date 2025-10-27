package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"github.com/telemac/natsservice"
)

type GreetingEndpoint struct {
	natsservice.Endpoint
}

func (e *GreetingEndpoint) Config() *natsservice.EndpointConfig {
	return &natsservice.EndpointConfig{
		Name:       "greet",
		QueueGroup: "workers",
	}
}

func (e *GreetingEndpoint) Handle(req micro.Request) {
	name := string(req.Data())
	message := fmt.Sprintf("Hello, %s !", name)
	req.Respond([]byte(message))
}

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		slog.Error("Failed to connect to NATS", "error", err)
		return
	}
	defer nc.Close()

	svc, err := natsservice.StartService(&natsservice.ServiceConfig{
		Ctx:     context.Background(),
		Nc:      nc,
		Logger:  slog.Default(),
		Name:    "demo",
		Group:   "demo", // all subjects will be prefixed with "demo."
		Version: "0.1.0",
	})
	if err != nil {
		slog.Error("Failed to start service", "error", err)
		return
	}
	defer svc.Stop()

	err = svc.AddEndpoint(&GreetingEndpoint{})
	if err != nil {
		slog.Error("Failed to add endpoint", "error", err)
		return
	}

	select {} // Keep running
}
