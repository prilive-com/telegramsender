# TelegramSender Library - Code Review (December 2025)

## Executive Summary

**Overall Assessment: 7/10** - Solid foundation with good production practices, but has several issues that need fixing before integration.

The library demonstrates good understanding of production requirements: circuit breakers, rate limiting, connection pooling, structured logging, and retry logic. However, there are **1 critical bug**, **2 high-priority issues**, and several medium/low items to address.

---

## 📊 Review Overview

| Category | Score | Notes |
|----------|-------|-------|
| Code Quality | 7/10 | Clean structure, some duplication |
| Security | 6/10 | Token redaction good, but logger bug and interface issues |
| Error Handling | 6/10 | Good retry logic, but error typing is weak |
| API Design | 6/10 | No interface, hard to mock/test |
| Resilience | 8/10 | Circuit breaker, rate limiting, backoff |
| Documentation | 7/10 | Good README, inline comments adequate |
| Testing | 3/10 | No tests found in archive |

---

## 🚨 CRITICAL BUG: Logger File Handle Leak

**File:** `telegramsender/logger.go:20-31`
**Severity:** CRITICAL - Memory leak in long-running applications

### The Problem

```go
func NewLogger(logLevel slog.Level, logFilePath string) (*slog.Logger, error) {
    // ...
    logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
    if err != nil {
        return nil, err
    }
    // This defer NEVER executes because err is never reassigned after this point
    defer func() {
        if err != nil {  // err is always nil here!
            logFile.Close()
        }
    }()
    // ... rest of function always succeeds
    return logger, nil  // err is nil, so defer doesn't close
}
```

The `err` variable is captured in the closure but never changes after the successful `os.OpenFile`. The defer condition `if err != nil` is **always false**, so the file handle is never managed.

### The Fix

Return the file handle so the caller can close it, or use a struct:

```go
// Option 1: Return file handle for caller to manage
func NewLogger(logLevel slog.Level, logFilePath string) (*slog.Logger, *os.File, error) {
    var logOutput io.Writer = os.Stdout
    var logFile *os.File

    if logFilePath != "" {
        if err := ensureLogPath(logFilePath); err != nil {
            return nil, nil, err
        }

        var err error
        logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
        if err != nil {
            return nil, nil, err
        }
        logOutput = io.MultiWriter(os.Stdout, logFile)
    }

    handler := slog.NewJSONHandler(logOutput, &slog.HandlerOptions{
        Level: logLevel,
    })

    return slog.New(handler), logFile, nil
}

// Option 2: Create a closeable logger struct
type Logger struct {
    *slog.Logger
    file *os.File
}

func (l *Logger) Close() error {
    if l.file != nil {
        return l.file.Close()
    }
    return nil
}
```

---

## ⚠️ HIGH PRIORITY ISSUES

### 1. No Interface for TelegramAPI - Breaks Testability

**File:** `telegramsender/telegram_api.go:28-34`

The `TelegramAPI` is a concrete struct with no interface. This makes mocking impossible for unit tests in your trading bot.

**Current:**
```go
type TelegramAPI struct {
    logger     *slog.Logger
    config     *Config
    httpClient *http.Client
    limiter    *rate.Limiter
    breaker    *gobreaker.CircuitBreaker
}
```

**Add interface:**
```go
// Sender defines the interface for sending Telegram messages
type Sender interface {
    SendMessage(ctx context.Context, request MessageRequest) (*MessageResult, error)
    SendPhoto(ctx context.Context, request PhotoRequest) (*MessageResult, error)
    SendPhotoFile(ctx context.Context, request PhotoFileRequest) (*MessageResult, error)
}

// Ensure TelegramAPI implements Sender
var _ Sender = (*TelegramAPI)(nil)
```

This allows your trading bot to mock notifications in tests:

```go
// In your trading bot tests
type MockSender struct {
    SentMessages []MessageRequest
    Error        error
}

func (m *MockSender) SendMessage(ctx context.Context, req MessageRequest) (*MessageResult, error) {
    m.SentMessages = append(m.SentMessages, req)
    return &MessageResult{MessageID: 123}, m.Error
}
```

