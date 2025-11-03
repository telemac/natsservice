package keyvalue

import (
	"context"
	"errors"
	"time"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrEmptyKey    = errors.New("empty is key")
	ErrInvalidTTL  = errors.New("invalid TTL value")
)

// SetOption is a functional option for Set operations
type SetOption func(*setOptions)

// setOptions holds all options for a Set operation
type setOptions struct {
	ttl time.Duration
}

// WithTTL sets a TTL for the key
//
// IMPORTANT: Per-key TTL is NOT supported in our JetStream KV implementation.
// Calling Set() with this option will return an error.
//
// For TTL support, use bucket-level TTL
func WithTTL(ttl time.Duration) SetOption {
	return func(opts *setOptions) {
		opts.ttl = ttl
	}
}

// KeyValuer defines basic key-value operations
type KeyValuer interface {
	Set(ctx context.Context, key string, value []byte, opts ...SetOption) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// TypedKeyValuer defines typed key-value operations
type TypedKeyValuer interface {
	SetTyped(ctx context.Context, key string, value interface{}, opts ...SetOption) error
	GetTyped(ctx context.Context, key string) (interface{}, error)
	DeleteTyped(ctx context.Context, key string) error
}
