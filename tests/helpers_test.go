package tests

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prilive-com/telegramsender/v2/telegramsender"
	"github.com/prilive-com/telegramsender/v2/tests/testconfig"
)

var cfg = testconfig.Load()

func TestValidateBotToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		valid bool
	}{
		{
			name:  "valid token format",
			token: "123456789:ABCDefGhIJKlmNoPQRsTUVwxyZ1234567890",
			valid: true,
		},
		{
			name:  "empty token",
			token: "",
			valid: false,
		},
		{
			name:  "no colon separator",
			token: "123456789ABCDefGhIJKlmNoPQRsTUVwxyZ",
			valid: false,
		},
		{
			name:  "non-numeric bot ID",
			token: "abc:ABCDefGhIJKlmNoPQRsTUVwxyZ1234567890",
			valid: false,
		},
		{
			name:  "token part too short",
			token: "123456789:short",
			valid: false,
		},
		{
			name:  "multiple colons invalid",
			token: "123456789:ABC:DEF:GhIJKlmNoPQRsTUVwxyZ",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We test via ValidateConfig since validateBotToken is unexported
			config := telegramsender.NewConfig(tt.token)
			config.LogFilePath = "logs/test.log" // Required field
			err := telegramsender.ValidateConfig(config)

			if tt.valid && err != nil && strings.Contains(err.Error(), "BOT_TOKEN") {
				t.Errorf("Expected valid token, got error: %v", err)
			}
			if !tt.valid && (err == nil || !strings.Contains(err.Error(), "BOT_TOKEN")) {
				t.Errorf("Expected BOT_TOKEN error for invalid token %q", tt.token)
			}
		})
	}
}

func TestValidateBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid telegram API URL",
			url:     cfg.BaseURL,
			wantErr: false,
		},
		{
			name:    "http scheme rejected",
			url:     "http://api.telegram.org",
			wantErr: true,
		},
		{
			name:    "unknown host rejected",
			url:     "https://malicious-site.com",
			wantErr: true,
		},
		{
			name:    "invalid URL format",
			url:     "not-a-valid-url",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := telegramsender.ValidateBaseURL(tt.url)
			gotErr := err != nil

			if gotErr != tt.wantErr {
				t.Errorf("ValidateBaseURL(%q) error = %v, wantErr = %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePhotoPath(t *testing.T) {
	// Create temp directory and file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_image.jpg")
	if err := os.WriteFile(tmpFile, []byte("fake image data"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		allowedDirs []string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid absolute path no restrictions",
			path:        tmpFile,
			allowedDirs: nil,
			wantErr:     false,
		},
		{
			name:        "valid path within allowed directory",
			path:        tmpFile,
			allowedDirs: []string{tmpDir},
			wantErr:     false,
		},
		{
			name:        "path outside allowed directories",
			path:        tmpFile,
			allowedDirs: []string{"/some/other/directory"},
			wantErr:     true,
			errContains: "not in allowed",
		},
		{
			name:        "relative path rejected",
			path:        "relative/path/image.jpg",
			allowedDirs: nil,
			wantErr:     true,
			errContains: "absolute",
		},
		{
			name:        "path traversal attempt with double dots",
			path:        tmpDir + "/../../../etc/passwd",
			allowedDirs: nil,
			wantErr:     true,
			errContains: "traversal",
		},
		{
			name:        "nonexistent file",
			path:        "/nonexistent/path/image.jpg",
			allowedDirs: nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := telegramsender.ValidatePhotoPath(tt.path, tt.allowedDirs)
			gotErr := err != nil

			if gotErr != tt.wantErr {
				t.Errorf("ValidatePhotoPath(%q) error = %v, wantErr = %v", tt.path, err, tt.wantErr)
			}

			if tt.errContains != "" && err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("Error %q should contain %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestValidateMessageRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     telegramsender.MessageRequest
		wantErr bool
	}{
		{
			name: "valid request with HTML",
			req: telegramsender.MessageRequest{
				ChatID:    cfg.ChatID,
				Text:      "Test message",
				ParseMode: "HTML",
			},
			wantErr: cfg.ChatID == 0, // Only error if no chat ID configured
		},
		{
			name: "zero chat ID",
			req: telegramsender.MessageRequest{
				ChatID: 0,
				Text:   "Test message",
			},
			wantErr: true,
		},
		{
			name: "empty text",
			req: telegramsender.MessageRequest{
				ChatID: 123456789,
				Text:   "",
			},
			wantErr: true,
		},
		{
			name: "invalid parse mode",
			req: telegramsender.MessageRequest{
				ChatID:    123456789,
				Text:      "Test",
				ParseMode: "INVALID",
			},
			wantErr: true,
		},
		{
			name: "valid Markdown parse mode",
			req: telegramsender.MessageRequest{
				ChatID:    123456789,
				Text:      "Test",
				ParseMode: "Markdown",
			},
			wantErr: false,
		},
		{
			name: "valid MarkdownV2 parse mode",
			req: telegramsender.MessageRequest{
				ChatID:    123456789,
				Text:      "Test",
				ParseMode: "MarkdownV2",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := telegramsender.ValidateMessageRequest(tt.req)
			gotErr := err != nil

			if gotErr != tt.wantErr {
				t.Errorf("ValidateMessageRequest() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMessageRequest_TextLength(t *testing.T) {
	// Create text that exceeds max message length
	maxLen := telegramsender.MaxMessageLength
	longText := strings.Repeat("a", maxLen+1)

	req := telegramsender.MessageRequest{
		ChatID: 123456789,
		Text:   longText,
	}

	err := telegramsender.ValidateMessageRequest(req)
	if err == nil {
		t.Errorf("Expected error for text exceeding %d characters", maxLen)
	}

	var validationErr *telegramsender.ValidationError
	if !errors.As(err, &validationErr) {
		t.Errorf("Expected ValidationError, got %T", err)
	}
}

func TestValidateConfig(t *testing.T) {
	// Use test config as base for valid configuration
	validToken := "123456789:ABCDefGhIJKlmNoPQRsTUVwxyZ1234567890"

	tests := []struct {
		name       string
		modifyFunc func(*telegramsender.Config)
		wantErr    bool
	}{
		{
			name:       "valid config",
			modifyFunc: func(c *telegramsender.Config) {},
			wantErr:    false,
		},
		{
			name:       "empty bot token",
			modifyFunc: func(c *telegramsender.Config) { c.BotToken = "" },
			wantErr:    true,
		},
		{
			name:       "invalid bot token",
			modifyFunc: func(c *telegramsender.Config) { c.BotToken = "invalid" },
			wantErr:    true,
		},
		{
			name:       "empty log path",
			modifyFunc: func(c *telegramsender.Config) { c.LogFilePath = "" },
			wantErr:    true,
		},
		{
			name:       "empty base URL",
			modifyFunc: func(c *telegramsender.Config) { c.BaseURL = "" },
			wantErr:    true,
		},
		{
			name:       "invalid base URL host",
			modifyFunc: func(c *telegramsender.Config) { c.BaseURL = "https://evil.com" },
			wantErr:    true,
		},
		{
			name:       "zero request timeout",
			modifyFunc: func(c *telegramsender.Config) { c.RequestTimeout = 0 },
			wantErr:    true,
		},
		{
			name:       "negative max retries",
			modifyFunc: func(c *telegramsender.Config) { c.MaxRetries = -1 },
			wantErr:    true,
		},
		{
			name:       "zero rate limit",
			modifyFunc: func(c *telegramsender.Config) { c.RateLimitRequests = 0 },
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh valid config for each test
			config := telegramsender.NewConfig(validToken,
				telegramsender.WithLogFilePath(cfg.LogFilePath),
			)
			tt.modifyFunc(config)

			err := telegramsender.ValidateConfig(config)
			gotErr := err != nil

			if gotErr != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
