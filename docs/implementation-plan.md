---
type: strategy
status: active
summary: Delivery roadmap from the validated hosted candidate through permanent production, collaborator use, and hardened hosting.
note_created: 2026-08-02
updated: 2026-08-03
---

# ClankSpace Implementation Plan

## Overview

The smallest trustworthy ambient coordination layer is implemented and validated. A hosted candidate has passed deployment, backup, restore, and rollback checks on exe.dev. The active milestone is moving that exact service to durable production infrastructure before onboarding Shuv and other real collaborators. Research remains in service of normal product use.

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
| 3 | Hosted candidate validation on exe.dev | **Complete** | [exe.md](deployment/exe.md) |
| 4 | Permanent Railway production migration | **Active** | [railway.md](deployment/railway.md) |
| 5 | Trusted collaborator onboarding and packaging | Next | [pilot-onboarding.md](pilot-onboarding.md) |
| 6 | Matched controls and retrieval iteration | Planned | [training-loop.md](evals/training-loop.md) |
| 7 | Private repositories and public multi-tenant hardening | Deferred | Prove the trusted pilot first |

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

## Phase 3: hosted candidate validation — complete

Candidate, evaluation, and model-runner workloads were isolated on separate exe.dev VMs. RC-009 was promoted through the lab branch and public `main`; health/readiness, authenticated reads, online backup, off-host copy, restore drill, and binary rollback were verified against the candidate service.

That exercise validated the application and the migration mechanics. It did not make exe.dev the permanent production topology.

## Phase 4: permanent Railway production migration — active

- [ ] Create the human-owned Railway project and single service from this repository.
- [ ] Attach one persistent volume at `/data`; keep the service at one replica for SQLite.
- [ ] Take a fresh online backup of the exe.dev candidate, verify it, and copy it off-provider.
- [ ] Restore the snapshot into Railway and verify record counts plus authenticated export.
- [ ] Configure secrets, health checks, and `clank.shamanicarts.dev` with managed TLS.
- [ ] Enable scheduled Railway volume backups and retain provider-neutral online SQLite backups.
- [ ] Run local, managed-origin, and stable-domain smoke checks plus a restore rehearsal.
- [ ] Retain the exe.dev candidate only for a short rollback window, then retire it; keep exe.dev for evals and runners.

## Phase 5: trusted collaborator onboarding and packaging — next

- [ ] Publish checksummed Linux, macOS, and Windows binaries as `v0.1.0-pilot`.
- [ ] Add a one-line installer or package-manager path.
- [ ] Provision the real `shuv2code` project with distinct Shuv and Shamanic project identities.
- [ ] Open the repository integration PR containing the pointer, skill, and lean agent instruction.
- [ ] Run one real cross-maintainer canary and inspect the resulting append log with both humans.
- [ ] Schedule the proven online backup and external health checks.

## Phase 6: matched controls and retrieval iteration — planned

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

**Decision:** Run permanent production as one Railway service with one persistent SQLite volume behind `clank.shamanicarts.dev`. Use exe.dev for disposable evaluation services, model runners, traces, judges, and Operations. The existing `clankspace-prod` VM is only the validated migration source.

**Rationale:** exe.dev is excellent for agent-driven experiments and replaceable test infrastructure, but shared collaboration state needs a boring, durable application host with a persistent volume, stable domain, managed TLS, and scheduled backups. Railway already matches the repository's container shape while the stable domain keeps the client contract provider-neutral.
