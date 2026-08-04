---
type: devlog
status: active
session_start: "02:20"
phase: "5"
subphase: "dogfood"
approval: user-directed
summary: Audit hosted onboarding, close authorization gaps, and make Git delivery provenance automatic enough for real dogfooding.
note_created: 2026-08-04
updated: 2026-08-04
---

# Auth audit and delivery provenance

## Intent

Finish the password/direct-invite onboarding campaign, then fix the deeper dogfooding gap exposed by the live ClankSpace board: notes may be intentionally sparse, but a collaborator must still be able to see which branch and commit produced them and which pull request or merge eventually delivered the work.

## Findings

- The seven-PR onboarding campaign had not registered real ClankSpace runs.
- The live project contained four notes and no structured commit or pull-request delivery evidence; one commit was mentioned only in free text.
- shuv2code was not connected to the hosted workspace.
- `run start` accepted branch/worktree but did not infer Git coordinates; `run end` recorded only outcome and free-text verification.
- The bearer/CLI invitation route missed the replica-authority `share_humans` gate used by the dashboard route, and invite-password attempts were not rate-limited.
- Approved documentation still mixed the superseded SMTP/magic-link plan with the deployed direct-invite/password model.

## Direction

- Keep decision text immutable and sparse.
- Capture origin and delivery provenance on the run, which decisions/checkpoints already reference.
- Automatically detect repository, branch, worktree, base, and HEAD at run start.
- Automatically detect delivered HEAD and the current GitHub PR at run end.
- Allow `run link` after delivery so a later PR or merge enriches the run without rewriting its notes.
- Show origin, delivered commit, PR state, and merge commit in the quiet append log.

## Progress

- Three-way code/spec/test audit completed.
- Authority and invite-rate-limit gaps fixed.
- Schema v13, API, CLI, MCP, replication, snapshot, and dashboard support for delivery provenance implemented locally.
- Migrated runs may backfill a missing repository during `run link` only when that repository is already attached to the same project; an existing run repository can never be replaced.
- Focused origin/delivery inheritance and replica-convergence tests added.
- Agent skill and active onboarding/design docs updated around automatic provenance and direct invitation links.

## Verification

Pending final focused tests, browser verification, production migration, and real ClankSpace/shuv2code dogfood.
