#!/bin/bash
# Gas Town afterShellExecution hook for Cursor
# Logs shell commands for auditing (optional)
#
# This hook is called by Cursor after executing any shell command.
# It can be used for auditing and debugging.

# Read JSON input from stdin
json_input=$(cat)

# Only log if GT_DEBUG is set
if [ -n "$GT_DEBUG" ]; then
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] $json_input" >> /tmp/gastown-audit.log
fi

# Exit successfully (no output needed for afterShellExecution)
exit 0
