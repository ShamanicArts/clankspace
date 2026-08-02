---
type: strategy
status: active
summary: Delivery roadmap for a real ClankSpace pilot and production deployment.
note_created: 2026-08-02
updated: 2026-08-02
---

# ClankSpace Implementation Plan

## Overview

Build the smallest trustworthy ambient coordination layer that lets two humans and their many agents move quickly without requiring constant direct synchronization. The first product must recreate the original failure mode: one agent should surface another maintainer’s concurrent trajectory and rationale before making a conflicting change.

## Architecture principles

1. Accrued intent is advisory evidence, not authority.
2. Every write is attributable to principal, agent, run, and execution context.
3. One workspace hosts many project spaces.
4. Project-scoped retrieval is default; cross-project traversal is explicit.
5. One Go binary and SQLite database remain deployable anywhere.
6. The skill controls writing discipline; the server enforces bounds, identity, idempotency, and privacy-critical exclusions.

## Phase overview

| Phase | Focus | Status | Detailed plan |
|---|---|---|---|
| 0 | Project setup | **Complete** | This document |
| 1 | Core coordination loop | **In progress** | [phase-1-core-loop.md](phases/phase-1-core-loop.md) |
| 2 | Human board and public GitHub evidence | Planned | [phase-2-board-github.md](phases/phase-2-board-github.md) |
| 3 | Railway production pilot | Planned | Create after local acceptance |
| 4 | Private repositories and hardened remote MCP | Deferred | Prove public pilot first |

## Phase 0: Project setup — complete

- [x] Repository structure and Git initialization
- [x] Lean agent instructions and compatibility pointer
- [x] Current design specification
- [x] Implementation plan and first phase documents
- [x] Knowledge base and portable project skill
- [x] Initial devlog and state tracking

## Phase 1: Core coordination loop — in progress

Implement workspaces/projects, principals/agents/runs, notes, trajectories, transactional event receipts, briefs, coordination warnings, CLI, and stdio MCP.

Success means the original Shuv/Shamanic scenario is represented in fixtures and a conflicting request produces a useful advisory warning without treating older context as an instruction.

## Phase 2: Human board and public GitHub evidence — planned

Build the workspace/project board, runtime-provenance views, repository attachment, public PR refresh, and visual conflict inspection.

## Phase 3: Railway production pilot — planned

Connect the private repository to Railway, mount `/data`, configure `clank.shamanicarts.dev`, create the workspace and invitations, enable volume backups, and complete a portable restore drill.

## Design decisions log

### D1 — Name

**Decision:** Product is ClankSpace; executable is `clank`.

**Rationale:** Shorter, cleaner, and avoids unintended social baggage around “Clanker.”

### D2 — Memory semantics

**Decision:** Notes record accrued intent and rationale at a moment in time. They are advisory and correctable, not canonical decisions.

**Rationale:** Their purpose is to create coordination moments, not make yesterday’s agent output override today’s human.

### D3 — Portable integration

**Decision:** CLI + stdio MCP + skill is the universal integration.

**Rationale:** Every harness can supply what provenance it knows and leave unavailable fields explicit rather than requiring provider-specific core code.

### D4 — Runtime provenance

**Decision:** Store harness/model/role/parent/automation/permission/Git context with each run; notes reference runs.

**Rationale:** Bad autonomous decisions must be diagnosable by the environment that produced them.

### D5 — Professional content boundary

**Decision:** Paraphrase only the minimum project-relevant implication; exclude raw quotes and personal/emotional/private material.

**Rationale:** ClankSpace is a professional coordination surface, not surveillance or transcript storage.

### D6 — GitHub scope

**Decision:** Link public repositories and PRs read-only without authentication first; private access later.

**Rationale:** Public evidence proves the product loop without introducing OAuth/App installation before it is needed.

### D7 — Hosting

**Decision:** Target Railway Hobby with a persistent volume and `clank.shamanicarts.dev`.

**Rationale:** No agents run on the host; managed long-running application hosting is a better fit than an agent-oriented VM.

