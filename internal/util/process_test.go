package util

import (
	"testing"
)

func TestProcessExistsNonExistent(t *testing.T) {
	// Using a very high PID that's unlikely to exist
	pid := 999999999
	if ProcessExists(pid) {
		t.Errorf("ProcessExists(%d) = true, want false for non-existent process", pid)
	}
}

func TestProcessExistsNegativePID(t *testing.T) {
	// Negative PIDs are invalid and should return false or may cause errors
	// depending on the platform, so just test that it doesn't panic
	_ = ProcessExists(-1)
}

func TestProcessExistsZero(t *testing.T) {
	// PID 0 is special (kernel process on Unix)
	// Test that we can call it without panicking
	_ = ProcessExists(0)
}
