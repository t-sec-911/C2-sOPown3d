# sOPown3d C2 - Usage Guide

Quick reference guide for using the PostgreSQL-enabled C2 server.

## Quick Start

### 1. Start PostgreSQL Database

```bash
just docker-up
```

### 2. Start Server

```bash
# With environment variable
DATABASE_URL="postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable" just dev-server

# Or build and run
just build
DATABASE_URL="postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable" ./build/darwin/server
```

### 3. Start Agent

```bash
# In another terminal
just dev-agent
```

### 4. Access Dashboard

Open your browser: http://localhost:8080

## Dashboard Features

### Statistics Panel
- **Total Agents**: Count of all registered agents
- **Active Agents**: Agents that beaconed in last 5 minutes
- **Total Executions**: All command executions recorded
- **Last Hour**: Executions in the past hour

### Active Agents Table
- Real-time list of connected agents
- Shows: Hostname, OS, Username, Last Seen, Status
- Auto-refreshes every 10 seconds
- Green badge = Active (beaconed recently)
- Red badge = Inactive (no beacon for 5+ minutes)

### Command Panel
1. Select an agent from the dropdown
2. Choose command type:
   - **shell**: Execute system command
   - **info**: Get system information
   - **ping**: Test agent connectivity
   - **persist**: Check persistence status
3. Enter payload (for shell commands)
4. Click "Send Command"
5. Wait for agent's next beacon to execute

### Recent Executions
- Shows last 20 command executions
- Displays: Time, Agent, Action, Payload, Output preview
- Auto-refreshes with other data

## Command Examples

### Execute Shell Command
1. Select agent: `DESKTOP-ABC (windows)`
2. Command: `shell`
3. Payload: `whoami`
4. Send Command

### Get System Info
1. Select agent
2. Command: `info`
3. Send Command (no payload needed)

### Test Connectivity
1. Select agent
2. Command: `ping`
3. Send Command

## API Usage

### Get Statistics
```bash
curl http://localhost:8080/api/stats | jq
```

### List All Agents
```bash
curl http://localhost:8080/api/agents | jq
```

### Get Agent Details
```bash
curl http://localhost:8080/api/agents/DESKTOP-ABC | jq
```

### Get Agent Command History
```bash
curl "http://localhost:8080/api/agents/DESKTOP-ABC/history?limit=10" | jq
```

### List Recent Executions
```bash
curl "http://localhost:8080/api/executions?limit=20" | jq
```

### Filter Executions by Agent
```bash
curl "http://localhost:8080/api/executions?agent_id=DESKTOP-ABC" | jq
```

## Database Operations

### View Database Tables
```bash
just docker-psql
\dt
```

### Query Agents
```bash
just docker-psql
SELECT * FROM agents;
```

### Query Executions
```bash
just docker-psql
SELECT * FROM command_executions ORDER BY executed_at DESC LIMIT 10;
```

### Exit PostgreSQL CLI
```bash
\q
```

## Docker Management

### View Logs
```bash
just docker-logs
```

### Check Status
```bash
just docker-status
```

### Stop Database
```bash
just docker-down
```

### Reset Database (Delete All Data)
```bash
just docker-reset
```

## Testing Scenarios

### Test 1: Basic Workflow

1. Start database and server
2. Start agent
3. Verify agent appears in dashboard (green badge)
4. Send shell command: `whoami`
5. Check Recent Executions for result

### Test 2: Fallback Mode

1. Start server with database
2. Stop Docker: `just docker-down`
3. Server logs: "Switching to in-memory storage"
4. Server continues working
5. Send commands (stored in memory)
6. Restart Docker: `just docker-up`
7. Server logs: "PostgreSQL connection restored!"

### Test 3: Agent Inactivity

1. Start agent
2. Verify active (green badge)
3. Stop agent
4. Wait 5+ minutes
5. Refresh dashboard
6. Agent should show inactive (red badge)

### Test 4: Multiple Agents

1. Start server
2. Start multiple agents (different terminals/machines)
3. Each appears in dashboard with unique ID
4. Send commands to specific agents
5. View individual agent histories

## Configuration Tips

### Change Agent Inactive Threshold

Edit `.env` or set environment variable:
```bash
export AGENT_INACTIVE_THRESHOLD_MINUTES=10
```

### Change Data Retention Period

```bash
export RETENTION_DAYS=7
```

### Disable Auto-Cleanup

```bash
export ENABLE_AUTO_CLEANUP=false
```

