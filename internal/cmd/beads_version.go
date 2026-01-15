// Package cmd provides CLI commands for the gt tool.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MinBeadsVersion is the minimum required beads version for Gas Town.
// This version must include custom type support (bd-i54l).
const MinBeadsVersion = "0.47.0"

// beadsVersionCheckTimeout is the maximum time to wait for bd --version.
// If bd hangs, we don't want to block the entire gt command.
const beadsVersionCheckTimeout = 3 * time.Second

// beadsVersionCacheFile stores the cached version check result.
// This avoids running bd --version on every gt command.
const beadsVersionCacheFile = ".gt-beads-version-cache"

// beadsVersionCacheTTL is how long we trust the cached version.
const beadsVersionCacheTTL = 1 * time.Hour

// beadsVersion represents a parsed semantic version.
type beadsVersion struct {
	major int
	minor int
	patch int
}

// versionCheckResult caches the beads version check result.
var (
	versionCheckOnce   sync.Once
	versionCheckResult error
	versionCheckDone   = make(chan struct{})
)

// parseBeadsVersion parses a version string like "0.44.0" into components.
func parseBeadsVersion(v string) (beadsVersion, error) {
	// Strip leading 'v' if present
	v = strings.TrimPrefix(v, "v")

	// Split on dots
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return beadsVersion{}, fmt.Errorf("invalid version format: %s", v)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return beadsVersion{}, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return beadsVersion{}, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch := 0
	if len(parts) >= 3 {
		// Handle versions like "0.44.0-dev" - take only numeric prefix
		patchStr := parts[2]
		if idx := strings.IndexFunc(patchStr, func(r rune) bool {
			return r < '0' || r > '9'
		}); idx != -1 {
			patchStr = patchStr[:idx]
		}
		if patchStr != "" {
			patch, err = strconv.Atoi(patchStr)
			if err != nil {
				return beadsVersion{}, fmt.Errorf("invalid patch version: %s", parts[2])
			}
		}
	}

	return beadsVersion{major: major, minor: minor, patch: patch}, nil
}

// compare returns -1 if v < other, 0 if equal, 1 if v > other.
func (v beadsVersion) compare(other beadsVersion) int {
	if v.major != other.major {
		if v.major < other.major {
			return -1
		}
		return 1
	}
	if v.minor != other.minor {
		if v.minor < other.minor {
			return -1
		}
		return 1
	}
	if v.patch != other.patch {
		if v.patch < other.patch {
			return -1
		}
		return 1
	}
	return 0
}

// getCacheFilePath returns the path to the version cache file.
func getCacheFilePath() string {
	// Use XDG cache or fallback to home directory
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		cacheDir = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheDir, "gastown", beadsVersionCacheFile)
}

// readCachedVersion reads the cached version if still valid.
func readCachedVersion() (string, bool) {
	cachePath := getCacheFilePath()
	if cachePath == "" {
		return "", false
	}

	info, err := os.Stat(cachePath)
	if err != nil {
		return "", false
	}

	// Check if cache is still valid
	if time.Since(info.ModTime()) > beadsVersionCacheTTL {
		return "", false
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return "", false
	}

	return strings.TrimSpace(string(data)), true
}

// writeCachedVersion writes the version to cache.
func writeCachedVersion(version string) {
	cachePath := getCacheFilePath()
	if cachePath == "" {
		return
	}

	// Ensure cache directory exists
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return
	}

	_ = os.WriteFile(cachePath, []byte(version), 0644)
}

// getBeadsVersion executes `bd --version` and parses the output.
// Returns the version string (e.g., "0.44.0") or error.
// Uses a timeout to prevent hanging if bd is unresponsive.
func getBeadsVersion() (string, error) {
	// First check cache
	if cached, ok := readCachedVersion(); ok {
		return cached, nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), beadsVersionCheckTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bd", "--version")
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("bd --version timed out after %v (bd may be hung)", beadsVersionCheckTimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("bd --version failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to run bd: %w (is beads installed?)", err)
	}

	// Parse output like "bd version 0.44.0 (dev)"
	// or "bd version 0.44.0"
	re := regexp.MustCompile(`bd version (\d+\.\d+(?:\.\d+)?(?:-\w+)?)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse beads version from: %s", strings.TrimSpace(string(output)))
	}

	version := matches[1]
	
	// Cache the result for future invocations
	writeCachedVersion(version)

	return version, nil
}

// CheckBeadsVersion verifies that the installed beads version meets the minimum requirement.
// Returns nil if the version is sufficient, or an error with details if not.
// This function is cached and only runs once per process.
func CheckBeadsVersion() error {
	// Use sync.Once to only check once per process
	versionCheckOnce.Do(func() {
		versionCheckResult = doCheckBeadsVersion()
		close(versionCheckDone)
	})
	return versionCheckResult
}

// doCheckBeadsVersion performs the actual version check.
func doCheckBeadsVersion() error {
	installedStr, err := getBeadsVersion()
	if err != nil {
		// If bd timed out, warn but don't block
		if strings.Contains(err.Error(), "timed out") {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			fmt.Fprintf(os.Stderr, "Continuing without version check. Run 'gt doctor' to diagnose.\n")
			return nil
		}
		return fmt.Errorf("cannot verify beads version: %w", err)
	}

	installed, err := parseBeadsVersion(installedStr)
	if err != nil {
		return fmt.Errorf("cannot parse installed beads version %q: %w", installedStr, err)
	}

	required, err := parseBeadsVersion(MinBeadsVersion)
	if err != nil {
		// This would be a bug in our code
		return fmt.Errorf("cannot parse required beads version %q: %w", MinBeadsVersion, err)
	}

	if installed.compare(required) < 0 {
		return fmt.Errorf("beads version %s is required, but %s is installed\n\nPlease upgrade beads: go install github.com/steveyegge/beads/cmd/bd@latest", MinBeadsVersion, installedStr)
	}

	return nil
}

// InvalidateBeadsVersionCache removes the cached version check.
// Useful when upgrading beads or debugging.
func InvalidateBeadsVersionCache() {
	cachePath := getCacheFilePath()
	if cachePath != "" {
		_ = os.Remove(cachePath)
	}
}
