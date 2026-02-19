package shared

import "time"

type AgentInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Username string `json:"username"`
}

type Command struct {
	ID      string `json:"id"`
	Action  string `json:"action"`
	Payload string `json:"payload,omitempty"`
}

type DashboardData struct {
	AgentInfo    string
	Output       string
	DefaultAgent string
}

type JitterConfig struct {
	MinSeconds float64
	MaxSeconds float64
}

type Agent struct {
	AgentID   string    `json:"agent_id"`
	Hostname  string    `json:"hostname"`
	OS        string    `json:"os"`
	Username  string    `json:"username"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	IsActive  bool      `json:"is_active"`
}

type Execution struct {
	ID             int       `json:"id"`
	AgentID        string    `json:"agent_id"`
	CommandAction  string    `json:"command_action"`
	CommandPayload string    `json:"command_payload"`
	Output         string    `json:"output"`
	ExecutedAt     time.Time `json:"executed_at"`
}

type ExecutionFilters struct {
	AgentID string
	Action  string
	Limit   int
	Offset  int
}

type Stats struct {
	TotalAgents      int `json:"total_agents"`
	ActiveAgents     int `json:"active_agents"`
	TotalExecutions  int `json:"total_executions"`
	RecentExecutions int `json:"recent_executions"`
}