---

### 2. TelegramResponse Doesn't Implement Error Interface

**File:** `telegramsender/telegram_api.go:163-164`

The code uses `errors.As(err, &telegramErr)` but `TelegramResponse` doesn't implement the `error` interface, so this **never matches**.

**Current (broken):**
```go
var telegramErr *TelegramResponse
if errors.As(err, &telegramErr) && telegramErr.RetryAfter > 0 {
    // This block NEVER executes because TelegramResponse isn't an error
}
```

**Fix - Make TelegramResponse implement error:**
```go
type TelegramResponse struct {
    OK          bool            `json:"ok"`
    Result      json.RawMessage `json:"result,omitempty"`
    ErrorCode   int             `json:"error_code,omitempty"`
    Description string          `json:"description,omitempty"`
    RetryAfter  time.Duration   `json:"-"`
}

// Implement error interface
func (r *TelegramResponse) Error() string {
    return fmt.Sprintf("telegram API error %d: %s", r.ErrorCode, r.Description)
}

// Update sendMessageOnce to return TelegramResponse as error
func (t *TelegramAPI) sendMessageOnce(ctx context.Context, request MessageRequest) (*MessageResult, error) {
    // ...
    if !telegramResp.OK {
        return nil, telegramResp  // Now this works with errors.As
    }
    // ...
}
```

---

### 3. Jitter Calculation is Deterministic, Not Random

**File:** `telegramsender/telegram_api.go:625-633`

**Current (predictable pattern):**
```go
func (t *TelegramAPI) calculateBackoff(attempt int) time.Duration {
    backoff := t.config.RetryInitialBackoff * time.Duration(math.Pow(t.config.RetryBackoffFactor, float64(attempt-1)))
    if backoff > t.config.RetryMaxBackoff {
        backoff = t.config.RetryMaxBackoff
    }
    // BUG: This alternates between 0.8 and 1.2 based on attempt number
    // Attempt 1: 0.8 + 0.4*1 = 1.2
    // Attempt 2: 0.8 + 0.4*0 = 0.8
    // Attempt 3: 0.8 + 0.4*1 = 1.2
    jitter := time.Duration(float64(backoff) * (0.8 + 0.4*float64(attempt%2)))
    return jitter
}
```

This creates a predictable pattern, not true jitter. If multiple clients hit the same rate limit, they'll all retry at the same time.

**Fix - Use proper random jitter:**
```go
import "math/rand"

func (t *TelegramAPI) calculateBackoff(attempt int) time.Duration {
    backoff := t.config.RetryInitialBackoff * time.Duration(math.Pow(t.config.RetryBackoffFactor, float64(attempt-1)))
    if backoff > t.config.RetryMaxBackoff {
        backoff = t.config.RetryMaxBackoff
    }
    // Add ±20% random jitter
    jitterFactor := 0.8 + rand.Float64()*0.4  // Random between 0.8 and 1.2
    return time.Duration(float64(backoff) * jitterFactor)
}
```

---

## ⚠️ MEDIUM PRIORITY ISSUES

### 4. No Message Text Length Validation

**File:** `telegramsender/telegram_api.go:132`

Telegram limits messages to **4096 UTF-8 characters**. Caption length is validated (line 286-288), but message text is not.

**Add validation:**
```go
const MaxMessageLength = 4096

func (t *TelegramAPI) SendMessage(ctx context.Context, request MessageRequest) (*MessageResult, error) {
    if err := ValidateConfig(t.config); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    // Add message length validation
    textLen := utf8.RuneCountInString(request.Text)
    if textLen > MaxMessageLength {
        return nil, fmt.Errorf("message exceeds %d character limit: %d chars", MaxMessageLength, textLen)
    }
    // ... rest of method
}
```

---

### 5. Rate Limiter is Global, Not Per-Chat

**File:** `telegramsender/telegram_api.go:124`

Telegram rate limits are **per chat** (1 msg/sec per private chat, 20 msg/min per group). Your current implementation uses a global rate limiter.

