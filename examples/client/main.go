package main

import (
	"context"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/telemac/natsservice"
	"github.com/telemac/natsservice/examples/service/endpoints/add"
)

func main() {
	// Connect to NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal("Failed to connect to NATS:", err)
	}
	defer nc.Close()

	ctx := context.Background()

	// Example 1: Call the add endpoint
	fmt.Println("Calling add endpoint...")
	result, err := CallAddEndpoint(ctx, nc, 5.5, 3.2)
	if err != nil {
		log.Printf("Error calling add endpoint: %v", err)
	} else {
		fmt.Printf("Result: %.2f + %.2f = %.2f\n", 5.5, 3.2, result)
	}

}

// CallAddEndpoint demonstrates how to call the add endpoint using the generic Request function
func CallAddEndpoint(ctx context.Context, nc *nats.Conn, a, b float64) (float64, error) {
	// Create the request using the add package types
	req := add.AddRequest{
		A: a,
		B: b,
	}

	// Call the generic Request function
	// Note: The subject should match the endpoint's subject configuration
	// The service uses Group: "demo" and the endpoint name is "add", resulting in "demo.add"
	response, err := natsservice.Request[add.AddRequest, add.AddResponse](
		ctx,
		nc,
		"demo.add", // Subject for the add endpoint
		req,
	)

	if err != nil {
		return 0, fmt.Errorf("failed to call add endpoint: %w", err)
	}

	return response.Result, nil
}
