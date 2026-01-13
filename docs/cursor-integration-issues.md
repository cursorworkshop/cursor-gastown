# Cursor CLI Integration Issues

This document outlines issues discovered while integrating [Gas Town](https://github.com/steveyegge/gastown) with Cursor CLI. These issues require investigation or input from the Cursor team.

## Context

Gas Town is a multi-agent workspace manager that orchestrates AI coding agents. We've successfully integrated Cursor CLI as an agent backend alongside Claude Code, but encountered several issues specific to Cursor.

**Repository**: [https://github.com/okc0mputex/gastown-cursor-cli](https://github.com/okc0mputex/gastown-cursor-cli) (fork with Cursor integration)

---

## Issue 1: Nudge Message Flooding in UI

### Description

When sending a single startup message to a Cursor agent session via tmux, the message appears duplicated 15-20+ times in the Cursor UI as separate conversation bubbles.

### Expected Behavior

A single message sent via `tmux send-keys` should appear once in the conversation.

### Actual Behavior

The same message is displayed multiple times:

```
 ┌────────────────────────────────────────────────────────────────────────────┐
 │ [GAS TOWN] gastown_cli/crew/max <- human • 2026-01-07T21:56 • start        │
 └────────────────────────────────────────────────────────────────────────────┘
 ┌────────────────────────────────────────────────────────────────────────────┐
 │ [GAS TOWN] gastown_cli/crew/max <- human • 2026-01-07T21:56 • start        │
 └────────────────────────────────────────────────────────────────────────────┘
 (... repeated 15-20 more times ...)
```

### Steps to Reproduce

1. Start cursor-agent in a tmux session:
  ```bash
   tmux new-session -d -s test-session
   tmux send-keys -t test-session 'cursor-agent -f' Enter
  ```
2. Wait for cursor-agent to initialize and trust the workspace
3. Send a message:
  ```bash
   tmux send-keys -t test-session -l '[GAS TOWN] test message'
   sleep 0.5
   tmux send-keys -t test-session Enter
  ```
4. Observe multiple duplicate message bubbles in the UI

### Environment

- cursor-agent version: 2026.01.09-231024f (latest as of 2026-01-12)
- macOS 15 (darwin 25.2.0)
- tmux 3.6a

### Questions for Cursor Team

1. Is this expected behavior when receiving input via tmux?
2. Is there a recommended way to inject messages programmatically?
3. Does cursor-agent have an API or IPC mechanism for receiving external messages?

---

## Issue 2: Workspace Trust Blocks Automation

### Description

When cursor-agent starts in a new workspace, it displays a trust dialog that requires manual user interaction. This blocks autonomous agent spawning in multi-agent workflows.

### Expected Behavior

Ability to pre-authorize workspaces or skip the trust dialog for automated/headless use cases.

### Actual Behavior

```
  ╭──────────────────────────────────────────────────────────────────────────╮
  │                                                                          │
  │  ⚠ Workspace Trust Required                                              │
  │                                                                          │
  │  Cursor Agent can execute code and access files in your workspace. You   │
  │  will also trust the MCP servers this workspace has enabled.             │
  │                                                                          │
  │  Do you want to mark this workspace as trusted?                          │
  │                                                                          │
  │    /Users/user/project                                                   │
  │                                                                          │
  │  ▶ [a] Trust this workspace                                              │
  │    [w] Trust this workspace, but don't enable all MCP servers            │
  │    [q] Quit                                                              │
  │                                                                          │
  ╰──────────────────────────────────────────────────────────────────────────╯
```

### Current Workaround

We send `tmux send-keys -t session 'a'` to accept, but this:

- Requires timing coordination
- May fail if the dialog hasn't rendered yet
- Doesn't work for fully autonomous scenarios

### Potential Solutions (As of 2026-01-08)

**Available flags that may help:**

- `--approve-mcps` - Automatically approve MCP servers (but only in headless/`--print` mode)
- `--print` / `-p` - Headless mode for non-interactive use
- `--api-key` - API key authentication for automation

**Note:** Web search suggests a `--trust-workspace` flag may exist in newer versions, but it's not present in v2026.01.02-80e4d9b. Need to verify with Cursor team.

### Questions for Cursor Team

1. Is there a configuration file to pre-trust workspaces (e.g., `~/.cursor/trusted-workspaces.json`)?
2. Does `--approve-mcps` work outside of headless mode?
3. Is there a `--trust-workspace` flag or equivalent? (Web docs mention it but not in current CLI help)
4. Can trust be inherited from parent directories?

---

## Issue 3: Session Resume Chat ID

### Description

Gas Town supports session resume for continuity across restarts. Claude Code outputs a session ID that can be passed to `--resume`. We need to understand how to capture and use Cursor's chat ID for the same purpose.

### New Findings (As of 2026-01-08)

cursor-agent v2026.01.02-80e4d9b now has session management commands:

```bash
# List previous sessions
cursor-agent ls

# Resume latest session
cursor-agent resume

# Resume specific chat
cursor-agent --resume [chatId]

# Create new chat and get its ID
cursor-agent create-chat
```

**Note:** `cursor-agent ls` requires a TTY and fails in non-interactive environments with "Raw mode is not supported" error.

### Remaining Questions for Cursor Team

1. Where does cursor-agent output the chat ID during a session? (for capture)
2. What's the format of the chat ID?
3. How to use `cursor-agent ls` in non-TTY environments (e.g., capture session list programmatically)?
4. Does `cursor-agent create-chat` return a usable chat ID for `--resume`?

---

## Issue 4: Process Detection for Session Health ✅ RESOLVED

### Description

Gas Town monitors agent health by checking the tmux pane's current command. For Claude Code, we check for "node" process. We need to know what process name cursor-agent reports.

### Resolution (As of 2026-01-08)

**Finding:** cursor-agent runs as a Node.js process, so tmux reports it as `node`, same as Claude Code.

```bash
$ tmux list-panes -t gt-gastown_cli-smoke-test -F '#{pane_current_command}'
node
```

### Updated Configuration

Gas Town now correctly detects cursor-agent sessions using:

```go
ProcessNames: []string{"cursor-agent", "node"}  // cursor-agent shows as "node" in tmux
```

**Note:** Both Claude Code and cursor-agent appear as `node` processes, which simplifies detection but means we can't distinguish between them by process name alone.

---

## Feature Requests

### 1. Programmatic Message Injection API

For multi-agent orchestration, we need to send messages to running agent sessions. An API or IPC mechanism would be more reliable than tmux keystroke injection.

**Proposed solutions:**

- Unix socket for receiving messages
- Named pipe support
- HTTP endpoint for local communication
- Environment variable for message injection path

### 2. Headless Trust Mode

For CI/CD and automated workflows, a way to run cursor-agent with pre-approved trust would be valuable.

**Current status:** `--print` mode enables headless operation, `--approve-mcps` handles MCP approval, but workspace trust still requires interaction in interactive mode.

**Proposed solutions:**

- `--trust` or `--no-trust-prompt` flag (mentioned in web docs but not in current CLI)
- Configuration file for trusted paths
- Environment variable `CURSOR_TRUST_WORKSPACE=1`

### 3. Session ID Output

Standardized way to capture session/chat ID for resume functionality.

**Current status:** `cursor-agent create-chat` can create a chat and return its ID. Need to verify this works with `--resume`.

**Proposed solutions:**

- Output chat ID to stderr on startup
- Write to a file specified by `--session-file`
- Environment variable for session ID path

---

## New cursor-agent Features (v2026.01.09-231024f)

The following features are available in cursor-agent that may help with Gas Town integration:


| Feature            | Command/Flag               | Description                           |
| ------------------ | -------------------------- | ------------------------------------- |
| Headless mode      | `--print` / `-p`           | Non-interactive scripting mode        |
| Auto-approve MCPs  | `--approve-mcps`           | Skip MCP approval (headless only)     |
| API authentication | `--api-key <key>`          | Programmatic authentication           |
| Session list       | `cursor-agent ls`          | List previous sessions (requires TTY) |
| Resume session     | `cursor-agent resume`      | Resume latest session                 |
| Create chat        | `cursor-agent create-chat` | Create new chat, return ID            |
| MCP management     | `cursor-agent mcp list`    | List configured MCP servers           |
| Custom workspace   | `--workspace <path>`       | Specify workspace directory           |
| Output formats     | `--output-format json`     | JSON output (with `--print`)          |
| List models        | `--list-models` / `models` | Show available AI models              |
| Generate rules     | `generate-rule` / `rule`   | Interactive Cursor rule generation    |


### MCP Subcommands

```bash
cursor-agent mcp login <identifier>      # Auth with MCP server
cursor-agent mcp list                    # List MCP servers
cursor-agent mcp list-tools <identifier> # List tools for an MCP
cursor-agent mcp enable <identifier>     # Add MCP to approved list (NEW)
cursor-agent mcp disable <identifier>    # Remove MCP from approved list
```

### Available Models (as of v2026.01.09)

```
auto                    - Auto
opus-4.5-thinking       - Claude 4.5 Opus (Thinking) [default]
opus-4.5                - Claude 4.5 Opus
sonnet-4.5              - Claude 4.5 Sonnet
sonnet-4.5-thinking     - Claude 4.5 Sonnet (Thinking)
gpt-5.2                 - GPT-5.2
gpt-5.2-high            - GPT-5.2 High
gpt-5.1-codex-max       - GPT-5.1 Codex Max
gemini-3-pro            - Gemini 3 Pro
gemini-3-flash          - Gemini 3 Flash
grok                    - Grok
```

---

## Contact

For questions about this integration or to discuss solutions:

- Gas Town Repository: [https://github.com/steveyegge/gastown](https://github.com/steveyegge/gastown)
- Integration Fork: [https://github.com/okc0mputex/gastown-cursor-cli](https://github.com/okc0mputex/gastown-cursor-cli)
- Related PR: [https://github.com/steveyegge/gastown/pull/247](https://github.com/steveyegge/gastown/pull/247)

---

## Appendix: Gas Town + Cursor Integration Status

### Working Features


| Feature                        | Status |
| ------------------------------ | ------ |
| Agent preset configuration     | ✅      |
| Hooks via `.cursor/hooks.json` | ✅      |
| Rules via `.cursor/rules/`     | ✅      |
| Session start/stop             | ✅      |
| Force mode (`-f`)              | ✅      |
| Tmux session management        | ✅      |


### Hooks Implementation

We've implemented Cursor hooks for Gas Town lifecycle events:

```json
{
  "version": 1,
  "hooks": {
    "beforeSubmitPrompt": [{"command": ".cursor/hooks/gastown-prompt.sh"}],
    "stop": [{"command": ".cursor/hooks/gastown-stop.sh"}],
    "afterShellExecution": [{"command": ".cursor/hooks/gastown-audit.sh"}]
  }
}
```

These hooks run Gas Town commands (`gt mail check`, `gt costs record`) at appropriate lifecycle points.

---

## Appendix: Beads Configuration Workaround

### Issue: valid_issue_types Not Respected During SQLite Import

When initializing a beads database with `bd init --from-jsonl`, the `valid_issue_types` config stored in the database is not applied during import validation. This causes import to fail with:

```
Import failed: error creating depth-0 issues: validation failed for issue 862: invalid issue type: agent
```

Gas Town uses custom issue types (`agent`, `role`, `rig`, `convoy`, `event`) that require `valid_issue_types` configuration.

### Workaround

Use `no-db: true` in `.beads/config.yaml` to operate in JSONL-only mode:

```yaml
# Use no-db mode: load from JSONL, no SQLite
# NOTE: Enabled due to beads bug where valid_issue_types config isn't respected
# during SQLite import. JSONL mode works correctly.
no-db: true

# Valid issue types for this repository (required for Gas Town agent beads)
valid-issue-types:
  - bug
  - feature
  - task
  - epic
  - chore
  - merge-request
  - molecule
  - gate
  - agent
  - role
  - rig
  - convoy
  - event

# Issue prefix for this repository
issue-prefix: gt
```

This is a beads issue, not a Cursor issue, and should be reported to the beads maintainers.