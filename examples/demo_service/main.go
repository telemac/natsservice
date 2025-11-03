package main

import (
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/telemac/goutils/task"
	"github.com/telemac/natsservice"
	"github.com/telemac/natsservice/examples/demo_service/endpoints/add"
	"github.com/telemac/natsservice/examples/demo_service/endpoints/endpoint1"
	"github.com/telemac/natsservice/examples/demo_service/endpoints/endpoint_can_panic"
	"github.com/telemac/natsservice/examples/demo_service/pkg/counter"
)

func main() {
	// Create cancellable context with 5s timeout
	ctx, cancel := task.NewCancellableContext(time.Second * 5)
	defer cancel()

	// Initialize logger with version
	log := slog.Default().With("version", "0.0.1")

	// Connect to NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Error("Failed to connect to NATS", "error", err)
		return
	}
	defer nc.Close()

	var js jetstream.JetStream
	js, err = jetstream.New(nc)
	if err != nil {
		log.Error("Failed to create JetStream", "error", err)
	}

	service, err := natsservice.StartService(&natsservice.ServiceConfig{
		Ctx:         ctx,
		Nc:          nc,
		Js:          js,
		Logger:      log.With("service", "demo-service"),
		Name:        "demo-service",
		Group:       "demo",
		Version:     "0.0.1",
		Description: "demo service",
		Metadata:    nil,
	})

	if err != nil {
		log.Error("Failed to start service", "error", err)
		return
	}
	defer service.Stop()

	commonCounter := &counter.CommonCounter{}

	endpoin1 := endpoint1.New(commonCounter)
	endpointCanPanic := endpoint_can_panic.New(commonCounter)
	addEndpoint := add.New()

	err = service.AddEndpoints(endpoin1, endpointCanPanic, addEndpoint)
	if err != nil {
		log.Error("Failed to add endpoint 1", "error", err)
		return
	}

	<-ctx.Done()
}
