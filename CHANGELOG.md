# Changelog

All notable changes to the Cursor Gas Town project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.1] - 2026-01-22

### Fixed
- **npm installer** - Download correct `cursor-gastown_*` release archives

## [0.1.0] - 2026-01-22

Initial public release of Cursor Gas Town - a multi-agent workspace manager for Cursor CLI.

### Added

#### Cursor CLI Compatibility
- **Cursor CLI hooks** - Support for `cursor-agent` lifecycle hooks
- **Role-Model Matrix** - Default model routing by role with provider templates
- **Council orchestration** - Multi-model routing patterns with metrics + profiles

#### Core Architecture
- **Town structure** - Hierarchical workspace with rigs, crews, and polecats
- **Rig management** - `gt rig add/list/remove` for project containers
- **Crew workspaces** - `gt crew add` for persistent developer workspaces
- **Polecat workers** - Transient agent workers managed by Witness

#### Agent Roles
- **Mayor** - Global coordinator for cross-rig work
- **Deacon** - Town-level lifecycle patrol and heartbeat
- **Witness** - Per-rig polecat lifecycle manager
- **Refinery** - Merge queue processor with code review
- **Crew** - Persistent developer workspaces
- **Polecat** - Transient worker agents

#### Work Management
- **Convoy system** - `gt convoy create/list/status` for tracking related work
- **Sling workflow** - `gt sling <bead> <rig>` to assign work to agents
- **Hook mechanism** - Work attached to agent hooks for pickup
- **Molecule workflows** - Formula-based multi-step task execution

#### Communication
- **Mail system** - `gt mail inbox/send/read` for agent messaging
- **Escalation protocol** - `gt escalate` with severity levels
- **Handoff mechanism** - `gt handoff` for context-preserving session cycling

#### Integration
- **Beads integration** - Issue tracking via beads (`bd` commands)
- **Tmux sessions** - Agent sessions in tmux with theming

#### Developer Experience
- **Status dashboard** - `gt status` for town overview
- **Activity feed** - `gt feed` for real-time event stream
- **Nudge system** - `gt nudge` for reliable message delivery to sessions

#### Infrastructure
- **Daemon mode** - Background lifecycle management
- **npm package** - Cross-platform binary distribution
- **GitHub Actions** - CI/CD workflows for releases
- **GoReleaser** - Multi-platform binary builds

#### Security
- **Security policy** - Responsible disclosure guidance (`SECURITY.md`)
- **Security audit workflow** - `security-audit` formula for focused reviews

#### Deployment
- **Release process** - Documented steps in `RELEASING.md`
- **Version tooling** - `scripts/bump-version.sh` for release prep