**Current:**
```go
limiter: rate.NewLimiter(rate.Limit(config.RateLimitRequests), config.RateLimitBurst),
```

**Better approach - per-chat limiters:**
```go
type TelegramAPI struct {
    // ...
    chatLimiters map[int64]*rate.Limiter
    limiterMu    sync.RWMutex
}

func (t *TelegramAPI) getChatLimiter(chatID int64) *rate.Limiter {
    t.limiterMu.RLock()
    limiter, exists := t.chatLimiters[chatID]
    t.limiterMu.RUnlock()
    
    if exists {
        return limiter
    }
    
    t.limiterMu.Lock()
    defer t.limiterMu.Unlock()
    
    // Double-check after acquiring write lock
    if limiter, exists = t.chatLimiters[chatID]; exists {
        return limiter
    }
    
    // Create new limiter for this chat (1 msg/sec with burst of 3)
    limiter = rate.NewLimiter(rate.Every(time.Second), 3)
    t.chatLimiters[chatID] = limiter
    return limiter
}
```

---

### 6. Duplicate Retry Logic (DRY Violation)

**Files:** Lines 142-197, 210-265, 295-350

`SendMessage`, `SendPhoto`, and `SendPhotoFile` have nearly identical retry loops. Extract to a helper:

```go
func (t *TelegramAPI) withRetry(ctx context.Context, operation func() (*MessageResult, error)) (*MessageResult, error) {
    var result *MessageResult
    var err error
    var serverRetryDelay time.Duration

    for attempt := 0; attempt <= t.config.MaxRetries; attempt++ {
        result, err = operation()
        if err == nil {
            return result, nil
        }

        if attempt == t.config.MaxRetries {
            break
        }

        if !t.isRetryable(err) {
            return nil, err
        }

        // Extract retry delay from error if present
        var telegramErr *TelegramResponse
        if errors.As(err, &telegramErr) && telegramErr.RetryAfter > 0 {
            serverRetryDelay = telegramErr.RetryAfter
        } else {
            serverRetryDelay = 0
        }

        backoff := serverRetryDelay
        if backoff == 0 {
            backoff = t.calculateBackoff(attempt + 1)
        }

        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(backoff):
        }
    }

    return nil, fmt.Errorf("max retries exceeded: %w", err)
}

// Usage:
func (t *TelegramAPI) SendMessage(ctx context.Context, request MessageRequest) (*MessageResult, error) {
    if err := ValidateConfig(t.config); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    return t.withRetry(ctx, func() (*MessageResult, error) {
        return t.sendMessageOnce(ctx, request)
    })
}
```

---

### 7. extractTelegramError Uses Fragile String Parsing

**File:** `telegramsender/telegram_api.go:656-686`

```go
func extractTelegramError(err error) *TelegramResponse {
    errMsg := err.Error()
    if strings.Contains(errMsg, "telegram API error") {
        if strings.Contains(errMsg, "403") {
            // ... this is fragile!
        }
    }
}
```

