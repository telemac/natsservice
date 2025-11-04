package natstools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// EmbeddedServer wraps an embedded NATS server with convenient access methods
type EmbeddedServer struct {
	server  *server.Server
	nc      *nats.Conn // In-process connection
	js      jetstream.JetStream
	opts    *EmbeddedOptions
	tcpConn *nats.Conn // Optional TCP connection
}

// EmbeddedOptions configures the embedded NATS server
type EmbeddedOptions struct {
	// Connection mode
	InProcessOnly bool // If true, use only in-process (no TCP)

	// TCP options (when not InProcessOnly)
	Port int    // 0 for random, -1 for no TCP
	Host string // Default "127.0.0.1"

	// Storage
	DataDir      string // Empty for memory-only
	JetStreamDir string // Empty for temp dir
	StoreOnDisk  bool   // Persist data

	// JetStream
	EnableJetStream bool  // Default true
	MaxMemory       int64 // JetStream memory (default 256MB)
	MaxStore        int64 // JetStream disk (default 1GB)

	// Logging
	EnableLogging bool   // Server logging
	LogLevel      string // DEBUG, INFO, WARN, ERROR

	// Advanced
	ClusterName string   // For clustering
	Routes      []string // Cluster routes
}

// DefaultOptions returns sensible defaults for embedded server
func DefaultOptions() *EmbeddedOptions {
	return &EmbeddedOptions{
		InProcessOnly:   false,
		DataDir:         "/tmp/embedded-test-nats",
		Host:            "127.0.0.1",
		Port:            4222,
		EnableJetStream: true,
		MaxMemory:       256 * 1024 * 1024,  // 256MB
		MaxStore:        1024 * 1024 * 1024, // 1GB
		EnableLogging:   false,
		LogLevel:        "ERROR",
	}
}

// StartEmbedded starts an in-process only embedded NATS server with JetStream
// This is the simplest way to get started - perfect for tests and development
func StartEmbedded() (*EmbeddedServer, error) {
	return StartEmbeddedWithOptions(DefaultOptions())
}

// StartEmbeddedInProcess starts an embedded server with custom options but forced in-process mode
func StartEmbeddedInProcess(opts *EmbeddedOptions) (*EmbeddedServer, error) {
	if opts == nil {
		opts = DefaultOptions()
	}
	opts.InProcessOnly = true
	return StartEmbeddedWithOptions(opts)
}

// StartEmbeddedWithOptions starts an embedded NATS server with full configuration control
func StartEmbeddedWithOptions(opts *EmbeddedOptions) (*EmbeddedServer, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Build server options
	serverOpts := &server.Options{
		DontListen:     opts.InProcessOnly,
		Host:           opts.Host,
		Port:           opts.Port,
		NoLog:          !opts.EnableLogging,
		NoSigs:         true,
		MaxControlLine: 2048,
		MaxPayload:     1024 * 1024, // 1MB default
	}

	// Configure storage directories
	if opts.DataDir != "" {
		serverOpts.StoreDir = opts.DataDir
	}

	// Configure JetStream if enabled
	if opts.EnableJetStream {
		serverOpts.JetStream = true
		serverOpts.JetStreamMaxMemory = opts.MaxMemory
		if serverOpts.JetStreamMaxMemory == 0 {
			serverOpts.JetStreamMaxMemory = 256 * 1024 * 1024 // 256MB default
		}

		serverOpts.JetStreamMaxStore = opts.MaxStore
		if serverOpts.JetStreamMaxStore == 0 {
			serverOpts.JetStreamMaxStore = 1024 * 1024 * 1024 // 1GB default
		}

		// Set JetStream storage directory
		if opts.JetStreamDir != "" {
			serverOpts.StoreDir = opts.JetStreamDir
		} else if opts.StoreOnDisk && opts.DataDir == "" {
			// Create temp dir if persisting but no dir specified
			tmpDir, err := os.MkdirTemp("", "nats-jetstream-*")
			if err != nil {
				return nil, fmt.Errorf("failed to create temp dir: %w", err)
			}
			serverOpts.StoreDir = tmpDir
		}
	}

	// Configure clustering if specified
	if opts.ClusterName != "" {
		serverOpts.Cluster = server.ClusterOpts{
			Name: opts.ClusterName,
			Host: opts.Host,
			Port: -1, // Cluster port will be assigned
		}
	}

	// Add routes if specified
	if len(opts.Routes) > 0 {
		// Routes configuration would go here
		// This would require URL parsing
	}

	// Set log level
	if opts.EnableLogging {
		switch opts.LogLevel {
		case "DEBUG":
			serverOpts.Debug = true
			serverOpts.Trace = true
		case "INFO":
			serverOpts.Debug = false
			serverOpts.Trace = false
		case "WARN", "ERROR":
			serverOpts.Debug = false
			serverOpts.Trace = false
		}
	}

	// Create and start server
	srv, err := server.NewServer(serverOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	if opts.EnableLogging {
		srv.ConfigureLogger()
	}

	// Start the server
	go srv.Start()

	// Wait for server to be ready
	if !srv.ReadyForConnections(5 * time.Second) {
		srv.Shutdown()
		return nil, fmt.Errorf("server failed to start within timeout")
	}

	// Create in-process connection
	nc, err := nats.Connect("", nats.InProcessServer(srv))
	if err != nil {
		srv.Shutdown()
		return nil, fmt.Errorf("failed to create in-process connection: %w", err)
	}

	// Setup JetStream if enabled
	var js jetstream.JetStream
	if opts.EnableJetStream {
		js, err = jetstream.New(nc)
		if err != nil {
			nc.Close()
			srv.Shutdown()
			return nil, fmt.Errorf("failed to create JetStream context: %w", err)
		}
	}

	es := &EmbeddedServer{
		server: srv,
		nc:     nc,
		js:     js,
		opts:   opts,
	}

	return es, nil
}

// TestServer creates an embedded server for testing with automatic cleanup
func TestServer(t *testing.T) (*EmbeddedServer, func()) {
	t.Helper()

	srv, err := StartEmbedded()
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}

	cleanup := func() {
		if err := srv.Shutdown(); err != nil {
			t.Errorf("Failed to shutdown test server: %v", err)
		}
	}

	return srv, cleanup
}

