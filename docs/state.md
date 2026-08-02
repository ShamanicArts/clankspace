---
type: state
status: active
summary: Production/evaluation services and the deterministic synthetic-world evaluation substrate are running on exe.dev.
note_created: 2026-08-02
updated: 2026-08-02
---

# State

## Current focus

Run the first approved Luna train/dev batch through the isolated evaluation service, classify failures, and iterate on retrieval, skill, and tool contracts.

## Active phase

The private production and evaluation services are deployed on separate exe.dev VMs. The runner has sanitized real-repository snapshots, immutable corpus storage, isolated project seeding, genuine resumed-turn rollout capture, and deterministic scoring. Luna workflows are drafted and fake-validated but have not run live.

## Blockers

- The first live Luna workforce and bounded command require explicit approval after the final manifest is shown.
- Semantic retrieval is not yet implemented; the first cohorts intentionally establish the FTS/path baseline.

## Decisions pending

- Whether failed evaluation projects should remain visible indefinitely or be archived after their immutable ledger evidence is synced.
- Which embedding implementation to test after baseline retrieval metrics are frozen.

## Last devlog

`docs/evals/training-loop.md` — active evaluation strategy and evidence boundary.
