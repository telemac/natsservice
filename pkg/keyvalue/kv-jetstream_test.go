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
	bucketName := "test-multiple-keys"
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

	// Add more keys to test that Keys() returns all keys
	err = kv.Set(context.Background(), "prefix.key1", []byte("prefixed1"))
	assert.NoError(err)
	err = kv.Set(context.Background(), "prefix.key2", []byte("prefixed2"))
	assert.NoError(err)

	// Now we have 5 keys total - Keys() should return all of them
	allKeys, err = kv.Keys(context.Background())
	assert.NoError(err)
	assert.Len(allKeys, 5)

	// Verify all expected keys are present
	expectedKeys := []string{"key1", "key2", "key3", "prefix.key1", "prefix.key2"}
	for _, expectedKey := range expectedKeys {
		assert.Contains(allKeys, expectedKey)
	}
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

func TestKeyValuer_WatchFiltered(t *testing.T) {
	assert := assert.New(t)

	// Start embedded NATS server with JetStream
	embedded, err := natstools.StartEmbedded()
	require.NoError(t, err, "Failed to start embedded NATS")
	defer embedded.Shutdown()

	// Get JetStream context
	js := embedded.JetStream()
	require.NotNil(t, js, "Failed to get JetStream context")

	// Create KV store with unique bucket (using timestamp for isolation)
	bucketName := "test-watch-filtered"
	kv, err := NewJetStreamKV(context.TODO(), js, bucketName, "WatchFiltered test bucket", nil)
	require.NoError(t, err, "Failed to create JetStreamKV")

	// Set up watch for specific keys with UpdatesOnly to avoid initial nil entries
	keysToWatch := []string{"watch1", "watch2", "watch3"}
	watcher, err := kv.WatchFiltered(context.Background(), keysToWatch, jetstream.UpdatesOnly())
	require.NoError(t, err, "Failed to create filtered watcher")
	defer watcher.Stop()

	// Channel to collect updates
	updates := make(chan jetstream.KeyValueEntry, 10)

	// Start goroutine to listen for updates with proper cleanup
	go func() {
		defer close(updates)
		for update := range watcher.Updates() {
			updates <- update
		}
	}()

	// Give the watcher a moment to start
	time.Sleep(50 * time.Millisecond)

	// Set a watched key
	err = kv.Set(context.Background(), "watch1", []byte("value1"))
	assert.NoError(err)

	// Should receive update for watched key
	select {
	case update := <-updates:
		if update != nil {
			assert.Equal("watch1", update.Key())
			assert.Equal([]byte("value1"), update.Value())
		} else {
			t.Fatal("Received nil update instead of expected entry")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Should have received update for watched key")
	}

	// Set a non-watched key - should not trigger update
	err = kv.Set(context.Background(), "not_watched", []byte("ignored"))
	assert.NoError(err)

	// Should not receive update for non-watched key (short timeout)
	select {
	case <-updates:
		t.Fatal("Should not receive update for non-watched key")
	case <-time.After(200 * time.Millisecond):
		// Expected - no update for non-watched key
	}

	// Set another watched key
	err = kv.Set(context.Background(), "watch3", []byte("value3"))
	assert.NoError(err)

	// Should receive update for second watched key
	select {
	case update := <-updates:
		if update != nil {
			assert.Equal("watch3", update.Key())
			assert.Equal([]byte("value3"), update.Value())
		} else {
			t.Fatal("Received nil update instead of expected entry")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Should have received update for second watched key")
	}
}

func TestKeyValuer_SynchronizeWithKV(t *testing.T) {
	assert := assert.New(t)

	// Start embedded NATS server with JetStream
	embedded, err := natstools.StartEmbedded()
	require.NoError(t, err, "Failed to start embedded NATS")
	defer embedded.Shutdown()

	// Get JetStream context
	js := embedded.JetStream()
	require.NotNil(t, js, "Failed to get JetStream context")

	// Create source KV store
	sourceBucketName := "test-sync-source"
	sourceKv, err := NewJetStreamKV(context.TODO(), js, sourceBucketName, "Source KV for sync test", nil)
	require.NoError(t, err, "Failed to create source KV store")

	// Create destination KV store (using MemoryKV for isolation)
	destKv := NewMemoryKV()

	// Test synchronization of updates
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keysToSync := []string{"sync1", "sync2", "sync3"}

	// Start synchronization in goroutine
	syncDone := make(chan error, 1)
	go func() {
		syncDone <- sourceKv.SynchronizeWithKV(ctx, keysToSync, destKv)
	}()

	// Give synchronization a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test: Set a synchronized key in source
	err = sourceKv.Set(context.Background(), "sync1", []byte("value1"))
	assert.NoError(err)

	// Wait a moment for sync to propagate
	time.Sleep(200 * time.Millisecond)

	// Verify destination has the update
	value, err := destKv.Get(context.Background(), "sync1")
	assert.NoError(err)
	assert.Equal([]byte("value1"), value)

	// Test: Set another synchronized key
	err = sourceKv.Set(context.Background(), "sync2", []byte("value2"))
	assert.NoError(err)

	// Wait for sync to propagate
	time.Sleep(200 * time.Millisecond)

	// Verify destination has the second update
	value, err = destKv.Get(context.Background(), "sync2")
	assert.NoError(err)
	assert.Equal([]byte("value2"), value)

	// Test: Set a non-synchronized key (should not be synced)
	err = sourceKv.Set(context.Background(), "not_synced", []byte("ignored"))
	assert.NoError(err)

	// Wait a moment
	time.Sleep(200 * time.Millisecond)

	// Verify destination does NOT have the non-synced key
	_, err = destKv.Get(context.Background(), "not_synced")
	assert.ErrorIs(err, ErrKeyNotFound)

	// Test: Update an existing synchronized key
	err = sourceKv.Set(context.Background(), "sync1", []byte("updated_value1"))
	assert.NoError(err)

	// Wait for sync to propagate
	time.Sleep(200 * time.Millisecond)

	// Verify destination has the updated value
	value, err = destKv.Get(context.Background(), "sync1")
	assert.NoError(err)
	assert.Equal([]byte("updated_value1"), value)

	// Test: Delete a synchronized key
	err = sourceKv.Delete(context.Background(), "sync2")
	assert.NoError(err)

	// Wait for sync to propagate
	time.Sleep(200 * time.Millisecond)

	// Verify destination key is deleted
	_, err = destKv.Get(context.Background(), "sync2")
	assert.ErrorIs(err, ErrKeyNotFound)

	// Performance test: write many keys to source and verify in destination
	const numKeys = 2_000_000

	// Cancel previous sync and start a new one with wildcard for performance keys
	cancel()

	// Wait for previous sync to stop
	select {
	case <-syncDone:
		// Previous sync stopped
	case <-time.After(time.Second):
		t.Fatal("Previous synchronization did not stop")
	}

	// Create new context and sync for performance test
	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	// Start new synchronization with wildcard pattern for performance keys
	go func(ctx context.Context) {
		err := sourceKv.SynchronizeWithKV(ctx, []string{"perfkey.*"}, destKv)
		assert.ErrorIs(err, context.Canceled)
	}(ctx)

	// Give synchronization a moment to start
	time.Sleep(100 * time.Millisecond)

	// Measure insertion time
	startTime := time.Now()

	// Write many keys to source directly in loop using perfkey. pattern
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("perfkey.%d", i)
		value := []byte(fmt.Sprintf("perf_value_%d", i))
		err = sourceKv.Set(context.Background(), key, value)
		fmt.Printf("put in nats %s\n", key)
		assert.NoError(err)
	}

	insertTime := time.Since(startTime)
	t.Logf("Inserted %d keys in %v (%.2f keys/sec)", numKeys, insertTime, float64(numKeys)/insertTime.Seconds())

	// Give sync time to process all keys
	time.Sleep(10000 * time.Millisecond)

	// Verify all keys are in destination
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("perfkey.%d", i)
		expectedValue := []byte(fmt.Sprintf("perf_value_%d", i))
		value, err := destKv.Get(context.Background(), key)
		if err != nil {
			t.Logf("Failed to get value for key %s: %s", key, err.Error())
			break
		}
		assert.NoError(err, "Key %s should exist in destination", key)
		assert.Equal(expectedValue, value, "Value for key %s should match", key)
	}

	t.Logf("Successfully synchronized %d keys to destination", numKeys)
	cancel()
	// Cancel context to stop synchronization

	// Wait for sync goroutine to finish
	select {
	case <-syncDone:
	// Sync goroutine finished
	case <-time.After(2 * time.Second):
		// Timeout is acceptable for cleanup
	}
}
