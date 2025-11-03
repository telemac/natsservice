package keyvalue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telemac/natsservice/pkg/natstools"
	"github.com/telemac/natsservice/pkg/typeregistry"
)

// Test struct for typed operations
type TestUser struct {
	ID    string
	Name  string
	Email string
	Age   int
}

type TestProduct struct {
	SKU   string
	Name  string
	Price float64
}

func setupTestKV(t *testing.T, withRegistry bool) (*JetStreamKV, func()) {
	assert := assert.New(t)

	// Start embedded NATS server with JetStream
	embedded, err := natstools.StartEmbedded()
	require.NoError(t, err, "Failed to start embedded NATS")

	// Get JetStream context
	js := embedded.JetStream()
	require.NotNil(t, js, "Failed to get JetStream context")

	// Setup type registry if needed
	var registry *typeregistry.Registry
	if withRegistry {
		registry = typeregistry.New()
		err = typeregistry.Register[TestUser](registry, "test.User")
		assert.NoError(err)
		err = typeregistry.Register[TestProduct](registry, "test.Product")
		assert.NoError(err)
	}

	// Create KV store
	kv, err := NewJetStreamKV(context.TODO(), js, "test-bucket", "Test bucket for unit tests", registry)
	require.NoError(t, err, "Failed to create JetStreamKV")

	cleanup := func() {
		embedded.Shutdown()
	}

	return kv, cleanup
}

func TestKeyValuer_BasicOperations(t *testing.T) {
	assert := assert.New(t)
	kv, cleanup := setupTestKV(t, false)
	defer cleanup()

	// Test Set and Get
	key := "test-key"
	value := []byte("test-value")

	err := kv.Set(context.Background(), key, value)
	assert.NoError(err)

	retrieved, err := kv.Get(context.Background(), key)
	assert.NoError(err)
	assert.Equal(value, retrieved)

	// Test Exists
	exists, err := kv.Exists(context.Background(), key)
	assert.NoError(err)
	assert.True(exists)

	// Test non-existent key
	exists, err = kv.Exists(context.Background(), "non-existent")
	assert.NoError(err)
	assert.False(exists)

	// Test Get non-existent key
	_, err = kv.Get(context.Background(), "non-existent")
	assert.ErrorIs(err, ErrKeyNotFound)

	// Test Delete
	err = kv.Delete(context.Background(), key)
	assert.NoError(err)

	exists, err = kv.Exists(context.Background(), key)
	assert.NoError(err)
	assert.False(exists)

	// Test Delete non-existent key (should not error)
	err = kv.Delete(context.Background(), "non-existent")
	assert.NoError(err)
}

func TestKeyValuer_EmptyKey(t *testing.T) {
	assert := assert.New(t)
	kv, cleanup := setupTestKV(t, false)
	defer cleanup()

	// Test empty key operations
	err := kv.Set(context.Background(), "", []byte("value"))
	assert.ErrorIs(err, ErrEmptyKey)

	_, err = kv.Get(context.Background(), "")
	assert.ErrorIs(err, ErrEmptyKey)

	err = kv.Delete(context.Background(), "")
	assert.ErrorIs(err, ErrEmptyKey)

	_, err = kv.Exists(context.Background(), "")
	assert.ErrorIs(err, ErrEmptyKey)
}

func TestKeyValuer_LargeValue(t *testing.T) {
	assert := assert.New(t)
	kv, cleanup := setupTestKV(t, false)
	defer cleanup()

	// Test with large value (1MB)
	largeValue := make([]byte, 1024*1024)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	err := kv.Set(context.Background(), "large-key", largeValue)
	assert.NoError(err)

	retrieved, err := kv.Get(context.Background(), "large-key")
	assert.NoError(err)
	assert.Equal(largeValue, retrieved)
}

