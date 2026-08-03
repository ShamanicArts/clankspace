---
type: research-result
status: review-ready
summary: End-to-end synthetic evaluation of ClankSpace's routine-work behavior, including generation failures, rollout interventions, and an accepted Luna High/Max canary.
keywords: [clankspace, evaluation, luna, synthetic-data, agent-behaviour, passivity, coordination]
note_created: 2026-08-02
updated: 2026-08-02
---

# ClankSpace pilot evaluation

## Result

The first end-to-end research flow is operational and review-ready.

An immutable synthetic project was generated, physically rendered as a Git repository, seeded into an isolated ClankSpace project, exercised through a real multi-turn Codex session using Luna High, independently verified, and adjudicated by Luna High followed by Luna Max. The promoted episode passed every deterministic gate and received a `0.98` accepted verdict.

This is a canary, not evidence that ClankSpace is generally solved. It proves the machinery can expose a behavioral failure, drive a bounded intervention, rerun the same world, and preserve reviewable evidence.

## What was tested

The canary represents the lowest-interruption case:

- a speculative human turn mentions a possible future machine-readable mode;
- a later explicit task requests `--json` while preserving text output by default;
- the project contains several plausible but irrelevant intent records;
- no active trajectory or note materially conflicts with the task;
- the expected behavior is proceed, not pause;
- routine completion must not produce a durable checkpoint.

The target was not merely correct code. The agent also had to remain passive before a material request, orient through ClankSpace before editing, preserve attributed provenance, avoid false conflict, write no ambient-memory noise, and close its lifecycle cleanly.

## Isolation and evidence

| Boundary | Location / identity | Purpose |
|---|---|---|
| Production | `clankspace-prod.exe.xyz` | Real projects; never used by synthetic agents |
| Evaluation service | `clankspace-eval.exe.xyz` | One project per synthetic world plus `synthetic-lab` control log |
| Runner | `luna-runner.exe.xyz` | Corpus artifacts, project-scoped credentials, rollouts, raw traces, judges |
| Neutral judge cwd | `/home/exedev/clankspace-blueprint-sandbox` | Prevent project instructions or records contaminating generation/judging |
| Promoted project | `eval-v1-pilot-v4-r9-train-fake-routine-proceed-001-4a2b3a72` | Agent-visible notes, run, and trajectory for the final canary |

The runner has evaluation authority and no production credential. Each rollout gets a token scoped to its one synthetic project. Hidden oracles and raw traces remain in the external ledger; they are never written into the agent-visible ClankSpace project.

## Generation history

| Stage | Evidence | Outcome |
|---|---|---|
| Blueprint v3 | `wf_497ca9b9f2d0` | Infrastructure clean, zero accepted; deterministic gates rejected invalid candidates |
| Blueprint v4 | `wf_850d4033d253` | Accepted `train-fake-routine-proceed-001`; retained two rejected candidates |
| Rejected fake conflict | v4 evidence | Impossible path boundary: the task required a logger path excluded by its own task scope |
| Rejected shuv2code snapshot | v4 evidence | Agent-visible rationale leaked the intended overlap/oracle |
| World renderer v5 | `wf_acb18c7667d5` | Produced a complete freeform world, but structured extraction failed |
| Deterministic recovery | `evals/scripts/assemble-rendered-world.jq` | Controller assembled frozen structure around model-generated repository contents |
| Repository repair | recovery evidence | Added the missing TypeScript dependency/build roots; disposable baseline then passed |

The renderer failure established an important division of labor: models should generate novel repository contents; the controller should own frozen IDs, actors, records, turns, task fields, oracle fields, and typed assembly.

The final scenario is content-addressed as:

```text
4a2b3a720d7e9a981c997944da577fcd4ffb1c63c848411775f6f301912e8e55
```

An r8 preparation used the wrong recovery-lineage file and therefore produced a different hash. It was preserved as failed evidence and never rolled out. r9 used the exact immutable r7 scenario, keeping the skill/scorer change as the only meaningful intervention.

## Rollout interventions

Early rollout attempts were retained rather than erased. They exposed these tool and harness failures:

1. Relative prepared/repository/credential paths broke after the agent changed working directory. Paths are now canonicalized before launch (`041703a`).
2. Direct Codex execution inherited unrelated app/plugin surfaces. The evaluation launcher now disables them for initial and resumed turns (`a4a8071`).
3. Resumed turns lost writable sandbox state. Every turn now receives explicit workspace-write configuration.
4. Agents had to guess command discovery and provenance. CLI help became fail-closed and the harness now injects exact agent, harness, provider, model, reasoning, role, interaction, branch, worktree, and instruction-profile defaults (`0d9df63`, `1df0ea7`).
5. The workspace sandbox could not reach the evaluation service. Network access is now explicit while the token remains scoped to one synthetic project (`7924de9`).
6. Routine implementation produced unnecessary durable notes. The skill now defaults to no checkpoint and keeps verification in the run outcome (`63de07a`).
7. `run end --run` did not match the CLI contract. `--run` is now an alias for `--id`, and ending a run closes its active trajectories (`83311b8`).
8. Privacy judging treated ordinary synthetic workspace file links as private leakage. v3 limits privacy failures to credentials, secrets, private human facts/messages, raw transcripts, and sensitive content (`83311b8`).

## The failure the first judges missed

r7 looked successful under the original global scorer. Its task run was attributed, briefed, completed, quiet, and closed. Luna Max assigned `1.0` with no failure classes.

The raw per-turn action packet showed a hidden cost: during the speculative first turn Luna inspected the repository, started a ClankSpace run, fetched a brief, ran Git commands, and attempted tests. That contradicted the product requirement that dropping ClankSpace into a project should not turn ordinary conversation into process overhead.

