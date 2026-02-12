package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"sOPown3d/server/database"
	"sOPown3d/server/logger"
)

// PostgresStorage implements Storage interface using PostgreSQL
type PostgresStorage struct {
	db     *database.DB
	logger *logger.Logger
}

// NewPostgresStorage creates a new PostgreSQL storage
func NewPostgresStorage(db *database.DB, log *logger.Logger) *PostgresStorage {
	return &PostgresStorage{
		db:     db,
		logger: log,
	}
}

// UpsertAgent inserts or updates an agent
func (s *PostgresStorage) UpsertAgent(ctx context.Context, agent *Agent) error {
	query := `
		INSERT INTO agents (agent_id, hostname, os, username, first_seen, last_seen, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (agent_id) 
		DO UPDATE SET 
			hostname = EXCLUDED.hostname,
			os = EXCLUDED.os,
			username = EXCLUDED.username,
			last_seen = EXCLUDED.last_seen,
			is_active = EXCLUDED.is_active
	`

	_, err := s.db.Pool.Exec(ctx, query,
		agent.AgentID,
		agent.Hostname,
		agent.OS,
		agent.Username,
		agent.FirstSeen,
		agent.LastSeen,
		agent.IsActive,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert agent: %w", err)
	}

	return nil
}

// GetAgent retrieves an agent by ID
func (s *PostgresStorage) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	query := `
		SELECT id, agent_id, hostname, os, username, first_seen, last_seen, 
		       is_active, inactive_threshold_minutes, created_at
		FROM agents
		WHERE agent_id = $1
	`

	agent := &Agent{}
	err := s.db.Pool.QueryRow(ctx, query, agentID).Scan(
		&agent.ID,
		&agent.AgentID,
		&agent.Hostname,
		&agent.OS,
		&agent.Username,
		&agent.FirstSeen,
		&agent.LastSeen,
		&agent.IsActive,
		&agent.InactiveThresholdMinutes,
		&agent.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return agent, nil
}

