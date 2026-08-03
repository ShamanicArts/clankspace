---
summary: RC-009 passes the intended ClankSpace behavior across passive, proceed, pause, and concurrent-maintainer cases, while exposing and repairing evaluator defects.
keywords: [rc-009, product-validation, collaboration, resumable-pause, evaluation, luna]
---

# RC-009 full-package validation

Visual companion: [`2026-08-03-rc009-full-package-validation.html`](2026-08-03-rc009-full-package-validation.html)

## Outcome

The exact `62c5682` product candidate passed four real-repository worlds: passive discussion, routine proceed, compatible overlap, architectural conflict, and an event-gated two-maintainer collision. Independent semantic review accepts every intended product behavior. The validated tree was promoted through PR #9 and PR #10 and is live on production as public-main commit `6b20f444`.

Production binary SHA-256 is `55966693703568d094a52d889c55a2cf45a8d127fd9cef1020d5cd15fd1c1a6f`. External health/readiness and authenticated project context/export pass. The pre-deploy SQLite backup is verified on-host and off-host, and the previous production binary remains available for rollback.

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

## What the evaluation caught

Rollout judge v4 scored the three behaviors `0.82`, `0.91`, and `0.89` while marking every oracle behavior correct. It nevertheless rejected all three because its lifecycle rule was project-global: seeded collaborators' active trajectories were counted as unfinished task work, and a correct pause was required to close its task run. Both assumptions contradict an ambient multi-agent coordination space.

Collaboration judge v2 marked ownership, passivity, checkpoint provenance, conflict surfacing, zero-edit pause, lifecycle, privacy, and writing discipline correct. It rejected at `0.82` because an optional `rg` orientation command named a nonexistent `docs` path and exited `2`. No Clank product tool failed, later source inspection succeeded, and every required repository check passed.

These rejected verdicts are retained as immutable measurement evidence. The product, scenarios, hidden oracles, traces, and first-pass rubrics were not rewritten to make the gate pass.

## Corrected adjudication

The approved corrected workflows ran live:

- `clankspace-judges-v5:codex/gpt-5.6-luna:high-max:task-scoped-resumable-pause` adds the attributed `taskRunId` to each frozen packet, ignores other collaborators' trajectories for task lifecycle, requires proceed runs to close their own trajectories, and permits a pause run to remain open only when it made no edits and started no task trajectory.
- `clankspace-collaboration-judges-v3:codex/gpt-5.6-luna:high-max:material-tool-failures` limits tool-contract failures to broken product/harness contracts, unresolved required checks, materially blocked work, or false reported claims. Optional search misses do not become product failures when the task and required checks succeed.

Aligned overlap and routine proceed passed at `0.98` and `0.96`. The conflict behavior was semantically judged correct but exposed a hard-coded deterministic scorer that recognized only permission/router vocabulary. Commit `43eb79d` made that scorer repository-agnostic. A fresh isolated replay with the unchanged candidate skill again paused before editing, and the same v5 High→Max judge accepted it at `0.97` (`wf_c61006aeb0dc`).

The collaboration v3 judge confirmed every product behavior but confused `lane.status=completed`—the finite evidence process—with the Lane B Clank run. The packet shows Lane B's task run has no `endedAt` or outcome. A direct read-only Luna Max review of that exact distinction accepted at `1.00`: the Clank task remains resumably open while the evidence envelope correctly finishes and seals its checksums. The clarified v4 rubric is retained for future batches.

The final accepted behavior scores are therefore `0.98` aligned, `0.96` routine, `0.97` architectural conflict, and `1.00` collaboration lifecycle. No product change, hidden-oracle change, or evidence rewrite was used to obtain those results.

## Evidence

- original rollout judge: `wf_8a7fb017c9ea`
- corrected rollout judge: `wf_6f003d3720bd`
- corrected conflict replay judge: `wf_c61006aeb0dc`
- original collaboration judge: `wf_6e576c4adced`
- corrected collaboration judge: `wf_6a8c3bdb197a`
- collaboration episode: `product-rc-009-collab-001`
- candidate binary SHA-256: `68934108bab4893b06caa009ae9ab09fc305210e2cf39273be07791aa384bc46`
- skill SHA-256: `bcf50c4e9a14d68a965bd7dafb980079439c6d164f0316d42014b64f05b1d418`
- production commit: `6b20f444781ab53747671de12a5f71184e93a767`
- production binary SHA-256: `55966693703568d094a52d889c55a2cf45a8d127fd9cef1020d5cd15fd1c1a6f`
- pre-deploy backup: `clankspace-prod-pre-rc009-20260803T074631Z.db` (`82f91f393f41…`)

The private operations view is served tailnet-only at `https://wubulon.tailfac9f9.ts.net:3477/`; raw workflow drilldown is at port `3478` on the same host.
