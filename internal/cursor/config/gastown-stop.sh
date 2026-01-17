#!/bin/bash
# Gas Town stop hook for Cursor
#
# PATHWAY: IDE ONLY
# This hook fires in Cursor IDE when the agent session ends.
# CLI pathway uses afterShellExecution instead (see gastown-shell.sh).
#
# Purpose: Record session costs and sync beads on completion.

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
