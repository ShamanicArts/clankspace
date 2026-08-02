---
type: meta
status: active
summary: Frontmatter contract for structured project documentation.
note_created: 2026-08-02
updated: 2026-08-02
---

# Project Frontmatter Schema

Every structured document includes `type`, `summary`, `note_created`, and `updated`.

| Type | Additional required fields |
|---|---|
| `devlog` | `status`, `session_start`, `session_end`, `phase`, `approval` |
| `phase` | `status`, `phase_number`, `prerequisite`, `estimated_effort` |
| `knowledge` | `keywords`, `related`, optional `last_verified` |
| `lessons` | none |
| `state` | `status` |
| `spec` | `status` (`draft`, `approved`, or `superseded`) |

Dates use `YYYY-MM-DD`; times use 24-hour `HH:MM`. Quote YAML values containing `: `.

