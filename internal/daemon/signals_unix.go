//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

func extraSignals() []os.Signal {
	return []os.Signal{syscall.SIGUSR1}
}

func isImmediateSignal(sig os.Signal) bool {
	return sig == syscall.SIGUSR1
}
