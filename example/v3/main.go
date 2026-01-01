package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/prilive-com/telegramsender/v2/telegramsender"
)

// This example demonstrates the new v3 API with simplified configuration.
// Run with: go run main.go
//
// Required: Set BOT_TOKEN in .env file or environment

func main() {
	// Load .env file
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable required")
	}

	// Option 1: Simple programmatic configuration
	client, err := telegramsender.New(token,
		telegramsender.WithMaxRetriesOption(5),
		telegramsender.WithRateLimitOption(30, 50),
		telegramsender.WithRetryOption(5, 200*time.Millisecond, 30*time.Second, 2.0),
		telegramsender.ProductionPreset(),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Option 2: Load from config file + env vars + programmatic overrides
	// client, err := telegramsender.NewFromConfig("config.yaml",
	//     telegramsender.WithLogger(customLogger),
	// )

	fmt.Println("Telegram sender client created successfully!")
	fmt.Printf("Config: MaxRetries=%d, RateLimit=%.1f/s\n",
		client.Config().MaxRetries,
		client.Config().RateLimitRequests,
	)

	// Example: Send a message (uncomment and set chat ID to test)
	// ctx := context.Background()
	// chatID := int64(123456789) // Replace with your chat ID
	// result, err := client.SendMessage(ctx, telegramsender.MessageRequest{
	//     ChatID: chatID,
	//     Text:   "Hello from v3 API!",
	// })
	// if err != nil {
	//     log.Fatalf("Failed to send message: %v", err)
	// }
	// fmt.Printf("Message sent! ID: %d\n", result.MessageID)

	_ = context.Background() // Suppress unused variable warning
}
