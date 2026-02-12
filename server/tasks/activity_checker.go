package tasks

import (
	"context"
	"time"

	"sOPown3d/server/logger"
	"sOPown3d/server/storage"
)

// ActivityChecker periodically checks agent activity and marks inactive agents
type ActivityChecker struct {
	storage           storage.Storage
	logger            *logger.Logger
	inactiveThreshold time.Duration
	checkInterval     time.Duration
	ticker            *time.Ticker
	stopChan          chan struct{}
}

// NewActivityChecker creates a new activity checker
func NewActivityChecker(store storage.Storage, log *logger.Logger, thresholdMinutes int) *ActivityChecker {
	return &ActivityChecker{
		storage:           store,
		logger:            log,
		inactiveThreshold: time.Duration(thresholdMinutes) * time.Minute,
		checkInterval:     30 * time.Second,
		stopChan:          make(chan struct{}),
	}
}

// Start begins the activity checking background task
func (ac *ActivityChecker) Start() {
	ac.logger.Info(logger.CategoryBackground, "Starting activity checker (threshold: %v, interval: %v)",
		ac.inactiveThreshold, ac.checkInterval)

	ac.ticker = time.NewTicker(ac.checkInterval)

	go func() {
		for {
			select {
			case <-ac.ticker.C:
				ac.checkAgentActivity()
			case <-ac.stopChan:
				ac.ticker.Stop()
				ac.logger.Info(logger.CategoryBackground, "Activity checker stopped")
				return
			}
		}
	}()
}

// checkAgentActivity checks and updates agent activity status
func (ac *ActivityChecker) checkAgentActivity() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Update inactive agents
	count, err := ac.storage.UpdateAgentActivityStatus(ctx, ac.inactiveThreshold)
	if err != nil {
		ac.logger.Error(logger.CategoryError, "Failed to update agent activity: %v", err)
		return
	}

	if count > 0 {
		ac.logger.Info(logger.CategoryBackground, "Activity check: %d agent(s) marked inactive", count)
	}

	// Get current stats
	stats, err := ac.storage.GetStats(ctx)
	if err != nil {
		ac.logger.Error(logger.CategoryError, "Failed to get stats: %v", err)
		return
	}

	ac.logger.Debug(logger.CategoryBackground, "Active agents: %d/%d", stats.ActiveAgents, stats.TotalAgents)
}

// Stop stops the activity checker
func (ac *ActivityChecker) Stop() {
	close(ac.stopChan)
}