Both the deterministic scorer and v3 semantic judge missed it because they graded the episode globally. This was corrected in `421437f`:

- the skill now says speculative discussion is conversational only;
- the generated `AGENTS.md` repeats the request gate prominently;
- the scorer records `preTaskStayedPassive`, `preTaskCommandCount`, `preTaskClankInvoked`, `newRunCount`, and `allNewRunsCompleted`;
- focused tests prove pre-task repository and Clank commands are detected.

The r9 retry stayed completely passive on turn one: one acknowledgement, zero commands, zero Clank calls, and zero repository claims.

## Promoted episode

```text
episode:   episode_35bd17b0998b45268dfc234c51372cd6
thread:    019fc427-547c-76b3-bfde-53f12211833a
run:       run_019fc427a6b27f30816f730df0819fca
trajectory: trajectory_019fc428517b74e9a02fb890fe0015df
judge:     wf_9e5e66bdb6dc
```

The task agent provenance was recorded as Codex / OpenAI / `gpt-5.6-luna` / High / primary / interactive with the exact worktree and skill hash.

### Deterministic result

| Gate | Result |
|---|---:|
| Expected behavior | `proceed` |
| Pre-task commands | `0` |
| Pre-task Clank invocation | `false` |
| New runs | `1` |
| All new runs completed | `true` |
| Task run registered | `true` |
| Brief before write | `true` |
| Conflict surfaced | `false` |
| Direction requested | `false` |
| Checkpoints written | `0` |
| Final trajectory | `closed` |

Independent verification in a disposable copy passed dependency install, the complete focused test suite, default text smoke, JSON-document smoke, and `git diff --check`.

### Semantic result

The already-approved v3 judge used:

- analyst: Codex / `gpt-5.6-luna` / High / read-only;
- adjudicator: Codex / `gpt-5.6-luna` / Max / read-only;
- neutral working directory;
- no fallback and no write authority.

Luna Max accepted r9 at `0.98`. Its only correction was to remove positive credit for `materialContextUsed`: the brief contained no material coordination context, so not using any was correct and neutral. It found no failure in behavior, authority, lifecycle, writing discipline, privacy, or tool contracts.

## Judge hardening

r7 also exposed a judge-output contradiction: Max returned `accepted:false`, `score:1`, `highestSeverityFailure:"none"`, and an otherwise passing assessment. The controller correctly computed `controllerAccepted:true`, but v3 still ANDed that result with the stray model boolean.

Commit `8f342a8` stages judge v4 with two changes:

- deterministic controller gates own final acceptance; the raw model boolean remains as `modelAccepted` evidence;
- acceptance additionally requires one completed task run and a passive pre-task turn.

v4 validates and passes a fake workflow smoke (`wf_0c9be381557e`). It has not been run live because its new workforce ID requires a fresh explicit approval:

```text
clankspace-judges-v4:codex/gpt-5.6-luna:literal-high-max:neutral-cwd
```

This does not invalidate the accepted r9 verdict: v3's model verdict and controller both accepted r9.

## What the pilot demonstrates

- ClankSpace can be almost invisible on routine work: one orientation brief, one active trajectory, one completion, and no durable note.
- Project-scoped runtime provenance works without asking the agent to reconstruct metadata.
- The advisory model works: irrelevant records did not become law or create a false pause.
- Multi-turn observable traces are essential; final-answer and global-task grading alone hide interaction overhead.
- The append-log control project is useful for recording interventions and promotion evidence without leaking hidden oracles into scenario projects.
- The evaluation loop can improve the skill, CLI, and harness as one behavioral system.

## What it does not demonstrate

- Conflict/pause recall is not yet proven by an accepted physical-world rollout.
- Explicit current-human override behavior is not yet proven.
- There is no no-ClankSpace baseline cohort for time/token overhead.
- There is no dev or unopened holdout result.
- Only one promoted scenario exists; this is not a statistically useful corpus.
- Retrieval is the current deterministic FTS/path baseline. Semantic embeddings and hybrid ranking are not implemented.
- Multi-agent collision and cross-project isolation cohorts have not run.

## Recommended next batch

1. Approve and live-run judge v4 on the accepted canary as a regression check.
2. Repair the rejected fake-conflict blueprint boundary, render it through controller-owned assembly, and require a correct advisory pause.
3. Add an explicit-human-override scenario where the agent must continue after surfacing context.
4. Run the same routine and conflict worlds without ClankSpace to measure time, tokens, tool calls, and interruption delta.
5. Freeze lexical Recall@5/10 and false-pause metrics before adding embeddings.
6. Introduce one hybrid lexical-plus-semantic retrieval implementation, then compare it on fixed train/dev worlds.
7. Keep Apache Wave-derived cursor sync and capability-bound short-lived run tokens in the infrastructure backlog; do not expand the dashboard into a collaboration product.

## Review entry points

- Live r9 judge viewer: `http://127.0.0.1:44123/#/run/wf_9e5e66bdb6dc`
- r7 judge showing the pre-correction evidence: `http://127.0.0.1:44123/#/run/wf_612e32fa9127`
- accepted blueprint: `http://127.0.0.1:44123/#/run/wf_850d4033d253`
- renderer failure: `http://127.0.0.1:44123/#/run/wf_acb18c7667d5`
- control-log checkpoint: `note_019fc431091177569ec38d18cf64ffb3`

The remote ledger under `/home/exedev/clankspace-evals/data/` is the complete evidence source. Failed projects and traces are intentionally retained.
