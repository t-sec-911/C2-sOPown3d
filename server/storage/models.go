package storage

import "time"

// Agent represents an agent in the system
type Agent struct {
	ID                       int       `json:"id"`
	AgentID                  string    `json:"agent_id"`
	Hostname                 string    `json:"hostname"`
	OS                       string    `json:"os"`
	Username                 string    `json:"username"`
	FirstSeen                time.Time `json:"first_seen"`
	LastSeen                 time.Time `json:"last_seen"`
	IsActive                 bool      `json:"is_active"`
	InactiveThresholdMinutes int       `json:"inactive_threshold_minutes"`
	CreatedAt                time.Time `json:"created_at"`
}

// Execution represents a command execution result
type Execution struct {
	ID             int       `json:"id"`
	AgentID        string    `json:"agent_id"`
	CommandAction  string    `json:"command_action"`
	CommandPayload string    `json:"command_payload"`
	Output         string    `json:"output"`
	ExecutedAt     time.Time `json:"executed_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// Stats represents system statistics
type Stats struct {
	TotalAgents        int    `json:"total_agents"`
	ActiveAgents       int    `json:"active_agents"`
	TotalExecutions    int    `json:"total_executions"`
	ExecutionsLastHour int    `json:"executions_last_hour"`
	DBStatus           string `json:"db_status"`
	InMemoryQueueSize  int    `json:"in_memory_queue_size,omitempty"`
}

// ExecutionFilters holds filters for querying executions
type ExecutionFilters struct {
	AgentID string
	Action  string
	Limit   int
	Offset  int
}
