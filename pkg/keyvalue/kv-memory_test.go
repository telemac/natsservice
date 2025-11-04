package keyvalue

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/telemac/natsservice/pkg/typeregistry"
)

func TestMemoryKV_BasicOperations(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	kv := NewMemoryKV()

	// Test Set and Get
	err := kv.Set(ctx, "key1", []byte("value1"))
	assert.NoError(err)

	value, err := kv.Get(ctx, "key1")
	assert.NoError(err)
	assert.Equal([]byte("value1"), value)

	// Test Exists
	exists, err := kv.Exists(ctx, "key1")
	assert.NoError(err)
	assert.True(exists)

	exists, err = kv.Exists(ctx, "nonexistent")
	assert.NoError(err)
	assert.False(exists)

	// Test Delete
	err = kv.Delete(ctx, "key1")
	assert.NoError(err)

	_, err = kv.Get(ctx, "key1")
	assert.Error(err)
	assert.Equal(ErrKeyNotFound, err)

	exists, err = kv.Exists(ctx, "key1")
	assert.NoError(err)
	assert.False(exists)
}

func TestMemoryKV_EmptyKeyErrors(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	kv := NewMemoryKV()

	// Test Set with empty key
	err := kv.Set(ctx, "", []byte("value"))
	assert.Error(err)
	assert.Equal(ErrEmptyKey, err)

	// Test Get with empty key
	_, err = kv.Get(ctx, "")
	assert.Error(err)
	assert.Equal(ErrEmptyKey, err)

	// Test Delete with empty key
	err = kv.Delete(ctx, "")
	assert.Error(err)
	assert.Equal(ErrEmptyKey, err)

	// Test Exists with empty key
	_, err = kv.Exists(ctx, "")
	assert.Error(err)
	assert.Equal(ErrEmptyKey, err)
}

func TestMemoryKV_NilValue(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	kv := NewMemoryKV()

	// Set nil value
	err := kv.Set(ctx, "key", nil)
	assert.NoError(err)

	// Get nil value
	value, err := kv.Get(ctx, "key")
	assert.NoError(err)
	assert.Nil(value)

	// Key should exist
	exists, err := kv.Exists(ctx, "key")
	assert.NoError(err)
	assert.True(exists)
}

func TestMemoryKV_OverwriteValue(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	kv := NewMemoryKV()

	// Set initial value
	err := kv.Set(ctx, "key", []byte("value1"))
	assert.NoError(err)

	// Overwrite with new value
	err = kv.Set(ctx, "key", []byte("value2"))
	assert.NoError(err)

	// Verify new value
	value, err := kv.Get(ctx, "key")
	assert.NoError(err)
	assert.Equal([]byte("value2"), value)
}

func TestMemoryKV_DeleteNonExistentKey(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	kv := NewMemoryKV()

	// Delete non-existent key should not error
	err := kv.Delete(ctx, "nonexistent")
	assert.NoError(err)
}

func TestMemoryKV_TTLNotSupported(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	kv := NewMemoryKV()

	// Test Set with TTL
	err := kv.Set(ctx, "key", []byte("value"), WithTTL(time.Minute))
	assert.Error(err)
	assert.Contains(err.Error(), "TTL is not supported")
}

func TestMemoryKV_ConcurrentOperations(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	kv := NewMemoryKV()

	const numGoroutines = 100
	const numOperations = 1000

	var wg sync.WaitGroup
	errs := make(chan error, numGoroutines*4) // 4 operations per goroutine

	// Concurrent Set operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations/numGoroutines; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := []byte(fmt.Sprintf("value-%d-%d", id, j))
				if err := kv.Set(ctx, key, value); err != nil {
					errs <- err
				}
			}
		}(i)
	}

	// Concurrent Get operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations/numGoroutines; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				_, err := kv.Get(ctx, key)
				// Don't treat ErrKeyNotFound as an error for this test
				if err != nil && err != ErrKeyNotFound {
					errs <- err
				}
			}
		}(i)
	}

	// Concurrent Exists operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations/numGoroutines; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				_, err := kv.Exists(ctx, key)
				if err != nil {
					errs <- err
				}
			}
		}(i)
	}

	// Concurrent Delete operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations/numGoroutines; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				if err := kv.Delete(ctx, key); err != nil {
					errs <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	// Check for any errors
	for err := range errs {
		assert.NoError(err)
	}
}

