# Add User Endpoint

**Subject:** `user_service.add`

**Description:** Adds a new user to the system with validation for required fields.

## Request Format

```json
{
  "user": {
    "first_name": "string",     // Required
    "last_name": "string",      // Required
    "email": "string",          // Required
    "birth": "YYYY-MM-DDTHH:MM:SSZ", // Optional
    "active": true              // Optional
  }
}
```

## Sample Command

```bash
nats req user_service.add '{"user":{"first_name":"Mike","last_name":"Brown","email":"mike.b@domain.com","active":false}}'
```

## Expected Response

```json
{
  "uuid": "1CUnZkM63_B9TUBZqCfYeyT"
}
```

## Notes

- The service automatically generates a UUID v7 for each new user
- Required fields: `first_name`, `last_name`, `email`
- Optional fields: `birth` (ISO 8601 format), `active` (boolean)