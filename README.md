# telegramsender v2

> **Note**: v1.x is deprecated. Please use v2 for all new projects. See [MIGRATION.md](MIGRATION.md) for upgrade guide.
>
> **New in v2.3**: Simplified configuration API with `New()` and `NewFromConfig()`. See [v3 API](#v3-api-simplified-configuration) below.

**telegramsender** is a production-ready Go library for sending Telegram bot messages with resilience, security, and observability features.

## Features

| Feature | Description |
|---------|-------------|
| **Message Types** | Text messages, photos (URL/file_id), local file uploads |
| **Resilience** | Retry with exponential backoff, circuit breaker, per-chat rate limiting |
| **Security** | TLS 1.2+, path traversal protection, URL whitelist, response size limits |
| **Testability** | `Sender` interface for mocking, typed errors with `errors.As()` support |
| **Observability** | Structured JSON logging via `log/slog` |
| **Configuration** | Environment variables or programmatic with functional options |

## Installation

```bash
go get github.com/prilive-com/telegramsender/v2@latest
```

Requires Go 1.24.3+

## Quick Start

```go
package main

import (
    "context"
    "log"
    "log/slog"
    "time"

    "github.com/prilive-com/telegramsender/v2/telegramsender"
)

func main() {
    // Load config from environment
    cfg, err := telegramsender.LoadConfig()
    if err != nil {
        log.Fatal(err)
    }

    // Create logger (MUST call Close() when done)
    logger, err := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
    if err != nil {
        log.Fatal(err)
    }
    defer logger.Close()

    // Validate config
    if err := telegramsender.ValidateConfig(cfg); err != nil {
        log.Fatal(err)
    }

    // Create API client
    api := telegramsender.NewTelegramAPI(logger, cfg)

    // Send message
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    result, err := api.SendMessage(ctx, telegramsender.MessageRequest{
        ChatID:    123456789,
        Text:      "<b>Hello</b> from telegramsender v2!",
        ParseMode: "HTML",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Message sent, ID: %d", result.MessageID)
}
```

## v3 API (Simplified Configuration)

The new v3 API provides a cleaner interface with support for programmatic options, environment variables, and config files with proper precedence.

### Simple Usage

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/prilive-com/telegramsender/v2/telegramsender"
)

func main() {
    token := os.Getenv("TELEGRAM_BOT_TOKEN")

    // Create client with options
    client, err := telegramsender.New(token,
        telegramsender.WithMaxRetriesOption(5),
        telegramsender.WithRateLimitOption(30, 50),
        telegramsender.ProductionPreset(),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Send message
    ctx := context.Background()
    result, err := client.SendMessage(ctx, telegramsender.MessageRequest{
        ChatID:    123456789,
        Text:      "Hello from v3 API!",
        ParseMode: "HTML",
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Message sent, ID: %d", result.MessageID)
}
```

### From Config File + Env Vars

```go
// Configuration precedence (highest to lowest):
// 1. Programmatic options (opts...)
// 2. Environment variables (TELEGRAM_*)
// 3. Config file
// 4. Default values

client, err := telegramsender.NewFromConfig("config.yaml",
    telegramsender.WithLogger(customLogger),  // Override from config
)
```

### Available Options

```go
// HTTP settings
telegramsender.WithRequestTimeoutOption(10*time.Second)
telegramsender.WithConnectionPool(50, 120*time.Second)

// Rate limiting
telegramsender.WithRateLimitOption(30, 50)  // requests/sec, burst

// Circuit breaker
telegramsender.WithBreakerConfig(5, 2*time.Minute, 60*time.Second)

// Retry settings (exponential backoff)
telegramsender.WithRetryOption(5, 200*time.Millisecond, 30*time.Second, 2.0)
telegramsender.WithMaxRetriesOption(5)

// Content limits
telegramsender.WithContentLimitsOption(1024, 10*1024*1024)

// Security
telegramsender.WithAllowedPhotoDirsOption("/app/uploads", "/tmp")

// Logging
telegramsender.WithLogger(slogLogger)
telegramsender.WithLogFile("logs/bot.log")

// Testing
telegramsender.WithHTTPClientOption(mockClient)

// Presets
telegramsender.ProductionPreset()
telegramsender.DevelopmentPreset()
telegramsender.HighThroughputPreset()
```

---

## Programmatic Configuration (Legacy)

> **Deprecated**: Use `New()` instead. This API will be removed in v4.

```go
cfg := telegramsender.NewConfig(
    "your-bot-token",
    telegramsender.WithMaxRetries(5),
    telegramsender.WithRateLimit(30, 60),
    telegramsender.WithRequestTimeout(15*time.Second),
    telegramsender.WithAllowedPhotoDirs("/app/uploads", "/tmp"),
)
```

## Error Handling

v2 provides typed errors for proper error handling:

```go
result, err := api.SendMessage(ctx, request)
if err != nil {
    var telegramErr *telegramsender.TelegramError
    if errors.As(err, &telegramErr) {
        log.Printf("Telegram error %d: %s", telegramErr.Code, telegramErr.Description)
        if telegramErr.RetryAfter > 0 {
            log.Printf("Retry after: %v", telegramErr.RetryAfter)
        }
    }

    var validationErr *telegramsender.ValidationError
    if errors.As(err, &validationErr) {
        log.Printf("Validation failed on %s: %s", validationErr.Field, validationErr.Message)
    }

    // Sentinel errors
    if errors.Is(err, telegramsender.ErrRateLimitExceeded) {
        // Handle rate limit
    }
}
```

## Testing with Mocks

```go
type MockSender struct {
    Messages []telegramsender.MessageRequest
}

func (m *MockSender) SendMessage(ctx context.Context, req telegramsender.MessageRequest) (*telegramsender.MessageResult, error) {
    m.Messages = append(m.Messages, req)
    return &telegramsender.MessageResult{MessageID: 123}, nil
}

// Use in your service
type NotificationService struct {
    sender telegramsender.Sender // Interface
}
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BOT_TOKEN` | *(required)* | Telegram Bot API token |
| `BASE_URL` | `https://api.telegram.org` | API base URL (whitelist enforced) |
| `LOG_FILE_PATH` | `logs/telegramsender.log` | Log file path |
| `REQUEST_TIMEOUT` | `10s` | HTTP request timeout |
| `MAX_RETRIES` | `3` | Max retry attempts |
| `RATE_LIMIT_REQUESTS` | `10` | Requests per second (global) |
| `RATE_LIMIT_BURST` | `20` | Burst size |
| `MAX_CAPTION_LENGTH` | `1024` | Max caption UTF-8 chars |
| `MAX_FILE_SIZE` | `10485760` | Max file size (10MB) |

See [env.example](env.example) for full list.

## Security Features

- **TLS 1.2+** enforced for all connections
- **Path traversal protection** for file uploads
- **URL whitelist** prevents redirect attacks
- **Response size limits** prevent memory exhaustion
- **Cryptographic jitter** prevents thundering herd

## Documentation

- [CHANGELOG.md](CHANGELOG.md) - Version history
- [MIGRATION.md](MIGRATION.md) - v1 to v2 migration guide
- [INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md) - Integration patterns

## License

MIT © 2025 Prilive Com
