package natsservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

// Servicer defines a service interface for managing endpoints and configuration.
// Stop stops the service and performs cleanup operations.
// Config retrieves the service's current configuration.
// AddEndpoint registers a new endpoint with the service.
type Servicer interface {
	Stop() error
	Config() *ServiceConfig
	AddEndpoint(endpointer Endpointer) error
	AddEndpoints(endpointer ...Endpointer) error
}

var _ Servicer = (*Service)(nil)

type Service struct {
	config   *ServiceConfig
	microSvc micro.Service
}

type ServiceConfig struct {
	Ctx         context.Context   // Service context for cancellation
	Nc          *nats.Conn        // NATS connection
	Logger      *slog.Logger      // Service logger
	Name        string            `json:"name"`               // Service name
	Group       string            `json:"group"`              // group, prefix all endpoint subjects if not empty
	Version     string            `json:"version"`            // Service version (must be SerVer)
	Description string            `json:"description"`        // Service description
	Metadata    map[string]string `json:"metadata,omitempty"` // Additional metadata
}

// Validate checks that all required fields are present
func (sc *ServiceConfig) Validate() error {
	if sc.Ctx == nil {
		return errors.New("missing context")
	}
	if sc.Nc == nil {
		return errors.New("nats connection required")
	}
	if sc.Logger == nil {
		return errors.New("logger required")
	}
	if sc.Name == "" {
		return errors.New("service name required")
	}
	if sc.Version == "" {
		return errors.New("service version required")
	}
	return nil
}

// StartService initializes and starts the NATS microservice
func StartService(config *ServiceConfig) (*Service, error) {
	svc := &Service{}
	// Validate configuration
	err := config.Validate()
	if err != nil {
		return svc, fmt.Errorf("invalid service config: %w", err)
	}
	svc.config = config

	// Build micro service configuration
	microConfig := micro.Config{
		Name:               svc.config.Name,
		Version:            svc.config.Version,
		Description:        svc.config.Description,
		Metadata:           svc.config.Metadata,
		QueueGroupDisabled: true,
	}

	// Create micro service
	svc.microSvc, err = micro.AddService(svc.config.Nc, microConfig)
	if err != nil {
		return svc, err
	}

	return svc, err
}

// Stop gracefully stops the NATS microservice
func (svc *Service) Stop() error {
	return svc.microSvc.Stop()
}

//func (svc *Service) Micro() micro.Service {
//	return svc.microSvc
//}

// Config returns the current service configuration
func (svc *Service) Config() *ServiceConfig {
	return svc.config
}

func (svc *Service) AddEndpoint(endpointer Endpointer) error {
	if endpointer == nil {
		return errors.New("nil endpointer")
	}
	endpointer.SetService(svc)
	config := endpointer.Config()
	if config == nil {
		return errors.New("missing endpoint config")
	}
	if config.Name == "" {
		return errors.New("missing endpoint name")
	}

	// Build endpoint options
	var opts []micro.EndpointOpt

	// Configure subject
	if config.Subject != "" {
		opts = append(opts, micro.WithEndpointSubject(config.Subject))
	}

	// Configure metadata
	if len(config.Metadata) > 0 && len(config.Metadata) == 0 {
		opts = append(opts, micro.WithEndpointMetadata(config.Metadata))
	}

	// Configure queue group
	if config.QueueGroup != "" {
		opts = append(opts, micro.WithEndpointQueueGroup(config.QueueGroup))
	} else {
		opts = append(opts, micro.WithEndpointQueueGroupDisabled())
	}

	if svc.config.Group != "" {
		return svc.microSvc.AddGroup(svc.config.Group).AddEndpoint(config.Name, endpointer, opts...)
	} else {
		return svc.microSvc.AddEndpoint(config.Name, endpointer, opts...)
	}
}

func (svc *Service) AddEndpoints(endpoints ...Endpointer) error {
	for _, endpoint := range endpoints {
		err := svc.AddEndpoint(endpoint)
		if err != nil {
			endpointName := endpoint.Config().Name
			return fmt.Errorf("could not add endpoint %s: %w", endpointName, err)
		}
	}
	return nil
}
