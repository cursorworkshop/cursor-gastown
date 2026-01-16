// gt is the Gas Town CLI for managing multi-agent workspaces.
package main

import (
	"os"
	"runtime"

	"github.com/steveyegge/gastown/internal/cmd"
)

func init() {
	// On macOS, avoid calling Security.framework for certificate verification.
	// Security.framework can hang indefinitely on some systems due to securityd issues.
	// Using fallback roots embeds Mozilla's cert bundle instead.
	if runtime.GOOS == "darwin" {
		// Only set if not already set by user
		if os.Getenv("GODEBUG") == "" {
			os.Setenv("GODEBUG", "x509usefallbackroots=1")
		}
	}
}

func main() {
	os.Exit(cmd.Execute())
}
