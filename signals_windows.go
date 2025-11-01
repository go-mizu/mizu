//go:build windows
// +build windows

package mizu

import (
	"os"
)

func defaultSignals() []os.Signal {
	// Windows reliably supports Interrupt; SIGTERM exists but is not commonly delivered.
	return []os.Signal{
		os.Interrupt, // Ctrl+C
	}
}
