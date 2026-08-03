---
summary: Final requirement-by-requirement audit for the observable ClankSpace product-validation night shift.
keywords: [completion-audit, operations, deployment, validation, terra, luna, rc-009]
audited_at: 2026-08-03T07:54:00Z
---

# Night-shift completion audit

This audit checks the repository, GitHub promotion path, runner, production, evaluation, tailnet observability, immutable corpora, adjudication, backups, rollback, and authenticated production reads. It does not infer completion from a plan or model summary.

## 1. Durable operations dashboard and shift log — proven

- The runner's `ops-dashboard` and `omega-viewer` processes remain supervised in tmux.
- The append-only operations journal records generation, invalid runs, interventions, candidate deployments, adjudication, gate passage, backup, merge, and production deployment.
- Tailnet-only Operations, raw workflows, and the visual report return HTTP 200 on ports `3477`, `3478`, and `3479`.
- The dashboard distinguishes fake schema checks as `preflight only` and currently reports RC-009 as passed.
- The final visual report was browser-verified at phone and desktop sizes with no console errors or page-content overflow.

## 2. Safe isolated hosting, backup, rollback, and deployment — proven

- Production, evaluation, and runner workloads remain separated across exe.dev VMs.
- Production runs public-main commit `6b20f444781ab53747671de12a5f71184e93a767`.
- Production binary SHA-256 is `55966693703568d094a52d889c55a2cf45a8d127fd9cef1020d5cd15fd1c1a6f`; `clank version` reports `6b20f444`.
- The deployment used same-filesystem atomic replacement and retained `/usr/local/bin/clank.rollback-35503c12-pre-rc009`.
- Fresh SQLite online backup `clankspace-prod-pre-rc009-20260803T074631Z.db` exists on-host and off-host, mode 0600, passes `PRAGMA integrity_check`, and matches SHA-256 `82f91f393f419bdf31fbcb125cf2b985c0b9a44662bb7d5f3446e7a8a43d6bd6`.
- Local and external `/healthz` and `/readyz` pass after restart.
- Authenticated `clank context` and project export pass against production without changing project state.

## 3. Seeded production-like product validation — proven for the frozen cohort

- Three independent Luna High sessions exercised frozen `go-chi/chi` and `rs/cors` MIT snapshots through two passive discussion turns and a real task.
- One event-gated collaboration episode used two independent Luna High identities, worktrees, credentials, and sessions against one shared evaluation project.
- Every pre-task discussion turn issued zero commands and zero Clank calls.
- Routine and compatible work proceeded without checkpoints; the incompatible matcher architecture paused before edits.
- Lane A published one coherent human-led checkpoint, implemented, verified, and closed. Lane B started only after the durable barrier, surfaced the live overlap, changed zero paths, created no trajectory, and kept its Clank run open for the human's answer.
- Independent repository checks and the collaboration checksum manifest pass.

## 4. Independent semantic acceptance — proven

- Corrected rollout v5 accepted aligned overlap at `0.98` and routine proceed at `0.96`.
- The first v5 conflict assessment found the behavior correct but exposed the deterministic scorer's permission/router vocabulary dependency. Commit `43eb79d` repaired the evaluator without changing the product or skill.
- A fresh isolated conflict replay again stayed passive, retrieved the relevant intent, paused before editing, and was accepted by the same Luna High→Max workforce at `0.97` (`wf_c61006aeb0dc`).
- Collaboration v3 confirmed every product behavior but conflated a completed evidence-capture lane with the open Clank task. The immutable packet shows Lane B has no `outcome` or `endedAt`. A direct read-only Luna Max review accepted the split lifecycle at `1.00` with no material failures.
- Original rejected verdicts remain preserved as evaluator-defect evidence; no candidate, hidden oracle, or old trace was rewritten.

## 5. Research materially improved the product — proven

- RC-006 exposed early work during discussion, wrong-lane yielding, and inaccurate verification reporting.
- RC-007 repaired incumbent/later-entrant ownership but exposed aligned over-pause and contradictory checkpoint provenance.
- RC-008 repaired passive discussion and compatible proceed behavior but exposed an ambiguous live-collision distinction.
- RC-009 added task-independent execution risk, coherent leadership/basis validation, exact CLI values, pre-auth mutation help, and passive bootstrap instructions.
- Observable deltas are conflict pre-task commands `7 → 0`, checkpoint mutation retries `2 → 0`, later-entrant changed files `3 → 0`, and correctly behaving frozen worlds `1 → 4`.
- Evaluator defects found during validation were repaired in the evaluator rather than mislabeled as product work.

## 6. Product and harness verification — proven

- Focused tests pass for `internal/service`, `internal/httpapi`, `cmd/clank`, `evals/harness`, and `evals/cmd/clank-ops`; `internal/mcpserver` builds and vets and has no test files.
- Focused `go vet` passes for the affected packages.
- The exact merged-main tree was tested and built in a detached clean worktree before deployment.
- `git diff --check` and gate JSON parsing pass.
- Production serves the expected advisory-authority notice after deployment.

## 7. GitHub promotion and review trail — proven, with one advisory caveat

- PR #9 merged the accepted gate into `lab/pilot-v1-base` at `626faeee`; its CodeRabbit check passed.
- PR #10 merged the exact lab state to public `main` at `6b20f444`.
- GitHub treated PR #10's CodeRabbit status as advisory, so an auto-merge request executed immediately while that review was still pending. Production deployment waited through focused clean-tree verification; the cumulative product changes had already passed their constituent PR checks and the still-pending advisory review had posted no finding at audit time.
- A post-deployment evidence PR records the final production commit, binary, backup, rollback, and smoke checks.

## 8. Claims boundary — explicit

RC-009 supports the exercised product claims: passive discussion, quiet routine work, compatible overlap, useful pre-edit conflict surfacing, coherent checkpoint provenance, and incumbent/later-entrant coordination.

It does not establish population-level reliability across models/repositories/seeds, lift over matched no-Clank controls, private-repository integration, or public multi-tenant hardening. Those remain the next product-driven experiments rather than reasons to withhold the validated pilot.

## Audit verdict

The requested night shift is complete: the product materially improved from measured failures, the final behavior passed deterministic and independent semantic validation, the evidence is observable from the phone-accessible dashboard, the repository is public, the validated tree was promoted through lab and main, and the exact merged build is running on production with verified backup and rollback paths.
