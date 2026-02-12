package logger

import (
	"fmt"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// Logger provides structured logging with emoji categories
type Logger struct {
	level LogLevel
}

// New creates a new logger with the specified level
func New(level string) *Logger {
	var l LogLevel
	switch level {
	case "DEBUG":
		l = DEBUG
	case "INFO":
		l = INFO
	case "WARN":
		l = WARN
	case "ERROR":
		l = ERROR
	default:
		l = INFO
	}
	return &Logger{level: l}
}

// timestamp returns current timestamp in a readable format
func timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// Info logs an informational message with category emoji
func (l *Logger) Info(category, format string, args ...interface{}) {
	if l.level <= INFO {
		message := fmt.Sprintf(format, args...)
		fmt.Printf("[%s] [INFO] %s %s\n", timestamp(), category, message)
	}
}

// Warn logs a warning message with category emoji
func (l *Logger) Warn(category, format string, args ...interface{}) {
	if l.level <= WARN {
		message := fmt.Sprintf(format, args...)
		fmt.Printf("[%s] [WARN] %s %s\n", timestamp(), category, message)
	}
}

// Error logs an error message with category emoji
func (l *Logger) Error(category, format string, args ...interface{}) {
	if l.level <= ERROR {
		message := fmt.Sprintf(format, args...)
		fmt.Printf("[%s] [ERROR] %s %s\n", timestamp(), category, message)
	}
}

// Debug logs a debug message (only shown at DEBUG level)
func (l *Logger) Debug(category, format string, args ...interface{}) {
	if l.level <= DEBUG {
		message := fmt.Sprintf(format, args...)
		fmt.Printf("[%s] [DEBUG] %s %s\n", timestamp(), category, message)
	}
}

// Standard category emojis for consistency
const (
	CategoryStartup    = "🚀"
	CategoryDatabase   = "🔌"
	CategoryBeacon     = "📥"
	CategoryCommand    = "📤"
	CategoryExecution  = "📝"
	CategoryBackground = "🔍"
	CategoryStorage    = "💾"
	CategorySync       = "🔄"
	CategoryCleanup    = "🧹"
	CategoryAPI        = "🌐"
	CategoryWarning    = "⚠️"
	CategoryError      = "❌"
	CategorySuccess    = "✓"
)
