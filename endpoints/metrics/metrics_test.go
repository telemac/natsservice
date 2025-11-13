package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/nats-io/nats.go/micro"
	"github.com/stretchr/testify/assert"
)

// mockCollector implements the Collector interface for testing
type mockCollector struct{}

func (m *mockCollector) CollectAllMetrics(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"test": "data",
	}, nil
}

// mockRequest implements micro.Request for testing
type mockRequest struct {
	data     []byte
	response []byte
	ctx      context.Context
}

func (m *mockRequest) Respond(data []byte, opts ...micro.RespondOpt) error {
	m.response = data
	return nil
}

func (m *mockRequest) RespondJSON(v interface{}, opts ...micro.RespondOpt) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	m.response = data
	return nil
}

func (m *mockRequest) Error(code, description string, data []byte, opts ...micro.RespondOpt) error {
	return nil
}

func (m *mockRequest) Data() []byte {
	return m.data
}

func (m *mockRequest) Subject() string {
	return "test.subject"
}

func (m *mockRequest) Reply() string {
	return "test.reply"
}

func (m *mockRequest) Headers() micro.Headers {
	return nil
}

func (m *mockRequest) Context() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

func TestMetricsEndpoint_Config(t *testing.T) {
	assert := assert.New(t)

	collector := &mockCollector{}
	endpoint := NewEndpoint(collector)
	config := endpoint.Config()

	assert.NotNil(config)
	assert.Equal("metrics", config.Name)
}

func TestMetricsEndpoint_Handle(t *testing.T) {
	assert := assert.New(t)

	collector := &mockCollector{}
	endpoint := NewEndpoint(collector)
	req := &mockRequest{
		data: []byte("{}"),
		ctx:  context.Background(),
	}

	// Should not panic
	endpoint.Handle(req)

	// Should have responded
	assert.NotEmpty(req.response)

	// Parse response
	var resp MetricsResponse
	err := json.Unmarshal(req.response, &resp)
	assert.NoError(err)

	// Should have timestamp
	assert.False(resp.Timestamp.IsZero())

	// Should have metrics or error
	assert.True(len(resp.Metrics) > 0 || resp.Error != "")
}

func TestMetricsEndpoint_CustomKeyFunc(t *testing.T) {
	assert := assert.New(t)

	collector := &mockCollector{}
	customKeyFunc := func(tenantID, location, machineID string) string {
		return fmt.Sprintf("custom-%s-%s-%s", tenantID, location, machineID)
	}

	endpoint := NewEndpointWithKV(&EndpointConfig{
		Collector: collector,
		TenantID:  "test",
		Location:  "dev",
		MachineID: "machine1",
		KeyFunc:   customKeyFunc,
	})

	// Test key generation
	key := endpoint.generateKey()
	assert.Equal("custom-test-dev-machine1", key)
}

func TestMetricsEndpoint_DefaultKeyFunc(t *testing.T) {
	assert := assert.New(t)

	collector := &mockCollector{}
	endpoint := NewEndpointWithKV(&EndpointConfig{
		Collector: collector,
		TenantID:  "test",
		Location:  "dev",
		MachineID: "machine1",
		KeyFunc:   nil, // Use default
	})

	// Test key generation
	key := endpoint.generateKey()
	assert.Equal("metrics.test.dev.machine1", key)
}
