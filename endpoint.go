package natsservice

import (
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go/micro"
)

// EndpointConfig holds configuration for individual endpoints
type EndpointConfig struct {
	Name       string            `json:"name"`                  // Endpoint name
	Metadata   map[string]string `json:"metadata,omitempty"`    // Endpoint metadata
	QueueGroup string            `json:"queue_group,omitempty"` // Queue group group
	Subject    string            `json:"subject,omitempty"`     // Custom subject
	UserData   any               `json:"-"`
}

// Endpoint is a base struct that provides common functionality for endpoints.
// It should be embedded in concrete endpoint implementations.
// Note: Endpoint does NOT implement Endpointer interface by itself -
// concrete implementations must provide Config() and Handle() methods.
type Endpoint struct {
	service *Service
}

func (e *Endpoint) Service() *Service {
	return e.service
}

func (e *Endpoint) SetService(s *Service) {
	e.service = s
}

type Endpointer interface {
	micro.Handler
	Config() *EndpointConfig
	Service() *Service
	SetService(*Service)
}

// UnmarshalRequest unmarshals request data and handles errors automatically
func UnmarshalRequest[T any](request micro.Request) (*T, error) {
	var result T
	if err := json.Unmarshal(request.Data(), &result); err != nil {
		request.Error("400", "invalid request format", nil)
		return nil, err
	}
	return &result, nil
}

// UnmarshalRequestWithLog unmarshals the request into the specified type and logs errors if unmarshaling fails.
func UnmarshalRequestWithLog[T any](request micro.Request, log *slog.Logger) (*T, error) {
	result, err := UnmarshalRequest[T](request)
	if err != nil {
		log.Error("failed to unmarshal request", "error", err)
	}
	return result, err
}

// RecoverPanic handles a panic occurring during the execution of an endpoint.
// It should be called as the first statement in an endpoint Handle method using `defer`.
//
// RecoverPanic automatically:
// - Catches any panic and prevents service crash
// - Logs the panic with service and endpoint context
// - Sends a "500 internal error" response to the client
//
// Usage:
//
//	func (e *MyEndpoint) Handle(request micro.Request) {
//		defer natsservice.RecoverPanic(e, request)
//		// Your endpoint logic here
//	}
//
// This ensures your service remains stable even when unexpected panics occur,
// providing graceful error handling and proper client responses.
func RecoverPanic(e Endpointer, request micro.Request) {
	if r := recover(); r != nil {
		log := e.Service().Logger().With(
			"service", e.Service().Config().Name,
			"endpoint", e.Config().Name,
		)
		log.Error("service endpoint panicked", "panic", r)
		request.Error("500", "internal error", nil)
	}
}
