/*
Package natstools provides utilities for working with embedded NATS servers and JetStream.

The package offers simple APIs for common use cases while supporting advanced configurations
when needed. The primary focus is on providing zero-friction embedded NATS servers that can
run in-process without TCP overhead, perfect for testing and embedded scenarios.

# Embedded Server

The embedded server functionality allows you to run a full NATS server within your Go application.
It supports two primary modes:

1. In-Process Only: Connections use net.Pipe internally, bypassing TCP entirely. This provides
   the best performance and is ideal for testing and single-process applications.

2. Hybrid Mode: Supports both in-process and TCP connections, allowing external clients to
   connect while maintaining fast in-process communication.

# Basic Usage

The simplest way to start an embedded server:

	srv, err := natstools.StartEmbedded()
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Shutdown()

	// Use the in-process connection
	nc := srv.Connection()
	js := srv.JetStream()

# Testing

For tests, use the TestServer helper:

	func TestMyFeature(t *testing.T) {
		srv, cleanup := natstools.TestServer(t)
		defer cleanup()

		// Your test code here
		nc := srv.Connection()
		// ...
	}

# Advanced Configuration

For production or advanced use cases:

	opts := &natstools.EmbeddedOptions{
		InProcessOnly:   false,        // Allow TCP connections
		Port:            4222,          // Specific port
		Host:            "0.0.0.0",     // Listen on all interfaces
		StoreOnDisk:     true,          // Persist JetStream data
		DataDir:         "/var/lib/nats",
		EnableJetStream: true,
		MaxMemory:       1024 * 1024 * 1024, // 1GB
		EnableLogging:   true,
		LogLevel:        "INFO",
	}

	srv, err := natstools.StartEmbeddedWithOptions(opts)

# Performance

In-process connections provide significant performance benefits over TCP:

- Zero network overhead
- No serialization/deserialization for transport
- Direct memory transfer
- No port conflicts
- Faster startup/shutdown

Benchmarks show in-process connections can be 2-3x faster than TCP for small messages
and even more for larger payloads.

# JetStream Support

JetStream is enabled by default and provides:

- Persistent messaging
- Stream processing
- Key/Value stores
- Object stores
- Message replay and retention

Example with JetStream:

	srv, _ := natstools.StartEmbedded()
	js := srv.JetStream()

	// Create a stream
	stream, _ := js.CreateStream(ctx, nats.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"orders.>"},
	})

# Multiple Connections

You can create multiple in-process connections to the same server:

	srv, _ := natstools.StartEmbedded()

	// Primary connection
	nc1 := srv.Connection()

	// Additional connections
	nc2, _ := srv.NewConnection()
	nc3, _ := srv.NewConnection()

This is useful for testing multi-client scenarios or isolating different parts
of your application.

# Production Use

While primarily designed for testing and development, the embedded server can be used
in production for specific use cases:

- Embedded applications that need messaging
- Desktop applications with background processing
- Edge computing scenarios
- Microservices that benefit from co-located messaging

For production use, consider:

- Enabling persistence with StoreOnDisk
- Setting appropriate memory limits
- Configuring logging for observability
- Using TCP mode if external access is needed
*/
package natstools