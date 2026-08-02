---
type: knowledge
keywords: [go, sqlite, wal, fts5, events, projections, receipts, sync]
related: [docs/design/spec.md]
summary: Technical shape of the ClankSpace monolith and durable command path.
last_verified: 2026-08-02
note_created: 2026-08-02
updated: 2026-08-02
---

# Architecture

One Go binary serves JSON, dashboard assets, CLI commands, and stdio MCP. One SQLite database on a local persistent volume stores events and projections.

Every mutation validates identity/context, enters a serialized write lane, checks idempotency and revision, appends an event, updates projections/search, persists a receipt, and commits before acknowledgment.

Local caches and exports are rebuildable. Never copy a live SQLite WAL database as a synchronization mechanism.

