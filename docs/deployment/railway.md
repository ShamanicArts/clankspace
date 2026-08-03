---
type: knowledge
summary: Permanent ClankSpace production target and exe.dev-to-Railway migration runbook.
keywords: [railway, deployment, volume, sqlite, backup, dns, migration]
related: [../knowledge/hosting.md, exe.md]
note_created: 2026-08-02
updated: 2026-08-03
---

# Railway permanent production

Railway is the permanent target for the trusted ClankSpace service. exe.dev remains the evaluation and runner control plane; its current `clankspace-prod` VM is only the validated migration source. The stable client contract is `https://clank.shamanicarts.dev`, not a provider hostname.

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

1. Create a human-owned Railway project from this GitHub repository. Hobby is sufficient for an initial low-traffic trusted pilot; move to Pro when team access or support requirements justify it.
2. Add a persistent volume mounted at `/data`.
3. Set the variables above and deploy with the included `Dockerfile` and `railway.toml`.
4. Pin the service to one replica. SQLite cannot safely scale horizontally across this volume.
5. Verify `/healthz` and `/readyz` on the Railway-provided origin before restoring data.

## Migration and cutover

1. Freeze writes to the temporary exe.dev candidate.
2. Create a fresh SQLite online backup, run `PRAGMA integrity_check`, record its SHA-256, and copy the completed snapshot off-provider.
3. Restore that snapshot into the Railway `/data` volume using a one-off maintenance command or Railway's current volume file-access mechanism. Never copy a live database and WAL piecemeal.
4. Start the service and compare project, note, trajectory, event, and receipt counts with the source. Verify an authenticated project context and deterministic export.
5. Exercise the dashboard, CLI, and MCP against the Railway-provided origin. Confirm advisory-authority notices and project isolation.
6. Add `clank.shamanicarts.dev` as a custom domain, create the DNS record Railway requests, wait for managed TLS, and repeat external health, readiness, authentication, and export checks through the stable hostname.
7. Commit only the stable hostname into collaborator repository pointers.
8. Keep the exe.dev candidate stopped but recoverable for a short rollback window. If Railway fails acceptance, return DNS to the frozen source and reconcile no new writes. After the window, retire the candidate VM and credentials.

## Backup and restore

Enable scheduled Railway volume backups—daily, weekly, and monthly as appropriate—for fast same-platform recovery. For a provider-neutral logical copy, run `clank project export` for each project. Do not copy a live `.db-wal` piecemeal or place the live SQLite directory in a sync drive.

A portable full-database backup uses SQLite's online backup operation, followed by `PRAGMA integrity_check`, a checksum, and an encrypted off-provider copy. Railway backups belong to the Railway project/environment recovery boundary, so they do not replace an off-provider snapshot. Deterministic project JSON exports provide project-level portability.

Before onboarding the first collaborator, restore the latest completed backup into a disposable service/volume and verify health, readiness, counts, and authenticated export. Repeat restore drills periodically and before material infrastructure changes.

## Acceptance gate

Permanent production is ready only when all of these pass:

- one Railway service, one replica, and a persistent `/data` volume;
- managed-origin and stable-domain health/readiness;
- authenticated CLI context, brief, write, and export;
- dashboard access and project isolation;
- source/destination count and export comparison;
- scheduled Railway backups plus a verified off-provider online backup;
- disposable restore rehearsal;
- documented DNS, application, and data rollback;
- no real project repository points at the exe.dev origin.

Official platform references: [volumes](https://docs.railway.com/volumes/reference), [scheduled backups](https://docs.railway.com/volumes/backups), [custom domains](https://docs.railway.com/networking/domains), and [pricing](https://docs.railway.com/pricing).
