# Get User Endpoint

**Subject:** `user_service.get`

**Description:** Retrieves a user from the system by their UUID.

## Request Format

```json
{
  "uuid": "string"  // Required - The UUID v7 of the user to retrieve
}
```

## Sample Command

```bash
nats req user_service.get '{"uuid":"1CUnZkM63_B9TUBZqCfYeyT"}'
```

## Expected Response

```json
{
  "user": {
    "first_name": "Mike",
    "last_name": "Brown",
    "email": "mike.b@domain.com",
    "birth": "0001-01-01T00:00:00Z",
    "active": false,
    "uuid": "1CUnZkM63_B9TUBZqCfYeyT"
  }
}
```

## Notes

- The endpoint uses UUID v7 format for user identification
- UUIDs are generated automatically when users are created via the `add` endpoint
- The complete user object is returned including all stored fields