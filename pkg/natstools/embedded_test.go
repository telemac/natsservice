package natstools

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartEmbedded(t *testing.T) {
	assert := assert.New(t)

	// Start embedded server with defaults
	srv, err := StartEmbedded()
	require.NoError(t, err)
	defer srv.Shutdown()

	// Verify server is running
	assert.True(srv.IsRunning())

	// Verify in-process connection works
	nc := srv.Connection()
	assert.NotNil(nc)
	assert.True(nc.IsConnected())

	// Verify JetStream is enabled
	js := srv.JetStream()
	assert.NotNil(js)

	// Verify no TCP URL (in-process only)
	assert.Empty(srv.ClientURL())

	// Test publishing and subscribing
	subject := "test.subject"
	message := []byte("Hello NATS")

	// Subscribe
	msgChan := make(chan *nats.Msg, 1)
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		msgChan <- msg
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Publish
	err = nc.Publish(subject, message)
	require.NoError(t, err)

	// Verify message received
	select {
	case msg := <-msgChan:
		assert.Equal(message, msg.Data)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestStartEmbeddedWithOptions(t *testing.T) {
	assert := assert.New(t)

	t.Run("InProcessOnly", func(t *testing.T) {
		opts := &EmbeddedOptions{
			InProcessOnly:   true,
			EnableJetStream: true,
			MaxMemory:       128 * 1024 * 1024, // 128MB
		}

		srv, err := StartEmbeddedWithOptions(opts)
		require.NoError(t, err)
		defer srv.Shutdown()

		assert.True(srv.IsRunning())
		assert.NotNil(srv.Connection())
		assert.NotNil(srv.JetStream())
		assert.Empty(srv.ClientURL())

		// Verify TCP connection fails
		_, err = srv.NewTCPConnection()
		assert.Error(err)
		assert.Contains(err.Error(), "in-process only")
	})


	t.Run("WithoutJetStream", func(t *testing.T) {
		opts := &EmbeddedOptions{
			InProcessOnly:   true,
			EnableJetStream: false,
		}

		srv, err := StartEmbeddedWithOptions(opts)
		require.NoError(t, err)
		defer srv.Shutdown()

		assert.True(srv.IsRunning())
		assert.NotNil(srv.Connection())
		assert.Nil(srv.JetStream())
	})
}

func TestNewConnection(t *testing.T) {
	assert := assert.New(t)

	srv, err := StartEmbedded()
	require.NoError(t, err)
	defer srv.Shutdown()

	// Create multiple in-process connections
	conn1 := srv.Connection()
	conn2, err := srv.NewConnection()
	require.NoError(t, err)
	defer conn2.Close()

	conn3, err := srv.NewConnection()
	require.NoError(t, err)
	defer conn3.Close()

	// All connections should be active
	assert.True(conn1.IsConnected())
	assert.True(conn2.IsConnected())
	assert.True(conn3.IsConnected())

	// Test cross-connection communication
	subject := "test.multi"
	received := make(chan int, 3)

	// Subscribe on each connection
	for i, conn := range []*nats.Conn{conn1, conn2, conn3} {
		connID := i + 1
		sub, err := conn.Subscribe(subject, func(msg *nats.Msg) {
			received <- connID
		})
		require.NoError(t, err)
		defer sub.Unsubscribe()
	}

	// Ensure subscriptions are ready
	time.Sleep(100 * time.Millisecond)

	// Publish from first connection
	err = conn1.Publish(subject, []byte("test"))
	require.NoError(t, err)

	// Flush to ensure message is sent
	err = conn1.Flush()
	require.NoError(t, err)

	// All should receive the message
	timeout := time.After(2 * time.Second)
	count := 0
	receivedConns := make(map[int]bool)
	for count < 3 {
		select {
		case connID := <-received:
			if !receivedConns[connID] {
				receivedConns[connID] = true
				count++
			}
		case <-timeout:
			t.Fatalf("Expected 3 messages, got %d (received from connections: %v)", count, receivedConns)
		}
	}
}

func TestTestServer(t *testing.T) {
	// This tests the test helper function
	srv, cleanup := TestServer(t)
	defer cleanup()

	assert.True(t, srv.IsRunning())
	assert.NotNil(t, srv.Connection())
	assert.True(t, srv.Connection().IsConnected())
}

func TestJetStream(t *testing.T) {
	assert := assert.New(t)

	srv, err := StartEmbedded()
	require.NoError(t, err)
	defer srv.Shutdown()

	js := srv.JetStream()
	require.NotNil(t, js)

	// Create a stream
	ctx := context.Background()
	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "TEST_STREAM",
		Subjects: []string{"test.>"},
		Storage:  jetstream.MemoryStorage,
	})
	require.NoError(t, err)
	assert.NotNil(stream)

	// Publish to the stream
	ack, err := js.Publish(ctx, "test.message", []byte("JetStream message"))
	require.NoError(t, err)
	assert.NotNil(ack)

	// Create a consumer
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          "test-consumer",
		Durable:       "test-consumer",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: "test.>",
	})
	require.NoError(t, err)
	assert.NotNil(consumer)

	// Fetch a message
	msgs, err := consumer.Fetch(1, jetstream.FetchMaxWait(2*time.Second))
	require.NoError(t, err)

	// Get message from the channel
	select {
	case receivedMsg := <-msgs.Messages():
		assert.Equal([]byte("JetStream message"), receivedMsg.Data())
		// Acknowledge the message
		err = receivedMsg.Ack()
		assert.NoError(err)
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestShutdown(t *testing.T) {
	assert := assert.New(t)

	srv, err := StartEmbedded()
	require.NoError(t, err)

	// Verify running
	assert.True(srv.IsRunning())
	nc := srv.Connection()
	assert.True(nc.IsConnected())

	// Shutdown
	err = srv.Shutdown()
	assert.NoError(err)

	// Verify stopped
	assert.False(srv.IsRunning())
	assert.False(nc.IsConnected())
}

func TestWaitForShutdown(t *testing.T) {
	assert := assert.New(t)

	srv, err := StartEmbedded()
	require.NoError(t, err)

	// Test context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = srv.WaitForShutdown(ctx)
	assert.Error(err)
	assert.Equal(context.DeadlineExceeded, err)

	// Cleanup
	srv.Shutdown()
}

func TestServerMetrics(t *testing.T) {
	assert := assert.New(t)

	srv, err := StartEmbedded()
	require.NoError(t, err)
	defer srv.Shutdown()

	// Check number of clients
	numClients := srv.NumClients()
	assert.True(numClients >= 1) // At least the in-process connection

	// Create another connection and verify count increases
	conn2, err := srv.NewConnection()
	require.NoError(t, err)
	defer conn2.Close()

	numClients2 := srv.NumClients()
	assert.True(numClients2 > numClients)
}

// Benchmark in-process vs TCP connections
func BenchmarkInProcessConnection(b *testing.B) {
	srv, err := StartEmbedded()
	require.NoError(b, err)
	defer srv.Shutdown()

	nc := srv.Connection()
	subject := "bench.subject"

	// Subscribe
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		// Handler
	})
	require.NoError(b, err)
	defer sub.Unsubscribe()

	message := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := nc.Publish(subject, message)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTCPConnection(b *testing.B) {
	srv, err := StartEmbeddedWithOptions(&EmbeddedOptions{
		InProcessOnly: false,
		Port:         0,
		EnableJetStream: false,
	})
	require.NoError(b, err)
	defer srv.Shutdown()

	nc, err := srv.NewTCPConnection()
	require.NoError(b, err)
	defer nc.Close()

	subject := "bench.subject"

	// Subscribe
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		// Handler
	})
	require.NoError(b, err)
	defer sub.Unsubscribe()

	message := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := nc.Publish(subject, message)
		if err != nil {
			b.Fatal(err)
		}
	}
}