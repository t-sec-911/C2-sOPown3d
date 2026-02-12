package database

import (
	"context"
	"time"
)

// IsHealthy checks if the database connection is healthy
func (db *DB) IsHealthy() bool {
	if db.Pool == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.Pool.Ping(ctx); err != nil {
		return false
	}

	return true
}

// GetStatus returns a human-readable status string
func (db *DB) GetStatus() string {
	if db.IsHealthy() {
		return "connected"
	}
	return "disconnected"
}
