package keyvalue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/telemac/natsservice/pkg/typeregistry"
)

// MemoryKV implements a thread-safe in-memory key-value store
type MemoryKV struct {
	mu       sync.RWMutex
	data     map[string][]byte
	registry *typeregistry.Registry
}

// Ensure MemoryKV implements KeyValuer and TypedKeyValuer
var _ KeyValuer = (*MemoryKV)(nil)
var _ TypedKeyValuer = (*MemoryKV)(nil)

// NewMemoryKV creates a new in-memory key-value store
func NewMemoryKV() *MemoryKV {
	return &MemoryKV{
		data: make(map[string][]byte),
	}
}

// NewMemoryKVWithOptions creates a new in-memory key-value store with options
func NewMemoryKVWithOptions(registry *typeregistry.Registry) *MemoryKV {
	return &MemoryKV{
		data:     make(map[string][]byte),
		registry: registry,
	}
}

// Set stores a key-value pair
func (m *MemoryKV) Set(ctx context.Context, key string, value []byte, opts ...SetOption) error {
	if key == "" {
		return ErrEmptyKey
	}

	// Process options (TTL is not supported in memory implementation)
	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}
	if options.ttl > 0 {
		return fmt.Errorf("TTL is not supported in memory implementation")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Handle nil value specially
	if value == nil {
		m.data[key] = nil
	} else {
		// Store a copy of the value to prevent external modifications
		valueCopy := make([]byte, len(value))
		copy(valueCopy, value)
		m.data[key] = valueCopy
	}

	return nil
}

// Get retrieves a value by key
func (m *MemoryKV) Get(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.data[key]
	if !exists {
		return nil, ErrKeyNotFound
	}

	// Handle nil value specially
	if value == nil {
		return nil, nil
	}

	// Return a copy to prevent external modifications
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	return valueCopy, nil
}

// Delete removes a key-value pair
func (m *MemoryKV) Delete(ctx context.Context, key string) error {
	if key == "" {
		return ErrEmptyKey
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	return nil
}

// Exists checks if a key exists
func (m *MemoryKV) Exists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, ErrEmptyKey
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.data[key]
	return exists, nil
}

// SetTyped stores a typed key-value pair
func (m *MemoryKV) SetTyped(ctx context.Context, key string, value interface{}, opts ...SetOption) error {
	if m.registry == nil {
		return fmt.Errorf("type registry is required for typed operations")
	}

	if key == "" {
		return ErrEmptyKey
	}

	// Process options (TTL is not supported in memory implementation)
	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}
	if options.ttl > 0 {
		return fmt.Errorf("TTL is not supported in memory implementation")
	}

	// Marshal the value with type information
	typedData, err := m.registry.MarshalTypedData(value)
	if err != nil {
		return fmt.Errorf("failed to marshal typed data: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert TypedData to JSON bytes for storage
	typedJSON, err := json.Marshal(typedData)
	if err != nil {
		return fmt.Errorf("failed to marshal typed data to JSON: %w", err)
	}

	m.data[key] = typedJSON
	return nil
}

// GetTyped retrieves a typed value by key
func (m *MemoryKV) GetTyped(ctx context.Context, key string) (interface{}, error) {
	if m.registry == nil {
		return nil, fmt.Errorf("type registry is required for typed operations")
	}

	if key == "" {
		return nil, ErrEmptyKey
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.data[key]
	if !exists {
		return nil, ErrKeyNotFound
	}

	// Unmarshal the value with type information
	var typedData typeregistry.TypedData
	if err := json.Unmarshal(data, &typedData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal typed data JSON: %w", err)
	}

	value, err := m.registry.UnmarshalTypedData(&typedData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal typed data: %w", err)
	}

	return value, nil
}

// DeleteTyped removes a typed key-value pair
func (m *MemoryKV) DeleteTyped(ctx context.Context, key string) error {
	return m.Delete(ctx, key)
}