// Package testconfig provides test configuration loaded from environment variables.
package testconfig

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prilive-com/telegramsender/v2/telegramsender"
)

// TestConfig holds configuration for tests loaded from environment.
type TestConfig struct {
	// BotToken is a valid Telegram bot token for integration tests
	BotToken string

	// ChatID is the chat to send test messages to
	ChatID int64

	// PhotoURL is a test photo URL
	PhotoURL string

	// AllowedPhotoDirs are directories allowed for photo uploads
	AllowedPhotoDirs []string

	// BaseURL is the Telegram API base URL
	BaseURL string

	// LogFilePath is the path for test logs
	LogFilePath string

	// RequestTimeout for HTTP requests
	RequestTimeout time.Duration

	// MaxRetries for failed requests
	MaxRetries int

	// RateLimitRequests per second
	RateLimitRequests float64

	// RateLimitBurst size
	RateLimitBurst int
}

// Load loads test configuration from environment variables.
// Returns a TestConfig with values from environment or sensible defaults.
func Load() *TestConfig {
	cfg := &TestConfig{
		BotToken:          os.Getenv("TEST_BOT_TOKEN"),
		PhotoURL:          getEnvOrDefault("TEST_PHOTO_URL", ""),
		BaseURL:           getEnvOrDefault("TEST_BASE_URL", telegramsender.DefaultBaseURL),
		LogFilePath:       getEnvOrDefault("TEST_LOG_FILE_PATH", "logs/test.log"),
		RequestTimeout:    parseDurationOrDefault("TEST_REQUEST_TIMEOUT", telegramsender.DefaultRequestTimeout),
		MaxRetries:        parseIntOrDefault("TEST_MAX_RETRIES", telegramsender.DefaultMaxRetries),
		RateLimitRequests: parseFloatOrDefault("TEST_RATE_LIMIT_REQUESTS", telegramsender.DefaultRateLimitRequests),
		RateLimitBurst:    parseIntOrDefault("TEST_RATE_LIMIT_BURST", telegramsender.DefaultRateLimitBurst),
	}

	// Parse chat ID
	if chatIDStr := os.Getenv("TEST_CHAT_ID"); chatIDStr != "" {
		if id, err := strconv.ParseInt(chatIDStr, 10, 64); err == nil {
			cfg.ChatID = id
		}
	}

	// Parse allowed photo dirs
	if dirs := os.Getenv("TEST_ALLOWED_PHOTO_DIRS"); dirs != "" {
		cfg.AllowedPhotoDirs = strings.Split(dirs, ",")
		for i, dir := range cfg.AllowedPhotoDirs {
			cfg.AllowedPhotoDirs[i] = strings.TrimSpace(dir)
		}
	}

	return cfg
}

// HasBotToken returns true if a valid bot token is configured.
func (c *TestConfig) HasBotToken() bool {
	return c.BotToken != "" && strings.Contains(c.BotToken, ":")
}

// HasChatID returns true if a chat ID is configured.
func (c *TestConfig) HasChatID() bool {
	return c.ChatID != 0
}

// CanRunIntegrationTests returns true if all required config for integration tests is present.
func (c *TestConfig) CanRunIntegrationTests() bool {
	return c.HasBotToken() && c.HasChatID()
}

// ToSenderConfig converts TestConfig to telegramsender.Config.
func (c *TestConfig) ToSenderConfig() *telegramsender.Config {
	return telegramsender.NewConfig(
		c.BotToken,
		telegramsender.WithBaseURL(c.BaseURL),
		telegramsender.WithLogFilePath(c.LogFilePath),
		telegramsender.WithRequestTimeout(c.RequestTimeout),
		telegramsender.WithMaxRetries(c.MaxRetries),
		telegramsender.WithRateLimit(c.RateLimitRequests, c.RateLimitBurst),
		telegramsender.WithAllowedPhotoDirs(c.AllowedPhotoDirs...),
	)
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseIntOrDefault(key string, defaultValue int) int {
	if str := os.Getenv(key); str != "" {
		if val, err := strconv.Atoi(str); err == nil {
			return val
		}
	}
	return defaultValue
}

func parseFloatOrDefault(key string, defaultValue float64) float64 {
	if str := os.Getenv(key); str != "" {
		if val, err := strconv.ParseFloat(str, 64); err == nil {
			return val
		}
	}
	return defaultValue
}

func parseDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if str := os.Getenv(key); str != "" {
		if val, err := time.ParseDuration(str); err == nil {
			return val
		}
	}
	return defaultValue
}
