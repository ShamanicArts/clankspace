---
type: state
status: active
summary: RC-009 matches the intended passive, proceed, conflict-pause, provenance, and incumbent/later-entrant behavior; production promotion is held for corrected semantic adjudication.
note_created: 2026-08-02
updated: 2026-08-03
---

# State

## Current focus

Complete the frozen RC-009 gate with corrected read-only semantic adjudication. Do not change the candidate, scenarios, hidden oracles, or traces while that gate is pending. Promote only after both corrected judges meet the `0.85` threshold and the full completion audit remains clean.

## Active phase

Production, evaluation, and the runner are isolated on separate exe.dev VMs. Production remains on public-main commit `35503c12` with binary SHA-256 `589c3d81dcfc`; evaluation runs candidate `62c5682` with binary SHA-256 `68934108bab4`. Both deployments pass external health and readiness checks.

RC-009 exercised three frozen single-agent worlds and one event-gated two-maintainer world on MIT repository snapshots:

- routine rs/cors work proceeded with no checkpoint;
- compatible go-chi overlap proceeded without interruption;
- an incompatible rs/cors matcher architecture paused before edits;
- go-chi Lane A published one coherent human-led checkpoint, implemented, verified, and closed its trajectory, while the later distinct Lane B paused with zero changed paths.

Every pre-task discussion turn stayed passive with zero commands. Independent repository checks pass and the collaboration evidence checksum manifest verifies.

First-pass rollout judge v4 and collaboration judge v2 preserved useful semantic findings but rejected on evaluator defects: project-global trajectory lifecycle, no resumable single-agent pause, and an immaterial optional-search failure classified as a product tool failure. The rejected outputs remain immutable evidence.

Corrected rollout judge v5 and collaboration judge v3 are authored, validated, and fake-run only. Their exact new workforce manifests require explicit approval before live use. Draft PR `#9` remains open against `lab/pilot-v1-base`; production is intentionally untouched.

## Operations and evidence

- tailnet operations: `https://wubulon.tailfac9f9.ts.net:3477/`
- raw workflow viewer: `https://wubulon.tailfac9f9.ts.net:3478/`
- visual RC-009 report: `https://wubulon.tailfac9f9.ts.net:3479/clankspace-rc009-product-validation.html`
- gate: `evals/gates/product-rc-009.result.json`
- review report: `docs/research_results/2026-08-03-rc009-full-package-validation.md`
- completion audit: `docs/research_results/2026-08-03-night-shift-completion-audit.md`
- draft PR: `https://github.com/ShamanicArts/clankspace/pull/9`

The operations journal is append-only on the persistent runner and the dashboard labels fake OmegaCode schema runs as `preflight only`, not product verdicts.

## Blocker

Live corrected adjudication requires explicit approval of both new manifests:

- `clankspace-judges-v5:codex/gpt-5.6-luna:high-max:task-scoped-resumable-pause`
- `clankspace-collaboration-judges-v3:codex/gpt-5.6-luna:high-max:material-tool-failures`

Until they run, RC-009 stays `blocked-adjudication` / `hold-evaluation-only`; PR #9 stays draft and production stays on `35503c12`.

## Decisions pending after RC-009

- Which local or hosted embedding implementation to test after lexical Recall@5/10 and false-pause baselines are frozen.
- The train/dev cohort size required before a semantic-retrieval holdout.
- Private-repository integration and public multi-tenant hardening boundaries.

## Last report

`docs/research_results/2026-08-03-rc009-full-package-validation.html` — phone-readable product iteration, collaboration sequence, evaluator defects, evidence, and the exact remaining gate.
