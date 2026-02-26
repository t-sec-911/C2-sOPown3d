//go:build !windows
// +build !windows

package agent

func executeLootCommand() string {
	return "Error: loot command only available on Windows"
}

func executeCheckAVCommand() string {
	return "Error: checkav command only available on Windows"
}

func executePrivescCommand() string {
	return "Error: privesc command only available on Windows"
}