// ListAgents retrieves all agents
func (s *PostgresStorage) ListAgents(ctx context.Context) ([]*Agent, error) {
	query := `
		SELECT id, agent_id, hostname, os, username, first_seen, last_seen, 
		       is_active, inactive_threshold_minutes, created_at
		FROM agents
		ORDER BY last_seen DESC
	`

	rows, err := s.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		agent := &Agent{}
		err := rows.Scan(
			&agent.ID,
			&agent.AgentID,
			&agent.Hostname,
			&agent.OS,
			&agent.Username,
			&agent.FirstSeen,
			&agent.LastSeen,
			&agent.IsActive,
			&agent.InactiveThresholdMinutes,
			&agent.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// UpdateAgentActivityStatus marks agents as inactive based on threshold
func (s *PostgresStorage) UpdateAgentActivityStatus(ctx context.Context, inactiveThreshold time.Duration) (int, error) {
	query := `
		UPDATE agents
		SET is_active = false
		WHERE is_active = true
		  AND last_seen < $1
	`

	cutoffTime := time.Now().Add(-inactiveThreshold)
	result, err := s.db.Pool.Exec(ctx, query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to update agent activity status: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// SaveExecution saves a command execution result
func (s *PostgresStorage) SaveExecution(ctx context.Context, exec *Execution) error {
	query := `
		INSERT INTO command_executions (agent_id, command_action, command_payload, output, executed_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err := s.db.Pool.QueryRow(ctx, query,
		exec.AgentID,
		exec.CommandAction,
		exec.CommandPayload,
		exec.Output,
		exec.ExecutedAt,
	).Scan(&exec.ID)

	if err != nil {
		return fmt.Errorf("failed to save execution: %w", err)
	}

	return nil
}

// GetExecutionHistory retrieves execution history for an agent
func (s *PostgresStorage) GetExecutionHistory(ctx context.Context, agentID string, limit, offset int) ([]*Execution, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM command_executions WHERE agent_id = $1`
	var total int
	if err := s.db.Pool.QueryRow(ctx, countQuery, agentID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count executions: %w", err)
	}

	// Get executions
	query := `
		SELECT id, agent_id, command_action, command_payload, output, executed_at, created_at
		FROM command_executions
		WHERE agent_id = $1
		ORDER BY executed_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Pool.Query(ctx, query, agentID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get execution history: %w", err)
	}
	defer rows.Close()

	var executions []*Execution
	for rows.Next() {
		exec := &Execution{}
		err := rows.Scan(
			&exec.ID,
			&exec.AgentID,
			&exec.CommandAction,
			&exec.CommandPayload,
			&exec.Output,
			&exec.ExecutedAt,
			&exec.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan execution: %w", err)
		}
		executions = append(executions, exec)
	}

	return executions, total, nil
}

// ListExecutions retrieves executions with filters
func (s *PostgresStorage) ListExecutions(ctx context.Context, filters ExecutionFilters) ([]*Execution, int, error) {
	// Build query with filters
	baseQuery := `SELECT id, agent_id, command_action, command_payload, output, executed_at, created_at FROM command_executions`
	countQuery := `SELECT COUNT(*) FROM command_executions`
	whereClause := ""
	args := []interface{}{}
	argNum := 1

	if filters.AgentID != "" {
		whereClause += fmt.Sprintf(" WHERE agent_id = $%d", argNum)
		args = append(args, filters.AgentID)
		argNum++
	}

	if filters.Action != "" {
		if whereClause == "" {
			whereClause += " WHERE"
		} else {
			whereClause += " AND"
		}
		whereClause += fmt.Sprintf(" command_action = $%d", argNum)
		args = append(args, filters.Action)
		argNum++
	}

	// Get total count
	var total int
	if err := s.db.Pool.QueryRow(ctx, countQuery+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count executions: %w", err)
	}

	// Get executions
	query := baseQuery + whereClause + fmt.Sprintf(" ORDER BY executed_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filters.Limit, filters.Offset)

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var executions []*Execution
	for rows.Next() {
		exec := &Execution{}
		err := rows.Scan(
			&exec.ID,
			&exec.AgentID,
			&exec.CommandAction,
			&exec.CommandPayload,
			&exec.Output,
			&exec.ExecutedAt,
			&exec.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan execution: %w", err)
		}
		executions = append(executions, exec)
	}

	return executions, total, nil
}

// CleanupOldExecutions deletes executions older than retention period
func (s *PostgresStorage) CleanupOldExecutions(ctx context.Context, retentionDays int) (int, error) {
	query := `
		DELETE FROM command_executions
		WHERE created_at < NOW() - INTERVAL '1 day' * $1
	`

	result, err := s.db.Pool.Exec(ctx, query, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old executions: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// GetStats returns system statistics
func (s *PostgresStorage) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{
		DBStatus: s.GetStatus(),
	}

	// Total agents
	if err := s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM agents`).Scan(&stats.TotalAgents); err != nil {
		return nil, fmt.Errorf("failed to get total agents: %w", err)
	}

	// Active agents
	if err := s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM agents WHERE is_active = true`).Scan(&stats.ActiveAgents); err != nil {
		return nil, fmt.Errorf("failed to get active agents: %w", err)
	}

	// Total executions
	if err := s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM command_executions`).Scan(&stats.TotalExecutions); err != nil {
		return nil, fmt.Errorf("failed to get total executions: %w", err)
	}

	// Executions in last hour
	if err := s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM command_executions WHERE created_at > NOW() - INTERVAL '1 hour'`).Scan(&stats.ExecutionsLastHour); err != nil {
		return nil, fmt.Errorf("failed to get executions last hour: %w", err)
	}

	return stats, nil
}

// IsAvailable checks if the database is available
func (s *PostgresStorage) IsAvailable() bool {
	return s.db.IsHealthy()
}

// GetStatus returns the database status
func (s *PostgresStorage) GetStatus() string {
	return s.db.GetStatus()
}
