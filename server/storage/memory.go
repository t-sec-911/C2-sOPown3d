package storage

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"sOPown3d/server/logger"
)

// MemoryStorage implements Storage interface using in-memory data structures
type MemoryStorage struct {
	agents         map[string]*Agent
	executions     []*Execution
	mu             sync.RWMutex
	logger         *logger.Logger
	executionIDSeq int
	agentIDSeq     int
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage(log *logger.Logger) *MemoryStorage {
	return &MemoryStorage{
		agents:     make(map[string]*Agent),
		executions: make([]*Execution, 0),
		logger:     log,
	}
}

// UpsertAgent inserts or updates an agent
func (s *MemoryStorage) UpsertAgent(ctx context.Context, agent *Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.agents[agent.AgentID]
	if exists {
		// Update existing agent
		existing.Hostname = agent.Hostname
		existing.OS = agent.OS
		existing.Username = agent.Username
		existing.LastSeen = agent.LastSeen
		existing.IsActive = agent.IsActive
	} else {
		// Insert new agent
		s.agentIDSeq++
		agent.ID = s.agentIDSeq
		agent.CreatedAt = time.Now()
		agent.FirstSeen = agent.LastSeen
		s.agents[agent.AgentID] = agent
	}

	return nil
}

// GetAgent retrieves an agent by ID
func (s *MemoryStorage) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, exists := s.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return agent, nil
}

// ListAgents retrieves all agents
func (s *MemoryStorage) ListAgents(ctx context.Context) ([]*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]*Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}

	// Sort by last_seen descending
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].LastSeen.After(agents[j].LastSeen)
	})

	return agents, nil
}

// UpdateAgentActivityStatus marks agents as inactive based on threshold
func (s *MemoryStorage) UpdateAgentActivityStatus(ctx context.Context, inactiveThreshold time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoffTime := time.Now().Add(-inactiveThreshold)
	count := 0

	for _, agent := range s.agents {
		if agent.IsActive && agent.LastSeen.Before(cutoffTime) {
			agent.IsActive = false
			count++
		}
	}

	return count, nil
}

// SaveExecution saves a command execution result
func (s *MemoryStorage) SaveExecution(ctx context.Context, exec *Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.executionIDSeq++
	exec.ID = s.executionIDSeq
	exec.CreatedAt = time.Now()
	s.executions = append(s.executions, exec)

	return nil
}

// GetExecutionHistory retrieves execution history for an agent
func (s *MemoryStorage) GetExecutionHistory(ctx context.Context, agentID string, limit, offset int) ([]*Execution, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter by agent ID
	filtered := make([]*Execution, 0)
	for _, exec := range s.executions {
		if exec.AgentID == agentID {
			filtered = append(filtered, exec)
		}
	}

	// Sort by executed_at descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ExecutedAt.After(filtered[j].ExecutedAt)
	})

	total := len(filtered)

	// Apply pagination
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	result := filtered[start:end]
	return result, total, nil
}

// ListExecutions retrieves executions with filters
func (s *MemoryStorage) ListExecutions(ctx context.Context, filters ExecutionFilters) ([]*Execution, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter executions
	filtered := make([]*Execution, 0)
	for _, exec := range s.executions {
		if filters.AgentID != "" && exec.AgentID != filters.AgentID {
			continue
		}
		if filters.Action != "" && exec.CommandAction != filters.Action {
			continue
		}
		filtered = append(filtered, exec)
	}

	// Sort by executed_at descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ExecutedAt.After(filtered[j].ExecutedAt)
	})

	total := len(filtered)

	// Apply pagination
	start := filters.Offset
	if start > total {
		start = total
	}
	end := start + filters.Limit
	if end > total {
		end = total
	}

	result := filtered[start:end]
	return result, total, nil
}

// CleanupOldExecutions deletes executions older than retention period
func (s *MemoryStorage) CleanupOldExecutions(ctx context.Context, retentionDays int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	count := 0

	// Filter out old executions
	newExecutions := make([]*Execution, 0)
	for _, exec := range s.executions {
		if exec.CreatedAt.After(cutoffTime) {
			newExecutions = append(newExecutions, exec)
		} else {
			count++
		}
	}

	s.executions = newExecutions
	return count, nil
}

// GetStats returns system statistics
func (s *MemoryStorage) GetStats(ctx context.Context) (*Stats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &Stats{
		DBStatus:        s.GetStatus(),
		TotalAgents:     len(s.agents),
		TotalExecutions: len(s.executions),
	}

	// Count active agents
	activeCount := 0
	for _, agent := range s.agents {
		if agent.IsActive {
			activeCount++
		}
	}
	stats.ActiveAgents = activeCount

	// Count executions in last hour
	oneHourAgo := time.Now().Add(-time.Hour)
	execsLastHour := 0
	for _, exec := range s.executions {
		if exec.CreatedAt.After(oneHourAgo) {
			execsLastHour++
		}
	}
	stats.ExecutionsLastHour = execsLastHour

	return stats, nil
}

// IsAvailable always returns true for in-memory storage
func (s *MemoryStorage) IsAvailable() bool {
	return true
}

// GetStatus returns the status
func (s *MemoryStorage) GetStatus() string {
	return "in-memory"
}

// GetQueueSize returns the number of items stored in memory (for resilient wrapper)
func (s *MemoryStorage) GetQueueSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.agents) + len(s.executions)
}
