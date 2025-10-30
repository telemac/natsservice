# Type Registry

A thread-safe, feature-rich type registry for Go that enables dynamic type registration, JSON marshaling/unmarshaling with type information, metadata support, and validation hooks.

## Features

- **Dynamic Type Registration**: Register struct types at runtime with custom or auto-generated names
- **Type-Safe JSON Marshaling**: Automatically includes type information in JSON output
- **CloudEvents Compatible**: Uses `type` and `data` fields aligned with CNCF CloudEvents spec
- **TypedData Structure**: First-class support for typed messaging patterns
- **Metadata Support**: Attach arbitrary metadata to registered types
- **Type Aliasing**: Register multiple names for the same type
- **Validation Hooks**: Add custom validation during unmarshaling
- **Batch Operations**: Register multiple types efficiently with a single lock
- **Namespace Discovery**: Find types by namespace prefix
- **Thread-Safe**: All operations are protected with RWMutex

## Installation

```go
import "github.com/telemac/natsservice/pkg/typeregistry"
```

## Basic Usage

### Creating a Registry

```go
registry := typeregistry.New()
```

### Registering Types

```go
type User struct {
    Name string
    Age  int
}

// Register with custom name
err := typeregistry.Register[User](registry, "app.User")

// Register with auto-generated name (package.TypeName)
err := typeregistry.Register[User](registry, "")

// Panic on registration failure
typeregistry.MustRegister[User](registry, "app.User")
```

### Creating Instances

```go
// Create a new instance of a registered type
instance, err := registry.New("app.User")
if err != nil {
    // Handle error
}

user := instance.(*User)
```

### JSON Marshaling/Unmarshaling

```go
// Marshal with type information
user := &User{Name: "John", Age: 30}
data, err := registry.Marshal(user)
// Output: {"type":"app.User","data":{"Name":"John","Age":30}}

// Unmarshal with automatic type detection
result, err := registry.Unmarshal(data)
user = result.(*User)

// Unmarshal specific type
result, err := registry.UnmarshalType("app.User", jsonData)
```

### Working with TypedData

The `TypedData` struct represents the CloudEvents-compatible format for type-safe messaging:

```go
// TypedData structure follows CloudEvents pattern
type TypedData struct {
    Type string          `json:"type"`  // Type identifier
    Data json.RawMessage `json:"data"`  // Payload
}

// Create TypedData directly
typed := typeregistry.NewTypedData("app.User", jsonData)

// Marshal to TypedData structure
user := &User{Name: "John", Age: 30}
typed, err := registry.MarshalTypedData(user)
// typed.Type = "app.User"
// typed.Data = {"Name":"John","Age":30}

// Unmarshal from TypedData
result, err := registry.UnmarshalTypedData(typed)
user = result.(*User)

// TypedData helper methods
typed.MarshalValue(user)        // Marshal value into Data field
typed.UnmarshalValue(&decoded)  // Unmarshal Data field into value
```

## Advanced Features

### Metadata Support

```go
// Register with metadata
metadata := map[string]interface{}{
    "version": "1.0",
    "author":  "john.doe",
    "schema":  "https://api.example.com/schemas/user.json",
}

err := typeregistry.RegisterWithMetadata[User](registry, "app.User", metadata)

// Retrieve metadata
meta, err := registry.GetMetadata("app.User")
version := meta["version"].(string)
```

### Validation Hooks

```go
// Register with validation
validate := func(v any) error {
    user, ok := v.(*User)
    if !ok {
        return errors.New("invalid type")
    }
    if user.Age < 0 || user.Age > 150 {
        return errors.New("invalid age")
    }
    if user.Name == "" {
        return errors.New("name is required")
    }
    return nil
}

err := typeregistry.RegisterWithValidation[User](registry, "app.User", validate)

// Validation is automatically applied during unmarshal
_, err = registry.UnmarshalType("app.User", jsonData)
// Returns error if validation fails
```

### Type Aliasing

```go
// Register a type
typeregistry.Register[User](registry, "app.models.User")

// Add aliases for backward compatibility or convenience
registry.AddAlias("User", "app.models.User")
registry.AddAlias("app.v1.User", "app.models.User")
registry.AddAlias("legacy.User", "app.models.User")

// All aliases work for type creation and unmarshaling
user1, _ := registry.New("User")
user2, _ := registry.New("app.v1.User")
user3, _ := registry.New("legacy.User")
// All create the same type
```

### Batch Registration

```go
entries := []typeregistry.TypeEntry{
    {
        Name: "app.User",
        Type: reflect.TypeOf(&User{}),
        Metadata: map[string]interface{}{
            "version": "1.0",
        },
    },
    {
        Name: "app.Order",
        Type: reflect.TypeOf(&Order{}),
        Metadata: map[string]interface{}{
            "version": "2.0",
        },
        Validate: validateOrder,
    },
}

// Register all types with a single lock acquisition
err := registry.RegisterBatch(entries)
```

### Type Discovery

```go
// Get all registered type names
names := registry.Registered()

// Find types by namespace
userTypes := registry.FindByNamespace("app.models")
// Returns: ["app.models.User", "app.models.Order", ...]

// Get type information
info, err := registry.GetTypeInfo("app.User")
// info.Type: reflect.Type
// info.Metadata: map[string]interface{}
// info.Aliases: []string
// info.Validate: func(any) error
```

### Type Management

```go
// Check if a value is registered
name, err := registry.NameOf(&User{Name: "John"})
if err == nil {
    fmt.Printf("Type is registered as: %s\n", name)
}

// Unregister a type (also removes all aliases)
err := registry.Unregister("app.User")

// Clear all registered types
registry.Clear()
```

## Error Handling

The package defines several error variables for common error conditions:

```go
var (
    ErrTypeNotValid      // Invalid type (must be pointer to struct)
    ErrTypeAlreadyExists // Type with this name already registered
    ErrTypeNotRegistered // Type not found in registry
    ErrMarshal          // JSON marshaling error
    ErrUnmarshal        // JSON unmarshaling error
)
```

## Thread Safety

All registry operations are thread-safe. The registry uses RWMutex for optimal concurrent read performance.

## Use Cases

- **Plugin Systems**: Dynamically load and register types from plugins
- **RPC/Messaging**: Serialize/deserialize messages with type information
- **Configuration Management**: Register configuration structs with validation
- **Data Migration**: Use metadata and aliases to handle schema evolution
- **Service Discovery**: Register service handlers with metadata

## Performance Considerations

- Type lookups are O(1) using hash maps
- Batch registration reduces lock contention
- Read operations use RLock for concurrent access
- Memory optimized by storing only normalized types
- JSON schema caching available via internal sync.Map

## Example: Building a Message System

```go
// Define message types
type CreateUser struct {
    Name  string
    Email string
}

type UpdateUser struct {
    ID    string
    Name  string
    Email string
}

// Create registry and register message types
registry := typeregistry.New()

typeregistry.MustRegister[CreateUser](registry, "msg.CreateUser")
typeregistry.MustRegister[UpdateUser](registry, "msg.UpdateUser")

// Send message
msg := &CreateUser{Name: "John", Email: "john@example.com"}
data, _ := registry.Marshal(msg)
// Send data over network...

// Receive message
received, _ := registry.Unmarshal(data)
switch msg := received.(type) {
case *CreateUser:
    // Handle user creation
case *UpdateUser:
    // Handle user update
}
```