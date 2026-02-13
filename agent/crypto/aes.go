package crypto

// This package is a wrapper for backward compatibility
// It re-exports functions from internal/agent/crypto

import "sOPown3d/internal/agent/crypto"

// Encrypt wraps the internal crypto.Encrypt function
func Encrypt(plaintext string) (string, error) {
	return crypto.Encrypt(plaintext)
}

// Decrypt wraps the internal crypto.Decrypt function
func Decrypt(ciphertext string) (string, error) {
	return crypto.Decrypt(ciphertext)
}
