package telegramsender

import (
	"log/slog"
	"time"
)

// Option configures a Client. Use With* functions to create options.
// This interface-based approach prevents misuse and enables type safety.
type Option interface {
	apply(*ClientConfig)
}

// optionFunc wraps a function to implement Option interface.
type optionFunc func(*ClientConfig)

func (f optionFunc) apply(c *ClientConfig) { f(c) }

// ClientConfig holds all configuration for a Client.
// Use DefaultClientConfig() to get sensible defaults.
type ClientConfig struct {
	// Required
	BotToken string

	// API endpoint
	BaseURL string

	// HTTP client settings
	RequestTimeout  time.Duration
	KeepAlive       time.Duration
	MaxIdleConns    int
	IdleConnTimeout time.Duration

	// Rate limiting
	RateLimitRequests float64
	RateLimitBurst    int

	// Circuit breaker
	BreakerMaxRequests uint32
	BreakerInterval    time.Duration
	BreakerTimeout     time.Duration

	// Retry settings (exponential backoff)
	MaxRetries          int
	RetryInitialBackoff time.Duration
	RetryMaxBackoff     time.Duration
	RetryBackoffFactor  float64

	// Content limits
	MaxCaptionLength int
	MaxFileSize      int

	// Security settings
	AllowedPhotoDirs []string

	// Logging
	LogFilePath string
	Logger      *slog.Logger

	// Custom HTTP client (for testing)
	HTTPClient HTTPClient
}

// DefaultClientConfig returns a ClientConfig with sensible defaults.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		BaseURL:             DefaultBaseURL,
		RequestTimeout:      DefaultRequestTimeout,
		KeepAlive:           DefaultKeepAlive,
		MaxIdleConns:        DefaultMaxIdleConns,
		IdleConnTimeout:     DefaultIdleConnTimeout,
		RateLimitRequests:   DefaultRateLimitRequests,
		RateLimitBurst:      DefaultRateLimitBurst,
		BreakerMaxRequests:  DefaultBreakerMaxRequests,
		BreakerInterval:     DefaultBreakerInterval,
		BreakerTimeout:      DefaultBreakerTimeout,
		MaxRetries:          DefaultMaxRetries,
		RetryInitialBackoff: DefaultRetryInitialBackoff,
		RetryMaxBackoff:     DefaultRetryMaxBackoff,
		RetryBackoffFactor:  DefaultRetryBackoffFactor,
		MaxCaptionLength:    MaxCaptionLengthDefault,
		MaxFileSize:         MaxFileSizeDefault,
		LogFilePath:         "logs/telegramsender.log",
	}
}

// WithBaseURL sets a custom base URL for the Telegram API.
func WithBaseURLOption(url string) Option {
	return optionFunc(func(c *ClientConfig) { c.BaseURL = url })
}

// WithRequestTimeout sets the HTTP request timeout.
func WithRequestTimeoutOption(d time.Duration) Option {
	return optionFunc(func(c *ClientConfig) { c.RequestTimeout = d })
}

// WithKeepAlive sets the HTTP keep-alive duration.
func WithKeepAliveOption(d time.Duration) Option {
	return optionFunc(func(c *ClientConfig) { c.KeepAlive = d })
}

// WithConnectionPool configures connection pool settings.
func WithConnectionPool(maxIdleConns int, idleConnTimeout time.Duration) Option {
	return optionFunc(func(c *ClientConfig) {
		c.MaxIdleConns = maxIdleConns
		c.IdleConnTimeout = idleConnTimeout
	})
}

// WithRateLimitOption sets rate limiting parameters.
func WithRateLimitOption(requestsPerSec float64, burst int) Option {
	return optionFunc(func(c *ClientConfig) {
		c.RateLimitRequests = requestsPerSec
		c.RateLimitBurst = burst
	})
}

// WithBreakerConfig configures the circuit breaker.
func WithBreakerConfig(maxRequests uint32, interval, timeout time.Duration) Option {
	return optionFunc(func(c *ClientConfig) {
		c.BreakerMaxRequests = maxRequests
		c.BreakerInterval = interval
		c.BreakerTimeout = timeout
	})
}

// WithRetryOption configures retry settings with exponential backoff.
func WithRetryOption(maxRetries int, initialBackoff, maxBackoff time.Duration, factor float64) Option {
	return optionFunc(func(c *ClientConfig) {
		c.MaxRetries = maxRetries
		c.RetryInitialBackoff = initialBackoff
		c.RetryMaxBackoff = maxBackoff
		c.RetryBackoffFactor = factor
	})
}

// WithMaxRetriesOption sets the maximum number of retries.
func WithMaxRetriesOption(n int) Option {
	return optionFunc(func(c *ClientConfig) { c.MaxRetries = n })
}

// WithContentLimits sets content length limits.
func WithContentLimitsOption(maxCaptionLength, maxFileSize int) Option {
	return optionFunc(func(c *ClientConfig) {
		c.MaxCaptionLength = maxCaptionLength
		c.MaxFileSize = maxFileSize
	})
}

// WithAllowedPhotoDirsOption sets the allowed directories for photo uploads.
func WithAllowedPhotoDirsOption(dirs ...string) Option {
	return optionFunc(func(c *ClientConfig) { c.AllowedPhotoDirs = dirs })
}

// WithLogger sets a custom slog.Logger.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(c *ClientConfig) { c.Logger = logger })
}

// WithLogFile sets the log file path.
func WithLogFile(path string) Option {
	return optionFunc(func(c *ClientConfig) { c.LogFilePath = path })
}

// WithHTTPClientOption sets a custom HTTP client (useful for testing).
func WithHTTPClientOption(client HTTPClient) Option {
	return optionFunc(func(c *ClientConfig) { c.HTTPClient = client })
}

// Presets for common configurations

// ProductionPreset returns options suitable for production environments.
func ProductionPreset() Option {
	return optionFunc(func(c *ClientConfig) {
		c.MaxRetries = 5
		c.RetryInitialBackoff = 200 * time.Millisecond
		c.RetryMaxBackoff = 30 * time.Second
		c.BreakerMaxRequests = 5
		c.BreakerTimeout = 60 * time.Second
		c.RateLimitRequests = 30
		c.RateLimitBurst = 50
	})
}

// DevelopmentPreset returns options suitable for development.
func DevelopmentPreset() Option {
	return optionFunc(func(c *ClientConfig) {
		c.MaxRetries = 2
		c.RetryInitialBackoff = 50 * time.Millisecond
		c.RetryMaxBackoff = 2 * time.Second
		c.BreakerMaxRequests = 2
		c.BreakerTimeout = 10 * time.Second
		c.RateLimitRequests = 10
		c.RateLimitBurst = 20
	})
}

// HighThroughputPreset returns options for high-throughput scenarios.
func HighThroughputPreset() Option {
	return optionFunc(func(c *ClientConfig) {
		c.MaxIdleConns = 50
		c.IdleConnTimeout = 120 * time.Second
		c.RateLimitRequests = 50
		c.RateLimitBurst = 100
		c.RequestTimeout = 30 * time.Second
	})
}
