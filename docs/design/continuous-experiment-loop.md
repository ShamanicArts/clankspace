---
type: spec
status: draft
summary: Reproducible multi-agent experiment loop that converts observed coordination failures into evidence-backed ClankSpace product changes.
keywords: [evaluation, terra, luna, collaboration, experiments, pull-requests, promotion]
note_created: 2026-08-02
updated: 2026-08-02
---

# ClankSpace Continuous Experiment and Improvement Loop

## 1. Outcome

The loop repeatedly answers one product question:

> When several humans and agents move quickly in the same project, does ClankSpace surface the minimum useful intent at the moment it can prevent divergence, without adding ceremony to ordinary work?

Each bounded batch begins from immutable repository evidence and ends with one reviewable **Experiment Batch Dossier**. The dossier connects observed behaviour to any proposed product change:

```text
public MIT source + synthetic collaboration premise
  → immutable scenario worlds
  → independent multi-session rollouts
  → deterministic scores + independent semantic judgment
  → failure clusters
  → minimal intervention PRs
  → exact replay + regression comparison
  → accept, reject, or request human direction
```

This is controlled product experimentation, not autonomous self-modification. Models may generate, run, analyse, and implement. A deterministic controller owns evidence integrity. Sol selects interventions and verifies claims. Human direction remains authoritative, and promotion to `main` initially requires human approval.

## 2. What a successful batch produces

The primary output is not raw traces and not a model-written score. It is a compact, inspectable dossier with five layers.

### 2.1 One-page batch verdict

The first screen or first page answers:

- what was tested and against which exact product commits;
- whether ClankSpace materially helped relative to no-Clank and current-product controls;
- the highest-severity failures found;
- which interventions were attempted;
- which PRs were accepted into the private lab branch, rejected, or need human judgment;
- whether the accumulated lab branch is eligible for a promotion PR to `main`.

Example:

```text
Batch B003 · cross-contributor coordination · 18 scenarios / 54 episodes

Verdict: KEEP ITERATING
Conflict recall                 71% → 89%     improved
Unnecessary pause rate           8% → 11%     regressed; gate failed
Relevant Recall@5               .76 → .91     improved
Routine checkpoint rate          4% →  4%     unchanged
Median orientation overhead     1.8s → 2.0s   within budget

Accepted into lab:  PR #31 — path-aware brief query
Rejected:           PR #32 — always refresh before edit
Needs human input:  F-017 — current user request reverses another maintainer's active trajectory
Main promotion:     NO
```

### 2.2 Coverage matrix

The dossier shows which collaboration behaviours were actually exercised, rather than summarising a heterogeneous batch as one accuracy number.

| Repository | Relationship | Timing | Expected behaviour | Arms |
|---|---|---|---|---|
| `go-chi/chi` snapshot | same-path conflict | A publishes before B briefs | pause and surface rationale | no-Clank/current/candidate |
| `Textualize/rich` snapshot | adjacent work | simultaneous start | proceed without ceremony | no-Clank/current/candidate |
| `lazygit` snapshot | compatible shared direction | B starts from stale context | refresh, align, proceed | current/candidate |
| synthetic canary | explicit human reversal | note changes mid-run | current direction wins | current/candidate |

### 2.3 Cross-contributor timelines

Every collaboration episode has a sequence view based only on observable events:

```text
12:00:00  controller releases A's task
12:00:02  A registers run and publishes trajectory T-14
12:00:02  controller releases B at barrier trajectory:T-14
12:00:04  B requests brief; T-14 ranked #1
12:00:08  A writes checkpoint N-22
12:00:11  B surfaces conflict to its human and waits
12:00:14  simulated human says preserve A's router and adapt B's change
12:00:27  B completes; no duplicate durable note written
```

The timeline links to the exact CLI calls, server receipts, public assistant responses, file changes, tests, and project export. Hidden chain-of-thought is neither requested nor recorded.

### 2.4 Failure and intervention cards

Each material failure becomes a stable, reproducible card:

```text
Failure F-017 · high severity · authority / retrieval boundary
Observed: B saw A's trajectory but described it as a settled project decision.
Expected: surface it as advisory contemporaneous intent and ask only if it changes the next move.
Evidence: E-B003-CHI-07-CURRENT-02, events 41–58
Confidence: high (deterministic record exposure + two independent judges)
Likely layer: skill wording and brief envelope, not ranking

Intervention I-009
Hypothesis: place authority state beside every returned item and tighten the conflict response example.
Owned surfaces: skill/SKILL.md, internal/domain/types.go, internal/service/service.go
Expected movement: authority errors down; no increase in pauses or response bytes
Candidate PR: #34
Replay result: fixed 5/5 target episodes; dev false-pause +0.0 pp
Decision: accept into lab
```

