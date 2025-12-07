# TelegramSender v2 Integration Guide

> **Note**: This is v2.x documentation. For migration from v1, see [MIGRATION.md](MIGRATION.md).

This guide helps developers integrate the telegramsender library into their applications for sending text messages and images via Telegram Bot API.

## Installation

```bash
go get github.com/prilive-com/telegramsender/v2@latest
```

Requires Go 1.24.3+

## Basic Setup

### 1. Environment Configuration

Set these environment variables in your application:

```bash
# Required
BOT_TOKEN=your_bot_token_here

# Optional (with defaults)
BASE_URL=https://api.telegram.org
LOG_FILE_PATH=logs/telegramsender.log
REQUEST_TIMEOUT=10s
RATE_LIMIT_REQUESTS=10
RATE_LIMIT_BURST=20
MAX_RETRIES=3
```

### 2. Initialize the Library

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
    // Load configuration from environment
    cfg, err := telegramsender.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Validate configuration
    if err := telegramsender.ValidateConfig(cfg); err != nil {
        log.Fatalf("Invalid configuration: %v", err)
    }

    // Setup logger - MUST call Close() when done
    logger, err := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    defer logger.Close() // v2 REQUIRED

    // Create API client
    api := telegramsender.NewTelegramAPI(logger, cfg)

    // Now you can use api.SendMessage(), api.SendPhoto(), api.SendPhotoFile()
}
```

### 3. Programmatic Configuration (Alternative)

```go
cfg := telegramsender.NewConfig(
    "your-bot-token",
    telegramsender.WithLogFilePath("logs/app.log"),
    telegramsender.WithMaxRetries(5),
    telegramsender.WithRateLimit(30, 60),
    telegramsender.WithRequestTimeout(15*time.Second),
    telegramsender.WithAllowedPhotoDirs("/app/uploads", "/tmp/photos"),
)
```

## Sending Text Messages

```go
func sendTextMessage(api *telegramsender.TelegramAPI, chatID int64) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    message := telegramsender.MessageRequest{
        ChatID:                chatID,
        Text:                  "<b>Hello</b> from my app!",
        ParseMode:             "HTML",        // or "Markdown", "MarkdownV2"
        DisableWebPagePreview: false,
        DisableNotification:   false,
        ReplyToMessageID:      0,             // 0 = no reply
    }

    result, err := api.SendMessage(ctx, message)
    if err != nil {
        return fmt.Errorf("failed to send message: %w", err)
    }

    log.Printf("Message sent successfully, ID: %d", result.MessageID)
    return nil
}
```

### HTML Formatting Examples

```go
// Bold, italic, links
text := `<b>Bold text</b>
<i>Italic text</i>
<a href="https://example.com">Link</a>
<code>inline code</code>
<pre>code block</pre>`

// Send with HTML parsing
message := telegramsender.MessageRequest{
    ChatID:    chatID,
    Text:      text,
    ParseMode: "HTML",
}
```

## Sending Images

### Method 1: Send Image from URL

```go
func sendImageFromURL(api *telegramsender.TelegramAPI, chatID int64) error {
    ctx := context.Background()

    photo := telegramsender.PhotoRequest{
        ChatID:    chatID,
        Photo:     "https://example.com/image.jpg",
        Caption:   "Check out this image!",
        ParseMode: "HTML",
    }

    result, err := api.SendPhoto(ctx, photo)
    if err != nil {
        return fmt.Errorf("failed to send photo: %w", err)
    }

    log.Printf("Photo sent successfully, ID: %d", result.MessageID)
    return nil
}
```

### Method 2: Reuse Previously Sent Image

```go
// When Telegram sends you an image, it includes a file_id
// You can reuse this file_id for faster sending
func sendImageByFileID(api *telegramsender.TelegramAPI, chatID int64, fileID string) error {
    photo := telegramsender.PhotoRequest{
        ChatID:  chatID,
        Photo:   fileID, // e.g., "AgACAgIAAxkBAAIC..."
        Caption: "Resending this image",
    }

    _, err := api.SendPhoto(context.Background(), photo)
    return err
}
```

### Method 3: Send Local File (v2)

```go
func sendLocalImage(api *telegramsender.TelegramAPI, chatID int64) error {
    photo := telegramsender.PhotoFileRequest{
        ChatID:    chatID,
        PhotoPath: "/app/uploads/image.jpg", // Must be in AllowedPhotoDirs
        Caption:   "Local file upload",
        ParseMode: "HTML",
    }

    _, err := api.SendPhotoFile(context.Background(), photo)
    return err
}
```

**Important**: Configure `AllowedPhotoDirs` to restrict which directories can be used:
```go
cfg := telegramsender.NewConfig(token,
    telegramsender.WithAllowedPhotoDirs("/app/uploads", "/tmp/photos"),
)
```

## Error Handling (v2)

v2 provides typed errors for proper error handling:

```go
func robustSend(api *telegramsender.TelegramAPI, chatID int64) {
    message := telegramsender.MessageRequest{
        ChatID: chatID,
        Text:   "Important message",
    }

    result, err := api.SendMessage(context.Background(), message)
    if err != nil {
        // Check for Telegram API errors
        var telegramErr *telegramsender.TelegramError
        if errors.As(err, &telegramErr) {
            log.Printf("Telegram error %d: %s", telegramErr.Code, telegramErr.Description)

            if telegramErr.RetryAfter > 0 {
                log.Printf("Retry after: %v", telegramErr.RetryAfter)
            }

            switch telegramErr.Code {
            case 403:
                log.Println("Bot was blocked by user or removed from group")
            case 400:
                log.Println("Bad request - check your parameters")
            case 401:
                log.Println("Invalid bot token")
            }
            return
        }

        // Check for validation errors
        var validationErr *telegramsender.ValidationError
        if errors.As(err, &validationErr) {
            log.Printf("Validation failed on %s: %s", validationErr.Field, validationErr.Message)
            return
        }

        // Check sentinel errors
        if errors.Is(err, telegramsender.ErrRateLimitExceeded) {
            log.Println("Rate limit exceeded")
        } else if errors.Is(err, telegramsender.ErrCircuitBreakerOpen) {
            log.Println("Circuit breaker is open, wait before retrying")
        } else if errors.Is(err, telegramsender.ErrMaxRetriesExceeded) {
            log.Println("All retries exhausted")
        } else {
            log.Printf("Unexpected error: %v", err)
        }
        return
    }

    log.Printf("Success! Message ID: %d", result.MessageID)
}
```

## Testing with Mocks (v2)

v2 provides a `Sender` interface for easy mocking:

```go
// Your service depends on the interface
type NotificationService struct {
    sender telegramsender.Sender // Interface, not *TelegramAPI
}

