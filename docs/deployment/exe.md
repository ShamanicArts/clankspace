---
type: knowledge
summary: Current small production deployment plus disposable, on-demand evaluation infrastructure on exe.dev.
keywords: [exe.dev, agents, compute, deployment, sqlite, systemd, evaluation, runner, migration]
related: [../knowledge/hosting.md, railway.md]
note_created: 2026-08-02
updated: 2026-08-04
---

# exe.dev agent infrastructure

## Current status

exe.dev is a reusable, general-purpose execution plane. ClankSpace currently uses one VM for its small trusted production pilot. Evaluation services, synthetic corpora, clean model sessions, traces, judges, scoring, and Operations use separately provisioned disposable VMs only while a campaign is active.

ClankSpace does not own the platform or define its future topology. Its workloads remain namespaced and isolated from other services by VM, credentials, storage, ports, and lifecycle. The stable domain and verified SQLite backups keep production portable to Railway or another host later.

Merged main `da2f515` is live at `https://clank.shamanicarts.dev`; `https://clankspace-prod.exe.xyz` is only the provider origin. The deployment serves one-prompt repository onboarding and the task guide at `/docs/`, runs schema version 11, and has signed synchronization enabled. Authentication intentionally remains `bootstrap` until SMTP is configured. The current database completed a round trip through Railway and was restored from a verified off-provider snapshot. Do not commit the provider origin into collaborator repositories.

## VM boundaries

| VM | Purpose | Data boundary |
|---|---|---|
| `clankspace-prod` | Current trusted pilot service | Real collaboration state; stable domain; off-provider backup required |
| `clankspace-eval` | Not provisioned by default | Recreate as a resettable synthetic-only service for an active campaign |
| `luna-runner` | Not provisioned by default | Recreate for corpus generation, clean sessions, traces, and scoring |

When provisioned, production, evaluation, and runner workloads use separate credentials, databases, process users, and HTTPS origins. Raw model transcripts belong in exported runner artifacts, never in a ClankSpace append log.

Before a runner is destroyed, immutable corpora, sanitized repository snapshots, workflow journals, and evidence bundles are exported to a mode-0700 local archive and checksum-verified. Mutable runtime credentials are not archived; they are revoked or allowed to die with the disposable VM. A reprovisioned runner receives fresh evaluation-only credentials and never a production token.

## Service shape

The same static `clank` binary runs under [`deploy/exe/clankspace.service`](../../deploy/exe/clankspace.service). Each temporary service listens on port `8000`, which exe.dev proxies with managed TLS.

Runtime files:

```text
/usr/local/bin/clank
/etc/clankspace/clankspace.env       mode 0600, root-owned
/var/lib/clankspace/clankspace.db    clankspace-owned
/etc/systemd/system/clankspace.service
```

Required environment:

```text
PORT=8000
CLANKSPACE_DATA_DIR=/var/lib/clankspace
CLANKSPACE_BASE_URL=https://<vm>.exe.xyz
CLANKSPACE_BOOTSTRAP_TOKEN=<generated secret>
CLANKSPACE_WORKSPACE_NAME=<candidate or evaluation name>
CLANKSPACE_OWNER_NAME=Shamanic
```

An exe.dev proxy remains private until the service passes local `/healthz` and `/readyz` checks. Evaluation origins are exposed only when a test requires generic CLI/MCP access. ClankSpace bearer tokens remain the application authorization boundary; there is no public signup.

## Verification

For each service:

1. Verify `systemctl is-active clankspace`.
2. Verify local `http://127.0.0.1:8000/healthz` and `/readyz`.
3. Expose the proxy only when the evaluation needs an external CLI/MCP client.
4. Verify the HTTPS origin from outside the VM.
5. Create a disposable project, issue a project-scoped token, use it from a clean client, and export the project.
6. Create an online backup, run `PRAGMA integrity_check`, copy it off-host, and retain the current binary for rollback.
7. Restore that completed snapshot into a disposable data directory, start a separate service instance against it, and verify health, readiness, and an authenticated project export before removing the disposable instance.
8. Verify an authenticated project context and export from an external client against the deployed service.

Never copy the live SQLite database or individual WAL files while the service is running. Use SQLite's online `.backup` path, verify the resulting database, and copy only that completed snapshot off-host.

Keep only `clankspace-prod` while the pilot is active. Export and delete evaluation/runner VMs after each campaign. Recreate them from repository automation and fresh credentials when the next product question warrants model spend.
