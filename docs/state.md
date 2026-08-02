---
type: state
status: active
summary: The isolated evaluation substrate is running; blueprint v4 is being prepared after a clean, correctly rejected v3 pilot.
note_created: 2026-08-02
updated: 2026-08-02
---

# State

## Current focus

Run the bounded Luna v4 blueprint pilot after approval, then render only accepted worlds and run genuine resumed-turn agent episodes through the isolated evaluation service.

## Active phase

The private production and evaluation services are deployed on separate exe.dev VMs. The runner has sanitized real-repository snapshots, immutable corpus storage, isolated project seeding, genuine resumed-turn rollout capture, and deterministic scoring. Blueprint run `wf_497ca9b9f2d0` completed without infrastructure failures and promoted no weak fixtures. Two fake designs passed Luna Max but were caught by deterministic actor-reference or conversation-shape gates. Luna Max rejected the real-snapshot design for contradictory and ambiguous active trajectories; the controller also caught an overlay-only boundary violation. Blueprint v4 constrains aliases, actor provenance, exact trajectory counts, checkpoint expectations, and snapshot commit semantics.

## Blockers

- The revised live Luna workforce and bounded command require explicit approval after the new manifest is shown.
- Semantic retrieval is not yet implemented; the first cohorts intentionally establish the FTS/path baseline.

## Decisions pending

- Whether failed evaluation projects should remain visible indefinitely or be archived after their immutable ledger evidence is synced.
- Which embedding implementation to test after baseline retrieval metrics are frozen.

## Last devlog

`docs/evals/training-loop.md` — active evaluation strategy and evidence boundary.
