# Cursor CLI Integration Notes

This fork of Gas Town uses Cursor CLI (`cursor-agent`) as the default agent backend.

**Repository**: [https://github.com/cursorworkshop/cursor-gastown](https://github.com/cursorworkshop/cursor-gastown)

---

## Hooks: Two Pathways

Gas Town supports two distinct execution pathways with different hook capabilities.

### Hook Support by Pathway

| Hook | CLI Pathway | IDE Pathway |
|------|-------------|-------------|
| `beforeShellExecution` | ✅ Primary | ✅ Available |
| `afterShellExecution` | ✅ Primary | ✅ Available |
| `beforeMCPExecution` | ✅ Available | ✅ Available |
| `afterMCPExecution` | ✅ Available | ✅ Available |
| `afterFileEdit` | ✅ Available | ✅ Available |
| `beforeSubmitPrompt` | ❌ | ✅ Primary |
| `stop` | ❌ | ✅ Primary |
| `afterAgentResponse` | ❌ | ✅ Available |

### CLI Pathway (`cursor-agent -p`)

For headless/automated execution. Full CLI support planned for future.

| Feature | Hook | Timing |
|---------|------|--------|
| Mail injection | `beforeShellExecution` | First command |
| Cost recording | `afterShellExecution` | Every 10 commands |
| Audit logging | `afterShellExecution` | Every command |
| Bead sync | — | Manual (`bd sync`) |

**Use for**: CI/CD, automated testing, batch operations, scripting.

### IDE Pathway (Cursor App)

For interactive development with full hook support.

| Feature | Hook | Timing |
|---------|------|--------|
| Mail injection | `beforeSubmitPrompt` | Before each prompt |
| Cost recording | `stop` | Session end |
| Bead sync | `stop` | Session end |
| Audit logging | `afterShellExecution` | Every command |

**Use for**: Production work, interactive sessions, full Gas Town features.

### Hook Files

| File | Pathway | Purpose |
|------|---------|---------|
| `gastown-prompt.sh` | IDE | Mail injection before prompt |
| `gastown-stop.sh` | IDE | Cost recording + sync on stop |
| `gastown-shell.sh` | Both | Shell hooks (CLI primary, IDE audit) |

### Future: CLI Parity

Cursor CLI hook support is expected to expand. When `beforeSubmitPrompt` and `stop` become available in CLI mode, Gas Town will automatically use them. The current CLI pathway provides functional coverage until then.

---

## Open Issue: Session Resume

### Description

Gas Town supports session resume for continuity across restarts. We need to understand how to capture and use Cursor's chat ID programmatically.

### Available Commands

```bash
cursor-agent ls              # List sessions (requires TTY)
cursor-agent resume          # Resume latest session
cursor-agent --resume [id]   # Resume specific chat
cursor-agent create-chat     # Create new chat, return ID
```

### Open Questions

1. Where does cursor-agent output the chat ID during a session?
2. How to use `cursor-agent ls` in non-TTY environments?
3. Does `cursor-agent create-chat` return a usable chat ID for `--resume`?

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
| `--resume [id]`    | Resume specific session        |

### Available Models

```
auto, opus-4.5-thinking, opus-4.5, sonnet-4.5, sonnet-4.5-thinking,
gpt-5.2, gpt-5.2-high, gpt-5.1-codex-max, gemini-3-pro, gemini-3-flash, grok
```
