package logger

import (
	"fmt"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	level LogLevel
}

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

func timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func (l *Logger) Info(category, format string, args ...interface{}) {
	if l.level <= INFO {
		message := fmt.Sprintf(format, args...)
		fmt.Printf("[%s] [INFO] %s %s\n", timestamp(), category, message)
	}
}

func (l *Logger) Warn(category, format string, args ...interface{}) {
	if l.level <= WARN {
		message := fmt.Sprintf(format, args...)
		fmt.Printf("[%s] [WARN] %s %s\n", timestamp(), category, message)
	}
}

func (l *Logger) Error(category, format string, args ...interface{}) {
	if l.level <= ERROR {
		message := fmt.Sprintf(format, args...)
		fmt.Printf("[%s] [ERROR] %s %s\n", timestamp(), category, message)
	}
}

func (l *Logger) Debug(category, format string, args ...interface{}) {
	if l.level <= DEBUG {
		message := fmt.Sprintf(format, args...)
		fmt.Printf("[%s] [DEBUG] %s %s\n", timestamp(), category, message)
	}
}

// Standard category emojis for consistency
const (
	CategoryStartup    = "[StartUp]"
	CategoryDatabase   = "[DB]"
	CategoryBeacon     = "[Beacon]"
	CategoryCommand    = "[CMD]"
	CategoryExecution  = "[Execution]"
	CategoryBackground = "[Background]"
	CategoryStorage    = "[Storage]"
	CategorySync       = "[Sync]"
	CategoryCleanup    = "[Cleanup]"
	CategoryAPI        = "[API]"
	CategoryWarning    = "⚠️"
	CategoryError      = "❌"
	CategorySuccess    = "✓"
	CategoryWebSocket  = "WS"
)
