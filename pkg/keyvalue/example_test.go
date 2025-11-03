package keyvalue_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/telemac/natsservice/pkg/keyvalue"
	"github.com/telemac/natsservice/pkg/natstools"
	"github.com/telemac/natsservice/pkg/typeregistry"
)

// Example demonstrates basic usage of JetStream KV
func Example_basic() {
	// Start embedded NATS server with JetStream
	embedded, err := natstools.StartEmbedded()
	if err != nil {
		log.Fatal(err)
	}
	defer embedded.Shutdown()

	// Get JetStream context
	js := embedded.JetStream()

	// Create KV store
	kv, err := keyvalue.NewJetStreamKV(context.TODO(), js, "my-bucket", "Basic example bucket", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Set a key-value pair
	err = kv.Set(context.Background(), "greeting", []byte("Hello, World!"))
	if err != nil {
		log.Fatal(err)
	}

	// Get the value
	value, err := kv.Get(context.Background(), "greeting")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Value: %s\n", value)

	// Check if key exists
	exists, err := kv.Exists(context.Background(), "greeting")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Key exists: %v\n", exists)

	// Delete the key
	err = kv.Delete(context.Background(), "greeting")
	if err != nil {
		log.Fatal(err)
	}

	exists, err = kv.Exists(context.Background(), "greeting")
	if err != nil {
		log.Fatal(err)
	}

}

// Example_withTTL demonstrates bucket-level TTL
func Example_withTTL() {
	// Start embedded NATS server with JetStream
	embedded, err := natstools.StartEmbedded()
	if err != nil {
		log.Fatal(err)
	}
	defer embedded.Shutdown()

	// Get JetStream context
	js := embedded.JetStream()

	// Create KV store with bucket-level TTL using direct JetStream configuration
	// ALL keys in this bucket will expire after 5 seconds
	kv, err := keyvalue.NewJetStreamKVWithOptions(context.TODO(), js, &jetstream.KeyValueConfig{
		Bucket: "ttl-bucket",
		TTL:    5 * time.Second, // Keys expire after 5 seconds
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Set a key - it will automatically expire after 5 seconds
	err = kv.Set(context.Background(), "temp-key", []byte("expires soon"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Key set with bucket TTL")

	// Note: Per-key TTL via WithTTL() is NOT supported
	// Attempting to use it will return an error:
	// err = kv.Set(context.Background(), "key", []byte("value"), keyvalue.WithTTL(1*time.Second))
	// This will return: "per-key TTL is not supported"

	// Output:
	// Key set with bucket TTL
}

// ExampleUser demonstrates typed operations
type ExampleUser struct {
	ID    string
	Name  string
	Email string
}

func Example_typed() {
	// Start embedded NATS server
	embedded, err := natstools.StartEmbedded()
	if err != nil {
		log.Fatal(err)
	}
	defer embedded.Shutdown()

	// Create type registry
	registry := typeregistry.New()
	err = typeregistry.Register[ExampleUser](registry, "example.User")
	if err != nil {
		log.Fatal(err)
	}

	// Get JetStream context
	js := embedded.JetStream()

	// Create KV store with registry
	kv, err := keyvalue.NewJetStreamKV(context.TODO(), js, "typed-bucket", "Typed operations example bucket", registry)
	if err != nil {
		log.Fatal(err)
	}

	// Store a typed value
	user := &ExampleUser{
		ID:    "user-001",
		Name:  "Alice",
		Email: "alice@example.com",
	}

	err = kv.SetTyped(context.Background(), "user.alice", user)
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve typed value
	retrieved, err := kv.GetTyped(context.Background(), "user.alice")
	if err != nil {
		log.Fatal(err)
	}

	retrievedUser := retrieved.(*ExampleUser)
	fmt.Printf("User: %s (%s)\n", retrievedUser.Name, retrievedUser.Email)

	// Output:
	// User: Alice (alice@example.com)
}

// Example_bucketOptions demonstrates bucket configuration
func Example_bucketOptions() {
	// Start embedded NATS server
	embedded, err := natstools.StartEmbedded()
	if err != nil {
		log.Fatal(err)
	}
	defer embedded.Shutdown()

	js := embedded.JetStream()

	// Create KV store with various options using direct JetStream configuration
	kv, err := keyvalue.NewJetStreamKVWithOptions(context.TODO(), js, &jetstream.KeyValueConfig{
		Bucket:      "config-bucket",
		Description: "Application configuration", // Human-readable description
		History:     10,                          // Keep 10 versions per key
		Replicas:    1,                           // Single replica
		MaxBytes:    10 * 1024 * 1024,            // 10MB max bucket size
		Storage:     jetstream.MemoryStorage,     // In-memory storage
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Get bucket status
	status, err := kv.Status(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Bucket: %s\n", status.Bucket())

	// Output:
	// Bucket: config-bucket
}
