package telegramsender

import (
	"io"
	"log/slog"
	"os"
)

// Logger wraps slog.Logger and manages the underlying file handle for proper cleanup.
type Logger struct {
	*slog.Logger
	file       *os.File
	isExternal bool // true if using external logger (don't close anything)
}

// Close releases the log file handle. Safe to call multiple times or on nil file.
// If the logger was created from an external logger, this is a no-op.
func (l *Logger) Close() error {
	if l.isExternal {
		return nil
	}
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// NewLoggerFromExternal wraps an existing slog.Logger for use with TelegramAPI.
// This allows integrating with existing application logging infrastructure.
// Close() is a no-op when using an external logger.
func NewLoggerFromExternal(logger *slog.Logger) *Logger {
	return &Logger{
		Logger:     logger,
		file:       nil,
		isExternal: true,
	}
}

// NewLogger creates a production-ready structured logger using Go's built-in log/slog.
// Logs are output in JSON format to stdout and optionally to a log file.
// The caller MUST call Logger.Close() when done to release the file handle.
func NewLogger(logLevel slog.Level, logFilePath string) (*Logger, error) {
	var logOutput io.Writer = os.Stdout
	var logFile *os.File

	if logFilePath != "" {
		// Ensure the directory exists
		if err := ensureLogPath(logFilePath); err != nil {
			return nil, err
		}

		var err error
		logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, err
		}
		logOutput = io.MultiWriter(os.Stdout, logFile)
	}

	handler := slog.NewJSONHandler(logOutput, &slog.HandlerOptions{
		Level: logLevel,
	})

	return &Logger{
		Logger:     slog.New(handler),
		file:       logFile,
		isExternal: false,
	}, nil
}

