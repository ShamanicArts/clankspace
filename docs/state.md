---
type: state
status: active
summary: Hosted accounts, signed replication, and one-prompt repository onboarding are merged and deployed to the lean exe.dev pilot.
note_created: 2026-08-02
updated: 2026-08-04
---

# State

## Current focus

Prepare the deployed hosted-replication candidate for real trusted-collaborator use. Human onboarding now uses local passwords and direct owner-generated invite links, removing SMTP as an operational dependency. The next product signal should come from real Shuv/Shamanic use, not a larger synthetic campaign.

PR #24 replaced manual first-run workspace plumbing with `clank setup`: the agent infers the Git repository and project, opens a short-lived browser request, and waits for one human approval. The server creates or reuses the project, links a supported public GitHub remote, and returns a project-only credential directly to the CLI. The CLI stores it outside the repository, installs the skill and pointer, adds one lean agent instruction, and verifies access. A fresh `go-chi/chi` checkout passed the complete browser-to-terminal flow without a token entering chat.

## Active phase

PR #21 merged invitation-only human accounts, many workspaces per
human, workspace roles, project-scoped agent keys, a quiet human management surface, signed
snapshot-plus-event replication, offline local writes, self-hosted authority, cloud mirrors,
replica revocation, encrypted local secrets, migration backups, and downgrade protection.

The final end-to-end campaign passed. It registered a hosted owner, invited a second human,
created separate workspaces, issued an agent key, paired a local replica, stopped the cloud,
recorded a conflicting local checkpoint, restarted and reconciled the cloud, created a
separate self-hosted authority, mirrored it to the cloud, imported it into a third peer, and
proved that a revoked replica could no longer publish. The run exposed one real transaction
bug in local-account promotion; the fix was applied and the failed step was replayed
successfully. Focused store, service, HTTP, client, sync, local-routing, and CLI tests pass.

The shareable validation index is
`docs/research_results/2026-08-03-hosted-replication-e2e.md`. The phone-readable visual
implementation report and six recordings are preserved outside the repository under
`~/.agent/diagrams/clankspace-e2e-final-2026-08-03/`; no live credentials are stored in the
repository.

Merged main `6904b3a` is deployed on exe.dev. Production schema version 12 passed
`PRAGMA integrity_check`; the original one workspace, one project, and four notes remain.
The service has a durable installation secret, signed-sync identity, retained prior binary,
and an integrity-checked pre-deployment snapshot both on-host and off-host. PR #26 added
Argon2id password sign-in, direct dashboard/CLI invite links, and a setup page that presents
the pending repository request before login and preserves it through approval. SMTP is no
longer an onboarding dependency. The interactive guide is served at
`https://clank.shamanicarts.dev/docs/` and leads with the one-prompt setup path. Manual token
and local-replica controls remain available as advanced fallbacks; raw implementation evidence
is kept out of the primary task flow.

Production runs from public `main` on `clankspace-prod.exe.xyz`, published through `clank.shamanicarts.dev`. The final Railway database was downloaded quiescently, passed `PRAGMA integrity_check`, and was restored with the matching production bootstrap credential. Strict-HTTPS health/readiness, CLI context, and deterministic authenticated export pass; the project contains the expected four notes. A persistent daily local timer performs SQLite's online backup remotely, pulls the completed snapshot off-provider, verifies integrity, and writes a checksum manifest; its first live run passed. Railway has no active deployment. The runner and evaluation VM evidence was checksum-verified locally before both disposable VMs were deleted.

RC-009 exercised three frozen single-agent worlds and one event-gated two-maintainer world on MIT repository snapshots:

- routine rs/cors work proceeded with no checkpoint;
- compatible go-chi overlap proceeded without interruption;
- an incompatible rs/cors matcher architecture paused before edits;
- go-chi Lane A published one coherent human-led checkpoint, implemented, verified, and closed its trajectory, while the later distinct Lane B paused with zero changed paths.

Every pre-task discussion turn stayed passive with zero commands. Independent repository checks pass and the collaboration evidence checksum manifest verifies.

First-pass rollout judge v4 and collaboration judge v2 preserved useful semantic findings but rejected on evaluator defects: project-global trajectory lifecycle, no resumable single-agent pause, and an immaterial optional-search failure classified as a product tool failure. The rejected outputs remain immutable evidence.

Corrected rollout judge v5 accepted aligned overlap at `0.98`, routine proceed at `0.96`, and a fresh architectural-conflict replay at `0.97`. The replay was required because the first v5 pass exposed a repository-specific deterministic scorer; commit `43eb79d` repaired it without changing the product or skill.

Collaboration judge v3 confirmed every product behavior but conflated the finite lane evidence process with the human-facing Clank task. The packet proves Lane B's task run has no outcome or `endedAt`; a direct read-only Luna Max review accepted the split lifecycle at `1.00`.

PR #9 merged the gate into `lab/pilot-v1-base`; PR #10 promoted that lab state to public `main`. The original migration used a fresh SQLite online backup, verified by `PRAGMA integrity_check` and SHA-256. Railway acceptance passed, then the quiescent Railway database was exported, integrity-checked, and restored to the current exe.dev pilot. Both completed snapshots remain off-provider.

## Operations and evidence

- tailnet operations: `https://wubulon.tailfac9f9.ts.net:3477/`
- raw workflow viewer: `https://wubulon.tailfac9f9.ts.net:3478/`
- visual RC-009 report: `https://wubulon.tailfac9f9.ts.net:3479/clankspace-rc009-product-validation.html`
- gate: `evals/gates/product-rc-009.result.json`
- review report: `docs/research_results/2026-08-03-rc009-full-package-validation.md`
- completion audit: `docs/research_results/2026-08-03-night-shift-completion-audit.md`
- hosted replication and docs: `https://github.com/ShamanicArts/clankspace/pull/21`
- living guide: `https://clank.shamanicarts.dev/docs/`

The operations journal and raw workflow evidence are preserved in the local runner export. No Operations or workflow viewer is currently hosted; those surfaces return with the next reprovisioned evaluation campaign.

## Production hosting status

RC-009 established that the product and portable deployment artifact are viable. A complete round-trip migration through Railway proved the provider boundary; production now runs on exe.dev while the trusted pilot is small. Cloudflare points the stable hostname to exe.dev and Railway's old ownership record/domain claim has been removed. Daily off-provider backup is active and the round trip exercised a full restore. The next product uncertainty is real collaborator behavior.

## Decisions pending after RC-009

- The periodic restore-drill cadence for the exe.dev pilot.
- When real usage justifies moving back to Railway or another managed persistent-volume host.
- Do not issue the exe.dev origin as a supported collaborator pointer; use `clank.shamanicarts.dev`.
- Which binary targets and installer surface to support for `v0.1.0-pilot`.
- The exact real `shuv2code` seed records and collaborator identity names.
- Which local or hosted embedding implementation to test after lexical Recall@5/10 and false-pause baselines are frozen.
- The train/dev cohort size required before a semantic-retrieval holdout.
- Private-repository integration and public multi-tenant hardening boundaries.

## Last report

`docs/research_results/2026-08-03-rc009-full-package-validation.html` — phone-readable product iteration, collaboration sequence, evaluator defects, evidence, and the exact remaining gate.
