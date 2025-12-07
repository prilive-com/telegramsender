package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/prilive-com/telegramsender/v2/telegramsender"
)

func TestTelegramError_Error(t *testing.T) {
	tests := []struct {
		name       string
		err        *telegramsender.TelegramError
		wantSubstr string
	}{
		{
			name:       "basic error",
			err:        telegramsender.NewTelegramError(400, "Bad Request"),
			wantSubstr: "400",
		},
		{
			name:       "error with retry after",
			err:        telegramsender.NewTelegramErrorWithRetry(429, "Too Many Requests", 30*time.Second),
			wantSubstr: "retry after",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if msg == "" {
				t.Error("Error() returned empty string")
			}
			if tt.wantSubstr != "" && !containsIgnoreCase(msg, tt.wantSubstr) {
				t.Errorf("Error() = %q, want substring %q", msg, tt.wantSubstr)
			}
		})
	}
}

func TestTelegramError_Is(t *testing.T) {
	err429 := telegramsender.NewTelegramError(429, "Too Many Requests")
	err400 := telegramsender.NewTelegramError(400, "Bad Request")

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "same error code matches",
			err:    err429,
			target: telegramsender.NewTelegramError(429, "Different message"),
			want:   true,
		},
		{
			name:   "different error code no match",
			err:    err429,
			target: err400,
			want:   false,
		},
		{
			name:   "non-telegram error no match",
			err:    err429,
			target: errors.New("some error"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTelegramError_IsRetryable(t *testing.T) {
	tests := []struct {
		name string
		code int
		want bool
	}{
		{name: "429 Too Many Requests", code: 429, want: true},
		{name: "500 Internal Server Error", code: 500, want: true},
		{name: "502 Bad Gateway", code: 502, want: true},
		{name: "503 Service Unavailable", code: 503, want: true},
		{name: "504 Gateway Timeout", code: 504, want: true},
		{name: "400 Bad Request", code: 400, want: false},
		{name: "401 Unauthorized", code: 401, want: false},
		{name: "403 Forbidden", code: 403, want: false},
		{name: "404 Not Found", code: 404, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := telegramsender.NewTelegramError(tt.code, "test")
			if got := err.IsRetryable(); got != tt.want {
				t.Errorf("IsRetryable() for code %d = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestTelegramError_As(t *testing.T) {
	originalErr := telegramsender.NewTelegramErrorWithRetry(429, "Rate limited", 60*time.Second)

	var telegramErr *telegramsender.TelegramError
	if !errors.As(originalErr, &telegramErr) {
		t.Fatal("errors.As should match TelegramError")
	}

	if telegramErr.Code != 429 {
		t.Errorf("Code = %d, want 429", telegramErr.Code)
	}

	if telegramErr.RetryAfter != 60*time.Second {
		t.Errorf("RetryAfter = %v, want 60s", telegramErr.RetryAfter)
	}
}

func TestValidationError(t *testing.T) {
	err := telegramsender.NewValidationError("chat_id", "cannot be zero")

	t.Run("Error message format", func(t *testing.T) {
		msg := err.Error()
		if !containsIgnoreCase(msg, "chat_id") {
			t.Errorf("Error() should contain field name, got %q", msg)
		}
		if !containsIgnoreCase(msg, "cannot be zero") {
			t.Errorf("Error() should contain message, got %q", msg)
		}
	})

	t.Run("errors.As works", func(t *testing.T) {
		var validationErr *telegramsender.ValidationError
		if !errors.As(err, &validationErr) {
			t.Error("errors.As should match ValidationError")
		}
	})
}

func TestConfigError(t *testing.T) {
	err := telegramsender.NewConfigError("BOT_TOKEN", "must be set")

	t.Run("Error message format", func(t *testing.T) {
		msg := err.Error()
		if !containsIgnoreCase(msg, "BOT_TOKEN") {
			t.Errorf("Error() should contain key name, got %q", msg)
		}
	})

	t.Run("errors.As works", func(t *testing.T) {
		var configErr *telegramsender.ConfigError
		if !errors.As(err, &configErr) {
			t.Error("errors.As should match ConfigError")
		}
	})
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrInvalidConfig", telegramsender.ErrInvalidConfig},
		{"ErrRateLimitExceeded", telegramsender.ErrRateLimitExceeded},
		{"ErrCircuitBreakerOpen", telegramsender.ErrCircuitBreakerOpen},
		{"ErrMaxRetriesExceeded", telegramsender.ErrMaxRetriesExceeded},
		{"ErrInvalidRequest", telegramsender.ErrInvalidRequest},
		{"ErrPathTraversal", telegramsender.ErrPathTraversal},
		{"ErrResponseTooLarge", telegramsender.ErrResponseTooLarge},
		{"ErrInvalidBaseURL", telegramsender.ErrInvalidBaseURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("Sentinel error should not be nil")
			}
			if tt.err.Error() == "" {
				t.Error("Sentinel error should have message")
			}
		})
	}
}

// Helper function
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			(len(s) > 0 && len(substr) > 0 &&
				(s[0] == substr[0] || s[0]+'a'-'A' == substr[0] || s[0] == substr[0]+'a'-'A') &&
				containsIgnoreCase(s[1:], substr[1:])) ||
			(len(s) > 0 && containsIgnoreCase(s[1:], substr)))
}
