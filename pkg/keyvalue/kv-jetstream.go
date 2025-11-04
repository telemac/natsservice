package keyvalue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/telemac/natsservice/pkg/typeregistry"
)

// JetStreamKV implements KeyValuer and TypedKeyValuer using NATS JetStream
type JetStreamKV struct {
	bucket   jetstream.KeyValue
	registry *typeregistry.Registry
}

// NewJetStreamKV creates a new JetStream-backed key-value store with default configuration
func NewJetStreamKV(ctx context.Context, js jetstream.JetStream, bucketName, description string, registry *typeregistry.Registry) (*JetStreamKV, error) {
	// Use NewJetStreamKVWithOptions with default configuration
	cfg := &jetstream.KeyValueConfig{
		Bucket:      bucketName,
		Description: description,
	}
	return NewJetStreamKVWithOptions(ctx, js, cfg, registry)
}

// NewJetStreamKVWithOptions creates a new JetStream KV store with custom configuration
func NewJetStreamKVWithOptions(ctx context.Context, js jetstream.JetStream, cfg *jetstream.KeyValueConfig, registry *typeregistry.Registry) (*JetStreamKV, error) {
	if js == nil {
		return nil, errors.New("jetstream instance is required")
	}
	if cfg == nil {
		return nil, errors.New("keyvalue config is required")
	}
	if cfg.Bucket == "" {
		return nil, errors.New("bucket name is required")
	}

	bucket, err := js.CreateOrUpdateKeyValue(ctx, *cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create or bind bucket: %w", err)
	}

	return &JetStreamKV{
		bucket:   bucket,
		registry: registry,
	}, nil
}

// --- KeyValuer Implementation ---

// Set stores a key-value pair
func (kv *JetStreamKV) Set(ctx context.Context, key string, value []byte, opts ...SetOption) error {
	if key == "" {
		return ErrEmptyKey
	}

	options := &setOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Per-key TTL is not supported in NATS JetStream KV
	// Use bucket-level TTL via WithDefaultTTL when creating the bucket
	if options.ttl > 0 {
		return fmt.Errorf("per-key TTL is not supported; use bucket-level TTL via WithDefaultTTL when creating the KV store")
	}

	_, err := kv.bucket.Put(ctx, key, value)
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	return nil
}

// Get retrieves a value by key
func (kv *JetStreamKV) Get(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}

	entry, err := kv.bucket.Get(ctx, key)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}

	return entry.Value(), nil
}

// Delete removes a key from the store
// WARNING : if the key does not exist, Delete won't return an error with jetstream kv
func (kv *JetStreamKV) Delete(ctx context.Context, key string) error {
	if key == "" {
		return ErrEmptyKey
	}

	err := kv.bucket.Purge(ctx, key)
	if err != nil && !errors.Is(err, jetstream.ErrKeyNotFound) {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	return nil
}

// Exists checks if a key exists without retrieving its value
func (kv *JetStreamKV) Exists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, ErrEmptyKey
	}

	_, err := kv.bucket.Get(ctx, key)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check key existence %s: %w", key, err)
	}

	return true, nil
}

// --- TypedKeyValuer Implementation ---

// SetTyped stores a typed value with automatic marshaling
func (kv *JetStreamKV) SetTyped(ctx context.Context, key string, value interface{}, opts ...SetOption) error {
	if kv.registry == nil {
		return errors.New("type registry is required for typed operations")
	}

	// Marshal value with type information
	typed, err := kv.registry.MarshalTypedData(value)
	if err != nil {
		return fmt.Errorf("failed to marshal typed value: %w", err)
	}

	// Convert TypedData to JSON bytes
	data, err := json.Marshal(typed)
	if err != nil {
		return fmt.Errorf("failed to marshal typed data: %w", err)
	}

	// Store using regular Set
	return kv.Set(ctx, key, data, opts...)
}

// GetTyped retrieves and unmarshals a typed value
func (kv *JetStreamKV) GetTyped(ctx context.Context, key string) (interface{}, error) {
	if kv.registry == nil {
		return nil, errors.New("type registry is required for typed operations")
	}

	// Get raw bytes
	data, err := kv.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// Unmarshal TypedData
	var typed typeregistry.TypedData
	if err := json.Unmarshal(data, &typed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal typed data: %w", err)
	}

	// Unmarshal to actual type using registry
	value, err := kv.registry.UnmarshalTypedData(&typed)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return value, nil
}