### Change Cleanup Time

Run cleanup at 2 AM instead of 3 AM:
```bash
export CLEANUP_HOUR=2
```

### Change Server Port

```bash
export PORT=9090
```

## Logging

The server uses structured logging with emoji indicators:

- 🚀 **Startup** - Server initialization
- 🔌 **Database** - Connection and migrations
- 📥 **Beacon** - Agent check-ins
- 📤 **Command** - Commands sent
- 📝 **Execution** - Results received
- 💾 **Storage** - Data persistence
- 🔍 **Background** - Background tasks
- 🔄 **Sync** - Data synchronization
- 🧹 **Cleanup** - Data cleanup
- ✓ **Success** - Operations succeeded
- ⚠️ **Warning** - Non-critical issues
- ❌ **Error** - Critical failures

Set log level:
```bash
export LOG_LEVEL=INFO  # or DEBUG, WARN, ERROR
```

## Common Workflows

### Daily Operations

Morning:
```bash
just docker-up       # Start database
just dev-server      # Start server
```

During the day:
- Monitor dashboard
- Send commands as needed
- Check execution history
- Monitor agent activity

Evening:
```bash
just docker-down     # Stop database
```

### Development Testing

```bash
# Terminal 1: Database
just docker-up

# Terminal 2: Server
DATABASE_URL="postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable" just dev-server

# Terminal 3: Agent
just dev-agent

# Terminal 4: Testing
curl http://localhost:8080/api/stats | jq
```

### Code Changes

After modifying code:
```bash
# Rebuild
just build

# Or run directly
go run server/main.go
```

## Best Practices

1. **Always check database status** in the dashboard (green = connected)
2. **Monitor background tasks** in server logs
3. **Use API endpoints** for automation/scripting
4. **Set appropriate retention** based on disk space
5. **Backup database** for important data (see DATABASE_SETUP.md)
6. **Review logs regularly** for errors/warnings
7. **Test fallback mode** to ensure resilience

## Troubleshooting

### Agent not appearing in dashboard
- Check agent is running: `ps aux | grep agent`
- Check server logs for beacon messages
- Verify agent can reach server (firewall/network)

### Commands not executing
- Verify agent is active (green badge)
- Commands execute on next beacon (wait ~2s)
- Check Recent Executions for results
- Review server logs for errors

### Database connection failed
- Verify Docker is running: `just docker-status`
- Check port 5433 is not blocked
- Verify DATABASE_URL is correct
- Server should fallback to in-memory automatically

### High memory usage
- Reduce retention days
- Enable auto-cleanup
- Run manual cleanup
- Monitor execution count in stats

## Advanced Usage

### Custom Agent ID

Agents use hostname by default. To customize, modify `agent/main.go`:

```go
info := gatherSystemInfo()
customID := "custom-agent-001"
```

### Batch Commands

Use API to send commands programmatically:

```bash
#!/bin/bash
AGENTS=$(curl -s http://localhost:8080/api/agents | jq -r '.agents[].agent_id')

for agent in $AGENTS; do
  curl -X POST http://localhost:8080/command \
    -H "Content-Type: application/json" \
    -d "{\"id\":\"$agent\",\"action\":\"shell\",\"payload\":\"hostname\"}"
done
```

### Export Data

Export executions to JSON:

```bash
curl "http://localhost:8080/api/executions?limit=1000" | jq > executions_backup.json
```

### Database Backup

Backup PostgreSQL:

```bash
docker-compose exec postgres pg_dump -U c2user c2_db > backup.sql
```

Restore:

```bash
docker-compose exec -T postgres psql -U c2user c2_db < backup.sql
```

## Educational Value

This project demonstrates:
- **C2 architecture** with persistence
- **RESTful API design**
- **Real-time monitoring** dashboards
- **Database integration** in Go
- **Resilient system design**
- **Background task scheduling**
- **Docker containerization**

Perfect for learning modern application development!

## Next Steps

After mastering the basics:
1. Explore the codebase in `server/` directory
2. Review database schema in DATABASE_SETUP.md
3. Experiment with API endpoints
4. Modify agent behavior in `agent/main.go`
5. Enhance dashboard UI
6. Add custom commands
7. Implement encryption (see crypto package)

## Support

For issues or questions:
1. Check logs: Server console and `just docker-logs`
2. Review DATABASE_SETUP.md for detailed docs
3. Check GitHub issues (if applicable)
4. Educational project - experiment and learn!
