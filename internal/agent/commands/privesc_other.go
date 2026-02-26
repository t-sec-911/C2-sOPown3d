//go:build !windows
// +build !windows

package commands

func Privesc() string {
	return "Privesc command is only available on Windows"
}