// DeleteTyped removes a typed key (same as Delete but for interface consistency)
func (kv *JetStreamKV) DeleteTyped(ctx context.Context, key string) error {
	return kv.Delete(ctx, key)
}

// --- Additional Helper Methods ---

// Keys returns all keys
func (kv *JetStreamKV) Keys(ctx context.Context) ([]string, error) {
	keyLister, err := kv.bucket.ListKeys(ctx, jetstream.IgnoreDeletes())
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	defer keyLister.Stop()

	var keys []string
	for key := range keyLister.Keys() {
		keys = append(keys, key)
	}

	return keys, nil
}

// Watch watches for changes to a key
func (kv *JetStreamKV) Watch(ctx context.Context, key string) (jetstream.KeyWatcher, error) {
	return kv.bucket.Watch(ctx, key)
}

// WatchAll watches for changes to all keys with optional prefix
func (kv *JetStreamKV) WatchAll(ctx context.Context) (jetstream.KeyWatcher, error) {
	return kv.bucket.WatchAll(ctx, jetstream.IgnoreDeletes())
}

// WatchFiltered watches multiple keys for changes based on specified filters and options. Returns a KeyWatcher or an error.
func (kv *JetStreamKV) WatchFiltered(ctx context.Context, keys []string, opts ...jetstream.WatchOpt) (jetstream.KeyWatcher, error) {
	return kv.bucket.WatchFiltered(ctx, keys, opts...)
}

// Purge deletes all versions of a key
func (kv *JetStreamKV) Purge(ctx context.Context, key string) error {
	if key == "" {
		return ErrEmptyKey
	}

	err := kv.bucket.Purge(ctx, key)
	if err != nil && !errors.Is(err, jetstream.ErrKeyNotFound) {
		return fmt.Errorf("failed to purge key %s: %w", key, err)
	}

	return nil
}

// Status returns the status of the KV bucket
func (kv *JetStreamKV) Status(ctx context.Context) (jetstream.KeyValueStatus, error) {
	return kv.bucket.Status(ctx)
}

// GetRevision gets a specific revision of a key
func (kv *JetStreamKV) GetRevision(ctx context.Context, key string, revision uint64) ([]byte, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}

	entry, err := kv.bucket.GetRevision(ctx, key, revision)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get key revision %s@%d: %w", key, revision, err)
	}

	return entry.Value(), nil
}

// History returns the history of values for a key
func (kv *JetStreamKV) History(ctx context.Context, key string) ([]jetstream.KeyValueEntry, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}

	entries, err := kv.bucket.History(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get key history %s: %w", key, err)
	}

	return entries, nil
}

// SynchronizeWithKV synchronizes a set of keys between the current KV store and a destination KeyValuer.
// It uses a filtered watcher to monitor changes to the specified keys and applies updates to the destination KV.
// Returns an error if the watcher fails, if the context is canceled, or if updates cannot be applied to the destination.
func (kv *JetStreamKV) SynchronizeWithKV(ctx context.Context, keys []string, destKv KeyValuer) error {
	keyWatcher, err := kv.WatchFiltered(ctx, keys)
	if err != nil {
		return err
	}
	defer func() {
		keyWatcher.Stop()
	}()

	count := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update, ok := <-keyWatcher.Updates():
			if !ok {
				return nil
			}
			if update == nil {
				continue // Skip nil entries
			}
			switch update.Operation() {
			case jetstream.KeyValuePut:
				err = destKv.Set(ctx, update.Key(), update.Value())
				if err != nil {
					return fmt.Errorf("failed to set value for key %s: %w", update.Key(), err)
				}
				//if count%100 == 0 {
				fmt.Printf("copy to kv %s\n", update.Key())
				//}
				count++
			case jetstream.KeyValueDelete:
				err = destKv.Delete(ctx, update.Key())
				if err != nil {
					return fmt.Errorf("failed to delete value for key %s: %w", update.Key(), err)
				}
			case jetstream.KeyValuePurge:
				err = destKv.Delete(ctx, update.Key())
			}

		}
	}
}
