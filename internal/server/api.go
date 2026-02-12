package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"sOPown3d/server/logger"
	"sOPown3d/server/storage"
)

func (s *Server) handleAPIAgents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	agents, err := s.store.ListAgents(ctx)
	if err != nil {
		s.logger.Error(logger.CategoryError, "failed to list agents: %v", err)
		http.Error(w, "failed to retrieve agents", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"agents": agents,
		"total":  len(agents),
	}

	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAPIAgentDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	agentID := ""
	isHistory := false

	if len(path) > len("/api/agents/") {
		remainder := path[len("/api/agents/"):]
		if len(remainder) > 0 {
			if len(remainder) > 8 && remainder[len(remainder)-8:] == "/history" {
				agentID = remainder[:len(remainder)-8]
				isHistory = true
			} else {
				agentID = remainder
			}
		}
	}

	if agentID == "" {
		http.Error(w, "agent ID required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if isHistory {
		limit := 50
		offset := 0

		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			fmt.Sscanf(limitStr, "%d", &limit)
		}
		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			fmt.Sscanf(offsetStr, "%d", &offset)
		}

		executions, total, err := s.store.GetExecutionHistory(ctx, agentID, limit, offset)
		if err != nil {
			s.logger.Error(logger.CategoryError, "failed to get execution history: %v", err)
			http.Error(w, "failed to retrieve history", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"agent_id":   agentID,
			"executions": executions,
			"total":      total,
			"limit":      limit,
			"offset":     offset,
		}

		_ = json.NewEncoder(w).Encode(response)
	} else {
		agent, err := s.store.GetAgent(ctx, agentID)
		if err != nil {
			s.logger.Error(logger.CategoryError, "failed to get agent: %v", err)
			http.Error(w, "agent not found", http.StatusNotFound)
			return
		}

		_, total, err := s.store.GetExecutionHistory(ctx, agentID, 0, 0)
		if err != nil {
			total = 0
		}

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

		_ = json.NewEncoder(w).Encode(response)
	}
}

func (s *Server) handleAPIExecutions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

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

	executions, total, err := s.store.ListExecutions(ctx, filters)
	if err != nil {
		s.logger.Error(logger.CategoryError, "failed to list executions: %v", err)
		http.Error(w, "failed to retrieve executions", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"executions": executions,
		"total":      total,
		"limit":      filters.Limit,
		"offset":     filters.Offset,
	}

	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := s.store.GetStats(ctx)
	if err != nil {
		s.logger.Error(logger.CategoryError, "failed to get stats: %v", err)
		http.Error(w, "failed to retrieve statistics", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(stats)
}
