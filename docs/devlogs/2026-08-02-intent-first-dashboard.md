---
type: devlog
status: complete
session_start: "14:27"
session_end: "14:48"
phase: "2"
subphase: "2a"
approval: review
summary: Reframe the dashboard around the coordination moment and the original two-maintainer project intent.
note_created: 2026-08-02
updated: 2026-08-02
---

# Intent-first dashboard

## Problem

The initial dashboard was visually pleasant but read as a generic administration surface for notes and trajectories. It exposed the data model rather than demonstrating why ClankSpace exists: one maintainer's agent should understand another maintainer's active direction and rationale before reversing related work.

## Product correction

- Added `PRODUCT.md` so the repository explicitly defines the users, coordination job, advisory trust model, anti-references, and design principles.
- Added `DESIGN.md` and `.impeccable/design.json` to preserve the accepted warm editorial visual system while preventing a return to generic dashboard patterns.
- Made the proposed-work check the primary project interaction.
- Compared a proposed move and intersecting active trajectory side by side, with Continue, Inspect, and Realign actions.
- Moved active work before historical context.
- Added an Agent View showing the bounded context the next agent carries forward.
- Recast notes as an attributed intent trail with contemporaneous rationale.
- Kept repository and PR activity as external evidence, visually downstream from intent.
- Replaced opaque provenance with project-principal and agent display names.
- Added instructional empty states for the declare → accrue → catch-divergence loop.

## Verification scenario

The controlled browser used two project identities:

- **Shuv · Codex session** declared provider-neutral cross-session control and an active permission/session trajectory.
- **Shamanic · shuv2code session** recorded observed regressions and the need to stabilize adjacent flows without erasing that direction.

Entering “Remove the new permission layer” against `apps/web/permissions` produced the full possible-divergence comparison, attributed Shuv's objective and rationale, and offered Continue, Inspect, or Realign without treating context as authority.

Verification covered dark and light appearances, desktop and iPhone layouts, action scrolling, public PR evidence, no horizontal overflow, and a clean browser console.
