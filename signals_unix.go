//go:build !windows

package mizu

import (
	"os"
	"syscall"
)

func defaultSignals() []os.Signal {
	return []os.Signal{
		os.Interrupt,    // Ctrl+C
		syscall.SIGTERM, // container/OS shutdown
		syscall.SIGQUIT, // quit
		syscall.SIGHUP,  // terminal hangup / reload
	}
}
