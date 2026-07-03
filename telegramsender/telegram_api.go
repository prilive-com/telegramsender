package telegramsender

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

/* ---------- types ---------- */

// TelegramAPI is the main client for sending messages via Telegram Bot API.
// It implements the Sender interface.
type TelegramAPI struct {
	logger       *Logger
	config       *Config
	httpClient   HTTPClient
	globalLimiter *rate.Limiter
	chatLimiters map[int64]*rate.Limiter
	limiterMu    sync.RWMutex
	breaker      *gobreaker.CircuitBreaker
}

// TelegramResponse represents a response from the Telegram API.
type TelegramResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
	Description string          `json:"description,omitempty"`
}

// MessageRequest represents a request to send a text message.
type MessageRequest struct {
	ChatID                int64       `json:"chat_id"`
	Text                  string      `json:"text"`
	ParseMode             string      `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool        `json:"disable_web_page_preview,omitempty"`
	DisableNotification   bool        `json:"disable_notification,omitempty"`
	ReplyToMessageID      int         `json:"reply_to_message_id,omitempty"`
	ReplyMarkup           interface{} `json:"reply_markup,omitempty"`
}

// MessageResult represents the result of a sent message.
type MessageResult struct {
	MessageID int `json:"message_id"`
}

// PhotoRequest represents a request to send a photo by URL or file_id.
type PhotoRequest struct {
	ChatID              int64       `json:"chat_id"`
	Photo               string      `json:"photo"`
	Caption             string      `json:"caption,omitempty"`
	ParseMode           string      `json:"parse_mode,omitempty"`
	DisableNotification bool        `json:"disable_notification,omitempty"`
	ReplyToMessageID    int         `json:"reply_to_message_id,omitempty"`
	ReplyMarkup         interface{} `json:"reply_markup,omitempty"`
}

// PhotoFileRequest represents a request to send a local photo file.
type PhotoFileRequest struct {
	ChatID              int64
	PhotoPath           string
	Caption             string
	ParseMode           string
	DisableNotification bool
	ReplyToMessageID    int
	ReplyMarkup         interface{}
}

/* ---------- constructor ---------- */

// NewTelegramAPI creates a new TelegramAPI client with the given configuration.
func NewTelegramAPI(logger *Logger, config *Config) *TelegramAPI {
	// Configure transport with TLS 1.2+ minimum
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   DefaultDialTimeout,
			KeepAlive: config.KeepAlive,
		}).DialContext,
		MaxIdleConns:        config.MaxIdleConns,
		IdleConnTimeout:     config.IdleConnTimeout,
		TLSHandshakeTimeout: DefaultTLSHandshakeTimeout,
		ForceAttemptHTTP2:   true,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	// Configure HTTP client
	httpClient := &http.Client{
		Timeout:   config.RequestTimeout,
		Transport: transport,
	}

	// Circuit breaker settings
	cbSettings := gobreaker.Settings{
		Name:        "TelegramAPICircuitBreaker",
		MaxRequests: config.BreakerMaxRequests,
		Interval:    config.BreakerInterval,
		Timeout:     config.BreakerTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.Requests > BreakerMinRequests &&
				float64(counts.TotalFailures)/float64(counts.Requests) >= BreakerErrorThreshold
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("circuit breaker state changed",
				"name", name,
				"from", from.String(),
				"to", to.String())
		},
	}

	return &TelegramAPI{
		logger:        logger,
		config:        config,
		httpClient:    httpClient,
		globalLimiter: rate.NewLimiter(rate.Limit(config.RateLimitRequests), config.RateLimitBurst),
		chatLimiters:  make(map[int64]*rate.Limiter),
		breaker:       gobreaker.NewCircuitBreaker(cbSettings),
	}
}

/* ---------- public methods ---------- */

// SendMessage sends a text message to the specified chat.
func (t *TelegramAPI) SendMessage(ctx context.Context, request MessageRequest) (*MessageResult, error) {
	if err := ValidateConfig(t.config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	if err := ValidateMessageRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	return t.withRetry(ctx, request.ChatID, func() (*MessageResult, error) {
		return t.sendMessageOnce(ctx, request)
	})
}

// SendPhoto sends a photo by URL or file_id to the specified chat.
func (t *TelegramAPI) SendPhoto(ctx context.Context, request PhotoRequest) (*MessageResult, error) {
	if err := ValidateConfig(t.config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	if err := ValidatePhotoRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	return t.withRetry(ctx, request.ChatID, func() (*MessageResult, error) {
		return t.sendPhotoOnce(ctx, request)
	})
}

// SendPhotoFile sends a local photo file to the specified chat.
func (t *TelegramAPI) SendPhotoFile(ctx context.Context, request PhotoFileRequest) (*MessageResult, error) {
	if err := ValidateConfig(t.config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	if err := ValidatePhotoFileRequest(request, t.config.MaxCaptionLength); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	// Validate photo path for security
	if err := ValidatePhotoPath(request.PhotoPath, t.config.AllowedPhotoDirs); err != nil {
		return nil, fmt.Errorf("photo path validation failed: %w", err)
	}

	// Check file exists and size
	fileInfo, err := os.Stat(request.PhotoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat image file: %w", err)
	}

	if fileInfo.Size() > int64(t.config.MaxFileSize) {
		return nil, NewValidationError("photo_path",
			fmt.Sprintf("file exceeds %d bytes limit: %d bytes", t.config.MaxFileSize, fileInfo.Size()))
	}

	return t.withRetry(ctx, request.ChatID, func() (*MessageResult, error) {
		return t.sendPhotoFileOnce(ctx, request)
	})
}

/* ---------- retry logic (extracted to eliminate duplication) ---------- */

// withRetry executes an operation with retry logic and exponential backoff.
func (t *TelegramAPI) withRetry(ctx context.Context, chatID int64, operation func() (*MessageResult, error)) (*MessageResult, error) {
	var result *MessageResult
	var err error

	for attempt := 0; attempt <= t.config.MaxRetries; attempt++ {
		result, err = operation()
		if err == nil {
			return result, nil
		}

		// Exit early if this is the last attempt
		if attempt == t.config.MaxRetries {
			break
		}

		// Check if the error is retryable
		if !t.isRetryable(err) {
			t.logger.Error("non-retryable error",
				"error", err,
				"attempt", attempt)
			return nil, err
		}

		// Determine backoff time
		backoff := t.determineBackoff(attempt+1, err)

		t.logger.Info("retrying request",
			"attempt", attempt+1,
			"backoff", backoff.String(),
			"chat_id", chatID)

		// Wait for backoff period or context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	return nil, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, err)
}

// determineBackoff calculates the backoff duration, respecting server-provided retry-after.
func (t *TelegramAPI) determineBackoff(attempt int, err error) time.Duration {
	// Check for Telegram error with RetryAfter
	var telegramErr *TelegramError
	if errors.As(err, &telegramErr) && telegramErr.RetryAfter > 0 {
		return telegramErr.RetryAfter
	}

	return t.calculateBackoff(attempt)
}

/* ---------- private methods ---------- */

func (t *TelegramAPI) sendMessageOnce(ctx context.Context, request MessageRequest) (*MessageResult, error) {
	// Per-chat rate limit
	if err := t.waitForChatRateLimit(ctx, request.ChatID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRateLimitExceeded, err)
	}

	// Global rate limit
	if err := t.globalLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRateLimitExceeded, err)
	}

	// Use circuit breaker
	resp, err := t.breaker.Execute(func() (interface{}, error) {
		return t.executeRequest(ctx, MethodSendMessage, request)
	})

	if err != nil {
		// Check if it's a circuit breaker error
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, fmt.Errorf("%w: %v", ErrCircuitBreakerOpen, err)
		}
		return nil, err
	}

	return t.parseMessageResult(resp)
}

func (t *TelegramAPI) sendPhotoOnce(ctx context.Context, request PhotoRequest) (*MessageResult, error) {
	// Per-chat rate limit
	if err := t.waitForChatRateLimit(ctx, request.ChatID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRateLimitExceeded, err)
	}

	// Global rate limit
	if err := t.globalLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRateLimitExceeded, err)
	}

	// Use circuit breaker
	resp, err := t.breaker.Execute(func() (interface{}, error) {
		return t.executeRequest(ctx, MethodSendPhoto, request)
	})

	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, fmt.Errorf("%w: %v", ErrCircuitBreakerOpen, err)
		}
		return nil, err
	}

	return t.parseMessageResult(resp)
}

func (t *TelegramAPI) sendPhotoFileOnce(ctx context.Context, request PhotoFileRequest) (*MessageResult, error) {
	// Per-chat rate limit
	if err := t.waitForChatRateLimit(ctx, request.ChatID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRateLimitExceeded, err)
	}

	// Global rate limit
	if err := t.globalLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRateLimitExceeded, err)
	}

	// Use circuit breaker
	resp, err := t.breaker.Execute(func() (interface{}, error) {
		return t.executeMultipartRequest(ctx, MethodSendPhoto, request)
	})

	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, fmt.Errorf("%w: %v", ErrCircuitBreakerOpen, err)
		}
		return nil, err
	}

	return t.parseMessageResult(resp)
}

// parseMessageResult safely parses the response into MessageResult.
func (t *TelegramAPI) parseMessageResult(resp interface{}) (*MessageResult, error) {
	telegramResp, ok := resp.(*TelegramResponse)
	if !ok {
		return nil, errors.New("unexpected response type from circuit breaker")
	}

	if !telegramResp.OK {
		return nil, NewTelegramError(telegramResp.ErrorCode, telegramResp.Description)
	}

	var msgResult MessageResult
	if err := json.Unmarshal(telegramResp.Result, &msgResult); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return &msgResult, nil
}

// waitForChatRateLimit waits for the per-chat rate limiter.
func (t *TelegramAPI) waitForChatRateLimit(ctx context.Context, chatID int64) error {
	limiter := t.getChatLimiter(chatID)
	return limiter.Wait(ctx)
}

// getChatLimiter returns or creates a rate limiter for a specific chat.
func (t *TelegramAPI) getChatLimiter(chatID int64) *rate.Limiter {
	t.limiterMu.RLock()
	limiter, exists := t.chatLimiters[chatID]
	t.limiterMu.RUnlock()

	if exists {
		return limiter
	}

	t.limiterMu.Lock()
	defer t.limiterMu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists = t.chatLimiters[chatID]; exists {
		return limiter
	}

	// Create new limiter for this chat
	limiter = rate.NewLimiter(rate.Limit(PerChatRateLimit), PerChatBurst)
	t.chatLimiters[chatID] = limiter
	return limiter
}

func (t *TelegramAPI) executeRequest(ctx context.Context, method string, payload interface{}) (*TelegramResponse, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/%s", t.config.BaseURL, t.config.BotToken, method)
	redactedURL := fmt.Sprintf("%s/bot[REDACTED]/%s", t.config.BaseURL, method)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %s: %w", redactedURL, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return t.parseResponse(resp, method, redactedURL)
}

func (t *TelegramAPI) executeMultipartRequest(ctx context.Context, method string, request PhotoFileRequest) (*TelegramResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	if err := writer.WriteField("chat_id", strconv.FormatInt(request.ChatID, 10)); err != nil {
		return nil, fmt.Errorf("failed to write chat_id: %w", err)
	}

	if request.Caption != "" {
		if err := writer.WriteField("caption", request.Caption); err != nil {
			return nil, fmt.Errorf("failed to write caption: %w", err)
		}
	}

	if request.ParseMode != "" {
		if err := writer.WriteField("parse_mode", request.ParseMode); err != nil {
			return nil, fmt.Errorf("failed to write parse_mode: %w", err)
		}
	}

	if request.DisableNotification {
		if err := writer.WriteField("disable_notification", "true"); err != nil {
			return nil, fmt.Errorf("failed to write disable_notification: %w", err)
		}
	}

	if request.ReplyToMessageID > 0 {
		if err := writer.WriteField("reply_to_message_id", strconv.Itoa(request.ReplyToMessageID)); err != nil {
			return nil, fmt.Errorf("failed to write reply_to_message_id: %w", err)
		}
	}

	// Open and copy file
	file, err := os.Open(request.PhotoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("photo", filepath.Base(request.PhotoPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	buf := make([]byte, FileReadBufferSize)
	if _, err := io.CopyBuffer(part, file, buf); err != nil {
		return nil, fmt.Errorf("failed to copy file contents: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/%s", t.config.BaseURL, t.config.BotToken, method)
	redactedURL := fmt.Sprintf("%s/bot[REDACTED]/%s", t.config.BaseURL, method)

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	return t.parseResponse(resp, method, redactedURL)
}

// parseResponse reads and parses the HTTP response with size limits.
func (t *TelegramAPI) parseResponse(resp *http.Response, method, redactedURL string) (*TelegramResponse, error) {
	// Limit response size to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, MaxResponseSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check if response was truncated
	if len(body) == MaxResponseSize {
		return nil, ErrResponseTooLarge
	}

	var telegramResp TelegramResponse
	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Handle error responses
	if !telegramResp.OK {
		retryAfter := resp.Header.Get("Retry-After")

		t.logger.Error("telegram API error",
			"method", method,
			"url", redactedURL,
			"status_code", resp.StatusCode,
			"error_code", telegramResp.ErrorCode,
			"description", telegramResp.Description,
			"retry_after", retryAfter)

		// Return typed error that can be used with errors.As
		if telegramResp.ErrorCode == 429 && retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				return nil, NewTelegramErrorWithRetry(
					telegramResp.ErrorCode,
					telegramResp.Description,
					time.Duration(seconds)*time.Second,
				)
			}
		}

		return nil, NewTelegramError(telegramResp.ErrorCode, telegramResp.Description)
	}

	return &telegramResp, nil
}

// calculateBackoff calculates exponential backoff with random jitter.
func (t *TelegramAPI) calculateBackoff(attempt int) time.Duration {
	backoff := t.config.RetryInitialBackoff * time.Duration(math.Pow(t.config.RetryBackoffFactor, float64(attempt-1)))
	if backoff > t.config.RetryMaxBackoff {
		backoff = t.config.RetryMaxBackoff
	}

	// Add random jitter (±20%) using crypto/rand for security
	jitterRange := int64(float64(backoff) * (JitterMax - JitterMin))
	if jitterRange > 0 {
		randomJitter, err := rand.Int(rand.Reader, big.NewInt(jitterRange))
		if err == nil {
			baseJitter := time.Duration(float64(backoff) * JitterMin)
			return baseJitter + time.Duration(randomJitter.Int64())
		}
	}

	return backoff
}

// isRetryable determines if an error is retryable.
func (t *TelegramAPI) isRetryable(err error) bool {
	// Context errors are not retryable
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}

	// Network timeouts are retryable
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Check for retryable Telegram errors
	var telegramErr *TelegramError
	if errors.As(err, &telegramErr) {
		return telegramErr.IsRetryable()
	}

	return false
}

/* ---------- health check ---------- */

// GetMeResponse represents the response from Telegram's getMe endpoint.
type GetMeResponse struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

// HealthCheck verifies connectivity to the Telegram API by calling the getMe endpoint.
// This is useful for health checks and readiness probes.
// Returns nil if the bot token is valid and the API is reachable.
func (t *TelegramAPI) HealthCheck(ctx context.Context) error {
	if err := ValidateConfig(t.config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/getMe", t.config.BaseURL, t.config.BotToken)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	// Parse response to verify it's a valid Telegram response
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return fmt.Errorf("failed to read health check response: %w", err)
	}

	var telegramResp TelegramResponse
	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return fmt.Errorf("failed to parse health check response: %w", err)
	}

	if !telegramResp.OK {
		return fmt.Errorf("health check failed: %s (code: %d)", telegramResp.Description, telegramResp.ErrorCode)
	}

	// Optionally parse the bot info (for logging/debugging)
	var botInfo GetMeResponse
	if err := json.Unmarshal(telegramResp.Result, &botInfo); err != nil {
		return fmt.Errorf("failed to parse bot info: %w", err)
	}

	t.logger.Debug("health check passed",
		"bot_id", botInfo.ID,
		"bot_username", botInfo.Username)

	return nil
}

// GetBotInfo returns information about the bot by calling the getMe endpoint.
// This is useful for verifying the bot configuration and getting the bot's username.
func (t *TelegramAPI) GetBotInfo(ctx context.Context) (*GetMeResponse, error) {
	if err := ValidateConfig(t.config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/getMe", t.config.BaseURL, t.config.BotToken)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var telegramResp TelegramResponse
	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !telegramResp.OK {
		return nil, NewTelegramError(telegramResp.ErrorCode, telegramResp.Description)
	}

	var botInfo GetMeResponse
	if err := json.Unmarshal(telegramResp.Result, &botInfo); err != nil {
		return nil, fmt.Errorf("failed to parse bot info: %w", err)
	}

	return &botInfo, nil
}
