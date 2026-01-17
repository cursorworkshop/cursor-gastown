#!/bin/bash
# Gas Town beforeSubmitPrompt hook for Cursor
#
# PATHWAY: IDE ONLY
# This hook fires in Cursor IDE before each user prompt.
# CLI pathway uses beforeShellExecution instead (see gastown-shell.sh).
#
# Purpose: Inject pending mail into conversation context.

set -e

# Read JSON input from stdin (required by Cursor hooks protocol)
json_input=$(cat)

# Export PATH to ensure gt is available
export PATH="$HOME/go/bin:$HOME/bin:$HOME/.local/bin:$PATH"

# Only run if we're in a Gas Town context (GT_ROLE is set)
if [ -n "$GT_ROLE" ]; then
    # Check for mail and inject into context
    # Run in background to not block the prompt
    gt mail check --inject 2>/dev/null &
fi

# Always allow the prompt to continue
cat << 'EOF'
{
  "continue": true
}
EOF
