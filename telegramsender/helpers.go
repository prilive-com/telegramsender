package telegramsender

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ensureLogPath creates all parent directories for the log file.
func ensureLogPath(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}

// ValidateConfig performs pre-flight sanity checks.
func ValidateConfig(cfg *Config) error {
	switch {
	case cfg.BotToken == "":
		return errors.New("BOT_TOKEN must be set")
	case !validateBotToken(cfg.BotToken):
		return errors.New("BOT_TOKEN format is invalid")
	case cfg.LogFilePath == "":
		return errors.New("LOG_FILE_PATH must be set")
	case cfg.BaseURL == "":
		return errors.New("BASE_URL must be set")
	// Validate timeout values are positive
	case cfg.RequestTimeout <= 0:
		return errors.New("REQUEST_TIMEOUT must be positive")
	case cfg.RetryInitialBackoff <= 0:
		return errors.New("RETRY_INITIAL_BACKOFF must be positive")
	case cfg.RetryMaxBackoff <= 0:
		return errors.New("RETRY_MAX_BACKOFF must be positive")
	case cfg.RetryBackoffFactor <= 0:
		return errors.New("RETRY_BACKOFF_FACTOR must be positive")
	case cfg.MaxRetries < 0:
		return errors.New("MAX_RETRIES must be non-negative")
	default:
		return nil
	}
}

// validateBotToken checks if the token has the correct format.
// Telegram bot tokens follow the pattern: 123456789:ABCDefGhIJKlmNoPQRsTUVwxyZ
func validateBotToken(token string) bool {
	// Check if the token contains a colon (which separates the bot ID from the token)
	if len(token) < 10 || !strings.Contains(token, ":") {
		return false
	}
	
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return false
	}
	
	// Check if the first part is a number
	_, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}
	
	// Check if the second part is of reasonable length (should be at least 30 chars for Telegram)
	if len(parts[1]) < 30 {
		return false
	}
	
	return true
}