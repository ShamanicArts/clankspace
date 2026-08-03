---
type: knowledge
keywords: [railway, exe.dev, hosting, domain, sqlite, backup, rollback, clank.shamanicarts.dev]
related: [docs/design/spec.md, docs/deployment/railway.md, docs/deployment/exe.md]
summary: Permanent Railway production target, exe.dev evaluation boundary, and migration/recovery plan.
last_verified: 2026-08-03
note_created: 2026-08-02
updated: 2026-08-03
---

# Hosting

## Hosting boundary

ClankSpace separates durable collaboration state from disposable research infrastructure:

| Environment | Purpose | Data boundary |
|---|---|---|
| Railway at `clank.shamanicarts.dev` | Permanent trusted-project service | Real collaboration state only; one persistent volume |
| exe.dev `clankspace-eval` | Resettable candidate and synthetic-project service | Evaluation data only |
| exe.dev `luna-runner` | Corpus generation, model rollouts, traces, judges, and Operations | Evaluation credentials only; no production credential |
| exe.dev `clankspace-prod` | Temporary validated migration source | Retained only through Railway cutover and rollback window |

The existing exe.dev origin proves that the build can be hosted, backed up, restored, and rolled back. It is not the address to commit into collaborator repositories and should not receive real project onboarding.

The reserved `clank.shamanicarts.dev` hostname is not yet routed and may return 404 until cutover. Repository pointers should use that stable hostname only after Railway health, migration, authentication, backup, and restore acceptance checks pass. Bearer authentication remains mandatory for every project operation and there is no public signup.

## Permanent runtime

Railway runs the included container with one persistent volume:

```text
one Railway service
one replica
/data/clankspace.db
CLANKSPACE_BASE_URL=https://clank.shamanicarts.dev
```

The service uses one process and one SQLite writer. Never run multiple replicas against the same database volume.

## Recovery

- Enable scheduled Railway volume backups for fast platform recovery.
- Schedule SQLite online backups while the service remains live, run `PRAGMA integrity_check`, and copy completed snapshots off-provider.
- Export projects deterministically for project-level portability.
- Before cutover, verify health, readiness, authentication, project counts, and an authenticated export on Railway.
- Rehearse a disposable restore before onboarding collaborators and periodically thereafter.
- Keep the exe.dev candidate and its prior binary only for the defined rollback window, then retire that VM.

Never copy the live database plus WAL files piecemeal or place the live data directory in a sync drive.

## Portability

ClankSpace remains one Go binary and one SQLite database. The stable domain prevents a future host move from changing repository pointers, project semantics, or the CLI/MCP protocol.

## Migration sequence

1. Take a fresh online backup on exe.dev, verify integrity and checksum, and keep an off-provider copy.
2. Provision the Railway service, persistent `/data` volume, secrets, and one-replica policy.
3. Restore the snapshot and compare project/record counts plus authenticated export.
4. Verify `/healthz`, `/readyz`, the dashboard, CLI, and MCP against the managed Railway origin.
5. Add `clank.shamanicarts.dev`, complete DNS/TLS, and repeat external checks through the stable domain.
6. Enable scheduled platform and off-provider backups, rehearse restore, then onboard collaborators.
7. After the rollback window, retire exe.dev `clankspace-prod`; retain `clankspace-eval` and `luna-runner` for research.
