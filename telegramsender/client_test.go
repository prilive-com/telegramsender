package telegramsender

import (
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env file from parent directory for tests
	_ = godotenv.Load("../.env")
}

// getTestBotToken returns a bot token from environment variable for testing.
// Loads from .env file or environment. Skips test if not available.
func getTestBotToken(t *testing.T) string {
	t.Helper()
	token := os.Getenv("TEST_BOT_TOKEN")
	if token == "" {
		// Also try BOT_TOKEN (used in .env)
		token = os.Getenv("BOT_TOKEN")
	}
	if token == "" || token == "your_telegram_bot_token_here" {
		t.Skip("TEST_BOT_TOKEN or BOT_TOKEN not set (update .env file)")
	}
	return token
}

func TestNew_RequiresToken(t *testing.T) {
	_, err := New("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestNew_ValidatesTokenFormat(t *testing.T) {
	_, err := New("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token format")
	}
}

func TestNew_WithValidToken(t *testing.T) {
	token := os.Getenv("TEST_BOT_TOKEN")
	if token == "" {
		t.Skip("TEST_BOT_TOKEN not set")
	}

	client, err := New(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer client.Close()

	if client.Config().BotToken != token {
		t.Error("token not set correctly")
	}
}

func TestNew_WithOptions(t *testing.T) {
	token := os.Getenv("TEST_BOT_TOKEN")
	if token == "" {
		t.Skip("TEST_BOT_TOKEN not set")
	}

	client, err := New(token,
		WithMaxRetriesOption(5),
		WithRateLimitOption(30, 50),
		WithRetryOption(5, 200*time.Millisecond, 30*time.Second, 2.0),
		WithBreakerConfig(10, 3*time.Minute, 90*time.Second),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer client.Close()

	cfg := client.Config()

	if cfg.MaxRetries != 5 {
		t.Errorf("expected MaxRetries 5, got %d", cfg.MaxRetries)
	}
	if cfg.RateLimitRequests != 30 {
		t.Errorf("expected RateLimitRequests 30, got %f", cfg.RateLimitRequests)
	}
	if cfg.RateLimitBurst != 50 {
		t.Errorf("expected RateLimitBurst 50, got %d", cfg.RateLimitBurst)
	}
	if cfg.RetryInitialBackoff != 200*time.Millisecond {
		t.Errorf("expected RetryInitialBackoff 200ms, got %v", cfg.RetryInitialBackoff)
	}
	if cfg.BreakerMaxRequests != 10 {
		t.Errorf("expected BreakerMaxRequests 10, got %d", cfg.BreakerMaxRequests)
	}
}

func TestDefaultClientConfig(t *testing.T) {
	cfg := DefaultClientConfig()

	if cfg.BaseURL != DefaultBaseURL {
		t.Errorf("expected BaseURL %s, got %s", DefaultBaseURL, cfg.BaseURL)
	}
	if cfg.RequestTimeout != DefaultRequestTimeout {
		t.Errorf("expected RequestTimeout %v, got %v", DefaultRequestTimeout, cfg.RequestTimeout)
	}
	if cfg.MaxRetries != DefaultMaxRetries {
		t.Errorf("expected MaxRetries %d, got %d", DefaultMaxRetries, cfg.MaxRetries)
	}
	if cfg.RateLimitRequests != DefaultRateLimitRequests {
		t.Errorf("expected RateLimitRequests %f, got %f", DefaultRateLimitRequests, cfg.RateLimitRequests)
	}
	if cfg.RetryBackoffFactor != DefaultRetryBackoffFactor {
		t.Errorf("expected RetryBackoffFactor %f, got %f", DefaultRetryBackoffFactor, cfg.RetryBackoffFactor)
	}
}

func TestPresets(t *testing.T) {
	t.Run("ProductionPreset", func(t *testing.T) {
		cfg := DefaultClientConfig()
		ProductionPreset().apply(&cfg)

		if cfg.MaxRetries != 5 {
			t.Errorf("expected MaxRetries 5, got %d", cfg.MaxRetries)
		}
		if cfg.RateLimitRequests != 30 {
			t.Errorf("expected RateLimitRequests 30, got %f", cfg.RateLimitRequests)
		}
	})

	t.Run("DevelopmentPreset", func(t *testing.T) {
		cfg := DefaultClientConfig()
		DevelopmentPreset().apply(&cfg)

		if cfg.MaxRetries != 2 {
			t.Errorf("expected MaxRetries 2, got %d", cfg.MaxRetries)
		}
		if cfg.BreakerTimeout != 10*time.Second {
			t.Errorf("expected BreakerTimeout 10s, got %v", cfg.BreakerTimeout)
		}
	})

	t.Run("HighThroughputPreset", func(t *testing.T) {
		cfg := DefaultClientConfig()
		HighThroughputPreset().apply(&cfg)

		if cfg.MaxIdleConns != 50 {
			t.Errorf("expected MaxIdleConns 50, got %d", cfg.MaxIdleConns)
		}
		if cfg.RateLimitRequests != 50 {
			t.Errorf("expected RateLimitRequests 50, got %f", cfg.RateLimitRequests)
		}
	})
}

func TestClient_ImplementsSender(t *testing.T) {
	// This is a compile-time check, but we add a runtime test for clarity
	var _ Sender = (*Client)(nil)
}

func TestValidateClientConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		token := getTestBotToken(t)
		cfg := ClientConfig{
			BotToken:          token,
			MaxRetries:        3,
			RateLimitRequests: 10,
			RateLimitBurst:    20,
			RequestTimeout:    10 * time.Second,
		}
		if err := validateClientConfig(&cfg); err != nil {
			t.Errorf("validateClientConfig() unexpected error = %v", err)
		}
	})

	t.Run("missing bot token", func(t *testing.T) {
		cfg := ClientConfig{
			MaxRetries:        3,
			RateLimitRequests: 10,
			RateLimitBurst:    20,
			RequestTimeout:    10 * time.Second,
		}
		if err := validateClientConfig(&cfg); err == nil {
			t.Error("validateClientConfig() expected error for missing token")
		}
	})

	t.Run("invalid bot token format", func(t *testing.T) {
		cfg := ClientConfig{
			BotToken:          "invalid",
			MaxRetries:        3,
			RateLimitRequests: 10,
			RateLimitBurst:    20,
			RequestTimeout:    10 * time.Second,
		}
		if err := validateClientConfig(&cfg); err == nil {
			t.Error("validateClientConfig() expected error for invalid token")
		}
	})

	t.Run("negative max retries", func(t *testing.T) {
		token := getTestBotToken(t)
		cfg := ClientConfig{
			BotToken:          token,
			MaxRetries:        -1,
			RateLimitRequests: 10,
			RateLimitBurst:    20,
			RequestTimeout:    10 * time.Second,
		}
		if err := validateClientConfig(&cfg); err == nil {
			t.Error("validateClientConfig() expected error for negative retries")
		}
	})

	t.Run("zero rate limit", func(t *testing.T) {
		token := getTestBotToken(t)
		cfg := ClientConfig{
			BotToken:          token,
			MaxRetries:        3,
			RateLimitRequests: 0,
			RateLimitBurst:    20,
			RequestTimeout:    10 * time.Second,
		}
		if err := validateClientConfig(&cfg); err == nil {
			t.Error("validateClientConfig() expected error for zero rate limit")
		}
	})
}
