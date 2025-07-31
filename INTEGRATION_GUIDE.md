# TelegramSender Library Integration Guide

This guide helps developers integrate the telegramsender library into their applications for sending text messages and images via Telegram Bot API.

## Installation

```bash
go get github.com/prilive-com/telegramsender
```

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
    
    "github.com/prilive-com/telegramsender/telegramsender"
)

func main() {
    // Load configuration
    cfg, err := telegramsender.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Validate configuration
    if err := telegramsender.ValidateConfig(cfg); err != nil {
        log.Fatalf("Invalid configuration: %v", err)
    }
    
    // Setup logger
    logger, err := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    
    // Create API client
    api := telegramsender.NewTelegramAPI(logger, cfg)
    
    // Now you can use api.SendMessage() and api.SendPhoto()
}
```

## Sending Text Messages

```go
func sendTextMessage(api *telegramsender.TelegramAPI, chatID int64) error {
    ctx := context.Background()
    
    message := telegramsender.MessageRequest{
        ChatID:                chatID,
        Text:                  "Hello from my app! 🚀",
        ParseMode:             "HTML",        // or "Markdown"
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
        Caption:   "Check out this image! 📸",
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

## Error Handling

The library provides automatic retry with exponential backoff for transient errors. Here's how to handle different error scenarios:

```go
func robustSend(api *telegramsender.TelegramAPI, chatID int64) {
    message := telegramsender.MessageRequest{
        ChatID: chatID,
        Text:   "Important message",
    }
    
    result, err := api.SendMessage(context.Background(), message)
    if err != nil {
        // The library already retried if appropriate
        // These errors are permanent failures
        
        if strings.Contains(err.Error(), "403") {
            log.Println("Bot was blocked by user or removed from group")
        } else if strings.Contains(err.Error(), "400") {
            log.Println("Bad request - check your parameters")
        } else if strings.Contains(err.Error(), "401") {
            log.Println("Invalid bot token")
        } else {
            log.Printf("Unexpected error: %v", err)
        }
        return
    }
    
    log.Printf("Success! Message ID: %d", result.MessageID)
}
```

## Best Practices

### 1. Reuse the API Client

```go
// DON'T create new client for each message
for _, msg := range messages {
    api := telegramsender.NewTelegramAPI(logger, cfg) // ❌ Bad
    api.SendMessage(ctx, msg)
}

// DO reuse the same client
api := telegramsender.NewTelegramAPI(logger, cfg) // ✅ Good
for _, msg := range messages {
    api.SendMessage(ctx, msg)
}
```

### 2. Handle Rate Limits

The library handles rate limiting automatically, but you can help by:

```go
// Batch messages with small delays
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

### 3. Image Size Considerations

- Telegram automatically compresses large images
- Maximum file size: 10MB for photos
- Recommended max dimensions: 1280px width
- Supported formats: JPEG, PNG, GIF, BMP, WEBP

### 4. Caption Length Limits

```go
// Captions have a 1024 character limit
caption := "Your long caption here..."
if len(caption) > 1024 {
    caption = caption[:1021] + "..."
}

photo := telegramsender.PhotoRequest{
    ChatID:  chatID,
    Photo:   imageURL,
    Caption: caption,
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
    api := telegramsender.NewTelegramAPI(logger, cfg)
    
    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        log.Println("Shutting down gracefully...")
        // Complete any pending sends
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
    telegram *telegramsender.TelegramAPI
    logger   *slog.Logger
}

func (ns *NotificationService) NotifyUser(userID int64, message string) error {
    return ns.telegram.SendMessage(context.Background(), 
        telegramsender.MessageRequest{
            ChatID: userID,
            Text:   message,
        })
}

func (ns *NotificationService) NotifyWithImage(userID int64, message, imageURL string) error {
    return ns.telegram.SendPhoto(context.Background(),
        telegramsender.PhotoRequest{
            ChatID:  userID,
            Photo:   imageURL,
            Caption: message,
        })
}
```

### 2. Alert System

```go
func sendAlert(api *telegramsender.TelegramAPI, alertChannel int64, severity, message string) error {
    emoji := map[string]string{
        "critical": "🔴",
        "warning":  "🟡",
        "info":     "🔵",
    }[severity]
    
    text := fmt.Sprintf("%s <b>%s Alert</b>\n\n%s\n\n<i>Time: %s</i>",
        emoji, strings.ToUpper(severity), message, time.Now().Format(time.RFC3339))
    
    return api.SendMessage(context.Background(),
        telegramsender.MessageRequest{
            ChatID:    alertChannel,
            Text:      text,
            ParseMode: "HTML",
        })
}
```

## Troubleshooting

### Common Issues

1. **"BOT_TOKEN must be set"** - Ensure environment variable is set
2. **"403 Forbidden"** - Bot was blocked or not in the chat
3. **"400 Bad Request"** - Invalid parameters (check chat_id, text length)
4. **Circuit breaker open** - Too many failures, will auto-recover
5. **Rate limit exceeded** - Automatic retry with backoff

### Debug Mode

Enable debug logging by setting log level:

```go
logger, _ := telegramsender.NewLogger(slog.LevelDebug, cfg.LogFilePath)
```

## Support

For issues or questions:
- Check the logs first (JSON format, easy to parse)
- Review error messages - they're descriptive
- Ensure your bot has proper permissions in groups
- Verify network connectivity to api.telegram.org

Remember: The library handles retries, rate limiting, and circuit breaking automatically. Focus on your business logic!