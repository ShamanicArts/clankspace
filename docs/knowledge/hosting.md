---
type: knowledge
keywords: [railway, hosting, domain, sqlite, volume, backup, clank.shamanicarts.dev]
related: [docs/design/spec.md]
summary: Production target and remaining deployment logistics.
last_verified: 2026-08-02
note_created: 2026-08-02
updated: 2026-08-02
---

# Hosting

Production target is one Railway service at `clank.shamanicarts.dev` with a persistent volume mounted at `/data`. The Go process listens on Railway’s `PORT`; the database is `/data/clankspace.db`.

Railway provides domain/TLS, restarts, health checks, metrics, and volume backups. ClankSpace must also generate portable SQLite snapshots so migration away from Railway remains simple.

Deployment is blocked only on the human-owned Railway account, project connection, and DNS records. Local implementation and container verification do not depend on those credentials.