func TestKeyValuer_MultipleKeys(t *testing.T) {
	assert := assert.New(t)

	// Start embedded NATS server with JetStream
	embedded, err := natstools.StartEmbedded()
	require.NoError(t, err, "Failed to start embedded NATS")
	defer embedded.Shutdown()

	// Get JetStream context
	js := embedded.JetStream()
	require.NotNil(t, js, "Failed to get JetStream context")

	// Create KV store with unique bucket (using timestamp for isolation)
	bucketName := fmt.Sprintf("test-multiple-keys-%d", time.Now().UnixNano())
	kv, err := NewJetStreamKV(context.TODO(), js, bucketName, "Multiple keys test bucket", nil)
	require.NoError(t, err, "Failed to create JetStreamKV")

	// Set multiple keys
	keys := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	for k, v := range keys {
		err := kv.Set(context.Background(), k, v)
		assert.NoError(err)
	}

	// Verify all keys exist
	for k, expectedValue := range keys {
		value, err := kv.Get(context.Background(), k)
		assert.NoError(err)
		assert.Equal(expectedValue, value)
	}

	// Test Keys() method - should have just the 3 keys we set
	allKeys, err := kv.Keys(context.Background())
	assert.NoError(err)
	assert.Len(allKeys, 3)

	// Test Keys with prefix (using dot instead of colon for NATS KV)
	err = kv.Set(context.Background(), "prefix.key1", []byte("prefixed1"))
	assert.NoError(err)
	err = kv.Set(context.Background(), "prefix.key2", []byte("prefixed2"))
	assert.NoError(err)

	// Now we have 5 keys total
	allKeys, err = kv.Keys(context.Background())
	assert.NoError(err)
	assert.Len(allKeys, 5)

	// Test prefix filtering
	prefixedKeys, err := kv.Keys(context.Background(), "prefix.")
	assert.NoError(err)
	assert.Len(prefixedKeys, 2)
}

