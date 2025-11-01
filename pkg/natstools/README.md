# natstools

NATS utilities for embedded servers and JetStream operations.

## Installation

```bash
go get github.com/telemac/natsservice/pkg/natstools
```

## Embedded NATS Server

Run a NATS server in-process without TCP overhead using `net.Pipe` internally.

### Quick Start

```go
import "github.com/telemac/natsservice/pkg/natstools"

// Simplest - in-process only with JetStream
srv, err := natstools.StartEmbedded()
defer srv.Shutdown()

nc := srv.Connection()
js := srv.JetStream()
```

### Configuration

```go
opts := &natstools.EmbeddedOptions{
    InProcessOnly:   false,        // Enable TCP if needed
    Port:            4222,         // 0 for random, -1 for none
    Host:            "127.0.0.1",
    EnableJetStream: true,
    MaxMemory:       256 << 20,    // 256MB
    MaxStore:        1 << 30,      // 1GB
    StoreOnDisk:     true,         // Persist data
    DataDir:         "/var/lib/nats",
}

srv, err := natstools.StartEmbeddedWithOptions(opts)
```

### Testing

```go
func TestFeature(t *testing.T) {
    srv, cleanup := natstools.TestServer(t)
    defer cleanup()

    // Use srv.Connection() and srv.JetStream()
}
```

### Multiple Connections

```go
srv, _ := natstools.StartEmbedded()
nc1 := srv.Connection()
nc2, _ := srv.NewConnection()  // Additional in-process connection
nc3, _ := srv.NewTCPConnection() // TCP connection (if not InProcessOnly)
```

### Performance

Benchmarks show in-process connections are ~14% faster than TCP:
- **In-Process**: 265 ns/op, 338 B/op
- **TCP**: 308 ns/op, 285 B/op

### API Reference

#### Server Management
- `StartEmbedded() (*EmbeddedServer, error)` - Start with defaults
- `StartEmbeddedWithOptions(opts) (*EmbeddedServer, error)` - Start with options
- `TestServer(t) (*EmbeddedServer, func())` - Test helper with cleanup

#### Connections
- `Connection() *nats.Conn` - Get in-process connection
- `NewConnection() (*nats.Conn, error)` - Create new in-process connection
- `NewTCPConnection() (*nats.Conn, error)` - Create TCP connection
- `JetStream() jetstream.JetStream` - Get JetStream context

#### Server Control
- `Shutdown() error` - Graceful shutdown
- `WaitForShutdown(ctx) error` - Wait with context
- `IsRunning() bool` - Check server status
- `NumClients() int` - Connected client count

## Use Cases

- **Testing**: No port conflicts, automatic cleanup
- **Development**: Fast iteration without external dependencies
- **Embedded Apps**: Ship NATS within your application
- **Microservices**: Co-located messaging for reduced latency

## Requirements

- Go 1.21+
- github.com/nats-io/nats.go v1.47+
- github.com/nats-io/nats-server/v2 v2.12+