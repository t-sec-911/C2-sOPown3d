package tasks

import (
	"context"
	"time"

	"sOPown3d/server/logger"
	"sOPown3d/server/storage"
)

// CleanupScheduler periodically deletes old command executions
type CleanupScheduler struct {
	storage       storage.Storage
	logger        *logger.Logger
	retentionDays int
	cleanupHour   int
	ticker        *time.Ticker
	stopChan      chan struct{}
}

// NewCleanupScheduler creates a new cleanup scheduler
func NewCleanupScheduler(store storage.Storage, log *logger.Logger, retentionDays, cleanupHour int) *CleanupScheduler {
	return &CleanupScheduler{
		storage:       store,
		logger:        log,
		retentionDays: retentionDays,
		cleanupHour:   cleanupHour,
		stopChan:      make(chan struct{}),
	}
}

// Start begins the cleanup scheduling background task
func (cs *CleanupScheduler) Start() {
	cs.logger.Info(logger.CategoryBackground, "Starting cleanup scheduler (retention: %d days, runs at %02d:00)",
		cs.retentionDays, cs.cleanupHour)

	// Calculate time until next cleanup
	nextCleanup := cs.calculateNextCleanup()
	cs.logger.Info(logger.CategoryCleanup, "Next cleanup scheduled for: %s", nextCleanup.Format("2006-01-02 15:04:05"))

	// Create ticker for 24 hours
	cs.ticker = time.NewTicker(24 * time.Hour)

	go func() {
		// Wait for first cleanup time
		time.Sleep(time.Until(nextCleanup))
		cs.runCleanup()

		// Then run every 24 hours
		for {
			select {
			case <-cs.ticker.C:
				cs.runCleanup()
			case <-cs.stopChan:
				cs.ticker.Stop()
				cs.logger.Info(logger.CategoryBackground, "Cleanup scheduler stopped")
				return
			}
		}
	}()
}

// calculateNextCleanup calculates the next cleanup time
func (cs *CleanupScheduler) calculateNextCleanup() time.Time {
	now := time.Now()
	nextCleanup := time.Date(now.Year(), now.Month(), now.Day(), cs.cleanupHour, 0, 0, 0, now.Location())

	// If cleanup hour has already passed today, schedule for tomorrow
	if nextCleanup.Before(now) {
		nextCleanup = nextCleanup.Add(24 * time.Hour)
	}

	return nextCleanup
}

// runCleanup executes the cleanup operation
func (cs *CleanupScheduler) runCleanup() {
	cs.logger.Info(logger.CategoryCleanup, "Running cleanup: deleting executions >%d days", cs.retentionDays)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deleted, err := cs.storage.CleanupOldExecutions(ctx, cs.retentionDays)
	if err != nil {
		cs.logger.Error(logger.CategoryError, "Cleanup failed: %v", err)
		return
	}

	if deleted > 0 {
		cs.logger.Info(logger.CategorySuccess, "Cleanup complete: %d executions deleted", deleted)
	} else {
		cs.logger.Info(logger.CategoryCleanup, "Cleanup complete: no old executions to delete")
	}
}

// Stop stops the cleanup scheduler
func (cs *CleanupScheduler) Stop() {
	close(cs.stopChan)
}

// RunNow manually triggers cleanup (for API endpoint)
func (cs *CleanupScheduler) RunNow() (int, error) {
	cs.logger.Info(logger.CategoryCleanup, "Manual cleanup triggered")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deleted, err := cs.storage.CleanupOldExecutions(ctx, cs.retentionDays)
	if err != nil {
		cs.logger.Error(logger.CategoryError, "Manual cleanup failed: %v", err)
		return 0, err
	}

	cs.logger.Info(logger.CategorySuccess, "Manual cleanup complete: %d executions deleted", deleted)
	return deleted, nil
}
