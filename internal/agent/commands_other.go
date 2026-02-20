//go:build !windows
// +build !windows

package agent

func executeLootCommand() string {
	return "Error: loot command only available on Windows"
}
