# Cursor Gas Town

Multi-agent orchestration for Cursor CLI with persistent work tracking.

> This is a fork of [Gas Town](https://github.com/steveyegge/gastown), converted to work with Cursor CLI.

## Overview

Cursor Gas Town coordinates multiple Cursor agents working on different tasks. Work state persists in git-backed hooks, enabling reliable multi-agent workflows that survive crashes and restarts.

| Challenge | Solution |
|-----------|----------|
| Agents lose context on restart | Work persists in git-backed hooks |
| Manual agent coordination | Built-in mailboxes and handoffs |
| 4-10 agents become chaotic | Scale to 20-30 agents |
| Work state lost in memory | Stored in Beads ledger |

## Default Model Selection (Role Matrix)

Gas Town routes work by role using these default models:

| Role | Default Model | Rationale |
|------|---------------|-----------|
| Mayor | opus-4.5-thinking | Strategic coordination |
| Polecat | sonnet-4.5 | Best coding model |
| Refinery | gpt-5.2-high | Different perspective |
| Witness | gemini-3-flash | Fast, cheap monitoring |

## Core Concepts

- **Mayor** - Your primary AI coordinator. Start here.
- **Town** - Workspace directory (e.g., `~/gt/`)
- **Rigs** - Project containers wrapping git repositories
- **Crew** - Your personal workspace within a rig
- **Polecats** - Ephemeral worker agents
- **Hooks** - Git worktree-based persistent storage
- **Convoys** - Work tracking units bundling multiple tasks
- **Beads** - Git-backed issue tracking system
- **Formulas** - Reusable workflow templates (design → implement → test → submit)

## Installation

### Prerequisites

- Go 1.23+
- Git 2.25+ (for worktree support)
- beads (bd) 0.47.0+
- tmux 3.0+ (recommended)
- Cursor CLI

### Setup

```bash
# Install Gas Town and Beads
go install github.com/cursorworkshop/cursor-gastown/cmd/gt@latest
go install github.com/steveyegge/beads/cmd/bd@latest
export PATH="$PATH:$HOME/go/bin"

# Create workspace
gt install ~/gt --git
cd ~/gt

# Add a project
gt rig add myproject https://github.com/you/repo.git

# Create crew workspace
gt crew add yourname --rig myproject

# Verify setup
gt doctor
```

---

## End-to-End Workflow: From Plan to Done

This is the core workflow. You have work to do—here's how Gas Town handles it.

### Step 1: Define Your Work as Beads

A **bead** is a trackable unit of work (like a GitHub issue, but git-backed).

```bash
# Navigate to your rig
cd ~/gt/myproject

# Create beads for your tasks
bd create "Add user authentication"           # Creates gt-abc
bd create "Fix login page styling"            # Creates gt-def
bd create "Write integration tests"           # Creates gt-ghi

# See your beads
bd list
```

**From a plan file?** If you have a `plan.md` with tasks, create a bead for each:

```bash
# For each task in your plan:
bd create "Task 1 from plan"
bd create "Task 2 from plan"
# etc.
```

### Step 2: Group Work into a Convoy

A **convoy** tracks related work and notifies you when it's done.

```bash
# Create a convoy tracking your beads
gt convoy create "Auth Feature" gt-abc gt-def gt-ghi --notify

# Output: Created convoy hq-cv-xyz tracking 3 issues
```

### Step 3: Assign Work to Agents

Use `gt sling` to assign beads to agents. This is **the** command for dispatching work.

```bash
# Assign to a rig (auto-spawns a polecat worker)
gt sling gt-abc myproject

# Assign multiple beads (each gets its own worker)
gt sling gt-abc gt-def gt-ghi myproject

# Assign to a specific worker
gt sling gt-abc myproject/polecats/Toast
```

### Step 4: Monitor Progress

```bash
# See all active convoys
gt convoy list

# Check convoy status
gt convoy status hq-cv-xyz

# See active agents
gt agents

# Attach to Mayor for coordination
gt mayor attach
```

### Step 5: Work Completes Automatically

When agents finish:
1. Bead status changes to `closed`
2. Convoy auto-closes when all tracked beads complete
3. You get notified (if `--notify` was set)

```bash
# See completed convoys
gt convoy list --all
```

---

## Alternative: Using Formulas (Structured Workflows)

Instead of raw beads, use **formulas** for multi-step workflows with dependencies.

```bash
# List available formulas
bd formula list

# Use the "shiny" formula (design → implement → review → test → submit)
gt sling shiny --var feature="User authentication" myproject

# The formula creates beads with proper dependencies
# Each step waits for its prerequisites
```

**Built-in formulas:**
- `shiny` - Full feature workflow (design, implement, review, test, submit)
- `code-review` - Review existing code
- `security-audit` - Security-focused review
- `design` - Design-only phase

---

## Quick Reference

| You want to... | Command |
|----------------|---------|
| Create a task | `bd create "Task title"` |
| See all tasks | `bd list` |
| Group tasks for tracking | `gt convoy create "Name" bead-1 bead-2` |
| Assign work to agent | `gt sling bead-id rigname` |
| See what's in flight | `gt convoy list` |
| See active agents | `gt agents` |
| Start the coordinator | `gt mayor attach` |
| Health check | `gt doctor` |

## Key Commands

### Workspace Setup

```bash
gt install <path>              # Initialize workspace
gt rig add <name> <repo>       # Add project
gt rig list                    # List projects
gt crew add <name> --rig <rig> # Create crew workspace
gt doctor                      # Health check
```

### Work Management (Beads)

```bash
bd create "Task title"         # Create a bead (task)
bd list                        # List all beads
bd show <bead-id>              # Show bead details
bd close <bead-id>             # Mark complete
```

### Work Assignment

```bash
gt sling <bead> <rig>          # Assign work (spawns worker)
gt sling <bead> <rig>/<worker> # Assign to specific worker
gt sling <formula> <rig>       # Run formula workflow
```

### Tracking & Monitoring

```bash
gt convoy create <name> [beads...]  # Group beads for tracking
gt convoy list                      # See active convoys
gt convoy status <id>               # Convoy details
gt agents                           # List active agents
```

### Agents

```bash
gt mayor attach                # Start Mayor (coordinator)
gt prime                       # Alternative to mayor attach
gt witness attach <rig>        # Attach to rig monitor
```

### Formulas

```bash
bd formula list                # List available formulas
gt sling <formula> --var key=val <rig>  # Run formula with variables
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

### General issues

```bash
gt doctor --fix    # Auto-repair common problems
gt doctor --verbose  # Detailed diagnostics
```

## Learn More

- [Understanding Gas Town](docs/understanding-gas-town.md) - Architecture deep dive
- [Installation Guide](docs/INSTALLING.md) - Detailed setup instructions
- [Reference](docs/reference.md) - Full command reference

## Origins

Gas Town was created by Steve Yegge for orchestrating AI coding agents. This fork adapts it for Cursor CLI, maintaining the same architecture while targeting the Cursor ecosystem.

## License

MIT