### 2.5 Promotion record

A machine-readable and human-readable promotion record states:

- baseline, candidate, skill, binary, server, corpus, and snapshot hashes;
- hard gates and their actual values;
- semantic judgments and disagreements;
- merged and rejected interventions;
- known residual failures;
- rollback commit;
- `accepted_to_lab`, `rejected`, `human_decision_required`, or `eligible_for_main`.

## 3. Evidence graph and immutable units

The controller assigns stable identifiers and never silently rewrites an observed run.

```text
Batch
├── SourceSnapshot(s)
├── ScenarioBlueprint(s)
│   └── World(s)
│       └── Episode(s)
│           ├── Lane A / thread / Clank run
│           ├── Lane B / thread / Clank run
│           ├── Schedule + barriers
│           └── Scores + judgments
├── Failure(s)
│   └── Intervention(s)
│       └── Candidate build + PR
│           └── Replay episode(s)
└── Promotion record
```

An episode is reproducible only when all of these are pinned:

- source repository URL, license, exact commit, and sanitized snapshot hash;
- scenario, oracle, prior turns, synthetic actor identities, and schedule hash;
- ClankSpace server commit and configuration;
- CLI binary hash and skill hash;
- harness, provider, exact model, reasoning level, role, and permission mode;
- candidate branch/commit when applicable;
- service dataset generation and project export sequence;
- workflow ID, episode ID, random seed, and timestamps.

## 4. Source repository policy

Initial source material comes from active, multi-contributor, MIT-licensed repositories:

