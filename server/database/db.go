package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"sOPown3d/server/config"
	"sOPown3d/server/logger"
)

// DB wraps the pgxpool connection pool
type DB struct {
	Pool   *pgxpool.Pool
	logger *logger.Logger
}

// Connect establishes a connection pool to PostgreSQL
func Connect(cfg *config.DatabaseConfig, log *logger.Logger) (*DB, error) {
	connString := cfg.GetConnectionString()

	// Parse connection string
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	log.Info(logger.CategoryDatabase, "Connecting to PostgreSQL...")

	// Create connection pool with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info(logger.CategorySuccess, "Database connected (pool: %d-%d connections)", cfg.MinConns, cfg.MaxConns)

	return &DB{
		Pool:   pool,
		logger: log,
	}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.Pool != nil {
		db.logger.Info(logger.CategoryDatabase, "Closing database connection pool...")
		db.Pool.Close()
	}
}

// Stats returns connection pool statistics
func (db *DB) Stats() string {
	if db.Pool == nil {
		return "Pool: not connected"
	}
	stat := db.Pool.Stat()
	return fmt.Sprintf(
		"Pool: total=%d idle=%d acquired=%d",
		stat.TotalConns(),
		stat.IdleConns(),
		stat.AcquiredConns(),
	)
}
