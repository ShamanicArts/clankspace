---
type: devlog
status: complete
session_start: "15:39"
session_end: "16:08"
phase: "dogfood"
subphase: "native-jj-provenance"
approval: user-directed
summary: Extend run provenance from Git-only delivery coordinates to native Jujutsu change evolution while retaining colocated GitHub evidence.
note_created: 2026-08-05
updated: 2026-08-05
---

# Native Jujutsu provenance

## Intent

ClankSpace already attached Git branch, commit, pull-request, and merge evidence to a run, but
JJ work appeared only through the backing Git repository. Preserve the JJ identity that matters
to collaborators without weakening the existing GitHub delivery path.

## Direction

- Keep delivery provenance on the run so linked notes and checkpoints inherit it.
- Record the JJ workspace, stable change ID, commit ID, and local bookmarks at run start.
- Record the same native coordinates at run end or later link time so a rewritten change retains
  its identity while exposing the starting and delivered commits.
- In colocated repositories, retain both JJ provenance and Git/GitHub evidence.
- Keep every field optional and never guess when the local tool cannot provide it.

## Delivered

- Added schema 14 and full domain/store/service/API/MCP/export support for native JJ origin and
  delivery fields on runs, plus inherited origin fields on trajectories.
- Extended event replication and signed workspace snapshots to validate and preserve JJ fields.
- Added CLI detection for native and colocated JJ repositories, including repository remote,
  workspace, full change/commit IDs, bookmarks, and bookmark-aware GitHub PR lookup.
- Added a one-retry legacy fallback with a visible warning so a new CLI can still start, link, and
  close runs against a pre-schema-14 server during a rolling upgrade.
- Added explicit CLI overrides and updated context/help output, the portable skill, design spec,
  hosted replication spec, and focused knowledge files.
- Updated the quiet project log to show Git and JJ provenance as separate wrapping facts.

## Verification

- Focused store, service, HTTP, sync-client, client, MCP, and CLI tests passed.
- Full `go test ./...`, `go vet ./...`, `go build ./cmd/clank`, JavaScript syntax, and whitespace
  checks passed.
- A controlled schema-14 service round-tripped a JJ run whose stable change ID stayed fixed while
  its commit, workspace, and delivery coordinates changed.
- The live pre-schema-14 service rejected the first JJ-aware end request as expected; the rollout
  fallback test now proves all three run mutations retry without JJ fields while preserving Git data.
- Desktop and iPhone-sized browser passes rendered the origin and delivered JJ provenance without
  horizontal overflow; the isolated service and fixture were removed after verification.
