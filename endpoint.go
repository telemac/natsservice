package natsservice

import "github.com/nats-io/nats.go/micro"

// EndpointConfig holds configuration for individual endpoints
type EndpointConfig struct {
	Name       string            `json:"name"`                  // Endpoint name
	Metadata   map[string]string `json:"metadata,omitempty"`    // Endpoint metadata
	QueueGroup string            `json:"queue_group,omitempty"` // Queue group group
	Subject    string            `json:"subject,omitempty"`     // Custom subject
	UserData   any               `json:"-"`
}

var _ Endpointer = (*Endpoint)(nil)

type Endpoint struct {
	Endpointer
	config  *EndpointConfig
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
