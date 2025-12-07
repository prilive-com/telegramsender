package telegramsender

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ensureLogPath creates all parent directories for the log file with secure permissions.
func ensureLogPath(path string) error {
	dir := filepath.Dir(path)
	// Use 0700 for directories - owner only access
	return os.MkdirAll(dir, 0o700)
}

// ValidateConfig performs pre-flight sanity checks on configuration.
func ValidateConfig(cfg *Config) error {
	if cfg.BotToken == "" {
		return NewConfigError("BOT_TOKEN", "must be set")
	}
	if !validateBotToken(cfg.BotToken) {
		return NewConfigError("BOT_TOKEN", "format is invalid")
	}
	if cfg.LogFilePath == "" {
		return NewConfigError("LOG_FILE_PATH", "must be set")
	}
	if err := validateLogPath(cfg.LogFilePath); err != nil {
		return err
	}
	if cfg.BaseURL == "" {
		return NewConfigError("BASE_URL", "must be set")
	}
	if err := ValidateBaseURL(cfg.BaseURL); err != nil {
		return err
	}
	if cfg.RequestTimeout <= 0 {
		return NewConfigError("REQUEST_TIMEOUT", "must be positive")
	}
	if cfg.RetryInitialBackoff <= 0 {
		return NewConfigError("RETRY_INITIAL_BACKOFF", "must be positive")
	}
	if cfg.RetryMaxBackoff <= 0 {
		return NewConfigError("RETRY_MAX_BACKOFF", "must be positive")
	}
	if cfg.RetryBackoffFactor <= 0 {
		return NewConfigError("RETRY_BACKOFF_FACTOR", "must be positive")
	}
	if cfg.MaxRetries < 0 {
		return NewConfigError("MAX_RETRIES", "must be non-negative")
	}
	if cfg.RateLimitRequests <= 0 {
		return NewConfigError("RATE_LIMIT_REQUESTS", "must be positive")
	}
	if cfg.RateLimitBurst <= 0 {
		return NewConfigError("RATE_LIMIT_BURST", "must be positive")
	}
	return nil
}

// validateBotToken checks if the token has the correct format.
// Telegram bot tokens follow the pattern: 123456789:ABCDefGhIJKlmNoPQRsTUVwxyZ
func validateBotToken(token string) bool {
	if len(token) < 10 || !strings.Contains(token, ":") {
		return false
	}

	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return false
	}

	// First part must be a number (bot ID)
	_, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}

	// Second part should be at least 30 chars
	if len(parts[1]) < 30 {
		return false
	}

	return true
}

// ValidateBaseURL validates that the base URL is a valid HTTPS URL pointing to allowed hosts.
func ValidateBaseURL(baseURL string) error {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return NewConfigError("BASE_URL", fmt.Sprintf("invalid URL: %v", err))
	}

	if parsed.Scheme != "https" {
		return NewConfigError("BASE_URL", "must use HTTPS")
	}

	// Check against whitelist of allowed hosts
	allowed := false
	for _, host := range AllowedBaseURLHosts {
		if parsed.Host == host {
			allowed = true
			break
		}
	}

	if !allowed {
		return NewConfigError("BASE_URL", fmt.Sprintf("host %q not in allowed list", parsed.Host))
	}

	return nil
}

// validateLogPath ensures the log path is safe and doesn't allow path traversal.
func validateLogPath(path string) error {
	// Clean the path to resolve any .. or . components
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return NewConfigError("LOG_FILE_PATH", "path traversal not allowed")
	}

	// Ensure the path doesn't point to sensitive system directories
	sensitiveRoots := []string{"/etc", "/bin", "/sbin", "/usr", "/var/log", "/root", "/home"}
	for _, root := range sensitiveRoots {
		if strings.HasPrefix(cleanPath, root+"/") || cleanPath == root {
			return NewConfigError("LOG_FILE_PATH", fmt.Sprintf("cannot write to system directory %s", root))
		}
	}

	return nil
}

// ValidatePhotoPath validates that the photo path is safe and exists.
// It prevents path traversal attacks and symlink attacks.
func ValidatePhotoPath(path string, allowedDirs []string) error {
	// Check for path traversal BEFORE cleaning (to catch ../../../etc/passwd attempts)
	if strings.Contains(path, "..") {
		return fmt.Errorf("%w: path contains '..'", ErrPathTraversal)
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Double check after cleaning
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("%w: path contains '..'", ErrPathTraversal)
	}

	// Ensure path is absolute
	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("%w: path must be absolute", ErrPathTraversal)
	}

	// Check if path is within allowed directories (if specified)
	if len(allowedDirs) > 0 {
		allowed := false
		for _, dir := range allowedDirs {
			cleanDir := filepath.Clean(dir)
			if strings.HasPrefix(cleanPath, cleanDir+string(filepath.Separator)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("%w: path not in allowed directories", ErrPathTraversal)
		}
	}

	// Check for symlink attacks
	fileInfo, err := os.Lstat(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: symlinks not allowed", ErrPathTraversal)
	}

	return nil
}

// ValidateMessageRequest validates a MessageRequest before sending.
func ValidateMessageRequest(req MessageRequest) error {
	if req.ChatID == 0 {
		return NewValidationError("chat_id", "cannot be zero")
	}
	if req.Text == "" {
		return NewValidationError("text", "cannot be empty")
	}

	// Validate message length (4096 UTF-8 characters max)
	textLen := utf8.RuneCountInString(req.Text)
	if textLen > MaxMessageLength {
		return NewValidationError("text", fmt.Sprintf("exceeds %d character limit: %d chars", MaxMessageLength, textLen))
	}

	// Validate parse mode if set
	if req.ParseMode != "" && req.ParseMode != "HTML" && req.ParseMode != "Markdown" && req.ParseMode != "MarkdownV2" {
		return NewValidationError("parse_mode", "must be HTML, Markdown, or MarkdownV2")
	}

	return nil
}

// ValidatePhotoRequest validates a PhotoRequest before sending.
func ValidatePhotoRequest(req PhotoRequest) error {
	if req.ChatID == 0 {
		return NewValidationError("chat_id", "cannot be zero")
	}
	if req.Photo == "" {
		return NewValidationError("photo", "cannot be empty")
	}

	// Validate parse mode if set
	if req.ParseMode != "" && req.ParseMode != "HTML" && req.ParseMode != "Markdown" && req.ParseMode != "MarkdownV2" {
		return NewValidationError("parse_mode", "must be HTML, Markdown, or MarkdownV2")
	}

	return nil
}

// ValidatePhotoFileRequest validates a PhotoFileRequest before sending.
func ValidatePhotoFileRequest(req PhotoFileRequest, maxCaptionLength int) error {
	if req.ChatID == 0 {
		return NewValidationError("chat_id", "cannot be zero")
	}
	if req.PhotoPath == "" {
		return NewValidationError("photo_path", "cannot be empty")
	}

	// Validate caption length if set
	if req.Caption != "" {
		captionLen := utf8.RuneCountInString(req.Caption)
		if captionLen > maxCaptionLength {
			return NewValidationError("caption", fmt.Sprintf("exceeds %d character limit: %d chars", maxCaptionLength, captionLen))
		}
	}

	// Validate parse mode if set
	if req.ParseMode != "" && req.ParseMode != "HTML" && req.ParseMode != "Markdown" && req.ParseMode != "MarkdownV2" {
		return NewValidationError("parse_mode", "must be HTML, Markdown, or MarkdownV2")
	}

	return nil
}

