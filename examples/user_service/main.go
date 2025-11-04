package main

import (
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/telemac/goutils/task"
	"github.com/telemac/natsservice"
	"github.com/telemac/natsservice/examples/user_service/endpoints"
	"github.com/telemac/natsservice/examples/user_service/pkg/user_store"
	keyvalue2 "github.com/telemac/natsservice/pkg/keyvalue"
)

const SERVICE_NAME = "user_service"
const SERVICE_VERSION = "1.0.0"

func main() {
	// Create cancellable context with 5s timeout
	ctx, cancel := task.NewCancellableContext(time.Second * 5)
	defer cancel()

	// Initialize logger with version
	log := slog.Default().With(
		"service", SERVICE_NAME,
		"version", SERVICE_VERSION,
	)

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
		Logger:      log,
		Name:        SERVICE_NAME,
		Group:       SERVICE_NAME,
		Version:     SERVICE_VERSION,
		Description: "User handling service",
		Metadata:    nil,
	})

	if err != nil {
		log.Error("Failed to start service", "error", err)
		return
	}
	defer service.Stop()

	// create jetstream kv user store
	kvStore, err := keyvalue2.NewJetStreamKV(ctx, js, "users", "", nil)
	if err != nil {
		log.Error("Failed to create key value store", "error", err)
		return
	}

	// create user store with undelrying jetstream kv
	userStore := user_store.NewKvUserStore(ctx, kvStore)

	err = service.AddEndpoints(
		endpoints.NewUserAddEndpoint(userStore),
		endpoints.NewUserGetEndpoint(userStore),
	)
	if err != nil {
		log.Error("Failed to add endpoints", "error", err)
		return
	}

	<-ctx.Done()
}
