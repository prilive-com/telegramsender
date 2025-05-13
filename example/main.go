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

	"github.com/prilive-com/telegramsender/telegramsender"
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

	// Initialize Telegram API client
	telegramAPI := telegramsender.NewTelegramAPI(logger, cfg)

	// Setup goroutine to handle shutdown
	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	logger.Info("Example Telegram sender running. Press Ctrl+C to stop.")

	// Demo: Send a message
	if len(os.Args) > 1 && os.Args[1] == "send" {
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
	} else {
		logger.Info("Run with 'send' argument to send a test message")
		logger.Info("Example: TEST_CHAT_ID=123456789 go run main.go send")
	}

	// Wait for context cancellation (from signal handler)
	<-ctx.Done()
	logger.Info("Example application shutting down")
}