func TestMemoryKV_TypedOperations(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	registry := typeregistry.New()
	kv := NewMemoryKVWithOptions(registry)

	// Register a test type
	err := typeregistry.Register[TestType](registry, "test.TestType")
	assert.NoError(err)

	// Test SetTyped and GetTyped
	testValue := &TestType{Name: "test", Age: 25}
	err = kv.SetTyped(ctx, "typed_key", testValue)
	assert.NoError(err)

	retrievedValue, err := kv.GetTyped(ctx, "typed_key")
	assert.NoError(err)

	// Type assert to verify the retrieved value
	typedValue, ok := retrievedValue.(*TestType)
	assert.True(ok)
	assert.Equal("test", typedValue.Name)
	assert.Equal(25, typedValue.Age)

	// Test DeleteTyped
	err = kv.DeleteTyped(ctx, "typed_key")
	assert.NoError(err)

	_, err = kv.GetTyped(ctx, "typed_key")
	assert.Error(err)
	assert.Equal(ErrKeyNotFound, err)
}

func TestMemoryKV_TypedOperationsWithoutRegistry(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	kv := NewMemoryKV() // No registry

	testValue := &TestType{Name: "test", Age: 25}

	// Test SetTyped without registry
	err := kv.SetTyped(ctx, "key", testValue)
	assert.Error(err)
	assert.Contains(err.Error(), "type registry is required")

	// Test GetTyped without registry
	_, err = kv.GetTyped(ctx, "key")
	assert.Error(err)
	assert.Contains(err.Error(), "type registry is required")
}

func TestMemoryKV_TypedOperationsWithEmptyKey(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	registry := typeregistry.New()
	kv := NewMemoryKVWithOptions(registry)

	testValue := &TestType{Name: "test", Age: 25}

	// Test SetTyped with empty key
	err := kv.SetTyped(ctx, "", testValue)
	assert.Error(err)
	assert.Equal(ErrEmptyKey, err)

	// Test GetTyped with empty key
	_, err = kv.GetTyped(ctx, "")
	assert.Error(err)
	assert.Equal(ErrEmptyKey, err)

	// Test DeleteTyped with empty key
	err = kv.DeleteTyped(ctx, "")
	assert.Error(err)
	assert.Equal(ErrEmptyKey, err)
}

func TestMemoryKV_ConcurrentTypedOperations(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	registry := typeregistry.New()
	kv := NewMemoryKVWithOptions(registry)

	// Register test type
	err := typeregistry.Register[TestType](registry, "test.TestType")
	assert.NoError(err)

	const numGoroutines = 50
	const numOperations = 100

	var wg sync.WaitGroup
	errs := make(chan error, numGoroutines*2)

	// Concurrent SetTyped operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations/numGoroutines; j++ {
				key := fmt.Sprintf("typed_key-%d-%d", id, j)
				value := &TestType{Name: fmt.Sprintf("name-%d-%d", id, j), Age: id + j}
				if err := kv.SetTyped(ctx, key, value); err != nil {
					errs <- err
				}
			}
		}(i)
	}

	// Concurrent GetTyped operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations/numGoroutines; j++ {
				key := fmt.Sprintf("typed_key-%d-%d", id, j)
				_, err := kv.GetTyped(ctx, key)
				// Don't treat ErrKeyNotFound as an error for this test
				if err != nil && err != ErrKeyNotFound {
					errs <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	// Check for any errors
	for err := range errs {
		assert.NoError(err)
	}
}

// TestType is a simple struct for typed operations testing
type TestType struct {
	Name string
	Age  int
}

func (t *TestType) TypeName() string {
	return "TestType"
}