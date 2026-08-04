---
type: devlog
status: complete
session_start: "02:20"
session_end: "04:21"
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
- PRs #33 and #34 merged and production migrated from schema 12 to 13 with a verified on-host and off-host backup.
- The real shuv2code project, repository, Shamanic agent identity, pointer, skill, and instruction are connected; its first run captured origin Git coordinates and later attached delivery branch, commit, and PR #76.
- Two bounded, maintainer-grounded shuv2code records seed the original collaboration problem without importing transcripts or synthetic data.

## Verification

- Three independent code/spec/test audits passed after their findings were fixed.
- Focused CLI, service, store, HTTP, GitHub, MCP, and sync tests passed; the origin-stream ordering test passed 20 repeated runs.
- A real schema-12 database with an ended run migrated to schema 13; event and snapshot round-trips preserved origin and delivery provenance.
- `go vet`, JavaScript syntax, and diff checks passed.
- Controlled desktop and iPhone-sized browser passes rendered origin branch/commit, delivery branch/commit, PR state, and merge evidence.
- Production reports build `4af2889`, schema 13, `PRAGMA integrity_check = ok`, and healthy stable-domain readiness. The pre-v13 backup checksum matches off-host.
- The controlled browser retained stale DNS for the retired Railway app, so the shuv2code setup request was approved through the authenticated operator endpoint. The waiting CLI still performed the normal exchange and installed every artifact; the temporary fallback credential was revoked.
- shuv2code PR #76 is open for its maintainer rather than merged unilaterally.
