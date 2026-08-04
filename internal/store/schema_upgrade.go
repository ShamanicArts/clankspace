package store

import (
	"database/sql"
	"fmt"
)

const latestSchemaVersion = 12

const hostedReplicationSchema = `
ALTER TABLE workspaces ADD COLUMN slug TEXT NOT NULL DEFAULT '';
ALTER TABLE workspaces ADD COLUMN authority_replica_id TEXT NOT NULL DEFAULT '';
ALTER TABLE workspaces ADD COLUMN created_by_user_id TEXT;
UPDATE workspaces SET slug = lower(replace(name, ' ', '-')) || '-' || substr(id, -6) WHERE slug = '';
CREATE UNIQUE INDEX idx_workspaces_slug ON workspaces(slug);

ALTER TABLE principals ADD COLUMN user_id TEXT;
ALTER TABLE principals ADD COLUMN portable_actor_id TEXT NOT NULL DEFAULT '';
ALTER TABLE principals ADD COLUMN created_by_membership_id TEXT;
UPDATE principals SET portable_actor_id = id WHERE portable_actor_id = '';
CREATE UNIQUE INDEX idx_principals_portable_actor ON principals(workspace_id, portable_actor_id);

ALTER TABLE api_tokens ADD COLUMN issued_by_membership_id TEXT;
ALTER TABLE api_tokens ADD COLUMN expires_at TEXT;
ALTER TABLE api_tokens ADD COLUMN last_used_at TEXT;

CREATE TABLE users (
  id TEXT PRIMARY KEY,
  email_normalized TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('active','suspended')),
  created_at TEXT NOT NULL,
  last_login_at TEXT
);

CREATE TABLE workspace_memberships (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id),
  user_id TEXT NOT NULL REFERENCES users(id),
  principal_id TEXT NOT NULL REFERENCES principals(id),
  role TEXT NOT NULL CHECK(role IN ('owner','member')),
  status TEXT NOT NULL CHECK(status IN ('active','suspended')),
  created_at TEXT NOT NULL,
  UNIQUE(workspace_id, user_id),
  UNIQUE(principal_id)
);

CREATE TABLE workspace_invites (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id),
  email_normalized TEXT NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('owner','member')),
  token_hash TEXT NOT NULL UNIQUE,
  invited_by_membership_id TEXT NOT NULL REFERENCES workspace_memberships(id),
  expires_at TEXT NOT NULL,
  accepted_by_user_id TEXT REFERENCES users(id),
  accepted_at TEXT,
  revoked_at TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE login_challenges (
  id TEXT PRIMARY KEY,
  email_normalized TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TEXT NOT NULL,
  consumed_at TEXT,
  request_fingerprint_hash TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE browser_sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  token_hash TEXT NOT NULL UNIQUE,
  csrf_hash TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  revoked_at TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE email_outbox (
  id TEXT PRIMARY KEY,
  recipient TEXT NOT NULL,
  template TEXT NOT NULL,
  payload_ciphertext BLOB NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  next_attempt_at TEXT NOT NULL,
  sent_at TEXT,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE TABLE installation_identity (
  id TEXT PRIMARY KEY,
  public_key BLOB NOT NULL,
  private_key_ciphertext BLOB NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE workspace_replicas (
  workspace_id TEXT NOT NULL REFERENCES workspaces(id),
  replica_id TEXT NOT NULL,
  display_name TEXT NOT NULL,
  base_url TEXT NOT NULL DEFAULT '',
  public_key BLOB NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('authority','replica')),
  capabilities_json TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('active','revoked')),
  accepted_through_sequence INTEGER,
  approved_at TEXT NOT NULL,
  revoked_at TEXT,
  PRIMARY KEY(workspace_id, replica_id)
);

CREATE TABLE domain_events (
  event_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  project_id TEXT,
  origin_replica_id TEXT NOT NULL,
  origin_sequence INTEGER NOT NULL,
  schema_version INTEGER NOT NULL,
  type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  run_id TEXT,
  causal_event_ids_json TEXT NOT NULL DEFAULT '[]',
  payload_json TEXT NOT NULL,
  occurred_at TEXT NOT NULL,
  previous_hash TEXT NOT NULL,
  event_hash TEXT NOT NULL,
  signature BLOB NOT NULL,
  ingested_at TEXT NOT NULL,
  UNIQUE(workspace_id, origin_replica_id, origin_sequence)
);

CREATE TABLE sync_heads (
  workspace_id TEXT NOT NULL,
  origin_replica_id TEXT NOT NULL,
  contiguous_sequence INTEGER NOT NULL,
  event_hash TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, origin_replica_id)
);

CREATE TABLE replica_links (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  remote_replica_id TEXT NOT NULL,
  remote_url TEXT NOT NULL,
  direction TEXT NOT NULL CHECK(direction IN ('push','pull','push_pull')),
  credential_ciphertext BLOB NOT NULL,
  interval_seconds INTEGER NOT NULL,
  last_attempt_at TEXT,
  last_success_at TEXT,
  last_error TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 1,
  UNIQUE(workspace_id, remote_replica_id)
);

CREATE TABLE replica_pairings (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id),
  token_hash TEXT NOT NULL UNIQUE,
  purpose TEXT NOT NULL CHECK(purpose IN ('add_replica','cloud_mirror')),
  capabilities_json TEXT NOT NULL,
  bound_user_id TEXT,
  expires_at TEXT NOT NULL,
  claimed_at TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE sync_snapshots (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id),
  covers_vector_json TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  snapshot_hash TEXT NOT NULL,
  signature BLOB NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX idx_memberships_user ON workspace_memberships(user_id, status);
CREATE INDEX idx_invites_email ON workspace_invites(email_normalized, expires_at);
CREATE INDEX idx_sessions_user ON browser_sessions(user_id, expires_at);
CREATE INDEX idx_outbox_due ON email_outbox(sent_at, next_attempt_at);
CREATE INDEX idx_domain_events_workspace_origin ON domain_events(workspace_id, origin_replica_id, origin_sequence);
`

