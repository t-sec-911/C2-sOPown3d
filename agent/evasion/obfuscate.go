package evasion

// Obfuscation XOR simple pour cacher les strings sensibles
func XorString(input string, key byte) string {
	output := make([]byte, len(input))
	for i := 0; i < len(input); i++ {
		output[i] = input[i] ^ key
	}
	return string(output)
}

// Strings sensibles obfusquées
var (
	// "cmd.exe" obfusqué
	CmdExe = XorString("cmd.exe", 0x55)

	// "/c" obfusqué
	CmdC = XorString("/c", 0x55)

	// "whoami" obfusqué
	Whoami = XorString("whoami", 0x55)

	// "shell" obfusqué
	ShellCmd = XorString("shell", 0x55)
)
