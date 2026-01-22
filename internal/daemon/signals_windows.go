//go:build windows

package daemon

import "os"

func extraSignals() []os.Signal {
	return nil
}

func isImmediateSignal(sig os.Signal) bool {
	return false
}
