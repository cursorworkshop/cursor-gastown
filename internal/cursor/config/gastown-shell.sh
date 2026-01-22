#!/bin/bash
# Gas Town shell execution hooks for Cursor
#
# Usage: gastown-shell.sh [before|after]
#
# beforeShellExecution: Called before shell commands run
#   Input:  {"command": "...", "cwd": "..."}
#   Output: {"permission": "allow"|"deny"|"ask", "user_message": "...", "agent_message": "..."}
#
# afterShellExecution: Called after shell commands complete
#   Input:  {"command": "...", "output": "...", "duration": N}
#   Output: (none expected, fire-and-forget)

HOOK_PHASE="${1:-after}"

# Read JSON input from stdin (required - must consume it)
input=$(cat)

# Export PATH to ensure gt is available
export PATH="$HOME/go/bin:$HOME/bin:$HOME/.local/bin:$PATH"

case "$HOOK_PHASE" in
    before)
        # Log if debugging
        if [ -n "$GT_DEBUG" ]; then
            cmd=$(echo "$input" | grep -o '"command":"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "?")
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] beforeShell: $cmd" >> /tmp/gastown-hooks.log
        fi
        
        # Always allow shell commands
        echo '{"permission": "allow"}'
        ;;
    after)
        # Log if debugging
        if [ -n "$GT_DEBUG" ]; then
            cmd=$(echo "$input" | grep -o '"command":"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "?")
            duration=$(echo "$input" | grep -o '"duration":[0-9]*' | cut -d':' -f2 2>/dev/null || echo "?")
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] afterShell: $cmd (${duration}ms)" >> /tmp/gastown-hooks.log
        fi
        
        # No output needed for after hook
        ;;
    *)
        echo "Usage: $0 [before|after]" >&2
        exit 1
        ;;
esac
