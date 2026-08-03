---
type: knowledge
summary: Private ClankSpace pilot and isolated evaluation deployment on exe.dev VMs.
keywords: [exe.dev, deployment, sqlite, systemd, evaluation, isolation]
related: [../knowledge/hosting.md]
note_created: 2026-08-02
updated: 2026-08-02
---

# exe.dev deployment

## VM boundaries

| VM | Purpose | Data boundary |
|---|---|---|
| `clankspace-prod` | Stable pilot for real project spaces | No synthetic data and no test-agent execution |
| `clankspace-eval` | Resettable service for generated corpora and agent rollouts | Synthetic and explicitly copied evaluation fixtures only |
| `luna-runner` | Corpus generation, clean agent sessions, traces, and scoring | No production bootstrap or project credentials |

Production and evaluation use separate bootstrap tokens, databases, process users, and HTTPS origins. Raw model transcripts belong on the runner or in evaluation artifacts, never in either ClankSpace append log.

The runner stores immutable corpora under `/home/exedev/clankspace-evals/data`, sanitized real-repository snapshots under `snapshots/`, and project credentials under the corpus-versioned `data/secrets/` tree. It has the evaluation bootstrap token only; no production token is installed.

## Service shape

The same static `clank` binary runs under [`deploy/exe/clankspace.service`](../../deploy/exe/clankspace.service). Each service listens on port `8000`, which exe.dev proxies with managed TLS.

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
CLANKSPACE_WORKSPACE_NAME=<pilot or evaluation name>
CLANKSPACE_OWNER_NAME=Shamanic
```

The exe.dev proxy remains private until the service passes local `/healthz` and `/readyz` checks. It is then made public because generic CLI/MCP clients cannot complete exe.dev's interactive web login. ClankSpace bearer tokens remain the application authorization boundary; there is no public signup.

## Verification

For each service:

1. Verify `systemctl is-active clankspace`.
2. Verify local `http://127.0.0.1:8000/healthz` and `/readyz`.
3. Set the exe.dev proxy to port `8000` and public visibility.
4. Verify the HTTPS origin from outside the VM.
5. Create a disposable project, issue a project-scoped token, use it from a clean client, and export the project.
6. Perform an off-host backup and restore drill before storing real collaboration context.

Never copy the live SQLite database or individual WAL files while the service is running. Use the SQLite online backup path or continuous replication once implemented.
