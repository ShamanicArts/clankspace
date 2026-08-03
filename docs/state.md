---
type: state
status: active
summary: RC-009 passed deterministic and independent semantic validation and is deployed on public main with backup and rollback verified.
note_created: 2026-08-02
updated: 2026-08-03
---

# State

## Current focus

Observe the now-production RC-009 behavior in real collaborator use while designing the next matched-control and retrieval experiments. Product work remains primary; research should only continue where it can improve the CLI/API/skill package.

## Active phase

Production, evaluation, and the runner are isolated on separate exe.dev VMs. Production now runs public-main commit `6b20f444` with binary SHA-256 `559666937035`; evaluation retains the frozen candidate `62c5682` with binary SHA-256 `68934108bab4`. Both deployments pass external health and readiness checks.

RC-009 exercised three frozen single-agent worlds and one event-gated two-maintainer world on MIT repository snapshots:

- routine rs/cors work proceeded with no checkpoint;
- compatible go-chi overlap proceeded without interruption;
- an incompatible rs/cors matcher architecture paused before edits;
- go-chi Lane A published one coherent human-led checkpoint, implemented, verified, and closed its trajectory, while the later distinct Lane B paused with zero changed paths.

Every pre-task discussion turn stayed passive with zero commands. Independent repository checks pass and the collaboration evidence checksum manifest verifies.

First-pass rollout judge v4 and collaboration judge v2 preserved useful semantic findings but rejected on evaluator defects: project-global trajectory lifecycle, no resumable single-agent pause, and an immaterial optional-search failure classified as a product tool failure. The rejected outputs remain immutable evidence.

Corrected rollout judge v5 accepted aligned overlap at `0.98`, routine proceed at `0.96`, and a fresh architectural-conflict replay at `0.97`. The replay was required because the first v5 pass exposed a repository-specific deterministic scorer; commit `43eb79d` repaired it without changing the product or skill.

Collaboration judge v3 confirmed every product behavior but conflated the finite lane evidence process with the human-facing Clank task. The packet proves Lane B's task run has no outcome or `endedAt`; a direct read-only Luna Max review accepted the split lifecycle at `1.00`.

PR #9 merged the gate into `lab/pilot-v1-base`; PR #10 promoted that lab state to public `main`. Exact merged-main build `6b20f444` is live on production. A fresh SQLite online backup exists on-host and off-host with SHA-256 `82f91f393f41`; the prior `35503c12` binary is retained as `clank.rollback-35503c12-pre-rc009`. Local and external health/readiness, authenticated context, and authenticated export pass.

## Operations and evidence

- tailnet operations: `https://wubulon.tailfac9f9.ts.net:3477/`
- raw workflow viewer: `https://wubulon.tailfac9f9.ts.net:3478/`
- visual RC-009 report: `https://wubulon.tailfac9f9.ts.net:3479/clankspace-rc009-product-validation.html`
- gate: `evals/gates/product-rc-009.result.json`
- review report: `docs/research_results/2026-08-03-rc009-full-package-validation.md`
- completion audit: `docs/research_results/2026-08-03-night-shift-completion-audit.md`
- draft PR: `https://github.com/ShamanicArts/clankspace/pull/9`

The operations journal is append-only on the persistent runner and the dashboard labels fake OmegaCode schema runs as `preflight only`, not product verdicts.

## Production status

RC-009 is `passed-production` / `promoted-to-production`. No product-gate or deployment blocker remains. The next uncertainty is population behavior: more models, repositories, seeds, matched no-Clank controls, and eventual semantic retrieval.

## Decisions pending after RC-009

- Which local or hosted embedding implementation to test after lexical Recall@5/10 and false-pause baselines are frozen.
- The train/dev cohort size required before a semantic-retrieval holdout.
- Private-repository integration and public multi-tenant hardening boundaries.

## Last report

`docs/research_results/2026-08-03-rc009-full-package-validation.html` — phone-readable product iteration, collaboration sequence, evaluator defects, evidence, and the exact remaining gate.
