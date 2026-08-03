---
type: knowledge
keywords: [railway, exe.dev, hosting, domain, sqlite, backup, rollback, clank.shamanicarts.dev]
related: [docs/design/spec.md, docs/deployment/railway.md, docs/deployment/exe.md]
summary: Current hosted pilot, standalone self-hosting, and the signed replica boundary between them.
last_verified: 2026-08-03
note_created: 2026-08-02
updated: 2026-08-03
---

# Hosting

## Hosting boundary

ClankSpace separates durable collaboration state from disposable evaluation compute. A stable domain, signed workspace events, portable bundles, and verified SQLite backups are the portability contract; provider VMs are replaceable.

| Environment | Purpose | Data boundary |
|---|---|---|
| exe.dev `clankspace-prod` via `clank.shamanicarts.dev` | Live trusted-project pilot | Real collaboration state only; one SQLite writer |
| Railway project and volume | Dormant migration fallback; no active deployment | Retained temporarily after the verified round trip |
| Local runner/eval archives | Research evidence and reproducibility material | Checksummed; credentials excluded |
| Reprovisioned eval/runner VMs | Active campaigns only | Synthetic data and fresh evaluation credentials only |

The current exe.dev origin is the small-network hosted pilot. Cloudflare publishes a DNS-only CNAME for `clank.shamanicarts.dev`, and exe.dev owns its managed certificate. The service remains invitation-only: there is no public signup.

## Current runtime

exe.dev runs the static binary under systemd:

```text
one VM
one clank process
/var/lib/clankspace/clankspace.db
CLANKSPACE_BASE_URL=https://clank.shamanicarts.dev
CLANKSPACE_AUTH_MODE=hybrid
CLANKSPACE_SYNC_ENABLED=true
```

The service uses one process and one SQLite writer. Never run multiple replicas against the same database volume.

The service runs as the unprivileged `clankspace` user. systemd owns startup and restart behavior; exe.dev terminates TLS and proxies the stable custom domain to port 8000.

Hosted mode also needs a durable installation secret and an SMTP sender. The installation secret encrypts the local replica signing key, mail outbox bodies, and stored replica credentials. It must be backed up separately from the database and restored with it. A file mail sink is for local E2E testing only.

## Replication boundary

- Every linked workspace names one authority instance for project structure and replica admission.
- A cloud-created workspace remains cloud-authoritative when a local instance joins it.
- A self-host-created workspace stays self-host-authoritative when it is mirrored to cloud.
- Approved replicas may append runs, notes, trajectories, and lifecycle edges offline, then synchronize later.
- Instances exchange bounded signed snapshots and domain events over HTTPS. They never share a live database or WAL.
- User accounts, email addresses, sessions, invitations, API keys, SMTP state, and replica credentials remain local to each host.
- Direct peers use the same explicit authority offer. There is no automatic discovery or transitive mesh.

## Recovery

- Schedule SQLite online backups while the service remains live, run `PRAGMA integrity_check`, and copy completed snapshots off-provider.
- The current operator-side persistent timer performs that flow daily and records a SHA-256 manifest; its initial live run passed.
- Export projects deterministically for project-level portability.
- Before any cutover, verify health, readiness, authentication, project counts, and an authenticated export on the target host.
- Rehearse a disposable restore before onboarding collaborators and periodically thereafter.
- Retain the prior binary and pre-migration snapshot for bounded rollback.

Never copy the live database plus WAL files piecemeal or place the live data directory in a sync drive.

## Portability

ClankSpace remains one Go binary and one SQLite database per instance. The stable domain prevents a future host move from changing repository pointers, project semantics, or the CLI/MCP protocol. Workspace bundles and signed replication move domain history between instances without making a provider's filesystem the data model.

## Host migration sequence

1. Freeze writes and create a fresh SQLite online backup.
2. Verify `PRAGMA integrity_check`, record a checksum, and retain an off-provider copy.
3. Provision one target service, persistent local filesystem, and matching production bootstrap credential.
4. Restore the snapshot and compare project/record counts plus authenticated export.
5. Verify health, readiness, dashboard, CLI, and MCP against the provider origin.
6. Repoint `clank.shamanicarts.dev`, claim managed TLS on the target, and repeat external checks.
7. Remove the old runtime only after the target passes; retain a bounded rollback snapshot rather than two writers.