This breaks if error message format changes. Use proper error types instead (see Issue #2 fix).

---

### 8. Go Version 1.24.2 Doesn't Exist

**File:** `go.mod:3`

```go
go 1.24.2
```

Go 1.24 doesn't exist yet. Latest stable is **1.23.x** (December 2025). Should be:
```go
go 1.23
```

---

## 📝 LOW PRIORITY / CODE QUALITY

### 9. No Config Constructor for Programmatic Use

`LoadConfig()` only reads from environment variables. Add a constructor for programmatic configuration:

```go
func NewConfig(botToken string, opts ...ConfigOption) *Config {
    cfg := &Config{
        BotToken:            botToken,
        BaseURL:             "https://api.telegram.org",
        RequestTimeout:      10 * time.Second,
        KeepAlive:           30 * time.Second,
        MaxIdleConns:        10,
        IdleConnTimeout:     90 * time.Second,
        RateLimitRequests:   10,
        RateLimitBurst:      20,
        BreakerMaxRequests:  5,
        BreakerInterval:     2 * time.Minute,
        BreakerTimeout:      60 * time.Second,
        MaxRetries:          3,
        RetryInitialBackoff: 100 * time.Millisecond,
        RetryMaxBackoff:     10 * time.Second,
        RetryBackoffFactor:  2.0,
        MaxCaptionLength:    1024,
        MaxFileSize:         10 * 1024 * 1024,
    }
    
    for _, opt := range opts {
        opt(cfg)
    }
    
    return cfg
}

type ConfigOption func(*Config)

func WithMaxRetries(n int) ConfigOption {
    return func(c *Config) { c.MaxRetries = n }
}

func WithRateLimit(requestsPerSec float64, burst int) ConfigOption {
    return func(c *Config) {
        c.RateLimitRequests = requestsPerSec
        c.RateLimitBurst = burst
    }
}
```

---

### 10. No Unit Tests

No `*_test.go` files found. Add tests for:

- `validateBotToken()` - edge cases
- `calculateBackoff()` - exponential growth, max cap
- `isRetryable()` - all error types
- `ValidateConfig()` - all validation rules
- Mock-based `SendMessage` tests

---

### 11. Limited Media Support

Only supports text and photos. Consider adding:

- `SendDocument()` - for files
- `SendVideo()` - for video
- `EditMessageText()` - for updating messages
- `DeleteMessage()` - for cleanup

---

## ✅ WHAT'S DONE WELL

1. **Circuit Breaker** - Proper use of `sony/gobreaker` with configurable thresholds
2. **Rate Limiting** - Uses `golang.org/x/time/rate` (though should be per-chat)
3. **Connection Pooling** - Configured `http.Transport` with keep-alive
4. **TLS Timeout** - `TLSHandshakeTimeout: 10 * time.Second`
5. **Token Security** - Redacted in logs: `bot[REDACTED]`
6. **Retry-After Handling** - Respects 429 response headers
7. **Structured Logging** - Uses modern `log/slog`
8. **Context Propagation** - All methods accept context
9. **Unicode Handling** - Uses `utf8.RuneCountInString` for caption validation
10. **File Permissions** - Log files created with `0600`
11. **HTTP/2 Support** - `ForceAttemptHTTP2: true`

---

## 🎯 ACTION ITEMS (Priority Order)

### CRITICAL (Fix Before Use)
1. ☐ Fix logger file handle leak
2. ☐ Make `TelegramResponse` implement `error` interface

### HIGH (Fix This Week)
3. ☐ Add `Sender` interface for testability
4. ☐ Fix jitter to use random values

### MEDIUM (Before Production)
5. ☐ Add message text length validation (4096 chars)
6. ☐ Implement per-chat rate limiting
7. ☐ Extract duplicate retry logic to helper
8. ☐ Fix go.mod version (1.24.2 → 1.23)

### LOW (Code Quality)
9. ☐ Add programmatic config constructor
10. ☐ Add unit tests
11. ☐ Consider additional media methods

---

## Integration with Your Trading Bot

Once the above issues are fixed, integration pattern:

```go
// internal/modules/notification/adapter/telegram/sender.go
package telegram

import (
    "context"
    "github.com/prilive-com/telegramsender/telegramsender"
)

type Adapter struct {
    client telegramsender.Sender  // Interface, not concrete type
    chatID int64
}

func NewAdapter(client telegramsender.Sender, chatID int64) *Adapter {
    return &Adapter{client: client, chatID: chatID}
}

func (a *Adapter) SendTradeNotification(ctx context.Context, trade Trade) error {
    msg := formatTradeMessage(trade)
    _, err := a.client.SendMessage(ctx, telegramsender.MessageRequest{
        ChatID:    a.chatID,
        Text:      msg,
        ParseMode: "HTML",
    })
    return err
}
```

---

## Conclusion

The telegramsender library is **production-capable with fixes**. The core architecture is sound - circuit breakers, rate limiting, and retry logic are all properly implemented. The critical issues (logger leak, error interface) must be fixed before integration.

After fixes, this library will integrate well with your trading bot's hexagonal architecture as a driven adapter in the notification module.