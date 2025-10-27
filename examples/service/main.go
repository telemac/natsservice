package main

import (
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/telemac/goutils/task"
	"github.com/telemac/natsservice"
	"github.com/telemac/natsservice/examples/service/endpoints/endpoint1"
	"github.com/telemac/natsservice/examples/service/endpoints/endpoint2"
	"github.com/telemac/natsservice/examples/service/pkg/counter"
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

	service, err := natsservice.StartService(&natsservice.ServiceConfig{
		Ctx:         ctx,
		Nc:          nc,
		Logger:      log,
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
	endpoin2 := endpoint2.New(commonCounter)
	err = service.AddEndpoints(endpoin1, endpoin2)
	if err != nil {
		log.Error("Failed to add endpoint 1", "error", err)
		return
	}

	<-ctx.Done()
}
