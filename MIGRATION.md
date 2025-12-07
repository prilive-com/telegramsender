# Migration Guide: v1 to v2

This guide helps you migrate from telegramsender v1.x to v2.0.0.

## Import Path Change

```go
// Before (v1)
import "github.com/prilive-com/telegramsender/telegramsender"

// After (v2)
import "github.com/prilive-com/telegramsender/v2/telegramsender"
```

Update your `go.mod`:
```bash
go get github.com/prilive-com/telegramsender/v2@latest
```

## Logger Changes (Breaking)

The most significant breaking change is the logger type.

### Before (v1)
```go
logger, err := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
if err != nil {
    log.Fatal(err)
}
// logger is *slog.Logger - no cleanup needed (but file handle leaked!)
```

### After (v2)
```go
logger, err := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
if err != nil {
    log.Fatal(err)
}
defer logger.Close() // REQUIRED: properly closes file handle

// logger embeds *slog.Logger, so all slog methods still work:
logger.Info("message", "key", "value")
```

## New Validation Errors

v2 validates requests before sending. Your code may now receive errors for:

- Empty `ChatID` (was `0`)
- Empty `Text` in `MessageRequest`
- Text exceeding 4096 characters
- Invalid `ParseMode` (must be "HTML", "Markdown", or "MarkdownV2")

```go
result, err := api.SendMessage(ctx, request)
if err != nil {
    var validationErr *telegramsender.ValidationError
    if errors.As(err, &validationErr) {
        // Handle validation error
        log.Printf("Validation failed on %s: %s", validationErr.Field, validationErr.Message)
    }
}
```

## Typed Telegram Errors

v2 returns typed `*TelegramError` instead of formatted strings:

### Before (v1)
```go
if strings.Contains(err.Error(), "429") {
    // Rate limited - but RetryAfter was never accessible!
}
```

### After (v2)
```go
var telegramErr *telegramsender.TelegramError
if errors.As(err, &telegramErr) {
    if telegramErr.Code == 429 {
        log.Printf("Rate limited, retry after: %v", telegramErr.RetryAfter)
    }
    if telegramErr.IsRetryable() {
        // Automatic retries already happened, this is permanent failure
    }
}
```

## Configuration Options

v2 adds a programmatic configuration constructor:

### Before (v1) - Environment only
```go
cfg, err := telegramsender.LoadConfig() // Only from env vars
```

### After (v2) - Programmatic option
```go
// Option 1: Still works
cfg, err := telegramsender.LoadConfig()

// Option 2: Programmatic configuration
cfg := telegramsender.NewConfig(
    "your-bot-token",
    telegramsender.WithMaxRetries(5),
    telegramsender.WithRateLimit(30, 60),
    telegramsender.WithAllowedPhotoDirs("/app/uploads", "/tmp"),
)
```

## SendPhotoFile Path Restrictions

v2 validates photo paths for security. If you use `SendPhotoFile`:

```go
// Configure allowed directories
cfg := telegramsender.NewConfig(token,
    telegramsender.WithAllowedPhotoDirs("/app/uploads", "/tmp/photos"),
)

// Or via environment (if using LoadConfig):
// No env var for this - use NewConfig for path restrictions

// Paths must be:
// 1. Absolute (not relative)
// 2. Not contain ".."
// 3. Within allowed directories (if configured)
// 4. Not be symlinks
```

## Interface for Testing

v2 provides a `Sender` interface for mocking:

```go
// Your code can now depend on the interface
type NotificationService struct {
    sender telegramsender.Sender // Interface, not concrete type
}

// In tests, create a mock
type MockSender struct {
    SendMessageFunc func(ctx context.Context, req telegramsender.MessageRequest) (*telegramsender.MessageResult, error)
}

func (m *MockSender) SendMessage(ctx context.Context, req telegramsender.MessageRequest) (*telegramsender.MessageResult, error) {
    return m.SendMessageFunc(ctx, req)
}
// ... implement other methods
```

## Sentinel Errors

v2 provides sentinel errors for error checking:

```go
if errors.Is(err, telegramsender.ErrRateLimitExceeded) {
    // Handle rate limit
}

if errors.Is(err, telegramsender.ErrCircuitBreakerOpen) {
    // Circuit breaker is open, wait before retrying
}

if errors.Is(err, telegramsender.ErrMaxRetriesExceeded) {
    // All retries exhausted
}

if errors.Is(err, telegramsender.ErrPathTraversal) {
    // Invalid photo path
}
```

## Go Version Requirement

v2 requires Go 1.24.3 or later. Update your `go.mod`:

```go
go 1.24.3
```

## Quick Migration Checklist

- [ ] Update import path to `v2`
- [ ] Add `defer logger.Close()` after creating logger
- [ ] Handle new `ValidationError` type if needed
- [ ] Use `errors.As()` for `TelegramError` instead of string matching
- [ ] Configure `AllowedPhotoDirs` if using `SendPhotoFile`
- [ ] Update Go version to 1.24.3+
- [ ] Run tests to verify migration

## Need Help?

If you encounter issues during migration, check:
1. [CHANGELOG.md](CHANGELOG.md) for detailed list of changes
2. [README.md](README.md) for updated usage examples
3. [INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md) for integration patterns
