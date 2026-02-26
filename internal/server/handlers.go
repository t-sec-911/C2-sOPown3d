package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"sOPown3d/pkg/shared"
	"sOPown3d/server/logger"
	"sOPown3d/server/storage"

	"github.com/gorilla/websocket"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := shared.DashboardData{
		AgentInfo:    "AgentID à récupérer dynamiquement",
		DefaultAgent: "Nicolass-MacBook-Pro.local",
	}

	if err := s.templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		s.logger.Error(logger.CategoryError, "template error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleBeacon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var agentInfo shared.AgentInfo
	if err := json.NewDecoder(r.Body).Decode(&agentInfo); err != nil {
		s.logger.Error(logger.CategoryError, "beacon JSON decode error: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	agentID := agentInfo.Hostname
	s.logger.Info(logger.CategoryBeacon, "Beacon received: agent=%s os=%s", agentID, agentInfo.OS)

	// Save agent to storage
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	agent := &storage.Agent{
		AgentID:  agentID,
		Hostname: agentInfo.Hostname,
		OS:       agentInfo.OS,
		Username: agentInfo.Username,
		LastSeen: time.Now().UTC(),
		IsActive: true,
	}

	if err := s.store.UpsertAgent(ctx, agent); err != nil {
		s.logger.Error(logger.CategoryError, "failed to save agent: %v", err)
	} else {
		s.logger.Info(logger.CategoryStorage, "Agent '%s' updated", agentID)
	}

	// Check for pending commands
	if cmd, exists := s.pendingCommands[agentID]; exists {
		_ = json.NewEncoder(w).Encode(cmd)
		delete(s.pendingCommands, agentID)
		s.lastCommandSent[agentID] = cmd
		s.logger.Info(logger.CategoryCommand, "Command delivered to agent '%s': action=%s", agentID, cmd.Action)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{}`))
}

func (s *Server) handleSendCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var cmd shared.Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	if cmd.ID == "" || cmd.Action == "" {
		http.Error(w, "id and action required", http.StatusBadRequest)
		return
	}

	s.pendingCommands[cmd.ID] = cmd
	s.logger.Info(logger.CategoryCommand, "Command queued: agent=%s action=%s payload=%s", cmd.ID, cmd.Action, cmd.Payload)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		AgentID string `json:"agent_id"`
		Output  string `json:"output"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	s.logger.Info(logger.CategoryExecution, "Execution result: agent=%s output=%d bytes", payload.AgentID, len(payload.Output))

	// Save execution to storage
	if cmd, exists := s.lastCommandSent[payload.AgentID]; exists {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		exec := &storage.Execution{
			AgentID:        payload.AgentID,
			CommandAction:  cmd.Action,
			CommandPayload: cmd.Payload,
			Output:         payload.Output,
			ExecutedAt:     time.Now(),
		}

		if err := s.store.SaveExecution(ctx, exec); err != nil {
			s.logger.Error(logger.CategoryError, "failed to save execution: %v", err)
		} else {
			s.logger.Info(logger.CategoryStorage, "Execution saved: agent=%s action=%s", payload.AgentID, cmd.Action)
		}

		delete(s.lastCommandSent, payload.AgentID)
	}

	// Send to WebSocket
	s.wsMu.RLock()
	conn, ok := s.wsClients[payload.AgentID]
	s.wsMu.RUnlock()

	if ok {
		s.logger.Info(logger.CategoryWebSocket, "Sending to WebSocket: agent=%s", payload.AgentID)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(payload.Output)); err != nil {
			s.logger.Error(logger.CategoryError, "WS write error: %v", err)
		}
	} else {
		s.logger.Warn(logger.CategoryWarning, "No WebSocket for agent=%s", payload.AgentID)
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent")
	if agentID == "" {
		http.Error(w, "missing agent param", http.StatusBadRequest)
		return
	}

	s.logger.Info(logger.CategoryWebSocket, "WebSocket request: agent=%s", agentID)

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error(logger.CategoryError, "WebSocket upgrade error: %v", err)
		return
	}

	s.wsMu.Lock()
	s.wsClients[agentID] = conn
	s.wsMu.Unlock()

	defer func() {
		s.wsMu.Lock()
		delete(s.wsClients, agentID)
		s.wsMu.Unlock()
		conn.Close()
		s.logger.Info(logger.CategoryWebSocket, "WebSocket closed: agent=%s", agentID)
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
