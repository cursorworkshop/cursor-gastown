#!/bin/bash
# Gas Town beforeSubmitPrompt hook for Cursor
# Runs before each user prompt to inject mail and prime context
#
# This hook is called by Cursor before submitting a user prompt.
# It runs gt mail check --inject to inject any pending mail into the conversation.

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
