//go:build !windows

package main

func setTerminalRaw() (func(), error) {
	// Fallback/No-op on non-Windows platforms for simplicity when x/term is not imported.
	// We can implement termios if needed, but keeping it simple is fine.
	return func() {}, nil
}
