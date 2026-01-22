#!/bin/bash
# Gas Town sessionStart hook for Cursor CLI
#
# Called when a new session starts. Uses additional_context to inject:
# - Session ID for attribution
# - Pending mail messages
# - Role context
#
# Input:  {"session_id": "...", "is_background_agent": bool, "composer_mode": "..."}
# Output: {"continue": true, "additional_context": "...", "env": {...}}

# Read JSON input from stdin
input=$(cat)

# Export PATH to ensure gt/bd are available
export PATH="$HOME/go/bin:$HOME/bin:$HOME/.local/bin:$PATH"

# Parse session_id from input (handle JSON with spaces)
# Match pattern: "session_id": "value" or "session_id":"value"
session_id=$(echo "$input" | sed -n 's/.*"session_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')

# Build context to inject
context=""

# Only inject context if we're in a Gas Town workspace (GT_ROLE set or detectable)
if [ -n "$GT_ROLE" ] || command -v gt &>/dev/null; then
    # Capture mail check output (suppress stderr)
    mail_output=$(gt mail check --inject 2>/dev/null || true)
    if [ -n "$mail_output" ]; then
        context="$mail_output"
    fi
fi

# Escape context for JSON (handle newlines, quotes, backslashes)
escape_json() {
    local str="$1"
    # Escape backslashes first, then quotes, then convert newlines
    printf '%s' "$str" | sed 's/\\/\\\\/g; s/"/\\"/g' | awk '{printf "%s\\n", $0}' | sed 's/\\n$//'
}

escaped_context=$(escape_json "$context")

# Build output JSON
if [ -n "$session_id" ]; then
    cat << EOF
{
  "continue": true,
  "env": {
    "GT_SESSION_ID": "$session_id",
    "CURSOR_SESSION_ID": "$session_id"
  },
  "additional_context": "$escaped_context"
}
EOF
else
    cat << EOF
{
  "continue": true,
  "additional_context": "$escaped_context"
}
EOF
fi
