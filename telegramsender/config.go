package telegramsender

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// API configuration
	BotToken            string
	BaseURL             string
	
	// HTTP client settings
	RequestTimeout      time.Duration
	KeepAlive           time.Duration
	MaxIdleConns        int
	IdleConnTimeout     time.Duration
	
	// Rate limiting
	RateLimitRequests   float64
	RateLimitBurst      int
	
	// Circuit breaker
	BreakerMaxRequests  uint32
	BreakerInterval     time.Duration
	BreakerTimeout      time.Duration
	
	// Retry settings
	MaxRetries          int
	RetryInitialBackoff time.Duration
	RetryMaxBackoff     time.Duration
	RetryBackoffFactor  float64
	
	// Logging
	LogFilePath         string
}

func LoadConfig() (*Config, error) {
	rateLimitRequests, err := strconv.ParseFloat(getEnv("RATE_LIMIT_REQUESTS", "10"), 64)
	if err != nil {
		return nil, err
	}

	rateLimitBurst, err := strconv.Atoi(getEnv("RATE_LIMIT_BURST", "20"))
	if err != nil {
		return nil, err
	}

	requestTimeout, err := time.ParseDuration(getEnv("REQUEST_TIMEOUT", "10s"))
	if err != nil {
		return nil, err
	}

	keepAlive, err := time.ParseDuration(getEnv("KEEP_ALIVE", "30s"))
	if err != nil {
		return nil, err
	}

	maxIdleConns, err := strconv.Atoi(getEnv("MAX_IDLE_CONNS", "10"))
	if err != nil {
		return nil, err
	}

	idleConnTimeout, err := time.ParseDuration(getEnv("IDLE_CONN_TIMEOUT", "90s"))
	if err != nil {
		return nil, err
	}

	breakerMaxRequests, err := strconv.ParseUint(getEnv("BREAKER_MAX_REQUESTS", "5"), 10, 32)
	if err != nil {
		return nil, err
	}

	breakerInterval, err := time.ParseDuration(getEnv("BREAKER_INTERVAL", "2m"))
	if err != nil {
		return nil, err
	}

	breakerTimeout, err := time.ParseDuration(getEnv("BREAKER_TIMEOUT", "60s"))
	if err != nil {
		return nil, err
	}

	maxRetries, err := strconv.Atoi(getEnv("MAX_RETRIES", "3"))
	if err != nil {
		return nil, err
	}

	retryInitialBackoff, err := time.ParseDuration(getEnv("RETRY_INITIAL_BACKOFF", "100ms"))
	if err != nil {
		return nil, err
	}

	retryMaxBackoff, err := time.ParseDuration(getEnv("RETRY_MAX_BACKOFF", "10s"))
	if err != nil {
		return nil, err
	}

	retryBackoffFactor, err := strconv.ParseFloat(getEnv("RETRY_BACKOFF_FACTOR", "2.0"), 64)
	if err != nil {
		return nil, err
	}

	return &Config{
		BotToken:            getEnv("BOT_TOKEN", ""),
		BaseURL:             getEnv("BASE_URL", "https://api.telegram.org"),
		
		RequestTimeout:      requestTimeout,
		KeepAlive:           keepAlive,
		MaxIdleConns:        maxIdleConns,
		IdleConnTimeout:     idleConnTimeout,
		
		RateLimitRequests:   rateLimitRequests,
		RateLimitBurst:      rateLimitBurst,
		
		BreakerMaxRequests:  uint32(breakerMaxRequests),
		BreakerInterval:     breakerInterval,
		BreakerTimeout:      breakerTimeout,
		
		MaxRetries:          maxRetries,
		RetryInitialBackoff: retryInitialBackoff,
		RetryMaxBackoff:     retryMaxBackoff,
		RetryBackoffFactor:  retryBackoffFactor,
		
		LogFilePath:         getEnv("LOG_FILE_PATH", "logs/telegramsender.log"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}