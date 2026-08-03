---
type: knowledge
keywords: [exe.dev, hosting, domain, sqlite, backup, rollback, clank.shamanicarts.dev]
related: [docs/design/spec.md, docs/deployment/exe.md]
summary: Active production topology, recovery boundary, and remaining stable-domain work.
last_verified: 2026-08-03
note_created: 2026-08-02
updated: 2026-08-03
---

# Hosting

## Active topology

ClankSpace currently uses three isolated exe.dev VMs:

| Service | Purpose | Data boundary |
|---|---|---|
| `clankspace-prod` | Trusted real collaborator projects | No synthetic agents or evaluation fixtures |
| `clankspace-eval` | Resettable candidate and synthetic-project service | Evaluation data only |
| `luna-runner` | Corpus generation, real model rollouts, traces, judges, and Operations | No production credential |

Production is reachable at `https://clankspace-prod.exe.xyz`. The origin is public so generic CLI and MCP clients can connect without exe.dev's interactive login; bearer authentication remains mandatory for every project operation. There is no public signup.

The reserved `clank.shamanicarts.dev` hostname is not yet routed to production and currently returns 404. Repository pointers should use the exe.dev origin until that route is finished, then migrate in one normal config change.

## Runtime

The production binary runs under a restricted systemd unit as the dedicated `clankspace` user:

```text
/usr/local/bin/clank
/etc/clankspace/clankspace.env
/var/lib/clankspace/clankspace.db
/etc/systemd/system/clankspace.service
```

The service uses one process and one SQLite writer. Do not run multiple replicas against the same database volume.

## Recovery

- SQLite online backup is performed while the service remains live.
- Each deployment takes a fresh integrity-checked backup and copies it off-host with mode `0600`.
- The replaced production binary is retained under an explicit rollback name.
- Deployment uses same-filesystem atomic replacement, restart polling, local/external health and readiness checks, and authenticated project export.
- A disposable restore drill has passed against the production backup shape.

Never copy the live database plus WAL files piecemeal or place the live data directory in a sync drive.

## Portability

ClankSpace remains one Go binary and one SQLite database. The Docker and Railway configuration are retained as alternative deployment targets. Moving hosts changes the service URL and operational runbook, not project semantics or the CLI/MCP protocol.

## Remaining operations work

- route `clank.shamanicarts.dev` to the production origin;
- schedule the proven online backup and off-host copy instead of invoking them only around deployments;
- add a small external health monitor;
- publish checksummed client binaries;
- add supported token listing, revocation, and rotation before expanding beyond the trusted group.