// Connection returns the in-process connection (always available)
func (e *EmbeddedServer) Connection() *nats.Conn {
	return e.nc
}

// JetStream returns the JetStream context, or nil if JetStream is not enabled
func (e *EmbeddedServer) JetStream() jetstream.JetStream {
	return e.js
}

// NewConnection creates an additional in-process connection to the server
func (e *EmbeddedServer) NewConnection() (*nats.Conn, error) {
	return nats.Connect("", nats.InProcessServer(e.server))
}

// ClientURL returns the TCP URL for client connections (empty if InProcessOnly)
func (e *EmbeddedServer) ClientURL() string {
	if e.opts.InProcessOnly {
		return ""
	}
	return e.server.ClientURL()
}

// NewTCPConnection creates a new TCP connection to the server (error if InProcessOnly)
func (e *EmbeddedServer) NewTCPConnection() (*nats.Conn, error) {
	if e.opts.InProcessOnly {
		return nil, fmt.Errorf("server is configured for in-process only connections")
	}

	url := e.ClientURL()
	if url == "" {
		return nil, fmt.Errorf("no TCP URL available")
	}

	return nats.Connect(url)
}

// Server returns the underlying NATS server instance
func (e *EmbeddedServer) Server() *server.Server {
	return e.server
}

// IsRunning checks if the server is still running
func (e *EmbeddedServer) IsRunning() bool {
	if e.server == nil {
		return false
	}
	// Check if server is still responding to connections
	return e.server.Running()
}

// Shutdown gracefully shuts down the embedded server and closes all connections
func (e *EmbeddedServer) Shutdown() error {
	if e.tcpConn != nil {
		e.tcpConn.Close()
	}

	if e.nc != nil {
		e.nc.Close()
	}

	if e.server != nil {
		e.server.Shutdown()

		// Clean up temp directory if it was created
		if e.opts.JetStreamDir == "" && e.opts.StoreOnDisk && e.server.StoreDir() != "" {
			if filepath.HasPrefix(e.server.StoreDir(), os.TempDir()) {
				os.RemoveAll(e.server.StoreDir())
			}
		}
	}

	return nil
}

// WaitForShutdown waits for the server to shutdown or context cancellation
func (e *EmbeddedServer) WaitForShutdown(ctx context.Context) error {
	if e.server == nil {
		return fmt.Errorf("server not initialized")
	}

	done := make(chan struct{})
	go func() {
		e.server.WaitForShutdown()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

// NumClients returns the number of connected clients
func (e *EmbeddedServer) NumClients() int {
	if e.server == nil {
		return 0
	}
	return e.server.NumClients()
}
