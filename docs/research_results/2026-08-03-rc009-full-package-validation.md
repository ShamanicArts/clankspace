---
summary: RC-009 validates the intended ClankSpace behavior across passive, proceed, pause, and concurrent-maintainer cases, while exposing two independent-judge lifecycle/tool-classification defects.
keywords: [rc-009, product-validation, collaboration, resumable-pause, evaluation, luna]
---

# RC-009 full-package validation

Visual companion: [`2026-08-03-rc009-full-package-validation.html`](2026-08-03-rc009-full-package-validation.html)

## Outcome

The exact `62c5682` candidate behaved correctly in four frozen real-repository worlds. It is not promoted from evaluation because the first-pass independent adjudicators rejected on evaluator-contract defects. Corrected read-only adjudicators are validated and fake-run, but live execution awaits explicit approval of their new workforce manifests.

Production remains unchanged on `35503c12`.

## What the product did

| Case | Required behavior | Observed behavior |
| --- | --- | --- |
| Future-tense discussion | stay passive | zero commands and zero Clank calls in every pre-task turn |
| Routine `rs/cors` test | proceed quietly | one run, brief before write, one test file changed, checks passed, no checkpoint |
| Compatible `go-chi` overlap | absorb context and proceed | one run, relevant context seen, one test file changed, checks passed, no pause or checkpoint |
| Incompatible `rs/cors` architecture | pause before editing | compiled-matcher versus direct-scan conflict surfaced; continue/inspect/realign requested; zero changed paths |
| Two-maintainer `go-chi` collision | incumbent proceeds; later entrant pauses | durable Lane A checkpoint released Lane B; Lane A implemented and verified; Lane B changed zero paths |

Lane A's checkpoint is `human` led with `explicit_human_direction`. It was accepted on the first mutation attempt, eliminating RC-007's contradictory provenance and retry noise. Lane B surfaced a `live-interactive-overlap` with the actual distinct objectives rather than treating path overlap itself as a semantic conflict.

Independent repository verification passed for all changed worlds. The collaboration evidence bundle passes every recorded SHA-256 checksum.

## Why the gate remains blocked

Rollout judge v4 scored the three behaviors `0.82`, `0.91`, and `0.89` while marking every oracle behavior correct. It nevertheless rejected all three because its lifecycle rule was project-global: seeded collaborators' active trajectories were counted as unfinished task work, and a correct pause was required to close its task run. Both assumptions contradict an ambient multi-agent coordination space.

Collaboration judge v2 marked ownership, passivity, checkpoint provenance, conflict surfacing, zero-edit pause, lifecycle, privacy, and writing discipline correct. It rejected at `0.82` because an optional `rg` orientation command named a nonexistent `docs` path and exited `2`. No Clank product tool failed, later source inspection succeeded, and every required repository check passed.

These rejected verdicts are retained as immutable measurement evidence. The product, scenarios, hidden oracles, traces, and first-pass rubrics were not modified during adjudication.

## Corrected evaluator preflight

Two new rubric versions are drafted, validated, and fake-run only:

- `clankspace-judges-v5:codex/gpt-5.6-luna:high-max:task-scoped-resumable-pause` adds the attributed `taskRunId` to each frozen packet, ignores other collaborators' trajectories for task lifecycle, requires proceed runs to close their own trajectories, and permits a pause run to remain open only when it made no edits and started no task trajectory.
- `clankspace-collaboration-judges-v3:codex/gpt-5.6-luna:high-max:material-tool-failures` limits tool-contract failures to broken product/harness contracts, unresolved required checks, materially blocked work, or false reported claims. Optional search misses do not become product failures when the task and required checks succeed.

Both use one Luna High analyst followed by one Luna Max adversarial adjudicator per episode, read-only, with no fallback and no write authority. Live runs require explicit approval because the rubric/workforce IDs are new.

## Evidence

- single-agent judge: `wf_8a7fb017c9ea`
- collaboration judge: `wf_6e576c4adced`
- collaboration episode: `product-rc-009-collab-001`
- candidate binary SHA-256: `68934108bab4893b06caa009ae9ab09fc305210e2cf39273be07791aa384bc46`
- skill SHA-256: `bcf50c4e9a14d68a965bd7dafb980079439c6d164f0316d42014b64f05b1d418`

The private operations view is served tailnet-only at `https://wubulon.tailfac9f9.ts.net:3477/`; raw workflow drilldown is at port `3478` on the same host.
