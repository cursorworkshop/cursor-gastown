#!/bin/bash
# Gas Town preCompact hook for Cursor
#
# Called before context window compaction/summarization.
# This is CRITICAL for long sessions - we output a message to remind
# the agent to run `gt prime` after compaction to restore context.
#
# Input:  {"trigger": "auto"|"manual", "context_usage_percent": N, ...}
# Output: {"user_message": "..."}

# Read JSON input from stdin (required - must consume it)
input=$(cat)

# Parse trigger and context usage for logging
trigger=$(echo "$input" | grep -o '"trigger":"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "unknown")
usage=$(echo "$input" | grep -o '"context_usage_percent":[0-9]*' | cut -d':' -f2 2>/dev/null || echo "?")

# Log compaction event for debugging
if [ -n "$GT_DEBUG" ]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] preCompact: trigger=$trigger usage=$usage%" >> /tmp/gastown-hooks.log
fi

# Output message that will be shown to user/agent
# This reminds the agent to refresh context after compaction
cat << 'EOF'
{
  "user_message": "[Gas Town] Context compacting. Run `gt prime` after compaction to restore role context and check for mail."
}
EOF
