package evasion

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Détection complète de sandbox/VM
func IsSandbox() bool {
	fmt.Println("\n🔍 Détection d'environnement...")
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
			fmt.Printf("  ⚠️ Processus VM détecté: %s\n", proc)
			suspicions++
		}
	}

	// 2. Vérifier les noms de CPU
	cpuName := getCPUName()
	vmCPUs := []string{"QEMU", "VirtualBox", "VMware", "KVM"}
	for _, vmCPU := range vmCPUs {
		if strings.Contains(cpuName, vmCPU) {
			fmt.Printf("  ⚠️ CPU VM détecté: %s\n", cpuName)
			suspicions++
			break
		}
	}

	// 3. Vérifier le nombre de CPUs
	if runtime.NumCPU() < 4 {
		fmt.Printf("  ⚠️ Peu de CPUs: %d\n", runtime.NumCPU())
		suspicions++
	}

	// 4. Test du temps
	fmt.Println("  ⏳ Test de temporisation...")
	start := time.Now()
	time.Sleep(2 * time.Second)
	elapsed := time.Since(start)

	if elapsed < 2*time.Second {
		fmt.Println("  ⚠️ Anomalie temporelle détectée!")
		suspicions++
	}

	// Décision finale
	fmt.Printf("🔍 Résultat: %d indicateurs suspects\n", suspicions)
	return suspicions >= 2
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
