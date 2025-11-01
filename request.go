package natsservice

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/telemac/natsservice/pkg/typeregistry"
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
	// Validate connection
	if nc == nil {
		return nil, fmt.Errorf("NATS connection is nil")
	}
	if !nc.IsConnected() {
		return nil, fmt.Errorf("NATS connection is not active")
	}

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
	// Validate connection
	if nc == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	if !nc.IsConnected() {
		return fmt.Errorf("NATS connection is not active")
	}

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

// TypedRequest makes a typed request to a NATS microservice endpoint
// The request type is looked up in the registry and included as a header.
// The response type is determined from the response header and unmarshaled accordingly.
// ctx the request context
// nc: NATS connection
// tr: type registry for looking up types
// subject: the subject to send the request to
// request: the request payload (must be registered in the type registry)
//
// Returns:
//   response: the response unmarshaled to the type specified in the response header
//   error: any error that occurred
func TypedRequest(ctx context.Context, nc *nats.Conn, tr *typeregistry.Registry, subject string, request any) (any, error) {
	if nc == nil {
		return nil, fmt.Errorf("NATS connection is nil")
	}
	if tr == nil {
		return nil, fmt.Errorf("type registry is nil")
	}

	if !nc.IsConnected() {
		return nil, fmt.Errorf("NATS connection is not active")
	}

	// Find the request type in the registry
	requestTypeName, err := tr.NameOf(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get request type name: %w", err)
	}

	// Marshal the request payload
	reqData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create a NATS message with the type header
	msg := &nats.Msg{
		Subject: subject,
		Data:    reqData,
		Header:  nats.Header{},
	}
	msg.Header.Set("X-Type", requestTypeName)

	// Send request and wait for response (with a default timeout)
	respMsg, err := nc.RequestMsgWithContext(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Get the type header from the response
	responseTypeName := respMsg.Header.Get("X-Type")
	if responseTypeName == "" {
		return nil, fmt.Errorf("response missing X-Type header")
	}

	// Unmarshal the response payload to the type specified in the response header
	responseValue, err := tr.UnmarshalType(responseTypeName, respMsg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal typed response: %w", err)
	}

	return responseValue, nil
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
	// Validate connection
	if nc == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	if !nc.IsConnected() {
		return fmt.Errorf("NATS connection is not active")
	}

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
