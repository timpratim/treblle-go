# Treblle Go SDK

The **Treblle Go SDK** provides real-time API monitoring, observability, and analytics for your Go applications. It captures request and response data, helping you debug, optimize, and maintain high-quality APIs effortlessly.

## Features 🚀

- API Monitoring & Observability 📡
- Automatic API Documentation 📖
- Performance & Security Analysis 🔍
- Real-time API Logging & Debugging 🛠️
- Asynchronous Processing for Performance ⚡
- CLI Tool for Debugging 🖥️
- **Customer & Trace ID Tracking** 🏷️
- **Sensitive Data Masking** 🔒

## Installation 📥

```bash
go get github.com/timpratim/treblle-go
```

## Quick Start 🚀

### 1️⃣ Configure Treblle

Import the package and configure it in your `main()` function:

```go
import "github.com/timpratim/treblle-go/treblle"

func main() {
    treblle.Configure(treblle.Configuration{
        APIKey:     "YOUR_API_KEY",
        ProjectID:  "YOUR_PROJECT_ID",
        MaskingEnabled: true, // Enable field masking
        AdditionalFieldsToMask: []string{"password", "credit_card", "ssn"}, // Custom fields to mask
    })
    
    // Start your application
}
```

### 2️⃣ Add Middleware

Integrate Treblle's middleware with your router.

#### Using `net/http`

```go
mux := http.NewServeMux()
mux.Handle("/", treblle.Middleware(http.HandlerFunc(myHandler)))
```

#### Using `chi`

```go
r := chi.NewRouter()
r.Use(treblle.Middleware)
```

## Tracking Customer & Trace IDs 🏷️

Treblle automatically extracts **customer and trace IDs** from request headers:

- `treblle-user-id`: Identifies the customer making the request.
- `treblle-tag-id`: Provides a unique traceable ID for debugging distributed systems.

Ensure your system includes these headers in requests:

```go
req.Header.Set("treblle-user-id", "12345")
req.Header.Set("treblle-tag-id", "abc-xyz-789")
```

Treblle will log and send this data for better observability.

## Masking Sensitive Data 🔒

Treblle provides **built-in masking** for sensitive fields before sending data. By default, the SDK masks fields like:

- `password`
- `card_number`
- `ssn`
- `api_key`

You can **add additional fields to mask**:

```go
treblle.Configure(treblle.Configuration{
    APIKey: "YOUR_API_KEY",
    ProjectID: "YOUR_PROJECT_ID",
    MaskingEnabled: true,
    AdditionalFieldsToMask: []string{"email", "token", "security_answer"},
})
```

Treblle ensures this data is **never sent** in logs or analytics.

## CLI Debugging 🖥️

The SDK includes a CLI tool for debugging:

```bash
go run github.com/timpratim/treblle-go/cmd/treblle-go -debug
```

This outputs:

- SDK Version
- API Key & Project ID (Masked for security)
- Configured Treblle URL

## Advanced Configuration ⚙️

```go
treblle.Configure(treblle.Configuration{
    APIKey:                  "YOUR_API_KEY",
    ProjectID:               "YOUR_PROJECT_ID",
    MaskingEnabled:          true,
    AsyncProcessingEnabled:  true, // Process requests asynchronously
    MaxConcurrentProcessing: 10,   // Limit async processing concurrency
    IgnoredEnvironments:     []string{"local", "testing"},
})
```

## Graceful Shutdown 🛑

Ensure all logs are sent before your application exits:

```go
treblle.GracefulShutdown()
```

## Error Handling 🛠️

Batch error collection is supported:

```go
treblle.Configure(treblle.Configuration{
    BatchErrorEnabled:   true,
    BatchErrorSize:      50,
    BatchFlushInterval:  time.Second * 5,
})
```

## Upgrading from Previous Versions 🔄

- Ensure **Go 1.21+** is installed.
- Update package imports from `github.com/treblle/treblle-go` to `github.com/timpratim/treblle-go`.
- Use `treblle.Configure()` to set up new async processing and error handling options.

## Contributing 🤝

Contributions are welcome! Open an issue or submit a PR on [GitHub](https://github.com/timpratim/treblle-go).

## License 📜

MIT License. See `LICENSE` for details.