const syncTransportSchema = `
ALTER TABLE workspace_replicas ADD COLUMN last_success_at TEXT;
ALTER TABLE workspace_replicas ADD COLUMN last_error TEXT NOT NULL DEFAULT '';

CREATE TABLE replica_credentials (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id),
  replica_id TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  scopes_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  revoked_at TEXT,
  UNIQUE(workspace_id, replica_id, id)
);

CREATE TABLE mirror_pairings (
  id TEXT PRIMARY KEY,
  token_hash TEXT NOT NULL UNIQUE,
  bound_user_id TEXT NOT NULL REFERENCES users(id),
  expires_at TEXT NOT NULL,
  claimed_at TEXT,
  created_at TEXT NOT NULL
);

CREATE INDEX idx_replica_credentials_token ON replica_credentials(token_hash, revoked_at);
`

const authRateLimitSchema = `
CREATE TABLE auth_rate_limits (
  key_hash TEXT PRIMARY KEY,
  window_started_at TEXT NOT NULL,
  request_count INTEGER NOT NULL
);
`

const repairWorkspaceSlugs = `
UPDATE workspaces SET slug = lower(replace(name, ' ', '-')) || '-' || substr(id, -6) WHERE slug = '';
`

const operatorSuspensionSchema = `
ALTER TABLE workspaces ADD COLUMN status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','suspended'));
`

const lifecycleEdgesSchema = `
CREATE TABLE note_lifecycle_edges (
  event_id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id),
  target_note_id TEXT NOT NULL REFERENCES notes(id),
  replacement_note_id TEXT,
  reason TEXT NOT NULL DEFAULT '',
  origin_replica_id TEXT NOT NULL,
  occurred_at TEXT NOT NULL
);
CREATE INDEX idx_note_lifecycle_target ON note_lifecycle_edges(target_note_id, occurred_at);
`

const localUsersSchema = `
ALTER TABLE users ADD COLUMN login_kind TEXT NOT NULL DEFAULT 'email' CHECK(login_kind IN ('email','local'));
`

