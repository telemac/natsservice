# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go NATS microservice framework that provides a simple abstraction layer on top of NATS microservices. The project consists of:

- `natsservice` package: Core service and endpoint abstractions
- Example service implementation demonstrating the framework usage

## Architecture

The framework is built around two main interfaces:

- **Servicer**: Manages the overall service lifecycle (start/stop/config)
- **Endpointer**: Handles individual endpoint logic and configuration

Key components:
- `Service`: Main service implementation that manages NATS microservice
- `Endpoint`: Base endpoint type that implements the Endpointer interface
- `ServiceConfig`: Service-level configuration (name, version, NATS connection, logger, etc.)
- `EndpointConfig`: Endpoint-level configuration (name, metadata, queue group, subject)

## Common Commands

### Build and Run
```bash
# Run the example service
go run examples/demo_service/main.go

# Build the module
go build ./...
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for specific package
go test ./natsservice
```

### Module Management
```bash
# Tidy dependencies
go mod tidy

# Download dependencies
go mod download
```

## Development Notes

- Services require a context, NATS connection, logger, and name at minimum
- Endpoints can be grouped using the Group field in ServiceConfig
- Queue groups can be configured per endpoint or disabled
- The framework uses Go 1.24.4 and depends on nats.io and telemac/goutils
- All service configurations are validated before startup