---
type: phase
status: in-progress
phase_number: "2"
prerequisite: Phase 1 complete
estimated_effort: 1 focused session
summary: Add the human project board and read-only public GitHub evidence.
note_created: 2026-08-02
updated: 2026-08-02
---

# Phase 2: Human board and public GitHub evidence

## Goal

Make accrued intent, runtime provenance, trajectories, warnings, repositories, and pull requests easy for humans to scan without turning the dashboard into a mandatory workflow.

## Sub-phases

### 2a: Dashboard

- [x] Token login and workspace/project navigation
- [x] Project overview, current notes, active trajectories, and warnings
- [x] Run provenance including harness/model/role/parent/automation context adjacent to records
- [x] Create and supersede notes
- [ ] Contest/redact administrative controls

### 2b: Public GitHub

- [x] Attach a public GitHub repository by URL
- [x] Cached repository and open-PR refresh
- [x] External evidence labels and links
- [x] Rate-limit/error visibility without breaking project memory

## Success criteria

- [ ] Dashboard interaction verified in controlled desktop and mobile browser (automation unavailable during bootstrap; see handoff).
- [x] Public PRs appear without a GitHub credential.
- [x] Imported prose never acquires intent/decision status automatically.
- [ ] Browser has no console errors or horizontal overflow (static syntax and responsive CSS verified; interactive browser check pending).
