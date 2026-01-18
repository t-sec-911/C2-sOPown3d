package shared

// Structure pour les données d'agent
type AgentInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Username string `json:"username"`
	Time     string `json:"time"`
}

// Structure pour les commandes
type Command struct {
	ID      string `json:"id"`
	Action  string `json:"action"`
	Payload string `json:"payload,omitempty"`
}
