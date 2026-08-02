---
type: knowledge
summary: Railway pilot deployment and recovery procedure.
keywords: [railway, deployment, volume, sqlite, backup, dns]
related: [../knowledge/hosting.md]
note_created: 2026-08-02
updated: 2026-08-02
---

# Railway pilot deployment

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

A full portable database backup command will use SQLite's online backup API in a later hardening phase. Until then, Railway volume snapshots are the full-instance recovery mechanism and JSON exports are the project-level portability mechanism.

## Current external handoff

The repository contains everything needed to deploy, but this bootstrap session had no Railway account/project credentials and no Railway CLI. Creating the service, volume, custom domain, DNS record, and backup schedule therefore remains a human-authenticated deployment step rather than a code blocker.
