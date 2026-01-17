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
			wantSubstr: "code=400",
		},
		{
			name:       "error with retry after",
			err:        telegramsender.NewTelegramErrorWithRetry(429, "Too Many Requests", 30*time.Second),
			wantSubstr: "retry_after=",
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
	// New behavior: errors.Is() matches against sentinel errors via Unwrap()
	errMessageNotFound := telegramsender.NewTelegramError(400, "Bad Request: message to edit not found")
	errChatNotFound := telegramsender.NewTelegramError(400, "Bad Request: chat not found")
	errGeneric := telegramsender.NewTelegramError(400, "Bad Request")

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "message not found matches sentinel",
			err:    errMessageNotFound,
			target: telegramsender.ErrMessageNotFound,
			want:   true,
		},
		{
			name:   "chat not found matches sentinel",
			err:    errChatNotFound,
			target: telegramsender.ErrChatNotFound,
			want:   true,
		},
		{
			name:   "generic error does not match specific sentinel",
			err:    errGeneric,
			target: telegramsender.ErrMessageNotFound,
			want:   false,
		},
		{
			name:   "non-telegram error no match",
			err:    errMessageNotFound,
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
		// Common errors
		{"ErrInvalidConfig", telegramsender.ErrInvalidConfig},
		{"ErrRateLimitExceeded", telegramsender.ErrRateLimitExceeded},
		{"ErrCircuitBreakerOpen", telegramsender.ErrCircuitBreakerOpen},
		{"ErrMaxRetriesExceeded", telegramsender.ErrMaxRetriesExceeded},
		{"ErrInvalidRequest", telegramsender.ErrInvalidRequest},
		{"ErrPathTraversal", telegramsender.ErrPathTraversal},
		{"ErrResponseTooLarge", telegramsender.ErrResponseTooLarge},
		{"ErrInvalidBaseURL", telegramsender.ErrInvalidBaseURL},
		// Message errors
		{"ErrMessageNotFound", telegramsender.ErrMessageNotFound},
		{"ErrMessageNotModified", telegramsender.ErrMessageNotModified},
		{"ErrMessageCantBeEdited", telegramsender.ErrMessageCantBeEdited},
		{"ErrMessageCantBeDeleted", telegramsender.ErrMessageCantBeDeleted},
		{"ErrMessageTooOld", telegramsender.ErrMessageTooOld},
		// Callback errors
		{"ErrInvalidCallbackData", telegramsender.ErrInvalidCallbackData},
		{"ErrCallbackQueryExpired", telegramsender.ErrCallbackQueryExpired},
		// Chat/user errors
		{"ErrChatNotFound", telegramsender.ErrChatNotFound},
		{"ErrBotKicked", telegramsender.ErrBotKicked},
		{"ErrBotBlocked", telegramsender.ErrBotBlocked},
		{"ErrUserDeactivated", telegramsender.ErrUserDeactivated},
		{"ErrNoRights", telegramsender.ErrNoRights},
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

func TestDetectSentinel(t *testing.T) {
	tests := []struct {
		description string
		wantErr     error
	}{
		{"Bad Request: message to edit not found", telegramsender.ErrMessageNotFound},
		{"Bad Request: message to delete not found", telegramsender.ErrMessageNotFound},
		{"Bad Request: message is not modified", telegramsender.ErrMessageNotModified},
		{"Bad Request: message can't be edited", telegramsender.ErrMessageCantBeEdited},
		{"Bad Request: message can't be deleted", telegramsender.ErrMessageCantBeDeleted},
		{"Forbidden: bot was kicked from the group chat", telegramsender.ErrBotKicked},
		{"Forbidden: bot was blocked by the user", telegramsender.ErrBotBlocked},
		{"Bad Request: chat not found", telegramsender.ErrChatNotFound},
		{"Bad Request: query is too old", telegramsender.ErrCallbackQueryExpired},
		{"Bad Request: BUTTON_DATA_INVALID", telegramsender.ErrInvalidCallbackData},
		{"Bad Request: not enough rights to send text messages", telegramsender.ErrNoRights},
		{"Bad Request: user is deactivated", telegramsender.ErrUserDeactivated},
		{"Bad Request: some unknown error", nil},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := telegramsender.NewTelegramError(400, tt.description)
			if tt.wantErr == nil {
				if err.Cause != nil {
					t.Errorf("expected nil Cause, got %v", err.Cause)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("errors.Is() = false, want true for %v", tt.wantErr)
				}
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
