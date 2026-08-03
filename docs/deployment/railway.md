---
type: knowledge
summary: Portable Railway alternative; not the active ClankSpace production topology.
keywords: [railway, deployment, volume, sqlite, backup, dns]
related: [../knowledge/hosting.md]
note_created: 2026-08-02
updated: 2026-08-03
---

# Railway deployment alternative

ClankSpace production currently runs on exe.dev. This document is retained because the product remains portable to one Railway service and one persistent SQLite volume; it is not the current deployment checklist or project blocker. See [exe.dev deployment](exe.md) for the live topology.

## Shape

Deploy one container and attach one persistent volume at `/data`. Never run two replicas against the same SQLite volume. The container respects Railway's `PORT`; `/readyz` verifies database reachability.

Required variables:

```text
CLANKSPACE_BOOTSTRAP_TOKEN=<long random value>
CLANKSPACE_WORKSPACE_NAME=ShamanicArts
CLANKSPACE_OWNER_NAME=Shamanic
CLANKSPACE_BASE_URL=https://clank.shamanicarts.dev
```

Optional `GITHUB_TOKEN` raises the public GitHub API rate limit. It does not enable private repositories in the pilot.

## Railway setup

1. Create a Hobby project from the private GitHub repository.
2. Add a persistent volume mounted at `/data`.
3. Set the variables above and deploy with the included `Dockerfile` and `railway.toml`.
4. Add `clank.shamanicarts.dev` as a custom domain, then create the DNS record Railway requests.
5. Verify `/readyz`, open the dashboard, authenticate, create a disposable project, export it, and remove it only after the pilot has a supported delete workflow.

## Backup and restore

Enable Railway volume backups. For a provider-neutral logical copy, run `clank project export` for each project. Do not copy a live `.db-wal` piecemeal or place the live SQLite directory in a sync drive.

A portable full-database backup can use SQLite's online backup operation, followed by `PRAGMA integrity_check` and an off-provider copy. Railway volume snapshots remain useful for quick platform recovery; deterministic project JSON exports provide project-level portability.

## Current external handoff

Moving the active pilot to Railway would require a human-owned Railway project, persistent volume, environment secrets, custom-domain/DNS configuration, health verification, and a restore rehearsal. It is an optional hosting migration, not unfinished core product work.
