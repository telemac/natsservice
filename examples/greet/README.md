# Greeting Service Example

A simple NATS microservice that responds with personalized greetings.

## Run the Service

```bash
go run main.go
```

The service will start and register the `demo.greet` endpoint.

## Test the Service

### Using nats CLI

```bash
# Send a request
nats req demo.greet "Alexandre"

# Expected response:
# Hello, Alexandre !
```

## How it Works

1. Service connects to NATS at `nats://localhost:4222`
2. Registers endpoint `greet` under service group `demo`
3. Full subject becomes: `demo.greet`
4. Responds with "Hello, {name} !"

## Service Info

```bash
# List all services
nats service list

# Show service details
nats service info demo
```