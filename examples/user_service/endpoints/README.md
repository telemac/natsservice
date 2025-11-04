# User Service

A NATS microservice for user management with CRUD operations.

## Service Configuration

- **Name**: `user_service`
- **Version**: `1.0.0`
- **Description**: User handling service

## Endpoints

### Add User
- **Documentation**: [ADD.md](./ADD.md)
- **Subject**: `user_service.add`
- **Purpose**: Creates new users with automatic UUID v7 generation

### Get User
- **Documentation**: [GET.md](./GET.md)
- **Subject**: `user_service.get`
- **Purpose**: Retrieves users by UUID v7

## Usage Pattern

1. **Create users** using the `add` endpoint
2. **Receive UUID v7** in response for each created user
3. **Retrieve users** using the `get` endpoint with the UUID

## Data Model

All users contain the following fields:
- `first_name` (required)
- `last_name` (required)
- `email` (required)
- `birth` (optional, ISO 8601 format)
- `active` (optional, boolean)
- `uuid` (auto-generated UUID v7)

## Storage

The service uses JetStream KV store for persistent user data with UUID-based key lookups.