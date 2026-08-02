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
- `fixtures/rendered/relaydesk-001.json`: deterministic non-model canary.

## Runner layout

```text
/home/exedev/clankspace-evals/
  bin/clank-eval
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

`synthetic-lab` is the ClankSpace control project. Every runnable world receives its own `eval-<corpus>-...` project. ClankSpace contains agent-visible coordination evidence and batch checkpoints; the external ledger alone contains hidden oracles and raw traces.

## Core commands

```bash
clank-eval snapshot --id <id> --source <repo> --ref <sha> --destination <dir> [--include <path> ...]
clank-eval validate --scenario <rendered-world.json>
clank-eval ingest-worlds --input <omegacode-output.json> --ledger <data-dir>
clank-eval prepare --scenario <world.json> --ledger <data-dir> --corpus <version> \
  --admin-env <eval-admin.env> --skill <SKILL.md> [--snapshot id=/path ...]
clank-eval rollout --prepared <prepared.json> --model gpt-5.6-luna --reasoning high --dry-run
```

Remove `--dry-run` only after the exact live workforce and command have been approved. A rollout starts a real persisted Codex thread, sends every prior human turn separately, resumes that same thread for the final task, and records only observable events and responses—not hidden reasoning.
