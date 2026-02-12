# PostgreSQL Database Setup Guide

This document explains how to set up and use PostgreSQL for the sOPown3d C2 project.

## Overview

The C2 server now includes PostgreSQL integration for persistent storage of:
- **Agent information** (hostname, OS, username, activity status)
- **Command execution results** (commands sent, outputs received)
- **Activity tracking** (agent last seen, active/inactive status)
- **Statistics** (execution counts, timing data)

## Features

✅ **Automatic Fallback**: Server continues working if database is unavailable (in-memory mode)  
✅ **Connection Pooling**: Efficient connection management with pgx  
✅ **Auto-Migration**: Database schema created automatically on startup  
✅ **Background Tasks**:
  - Activity Checker: Marks agents inactive after 5 minutes (configurable)
  - Cleanup Scheduler: Deletes old executions after 30 days (configurable)
✅ **API Endpoints**: Query agents, executions, and statistics  
✅ **Enhanced Dashboard**: Real-time monitoring with auto-refresh

## Quick Start

### 1. Start PostgreSQL

```bash
# Start PostgreSQL in Docker
just docker-up

# Or manually:
docker-compose up -d
```

**Database credentials:**
- Host: `localhost`
- Port: `5433` (using 5433 to avoid conflict with existing PostgreSQL)
- User: `c2user`
- Password: `c2pass`
- Database: `c2_db`

### 2. Configure Environment

**Option A: Environment Variables (Recommended)**

```bash
# Set the DATABASE_URL
export DATABASE_URL="postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable"

# Start the server
just dev-server
```

**Option B: Config File**

```bash
# Copy example config
cp config.example.json config.json

# Edit config.json with your settings
# Then start server
go run server/main.go
```

### 3. Verify Setup

Check the server logs for successful connection:

```
[2026-02-05 16:48:03] [INFO] 🔌 Connecting to PostgreSQL...
[2026-02-05 16:48:03] [INFO] ✓ Database connected (pool: 5-25 connections)
[2026-02-05 16:48:03] [INFO] 🔌 Running database migrations...
[2026-02-05 16:48:03] [INFO] ✓ Table 'agents' created/verified
[2026-02-05 16:48:03] [INFO] ✓ Table 'command_executions' created/verified
```

## Database Schema

### `agents` Table

Stores information about connected agents.

```sql
CREATE TABLE agents (
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

-- Indexes for performance
CREATE INDEX idx_agents_agent_id ON agents(agent_id);
CREATE INDEX idx_agents_last_seen ON agents(last_seen);
CREATE INDEX idx_agents_is_active ON agents(is_active);
```

### `command_executions` Table

Stores command execution history.

```sql
CREATE TABLE command_executions (
    id SERIAL PRIMARY KEY,
    agent_id VARCHAR(255) NOT NULL,
    command_action VARCHAR(100) NOT NULL,
    command_payload TEXT,
    output TEXT,
    executed_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_executions_agent_id ON command_executions(agent_id);
CREATE INDEX idx_executions_executed_at ON command_executions(executed_at);
CREATE INDEX idx_executions_created_at ON command_executions(created_at);
```

## Configuration Options

### Environment Variables

All configuration can be done via environment variables (`.env` file or shell exports):

```bash
# Database Connection
DATABASE_URL=postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable

# Connection Pool
DB_MAX_CONNS=25
DB_MIN_CONNS=5

# Server
PORT=8080

# Activity Tracking
AGENT_INACTIVE_THRESHOLD_MINUTES=5

# Data Retention
RETENTION_DAYS=30
ENABLE_AUTO_CLEANUP=true
CLEANUP_HOUR=3

# Logging
LOG_LEVEL=INFO
```

### Config File

Alternatively, use `config.json`:

```json
{
  "database": {
    "host": "localhost",
    "port": 5433,
    "user": "c2user",
    "password": "c2pass",
    "database": "c2_db",
    "sslmode": "disable",
    "max_conns": 25,
    "min_conns": 5
  },
  "server": {
    "port": "8080",
    "host": "127.0.0.1"
  },
  "features": {
    "agent_inactive_threshold_minutes": 5,
    "retention_days": 30,
    "enable_auto_cleanup": true,
    "cleanup_hour": 3
  },
  "logging": {
    "level": "INFO"
  }
}
```

## Docker Commands

```bash
# Start PostgreSQL
just docker-up

# Stop PostgreSQL
just docker-down

# View logs
just docker-logs

# Connect to PostgreSQL CLI
just docker-psql

# Check status
just docker-status

# Reset database (delete all data)
just docker-reset
```

## API Endpoints

### Statistics
```bash
GET /api/stats
```

