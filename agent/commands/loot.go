package commands

import (
	"fmt"
	"os"
	"path/filepath"
)

// Cherche des fichiers sensibles
func SearchSensitiveFiles() {
	fmt.Println("\n🔍 Scan pour fichiers sensibles...")

	// Récupérer le dossier utilisateur
	userHome, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("  ❌ Erreur:", err)
		return
	}

	// Extensions de fichiers sensibles
	extensions := []string{".kdbx", ".key", ".pem", ".ppk", ".conf", ".config", ".env", ".rdp"}

	// Dossiers à scanner
	scanPaths := []string{
		userHome + "\\Desktop",
		userHome + "\\Documents",
		userHome + "\\.ssh",
		userHome + "\\AppData\\Roaming\\KeePass",
	}

	fichierTrouve := 0

	// Scanner chaque dossier
	for _, path := range scanPaths {
		// Vérifier si le dossier existe
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue // Dossier n'existe pas, on passe au suivant
		}

		fmt.Printf("  📁 Scan: %s\n", path)

		// Ouvrir le dossier
		files, err := os.ReadDir(path)
		if err != nil {
			continue
		}

		// Parcourir les fichiers
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			// Vérifier l'extension
			ext := filepath.Ext(file.Name())
			for _, sensiExt := range extensions {
				if ext == sensiExt {
					fmt.Printf("    🔍 Fichier trouvé: %s\n", file.Name())
					fichierTrouve++
					break
				}
			}
		}
	}

	// Scanner spécial pour les clés SSH
	sshPath := userHome + "\\.ssh"
	if files, err := os.ReadDir(sshPath); err == nil {
		for _, file := range files {
			if !file.IsDir() {
				if file.Name() == "id_rsa" ||
					file.Name() == "id_dsa" ||
					file.Name() == "authorized_keys" ||
					file.Name() == "known_hosts" {
					fmt.Printf("    🔐 Clé SSH: %s\n", file.Name())
					fichierTrouve++
				}
			}
		}
	}

	fmt.Printf("\n📊 Scan terminé: %d fichiers trouvés\n", fichierTrouve)
}
