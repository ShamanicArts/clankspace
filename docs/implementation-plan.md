---
type: strategy
status: active
summary: Delivery roadmap from the live trusted pilot to easier distribution, real collaborator use, and hardened hosting.
note_created: 2026-08-02
updated: 2026-08-03
---

# ClankSpace Implementation Plan

## Overview

The smallest trustworthy ambient coordination layer is implemented, validated, and running as a trusted-collaborator production pilot. The roadmap now shifts from proving the original collision to making the client easy to distribute, onboarding Shuv and other real collaborators, measuring value during normal work, and hardening only the boundaries demanded by that use.

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
| 1 | Core coordination loop | **Complete** | [phase-1-core-loop.md](phases/phase-1-core-loop.md) |
| 2 | Human log and public GitHub evidence | **Complete** | [phase-2-board-github.md](phases/phase-2-board-github.md) |
| 3 | Hosted production pilot on exe.dev | **Complete** | [exe.md](deployment/exe.md) |
| 4 | Trusted collaborator onboarding and packaging | **Active** | [pilot-onboarding.md](pilot-onboarding.md) |
| 5 | Matched controls and retrieval iteration | Planned | [training-loop.md](evals/training-loop.md) |
| 6 | Private repositories and public multi-tenant hardening | Deferred | Prove the trusted pilot first |

## Phase 0: Project setup — complete

- [x] Repository structure and Git initialization
- [x] Lean agent instructions and compatibility pointer
- [x] Current design specification
- [x] Implementation plan and first phase documents
- [x] Knowledge base and portable project skill
- [x] Initial devlog and state tracking

## Phase 1: Core coordination loop — complete

Implement workspaces/projects, principals/agents/runs, notes, trajectories, transactional event receipts, briefs, coordination warnings, CLI, and stdio MCP.

Success means the original Shuv/Shamanic scenario is represented in fixtures and a conflicting request produces a useful advisory warning without treating older context as an instruction.

## Phase 2: Human log and public GitHub evidence — complete

Expose the agent-maintained project log with adjacent runtime provenance, light human governance, repository attachment, and public PR evidence. Conflict inspection remains in the agent workflow.

## Phase 3: hosted production pilot — complete

Production, evaluation, and model-runner workloads are isolated on separate exe.dev VMs. RC-009 was promoted through the lab branch and public `main`; production health/readiness, authenticated reads, online backup, off-host copy, restore drill, and binary rollback are verified.

Railway remains a portable alternative rather than the active topology. The stable `clank.shamanicarts.dev` route is reserved but not yet connected to the production origin.

## Phase 4: trusted collaborator onboarding and packaging — active

- [ ] Publish checksummed Linux, macOS, and Windows binaries as `v0.1.0-pilot`.
- [ ] Add a one-line installer or package-manager path.
- [ ] Route `clank.shamanicarts.dev` to the production service.
- [ ] Provision the real `shuv2code` project with distinct Shuv and Shamanic project identities.
- [ ] Open the repository integration PR containing the pointer, skill, and lean agent instruction.
- [ ] Run one real cross-maintainer canary and inspect the resulting append log with both humans.
- [ ] Schedule the proven online backup and external health checks.

## Phase 5: matched controls and retrieval iteration — planned

Measure time, token, tool-call, interruption, false-pause, and useful-conflict deltas against matched no-Clank runs. Expand repository/model/seed coverage before introducing embeddings. Add semantic retrieval only when deterministic lexical baselines and failure cases are frozen.

## Research backlog

- [Researched](research_results/2026-08-02-apache-wave-patterns.md): extract Apache Wave's snapshot-plus-delta history, versioned receipts, participant access, and robot-capability patterns without inheriting OT, federation, or the Wave UI.
- Evaluate API-token-router guardrails from that research: method-scoped short-lived run tokens should enforce project, actor, tool, and write boundaries outside model prompts while records remain bounded CLI/MCP output.

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

**Decision:** Run the first trusted production pilot on a persistent exe.dev VM, with separate evaluation and runner VMs. Keep Railway/Docker portability and reserve `clank.shamanicarts.dev` as the stable client endpoint.

**Rationale:** exe.dev was immediately available, supports the current operational workflow, and now has verified systemd restart, backup, restore, and rollback behavior. The product remains one portable Go binary and SQLite database, so moving to Railway or another managed host later does not change the client contract.
