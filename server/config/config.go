package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	Database DatabaseConfig `json:"database"`
	Server   ServerConfig   `json:"server"`
	Features FeatureConfig  `json:"features"`
	Logging  LoggingConfig  `json:"logging"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"sslmode"`
	MaxConns int    `json:"max_conns"`
	MinConns int    `json:"min_conns"`
}

// ServerConfig holds server settings
type ServerConfig struct {
	Port string `json:"port"`
	Host string `json:"host"`
}

// FeatureConfig holds feature flags and settings
type FeatureConfig struct {
	AgentInactiveThresholdMinutes int  `json:"agent_inactive_threshold_minutes"`
	RetentionDays                 int  `json:"retention_days"`
	EnableAutoCleanup             bool `json:"enable_auto_cleanup"`
	CleanupHour                   int  `json:"cleanup_hour"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level string `json:"level"`
}

// Load loads configuration from environment variables or config file
// Priority: Environment Variables > Config File > Defaults
func Load() (*Config, error) {
	config := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "c2user",
			Password: "c2pass",
			Database: "c2_db",
			SSLMode:  "disable",
			MaxConns: 25,
			MinConns: 5,
		},
		Server: ServerConfig{
			Port: "8080",
			Host: "0.0.0.0", // Default to accept network connections (can be changed to 127.0.0.1 for localhost-only)
		},
		Features: FeatureConfig{
			AgentInactiveThresholdMinutes: 5,
			RetentionDays:                 30,
			EnableAutoCleanup:             true,
			CleanupHour:                   3,
		},
		Logging: LoggingConfig{
			Level: "INFO",
		},
	}

	// Try to load from config file first (as fallback)
	if _, err := os.Stat("config.json"); err == nil {
		data, err := os.ReadFile("config.json")
		if err != nil {
			return nil, fmt.Errorf("failed to read config.json: %w", err)
		}
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config.json: %w", err)
		}
	}

	// Override with environment variables (highest priority)
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		// DATABASE_URL takes precedence and overrides individual settings
		// Format: postgres://user:password@host:port/database?sslmode=disable
		// For simplicity, we'll parse it in the database package
		// Here we just set a flag that it exists
		config.Database.Host = dbURL // We'll use this as the full connection string
	} else {
		// Individual database settings
		if host := os.Getenv("DB_HOST"); host != "" {
			config.Database.Host = host
		}
		if port := os.Getenv("DB_PORT"); port != "" {
			if p, err := strconv.Atoi(port); err == nil {
				config.Database.Port = p
			}
		}
		if user := os.Getenv("DB_USER"); user != "" {
			config.Database.User = user
		}
		if pass := os.Getenv("DB_PASSWORD"); pass != "" {
			config.Database.Password = pass
		}
		if dbName := os.Getenv("DB_NAME"); dbName != "" {
			config.Database.Database = dbName
		}
		if sslMode := os.Getenv("DB_SSLMODE"); sslMode != "" {
			config.Database.SSLMode = sslMode
		}
	}

	if maxConns := os.Getenv("DB_MAX_CONNS"); maxConns != "" {
		if mc, err := strconv.Atoi(maxConns); err == nil {
			config.Database.MaxConns = mc
		}
	}
	if minConns := os.Getenv("DB_MIN_CONNS"); minConns != "" {
		if mc, err := strconv.Atoi(minConns); err == nil {
			config.Database.MinConns = mc
		}
	}

	// Server settings
	if port := os.Getenv("PORT"); port != "" {
		config.Server.Port = port
	}
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}

	// Feature settings
	if threshold := os.Getenv("AGENT_INACTIVE_THRESHOLD_MINUTES"); threshold != "" {
		if t, err := strconv.Atoi(threshold); err == nil {
			config.Features.AgentInactiveThresholdMinutes = t
		}
	}
	if retention := os.Getenv("RETENTION_DAYS"); retention != "" {
		if r, err := strconv.Atoi(retention); err == nil {
			config.Features.RetentionDays = r
		}
	}
	if cleanup := os.Getenv("ENABLE_AUTO_CLEANUP"); cleanup != "" {
		config.Features.EnableAutoCleanup = cleanup == "true"
	}
	if hour := os.Getenv("CLEANUP_HOUR"); hour != "" {
		if h, err := strconv.Atoi(hour); err == nil {
			config.Features.CleanupHour = h
		}
	}

	// Logging settings
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}

	return config, nil
}

// GetConnectionString builds a PostgreSQL connection string from config
func (c *DatabaseConfig) GetConnectionString() string {
	// If Host looks like a full connection string (contains "postgres://"), use it directly
	if len(c.Host) > 10 && c.Host[:10] == "postgres:/" {
		return c.Host
	}

	// Otherwise, build from individual components
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
	)
}
