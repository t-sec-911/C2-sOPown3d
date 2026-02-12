package database

import (
	"context"
	"fmt"

	"sOPown3d/server/logger"
)

const createAgentsTable = `
CREATE TABLE IF NOT EXISTS agents (
    id SERIAL PRIMARY KEY,
    agent_id VARCHAR(255) UNIQUE NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    os VARCHAR(50) NOT NULL,
    username VARCHAR(255),
    first_seen TIMESTAMP NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMP NOT NULL DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    inactive_threshold_minutes INTEGER DEFAULT 5,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agents_agent_id ON agents(agent_id);
CREATE INDEX IF NOT EXISTS idx_agents_last_seen ON agents(last_seen);
CREATE INDEX IF NOT EXISTS idx_agents_is_active ON agents(is_active);
`

const createExecutionsTable = `
CREATE TABLE IF NOT EXISTS command_executions (
    id SERIAL PRIMARY KEY,
    agent_id VARCHAR(255) NOT NULL,
    command_action VARCHAR(100) NOT NULL,
    command_payload TEXT,
    output TEXT,
    executed_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_executions_agent_id ON command_executions(agent_id);
CREATE INDEX IF NOT EXISTS idx_executions_executed_at ON command_executions(executed_at);
CREATE INDEX IF NOT EXISTS idx_executions_created_at ON command_executions(created_at);
`

// RunMigrations executes all database migrations
func (db *DB) RunMigrations(ctx context.Context) error {
	db.logger.Info(logger.CategoryDatabase, "Running database migrations...")

	// Create agents table
	if _, err := db.Pool.Exec(ctx, createAgentsTable); err != nil {
		return fmt.Errorf("failed to create agents table: %w", err)
	}
	db.logger.Info(logger.CategorySuccess, "Table 'agents' created/verified")

	// Create command_executions table
	if _, err := db.Pool.Exec(ctx, createExecutionsTable); err != nil {
		return fmt.Errorf("failed to create command_executions table: %w", err)
	}
	db.logger.Info(logger.CategorySuccess, "Table 'command_executions' created/verified")

	db.logger.Info(logger.CategorySuccess, "Migrations complete")
	return nil
}