func TestTypedKeyValuer_Operations(t *testing.T) {
	assert := assert.New(t)
	kv, cleanup := setupTestKV(t, true)
	defer cleanup()

	// Test SetTyped and GetTyped with User
	user := &TestUser{
		ID:    "user-123",
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	err := kv.SetTyped(context.Background(), "user.123", user)
	assert.NoError(err)

	retrieved, err := kv.GetTyped(context.Background(), "user.123")
	assert.NoError(err)

	retrievedUser, ok := retrieved.(*TestUser)
	assert.True(ok)
	assert.Equal(user, retrievedUser)

	// Test with Product
	product := &TestProduct{
		SKU:   "PROD-001",
		Name:  "Laptop",
		Price: 999.99,
	}

	err = kv.SetTyped(context.Background(), "product.001", product)
	assert.NoError(err)

	retrieved, err = kv.GetTyped(context.Background(), "product.001")
	assert.NoError(err)

	retrievedProduct, ok := retrieved.(*TestProduct)
	assert.True(ok)
	assert.Equal(product, retrievedProduct)

	// Test DeleteTyped
	err = kv.DeleteTyped(context.Background(), "user.123")
	assert.NoError(err)

	_, err = kv.GetTyped(context.Background(), "user.123")
	assert.ErrorIs(err, ErrKeyNotFound)
}

func TestTypedKeyValuer_NoRegistry(t *testing.T) {
	assert := assert.New(t)
	kv, cleanup := setupTestKV(t, false) // No registry
	defer cleanup()

	user := &TestUser{
		ID:   "user-123",
		Name: "John Doe",
	}

	// Should error without registry
	err := kv.SetTyped(context.Background(), "user.123", user)
	assert.Error(err)
	assert.Contains(err.Error(), "registry is required")

	// GetTyped should also error
	_, err = kv.GetTyped(context.Background(), "user.123")
	assert.Error(err)
	assert.Contains(err.Error(), "registry is required")
}

func TestKeyValuer_History(t *testing.T) {
	assert := assert.New(t)

	// Start embedded NATS
	embedded, err := natstools.StartEmbedded()
	require.NoError(t, err)
	defer embedded.Shutdown()

	js := embedded.JetStream()
	require.NotNil(t, js)

	// Create KV with history enabled using direct JetStream configuration
	kv, err := NewJetStreamKVWithOptions(context.TODO(), js, &jetstream.KeyValueConfig{
		Bucket:  "history-bucket",
		History: 10, // Keep 10 versions
	}, nil)
	require.NoError(t, err)

	key := "versioned-key"

	// Set multiple versions
	for i := 1; i <= 5; i++ {
		value := []byte(string(rune('a' + i - 1)))
		err := kv.Set(context.Background(), key, value)
		assert.NoError(err)
	}

	// Get current value
	current, err := kv.Get(context.Background(), key)
	assert.NoError(err)
	assert.Equal([]byte("e"), current)

	// Get history
	history, err := kv.History(context.Background(), key)
	assert.NoError(err)
	assert.GreaterOrEqual(len(history), 5)

	// Get specific revision
	if len(history) >= 2 {
		oldValue, err := kv.GetRevision(context.Background(), key, history[0].Revision())
		assert.NoError(err)
		assert.Equal([]byte("a"), oldValue)
	}
}

func TestKeyValuer_Purge(t *testing.T) {
	assert := assert.New(t)

	// Start embedded NATS
	embedded, err := natstools.StartEmbedded()
	require.NoError(t, err)
	defer embedded.Shutdown()

	js := embedded.JetStream()
	require.NotNil(t, js)

	// Create KV with history using direct JetStream configuration
	kv, err := NewJetStreamKVWithOptions(context.TODO(), js, &jetstream.KeyValueConfig{
		Bucket:  "purge-bucket",
		History: 10, // Keep 10 versions
	}, nil)
	require.NoError(t, err)

	key := "purge-key"

	// Set multiple versions
	for i := 1; i <= 3; i++ {
		err := kv.Set(context.Background(), key, []byte(string(rune('a'+i-1))))
		assert.NoError(err)
	}

	// Purge all versions
	err = kv.Purge(context.Background(), key)
	assert.NoError(err)

	// Key should not exist
	exists, err := kv.Exists(context.Background(), key)
	assert.NoError(err)
	assert.False(exists)

	// After purge, attempting to get should return key not found
	_, err = kv.Get(context.Background(), key)
	assert.ErrorIs(err, ErrKeyNotFound)
}

func TestKeyValuer_Status(t *testing.T) {
	assert := assert.New(t)
	kv, cleanup := setupTestKV(t, false)
	defer cleanup()

	// Add some data
	err := kv.Set(context.Background(), "key1", []byte("value1"))
	assert.NoError(err)
	err = kv.Set(context.Background(), "key2", []byte("value2"))
	assert.NoError(err)

	// Get status
	status, err := kv.Status(context.Background())
	assert.NoError(err)
	assert.NotNil(status)
	assert.Equal("test-bucket", status.Bucket())
	assert.GreaterOrEqual(status.Values(), uint64(2))
}

func TestBucketOptions(t *testing.T) {
	assert := assert.New(t)

	// Start embedded NATS
	embedded, err := natstools.StartEmbedded()
	require.NoError(t, err)
	defer embedded.Shutdown()

	js := embedded.JetStream()
	require.NotNil(t, js)

	// Create KV with various options using direct JetStream configuration
	kv, err := NewJetStreamKVWithOptions(context.TODO(), js, &jetstream.KeyValueConfig{
		Bucket:      "options-bucket",
		Description: "Test bucket with options",
		History:     5,                       // Keep 5 versions
		Replicas:    1,                       // Single replica
		MaxBytes:    1024 * 1024,             // 1MB max bucket size
		Storage:     jetstream.MemoryStorage, // In-memory storage
		// Note: MaxValueSize is set at stream level, not in KeyValueConfig
	}, nil)
	require.NoError(t, err)

	// Verify bucket was created
	status, err := kv.Status(context.Background())
	assert.NoError(err)
	assert.Equal("options-bucket", status.Bucket())
}

func TestKeyValuer_Concurrent(t *testing.T) {
	assert := assert.New(t)
	kv, cleanup := setupTestKV(t, false)
	defer cleanup()

	// Run concurrent operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			key := string(rune('a' + id))
			value := []byte(key)

			// Set
			err := kv.Set(context.Background(), key, value)
			assert.NoError(err)

			// Get
			retrieved, err := kv.Get(context.Background(), key)
			assert.NoError(err)
			assert.Equal(value, retrieved)

			// Delete
			err = kv.Delete(context.Background(), key)
			assert.NoError(err)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
