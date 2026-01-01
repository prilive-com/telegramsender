package telegramsender

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

// Client is the main entry point for sending Telegram messages.
// Use New() or NewFromConfig() to create a Client.
type Client struct {
	config ClientConfig
	api    *TelegramAPI
	logger *Logger
}

// validate is the shared validator instance
var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())

	// Use json tags in error messages
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			return fld.Name
		}
		return name
	})

	// Register custom bot token validator
	validate.RegisterValidation("bottoken", validateBotTokenField)
}

// validateBotTokenField is a validator.Func for bot token format
func validateBotTokenField(fl validator.FieldLevel) bool {
	token := fl.Field().String()
	if token == "" {
		return true // Let 'required' handle empty
	}
	return ValidateBotToken(token) == nil
}

// New creates a new Client with the given bot token and options.
// This is the recommended way to create a Client programmatically.
//
// Example:
//
//	client, err := telegramsender.New(os.Getenv("TELEGRAM_BOT_TOKEN"),
//	    telegramsender.WithMaxRetriesOption(5),
//	    telegramsender.WithRateLimitOption(30, 50),
//	    telegramsender.WithLogger(logger),
//	)
func New(botToken string, opts ...Option) (*Client, error) {
	cfg := DefaultClientConfig()
	cfg.BotToken = botToken

	// Apply options
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	// Validate
	if err := validateClientConfig(&cfg); err != nil {
		return nil, err
	}

	return newClient(cfg)
}

// NewFromConfig creates a Client by loading configuration from multiple sources.
// Configuration precedence (highest to lowest):
//  1. Programmatic options (opts...)
//  2. Environment variables (TELEGRAM_*)
//  3. Config file (if path provided)
//  4. Default values
//
// Example:
//
//	client, err := telegramsender.NewFromConfig("config.yaml",
//	    telegramsender.WithLogger(logger),  // Override from config
//	)
func NewFromConfig(configPath string, opts ...Option) (*Client, error) {
	cfg, err := LoadClientConfig(configPath, opts...)
	if err != nil {
		return nil, err
	}
	return newClient(*cfg)
}

// LoadClientConfig loads configuration from file, env vars, and applies options.
func LoadClientConfig(configPath string, opts ...Option) (*ClientConfig, error) {
	k := koanf.New(".")

	// 1. DEFAULTS (lowest priority)
	if err := k.Load(structs.Provider(DefaultClientConfig(), "koanf"), nil); err != nil {
		return nil, fmt.Errorf("loading defaults: %w", err)
	}

	// 2. CONFIG FILE (if exists)
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
				return nil, fmt.Errorf("loading config file: %w", err)
			}
		}
	}

	// 3. ENVIRONMENT VARIABLES (TELEGRAM_*)
	if err := k.Load(env.Provider("TELEGRAM_", ".", func(s string) string {
		// TELEGRAM_BOT_TOKEN -> bot_token
		key := strings.ToLower(strings.TrimPrefix(s, "TELEGRAM_"))
		return strings.ReplaceAll(key, "_", ".")
	}), nil); err != nil {
		return nil, fmt.Errorf("loading env vars: %w", err)
	}

	// Unmarshal to struct
	var cfg ClientConfig
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	// 4. PROGRAMMATIC OPTIONS (highest priority)
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	// Validate
	if err := validateClientConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateClientConfig validates the configuration and returns user-friendly errors.
func validateClientConfig(cfg *ClientConfig) error {
	// Custom validation logic
	if cfg.BotToken == "" {
		return fmt.Errorf("bot_token: required (set via TELEGRAM_BOT_TOKEN env var)")
	}

	if err := ValidateBotToken(cfg.BotToken); err != nil {
		return fmt.Errorf("bot_token: %w (format: 123456789:ABCdefGHI...)", err)
	}

	if cfg.MaxRetries < 0 {
		return fmt.Errorf("max_retries: must be >= 0")
	}

	if cfg.RateLimitRequests <= 0 {
		return fmt.Errorf("rate_limit_requests: must be > 0")
	}

	if cfg.RateLimitBurst <= 0 {
		return fmt.Errorf("rate_limit_burst: must be > 0")
	}

	if cfg.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout: must be > 0")
	}

	return nil
}

// newClient creates the internal client from validated config.
func newClient(cfg ClientConfig) (*Client, error) {
	// Create logger
	var logger *Logger
	var err error

	if cfg.Logger != nil {
		logger = &Logger{Logger: cfg.Logger}
	} else {
		logger, err = NewLogger(0, cfg.LogFilePath)
		if err != nil {
			return nil, fmt.Errorf("creating logger: %w", err)
		}
	}

	// Convert ClientConfig to the internal Config format
	internalConfig := &Config{
		BotToken:            SecretToken(cfg.BotToken),
		BaseURL:             cfg.BaseURL,
		RequestTimeout:      cfg.RequestTimeout,
		KeepAlive:           cfg.KeepAlive,
		MaxIdleConns:        cfg.MaxIdleConns,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		RateLimitRequests:   cfg.RateLimitRequests,
		RateLimitBurst:      cfg.RateLimitBurst,
		BreakerMaxRequests:  cfg.BreakerMaxRequests,
		BreakerInterval:     cfg.BreakerInterval,
		BreakerTimeout:      cfg.BreakerTimeout,
		MaxRetries:          cfg.MaxRetries,
		RetryInitialBackoff: cfg.RetryInitialBackoff,
		RetryMaxBackoff:     cfg.RetryMaxBackoff,
		RetryBackoffFactor:  cfg.RetryBackoffFactor,
		MaxCaptionLength:    cfg.MaxCaptionLength,
		MaxFileSize:         cfg.MaxFileSize,
		AllowedPhotoDirs:    cfg.AllowedPhotoDirs,
		LogFilePath:         cfg.LogFilePath,
	}

	// Create the API client
	api := NewTelegramAPI(logger, internalConfig)

	return &Client{
		config: cfg,
		api:    api,
		logger: logger,
	}, nil
}

// Config returns a copy of the client configuration.
func (c *Client) Config() ClientConfig {
	return c.config
}

// SendMessage sends a text message to the specified chat.
func (c *Client) SendMessage(ctx context.Context, request MessageRequest) (*MessageResult, error) {
	return c.api.SendMessage(ctx, request)
}

// SendPhoto sends a photo by URL or file_id to the specified chat.
func (c *Client) SendPhoto(ctx context.Context, request PhotoRequest) (*MessageResult, error) {
	return c.api.SendPhoto(ctx, request)
}

// SendPhotoFile sends a local photo file to the specified chat.
func (c *Client) SendPhotoFile(ctx context.Context, request PhotoFileRequest) (*MessageResult, error) {
	return c.api.SendPhotoFile(ctx, request)
}

// Close releases resources held by the client.
func (c *Client) Close() error {
	if c.logger != nil {
		return c.logger.Close()
	}
	return nil
}

// API returns the underlying TelegramAPI for advanced usage.
// Prefer using the Client methods directly.
func (c *Client) API() *TelegramAPI {
	return c.api
}

// Compile-time check that Client implements Sender
var _ Sender = (*Client)(nil)
