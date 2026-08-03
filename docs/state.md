---
type: state
status: active
summary: RC-009 passed deterministic and independent semantic validation; promotion through lab, main, and production is in progress.
note_created: 2026-08-02
updated: 2026-08-03
---

# State

## Current focus

Promote the validated RC-009 product from the evaluation branch through `lab/pilot-v1-base` and public `main`, then deploy the exact merged build with a fresh production backup and rollback binary.

## Active phase

Production, evaluation, and the runner are isolated on separate exe.dev VMs. Production remains on public-main commit `35503c12` with binary SHA-256 `589c3d81dcfc`; evaluation runs candidate `62c5682` with binary SHA-256 `68934108bab4`. Both deployments pass external health and readiness checks.

RC-009 exercised three frozen single-agent worlds and one event-gated two-maintainer world on MIT repository snapshots:

- routine rs/cors work proceeded with no checkpoint;
- compatible go-chi overlap proceeded without interruption;
- an incompatible rs/cors matcher architecture paused before edits;
- go-chi Lane A published one coherent human-led checkpoint, implemented, verified, and closed its trajectory, while the later distinct Lane B paused with zero changed paths.

Every pre-task discussion turn stayed passive with zero commands. Independent repository checks pass and the collaboration evidence checksum manifest verifies.

First-pass rollout judge v4 and collaboration judge v2 preserved useful semantic findings but rejected on evaluator defects: project-global trajectory lifecycle, no resumable single-agent pause, and an immaterial optional-search failure classified as a product tool failure. The rejected outputs remain immutable evidence.

Corrected rollout judge v5 accepted aligned overlap at `0.98`, routine proceed at `0.96`, and a fresh architectural-conflict replay at `0.97`. The replay was required because the first v5 pass exposed a repository-specific deterministic scorer; commit `43eb79d` repaired it without changing the product or skill.

Collaboration judge v3 confirmed every product behavior but conflated the finite lane evidence process with the human-facing Clank task. The packet proves Lane B's task run has no outcome or `endedAt`; a direct read-only Luna Max review accepted the split lifecycle at `1.00`. RC-009 is `passed` / `promote-to-lab-base`. Draft PR `#9` remains open only for the mechanical promotion sequence; production is still untouched at this point.

## Operations and evidence

- tailnet operations: `https://wubulon.tailfac9f9.ts.net:3477/`
- raw workflow viewer: `https://wubulon.tailfac9f9.ts.net:3478/`
- visual RC-009 report: `https://wubulon.tailfac9f9.ts.net:3479/clankspace-rc009-product-validation.html`
- gate: `evals/gates/product-rc-009.result.json`
- review report: `docs/research_results/2026-08-03-rc009-full-package-validation.md`
- completion audit: `docs/research_results/2026-08-03-night-shift-completion-audit.md`
- draft PR: `https://github.com/ShamanicArts/clankspace/pull/9`

The operations journal is append-only on the persistent runner and the dashboard labels fake OmegaCode schema runs as `preflight only`, not product verdicts.

## Promotion status

No product-gate blocker remains. The outstanding steps are PR #9 → lab, lab → main, fresh backup, exact-build production deployment, authenticated smoke verification, and a final evidence audit.

## Decisions pending after RC-009

- Which local or hosted embedding implementation to test after lexical Recall@5/10 and false-pause baselines are frozen.
- The train/dev cohort size required before a semantic-retrieval holdout.
- Private-repository integration and public multi-tenant hardening boundaries.

## Last report

`docs/research_results/2026-08-03-rc009-full-package-validation.html` — phone-readable product iteration, collaboration sequence, evaluator defects, evidence, and the exact remaining gate.
