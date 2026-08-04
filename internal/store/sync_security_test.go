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
