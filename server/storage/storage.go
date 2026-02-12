package storage

import (
	"context"
	"time"
)

// Storage defines the interface for data persistence operations
type Storage interface {
	// Agent operations
	UpsertAgent(ctx context.Context, agent *Agent) error
	GetAgent(ctx context.Context, agentID string) (*Agent, error)
	ListAgents(ctx context.Context) ([]*Agent, error)
	UpdateAgentActivityStatus(ctx context.Context, inactiveThreshold time.Duration) (int, error)

	// Execution operations
	SaveExecution(ctx context.Context, exec *Execution) error
	GetExecutionHistory(ctx context.Context, agentID string, limit, offset int) ([]*Execution, int, error)
	ListExecutions(ctx context.Context, filters ExecutionFilters) ([]*Execution, int, error)

	// Cleanup operations
	CleanupOldExecutions(ctx context.Context, retentionDays int) (int, error)

	// Stats operations
	GetStats(ctx context.Context) (*Stats, error)

	// Health check
	IsAvailable() bool
	GetStatus() string
}