func NewNotificationService(sender telegramsender.Sender) *NotificationService {
    return &NotificationService{sender: sender}
}

func (ns *NotificationService) NotifyUser(ctx context.Context, userID int64, message string) error {
    _, err := ns.sender.SendMessage(ctx, telegramsender.MessageRequest{
        ChatID: userID,
        Text:   message,
    })
    return err
}

// In tests, create a mock
type MockSender struct {
    Messages []telegramsender.MessageRequest
    Err      error
}

func (m *MockSender) SendMessage(ctx context.Context, req telegramsender.MessageRequest) (*telegramsender.MessageResult, error) {
    if m.Err != nil {
        return nil, m.Err
    }
    m.Messages = append(m.Messages, req)
    return &telegramsender.MessageResult{MessageID: 123}, nil
}

func (m *MockSender) SendPhoto(ctx context.Context, req telegramsender.PhotoRequest) (*telegramsender.MessageResult, error) {
    return &telegramsender.MessageResult{MessageID: 124}, nil
}

func (m *MockSender) SendPhotoFile(ctx context.Context, req telegramsender.PhotoFileRequest) (*telegramsender.MessageResult, error) {
    return &telegramsender.MessageResult{MessageID: 125}, nil
}

// Test example
func TestNotifyUser(t *testing.T) {
    mock := &MockSender{}
    service := NewNotificationService(mock)

    err := service.NotifyUser(context.Background(), 123456789, "Hello!")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if len(mock.Messages) != 1 {
        t.Fatalf("expected 1 message, got %d", len(mock.Messages))
    }

    if mock.Messages[0].Text != "Hello!" {
        t.Errorf("unexpected message text: %s", mock.Messages[0].Text)
    }
}
```

## Advanced Usage

### 1. Context with Timeout

```go
func sendWithTimeout(api *telegramsender.TelegramAPI, chatID int64) error {
    // Create context with 30-second timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    message := telegramsender.MessageRequest{
        ChatID: chatID,
        Text:   "This message has a timeout",
    }

    _, err := api.SendMessage(ctx, message)
    return err
}
```

### 2. Reply to Specific Message

```go
func replyToMessage(api *telegramsender.TelegramAPI, chatID int64, replyToID int) error {
    message := telegramsender.MessageRequest{
        ChatID:           chatID,
        Text:             "This is a reply",
        ReplyToMessageID: replyToID,
    }

    _, err := api.SendMessage(context.Background(), message)
    return err
}
```

### 3. Send with Custom Keyboard

```go
func sendWithKeyboard(api *telegramsender.TelegramAPI, chatID int64) error {
    // Inline keyboard
    keyboard := map[string]interface{}{
        "inline_keyboard": [][]map[string]string{
            {
                {"text": "Button 1", "callback_data": "btn1"},
                {"text": "Button 2", "callback_data": "btn2"},
            },
            {
                {"text": "URL Button", "url": "https://example.com"},
            },
        },
    }

    message := telegramsender.MessageRequest{
        ChatID:      chatID,
        Text:        "Choose an option:",
        ReplyMarkup: keyboard,
    }

    _, err := api.SendMessage(context.Background(), message)
    return err
}
```

## Best Practices

### 1. Reuse the API Client

```go
// DON'T create new client for each message
for _, msg := range messages {
    api := telegramsender.NewTelegramAPI(logger, cfg) // Bad
    api.SendMessage(ctx, msg)
}

