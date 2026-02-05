package shared

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
	AgentInfo    string // Voir avec Robin pour le hash unique
	Output       string
	DefaultAgent string
}

type JitterConfig struct {
	MinSeconds float64 // Minimum jitter in seconds
	MaxSeconds float64 // Maximum jitter in seconds
}
