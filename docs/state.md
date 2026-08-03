---
type: state
status: active
summary: Railway production and stable DNS are live; managed TLS, backup scheduling, and restore acceptance remain before collaborator onboarding.
note_created: 2026-08-02
updated: 2026-08-03
---

# State

## Current focus

Finish the stable-domain and recovery gate for the live Railway deployment, then hand the pilot to Shuv and other trusted collaborators. exe.dev remains the reusable agent-compute plane; ClankSpace eval and runner services are one isolated workload on it. Research remains subordinate to observed product friction.

## Active phase

Permanent production is deployed from public `main` to a single Railway service and persistent `/data` volume in EU West. The Railway-managed origin passes external health, readiness, authenticated context, and deterministic project export checks. The restored database contains the expected project and four notes. `clankspace-prod` on exe.dev is stopped and frozen as the short-lived rollback source; evaluation and runner VMs remain isolated agent-compute workloads.

RC-009 exercised three frozen single-agent worlds and one event-gated two-maintainer world on MIT repository snapshots:

- routine rs/cors work proceeded with no checkpoint;
- compatible go-chi overlap proceeded without interruption;
- an incompatible rs/cors matcher architecture paused before edits;
- go-chi Lane A published one coherent human-led checkpoint, implemented, verified, and closed its trajectory, while the later distinct Lane B paused with zero changed paths.

Every pre-task discussion turn stayed passive with zero commands. Independent repository checks pass and the collaboration evidence checksum manifest verifies.

First-pass rollout judge v4 and collaboration judge v2 preserved useful semantic findings but rejected on evaluator defects: project-global trajectory lifecycle, no resumable single-agent pause, and an immaterial optional-search failure classified as a product tool failure. The rejected outputs remain immutable evidence.

Corrected rollout judge v5 accepted aligned overlap at `0.98`, routine proceed at `0.96`, and a fresh architectural-conflict replay at `0.97`. The replay was required because the first v5 pass exposed a repository-specific deterministic scorer; commit `43eb79d` repaired it without changing the product or skill.

Collaboration judge v3 confirmed every product behavior but conflated the finite lane evidence process with the human-facing Clank task. The packet proves Lane B's task run has no outcome or `endedAt`; a direct read-only Luna Max review accepted the split lifecycle at `1.00`.

PR #9 merged the gate into `lab/pilot-v1-base`; PR #10 promoted that lab state to public `main`. The migration used a fresh SQLite online backup, verified by `PRAGMA integrity_check` and SHA-256, with a completed copy retained off-provider. That snapshot is restored on Railway and its authenticated export matches the expected project data. PR #15 hardened Railway volume ownership and permanently drops the service to the unprivileged `clank` user after narrowly scoped mount preparation.

## Operations and evidence

- tailnet operations: `https://wubulon.tailfac9f9.ts.net:3477/`
- raw workflow viewer: `https://wubulon.tailfac9f9.ts.net:3478/`
- visual RC-009 report: `https://wubulon.tailfac9f9.ts.net:3479/clankspace-rc009-product-validation.html`
- gate: `evals/gates/product-rc-009.result.json`
- review report: `docs/research_results/2026-08-03-rc009-full-package-validation.md`
- completion audit: `docs/research_results/2026-08-03-night-shift-completion-audit.md`
- draft PR: `https://github.com/ShamanicArts/clankspace/pull/9`

The operations journal is append-only on the persistent runner and the dashboard labels fake OmegaCode schema runs as `preflight only`, not product verdicts.

## Production migration status

RC-009's frozen result established that the product behavior and deployment artifact were viable. Railway now owns the permanent runtime and migrated collaboration state. Cloudflare publishes the required CNAME and ownership TXT records for `clank.shamanicarts.dev`; Railway-managed certificate issuance is pending. The remaining deployment gate is valid stable-domain TLS, a scheduled backup policy, and a disposable restore rehearsal. The next product uncertainty is population behavior: more models, repositories, seeds, matched no-Clank controls, and eventual semantic retrieval.

## Decisions pending after RC-009

- Whether to upgrade Railway to Pro for native volume backups/PITR or operate a verified external online-backup schedule. Pro is the preferred production path.
- The exact rollback-window duration before deleting the stopped exe.dev migration source.
- Do not issue the exe.dev origin as a supported collaborator pointer; use `clank.shamanicarts.dev` after acceptance.
- Which binary targets and installer surface to support for `v0.1.0-pilot`.
- The exact real `shuv2code` seed records and collaborator identity names.
- Which local or hosted embedding implementation to test after lexical Recall@5/10 and false-pause baselines are frozen.
- The train/dev cohort size required before a semantic-retrieval holdout.
- Private-repository integration and public multi-tenant hardening boundaries.

## Last report

`docs/research_results/2026-08-03-rc009-full-package-validation.html` — phone-readable product iteration, collaboration sequence, evaluator defects, evidence, and the exact remaining gate.
