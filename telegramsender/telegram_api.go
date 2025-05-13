package telegramsender

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

/* ---------- types ---------- */

type TelegramAPI struct {
	logger     *slog.Logger
	config     *Config
	httpClient *http.Client
	limiter    *rate.Limiter
	breaker    *gobreaker.CircuitBreaker
}

type TelegramResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
	Description string          `json:"description,omitempty"`
	// RetryAfter is not part of the API response, but used internally
	// to pass the Retry-After header value for rate limit handling
	RetryAfter  time.Duration   `json:"-"`
}

type MessageRequest struct {
	ChatID                int64       `json:"chat_id"`
	Text                  string      `json:"text"`
	ParseMode             string      `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool        `json:"disable_web_page_preview,omitempty"`
	DisableNotification   bool        `json:"disable_notification,omitempty"`
	ReplyToMessageID      int         `json:"reply_to_message_id,omitempty"`
	ReplyMarkup           interface{} `json:"reply_markup,omitempty"`
}

type MessageResult struct {
	MessageID int `json:"message_id"`
}

/* ---------- constructor ---------- */

func NewTelegramAPI(logger *slog.Logger, config *Config) *TelegramAPI {
	// Configure transport for connection pooling
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: config.KeepAlive,
		}).DialContext,
		MaxIdleConns:        config.MaxIdleConns,
		IdleConnTimeout:     config.IdleConnTimeout,
		TLSHandshakeTimeout: 10 * time.Second,
		ForceAttemptHTTP2:   true,
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
			// Trip the circuit breaker when error rate exceeds 50%
			// and we have at least 5 requests
			return counts.Requests > 5 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("circuit breaker state changed",
				"name", name,
				"from", from.String(),
				"to", to.String())
		},
	}

	return &TelegramAPI{
		logger:     logger,
		config:     config,
		httpClient: httpClient,
		limiter:    rate.NewLimiter(rate.Limit(config.RateLimitRequests), config.RateLimitBurst),
		breaker:    gobreaker.NewCircuitBreaker(cbSettings),
	}
}

/* ---------- public methods ---------- */

// SendMessage sends a text message to the specified chat
func (t *TelegramAPI) SendMessage(ctx context.Context, request MessageRequest) (*MessageResult, error) {
	if err := ValidateConfig(t.config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	var result *MessageResult
	var err error
	var serverRetryDelay time.Duration

	// Apply retry with exponential backoff
	for attempt := 0; attempt <= t.config.MaxRetries; attempt++ {
		// Main request (first attempt or after backoff)
		result, err = t.sendMessageOnce(ctx, request)
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

		// Check for rate limit response with Retry-After header
		var telegramErr *TelegramResponse
		if errors.As(err, &telegramErr) && telegramErr.RetryAfter > 0 {
			serverRetryDelay = telegramErr.RetryAfter
			t.logger.Warn("received rate limit response",
				"retry_after", serverRetryDelay.String(),
				"attempt", attempt)
		} else {
			serverRetryDelay = 0
		}

		// Determine backoff time for next attempt
		var backoff time.Duration
		if serverRetryDelay > 0 {
			backoff = serverRetryDelay
		} else {
			backoff = t.calculateBackoff(attempt + 1)
		}

		t.logger.Info("retrying request",
			"attempt", attempt+1,
			"backoff", backoff.String(),
			"using_server_delay", serverRetryDelay > 0)

		// Wait for backoff period or context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}
	
	// If we've exhausted all retries, return the last error
	return nil, fmt.Errorf("max retries exceeded: %w", err)
}

/* ---------- private methods ---------- */

func (t *TelegramAPI) sendMessageOnce(ctx context.Context, request MessageRequest) (*MessageResult, error) {
	// Rate limit check
	if err := t.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Use circuit breaker
	resp, err := t.breaker.Execute(func() (interface{}, error) {
		return t.executeRequest(ctx, "sendMessage", request)
	})

	if err != nil {
		return nil, err
	}

	telegramResp := resp.(*TelegramResponse)
	if !telegramResp.OK {
		return nil, fmt.Errorf("telegram API error: %d %s", telegramResp.ErrorCode, telegramResp.Description)
	}

	var msgResult MessageResult
	if err := json.Unmarshal(telegramResp.Result, &msgResult); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return &msgResult, nil
}

func (t *TelegramAPI) executeRequest(ctx context.Context, method string, payload interface{}) (*TelegramResponse, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build the actual URL with the token
	url := fmt.Sprintf("%s/bot%s/%s", t.config.BaseURL, t.config.BotToken, method)
	
	// Create a redacted URL for logging that hides the token
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

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var telegramResp TelegramResponse
	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Handle non-OK responses as errors
	if !telegramResp.OK {
		// Add the Retry-After header if present (for rate limiting responses)
		retryAfter := resp.Header.Get("Retry-After")
		
		t.logger.Error("telegram API error",
			"method", method,
			"url", redactedURL,
			"status_code", resp.StatusCode,
			"error_code", telegramResp.ErrorCode,
			"description", telegramResp.Description,
			"retry_after", retryAfter)
		
		// If this is a rate limit error and has a Retry-After header,
		// attach it to the error to be used by retry logic
		if telegramResp.ErrorCode == 429 && retryAfter != "" {
			// Parse the Retry-After value (in seconds)
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				telegramResp.RetryAfter = time.Duration(seconds) * time.Second
			}
		}
	}

	return &telegramResp, nil
}

func (t *TelegramAPI) calculateBackoff(attempt int) time.Duration {
	backoff := t.config.RetryInitialBackoff * time.Duration(math.Pow(t.config.RetryBackoffFactor, float64(attempt-1)))
	if backoff > t.config.RetryMaxBackoff {
		backoff = t.config.RetryMaxBackoff
	}
	// Add jitter (±20%)
	jitter := time.Duration(float64(backoff) * (0.8 + 0.4*float64(attempt%2)))
	return jitter
}

func (t *TelegramAPI) isRetryable(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}

	// Handle specific Telegram error codes that are retryable
	// 429 - Too Many Requests
	// 500, 502, 503, 504 - Server errors
	if telegramErr := extractTelegramError(err); telegramErr != nil {
		code := telegramErr.ErrorCode
		return code == 429 || code >= 500 && code <= 504
	}

	return false
}

func extractTelegramError(err error) *TelegramResponse {
	// Check if the error message contains Telegram error information
	errMsg := err.Error()
	if strings.Contains(errMsg, "telegram API error") {
		// This is a best-effort extraction from our formatted error message
		if strings.Contains(errMsg, "403") {
			return &TelegramResponse{
				OK:          false,
				ErrorCode:   403,
				Description: "Forbidden",
			}
		} else if strings.Contains(errMsg, "429") {
			return &TelegramResponse{
				OK:          false,
				ErrorCode:   429,
				Description: "Too Many Requests",
			}
		} else if strings.Contains(errMsg, "500") || 
		          strings.Contains(errMsg, "502") || 
		          strings.Contains(errMsg, "503") || 
		          strings.Contains(errMsg, "504") {
			return &TelegramResponse{
				OK:          false,
				ErrorCode:   500,
				Description: "Server Error",
			}
		}
	}
	
	return nil
}