package evasion

import (
	"math/rand"
	"os"
	"runtime"
	"time"
)

// Vérifie si on est dans une sandbox/VM
func IsSandbox() bool {
	suspicions := 0

	// 1. Vérifier les cores CPU (sandboxes ont souvent < 4 cores)
	if runtime.NumCPU() < 4 {
		suspicions++
	}

	// 2. Vérifier le username (common sandbox users)
	username := os.Getenv("USERNAME")
	sandboxUsers := []string{"WDAGUtilityAccount", "vmware", "vboxuser", "sandbox"}
	for _, user := range sandboxUsers {
		if username == user {
			suspicions++
			break
		}
	}

	// 3. Vérifier le hostname
	hostname, _ := os.Hostname()
	sandboxHosts := []string{"SANDBOX", "VM", "VIRTUAL", "DESKTOP-"}
	for _, host := range sandboxHosts {
		if len(hostname) >= len(host) && hostname[:len(host)] == host {
			suspicions++
			break
		}
	}

	// 4. Vérifier l'uptime (sandboxes démarrent souvent depuis peu)
	// Pas implémenté ici pour simplicité

	return suspicions >= 2 // 2+ indicateurs = probable sandbox
}

// Sleep avec variation pour éviter détection
func SleepWithJitter(seconds int) {
	// Jitter entre 0 et 3 secondes
	jitter := time.Duration(rand.Intn(3000)) * time.Millisecond

	// Dormir
	time.Sleep(time.Duration(seconds)*time.Second + jitter)
}
