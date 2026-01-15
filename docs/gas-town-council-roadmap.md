# Gas Town Council Roadmap

Multi-model orchestration for Gas Town using Cursor CLI.

## Vision

Transform Gas Town from a Claude Code-only orchestrator into **Gas Town Council** - a multi-model orchestration system built on Cursor CLI. The key insight: match models to roles for better code review (different model families catch different bugs), cost optimization (not every task needs Opus-level intelligence), and vendor resilience.

## Why Multi-Model?

### 1. Perspective Diversity ("Second Opinion" Pattern)
Different models have different blindspots. A GPT-5.2 reviewer catches bugs Claude missed, and vice versa. This is especially valuable for the Refinery role.

### 2. Cost Optimization
Not every task needs flagship model intelligence:

| Role     | Opus 4.5 Cost | Optimized Model | Savings |
|----------|---------------|-----------------|---------|
| Witness  | $3.75/hr      | Gemini Flash $0.30 | 92%  |
| Polecat  | $15.00/hr     | Sonnet $3.00   | 80%     |
| Dogs     | $7.50/hr      | Gemini Flash $0.10 | 99%  |

### 3. Vendor Resilience
When one provider hits rate limits or has an outage, work continues on other providers. Single-model systems halt entirely.

## Role-Model Matrix

| Role | Model | Rationale |
|------|-------|-----------|
| Mayor | opus-4.5-thinking | Strategic coordination requires sustained reasoning |
| Polecats (Complex) | sonnet-4.5 | Best coding model for multi-file tasks |
| Polecats (Routine) | gpt-5.2 / gemini-3-flash | Cost-effective for well-defined tasks |
| Refinery | gpt-5.2-high | Different model family = fresh perspective |
| Witness | gemini-3-flash | Fast, cheap monitoring |
| Deacon | gemini-3-flash | Lightweight lifecycle management |
| Crew | auto | User preference for interactive work |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Council Layer (NEW)                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Role Config │  │Model Router │  │  Cursor Adapter     │  │
│  │ council.toml│  │ complexity  │  │  session mgmt       │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────┼───────────────────────────────┐
│                    Gas Town Core                             │
│  Mayor  │  Witness  │  Refinery  │  Polecats  │  Convoys    │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────┼───────────────────────────────┐
│                    Cursor CLI                                │
│  cursor-agent  │  Hooks System  │  Session Management        │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Foundation - Cursor CLI Adapter ✅

**Status**: Complete

- `internal/cursor/session.go` - Session capture and resume
- `internal/cursor/adapter.go` - Gas Town to Cursor CLI translation
- `internal/cursor/hooks.go` - Hook installation

### Phase 2: Model Router - The Council's Brain ✅

**Status**: Complete

- `internal/council/config.go` - Configuration parser (TOML/JSON)
- `internal/council/router.go` - Model routing with complexity analysis
- `internal/council/fallback.go` - Provider fallback and circuit breaker
- `internal/cmd/council.go` - CLI commands (`gt council *`)

### Phase 3: Role-Specific Prompting ✅

**Status**: Complete

- `internal/templates/roles/refinery-openai.md.tmpl` - Structured format for GPT code review
- `internal/templates/roles/polecat-openai.md.tmpl` - Decision matrix format for GPT workers
- `internal/templates/roles/witness-google.md.tmpl` - Explicit grounding for Gemini monitoring
- `internal/templates/templates.go` - Provider-aware template selection
- `gt council templates` - Show available provider templates

**Key insight**: Different models respond better to different prompt styles:
- OpenAI (GPT): Structured formats, decision matrices, explicit constraints
- Google (Gemini): Grounded context, step-by-step explanations
- Anthropic (Claude): Conversational style (uses default templates)

### Phase 4: Observability & Analytics ✅

**Status**: Complete

- `internal/council/metrics.go` - Metrics collection and storage
- `gt council stats` - Show model performance statistics
- `gt council compare <model1> <model2>` - Compare two models

**Metrics tracked**:
- Task counts by role/model/provider
- Success rates and failure counts
- Duration and cost per task
- Rate limit hits per provider
- Cost savings vs single-model baseline

### Phase 5: Advanced Patterns ✅

**Status**: Complete

- `internal/council/patterns.go` - Pattern implementations
- `gt council chains` - List chain patterns
- `gt council ensembles` - List ensemble patterns  
- `gt council pattern <name>` - Show pattern details

**Chain-of-Models**:
- Sequential model execution with output transformation
- Predefined chains: `code-review`, `architecture`, `bug-fix`
- Support for context passing and error handling

**Ensemble Voting**:
- Parallel model execution with voting
- Strategies: majority, consensus, weighted, best
- Predefined ensembles: `critical-decision`, `fast-consensus`, `quality`

### Phase 6: Community & Ecosystem (Planned)

**Goal**: Make adoption easy.

Deliverables:
- Config sharing (`gt council config export/import`)
- Community profile repository
- Tutorial content

## Configuration

The Council is configured via `.beads/council.toml`:

```toml
version = 1

[roles.mayor]
model = "opus-4.5-thinking"
fallback = ["sonnet-4.5", "gpt-5.2-high"]
rationale = "Strategic coordination requires sustained reasoning"

[roles.polecat]
model = "sonnet-4.5"
fallback = ["gpt-5.2", "gemini-3-flash"]
rationale = "Best coding model for multi-file tasks"
complexity_routing = true

[roles.polecat.complexity]
high = "opus-4.5"
medium = "sonnet-4.5"
low = "gemini-3-flash"

[roles.refinery]
model = "gpt-5.2-high"
fallback = ["opus-4.5", "sonnet-4.5"]
rationale = "Different model family provides fresh perspective"

[roles.witness]
model = "gemini-3-flash"
fallback = ["sonnet-4.5"]
rationale = "Fast, cost-effective monitoring"

[defaults]
model = "sonnet-4.5"
fallback = ["gpt-5.2", "gemini-3-flash"]

[providers.anthropic]
enabled = true
priority = 100
rate_limit = 60

[providers.openai]
enabled = true
priority = 90
rate_limit = 60

[providers.google]
enabled = true
priority = 80
rate_limit = 60
```

## CLI Commands

```bash
# Show configuration
gt council show
gt council show --json

# View/set role models
gt council role mayor
gt council set mayor opus-4.5-thinking
gt council fallback mayor sonnet-4.5 gpt-5.2

# Test routing
gt council route polecat
gt council route polecat --complexity high

# Provider status
gt council providers

# Initialize config
gt council init
gt council init --force
```

## Success Metrics

### Phase 1-2 (Foundation)
- [x] Run sessions with Cursor CLI
- [x] Different roles can use different models
- [ ] Handoff works reliably across models

### Phase 3-4 (Refinement)
- [ ] Documented cost savings vs single-model (target: 40%+)
- [ ] Measured quality improvements from model diversity
- [ ] Dashboard showing real-time model performance

### Phase 5-6 (Community)
- [ ] 25+ external users
- [ ] 5+ community-contributed model profiles
- [ ] Workshop material incorporating Gas Town Council

## Related Documents

- [Cursor Integration Issues](cursor-integration-issues.md)
- [Understanding Gas Town](understanding-gas-town.md)
- [Propulsion Principle](propulsion-principle.md)

## References

- [Cursor CLI Documentation](https://cursor.com/docs/cli/overview)
- [Cursor Hooks](https://cursor.com/docs/agent/hooks)
- [Gas Town Council Design Doc](Multi-Model%20Orchestration%20for%20Cursor%20CLI.pdf)
