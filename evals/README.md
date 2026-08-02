# ClankSpace evaluations

This directory defines versioned scenario contracts, curriculum cells, deterministic preparation tooling, and fixed evaluation assets. Generated corpora and raw model rollouts live on the isolated exe.dev runner and are imported only as bounded, reviewed evidence artifacts.

The scenario contract is [`schema/scenario.schema.json`](schema/scenario.schema.json). It separates rendered project context from an oracle that is withheld from the rollout agent.

See [`docs/evals/training-loop.md`](../docs/evals/training-loop.md) for dataset discipline, experimental cohorts, metrics, and promotion gates.

## Components

- `curriculum/v1.cells.json`: train/dev cells available for iteration.
- `curriculum/v1.holdout.cells.json`: physically separate holdout cells; do not open before a release candidate.
- `schema/scenario.schema.json`: rendered-world contract. The hidden oracle is stored here but never copied into the repository or ClankSpace.
- `harness/`: Git-world construction, safe snapshotting, ClankSpace seeding, immutable ledger ingestion, resumed-turn rollouts, and deterministic scoring.
- `cmd/clank-eval`: operational CLI built with `go build ./evals/cmd/clank-eval`.
- `bin/codex-eval`: lean Codex launcher for model workers; disables unrelated plugin/tool surfaces while preserving the authenticated Codex provider.
- `fixtures/rendered/relaydesk-001.json`: deterministic non-model canary.
- `fixtures/collaboration/two-lane-v2.json`: contract fixture for the first event-gated collaboration pilot.

## Runner layout

```text
/home/exedev/clankspace-evals/
  bin/clank-eval
  bin/codex-eval
  snapshots/<snapshot-id>/             sanitized one-commit repositories
  snapshot-bundles/<snapshot-id>.bundle
  data/corpora/<version>/<split>/<scenario>/<sha256>/
    scenario.json                      immutable; includes hidden oracle
    prepared.json                      project/repository/alias mapping
    repo/                              agent-visible Git world; no oracle
    traces/<episode>/                  per-turn JSONL, stderr, responses, export, score
  data/secrets/<version>/<sha256>/      project credentials; mode 0600
  data/generation-runs/<workflow-id>/   immutable OmegaCode output and accepted worlds
```

Blueprint workers use `/home/exedev/clankspace-blueprint-sandbox`, outside the repository and its `.clankspace.json`. Only sanitized snapshots named by the curriculum are copied there. Run OmegaCode with `CODEX_BIN=/home/exedev/clankspace-evals/bin/codex-eval` so account plugins and unrelated MCPs cannot enter the generation context.

`synthetic-lab` is the ClankSpace control project. Every runnable world receives its own `eval-<corpus>-...` project. ClankSpace contains agent-visible coordination evidence and batch checkpoints; the external ledger alone contains hidden oracles and raw traces.

## Core commands

```bash
clank-eval snapshot --id <id> --source <repo> --ref <sha> --destination <dir> [--include <path> ...]
clank-eval validate --scenario <rendered-world.json>
clank-eval ingest-worlds --input <omegacode-output.json> --ledger <data-dir>
clank-eval prepare --scenario <world.json> --ledger <data-dir> --corpus <version> \
  --admin-env <eval-admin.env> --skill <SKILL.md> [--snapshot id=/path ...]
clank-eval validate-collaboration --scenario <two-lane-v2.json>
clank-eval prepare-collaboration --scenario <two-lane-v2.json> --ledger <data-dir> --corpus v2 \
  --admin-env <eval-admin.env> --skill <SKILL.md> --snapshot id=/sanitized-snapshot-repository
clank-eval rollout --prepared <prepared.json> --model gpt-5.6-luna --reasoning high --dry-run
clank-eval collaboration-rollout --prepared <prepared-v2.json> --repository <clean-baseline-repo> \
  --credentials-dir <secrets-dir> --episode <immutable-episode-id> --server-config <frozen-public-config> \
  --server-commit <clankspace-commit> --dry-run
```

Remove `--dry-run` only after the exact live workforce and command have been approved. A rollout starts a real persisted Codex thread, sends every prior human turn separately, resumes that same thread for the final task, and records only observable events and responses—not hidden reasoning.

`prepare-collaboration` is a separate v2 path; it does not alter v1 preparation or rollout. It content-addresses the scenario, verifies its source snapshot manifest (source URL and commit, sanitized head, bundle hash, and MIT license), builds a clean injected repository, creates one isolated shared project, seeds prior records and trajectories, and issues one distinct mode-0600 `<lane-id>.json` credential below the ledger's external secrets tree. `prepared.json` contains no token or credential path. Re-running the command resolves the same frozen prepared artifact without writing a new project or secret.

`collaboration-rollout` requires the two credential files from that external secrets tree. Its default launcher is `/home/exedev/clankspace-evals/bin/codex-eval`, matching the service-VM runner layout; another launcher must be passed explicitly with `--codex`. Before a live run it hashes and therefore preflights that executable plus the supplied server configuration. The runner first clones and launches lane A, then polls runs plus the project export until it observes the declared checkpoint from exactly one matching lane-A run. It does not clone or launch lane B before that durable observation. The timeout, an observer failure, cancellation, an ambiguous run, or lane A exiting without the checkpoint leaves lane B unstarted and writes an incomplete evidence episode.

Each live episode writes `schedule.json`, exact launched commands, observable Codex JSONL and stderr, public responses, run/project snapshots, `barrier.json`, Git results, required task-check results, lane results, `deterministic-score.json`, `collaboration.json`, `dossier.html`, and `SHA256SUMS` below `traces/<episode>/`. `collaboration.json` pins the Codex launcher hash plus supplied server commit and configuration hash. The dossier is a self-contained offline HTML file with local links only. Credential files, credential contents, and hidden ledger oracles are not copied into worktrees or report artifacts. The dry-run prints the exact planned commands and filesystem paths and does not create worktrees or launch Codex.
