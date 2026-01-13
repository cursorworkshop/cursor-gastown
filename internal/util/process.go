// Package util provides utility functions for Gas Town.
// This file was created as part of an E2E polecat workflow test.
package util

import (
	"os"
	"syscall"
)

// ProcessExists checks if a process with the given PID exists.
// It sends signal 0 to the process, which doesn't actually send a signal
// but does perform error checking to see if the process exists.
func ProcessExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks if process exists without sending a real signal
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
