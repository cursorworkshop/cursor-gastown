# Cursor Gas Town

Multi-agent orchestration for Cursor CLI with persistent work tracking.

> This is a fork of [Gas Town](https://github.com/steveyegge/gastown), originally built for Claude Code. This repo converts it to work with Cursor CLI.

## Overview

Cursor Gas Town coordinates multiple Cursor agents working on different tasks. Work state persists in git-backed hooks, enabling reliable multi-agent workflows that survive crashes and restarts.

| Challenge | Solution |
|-----------|----------|
| Agents lose context on restart | Work persists in git-backed hooks |
| Manual agent coordination | Built-in mailboxes and handoffs |
| 4-10 agents become chaotic | Scale to 20-30 agents |
| Work state lost in memory | Stored in Beads ledger |

## Core Concepts

- **Mayor** - Your primary AI coordinator. Start here.
- **Town** - Workspace directory (e.g., `~/gt/`)
- **Rigs** - Project containers wrapping git repositories
- **Crew** - Your personal workspace within a rig
- **Polecats** - Ephemeral worker agents
- **Hooks** - Git worktree-based persistent storage
- **Convoys** - Work tracking units bundling multiple tasks
- **Beads** - Git-backed issue tracking system

## Installation

### Prerequisites

- Go 1.23+
- Git 2.25+ (for worktree support)
- beads (bd) 0.47.0+
- tmux 3.0+ (recommended)
- Cursor CLI

### Setup

```bash
# Install
go install github.com/Cursor-Workshop/cursor-gastown/cmd/gt@latest
export PATH="$PATH:$HOME/go/bin"

# Create workspace
gt install ~/gt --git
cd ~/gt

# Add a project
gt rig add myproject https://github.com/you/repo.git

# Create crew workspace
gt crew add yourname --rig myproject
cd myproject/crew/yourname

# Start Mayor
gt mayor attach
```

## Quick Start

```bash
# 1. Start the Mayor
gt mayor attach

# 2. Create a convoy
gt convoy create "Feature X" issue-123 issue-456 --notify --human

# 3. Assign work
gt sling issue-123 myproject

# 4. Track progress
gt convoy list

# 5. Monitor agents
gt agents
```

## Key Commands

### Workspace

```bash
gt install <path>              # Initialize workspace
gt rig add <name> <repo>       # Add project
gt rig list                    # List projects
gt crew add <name> --rig <rig> # Create crew workspace
```

### Agents

```bash
gt agents                      # List active agents
gt sling <issue> <rig>         # Assign work to agent
gt mayor attach                # Start Mayor session
gt prime                       # Alternative to mayor attach
```

### Convoys

```bash
gt convoy create <name> [issues...]  # Create convoy
gt convoy list                       # List all convoys
gt convoy show [id]                  # Show details
gt convoy add-issue <issue>          # Add issue
```

### Beads

```bash
bd formula list          # List formulas
bd cook <formula>        # Execute formula
bd mol pour <formula>    # Create trackable instance
bd mol list              # List active instances
```

## Dashboard

```bash
gt dashboard --port 8080
open http://localhost:8080
```

## Shell Completions

```bash
# Bash
gt completion bash > /etc/bash_completion.d/gt

# Zsh
gt completion zsh > "${fpath[1]}/_gt"

# Fish
gt completion fish > ~/.config/fish/completions/gt.fish
```

## Troubleshooting

### Agents lose connection

```bash
gt hooks list
gt hooks repair
```

### Convoy stuck

```bash
gt convoy refresh <convoy-id>
```

### Mayor not responding

```bash
gt mayor detach
gt mayor attach
```

## Origins

Gas Town was created by Steve Yegge for orchestrating Claude Code agents. This fork adapts it for Cursor CLI, maintaining the same architecture while targeting the Cursor ecosystem.

## License

MIT
