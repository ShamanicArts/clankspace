---
type: knowledge
summary: ClankSpace workloads on the reusable exe.dev agent-compute plane plus the temporary Railway migration source.
keywords: [exe.dev, agents, compute, deployment, sqlite, systemd, evaluation, runner, migration]
related: [../knowledge/hosting.md, railway.md]
note_created: 2026-08-02
updated: 2026-08-03
---

# exe.dev agent infrastructure

## Current status

exe.dev is a reusable, general-purpose agent execution plane. ClankSpace currently uses it for disposable evaluation services, synthetic corpora, clean model sessions, traces, judges, scoring, and Operations; future projects may run their own isolated agent services, automations, browser/CLI workers, and evaluation workloads on the same platform.

ClankSpace does not own the platform or define its future topology. Its workloads must remain namespaced, replaceable, and isolated from other services by VM or equivalent runtime boundary, credentials, storage, ports, and lifecycle. exe.dev is not the permanent home for trusted ClankSpace collaboration state.

The validated RC-009 build remains temporarily reachable at `https://clankspace-prod.exe.xyz`. It passes local/external health and readiness plus authenticated project export, and its online backup, off-host copy, restore drill, and prior-binary rollback are verified. Treat it as the Railway migration source and short-lived rollback candidate. Do not commit this origin into collaborator repositories or provision real collaborator projects there.

## VM boundaries

| VM | Purpose | Data boundary |
|---|---|---|
| `clankspace-prod` | Temporary validated Railway migration source | Freeze for migration; retire after the cutover rollback window |
| `clankspace-eval` | Resettable service for generated corpora and agent rollouts | Synthetic and explicitly copied evaluation fixtures only |
| `luna-runner` | Corpus generation, clean agent sessions, traces, and scoring | No permanent-production bootstrap or project credentials |

The temporary candidate and evaluation service use separate bootstrap tokens, databases, process users, and HTTPS origins. Raw model transcripts belong on the runner or in evaluation artifacts, never in a ClankSpace append log.

The runner stores immutable corpora under `/home/exedev/clankspace-evals/data/corpora`, sanitized real-repository snapshots under `snapshots/`, and mutable mode-0600 runtime credentials in the separate `data/secrets` subtree. Credential files remain outside agent-visible repositories, corpus artifacts, evidence bundles, and checksum manifests. The runner has the evaluation bootstrap token only; no permanent-production token is installed. Tailnet-only Operations and raw workflow views are served from the runner through local SSH forwarding.

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

After Railway passes stable-domain and restore acceptance, retain `clankspace-prod` only for the agreed rollback window, then destroy that VM and its application credentials. Keep `clankspace-eval` and `luna-runner` as the ClankSpace workload allocation on the broader agent platform; they may evolve or be replaced without changing the product service.