const actorOriginsSchema = `
ALTER TABLE principals ADD COLUMN origin_replica_id TEXT NOT NULL DEFAULT '';
`

const projectionBlockersSchema = `
CREATE TABLE projection_blockers (
  event_id TEXT PRIMARY KEY REFERENCES domain_events(event_id),
  workspace_id TEXT NOT NULL,
  schema_version INTEGER NOT NULL,
  reason TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX idx_projection_blockers_workspace ON projection_blockers(workspace_id, created_at);
`

const setupRequestsSchema = `
CREATE TABLE setup_requests (
  id TEXT PRIMARY KEY,
  device_code_hash TEXT NOT NULL UNIQUE,
  user_code_hash TEXT NOT NULL UNIQUE,
  code_challenge TEXT NOT NULL,
  project_slug TEXT NOT NULL,
  project_name TEXT NOT NULL,
  repository_url TEXT NOT NULL DEFAULT '',
  agent_name TEXT NOT NULL,
  membership_id TEXT REFERENCES workspace_memberships(id),
  project_id TEXT REFERENCES projects(id),
  credential_ciphertext BLOB,
  approved_at TEXT,
  consumed_at TEXT,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX idx_setup_requests_expires ON setup_requests(expires_at, consumed_at);
`

const localPasswordsSchema = `
ALTER TABLE users ADD COLUMN password_hash TEXT NOT NULL DEFAULT '';
`

func applyMigrations(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		return err
	}
	if version > latestSchemaVersion {
		return fmt.Errorf("database schema version %d is newer than this binary supports (%d); refusing to open", version, latestSchemaVersion)
	}
	if version == 0 {
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(1,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`); err != nil {
			return err
		}
		version = 1
	}
	if version < 2 {
		if _, err = tx.Exec(hostedReplicationSchema); err != nil {
			return fmt.Errorf("migration 2 hosted replication: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(2,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`); err != nil {
			return err
		}
	}
	if version < 3 {
		if _, err = tx.Exec(syncTransportSchema); err != nil {
			return fmt.Errorf("migration 3 sync transport: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(3,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`); err != nil {
			return err
		}
	}
	if version < 4 {
		if _, err = tx.Exec(authRateLimitSchema); err != nil {
			return fmt.Errorf("migration 4 auth rate limits: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(4,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`); err != nil {
			return err
		}
	}
	if version < 5 {
		if _, err = tx.Exec(repairWorkspaceSlugs); err != nil {
			return fmt.Errorf("migration 5 workspace slugs: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(5,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`); err != nil {
			return err
		}
	}
	if version < 6 {
		if _, err = tx.Exec(operatorSuspensionSchema); err != nil {
			return fmt.Errorf("migration 6 operator suspension: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(6,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`); err != nil {
			return err
		}
	}
	if version < 7 {
		if _, err = tx.Exec(lifecycleEdgesSchema); err != nil {
			return fmt.Errorf("migration 7 lifecycle edges: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(7,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`); err != nil {
			return err
		}
	}
	if version < 8 {
		if _, err = tx.Exec(localUsersSchema); err != nil {
			return fmt.Errorf("migration 8 local users: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(8,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`); err != nil {
			return err
		}
	}
	if version < 9 {
		if _, err = tx.Exec(actorOriginsSchema); err != nil {
			return fmt.Errorf("migration 9 actor origins: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(9,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`); err != nil {
			return err
		}
	}
	if version < 10 {
		if _, err = tx.Exec(projectionBlockersSchema); err != nil {
			return fmt.Errorf("migration 10 projection blockers: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(10,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`); err != nil {
			return err
		}
	}
	if version < 11 {
		if _, err = tx.Exec(setupRequestsSchema); err != nil {
			return fmt.Errorf("migration 11 setup requests: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(11,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`); err != nil {
			return err
		}
	}
	if version < 12 {
		if _, err = tx.Exec(localPasswordsSchema); err != nil {
			return fmt.Errorf("migration 12 local passwords: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(12,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`); err != nil {
			return err
		}
	}
	return tx.Commit()
}