// DO reuse the same client
api := telegramsender.NewTelegramAPI(logger, cfg) // Good
for _, msg := range messages {
    api.SendMessage(ctx, msg)
}
```

### 2. Always Close the Logger

```go
logger, err := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
if err != nil {
    log.Fatal(err)
}
defer logger.Close() // ALWAYS close!
```

### 3. Handle Rate Limits

The library handles rate limiting automatically with per-chat limiters, but you can help by:

```go
// Batch messages with small delays for bulk sending
messages := []string{"msg1", "msg2", "msg3"}
for i, text := range messages {
    msg := telegramsender.MessageRequest{
        ChatID: chatID,
        Text:   text,
    }

    api.SendMessage(ctx, msg)

    // Add small delay between messages (optional)
    if i < len(messages)-1 {
        time.Sleep(100 * time.Millisecond)
    }
}
```

### 4. Content Limits

| Content | Limit |
|---------|-------|
| Message text | 4096 UTF-8 characters |
| Caption | 1024 UTF-8 characters |
| Photo file | 10MB |
| Photo dimensions | 1280px recommended |

```go
// The library validates automatically, but you can pre-check:
caption := "Your long caption here..."
captionLen := utf8.RuneCountInString(caption)
if captionLen > 1024 {
    caption = string([]rune(caption)[:1021]) + "..."
}
```

## Production Deployment

### 1. Environment Variables

```bash
# Production configuration
BOT_TOKEN=your_production_token
LOG_FILE_PATH=/var/log/myapp/telegram.log
RATE_LIMIT_REQUESTS=30  # Increase for production
RATE_LIMIT_BURST=60
MAX_RETRIES=5
RETRY_MAX_BACKOFF=30s
```

### 2. Graceful Shutdown

```go
func main() {
    cfg, _ := telegramsender.LoadConfig()
    logger, _ := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
    api := telegramsender.NewTelegramAPI(logger, cfg)

    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        log.Println("Shutting down gracefully...")
        logger.Close() // Clean up logger
        os.Exit(0)
    }()

    // Your application logic here
}
```

### 3. Monitoring

Monitor these metrics in production:
- Message send success/failure rates
- Circuit breaker state changes
- Rate limit hits
- Response times

The library logs all important events in JSON format for easy parsing.

## Common Integration Patterns

### 1. Notification Service

```go
type NotificationService struct {
    sender telegramsender.Sender
    logger *telegramsender.Logger
}

func (ns *NotificationService) NotifyUser(ctx context.Context, userID int64, message string) error {
    _, err := ns.sender.SendMessage(ctx, telegramsender.MessageRequest{
        ChatID: userID,
        Text:   message,
    })
    return err
}

func (ns *NotificationService) NotifyWithImage(ctx context.Context, userID int64, message, imageURL string) error {
    _, err := ns.sender.SendPhoto(ctx, telegramsender.PhotoRequest{
        ChatID:  userID,
        Photo:   imageURL,
        Caption: message,
    })
    return err
}
```

### 2. Alert System

```go
func sendAlert(sender telegramsender.Sender, alertChannel int64, severity, message string) error {
    emoji := map[string]string{
        "critical": "🔴",
        "warning":  "🟡",
        "info":     "🔵",
    }[severity]

    text := fmt.Sprintf("%s <b>%s Alert</b>\n\n%s\n\n<i>Time: %s</i>",
        emoji, strings.ToUpper(severity), message, time.Now().Format(time.RFC3339))

    _, err := sender.SendMessage(context.Background(), telegramsender.MessageRequest{
        ChatID:    alertChannel,
        Text:      text,
        ParseMode: "HTML",
    })
    return err
}
```

## Troubleshooting

### Common Issues

1. **"BOT_TOKEN must be set"** - Ensure environment variable is set
2. **"403 Forbidden"** - Bot was blocked or not in the chat
3. **"400 Bad Request"** - Invalid parameters (check chat_id, text length)
4. **Circuit breaker open** - Too many failures, will auto-recover
5. **Rate limit exceeded** - Automatic retry with backoff
6. **"path not in allowed directories"** - Configure `AllowedPhotoDirs` for file uploads

### Debug Mode

Enable debug logging by setting log level:

```go
logger, _ := telegramsender.NewLogger(slog.LevelDebug, cfg.LogFilePath)
```

## Security Features (v2)

- **TLS 1.2+** enforced for all connections
- **Path traversal protection** with configurable allowed directories
- **URL whitelist** prevents redirect attacks
- **Response size limits** prevent memory exhaustion
- **Cryptographic jitter** prevents thundering herd

## Support

For issues or questions:
- Check the logs first (JSON format, easy to parse)
- Review error messages - they're descriptive with typed errors
- Ensure your bot has proper permissions in groups
- Verify network connectivity to api.telegram.org
- See [MIGRATION.md](MIGRATION.md) for v1 to v2 upgrade guide

Remember: The library handles retries, rate limiting, and circuit breaking automatically. Focus on your business logic!
