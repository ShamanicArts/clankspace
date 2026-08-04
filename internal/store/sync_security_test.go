package store

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ShamanicArts/clankspace/internal/domain"
)

func signedTestEvent(t *testing.T, privateKey ed25519.PrivateKey, event domain.DomainEvent) domain.DomainEvent {
	t.Helper()
	unsigned, err := unsignedEventBytes(event)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(unsigned)
	event.EventHash = "sha256:" + hex.EncodeToString(hash[:])
	event.Signature = "ed25519:" + base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(event.EventHash)))
	return event
}

func TestImportedEventsCannotEscapeReplicaAuthorityOrWorkspace(t *testing.T) {
	ctx := context.Background()
	db, err := OpenWithSecret(filepath.Join(t.TempDir(), "clankspace.db"), "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	principal, err := db.EnsureBootstrap(ctx, "bootstrap", "Target", "Owner")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.EnsureInstallationIdentity(ctx, "authority", "https://authority.test"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.EnsureWorkspaceAuthority(ctx, principal.WorkspaceID); err != nil {
		t.Fatal(err)
	}
	project, _, err := db.CreateProject(ctx, principal, "project", "target", "Target", "")
	if err != nil {
		t.Fatal(err)
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	replicaID := "replica_untrusted"
	if _, err = db.db.ExecContext(ctx, `INSERT INTO workspace_replicas(workspace_id,replica_id,display_name,base_url,public_key,role,capabilities_json,status,approved_at) VALUES(?,?,?,?,?,'replica','["pull","push"]','active',?)`, principal.WorkspaceID, replicaID, "Untrusted replica", "https://replica.test", publicKey, ts(now())); err != nil {
		t.Fatal(err)
	}
	actor := domain.PortableActor{PrincipalID: "principal_untrusted", PortableID: "actor_untrusted", DisplayName: "Untrusted", Kind: "human", OriginReplicaID: replicaID}
	forgedProject := domain.Project{ID: "project_forged", WorkspaceID: principal.WorkspaceID, Slug: "forged", Name: "Forged", CreatedAt: now()}
	payload, _ := json.Marshal(forgedProject)
	event := signedTestEvent(t, privateKey, domain.DomainEvent{SchemaVersion: 1, EventID: "event_forged_project", WorkspaceID: principal.WorkspaceID, ProjectID: forgedProject.ID, OriginReplicaID: replicaID, OriginSequence: 1, Type: "project.created", EntityID: forgedProject.ID, ActorID: actor.PortableID, Actor: actor, OccurredAt: now(), Payload: payload, PreviousHash: genesisHash})
	if _, err = db.ImportEvents(ctx, principal.WorkspaceID, []domain.DomainEvent{event}); err == nil || !strings.Contains(err.Error(), "authority") {
		t.Fatalf("replica-created project was accepted: %v", err)
	}

	otherUserID, err := db.EnsureLocalUserForPrincipal(ctx, principal)
	if err != nil {
		t.Fatal(err)
	}
	otherWorkspace, otherMembership, err := db.CreateWorkspaceForUser(ctx, otherUserID, "other", "Other")
	if err != nil {
		t.Fatal(err)
	}
	otherActor, err := db.PrincipalForMembership(ctx, otherMembership)
	if err != nil {
		t.Fatal(err)
	}
	otherProject, _, err := db.CreateProject(ctx, otherActor, "other-project", "other-project", "Other project", "")
	if err != nil {
		t.Fatal(err)
	}
	note := domain.Note{ID: "note_forged", ProjectID: otherProject.ID, PrincipalID: actor.PrincipalID, Kind: "decision", Title: "Cross workspace", Summary: "Must not land", LedBy: "agent", DirectionBasis: "autonomous_agent_judgment", Status: "current", Revision: 1, CreatedAt: now(), UpdatedAt: now()}
	payload, _ = json.Marshal(note)
	event = signedTestEvent(t, privateKey, domain.DomainEvent{SchemaVersion: 1, EventID: "event_cross_workspace", WorkspaceID: principal.WorkspaceID, ProjectID: otherProject.ID, OriginReplicaID: replicaID, OriginSequence: 1, Type: "note.recorded", EntityID: note.ID, ActorID: actor.PortableID, Actor: actor, OccurredAt: now(), Payload: payload, PreviousHash: genesisHash})
	if _, err = db.ImportEvents(ctx, principal.WorkspaceID, []domain.DomainEvent{event}); err == nil || !strings.Contains(err.Error(), "workspace") {
		t.Fatalf("cross-workspace event was accepted: %v", err)
	}
	if _, err = db.GetProject(ctx, principal.WorkspaceID, project.ID); err != nil {
		t.Fatalf("valid target project was damaged: %v", err)
	}
	_ = otherWorkspace
}

func TestUnknownSignedSchemaIsRetainedAndBlocksReadiness(t *testing.T) {
	ctx := context.Background()
	db, err := OpenWithSecret(filepath.Join(t.TempDir(), "clankspace.db"), "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	principal, err := db.EnsureBootstrap(ctx, "bootstrap", "Target", "Owner")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.EnsureInstallationIdentity(ctx, "authority", "https://authority.test"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.EnsureWorkspaceAuthority(ctx, principal.WorkspaceID); err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	replicaID := "replica_future"
	if _, err = db.db.ExecContext(ctx, `INSERT INTO workspace_replicas(workspace_id,replica_id,display_name,base_url,public_key,role,capabilities_json,status,approved_at) VALUES(?,?,?,?,?,'replica','["pull","push"]','active',?)`, principal.WorkspaceID, replicaID, "Future replica", "https://replica.test", publicKey, ts(now())); err != nil {
		t.Fatal(err)
	}
	actor := domain.PortableActor{PrincipalID: "principal_future", PortableID: "actor_future", DisplayName: "Future", Kind: "agent", OriginReplicaID: replicaID}
	event := signedTestEvent(t, privateKey, domain.DomainEvent{SchemaVersion: 2, EventID: "event_future", WorkspaceID: principal.WorkspaceID, ProjectID: "project_future", OriginReplicaID: replicaID, OriginSequence: 1, Type: "future.recorded", EntityID: "future_1", ActorID: actor.PortableID, Actor: actor, OccurredAt: now(), Payload: json.RawMessage(`{"future":true}`), PreviousHash: genesisHash})
	if imported, importErr := db.ImportEvents(ctx, principal.WorkspaceID, []domain.DomainEvent{event}); importErr != nil || imported != 1 {
		t.Fatalf("future event was not retained: imported=%d err=%v", imported, importErr)
	}
	if blockers, blockerErr := db.ProjectionBlockerCount(ctx); blockerErr != nil || blockers != 1 {
		t.Fatalf("projection blocker = %d, %v", blockers, blockerErr)
	}
}

func TestOpenRefusesDatabaseFromNewerBinary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clankspace.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.db.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(999,'future')`); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = Open(path); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("newer schema was opened: %v", err)
	}
}

func TestMigrationCreatesVerifiedPreMigrationBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.db")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = legacy.Exec(schema); err != nil {
		t.Fatal(err)
	}
	if _, err = legacy.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(1,'legacy')`); err != nil {
		t.Fatal(err)
	}
	if _, err = legacy.Exec(`INSERT INTO workspaces(id,name,created_at) VALUES('ws_legacy','Legacy','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err = legacy.Close(); err != nil {
		t.Fatal(err)
	}
	migrated, err := OpenWithSecret(path, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if err = migrated.Close(); err != nil {
		t.Fatal(err)
	}
	backups, err := filepath.Glob(path + ".pre-migration-v1-*")
	if err != nil || len(backups) != 1 {
		t.Fatalf("pre-migration backups = %#v, %v", backups, err)
	}
	backup, err := sql.Open("sqlite", backups[0])
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	var integrity string
	if err = backup.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("backup integrity = %q, %v", integrity, err)
	}
	var workspaceName string
	if err = backup.QueryRow(`SELECT name FROM workspaces WHERE id='ws_legacy'`).Scan(&workspaceName); err != nil || workspaceName != "Legacy" {
		t.Fatalf("backup data = %q, %v", workspaceName, err)
	}
}

func TestSchema12RunMigratesWithSafeDeliveryDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schema12.db")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = legacy.Exec(schema); err != nil {
		t.Fatal(err)
	}
	if _, err = legacy.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(1,'legacy')`); err != nil {
		t.Fatal(err)
	}
	steps := []string{hostedReplicationSchema, syncTransportSchema, authRateLimitSchema, repairWorkspaceSlugs, operatorSuspensionSchema, lifecycleEdgesSchema, localUsersSchema, actorOriginsSchema, projectionBlockersSchema, setupRequestsSchema, localPasswordsSchema}
	for index, step := range steps {
		version := index + 2
		if _, err = legacy.Exec(step); err != nil {
			t.Fatalf("apply legacy migration %d: %v", version, err)
		}
		if _, err = legacy.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(?,'legacy')`, version); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = legacy.Exec(`INSERT INTO workspaces(id,name,created_at,slug,status) VALUES('ws_old','Old','2026-08-01T00:00:00Z','old','active')`); err != nil {
		t.Fatal(err)
	}
	if _, err = legacy.Exec(`INSERT INTO principals(id,workspace_id,display_name,kind,created_at,portable_actor_id,origin_replica_id) VALUES('principal_old','ws_old','Old agent','project','2026-08-01T00:00:00Z','actor_old','')`); err != nil {
		t.Fatal(err)
	}
	if _, err = legacy.Exec(`INSERT INTO projects(id,workspace_id,slug,name,created_at) VALUES('project_old','ws_old','old-project','Old project','2026-08-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err = legacy.Exec(`INSERT INTO agents(id,principal_id,name,created_at) VALUES('agent_old','principal_old','Old agent','2026-08-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err = legacy.Exec(`INSERT INTO runs(id,project_id,agent_id,principal_id,role,run_type,branch,base_sha,head_sha,started_at,ended_at,outcome,verification) VALUES('run_old','project_old','agent_old','principal_old','primary','interactive','legacy-branch','1111111111111111111111111111111111111111','2222222222222222222222222222222222222222','2026-08-01T00:00:00Z','2026-08-01T01:00:00Z','completed','legacy checks passed')`); err != nil {
		t.Fatal(err)
	}
	if err = legacy.Close(); err != nil {
		t.Fatal(err)
	}
	migrated, err := OpenWithSecret(path, "schema12-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer migrated.Close()
	run, err := migrated.GetRun(t.Context(), "run_old")
	if err != nil || run.Branch != "legacy-branch" || run.PullRequestURL != "" || run.DeliveryBranch != "" {
		t.Fatalf("migrated run = %#v, %v", run, err)
	}
	principal := domain.Principal{ID: "principal_old", WorkspaceID: "ws_old", DisplayName: "Old agent", Kind: "project"}
	linked, _, err := migrated.LinkRunDelivery(t.Context(), principal, "legacy-link", run.ID, domain.LinkRunDeliveryInput{DeliveryBranch: "release", HeadSHA: "3333333333333333333333333333333333333333"})
	if err != nil || linked.Branch != "legacy-branch" || linked.DeliveryBranch != "release" {
		t.Fatalf("link migrated run = %#v, %v", linked, err)
	}
}

func TestEventsAfterPreservesOriginSequenceWhenTimestampsRegress(t *testing.T) {
	db, err := OpenWithSecret(filepath.Join(t.TempDir(), "ordering.db"), "ordering-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	principal, err := db.EnsureBootstrap(t.Context(), "ordering-token", "Ordering", "Agent")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.EnsureInstallationIdentity(t.Context(), "ordering", "https://ordering.test"); err != nil {
		t.Fatal(err)
	}
	if err = db.EnsureAllWorkspaceAuthorities(t.Context()); err != nil {
		t.Fatal(err)
	}
	project, _, err := db.CreateProject(t.Context(), principal, "ordering-project", "ordering", "Ordering", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = db.StartRun(t.Context(), principal, "ordering-run", domain.StartRunInput{ProjectID: project.ID, AgentName: "Agent"}); err != nil {
		t.Fatal(err)
	}
	origin := db.LocalReplicaID()
	if _, err = db.db.Exec(`UPDATE domain_events SET occurred_at=CASE origin_sequence WHEN 1 THEN '2026-08-04T02:00:00Z' WHEN 2 THEN '2026-08-04T01:00:00Z' ELSE occurred_at END WHERE origin_replica_id=?`, origin); err != nil {
		t.Fatal(err)
	}
	events, _, err := db.EventsAfter(t.Context(), principal.WorkspaceID, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	var sequences []int64
	for _, event := range events {
		if event.OriginReplicaID == origin {
			sequences = append(sequences, event.OriginSequence)
		}
	}
	if len(sequences) != 2 || sequences[0] != 1 || sequences[1] != 2 {
		t.Fatalf("origin sequence order = %#v", sequences)
	}
}
