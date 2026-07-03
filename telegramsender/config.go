package telegramsender

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the Telegram API client.
type Config struct {
	// API configuration
	BotToken string
	BaseURL  string

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

	// Retry settings
	MaxRetries          int
	RetryInitialBackoff time.Duration
	RetryMaxBackoff     time.Duration
	RetryBackoffFactor  float64

	// Content limits
	MaxCaptionLength int // Maximum caption length in UTF-8 characters (default: 1024)
	MaxFileSize      int // Maximum file size in bytes (default: 10MB)

	// Security settings
	AllowedPhotoDirs []string // Allowed directories for photo uploads (empty = no restriction)

	// Logging
	LogFilePath    string       // File path for logging (ignored if ExternalLogger is set)
	ExternalLogger *slog.Logger // Optional external logger (takes precedence over LogFilePath)
}

// ConfigOption is a functional option for configuring Config.
type ConfigOption func(*Config)

// NewConfig creates a new Config with the given bot token and options.
// Use this for programmatic configuration instead of environment variables.
func NewConfig(botToken string, opts ...ConfigOption) *Config {
	cfg := &Config{
		BotToken:            botToken,
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

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithBaseURL sets a custom base URL (must be in allowed hosts list).
func WithBaseURL(url string) ConfigOption {
	return func(c *Config) { c.BaseURL = url }
}

// WithRequestTimeout sets the HTTP request timeout.
func WithRequestTimeout(d time.Duration) ConfigOption {
	return func(c *Config) { c.RequestTimeout = d }
}

// WithKeepAlive sets the HTTP keep-alive duration.
func WithKeepAlive(d time.Duration) ConfigOption {
	return func(c *Config) { c.KeepAlive = d }
}

// WithMaxIdleConns sets the maximum idle connections.
func WithMaxIdleConns(n int) ConfigOption {
	return func(c *Config) { c.MaxIdleConns = n }
}

// WithIdleConnTimeout sets the idle connection timeout.
func WithIdleConnTimeout(d time.Duration) ConfigOption {
	return func(c *Config) { c.IdleConnTimeout = d }
}

// WithRateLimit sets the rate limit parameters.
func WithRateLimit(requestsPerSec float64, burst int) ConfigOption {
	return func(c *Config) {
		c.RateLimitRequests = requestsPerSec
		c.RateLimitBurst = burst
	}
}

// WithCircuitBreaker sets circuit breaker parameters.
func WithCircuitBreaker(maxRequests uint32, interval, timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.BreakerMaxRequests = maxRequests
		c.BreakerInterval = interval
		c.BreakerTimeout = timeout
	}
}

// WithRetry sets retry parameters.
func WithRetry(maxRetries int, initialBackoff, maxBackoff time.Duration, factor float64) ConfigOption {
	return func(c *Config) {
		c.MaxRetries = maxRetries
		c.RetryInitialBackoff = initialBackoff
		c.RetryMaxBackoff = maxBackoff
		c.RetryBackoffFactor = factor
	}
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(n int) ConfigOption {
	return func(c *Config) { c.MaxRetries = n }
}

// WithContentLimits sets content length limits.
func WithContentLimits(maxCaptionLength, maxFileSize int) ConfigOption {
	return func(c *Config) {
		c.MaxCaptionLength = maxCaptionLength
		c.MaxFileSize = maxFileSize
	}
}

// WithAllowedPhotoDirs sets the allowed directories for photo uploads.
func WithAllowedPhotoDirs(dirs ...string) ConfigOption {
	return func(c *Config) { c.AllowedPhotoDirs = dirs }
}

// WithLogFilePath sets the log file path.
func WithLogFilePath(path string) ConfigOption {
	return func(c *Config) { c.LogFilePath = path }
}

// WithExternalLogger sets an external slog.Logger to use instead of creating a file-based logger.
// When an external logger is provided, LogFilePath is ignored.
// This is useful for integrating with existing application logging infrastructure.
func WithExternalLogger(logger *slog.Logger) ConfigOption {
	return func(c *Config) { c.ExternalLogger = logger }
}

// LoadConfig loads configuration from environment variables.
// For programmatic configuration, use NewConfig instead.
func LoadConfig() (*Config, error) {
	rateLimitRequests, err := parseEnvFloat("RATE_LIMIT_REQUESTS", DefaultRateLimitRequests)
	if err != nil {
		return nil, err
	}

	rateLimitBurst, err := parseEnvInt("RATE_LIMIT_BURST", DefaultRateLimitBurst)
	if err != nil {
		return nil, err
	}

	requestTimeout, err := parseEnvDuration("REQUEST_TIMEOUT", DefaultRequestTimeout)
	if err != nil {
		return nil, err
	}

	keepAlive, err := parseEnvDuration("KEEP_ALIVE", DefaultKeepAlive)
	if err != nil {
		return nil, err
	}

	maxIdleConns, err := parseEnvInt("MAX_IDLE_CONNS", DefaultMaxIdleConns)
	if err != nil {
		return nil, err
	}

	idleConnTimeout, err := parseEnvDuration("IDLE_CONN_TIMEOUT", DefaultIdleConnTimeout)
	if err != nil {
		return nil, err
	}

	breakerMaxRequests, err := parseEnvUint32("BREAKER_MAX_REQUESTS", DefaultBreakerMaxRequests)
	if err != nil {
		return nil, err
	}

	breakerInterval, err := parseEnvDuration("BREAKER_INTERVAL", DefaultBreakerInterval)
	if err != nil {
		return nil, err
	}

	breakerTimeout, err := parseEnvDuration("BREAKER_TIMEOUT", DefaultBreakerTimeout)
	if err != nil {
		return nil, err
	}

	maxRetries, err := parseEnvInt("MAX_RETRIES", DefaultMaxRetries)
	if err != nil {
		return nil, err
	}

	retryInitialBackoff, err := parseEnvDuration("RETRY_INITIAL_BACKOFF", DefaultRetryInitialBackoff)
	if err != nil {
		return nil, err
	}

	retryMaxBackoff, err := parseEnvDuration("RETRY_MAX_BACKOFF", DefaultRetryMaxBackoff)
	if err != nil {
		return nil, err
	}

	retryBackoffFactor, err := parseEnvFloat("RETRY_BACKOFF_FACTOR", DefaultRetryBackoffFactor)
	if err != nil {
		return nil, err
	}

	maxCaptionLength, err := parseEnvInt("MAX_CAPTION_LENGTH", MaxCaptionLengthDefault)
	if err != nil {
		return nil, err
	}

	maxFileSize, err := parseEnvInt("MAX_FILE_SIZE", MaxFileSizeDefault)
	if err != nil {
		return nil, err
	}

	return &Config{
		BotToken: getEnv("BOT_TOKEN", ""),
		BaseURL:  getEnv("BASE_URL", DefaultBaseURL),

		RequestTimeout:  requestTimeout,
		KeepAlive:       keepAlive,
		MaxIdleConns:    maxIdleConns,
		IdleConnTimeout: idleConnTimeout,

		RateLimitRequests: rateLimitRequests,
		RateLimitBurst:    rateLimitBurst,

		BreakerMaxRequests: breakerMaxRequests,
		BreakerInterval:    breakerInterval,
		BreakerTimeout:     breakerTimeout,

		MaxRetries:          maxRetries,
		RetryInitialBackoff: retryInitialBackoff,
		RetryMaxBackoff:     retryMaxBackoff,
		RetryBackoffFactor:  retryBackoffFactor,

		MaxCaptionLength: maxCaptionLength,
		MaxFileSize:      maxFileSize,

		LogFilePath: getEnv("LOG_FILE_PATH", "logs/telegramsender.log"),
	}, nil
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func parseEnvInt(key string, defaultValue int) (int, error) {
	str := getEnv(key, "")
	if str == "" {
		return defaultValue, nil
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return val, nil
}

func parseEnvUint32(key string, defaultValue uint32) (uint32, error) {
	str := getEnv(key, "")
	if str == "" {
		return defaultValue, nil
	}
	val, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return uint32(val), nil
}

func parseEnvFloat(key string, defaultValue float64) (float64, error) {
	str := getEnv(key, "")
	if str == "" {
		return defaultValue, nil
	}
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return val, nil
}

func parseEnvDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	str := getEnv(key, "")
	if str == "" {
		return defaultValue, nil
	}
	val, err := time.ParseDuration(str)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return val, nil
}

