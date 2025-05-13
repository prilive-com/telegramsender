package telegramsender

import (
	"io"
	"log/slog"
	"os"
)

// NewLogger creates a production-ready structured logger using Go's built-in log/slog.
// Logs are output in JSON format to stdout and optionally to a log file.
func NewLogger(logLevel slog.Level, logFilePath string) (*slog.Logger, error) {
	var logOutput io.Writer = os.Stdout

	if logFilePath != "" {
		// Ensure the directory exists
		if err := ensureLogPath(logFilePath); err != nil {
			return nil, err
		}

		// Open log file with more restrictive permissions (0600)
		logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, err
		}
		// Ensure the file is closed if there's an error later in this function
		defer func() {
			if err != nil {
				logFile.Close()
			}
		}()
		logOutput = io.MultiWriter(os.Stdout, logFile)
	}

	handler := slog.NewJSONHandler(logOutput, &slog.HandlerOptions{
		Level: logLevel,
	})

	logger := slog.New(handler)
	return logger, nil
}