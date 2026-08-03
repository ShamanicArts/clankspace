---
type: knowledge
keywords: [railway, exe.dev, hosting, domain, sqlite, backup, rollback, clank.shamanicarts.dev]
related: [docs/design/spec.md, docs/deployment/railway.md, docs/deployment/exe.md]
summary: Permanent Railway production, ClankSpace's allocation on the reusable exe.dev agent plane, and migration/recovery.
last_verified: 2026-08-03
note_created: 2026-08-02
updated: 2026-08-03
---

# Hosting

## Hosting boundary

ClankSpace separates durable collaboration state from reusable agent compute. exe.dev is a broader execution platform for current and future agent services; the rows below describe only ClankSpace's isolated allocation on it.

| Environment | Purpose | Data boundary |
|---|---|---|
| Railway at `clank.shamanicarts.dev` | Live permanent trusted-project service; managed TLS pending | Real collaboration state only; one persistent volume |
| exe.dev `clankspace-eval` | ClankSpace's resettable candidate and synthetic-project service | Evaluation data only; isolated from other agent services |
| exe.dev `luna-runner` | ClankSpace corpus generation, model rollouts, traces, judges, and Operations | Evaluation credentials only; no production credential |
| exe.dev `clankspace-prod` | Stopped migration rollback source | Frozen through the Railway rollback window, then retired |

The existing exe.dev origin proved that the build could be hosted, backed up, restored, and rolled back. It is stopped and must not receive new writes or collaborator traffic.

The Railway-managed origin is healthy and serves the restored project data. Cloudflare publishes a DNS-only CNAME for `clank.shamanicarts.dev` plus Railway's ownership TXT record. Railway recognizes traffic routing as propagated; certificate issuance is pending. Repository pointers should use the stable hostname only after valid TLS plus backup and restore acceptance pass. Bearer authentication remains mandatory for every project operation and there is no public signup.

## Permanent runtime

Railway runs the included container with one persistent volume:

```text
one Railway service
one replica
/data/clankspace.db
CLANKSPACE_BASE_URL=https://clank.shamanicarts.dev
```

The service uses one process and one SQLite writer. Never run multiple replicas against the same database volume.

The live project is human-owned, runs one replica in Railway's EU West region, and mounts its colocated volume at `/data`. The container prepares only the mount and SQLite files as root, then executes the service as the unprivileged `clank` user.

## Recovery

- On Railway Pro, enable scheduled volume backups for fast platform recovery. On Trial/Hobby, an external online-backup schedule is mandatory before collaborator onboarding because native volume backups/PITR are unavailable.
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
7. After the rollback window, retire exe.dev `clankspace-prod`; retain or replace the isolated ClankSpace eval/runner allocation without constraining other agent services on the platform.
