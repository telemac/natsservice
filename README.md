# NATS Service

Lightweight Go NATS microservice framework.

## 1. Start a Service

```go
nc, _ := nats.Connect(nats.DefaultURL)
defer nc.Close()

svc, err := natsservice.StartService(&natsservice.ServiceConfig{
    Ctx:    context.Background(),
    Nc:     nc,
    Logger: slog.Default(),
    Name:   "my-service",
})
if err != nil { panic(err) }
defer svc.Stop()
```

## 2. Write an Endpoint

```go
type GreetingEndpoint struct {
    natsservice.Endpoint
}

func (e *GreetingEndpoint) Config() *natsservice.EndpointConfig {
    return &natsservice.EndpointConfig{
        Name:       "greet",
        QueueGroup: "workers",
    }
}

func (e *GreetingEndpoint) Handle(req micro.Request) {
    name := string(req.Data())
    message := fmt.Sprintf("Hello, %s !", name)
    req.Respond([]byte(message))
}
```

## 3. Add Endpoint to Service

```go
err = svc.AddEndpoint(&GreetingEndpoint{})
if err != nil { panic(err) }
```

## Examples

### Basic Greeting Service
See [examples/greet](./examples/greet) - A minimal service that responds with personalized greetings.

```bash
cd examples/greet
go run main.go
nats req demo.greet "Alexandre"
# Response: Hello, Alexandre !
```

### Multi-Endpoint Service
See [examples/service](./examples/service) - Demonstrates multiple endpoints sharing a common counter.



