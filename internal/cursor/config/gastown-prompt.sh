#!/bin/bash
# Gas Town beforeSubmitPrompt hook for Cursor
#
# Called right after user hits send but before backend request.
# This hook can block submission but cannot inject context.
# Use sessionStart for context injection.
#
# Input:  {"prompt": "...", "attachments": [...]}
# Output: {"continue": true|false, "user_message": "..."}

# Read JSON input from stdin (required - must consume it)
cat > /dev/null

# Always allow the prompt to continue
# Context injection happens at sessionStart, not here
echo '{"continue": true}'
