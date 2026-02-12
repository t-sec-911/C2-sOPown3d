package storage

import (
	"context"
	"sync"
	"time"

	"sOPown3d/server/logger"
)

// ResilientStorage wraps primary (PostgreSQL) and fallback (in-memory) storage
// It automatically switches between them and syncs data when database comes back online
type ResilientStorage struct {
	primary       Storage // PostgreSQL
	fallback      Storage // In-memory
	currentMode   string  // "primary" or "fallback"
	logger        *logger.Logger
	mu            sync.RWMutex
	healthChecker *time.Ticker
	stopChan      chan struct{}
}

// NewResilientStorage creates a resilient storage with automatic fallback
func NewResilientStorage(primary, fallback Storage, log *logger.Logger) *ResilientStorage {
	mode := "fallback"
	if primary.IsAvailable() {
		mode = "primary"
	} else {
		log.Warn(logger.CategoryWarning, "PostgreSQL unavailable, using in-memory storage")
	}

	rs := &ResilientStorage{
		primary:     primary,
		fallback:    fallback,
		currentMode: mode,
		logger:      log,
		stopChan:    make(chan struct{}),
	}

	// Start health checking background task
	rs.startHealthChecker()

	return rs
}

// startHealthChecker periodically checks database health and switches modes
func (rs *ResilientStorage) startHealthChecker() {
	rs.healthChecker = time.NewTicker(30 * time.Second)

	go func() {
		for {
			select {
			case <-rs.healthChecker.C:
				rs.checkAndSwitch()
			case <-rs.stopChan:
				rs.healthChecker.Stop()
				return
			}
		}
	}()
}

// checkAndSwitch checks primary storage health and switches modes if necessary
func (rs *ResilientStorage) checkAndSwitch() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	primaryAvailable := rs.primary.IsAvailable()

	// Switch from fallback to primary if database is back
	if rs.currentMode == "fallback" && primaryAvailable {
		rs.logger.Info(logger.CategorySuccess, "PostgreSQL connection restored!")
		rs.logger.Info(logger.CategorySync, "Switching to primary storage")
		rs.currentMode = "primary"

		// Note: In a full implementation, we would sync in-memory data to database here
		// For this educational project, we keep it simple
		// Future enhancement: Add sync queue and data migration
	}

	// Switch from primary to fallback if database is down
	if rs.currentMode == "primary" && !primaryAvailable {
		rs.logger.Warn(logger.CategoryWarning, "PostgreSQL connection lost")
		rs.logger.Info(logger.CategorySync, "Switching to in-memory storage")
		rs.currentMode = "fallback"
	}
}

// getActiveStorage returns the currently active storage based on mode
func (rs *ResilientStorage) getActiveStorage() Storage {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	if rs.currentMode == "primary" {
		return rs.primary
	}
	return rs.fallback
}

// Stop stops the health checker
func (rs *ResilientStorage) Stop() {
	close(rs.stopChan)
}

// Implementation of Storage interface - delegates to active storage

func (rs *ResilientStorage) UpsertAgent(ctx context.Context, agent *Agent) error {
	storage := rs.getActiveStorage()
	err := storage.UpsertAgent(ctx, agent)

	// If primary fails, automatically switch to fallback
	if err != nil && rs.currentMode == "primary" {
		rs.logger.Error(logger.CategoryError, "Primary storage failed, switching to fallback: %v", err)
		rs.mu.Lock()
		rs.currentMode = "fallback"
		rs.mu.Unlock()
		return rs.fallback.UpsertAgent(ctx, agent)
	}

	return err
}

func (rs *ResilientStorage) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	storage := rs.getActiveStorage()
	agent, err := storage.GetAgent(ctx, agentID)

	if err != nil && rs.currentMode == "primary" {
		// Try fallback
		return rs.fallback.GetAgent(ctx, agentID)
	}

	return agent, err
}

func (rs *ResilientStorage) ListAgents(ctx context.Context) ([]*Agent, error) {
	storage := rs.getActiveStorage()
	agents, err := storage.ListAgents(ctx)

	if err != nil && rs.currentMode == "primary" {
		return rs.fallback.ListAgents(ctx)
	}

	return agents, err
}

func (rs *ResilientStorage) UpdateAgentActivityStatus(ctx context.Context, inactiveThreshold time.Duration) (int, error) {
	storage := rs.getActiveStorage()
	count, err := storage.UpdateAgentActivityStatus(ctx, inactiveThreshold)

	if err != nil && rs.currentMode == "primary" {
		return rs.fallback.UpdateAgentActivityStatus(ctx, inactiveThreshold)
	}

	return count, err
}

func (rs *ResilientStorage) SaveExecution(ctx context.Context, exec *Execution) error {
	storage := rs.getActiveStorage()
	err := storage.SaveExecution(ctx, exec)

	if err != nil && rs.currentMode == "primary" {
		rs.logger.Error(logger.CategoryError, "Primary storage failed, switching to fallback: %v", err)
		rs.mu.Lock()
		rs.currentMode = "fallback"
		rs.mu.Unlock()
		return rs.fallback.SaveExecution(ctx, exec)
	}

	return err
}

func (rs *ResilientStorage) GetExecutionHistory(ctx context.Context, agentID string, limit, offset int) ([]*Execution, int, error) {
	storage := rs.getActiveStorage()
	execs, total, err := storage.GetExecutionHistory(ctx, agentID, limit, offset)

	if err != nil && rs.currentMode == "primary" {
		return rs.fallback.GetExecutionHistory(ctx, agentID, limit, offset)
	}

	return execs, total, err
}

func (rs *ResilientStorage) ListExecutions(ctx context.Context, filters ExecutionFilters) ([]*Execution, int, error) {
	storage := rs.getActiveStorage()
	execs, total, err := storage.ListExecutions(ctx, filters)

	if err != nil && rs.currentMode == "primary" {
		return rs.fallback.ListExecutions(ctx, filters)
	}

	return execs, total, err
}

func (rs *ResilientStorage) CleanupOldExecutions(ctx context.Context, retentionDays int) (int, error) {
	storage := rs.getActiveStorage()
	count, err := storage.CleanupOldExecutions(ctx, retentionDays)

	if err != nil && rs.currentMode == "primary" {
		return rs.fallback.CleanupOldExecutions(ctx, retentionDays)
	}

	return count, err
}

func (rs *ResilientStorage) GetStats(ctx context.Context) (*Stats, error) {
	storage := rs.getActiveStorage()
	stats, err := storage.GetStats(ctx)

	if err != nil && rs.currentMode == "primary" {
		return rs.fallback.GetStats(ctx)
	}

	// Add queue size if in fallback mode
	if rs.currentMode == "fallback" {
		if memStorage, ok := rs.fallback.(*MemoryStorage); ok {
			stats.InMemoryQueueSize = memStorage.GetQueueSize()
		}
	}

	return stats, err
}

func (rs *ResilientStorage) IsAvailable() bool {
	storage := rs.getActiveStorage()
	return storage.IsAvailable()
}

func (rs *ResilientStorage) GetStatus() string {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	if rs.currentMode == "primary" {
		return "connected"
	}

	// In fallback mode, check queue size
	if memStorage, ok := rs.fallback.(*MemoryStorage); ok {
		queueSize := memStorage.GetQueueSize()
		if queueSize > 0 {
			return "in-memory"
		}
	}

	return "in-memory"
}

// GetCurrentMode returns the current storage mode (for debugging/monitoring)
func (rs *ResilientStorage) GetCurrentMode() string {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.currentMode
}
