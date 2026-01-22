# Cursor CLI Integration Notes

This fork of Gas Town uses Cursor CLI (`cursor-agent`) as the default agent backend.

**Repository**: [https://github.com/cursorworkshop/cursor-gastown](https://github.com/cursorworkshop/cursor-gastown)

---

## Hooks: Full CLI Support (v2026.01.17+)

As of Cursor CLI v2026.01.17, **all critical hooks are available in CLI mode**. Gas Town now
has full parity between IDE and CLI pathways.

### Hook Support

| Hook | CLI | IDE | Purpose |
|------|-----|-----|---------|
| `sessionStart` | ✅ | ✅ | Inject initial context, set env vars |
| `beforeSubmitPrompt` | ✅ | ✅ | Gate prompt submission |
| `preCompact` | ✅ | ✅ | Observe/prepare for context compaction |
| `stop` | ✅ | ✅ | Cleanup, cost recording, sync |
| `beforeShellExecution` | ✅ | ✅ | Permission gating for shell commands |
| `afterShellExecution` | ✅ | ✅ | Audit logging |
| `sessionEnd` | ✅ | ✅ | Fire-and-forget cleanup |

### Gas Town Hook Implementation

| Hook | Script | Purpose |
|------|--------|---------|
| `sessionStart` | `gastown-session-start.sh` | Inject mail via `additional_context`, set `GT_SESSION_ID` |
| `beforeSubmitPrompt` | `gastown-prompt.sh` | Allow prompt (context injected at session start) |
| `preCompact` | `gastown-precompact.sh` | Remind agent to run `gt prime` after compaction |
| `stop` | `gastown-stop.sh` | Record costs, sync beads |
| `beforeShellExecution` | `gastown-shell.sh` | Permission (always allow) |
| `afterShellExecution` | `gastown-shell.sh` | Audit logging (when `GT_DEBUG` set) |

### Hook Input/Output Schemas

**sessionStart**:
```json
// Input
{"session_id": "...", "is_background_agent": bool, "composer_mode": "agent"|"ask"|"edit"}

// Output
{"continue": true, "env": {"KEY": "value"}, "additional_context": "text to inject"}
```

**beforeSubmitPrompt**:
```json
// Input
{"prompt": "...", "attachments": [...]}

// Output
{"continue": true|false, "user_message": "shown when blocked"}
```

**preCompact**:
```json
// Input
{"trigger": "auto"|"manual", "context_usage_percent": 85, ...}

// Output
{"user_message": "shown to user/agent"}
```

**stop**:
```json
// Input
{"status": "completed"|"aborted"|"error", "loop_count": 0}

// Output
{"followup_message": "auto-submit as next prompt"} // optional
```

**beforeShellExecution**:
```json
// Input
{"command": "...", "cwd": "..."}

// Output
{"permission": "allow"|"deny"|"ask", "user_message": "...", "agent_message": "..."}
```

---

## Session ID Attribution

The `sessionStart` hook receives a `session_id` from Cursor. Gas Town:
1. Captures this in the hook
2. Sets `GT_SESSION_ID` and `CURSOR_SESSION_ID` environment variables
3. Uses it for cost attribution and session tracking

---

## Working Features

| Feature                      | Status |
| ---------------------------- | ------ |
| Default agent = cursor-agent | YES    |
| Force mode (`-f`)            | YES    |
| Tmux session management      | YES    |
| Task execution               | YES    |
| Multi-file tasks             | YES    |
| Workspace trust bypass       | YES    |
| sessionStart hook            | YES (v2026.01.17+) |
| beforeSubmitPrompt hook      | YES (v2026.01.17+) |
| preCompact hook              | YES (v2026.01.17+) |
| stop hook                    | YES (v2026.01.17+) |

---

## cursor-agent Reference

### Useful Flags

| Flag               | Description                    |
| ------------------ | ------------------------------ |
| `-f`               | Force mode (bypass trust)      |
| `-p` / `--print`   | Headless mode                  |
| `--approve-mcps`   | Auto-approve MCP servers       |
| `--workspace`      | Specify workspace directory    |
| `--output-format`  | JSON output (with `--print`)   |

### Available Models

```
auto, opus-4.5-thinking, opus-4.5, sonnet-4.5, sonnet-4.5-thinking,
gpt-5.2, gpt-5.2-high, gpt-5.1-codex-max, gemini-3-pro, gemini-3-flash, grok
```

### Version Check

```bash
cursor-agent --version
# Should show v2026.01.17 or later for full hook support
```

---

## Hook Behavior by Mode

| Hook | `-p` (headless) | `-f` (interactive) | IDE |
|------|-----------------|-------------------|-----|
| `sessionStart` | ✅ Fires | ✅ Fires | ✅ Fires |
| `beforeSubmitPrompt` | ❌ No | ✅ Fires | ✅ Fires |
| `preCompact` | ❓ Only on compaction | ❓ Only on compaction | ✅ Fires |
| `stop` | ❌ No | ✅ Fires | ✅ Fires |
| `sessionEnd` | ✅ Fires | ✅ Fires | ✅ Fires |
| `beforeShellExecution` | ✅ Fires | ✅ Fires | ✅ Fires |
| `afterShellExecution` | ✅ Fires | ✅ Fires | ✅ Fires |

**Key insight**: For headless (`-p`) mode, use `sessionStart` for initial context injection
and `sessionEnd` for cleanup. The `beforeSubmitPrompt` and `stop` hooks only fire in
interactive modes.
