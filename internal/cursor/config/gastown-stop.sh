#!/bin/bash
# Gas Town stop hook for Cursor
# Runs when the agent loop ends to record costs and sync state
#
# This hook is called by Cursor when the agent completes or is aborted.
# It records session costs and syncs beads.

set -e

# Read JSON input from stdin (required by Cursor hooks protocol)
json_input=$(cat)

# Export PATH to ensure gt is available
export PATH="$HOME/go/bin:$HOME/bin:$HOME/.local/bin:$PATH"

# Parse the status from input
status=$(echo "$json_input" | grep -o '"status":"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "unknown")

# Only run if we're in a Gas Town context (GT_ROLE is set)
if [ -n "$GT_ROLE" ]; then
    # Record costs
    gt costs record 2>/dev/null || true
    
    # Sync beads if bd is available
    if command -v bd &> /dev/null; then
        bd sync 2>/dev/null || true
    fi
fi

# Output empty JSON (no followup_message - don't auto-continue)
echo '{}'