Response:
```json
{
  "total_agents": 5,
  "active_agents": 3,
  "total_executions": 156,
  "executions_last_hour": 12,
  "db_status": "connected"
}
```

### List Agents
```bash
GET /api/agents
```

### Get Agent Details
```bash
GET /api/agents/{agent_id}
```

### Get Agent History
```bash
GET /api/agents/{agent_id}/history?limit=50&offset=0
```

### List Executions
```bash
GET /api/executions?limit=100&offset=0&agent_id=&action=
```

## Background Tasks

### Activity Checker

Runs every 30 seconds to mark agents as inactive if they haven't sent a beacon in the configured threshold (default: 5 minutes).

```
[2026-02-05 15:35:00] [INFO] 🔍 Activity check: 1 agent marked inactive
```

### Cleanup Scheduler

Runs daily at the configured hour (default: 3 AM) to delete old command executions.

```
[2026-02-05 03:00:00] [INFO] 🧹 Running cleanup: deleting executions >30 days
[2026-02-05 03:00:01] [INFO] ✓ Cleanup complete: 1,234 executions deleted
```

## Resilient Storage

The server uses a **resilient storage layer** that automatically handles database failures:

### Normal Operation (PostgreSQL Available)
```
[INFO] 🔌 Connecting to PostgreSQL...
[INFO] ✓ Database connected (pool: 5-25 connections)
```

### Fallback Mode (PostgreSQL Unavailable)
```
[WARN] ⚠️ PostgreSQL unavailable: connection refused
[INFO] 💾 Using in-memory storage (data will be synced when DB is available)
```

### Recovery (PostgreSQL Comes Back Online)
```
[INFO] ✓ PostgreSQL connection restored!
[INFO] 🔄 Switching to primary storage
```

## Testing

### Manual Testing

1. Start PostgreSQL:
   ```bash
   just docker-up
   ```

2. Start the server:
   ```bash
   DATABASE_URL="postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable" go run server/main.go
   ```

3. Start an agent in another terminal:
   ```bash
   just dev-agent
   ```

4. Open dashboard: http://localhost:8080

5. Test API endpoints:
   ```bash
   curl http://localhost:8080/api/stats | jq
   curl http://localhost:8080/api/agents | jq
   ```

### Testing Fallback Mechanism

1. Start server with database
2. Stop Docker: `just docker-down`
3. Server should log: "Switching to in-memory storage"
4. Server continues working
5. Restart Docker: `just docker-up`
6. Server should log: "PostgreSQL connection restored!"

## Troubleshooting

### Port 5432 Already in Use

If you see: `Bind for 0.0.0.0:5432 failed: port is already allocated`

Solution: The docker-compose.yml uses port **5433** to avoid conflicts. Update your DATABASE_URL to use port 5433.

### Connection Refused

If you see: `PostgreSQL unavailable: connection refused`

1. Check Docker is running: `docker ps | grep postgres`
2. Check port is correct: should be **5433**
3. Wait a few seconds for PostgreSQL to initialize
4. Check logs: `just docker-logs`

### Authentication Failed

If you see: `password authentication failed for user "c2user"`

Check your DATABASE_URL credentials match the docker-compose.yml:
- User: `c2user`
- Password: `c2pass`
- Database: `c2_db`

### Tables Not Created

If migrations don't run:

1. Check database logs: `just docker-logs`
2. Connect to database: `just docker-psql`
3. Manually check tables: `\dt`
4. Reset database: `just docker-reset`

## Advanced Usage

### Custom Database Connection

To use a different PostgreSQL instance:

```bash
export DATABASE_URL="postgres://myuser:mypass@myhost:5432/mydb?sslmode=require"
```

### Disable Auto-Cleanup

Set in config:
```json
{
  "features": {
    "enable_auto_cleanup": false
  }
}
```

Or environment variable:
```bash
export ENABLE_AUTO_CLEANUP=false
```

### Manual Cleanup Trigger

Use the API endpoint (future feature):
```bash
curl -X DELETE "http://localhost:8080/api/executions/cleanup?days=30"
```

## Database Queries

### View All Agents
```sql
SELECT * FROM agents ORDER BY last_seen DESC;
```

### View Recent Executions
```sql
SELECT * FROM command_executions 
ORDER BY executed_at DESC 
LIMIT 20;
```

### Count Active Agents
```sql
SELECT COUNT(*) FROM agents WHERE is_active = true;
```

### Get Agent Activity Summary
```sql
SELECT 
    agent_id, 
    hostname, 
    COUNT(*) as execution_count,
    MAX(executed_at) as last_execution
FROM command_executions 
GROUP BY agent_id, hostname
ORDER BY execution_count DESC;
```
