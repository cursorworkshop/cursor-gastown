#!/bin/bash
# Gas Town stop hook for Cursor
#
# Called when the agent loop ends.
# Records session costs and syncs beads.
#
# Input:  {"status": "completed"|"aborted"|"error", "loop_count": N}
# Output: {"followup_message": "..."} - optional, triggers another turn

# Read JSON input from stdin (required - must consume it)
input=$(cat)

# Export PATH to ensure gt/bd are available
export PATH="$HOME/go/bin:$HOME/bin:$HOME/.local/bin:$PATH"

# Parse status for logging
status=$(echo "$input" | grep -o '"status":"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "unknown")

# Log stop event for debugging
if [ -n "$GT_DEBUG" ]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] stop: status=$status" >> /tmp/gastown-hooks.log
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

# Output empty JSON (no followup_message - don't auto-continue)
echo '{}'
