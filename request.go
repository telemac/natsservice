package natsservice

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

// Request makes a generic request to a NATS microservice endpoint
// ctx: context for the request
// nc: NATS connection
// subject: the subject to send the request to
// request: the request payload (any type that can be marshaled to JSON)
//
// Returns:
//   response: the response unmarshaled into the provided type
//   error: any error that occurred
func Request[TRequest, TResponse any](
	ctx context.Context,
	nc *nats.Conn,
	subject string,
	request TRequest,
) (*TResponse, error) {

	// Marshal the request
	reqData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request and wait for response
	msg, err := nc.RequestWithContext(ctx, subject, reqData)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Unmarshal response
	var response TResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// RequestAsync makes an asynchronous request to a NATS microservice endpoint
// nc: NATS connection
// subject: the subject to send the request to
// request: the request payload (any type that can be marshaled to JSON)
// handler: function to handle the response
//
// Returns:
//   error: any error that occurred while sending the request
func RequestAsync[TRequest any](
	nc *nats.Conn,
	subject string,
	request TRequest,
	handler func(*nats.Msg),
) error {
	// Marshal the request
	reqData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create inbox for response
	inbox := nats.NewInbox()

	// Subscribe to inbox for the response
	sub, err := nc.Subscribe(inbox, handler)
	if err != nil {
		return fmt.Errorf("failed to subscribe for response: %w", err)
	}

	// Auto-unsubscribe after one message
	sub.AutoUnsubscribe(1)

	// Publish request with reply subject
	err = nc.PublishRequest(subject, inbox, reqData)
	if err != nil {
		return fmt.Errorf("failed to publish request: %w", err)
	}

	return nil
}

// Publish publishes a message to a NATS subject without expecting a response
// nc: NATS connection
// subject: the subject to publish to
// request: the request payload (any type that can be marshaled to JSON)
//
// Returns:
//   error: any error that occurred while publishing
func Publish[TRequest any](
	nc *nats.Conn,
	subject string,
	request TRequest,
) error {
	// Marshal the request
	reqData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Publish message
	err = nc.Publish(subject, reqData)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}
