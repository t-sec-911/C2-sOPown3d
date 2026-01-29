package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// Clé de test (32 bytes = AES-256)
var aesKey = []byte("0123456789abcdef0123456789abcdef")

// Chiffrer un texte
func Encrypt(plaintext string) (string, error) {
	// Convertir en bytes
	plaintextBytes := []byte(plaintext)

	// Créer cipher AES
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}

	// Mode GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Générer nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Chiffrer
	ciphertext := gcm.Seal(nonce, nonce, plaintextBytes, nil)

	// Convertir en base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Déchiffrer
func Decrypt(ciphertextB64 string) (string, error) {
	// Décoder base64
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}

	// Créer cipher AES
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}

	// Mode GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Extraire nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("message trop court")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Déchiffrer
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
