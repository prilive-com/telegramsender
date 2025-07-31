# telegramsender

**telegramsender** is a production‑ready, Go 1.24+ library and companion example application that lets you send Telegram bot messages with resilience, performance, and observability features for production environments in Docker or Kubernetes.

---

## ✨ What you get

| Capability            | Details                                                                                            |
| --------------------- | -------------------------------------------------------------------------------------------------- |
| Message Types         | Send text messages and photos with captions, HTML/Markdown formatting, and reply options           |
| Resilient sender      | Retry with exponential backoff, circuit-breaker (sony/gobreaker), rate limiting (golang.org/x/time/rate) |
| Performance           | Connection pooling, optimized HTTP client, efficient message handling                              |
| Observability         | Go 1.24 `log/slog` JSON logs, structured errors                                                   |
| Configurable          | Everything via env‑vars → Config struct (defaults supplied)                                        |
| Container‑ready       | Multi‑stage Dockerfile, Docker‑Compose example, environment file sample                            |
| Kubernetes‑native     | Ready for deployment in any CNCF-conformant cluster                                                |

---

## 🏗️ Architecture

```
┌──────────────┐   Message Request    ┌──────────────┐   HTTP Client   ┌────────┐
│Your App Logic│ ────────────────────▶│  TelegramAPI │ ───────────────▶│Telegram│
└──────────────┘                      │ (rate‑limit) │                 └────────┘
                                      │ (circuit‑br) │
                                      │ (retry)      │
                                      └──────────────┘
```

* **TelegramAPI**
  * Manages HTTP connection pool
  * Implements rate-limiting & circuit-breaking
  * Handles retries with exponential backoff
  * Provides idempotent message sending
* **Config**
  * Populated entirely from environment variables with sensible defaults

---

## 🚀 Usage

### As a Library

```go
import "github.com/prilive-com/telegramsender/telegramsender"

// Initialize
cfg, _ := telegramsender.LoadConfig()
logger, _ := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
api := telegramsender.NewTelegramAPI(logger, cfg)

// Send text message
msgRequest := telegramsender.MessageRequest{
    ChatID:    123456789,
    Text:      "Hello, World!",
    ParseMode: "HTML",
}
result, err := api.SendMessage(ctx, msgRequest)

// Send photo
photoRequest := telegramsender.PhotoRequest{
    ChatID:    123456789,
    Photo:     "https://example.com/photo.jpg", // or file_id
    Caption:   "Beautiful photo! 📸",
    ParseMode: "HTML",
}
result, err := api.SendPhoto(ctx, photoRequest)
```

### Example Application

```bash
# Send a test text message
TEST_CHAT_ID=123456789 go run example/main.go send

# Send a test photo
TEST_CHAT_ID=123456789 go run example/main.go sendphoto

# Send a custom photo
TEST_CHAT_ID=123456789 TEST_PHOTO_URL="https://example.com/image.jpg" go run example/main.go sendphoto
```

---

## Environment Variables

| Variable                    | Default                 | Description                                 |
| --------------------------- | ----------------------- | ------------------------------------------- |
| `BOT_TOKEN`                 | *(required)*            | Your Telegram Bot API token                 |
| `BASE_URL`                  | `https://api.telegram.org` | Base URL for Telegram API                |
| `LOG_FILE_PATH`             | `logs/telegramsender.log` | File plus JSON logs to stdout            |
| `REQUEST_TIMEOUT`           | `10s`                   | Timeout for HTTP requests                   |
| `KEEP_ALIVE`                | `30s`                   | HTTP keep-alive duration                    |
| `MAX_IDLE_CONNS`            | `10`                    | Max idle connections in HTTP pool           |
| `IDLE_CONN_TIMEOUT`         | `90s`                   | Idle connection timeout                     |
| `RATE_LIMIT_REQUESTS`       | `10`                    | Allowed requests per second                 |
| `RATE_LIMIT_BURST`          | `20`                    | Extra burst tokens                          |
| `BREAKER_MAX_REQUESTS`      | `5`                     | Requests allowed in half‑open state         |
| `BREAKER_INTERVAL`          | `2m`                    | Window that resets failure counters         |
| `BREAKER_TIMEOUT`           | `60s`                   | How long breaker stays open                 |
| `MAX_RETRIES`               | `3`                     | Maximum number of retries on failure        |
| `RETRY_INITIAL_BACKOFF`     | `100ms`                 | Initial backoff time for first retry        |
| `RETRY_MAX_BACKOFF`         | `10s`                   | Maximum backoff time for retries           |
| `RETRY_BACKOFF_FACTOR`      | `2.0`                   | Multiplier for exponential backoff         |

---

## 📜 License

MIT © 2025 Prilive Com