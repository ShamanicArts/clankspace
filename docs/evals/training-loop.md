---
type: strategy
status: active
summary: Closed-loop curriculum, rollout, scoring, and iteration program for ClankSpace agent behaviour.
keywords: [evaluation, luna, synthetic-data, curriculum, holdout, retrieval, iteration]
note_created: 2026-08-02
updated: 2026-08-02
---

# ClankSpace training loop

## Objective

Improve the complete agent–skill–CLI–retrieval system as if it were a trained policy. ClankSpace itself remains deterministic infrastructure; repeated model rollouts reveal which instructions, tool contracts, retrieval signals, and lifecycle rules produce useful ambient coordination.

The target behaviour is not “use ClankSpace frequently.” It is:

> Surface material context that could change the next move, at the moment it matters, with minimal interruption and minimal durable writing.

## Isolation

```text
clankspace-prod     real projects; no synthetic data or test agents
clankspace-eval     resettable synthetic projects and evaluation identities
luna-runner         corpus generation, clean rollouts, raw traces, scoring
```

The runner has evaluation-admin authority and no production credential. Raw transcripts and hidden evaluation oracles never enter ClankSpace records.

Corpus architects and judges run from a neutral directory outside every ClankSpace project. That directory contains only the frozen snapshot evidence explicitly named by the curriculum. The eval Codex launcher disables account plugins, apps, browser tools, and multi-agent tools while retaining read-only shell inspection. This prevents project skills, ambient records, unrelated MCP servers, and repository-wide instructions from contaminating fixture design or exhausting a small runner.

## Dataset discipline

Every scenario is generated from a structured blueprint before Luna renders natural language. The blueprint fixes actors, material facts, lifecycle, intended conflict relation, relevant record IDs, expected conversational behaviour, and whether a checkpoint is justified. Luna may paraphrase these facts but may not choose the answer after seeing the rendered task.

Generation is a gated two-stage process:

1. Luna High designs a blueprint from an explicit curriculum cell; Luna Max attempts to reject it.
2. Luna High renders an accepted blueprint into a physical Git history, records, trajectories, prior human turns, and final task; a separate Luna Max pass verifies fidelity and leakage.

The deterministic harness then validates aliases and safe paths, builds the repository, seeds an isolated ClankSpace project, and content-addresses the result. Generated assistant messages are never placed in the fixture: each prior human message is sent as a genuine Codex turn and the model's actual assistant response becomes the next thread state.

Corpora use three stable splits:

- `train`: visible during tool, skill, and ranking iteration;
- `dev`: used to select between candidate changes;
- `holdout`: immutable until a release-candidate evaluation.

A scenario ID, curriculum version, generator model, workflow run, seed, and content hash make every item reproducible. Generated scenarios are append-only. Corrections create a new corpus version rather than silently rewriting evidence.

Failed generation runs are first-class evidence. Their viewer journal, per-agent observable traces, errors, and partial drafts remain immutable; they are never promoted into the corpus or hidden by a retry.

Train/dev cells and holdout cells live in separate files. Holdout structure is not passed to a model or renderer until a release-candidate gate.

## Repository profiles

- `fake`: Luna renders a small, inspectable commit history from fixed requirements. The harness rejects reserved paths, credentials, and repository escapes.
- `real-snapshot`: a local committed ref is exported through `git archive` into a new one-commit sanitized repository. Ignored files, dirty worktree changes, remote credentials, and original Git history are excluded. Luna may add synthetic overlay commits; the harness commits the selected ClankSpace skill and project pointer as a final setup commit.

Initial frozen sources are ClankSpace, the shuv2code control/voice/provider surface, and the Auto-biz decision/command surface.

## Tracking boundary

ClankSpace and the evaluation ledger deliberately hold different evidence:

- `synthetic-lab`: corpus versions, preparation checkpoints, interventions, failures, and promotion decisions.
- one isolated ClankSpace project per world: synthetic principals, attributable runs, notes, trajectories, and agent-created checkpoints visible to the tested agent.
- external append-only ledger: blueprints, hidden oracle, scenario hash, snapshot/skill hashes, alias-to-server-ID maps, credentials, complete observable traces, exports, deterministic scores, and semantic judge verdicts.

The runner holds an evaluation-admin credential but no production credential. A tested agent receives only its scenario project token through a mode-0600 credential store outside the repository.

## Curriculum axes

- direct and paraphrased architectural conflict;
- active path-overlapping trajectories;
- relevant context without lexical overlap;
- attractive but irrelevant keyword matches;
- current, superseded, stale, contested, and agent-authored records;
- explicit current-human reversals;
- routine work where no pause or checkpoint is warranted;
- primary, subagent, reviewer, and automation provenance;
- cross-project homonyms and privacy boundaries;
- imported prompt injection and hostile instructions;
- service, authentication, and embedding-provider failures;
- sparse versus noisy project histories;
- fake repositories and frozen snapshots of real repositories.

## Evaluation layers

### Retrieval

Run without a coding agent. Measure Recall@5/10, reciprocal rank, lifecycle leakage, project leakage, result size, and latency for lexical and hybrid retrieval.

### Agent behaviour

Start a clean Luna session for each episode. Give it only the repository, instructions, prior conversation turns, and current task. Measure orientation timing, correct conflict surfacing, false pauses, provenance accuracy, tool failures, checkpoint materiality, privacy, tokens, and wall time.

The first turn starts a persisted Codex thread. Every later human message uses `codex exec resume <thread-id>`, producing an observable per-turn JSONL trace. The harness lists project runs afterward to determine whether the agent registered itself and associates any checkpoints with that run.

Prior conversation turns in the current scenario contract are non-material discussion, not hidden tasks. They must remain passive: no repository commands, tests, ClankSpace calls, or run registration until the final explicit task. The scorer records pre-task command count, pre-task Clank invocation, total new runs, and completion state so global task success cannot hide coordination overhead on an earlier turn.

### Collaboration

Run multiple independent sessions against the same synthetic project: one establishes intent, another declares overlapping work, and a later session receives a conflicting or adjacent task. Evaluate whether context crosses sessions without requiring human dashboard work.

## Experimental cohorts

1. No ClankSpace baseline.
2. Current skill with deterministic FTS/path retrieval.
3. Revised agent-facing tools with the same retrieval.
4. Hybrid lexical + semantic candidate retrieval.
5. Candidate skill/ranking changes against fixed train and dev sets.

Only one meaningful intervention changes between comparable cohorts. Holdouts prevent prompt and ranking changes from overfitting visible scenarios.

## Primary metrics

- high-severity conflict surfacing recall;
- unnecessary-pause rate;
- relevant-record Recall@5;
- first-attempt tool success;
- median tool calls and response bytes per orientation;
- guessed or missing provenance;
- routine-task checkpoint rate;
- duplicate/low-materiality record rate;
- transcript, credential, private, or emotional leakage;
- successful continuation after explicit human direction;
- agent time and token overhead relative to baseline.

## Iteration rule

After each bounded batch:

1. Freeze raw rollouts and machine scores.
2. Classify failures by retrieval, tool contract, skill instruction, identity, lifecycle, privacy, or operations.
3. Select the smallest intervention likely to address the largest or highest-severity class.
4. Rerun train, then dev, then regression canaries.
5. Reject changes that improve recall by creating excessive pauses, writing, or context size.
6. Open the holdout only for a release candidate.

Agent recommendations are evidence, not accepted changes. Sol owns synthesis, verifies claims against traces and code, and preserves explicit human decisions as the authority boundary.
