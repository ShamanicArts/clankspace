package store

const schema = `
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;

CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workspaces (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS principals (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id),
  display_name TEXT NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('human','project')),
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS api_tokens (
  id TEXT PRIMARY KEY,
  principal_id TEXT NOT NULL REFERENCES principals(id),
  token_hash TEXT NOT NULL UNIQUE,
  token_prefix TEXT NOT NULL,
  scopes_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  revoked_at TEXT
);

CREATE TABLE IF NOT EXISTS projects (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id),
  slug TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  UNIQUE(workspace_id, slug)
);

CREATE TABLE IF NOT EXISTS project_principals (
  project_id TEXT NOT NULL REFERENCES projects(id),
  principal_id TEXT NOT NULL REFERENCES principals(id),
  role TEXT NOT NULL CHECK(role IN ('agent','viewer')),
  created_at TEXT NOT NULL,
  PRIMARY KEY(project_id, principal_id)
);

CREATE TABLE IF NOT EXISTS repositories (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id),
  url TEXT NOT NULL,
  host TEXT NOT NULL,
  owner TEXT NOT NULL,
  name TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'public',
  description TEXT NOT NULL DEFAULT '',
  default_branch TEXT NOT NULL DEFAULT '',
  stars INTEGER NOT NULL DEFAULT 0,
  etag TEXT NOT NULL DEFAULT '',
  synced_at TEXT,
  sync_error TEXT NOT NULL DEFAULT '',
  UNIQUE(workspace_id, host, owner, name)
);

CREATE TABLE IF NOT EXISTS project_repositories (
  project_id TEXT NOT NULL REFERENCES projects(id),
  repository_id TEXT NOT NULL REFERENCES repositories(id),
  is_primary INTEGER NOT NULL DEFAULT 0,
  path_scope TEXT NOT NULL DEFAULT '',
  PRIMARY KEY(project_id, repository_id)
);

CREATE TABLE IF NOT EXISTS agents (
  id TEXT PRIMARY KEY,
  principal_id TEXT NOT NULL REFERENCES principals(id),
  name TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(principal_id, name)
);

CREATE TABLE IF NOT EXISTS runs (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id),
  agent_id TEXT NOT NULL REFERENCES agents(id),
  principal_id TEXT NOT NULL REFERENCES principals(id),
  harness TEXT NOT NULL DEFAULT '',
  harness_version TEXT NOT NULL DEFAULT '',
  provider TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  reasoning TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL,
  parent_run_id TEXT,
  root_run_id TEXT,
  run_type TEXT NOT NULL,
  permission_mode TEXT NOT NULL DEFAULT '',
  interaction_mode TEXT NOT NULL DEFAULT '',
  repository_id TEXT,
  branch TEXT NOT NULL DEFAULT '',
  worktree TEXT NOT NULL DEFAULT '',
  base_sha TEXT NOT NULL DEFAULT '',
  head_sha TEXT NOT NULL DEFAULT '',
  objective TEXT NOT NULL DEFAULT '',
  instruction_profile_json TEXT NOT NULL DEFAULT '[]',
  started_at TEXT NOT NULL,
  ended_at TEXT,
  outcome TEXT NOT NULL DEFAULT '',
  verification TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS notes (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id),
  run_id TEXT REFERENCES runs(id),
  principal_id TEXT NOT NULL REFERENCES principals(id),
  kind TEXT NOT NULL CHECK(kind IN ('intent','decision','understanding','observation','checkpoint')),
  title TEXT NOT NULL,
  summary TEXT NOT NULL,
  rationale TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK(status IN ('current','superseded','stale','contested','withdrawn')),
  led_by TEXT NOT NULL CHECK(led_by IN ('human','agent','joint','external')),
  direction_basis TEXT NOT NULL,
  confidence TEXT NOT NULL,
  verification TEXT NOT NULL DEFAULT '',
  source_ref TEXT NOT NULL DEFAULT '',
  paths_json TEXT NOT NULL DEFAULT '[]',
  repository_id TEXT,
  pull_request_url TEXT NOT NULL DEFAULT '',
  revision INTEGER NOT NULL DEFAULT 1,
  superseded_by TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
  note_id UNINDEXED,
  project_id UNINDEXED,
  title,
  summary,
  rationale,
  tokenize = 'porter unicode61'
);

CREATE TABLE IF NOT EXISTS trajectories (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id),
  run_id TEXT NOT NULL REFERENCES runs(id),
  principal_id TEXT NOT NULL REFERENCES principals(id),
  objective TEXT NOT NULL,
  rationale TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK(status IN ('active','paused','closed','abandoned')),
  paths_json TEXT NOT NULL DEFAULT '[]',
  repository_id TEXT,
  branch TEXT NOT NULL DEFAULT '',
  base_sha TEXT NOT NULL DEFAULT '',
  head_sha TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS external_artifacts (
  id TEXT PRIMARY KEY,
  repository_id TEXT NOT NULL REFERENCES repositories(id),
  kind TEXT NOT NULL,
  external_id TEXT NOT NULL,
  title TEXT NOT NULL,
  url TEXT NOT NULL,
  state TEXT NOT NULL,
  author TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  UNIQUE(repository_id, kind, external_id)
);

CREATE TABLE IF NOT EXISTS events (
  sequence INTEGER PRIMARY KEY AUTOINCREMENT,
  id TEXT NOT NULL UNIQUE,
  workspace_id TEXT NOT NULL,
  project_id TEXT,
  principal_id TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  run_id TEXT,
  type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  occurred_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS receipts (
  id TEXT PRIMARY KEY,
  actor_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  event_id TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  response_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(actor_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_projects_workspace ON projects(workspace_id);
CREATE INDEX IF NOT EXISTS idx_project_principals_principal ON project_principals(principal_id, project_id);
CREATE INDEX IF NOT EXISTS idx_runs_project ON runs(project_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_notes_project ON notes(project_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_trajectories_project ON trajectories(project_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_workspace_sequence ON events(workspace_id, sequence);
`
