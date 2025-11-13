package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nats.go/micro"
	"github.com/telemac/natsservice"
)

// Collector is the interface for collecting system metrics
// Implementations must be safe for concurrent use
type Collector interface {
	CollectAllMetrics(ctx context.Context) (map[string]interface{}, error)
}

// MetricsResponse represents the response structure for metrics requests
type MetricsResponse struct {
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
	Error     string                 `json:"error,omitempty"`
}

// Endpoint handles metrics requests using an injected collector
type Endpoint struct {
	natsservice.Endpoint
	collector Collector
	kv        jetstream.KeyValue
	ctx       context.Context
	tenantID  string
	location  string
	machineID string
	keyFunc   func(tenantID, location, machineID string) string
}

// EndpointConfig holds configuration for creating a metrics endpoint
type EndpointConfig struct {
	Collector Collector
	Kv        jetstream.KeyValue
	Ctx       context.Context
	TenantID  string
	Location  string
	MachineID string
	KeyFunc   func(tenantID, location, machineID string) string // Optional: custom key generation
}

// NewEndpoint creates a new metrics endpoint with the provided collector
func NewEndpoint(collector Collector) *Endpoint {
	return &Endpoint{
		collector: collector,
	}
}

// NewEndpointWithKV creates a new metrics endpoint with KV support
func NewEndpointWithKV(cfg *EndpointConfig) *Endpoint {
	return &Endpoint{
		collector: cfg.Collector,
		kv:        cfg.Kv,
		ctx:       cfg.Ctx,
		tenantID:  cfg.TenantID,
		location:  cfg.Location,
		machineID: cfg.MachineID,
		keyFunc:   cfg.KeyFunc,
	}
}

// Config returns the endpoint configuration
func (e *Endpoint) Config() *natsservice.EndpointConfig {
	return &natsservice.EndpointConfig{
		Name: "metrics",
	}
}

// Handle processes a metrics request and returns system metrics
func (e *Endpoint) Handle(req micro.Request) {
	defer natsservice.RecoverPanic(e, req)

	// Collect all metrics with background context
	metricsData, err := e.collector.CollectAllMetrics(context.Background())

	// Build response
	resp := MetricsResponse{
		Timestamp: time.Now(),
		Metrics:   metricsData,
	}
	if err != nil {
		resp.Error = err.Error()
	}

	data, _ := json.Marshal(resp)

	// Write to KV if available (on-demand update)
	if e.kv != nil && e.ctx != nil && e.tenantID != "" && e.location != "" && e.machineID != "" {
		key := e.generateKey()
		e.kv.Put(e.ctx, key, data)
		// Ignore errors - respond even if KV write fails
	}

	req.Respond(data)
}

// generateKey creates a KV key using custom KeyFunc or default format
func (e *Endpoint) generateKey() string {
	if e.keyFunc != nil {
		return e.keyFunc(e.tenantID, e.location, e.machineID)
	}
	// Default key format
	return fmt.Sprintf("metrics.%s.%s.%s", e.tenantID, e.location, e.machineID)
}
