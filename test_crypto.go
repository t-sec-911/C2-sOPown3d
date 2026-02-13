package main

import (
	"fmt"
	"sOPown3d/internal/agent/crypto"
)

func main() {
	fmt.Println("=== Test Chiffrement AES ===")
	fmt.Println("Usage académique uniquement")
	fmt.Println()

	// Message de test
	message := "sOPown3d secret message"

	fmt.Println("1. Message original:")
	fmt.Println("   ", message)
	fmt.Println()

	// Chiffrer
	fmt.Println("2. Chiffrement...")
	encrypted, err := crypto.Encrypt(message)
	if err != nil {
		fmt.Println("   ❌ Erreur:", err)
		return
	}
	fmt.Println("   Message chiffré:")
	fmt.Println("   ", encrypted)
	fmt.Println()

	// Déchiffrer
	fmt.Println("3. Déchiffrement...")
	decrypted, err := crypto.Decrypt(encrypted)
	if err != nil {
		fmt.Println("   ❌ Erreur:", err)
		return
	}
	fmt.Println("   Message déchiffré:")
	fmt.Println("   ", decrypted)
	fmt.Println()

	// Vérification
	if message == decrypted {
		fmt.Println("✅ SUCCÈS : Chiffrement/déchiffrement fonctionne !")
	} else {
		fmt.Println("❌ ÉCHEC : Les messages ne correspondent pas")
	}
}
