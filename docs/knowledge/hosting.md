---
type: knowledge
keywords: [railway, exe.dev, hosting, domain, sqlite, backup, rollback, clank.shamanicarts.dev]
related: [docs/design/spec.md, docs/deployment/railway.md, docs/deployment/exe.md]
summary: Current exe.dev pilot, disposable evaluation infrastructure, and the validated Railway migration path.
last_verified: 2026-08-03
note_created: 2026-08-02
updated: 2026-08-03
---

# Hosting

## Hosting boundary

ClankSpace separates durable collaboration state from disposable evaluation compute. The stable domain and SQLite backup are the portable contract; provider VMs are replaceable.

| Environment | Purpose | Data boundary |
|---|---|---|
| exe.dev `clankspace-prod` via `clank.shamanicarts.dev` | Live trusted-project pilot | Real collaboration state only; one SQLite writer |
| Railway project and volume | Dormant migration fallback; no active deployment | Retained temporarily after the verified round trip |
| Local runner/eval archives | Research evidence and reproducibility material | Checksummed; credentials excluded |
| Reprovisioned eval/runner VMs | Active campaigns only | Synthetic data and fresh evaluation credentials only |

The current exe.dev origin serves the Railway-restored database. Cloudflare publishes a DNS-only CNAME for `clank.shamanicarts.dev`, and exe.dev owns its managed certificate. Strict-HTTPS health/readiness, CLI context, and authenticated export pass. Bearer authentication remains mandatory and there is no public signup.

## Current runtime

exe.dev runs the static binary under systemd:

```text
one VM
one clank process
/var/lib/clankspace/clankspace.db
CLANKSPACE_BASE_URL=https://clank.shamanicarts.dev
```

The service uses one process and one SQLite writer. Never run multiple replicas against the same database volume.

The service runs as the unprivileged `clankspace` user. systemd owns startup and restart behavior; exe.dev terminates TLS and proxies the stable custom domain to port 8000.

## Recovery

- Schedule SQLite online backups while the service remains live, run `PRAGMA integrity_check`, and copy completed snapshots off-provider.
- Export projects deterministically for project-level portability.
- Before any cutover, verify health, readiness, authentication, project counts, and an authenticated export on the target host.
- Rehearse a disposable restore before onboarding collaborators and periodically thereafter.
- Retain the prior binary and pre-migration snapshot for bounded rollback.

Never copy the live database plus WAL files piecemeal or place the live data directory in a sync drive.

## Portability

ClankSpace remains one Go binary and one SQLite database. The stable domain prevents a future host move from changing repository pointers, project semantics, or the CLI/MCP protocol.

## Host migration sequence

1. Freeze writes and create a fresh SQLite online backup.
2. Verify `PRAGMA integrity_check`, record a checksum, and retain an off-provider copy.
3. Provision one target service, persistent local filesystem, and matching production bootstrap credential.
4. Restore the snapshot and compare project/record counts plus authenticated export.
5. Verify health, readiness, dashboard, CLI, and MCP against the provider origin.
6. Repoint `clank.shamanicarts.dev`, claim managed TLS on the target, and repeat external checks.
7. Remove the old runtime only after the target passes; retain a bounded rollback snapshot rather than two writers.
