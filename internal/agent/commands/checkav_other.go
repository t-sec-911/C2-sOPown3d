//go:build !windows
// +build !windows

package commands

func CheckAV() string {
	return "CheckAV command is only available on Windows"
}
