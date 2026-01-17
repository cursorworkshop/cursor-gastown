#!/bin/bash
# Gas Town shell execution hook for Cursor
#
# Usage: gastown-shell.sh [before|after]
#
# TWO PATHWAYS:
#
# CLI PATHWAY (cursor-agent -p):
#   - beforeShellExecution: Mail injection, session setup
#   - afterShellExecution: Audit logging, cost recording
#   - No beforeSubmitPrompt or stop hooks available
#
# IDE PATHWAY (Cursor App):
#   - beforeSubmitPrompt: Mail injection (gastown-prompt.sh)
#   - stop: Cost recording, bead sync (gastown-stop.sh)
#   - afterShellExecution: Audit logging only (this script)
#
# This script handles shell hooks for BOTH pathways with clear behavior.

HOOK_PHASE="${1:-after}"

# Read JSON input from stdin (required by Cursor hooks protocol)
json_input=$(cat)

# Export PATH to ensure gt is available
export PATH="$HOME/go/bin:$HOME/bin:$HOME/.local/bin:$PATH"

# Session state directory
STATE_DIR="/tmp/gastown-session-${GT_SESSION_ID:-$$}"

#--- BEFORE SHELL EXECUTION ---#
handle_before() {
    # Skip if not in Gas Town context
    if [ -z "$GT_ROLE" ]; then
        output_permission
        return
    fi

    # CLI PATHWAY: Mail injection on first command
    # (IDE uses beforeSubmitPrompt instead)
    if [ ! -f "$STATE_DIR/mail-checked" ]; then
        mkdir -p "$STATE_DIR"
        touch "$STATE_DIR/mail-checked"
        gt mail check --inject 2>/dev/null &
    fi

    output_permission
}

#--- AFTER SHELL EXECUTION ---#
handle_after() {
    # Skip if not in Gas Town context
    if [ -z "$GT_ROLE" ]; then
        exit 0
    fi

    # BOTH PATHWAYS: Audit logging (when GT_DEBUG set)
    if [ -n "$GT_DEBUG" ]; then
        timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        echo "[$timestamp] $json_input" >> /tmp/gastown-audit.log
    fi

    # CLI PATHWAY: Periodic cost recording
    # (IDE uses stop hook instead)
    mkdir -p "$STATE_DIR"
    count=$(cat "$STATE_DIR/cmd-count" 2>/dev/null || echo "0")
    count=$((count + 1))
    echo "$count" > "$STATE_DIR/cmd-count"
    
    # Record costs every 10 commands in CLI mode
    if [ $((count % 10)) -eq 0 ]; then
        gt costs record 2>/dev/null &
    fi

    exit 0
}

#--- OUTPUT HELPERS ---#
output_permission() {
    cat << 'EOF'
{
  "permission": "allow"
}
EOF
}

#--- MAIN ---#
case "$HOOK_PHASE" in
    before)
        handle_before
        ;;
    after)
        handle_after
        ;;
    *)
        echo "Usage: $0 [before|after]" >&2
        exit 1
        ;;
esac