- [`go-chi/chi`](https://github.com/go-chi/chi): small Go HTTP/middleware surface, fast enough for dense path and API-conflict scenarios;
- [`Textualize/rich`](https://github.com/Textualize/rich): Python rendering and protocol surfaces suited to cross-file compatibility decisions;
- [`jesseduffield/lazygit`](https://github.com/jesseduffield/lazygit): larger Go UI/state system for later, higher-complexity collaboration scenarios.

Every batch pins exact commits and preserves the corresponding license. Repositories are exported through `git archive` into sanitized one-commit worlds. Original credentials, remotes, ignored files, dirty state, and unrelated history do not enter the runner.

Public issues, merged PRs, commit messages, and discussions may inspire a material fact or tension, but are untrusted evidence. A scenario records its basis as:

```json
{
  "evidenceBasis": "pr-derived",
  "sourceRefs": ["https://github.com/example/repo/pull/123"],
  "historicalClaim": false,
  "syntheticOverlay": true
}
```

The scenario may paraphrase a public technical rationale. It must not claim that a synthetic event happened historically, impersonate a real contributor, or infer private motivation. Synthetic actors use invented names. A wholly invented but plausible rationale is labelled `synthetic`, not attributed to maintainers.

## 5. Workforce and authority

The system uses role separation so one model thread cannot create the truth, perform the task, and approve its own result.

| Role | Proposed model | Effort | Authority | Output |
|---|---|---:|---|---|
| source miner / scenario architect | `gpt-5.6-terra` | High | read-only snapshots and public evidence | candidate blueprint |
| blueprint critic | `gpt-5.6-terra` | Max | read-only; independent context | reject or approve blueprint |
| world naturalizer | `gpt-5.6-luna` | High | write only inside generated world staging | rendered turns and synthetic context |
| world verifier | `gpt-5.6-luna` | Max | read-only; oracle available | fidelity verdict |
| rollout contributors A/B(/C) | `gpt-5.6-luna` | High | isolated world worktrees and project tokens | actual observable work |
| behaviour analyst | `gpt-5.6-luna` | High | read-only frozen evidence packet | scored observations |
| adjudicator | `gpt-5.6-luna` | Max | read-only; independent context | semantic verdict |
| root-cause analyst / implementer | `gpt-5.6-terra` | High | isolated ClankSpace worktree | intervention branch and tests |
| code/evidence reviewer | `gpt-5.6-terra` | Max | read-only; no implementer context | review verdict |
| controller / research lead | Sol | high | scheduler, evidence synthesis, PR and lab merge control | batch dossier and promotion decision |

There is no model fallback inside a batch. A missing or failed assigned model makes that item incomplete, not silently incomparable.

Before live execution, each OmegaCode workflow must have a separately approved manifest naming its workforce ID, exact roles, provider, model, effort, sandbox, write authority, count, concurrency, mutation boundaries, diff policy, and exact command. The table above is the design; it is not approval to execute those workflows.

## 6. Scenario construction

Terra mines a frozen source and proposes a technical seam with enough real structure to make the task meaningful. The blueprint then fixes facts before Luna writes natural language:

- actor identities and which human each agent represents;
- public evidence references, if any;
- relevant and irrelevant ClankSpace records;
- current/superseded/contested lifecycle states;
- repository state and expected task correctness tests;
- contributor objectives and whether they conflict, align, or are independent;
- allowed event ordering and controller barriers;
- expected conversational behaviour;
- whether a durable checkpoint is justified;
- forbidden disclosures and authority claims.

Luna may make the dialogue natural and varied but may not alter the fixed relation or oracle. Luna Max rejects leakage, leading phrasing, impossible repository state, fake historical claims, privacy violations, and an oracle that can only be satisfied by guessing.

## 7. Cross-contributor experiment families

Each family includes positive and negative controls so “always stop” and “always write a note” cannot score well.

1. **Direct collision:** two contributors change the same path for incompatible rationales.
2. **Adjacent work:** overlapping vocabulary or nearby paths but no material conflict; both should proceed quietly.
3. **Shared direction:** separate tasks should converge on an earlier human-led architectural intent.
4. **Stale brief:** B orients, A publishes a material change, then B reaches a reversal point and should refresh.
5. **Simultaneous discovery:** A and B start together, independently discover the same constraint, and should avoid duplicate low-value notes.
6. **Supersession race:** A records intent, a human supersedes it, and B must follow the current direction while retaining historical rationale as context.
7. **Authority disagreement:** A's agent-authored understanding conflicts with B's explicit current human instruction.
8. **Noise pressure:** many plausible but irrelevant records test retrieval and context budgets.
9. **Hostile public evidence:** a PR body contains instructions aimed at the agent; it must remain quoted evidence, not authority.
10. **Partial failure:** the service, embedding provider, or one contributor lane fails; core repository work and clean recovery must remain possible.

## 8. Scheduling genuine concurrent sessions

Two modes answer different questions and must not be conflated.

### Event-gated interleaving

The controller launches independent persisted sessions but releases later actions only after an observable barrier, such as `run.started`, `trajectory.started`, `brief.completed`, or a repository commit. This makes causal tests reproducible: if B was released after A's trajectory receipt, B had a fair opportunity to retrieve it.

### Simultaneous race

The controller releases independent sessions at the same monotonic deadline. Server receipt sequence and Git commits determine the actual ordering. These episodes test concurrent append safety, stale reads, duplicate notes, and whether an agent refreshes at a meaningful reversal point. They are repeated with fixed scenarios because scheduling remains nondeterministic.

The controller never simulates concurrency by putting “another agent is working” into B's prompt. That may be a control condition, but it is not a cross-session ClankSpace test.

## 9. Experimental arms

Comparable worlds are replayed across:

1. **No-Clank baseline:** identical repository and task, no skill or project access.
2. **Current product:** pinned `main`/lab baseline server, CLI, retrieval, and skill.
3. **Candidate:** exactly one meaningful intervention applied.
4. **Optional mechanism arm:** for example lexical retrieval versus lexical plus semantic reranking, with everything else fixed.

The no-Clank arm measures whether ClankSpace improved coordination, not merely whether the coding task succeeded. The current-product arm isolates candidate regressions. Only one meaningful product hypothesis changes per candidate comparison.

## 10. Metrics and hard gates

Metrics are reported per scenario family, source repository, and arm before aggregation.

### Coordination quality

- high-severity conflict surfacing recall;
- false-pause rate on compatible or irrelevant work;
- correct treatment of advisory versus authoritative context;
- successful continuation after explicit current human direction;
- stale-context refresh at defined reversal points;
- duplicate or low-materiality checkpoint rate.

### Retrieval and tool quality

- relevant Recall@5/10 and reciprocal rank;
- lifecycle and project leakage;
- first-attempt CLI/MCP success;
- median tool calls, bytes, latency, and retries;
- lexical versus semantic contribution to surfaced candidates.

### Work quality and overhead

- repository task tests and static checks;
- unintended path changes;
- pre-task passivity during discussion turns;
- time and token overhead against no-Clank;
- run/trajectory completion and provenance accuracy.

### Safety

- transcript, credential, private, emotional, or inferred-personal leakage;
- prompt-injection obedience from imported evidence;
- fabricated attribution or historical claims;
- cross-project data exposure.

A candidate cannot be accepted because a semantic judge likes it. Required task checks, privacy, project isolation, idempotency, and lifecycle invariants are deterministic hard gates. Initial numerical thresholds are calibrated from the first two batches, then versioned; they are never moved retroactively to rescue a candidate.

## 11. Failure triage and intervention selection

Sol freezes all rollouts before analysis and clusters failures by the narrowest likely layer:

```text
scenario/oracle · retrieval/ranking · CLI/MCP contract · skill instruction
identity/provenance · lifecycle/concurrency · service/storage · evaluator/scorer
```

The first question is whether the product failed or the experiment measured the wrong thing. An evaluator defect creates a versioned evaluator intervention and forces affected episodes to be rescored or rerun; it does not count as a product improvement.

An intervention is eligible for implementation when it has:

- a reproducible failure packet;
- a falsifiable cause hypothesis;
- the smallest plausible changed surface;
- an expected metric movement and named regression risks;
- target replay, dev, and canary test plans;
- no need to inspect hidden reasoning.

Terra implements in an isolated worktree. A separate Terra Max reviewer receives the failure packet, diff, tests, and frozen comparison, but not the implementer's private working context.

## 12. Private branch and pull-request flow

GitHub branch visibility is repository-level. `ShamanicArts/clankspace` is currently private, so its lab branches and PRs are private too.

```text
main
  └── lab/b003-base                     pinned batch integration base
       ├── exp/b003/f017-authority-envelope  → PR into lab/b003-base
       └── exp/b003/f021-brief-ranking       → PR into lab/b003-base

lab/b003-base → human-reviewed promotion PR → main
```

Rules:

- every batch base is pinned; a later batch starts a new base rather than mutating old evidence;
- one product hypothesis per intervention PR;
- Terra may push its assigned intervention branch but never merge it;
- Sol may merge a passing intervention into the batch lab branch only after focused tests, target replay, dev comparison, regression canaries, and independent review;
- accepted lab PRs are squash-merged so one lab commit maps to one intervention and PR;
- rejected PRs remain closed with their evidence packet; failed ideas are useful research output;
- promotion from a batch lab branch to `main` uses a merge PR and initially requires human approval;
- rollback is the parent of the intervention squash or the prior promoted lab commit, recorded explicitly.

Each PR body is generated from evidence and includes failure ID, episode links, hypothesis, changed surfaces, before/after metrics, regressions, verification commands, ClankSpace control-note IDs, and rollback point.

## 13. Artifact contract

The external ledger stores the complete research record:

```text
batches/b003/
  batch-manifest.json
  source-snapshots.json
  scenario-blueprints.jsonl
  worlds/<scenario-hash>/
    scenario.json
    prepared.json
    repo/
  episodes/<episode-id>/
    schedule.json
    lanes/<actor>/turn-*/events.jsonl
    lanes/<actor>/turn-*/response.md
    server-receipts.jsonl
    repository-result.json
    project-export.json
    deterministic-score.json
    semantic-verdicts.jsonl
  failures.jsonl
  interventions.jsonl
  pull-requests.jsonl
  comparisons.json
  promotion.json
  report.html
  SHA256SUMS
```

`report.html` is self-contained and can be copied off the private runner for review. It links locally to the immutable evidence bundle and never depends on a live site.

ClankSpace's `synthetic-lab` project receives only concise professional control records:

- batch started, with manifest and baseline refs;
- a material failure class selected for intervention;
- an intervention accepted or rejected, with rationale and PR;
- batch promotion or stop decision.

Raw transcripts, hidden oracles, credentials, private model reasoning, and noisy per-episode status do not enter ClankSpace.

## 14. Semantic retrieval experiment

Semantic search begins as candidate reranking, not an opaque replacement for deterministic retrieval:

```text
project/lifecycle/path filters
  → bounded FTS candidate set
  → embedding similarity rerank or expansion
  → explicit score components in the brief envelope
```

The batch records embedding model/version, text projection hash, vector dimensions, candidate set, lexical score, semantic score, final rank, latency, and provider failure. An embedding outage falls back to deterministic FTS and is surfaced in provenance without blocking normal work.

The first semantic cohort targets paraphrased intent with low lexical overlap and includes noisy semantic near-neighbours. It must improve relevant Recall@5 without breaching context, latency, false-pause, or privacy gates.

## 15. Batch cadence and stopping rules

The loop is deliberately bounded. A proposed first full batch is:

- 3 frozen MIT repository snapshots;
- 12 train scenarios and 4 dev scenarios;
- 3 arms for core cells: no-Clank, current, candidate;
- 2 independent Luna High repetitions for deterministic interleavings;
- 5 repetitions for simultaneous-race cells;
- at most 3 selected failure classes;
- at most 2 concurrent intervention PRs;
- holdout remains sealed.

Stop the batch when any of these occurs:

- all selected interventions have accepted or rejected verdicts;
- two consecutive triage passes find no new high-confidence product failure;
- the fixed episode, token, time, or intervention budget is exhausted;
- infrastructure invalidates comparability;
- a safety or authority issue requires human direction.

The next batch begins from the accepted lab head and a new immutable manifest. It does not quietly append new scenarios or candidate changes to the old comparison.

## 16. First implementation increments

1. Extend the scenario contract from one task actor to multiple lanes plus an explicit schedule and source-evidence provenance.
2. Add controller event barriers and simultaneous-release support to the rollout harness.
3. Record CLI calls, server receipts, per-lane threads, repository outcomes, and cross-lane timelines in one episode envelope.
4. Add baseline/current/candidate comparison and failure/intervention schemas.
5. Generate the self-contained batch dossier from immutable artifacts.
6. Add branch/PR automation targeting a pinned private lab branch; keep merges disabled until gates are proven.
7. Freeze the first `go-chi/chi` snapshot and run one direct-collision plus one adjacent-work canary before expanding repositories.
8. Introduce semantic reranking only after the collaboration controller and comparison report are trustworthy.

## 17. Risks and countermeasures

| Risk | Consequence | Countermeasure |
|---|---|---|
| Synthetic scenarios reward theatrical Clank usage | apparent gains, poor real-world value | no-Clank controls, routine/adjacent negatives, real snapshots, overhead gates |
| Same model grades itself | correlated blind spots | independent contexts, hard gates, Luna Max adjudication, Sol verification |
| PR-derived scenario misrepresents a maintainer | reputational or privacy harm | synthetic actors, paraphrase, source links, `historicalClaim: false` |
| Concurrent results are irreproducible | ambiguous regressions | event-gated lane plus repeated simultaneous-race lane |
| Agent fixes the evaluator instead of product | benchmark gaming | separate evaluator/product intervention types and ownership |
| Lab branch accumulates interacting changes | unclear causality | pinned batch base, one hypothesis per PR, replay after every merge |
| More retrieval causes more interruption | product becomes bureaucracy | false-pause, bytes, calls, latency, and checkpoint gates |
| ClankSpace becomes its own noisy research log | contamination of tested memory | detailed external ledger; only four material control-note types |

## 18. Decisions in this proposal

**Pick:** the Batch Dossier is the review unit.

**Why not a dashboard of aggregate scores:** it hides causal evidence and makes bad merges hard to audit.

**Pick:** real repository structure with explicitly synthetic collaboration stories.

**Why not reenact actual maintainers:** public PR evidence rarely contains the complete private rationale, and pretending otherwise damages provenance.

**Pick:** genuine independent sessions with event barriers and repeated races.

**Why not prompt-only simulation:** it does not test the append log, retrieval timing, identity, or concurrent writes.

**Pick:** private per-batch integration branches and evidence-backed PRs.

**Why not let agents patch `main` directly:** it destroys causal comparisons and gives the experiment the authority to approve itself.

**Pick:** Sol owns synthesis and lab merges; humans own initial `main` promotion.

**Why not make every merge manual:** the lab should move at agent speed once deterministic gates are trusted, while the product release boundary remains deliberate.

## 19. Exit condition for the first research cycle

The research flow is ready for human review when one complete two-contributor canary has produced:

- a licensed, pinned real-repository world;
- two actual independent Luna sessions sharing one ClankSpace project;
- an event-gated cross-contributor timeline;
- no-Clank and current-product results;
- at least one honestly classified failure or a defensible no-failure result;
- one Terra intervention PR or a documented reason not to change the product;
- exact replay and dev/canary comparison;
- a self-contained dossier and promotion record whose claims can be followed to observable evidence.

Only then should the controller scale scenario count, introduce simultaneous races, or begin semantic retrieval experiments.
