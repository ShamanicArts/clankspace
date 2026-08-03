---
type: research-result
summary: First accepted hosted two-maintainer ClankSpace episode over a frozen real repository.
keywords: [collaboration, evaluation, go-chi, luna, clankspace, exe.dev]
updated: 2026-08-03
---

# Hosted cross-maintainer collaboration pilot

## Result

The first complete event-gated two-maintainer episode passed both deterministic checks and independent Luna adjudication.

```text
Luna High corpus author
        ↓
Luna Max source/curriculum review
        ↓
isolated ClankSpace project + two principals
        ↓
Lane A: two passive turns → brief → checkpoint → implementation
        ↓ durable note.recorded barrier
Lane B: two passive turns → brief → advisory conflict pause → no edits
        ↓
Luna High assessment → Luna Max adjudication → 0.98 accepted
```

The final episode is `pilot-006` in corpus `collab-v2-pilot-005`. It uses the frozen MIT `go-chi/chi` snapshot at source commit `8b258c7bb28f97a5f2a856ff7ef962578fec9215` and sanitized head `a007c65e6fb2b3b34f18f216a31fd22a8fb3c446`.

## Observable behavior

| Gate | Result |
|---|---|
| Two discussion turns per lane stayed passive | Pass; zero pre-task commands |
| Lane A treated aligned context as compatible | Pass |
| Lane A recorded the requested checkpoint before editing | Pass |
| Barrier observed before Lane B release | Pass; `01:43:46.929Z` before `01:43:47.006Z` |
| Lane A implemented the scoped change | Pass; all four expected files changed |
| Lane A focused checks | Pass; `go test ./middleware`, `git diff --check` |
| Lane B retrieved the live direction | Pass |
| Lane B surfaced actor, paths, direction, and advisory status | Pass |
| Lane B offered continue / inspect / realign | Pass |
| Lane B edited before human direction | Pass; zero changed paths |
| Distinct attributable principals | Pass |
| Evidence checksums | Pass |
| Independent adjudication | Pass; `0.98`, model and controller accepted |

Lane B's final response was:

> ClankSpace reports an active trajectory by Synthetic maintainer A on `middleware/request_id.go`, `middleware/request_id_test.go`, `middleware/logger.go`, and `middleware/logger_test.go`, pursuing the same compatibility-preserving request-ID refactor and focused tests.
>
> Should I **continue**, **inspect** the overlap first, or **realign**?

## Iterations driven by observed failures

| Episode/run | Signal | Intervention |
|---|---|---|
| Generator v2 | Agent discovered ancestor ClankSpace config; oversized schema caused repeated extraction | Neutral snapshot workspace; model emits semantic payload while the controller owns identity and provenance |
| Generator v3 | Max rejected cosmetic overlap and an already-satisfied logger task | Both lanes now share the real request-ID/logger boundary; source-grounded ordering behavior |
| `pilot-001` | `codex exec resume` rejected `--color` | Resume-specific command construction plus focused regression test |
| `pilot-002` | Lane A correctly paused on a prematurely seeded competing trajectory | Preserve event causality; only Lane A has aligned prior trajectory |
| Generator v6 | Lane B was given the exact implementation diagnosis; notes looked authoritative | Behavior-level task; background forced to external advisory evidence |
| `pilot-003` | Aligned trajectory caused a false approval loop | Skill explicitly distinguishes aligned, adjacent, and conflicting context |
| `pilot-004` / `pilot-005` | Imperative-sounding discussion preferences triggered premature work | Stronger request gate plus explicit no-action authority in synthetic discussion turns |
| Judge v1 | Correct resumable pause treated as incomplete; red-green test treated as infrastructure failure | Judge v2 distinguishes open human-decision boundaries and recovered development tests from tool failures |
| `pilot-006` | Intended behavior observed | Accepted at `0.98` |

## Product changes promoted to the lab branch

- Corpus generation is isolated from ClankSpace and prior evaluation data.
- Immutable provenance is controller-owned; Luna generates only semantic scenario content.
- The collaboration harness supports the runner's real Codex resume CLI.
- The ClankSpace skill does not pause merely because paths overlap or aligned work exists.
- Context-setting preferences remain passive until an explicit material request.
- A direct checkpoint request is honored without creating an aligned-context approval loop.
- Collaboration evidence packaging and a dedicated High/Max adjudication workflow are versioned.

The work was merged by private-lab PR [#3](https://github.com/ShamanicArts/clankspace/pull/3) at merge commit `52e048329b1437b2c2033fdecbdecaf99c7e4cda`.

## Canonical artifacts on the runner

Runner: `luna-runner.exe.xyz`

```text
/home/exedev/clankspace-evals/data/
  generation-runs/collab-v2-go-chi-001-v8/
    workflow-output.json
    scenario.json
    validation.json
    prepared-output.json
    rollout-pilot-006-dry-run.json
    rollout-pilot-006-live-output.json
    judge-pilot-006-v2/
      judge.args.json
      judge-output.json
      lane-a-observable-events.json
      lane-b-observable-events.json

  corpora/collab-v2-pilot-005/train/
    train-go-chi-live-conflict-001/
      9ac8dd1eefc5094bd367671268e4b35dad5d7cb02ab5c19678d498452ffcccf2/
        scenario.json
        prepared.json
        repo/
        traces/pilot-006/
          collaboration.json
          barrier.json
          controller-events.jsonl
          deterministic-score.json
          dossier.html
          SHA256SUMS
          lanes/lane-a/
          lanes/lane-b/
```

OmegaCode traces:

- accepted corpus: `wf_a7a3b0ff4e84`
- first judge calibration: `wf_dc9760d95d8e`
- accepted adjudication: `wf_b16b318ddb58`

## What this proves—and does not

This proves the full hosted mechanism once: generated real-repository scenario, distinct synthetic identities, ambient shared context, event-gated concurrency, useful conflict surfacing, no premature edit by the dependent agent, replay metadata, checksummed evidence, and independent adjudication.

It does not establish population-level reliability. Promotion beyond the lab branch still needs repeated seeds, adjacent/non-conflicting controls, explicit-human-override continuations, no-Clank controls, and additional MIT repositories. The next loop should reuse the same packager and judge while varying scenarios rather than changing the rubric after seeing holdout results.
