package tests

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/prilive-com/telegramsender/v2/telegramsender"
	"github.com/prilive-com/telegramsender/v2/tests/testconfig"
)

// Integration tests require TEST_BOT_TOKEN and TEST_CHAT_ID to be set.
// These tests actually call the Telegram API.

func TestIntegration_SendMessage(t *testing.T) {
	cfg := testconfig.Load()
	if !cfg.CanRunIntegrationTests() {
		t.Skip("Skipping integration test: TEST_BOT_TOKEN and TEST_CHAT_ID required")
	}

	senderCfg := cfg.ToSenderConfig()
	logger, err := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	api := telegramsender.NewTelegramAPI(logger, senderCfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := telegramsender.MessageRequest{
		ChatID:    cfg.ChatID,
		Text:      "Integration test message from telegramsender v2",
		ParseMode: "HTML",
	}

	result, err := api.SendMessage(ctx, req)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if result.MessageID == 0 {
		t.Error("Expected non-zero message ID")
	}

	t.Logf("Message sent successfully, ID: %d", result.MessageID)
}

func TestIntegration_SendMessageWithFormatting(t *testing.T) {
	cfg := testconfig.Load()
	if !cfg.CanRunIntegrationTests() {
		t.Skip("Skipping integration test: TEST_BOT_TOKEN and TEST_CHAT_ID required")
	}

	senderCfg := cfg.ToSenderConfig()
	logger, err := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	api := telegramsender.NewTelegramAPI(logger, senderCfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := telegramsender.MessageRequest{
		ChatID:    cfg.ChatID,
		Text:      "<b>Bold</b> and <i>italic</i> test from v2",
		ParseMode: "HTML",
	}

	result, err := api.SendMessage(ctx, req)
	if err != nil {
		t.Fatalf("SendMessage with formatting failed: %v", err)
	}

	t.Logf("Formatted message sent, ID: %d", result.MessageID)
}

func TestIntegration_SendPhoto(t *testing.T) {
	cfg := testconfig.Load()
	if !cfg.CanRunIntegrationTests() {
		t.Skip("Skipping integration test: TEST_BOT_TOKEN and TEST_CHAT_ID required")
	}

	senderCfg := cfg.ToSenderConfig()
	logger, err := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	api := telegramsender.NewTelegramAPI(logger, senderCfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	photoURL := cfg.PhotoURL
	if photoURL == "" {
		photoURL = "https://via.placeholder.com/300x200/007bff/ffffff?text=Test"
	}

	req := telegramsender.PhotoRequest{
		ChatID:    cfg.ChatID,
		Photo:     photoURL,
		Caption:   "Integration test photo from telegramsender v2",
		ParseMode: "HTML",
	}

	result, err := api.SendPhoto(ctx, req)
	if err != nil {
		t.Fatalf("SendPhoto failed: %v", err)
	}

	if result.MessageID == 0 {
		t.Error("Expected non-zero message ID")
	}

	t.Logf("Photo sent successfully, ID: %d", result.MessageID)
}

func TestIntegration_ContextCancellation(t *testing.T) {
	cfg := testconfig.Load()
	if !cfg.CanRunIntegrationTests() {
		t.Skip("Skipping integration test: TEST_BOT_TOKEN and TEST_CHAT_ID required")
	}

	senderCfg := cfg.ToSenderConfig()
	logger, err := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	api := telegramsender.NewTelegramAPI(logger, senderCfg)

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := telegramsender.MessageRequest{
		ChatID: cfg.ChatID,
		Text:   "This should not be sent",
	}

	_, err = api.SendMessage(ctx, req)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

func TestIntegration_InvalidChatID(t *testing.T) {
	cfg := testconfig.Load()
	if !cfg.HasBotToken() {
		t.Skip("Skipping integration test: TEST_BOT_TOKEN required")
	}

	senderCfg := cfg.ToSenderConfig()
	logger, err := telegramsender.NewLogger(slog.LevelInfo, cfg.LogFilePath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	api := telegramsender.NewTelegramAPI(logger, senderCfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := telegramsender.MessageRequest{
		ChatID: 999999999999, // Invalid chat ID
		Text:   "This should fail",
	}

	_, err = api.SendMessage(ctx, req)
	if err == nil {
		t.Error("Expected error for invalid chat ID")
	}

	t.Logf("Got expected error: %v", err)
}
