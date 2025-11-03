# NATS Service Client Example

This directory contains an example of how to use the `natsservice` package to communicate with NATS microservices using the synchronous request-response pattern.

## Overview

This example demonstrates how to use the `Request[T]` function to call the `add` endpoint of the demo service. It shows type-safe communication using the shared request/response types from the service's endpoint package.

## Prerequisites

Make sure you have a NATS server running:

```bash
# Start NATS server with Docker
docker run -d --name nats -p 4222:4222 nats:latest

# Or install and run locally
nats-server
```

## Running the Example

1. Start a NATS server (see Prerequisites above)
2. Start the demo service from the `examples/demo_service` directory:
   ```bash
   cd ../demo_service
   go run main.go
   ```
3. Run the client example:
   ```bash
   go run main.go
   ```

## Example Code

The example demonstrates calling the `add` endpoint:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/nats-io/nats.go"
    "github.com/telemac/natsservice"
    "github.com/telemac/natsservice/examples/demo_service/endpoints/add"
)

func main() {
    // Connect to NATS
    nc, err := nats.Connect(nats.DefaultURL)
    if err != nil {
        log.Fatal("Failed to connect to NATS:", err)
    }
    defer nc.Close()

    ctx := context.Background()

    // Call the add endpoint
    result, err := CallAddEndpoint(ctx, nc, 5.5, 3.2)
    if err != nil {
        log.Printf("Error calling add endpoint: %v", err)
    } else {
        fmt.Printf("Result: %.2f + %.2f = %.2f\n", 5.5, 3.2, result)
    }
}

func CallAddEndpoint(ctx context.Context, nc *nats.Conn, a, b float64) (float64, error) {
    // Create the request using the add package types
    req := add.AddRequest{
        A: a,
        B: b,
    }

    // Call the generic Request function
    // Subject: "demo.add" (service group "demo" + endpoint name "add")
    response, err := natsservice.Request[add.AddRequest, add.AddResponse](
        ctx,
        nc,
        "demo.add",
        req,
    )

    if err != nil {
        return 0, fmt.Errorf("failed to call add endpoint: %w", err)
    }

    return response.Result, nil
}
```

## Key Points

- **Type Safety**: Uses the actual request/response types from the service's endpoint package
- **Subject Format**: The subject follows the pattern `{group}.{endpoint_name}` (e.g., "demo.add")
- **Error Handling**: Wraps errors with context for better debugging
- **Shared Types**: Import types directly from the service implementation for consistency