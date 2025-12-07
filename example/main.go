package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prilive-com/telegramsender/v2/telegramsender"
)

func main() {
	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture system signals (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration from environment variables
	cfg, err := telegramsender.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate configuration immediately
	if err := telegramsender.ValidateConfig(cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Setup structured logging
	logger, err := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close() // Properly close log file handle

	// Initialize Telegram API client
	telegramAPI := telegramsender.NewTelegramAPI(logger, cfg)

	// Setup goroutine to handle shutdown
	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	logger.Info("Example Telegram sender running. Press Ctrl+C to stop.")

	// Check command line arguments
	if len(os.Args) < 2 {
		logger.Info("Usage:")
		logger.Info("  go run main.go send         - Send a test text message")
		logger.Info("  go run main.go sendphoto    - Send a test photo")
		logger.Info("Environment: TEST_CHAT_ID must be set")
		<-ctx.Done()
		return
	}

	// Demo: Send a message or photo
	command := os.Args[1]
	switch command {
	case "send":
		// Create a context with timeout for the message sending operation
		sendCtx, sendCancel := context.WithTimeout(ctx, 30*time.Second)
		defer sendCancel()

		// Test chat ID (must be set in environment or passed as argument)
		chatID := int64(0)
		if envChatID := os.Getenv("TEST_CHAT_ID"); envChatID != "" {
			if _, err := fmt.Sscanf(envChatID, "%d", &chatID); err != nil {
				logger.Error("Invalid TEST_CHAT_ID format", "error", err)
				return
			}
		}

		if chatID == 0 {
			logger.Error("TEST_CHAT_ID environment variable must be set to send a test message")
			return
		}

		// Prepare and send message
		msgRequest := telegramsender.MessageRequest{
			ChatID:                chatID,
			Text:                  "Hello from TelegramSender! 🚀\nThis is a test message.",
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
		}

		logger.Info("Sending test message", "chat_id", chatID)
		result, err := telegramAPI.SendMessage(sendCtx, msgRequest)
		if err != nil {
			logger.Error("Failed to send message", "error", err)
			return
		}

		logger.Info("Message sent successfully", "message_id", result.MessageID)

	case "sendphoto":
		// Create a context with timeout for the photo sending operation
		sendCtx, sendCancel := context.WithTimeout(ctx, 30*time.Second)
		defer sendCancel()

		// Test chat ID (must be set in environment or passed as argument)
		chatID := int64(0)
		if envChatID := os.Getenv("TEST_CHAT_ID"); envChatID != "" {
			if _, err := fmt.Sscanf(envChatID, "%d", &chatID); err != nil {
				logger.Error("Invalid TEST_CHAT_ID format", "error", err)
				return
			}
		}

		if chatID == 0 {
			logger.Error("TEST_CHAT_ID environment variable must be set to send a test photo")
			return
		}

		// Get photo URL or file_id from environment or use a default test image
		photoURL := os.Getenv("TEST_PHOTO_URL")
		if photoURL == "" {
			// Using a small test image from via.placeholder.com (a legitimate placeholder image service)
			photoURL = "https://via.placeholder.com/300x200/007bff/ffffff?text=TelegramSender+Test"
			logger.Info("Using default test photo URL", "url", photoURL)
		}

		// Prepare and send photo
		photoRequest := telegramsender.PhotoRequest{
			ChatID:    chatID,
			Photo:     photoURL,
			Caption:   "Hello from TelegramSender! 📸\nThis is a test photo with <b>HTML</b> formatting.",
			ParseMode: "HTML",
		}

		logger.Info("Sending test photo", "chat_id", chatID)
		result, err := telegramAPI.SendPhoto(sendCtx, photoRequest)
		if err != nil {
			logger.Error("Failed to send photo", "error", err)
			return
		}

		logger.Info("Photo sent successfully", "message_id", result.MessageID)

	default:
		logger.Error("Unknown command", "command", command)
		logger.Info("Usage:")
		logger.Info("  go run main.go send         - Send a test text message")
		logger.Info("  go run main.go sendphoto    - Send a test photo")
	}

	// Wait for context cancellation (from signal handler)
	<-ctx.Done()
	logger.Info("Example application shutting down")
}

