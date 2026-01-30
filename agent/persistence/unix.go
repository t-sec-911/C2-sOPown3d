//go:build !windows
// +build !windows

package persistence

import "fmt"

// AddToStartup is a no-op on non-Windows systems
func AddToStartup(agentPath string) error {
	fmt.Println("[Persistence] Info: Persistence features only available on Windows")
	return nil
}

// CheckStartup always returns false on non-Windows systems
func CheckStartup() (bool, string) {
	return false, ""
}

// RemoveFromStartup is a no-op on non-Windows systems
func RemoveFromStartup() error {
	return nil
}
