#!/bin/bash
# Gas Town sessionEnd hook for Cursor
#
# Called when a session ends. Fires reliably in both CLI and IDE modes.
# Use this for cleanup, cost recording, and bead sync.
#
# Input:  {"session_id": "...", "reason": "completed"|"aborted"|"error"|..., "duration_ms": N, ...}
# Output: (fire-and-forget, no output expected)

# Read JSON input from stdin (required - must consume it)
input=$(cat)

# Export PATH to ensure gt/bd are available
export PATH="$HOME/go/bin:$HOME/bin:$HOME/.local/bin:$PATH"

# Parse reason for logging
reason=$(echo "$input" | grep -o '"reason":"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "unknown")
duration=$(echo "$input" | grep -o '"duration_ms":[0-9]*' | cut -d':' -f2 2>/dev/null || echo "?")

# Log session end for debugging
if [ -n "$GT_DEBUG" ]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] sessionEnd: reason=$reason duration=${duration}ms" >> /tmp/gastown-hooks.log
fi

# Only run cost/sync if we're in a Gas Town context
if [ -n "$GT_ROLE" ]; then
    # Record session costs (suppress all output)
    gt costs record >/dev/null 2>&1 || true
    
    # Sync beads if bd is available (suppress all output)
    if command -v bd &>/dev/null; then
        bd sync >/dev/null 2>&1 || true
    fi
fi

# No output needed - fire and forget
