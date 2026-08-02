---
type: phase
status: complete
phase_number: "1"
prerequisite: Phase 0 complete
estimated_effort: 1-2 focused sessions
summary: Deliver the attributed note, trajectory, brief, warning, CLI, and MCP loop.
note_created: 2026-08-02
updated: 2026-08-02
---

# Phase 1: Core coordination loop

## Goal

Provide a complete headless workflow from project/run registration through material note and trajectory capture to a bounded brief that surfaces likely coordination conflicts.

## Sub-phases

### 1a: Durable domain

- [x] Embedded migrations and WAL configuration
- [x] Workspace/project/repository entities
- [x] Human owner and project-scoped principals, agents, and richly attributed runs
- [x] Intent/decision/understanding notes and trajectories
- [x] Event and idempotency receipt transaction
- [x] Bounded field and secret-pattern validation

### 1b: Retrieval and coordination

- [x] Structured project brief
- [x] Path and keyword overlap candidates
- [x] Current and superseded note lifecycle
- [ ] Stale and contested transitions (deferred; schema values reserved)
- [x] Original Discord scenario fixture and test

### 1c: Universal agent surface

- [x] CLI commands for project, run, note, trajectory, brief, why, export, and project keys
- [x] Stdio MCP tools backed by the same API/service
- [x] Portable agent skill and repository instruction snippet

## Success criteria

- [x] Exact retry produces one event and one stable receipt.
- [x] Runtime provenance differentiates primary, subagent, and automation runs.
- [x] A conflicting request surfaces Shuv’s provider-neutral-control trajectory and rationale.
- [x] The response explicitly says context is advisory.
- [x] Focused tests pass and the binary builds.
