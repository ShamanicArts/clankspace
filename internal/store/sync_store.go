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
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/domain"
)

const genesisHash = "sha256:genesis"

type ReplicaClaim struct {
	ReplicaID    string   `json:"replicaId"`
	DisplayName  string   `json:"displayName"`
	BaseURL      string   `json:"baseUrl"`
	PublicKey    string   `json:"publicKey"`
	Capabilities []string `json:"capabilities"`
}

type ReplicaClaimResult struct {
	Workspace  domain.Workspace         `json:"workspace"`
	Authority  domain.Replica           `json:"authority"`
	Credential string                   `json:"credential"`
	Snapshot   domain.WorkspaceSnapshot `json:"snapshot"`
}

type MirrorClaimResult struct {
	Workspace  domain.Workspace `json:"workspace"`
	Replica    domain.Replica   `json:"replica"`
	Credential string           `json:"credential"`
}

func (s *Store) EnsureInstallationIdentity(ctx context.Context, displayName, baseURL string) (domain.Replica, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = "ClankSpace replica"
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	var id, created string
	var publicKey, encryptedPrivate []byte
	err := s.db.QueryRowContext(ctx, `SELECT id,public_key,private_key_ciphertext,created_at FROM installation_identity LIMIT 1`).Scan(&id, &publicKey, &encryptedPrivate, &created)
	if errors.Is(err, sql.ErrNoRows) {
		generatedPublic, privateKey, keyErr := ed25519.GenerateKey(rand.Reader)
		if keyErr != nil {
			return domain.Replica{}, keyErr
		}
		publicKey = generatedPublic
		encryptedPrivate, keyErr = s.seal(privateKey)
		if keyErr != nil {
			return domain.Replica{}, keyErr
		}
		id, created = newID("replica"), ts(now())
		if _, err = s.db.ExecContext(ctx, `INSERT INTO installation_identity(id,public_key,private_key_ciphertext,created_at) VALUES(?,?,?,?)`, id, publicKey, encryptedPrivate, created); err != nil {
			return domain.Replica{}, err
		}
	} else if err != nil {
		return domain.Replica{}, err
	}
	privateKey, err := s.openSealed(encryptedPrivate)
	if err != nil {
		return domain.Replica{}, err
	}
	if len(publicKey) != ed25519.PublicKeySize || len(privateKey) != ed25519.PrivateKeySize {
		return domain.Replica{}, errors.New("installation signing key is invalid")
	}
	s.replicaMu.Lock()
	s.replicaID, s.replicaName, s.replicaBaseURL = id, displayName, strings.TrimRight(baseURL, "/")
	s.replicaPublic, s.replicaPrivate = append(ed25519.PublicKey(nil), publicKey...), append(ed25519.PrivateKey(nil), privateKey...)
	s.replicaMu.Unlock()
	return domain.Replica{ID: id, DisplayName: displayName, BaseURL: s.replicaBaseURL, PublicKey: base64.RawURLEncoding.EncodeToString(publicKey), Status: "active", ApprovedAt: parseTime(created)}, nil
}

func (s *Store) localReplica() (string, string, string, ed25519.PublicKey, ed25519.PrivateKey, bool) {
	s.replicaMu.RLock()
	defer s.replicaMu.RUnlock()
	if s.replicaID == "" || len(s.replicaPrivate) != ed25519.PrivateKeySize {
		return "", "", "", nil, nil, false
	}
	return s.replicaID, s.replicaName, s.replicaBaseURL, append(ed25519.PublicKey(nil), s.replicaPublic...), append(ed25519.PrivateKey(nil), s.replicaPrivate...), true
}

func (s *Store) EnsureWorkspaceAuthority(ctx context.Context, workspaceID string) (domain.Replica, error) {
	id, name, baseURL, publicKey, _, ok := s.localReplica()
	if !ok {
		return domain.Replica{}, errors.New("installation identity is not initialized")
	}
	t := now()
	capabilities := `["manage","pull","push","share_humans"]`
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Replica{}, err
	}
	defer tx.Rollback()
	var authority string
	if err = tx.QueryRowContext(ctx, `SELECT authority_replica_id FROM workspaces WHERE id=?`, workspaceID).Scan(&authority); err != nil {
		return domain.Replica{}, err
	}
	if authority == "" {
		authority = id
		if _, err = tx.ExecContext(ctx, `UPDATE workspaces SET authority_replica_id=? WHERE id=?`, id, workspaceID); err != nil {
			return domain.Replica{}, err
		}
	}
	role := "replica"
	if authority == id {
		role = "authority"
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_replicas(workspace_id,replica_id,display_name,base_url,public_key,role,capabilities_json,status,approved_at) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(workspace_id,replica_id) DO UPDATE SET display_name=excluded.display_name,base_url=excluded.base_url,public_key=excluded.public_key,role=excluded.role,capabilities_json=excluded.capabilities_json,status='active'`, workspaceID, id, name, baseURL, publicKey, role, capabilities, "active", ts(t)); err != nil {
		return domain.Replica{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE principals SET origin_replica_id=? WHERE workspace_id=? AND origin_replica_id=''`, id, workspaceID); err != nil {
		return domain.Replica{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.Replica{}, err
	}
	return domain.Replica{ID: id, WorkspaceID: workspaceID, DisplayName: name, BaseURL: baseURL, PublicKey: base64.RawURLEncoding.EncodeToString(publicKey), Role: role, Capabilities: []string{"manage", "pull", "push", "share_humans"}, Status: "active", ApprovedAt: t}, nil
}

func (s *Store) EnsureAllWorkspaceAuthorities(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM workspaces`)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err = s.EnsureWorkspaceAuthority(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) IsWorkspaceAuthority(ctx context.Context, workspaceID string) (bool, error) {
	id, _, _, _, _, ok := s.localReplica()
	if !ok {
		return true, nil
	}
	var authority string
	err := s.db.QueryRowContext(ctx, `SELECT authority_replica_id FROM workspaces WHERE id=?`, workspaceID).Scan(&authority)
	return authority == "" || authority == id, err
}

func (s *Store) WorkspaceAuthorityID(ctx context.Context, workspaceID string) (string, error) {
	var authority string
	err := s.db.QueryRowContext(ctx, `SELECT authority_replica_id FROM workspaces WHERE id=?`, workspaceID).Scan(&authority)
	return authority, err
}

func (s *Store) CanShareHumans(ctx context.Context, workspaceID string) (bool, error) {
	id, _, _, _, _, ok := s.localReplica()
	if !ok {
		return true, nil
	}
	var role, capabilities, status string
	err := s.db.QueryRowContext(ctx, `SELECT role,capabilities_json,status FROM workspace_replicas WHERE workspace_id=? AND replica_id=?`, workspaceID, id).Scan(&role, &capabilities, &status)
	if err != nil {
		return false, err
	}
	return status == "active" && (role == "authority" || strings.Contains(capabilities, `"share_humans"`)), nil
}

func (s *Store) ListReplicas(ctx context.Context, workspaceID string) ([]domain.Replica, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT replica_id,display_name,base_url,public_key,role,capabilities_json,status,accepted_through_sequence,approved_at,revoked_at,last_success_at,last_error FROM workspace_replicas WHERE workspace_id=? ORDER BY role,display_name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Replica{}
	for rows.Next() {
		var item domain.Replica
		var publicKey []byte
		var capabilities, approved string
		var acceptedThrough sql.NullInt64
		var revoked, success sql.NullString
		if err = rows.Scan(&item.ID, &item.DisplayName, &item.BaseURL, &publicKey, &item.Role, &capabilities, &item.Status, &acceptedThrough, &approved, &revoked, &success, &item.LastError); err != nil {
			return nil, err
		}
		item.WorkspaceID, item.PublicKey, item.ApprovedAt = workspaceID, base64.RawURLEncoding.EncodeToString(publicKey), parseTime(approved)
		_ = json.Unmarshal([]byte(capabilities), &item.Capabilities)
		if revoked.Valid {
			value := parseTime(revoked.String)
			item.RevokedAt = &value
		}
		if acceptedThrough.Valid {
			value := acceptedThrough.Int64
			item.AcceptedThroughSequence = &value
		}
		if success.Valid {
			value := parseTime(success.String)
			item.LastSuccessAt = &value
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func isReplicableEvent(eventType string) bool {
	switch eventType {
	case "project.created", "run.started", "run.ended", "run.delivery_linked", "note.recorded", "note.superseded", "trajectory.started", "repository.attached":
		return true
	default:
		return false
	}
}

func portableEventPayload(eventType string, response []byte) ([]byte, string) {
	projectID := ""
	if eventType == "run.started" || eventType == "run.ended" || eventType == "run.delivery_linked" {
		var run domain.Run
		if json.Unmarshal(response, &run) == nil {
			run.Worktree = ""
			if eventType == "run.ended" {
				run.DeliveryBranch = ""
				run.DeliveryJJWorkspace = ""
				run.DeliveryJJChangeID = ""
				run.DeliveryJJCommitID = ""
				run.DeliveryJJBookmarks = nil
				run.PullRequestURL = ""
				run.PullRequestNumber = 0
				run.PullRequestState = ""
				run.MergeCommitSHA = ""
				run.MergedAt = nil
			}
			projectID = run.ProjectID
			if body, err := json.Marshal(run); err == nil {
				return body, projectID
			}
		}
	}
	return response, projectID
}

func appendUnique(items []string, value string) []string {
	if value == "" {
		return items
	}
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func eventCauseTx(ctx context.Context, tx *sql.Tx, workspaceID, eventType, entityID string) string {
	var eventID string
	_ = tx.QueryRowContext(ctx, `SELECT event_id FROM domain_events WHERE workspace_id=? AND type=? AND entity_id=? ORDER BY origin_sequence LIMIT 1`, workspaceID, eventType, entityID).Scan(&eventID)
	return eventID
}

func unsignedEventBytes(event domain.DomainEvent) ([]byte, error) {
	event.EventHash = ""
	event.Signature = ""
	return json.Marshal(event)
}

func (s *Store) appendDomainEventTx(ctx context.Context, tx *sql.Tx, eventID, workspaceID, projectID, principalID, runID, eventType, entityID string, response []byte, occurredAt time.Time) (string, int64, error) {
	if !isReplicableEvent(eventType) {
		return "", 0, nil
	}
	replicaID, _, _, _, privateKey, ok := s.localReplica()
	if !ok {
		return "", 0, nil
	}
	var capabilities, status string
	if err := tx.QueryRowContext(ctx, `SELECT capabilities_json,status FROM workspace_replicas WHERE workspace_id=? AND replica_id=?`, workspaceID, replicaID).Scan(&capabilities, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, nil
		}
		return "", 0, err
	}
	if status != "active" || !strings.Contains(capabilities, `"push"`) {
		return "", 0, errors.New("local replica may not publish workspace events")
	}
	var actor domain.PortableActor
	if _, err := tx.ExecContext(ctx, `UPDATE principals SET origin_replica_id=? WHERE id=? AND workspace_id=? AND origin_replica_id=''`, replicaID, principalID, workspaceID); err != nil {
		return "", 0, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT id,portable_actor_id,display_name,kind,origin_replica_id FROM principals WHERE id=? AND workspace_id=?`, principalID, workspaceID).Scan(&actor.PrincipalID, &actor.PortableID, &actor.DisplayName, &actor.Kind, &actor.OriginReplicaID); err != nil {
		return "", 0, err
	}
	if actor.OriginReplicaID != replicaID {
		return "", 0, errors.New("local replica cannot publish for an actor owned by another replica")
	}
	var sequence int64
	previousHash := genesisHash
	err := tx.QueryRowContext(ctx, `SELECT contiguous_sequence,event_hash FROM sync_heads WHERE workspace_id=? AND origin_replica_id=?`, workspaceID, replicaID).Scan(&sequence, &previousHash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", 0, err
	}
	sequence++
	payload, payloadProject := portableEventPayload(eventType, response)
	if projectID == "" {
		projectID = payloadProject
	}
	if eventType == "project.created" && projectID == "" {
		projectID = entityID
	}
	causalEventIDs := []string{}
	if projectID != "" && eventType != "project.created" {
		causalEventIDs = appendUnique(causalEventIDs, eventCauseTx(ctx, tx, workspaceID, "project.created", projectID))
	}
	if eventType == "run.ended" || eventType == "run.delivery_linked" {
		causalEventIDs = appendUnique(causalEventIDs, eventCauseTx(ctx, tx, workspaceID, "run.started", entityID))
	} else if runID != "" {
		causalEventIDs = appendUnique(causalEventIDs, eventCauseTx(ctx, tx, workspaceID, "run.started", runID))
	}
	if eventType == "note.superseded" {
		causalEventIDs = appendUnique(causalEventIDs, eventCauseTx(ctx, tx, workspaceID, "note.recorded", entityID))
	}
	event := domain.DomainEvent{SchemaVersion: 1, EventID: eventID, WorkspaceID: workspaceID, ProjectID: projectID, OriginReplicaID: replicaID, OriginSequence: sequence, Type: eventType, EntityID: entityID, ActorID: actor.PortableID, Actor: actor, RunID: runID, CausalEventIDs: causalEventIDs, OccurredAt: occurredAt, Payload: payload, PreviousHash: previousHash}
	unsigned, err := unsignedEventBytes(event)
	if err != nil {
		return "", 0, err
	}
	hash := sha256.Sum256(unsigned)
	event.EventHash = "sha256:" + hex.EncodeToString(hash[:])
	event.Signature = "ed25519:" + base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(event.EventHash)))
	causes, _ := json.Marshal(event.CausalEventIDs)
	if _, err = tx.ExecContext(ctx, `INSERT INTO domain_events(event_id,workspace_id,project_id,origin_replica_id,origin_sequence,schema_version,type,entity_id,actor_id,run_id,causal_event_ids_json,payload_json,occurred_at,previous_hash,event_hash,signature,ingested_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, event.EventID, event.WorkspaceID, nullable(event.ProjectID), event.OriginReplicaID, event.OriginSequence, event.SchemaVersion, event.Type, event.EntityID, event.ActorID, nullable(event.RunID), string(causes), string(event.Payload), ts(event.OccurredAt), event.PreviousHash, event.EventHash, []byte(event.Signature), ts(now())); err != nil {
		return "", 0, err
	}
	if event.Type == "note.superseded" {
		if err = recordLifecycleEdgeTx(ctx, tx, event); err != nil {
			return "", 0, err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO sync_heads(workspace_id,origin_replica_id,contiguous_sequence,event_hash,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(workspace_id,origin_replica_id) DO UPDATE SET contiguous_sequence=excluded.contiguous_sequence,event_hash=excluded.event_hash,updated_at=excluded.updated_at`, workspaceID, replicaID, sequence, event.EventHash, ts(now())); err != nil {
		return "", 0, err
	}
	return replicaID, sequence, nil
}

func (s *Store) domainEventInfoTx(ctx context.Context, tx *sql.Tx, eventID string) (string, int64) {
	var replicaID string
	var sequence int64
	_ = tx.QueryRowContext(ctx, `SELECT origin_replica_id,origin_sequence FROM domain_events WHERE event_id=?`, eventID).Scan(&replicaID, &sequence)
	return replicaID, sequence
}

func (s *Store) CreateReplicaOffer(ctx context.Context, membership domain.Membership, capabilities []string) (string, time.Time, error) {
	if membership.Role != "owner" {
		return "", time.Time{}, errors.New("workspace owner required")
	}
	authority, err := s.IsWorkspaceAuthority(ctx, membership.WorkspaceID)
	if err != nil || !authority {
		if err == nil {
			err = errors.New("replica offers must be created on the workspace authority")
		}
		return "", time.Time{}, err
	}
	if len(capabilities) == 0 {
		capabilities = []string{"pull", "push"}
	}
	token, err := randomToken("pair_")
	if err != nil {
		return "", time.Time{}, err
	}
	body, _ := json.Marshal(capabilities)
	expires := now().Add(15 * time.Minute)
	_, err = s.db.ExecContext(ctx, `INSERT INTO replica_pairings(id,workspace_id,token_hash,purpose,capabilities_json,expires_at,created_at) VALUES(?,?,?,?,?,?,?)`, newID("pairing"), membership.WorkspaceID, hashToken(token), "add_replica", string(body), ts(expires), ts(now()))
	return token, expires, err
}

func (s *Store) ClaimReplicaOffer(ctx context.Context, code string, claim ReplicaClaim) (ReplicaClaimResult, error) {
	publicKey, err := base64.RawURLEncoding.DecodeString(claim.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return ReplicaClaimResult{}, errors.New("replica public key is invalid")
	}
	t := now()
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReplicaClaimResult{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	var pairingID, workspaceID, capabilitiesJSON string
	if err = tx.QueryRowContext(ctx, `SELECT id,workspace_id,capabilities_json FROM replica_pairings WHERE token_hash=? AND claimed_at IS NULL AND expires_at>?`, hashToken(code), ts(t)).Scan(&pairingID, &workspaceID, &capabilitiesJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReplicaClaimResult{}, ErrNotFound
		}
		return ReplicaClaimResult{}, err
	}
	if claim.ReplicaID == "" {
		return ReplicaClaimResult{}, errors.New("replica ID is required")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_replicas(workspace_id,replica_id,display_name,base_url,public_key,role,capabilities_json,status,approved_at) VALUES(?,?,?,?,?,'replica',?,'active',?)`, workspaceID, claim.ReplicaID, claim.DisplayName, strings.TrimRight(claim.BaseURL, "/"), publicKey, capabilitiesJSON, ts(t)); err != nil {
		return ReplicaClaimResult{}, err
	}
	credential, err := randomToken("replica_token_")
	if err != nil {
		return ReplicaClaimResult{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO replica_credentials(id,workspace_id,replica_id,token_hash,scopes_json,created_at) VALUES(?,?,?,?,?,?)`, newID("replica_credential"), workspaceID, claim.ReplicaID, hashToken(credential), capabilitiesJSON, ts(t)); err != nil {
		return ReplicaClaimResult{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE replica_pairings SET claimed_at=? WHERE id=?`, ts(t), pairingID); err != nil {
		return ReplicaClaimResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return ReplicaClaimResult{}, err
	}
	snapshot, err := s.buildWorkspaceSnapshot(ctx, workspaceID)
	if err != nil {
		return ReplicaClaimResult{}, err
	}
	workspace, err := s.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return ReplicaClaimResult{}, err
	}
	replicas, err := s.ListReplicas(ctx, workspaceID)
	if err != nil {
		return ReplicaClaimResult{}, err
	}
	var authority domain.Replica
	for _, replica := range replicas {
		if replica.Role == "authority" {
			authority = replica
			break
		}
	}
	return ReplicaClaimResult{Workspace: workspace, Authority: authority, Credential: credential, Snapshot: snapshot}, nil
}

func (s *Store) CreateMirrorOffer(ctx context.Context, userID string) (string, time.Time, error) {
	var status string
	if err := s.db.QueryRowContext(ctx, `SELECT status FROM users WHERE id=?`, userID).Scan(&status); err != nil {
		return "", time.Time{}, err
	}
	if status != "active" {
		return "", time.Time{}, errors.New("active user required")
	}
	token, err := randomToken("mirror_")
	if err != nil {
		return "", time.Time{}, err
	}
	expires := now().Add(15 * time.Minute)
	_, err = s.db.ExecContext(ctx, `INSERT INTO mirror_pairings(id,token_hash,bound_user_id,expires_at,created_at) VALUES(?,?,?,?,?)`, newID("mirror_pairing"), hashToken(token), userID, ts(expires), ts(now()))
	return token, expires, err
}

func (s *Store) ClaimMirrorOffer(ctx context.Context, code string, snapshot domain.WorkspaceSnapshot, authority domain.Replica) (MirrorClaimResult, error) {
	var pairingID, userID string
	t := now()
	s.writeMu.Lock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.writeMu.Unlock()
		return MirrorClaimResult{}, err
	}
	err = tx.QueryRowContext(ctx, `SELECT id,bound_user_id FROM mirror_pairings WHERE token_hash=? AND claimed_at IS NULL AND expires_at>?`, hashToken(code), ts(t)).Scan(&pairingID, &userID)
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE mirror_pairings SET claimed_at=? WHERE id=? AND claimed_at IS NULL`, ts(t), pairingID)
	}
	if err == nil {
		err = tx.Commit()
	} else {
		_ = tx.Rollback()
	}
	s.writeMu.Unlock()
	if errors.Is(err, sql.ErrNoRows) {
		return MirrorClaimResult{}, ErrNotFound
	}
	if err != nil {
		return MirrorClaimResult{}, err
	}
	if err = s.ImportWorkspaceSnapshot(ctx, snapshot, authority, userID); err != nil {
		return MirrorClaimResult{}, err
	}
	credential, err := randomToken("replica_token_")
	if err != nil {
		return MirrorClaimResult{}, err
	}
	capabilities := `["pull","push"]`
	if _, err = s.db.ExecContext(ctx, `INSERT INTO replica_credentials(id,workspace_id,replica_id,token_hash,scopes_json,created_at) VALUES(?,?,?,?,?,?)`, newID("replica_credential"), snapshot.Workspace.ID, authority.ID, hashToken(credential), capabilities, ts(now())); err != nil {
		return MirrorClaimResult{}, err
	}
	replicas, err := s.ListReplicas(ctx, snapshot.Workspace.ID)
	if err != nil {
		return MirrorClaimResult{}, err
	}
	localID := s.LocalReplicaID()
	var local domain.Replica
	for _, replica := range replicas {
		if replica.ID == localID {
			local = replica
			break
		}
	}
	workspace, err := s.GetWorkspace(ctx, snapshot.Workspace.ID)
	if err != nil {
		return MirrorClaimResult{}, err
	}
	return MirrorClaimResult{Workspace: workspace, Replica: local, Credential: credential}, nil
}

func (s *Store) GetWorkspace(ctx context.Context, id string) (domain.Workspace, error) {
	var workspace domain.Workspace
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT id,slug,name,authority_replica_id,created_at,status FROM workspaces WHERE id=?`, id).Scan(&workspace.ID, &workspace.Slug, &workspace.Name, &workspace.AuthorityReplicaID, &created, &workspace.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return workspace, ErrNotFound
	}
	workspace.CreatedAt = parseTime(created)
	return workspace, err
}

func (s *Store) syncHeads(ctx context.Context, workspaceID string) ([]domain.SyncHead, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT origin_replica_id,contiguous_sequence,event_hash FROM sync_heads WHERE workspace_id=? ORDER BY origin_replica_id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	heads := []domain.SyncHead{}
	for rows.Next() {
		var head domain.SyncHead
		if err = rows.Scan(&head.OriginReplicaID, &head.Sequence, &head.EventHash); err != nil {
			return nil, err
		}
		heads = append(heads, head)
	}
	return heads, rows.Err()
}

func (s *Store) SyncHeads(ctx context.Context, workspaceID string) ([]domain.SyncHead, error) {
	return s.syncHeads(ctx, workspaceID)
}

func (s *Store) BuildWorkspaceSnapshot(ctx context.Context, workspaceID string) (domain.WorkspaceSnapshot, error) {
	// Every content mutation and imported event uses writeMu. Holding it across
	// the projection reads and head read gives the snapshot one coherent cut of
	// history without sharing SQLite pages between processes.
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.buildWorkspaceSnapshot(ctx, workspaceID)
}

func (s *Store) buildWorkspaceSnapshot(ctx context.Context, workspaceID string) (domain.WorkspaceSnapshot, error) {
	workspace, err := s.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return domain.WorkspaceSnapshot{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,portable_actor_id,display_name,kind,origin_replica_id FROM principals WHERE workspace_id=? ORDER BY id`, workspaceID)
	if err != nil {
		return domain.WorkspaceSnapshot{}, err
	}
	actors := []domain.PortableActor{}
	for rows.Next() {
		var actor domain.PortableActor
		if err = rows.Scan(&actor.PrincipalID, &actor.PortableID, &actor.DisplayName, &actor.Kind, &actor.OriginReplicaID); err != nil {
			rows.Close()
			return domain.WorkspaceSnapshot{}, err
		}
		actors = append(actors, actor)
	}
	rows.Close()
	agentRows, err := s.db.QueryContext(ctx, `SELECT a.id,a.principal_id,a.name,a.created_at FROM agents a JOIN principals p ON p.id=a.principal_id WHERE p.workspace_id=? ORDER BY a.id`, workspaceID)
	if err != nil {
		return domain.WorkspaceSnapshot{}, err
	}
	agents := []domain.Agent{}
	for agentRows.Next() {
		var agent domain.Agent
		var created string
		if err = agentRows.Scan(&agent.ID, &agent.PrincipalID, &agent.Name, &created); err != nil {
			agentRows.Close()
			return domain.WorkspaceSnapshot{}, err
		}
		agent.CreatedAt = parseTime(created)
		agents = append(agents, agent)
	}
	agentRows.Close()
	projects, err := s.ListProjects(ctx, workspaceID)
	if err != nil {
		return domain.WorkspaceSnapshot{}, err
	}
	projectSnapshots := make([]domain.ProjectSnapshot, 0, len(projects))
	for _, project := range projects {
		runs, runErr := s.ListAllRuns(ctx, project.ID)
		if runErr != nil {
			return domain.WorkspaceSnapshot{}, runErr
		}
		for index := range runs {
			runs[index].Worktree = ""
		}
		notes, noteErr := s.ListNotes(ctx, project.ID, -1)
		if noteErr != nil {
			return domain.WorkspaceSnapshot{}, noteErr
		}
		for index := range notes {
			notes[index].Run = nil
		}
		trajectories, trajectoryErr := s.ListTrajectories(ctx, project.ID, false)
		if trajectoryErr != nil {
			return domain.WorkspaceSnapshot{}, trajectoryErr
		}
		for index := range trajectories {
			trajectories[index].Run = nil
		}
		repositories, repositoryErr := s.ListRepositories(ctx, project.ID)
		if repositoryErr != nil {
			return domain.WorkspaceSnapshot{}, repositoryErr
		}
		for index := range repositories {
			repositories[index].Pulls = nil
		}
		edges, edgeErr := s.ListLifecycleEdges(ctx, project.ID)
		if edgeErr != nil {
			return domain.WorkspaceSnapshot{}, edgeErr
		}
		projectSnapshots = append(projectSnapshots, domain.ProjectSnapshot{Project: project, Runs: runs, Notes: notes, Trajectories: trajectories, Repositories: repositories, LifecycleEdges: edges})
	}
	heads, err := s.syncHeads(ctx, workspaceID)
	if err != nil {
		return domain.WorkspaceSnapshot{}, err
	}
	replicas, err := s.ListReplicas(ctx, workspaceID)
	if err != nil {
		return domain.WorkspaceSnapshot{}, err
	}
	snapshot := domain.WorkspaceSnapshot{SchemaVersion: 1, Workspace: workspace, Actors: actors, Agents: agents, Replicas: replicas, Projects: projectSnapshots, Heads: heads, CreatedAt: now()}
	unsigned := snapshot
	unsigned.SnapshotHash, unsigned.Signature = "", ""
	body, err := json.Marshal(unsigned)
	if err != nil {
		return domain.WorkspaceSnapshot{}, err
	}
	hash := sha256.Sum256(body)
	snapshot.SnapshotHash = "sha256:" + hex.EncodeToString(hash[:])
	_, _, _, _, privateKey, ok := s.localReplica()
	if !ok {
		return domain.WorkspaceSnapshot{}, errors.New("installation identity is not initialized")
	}
	snapshot.Signature = "ed25519:" + base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(snapshot.SnapshotHash)))
	return snapshot, nil
}

func normalizeCapabilities(items []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if (item == "pull" || item == "push" || item == "share_humans") && !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Store) AuthenticateReplica(ctx context.Context, token, workspaceID string) (domain.Replica, []string, error) {
	var replica domain.Replica
	var publicKey []byte
	var capabilities, approved string
	err := s.db.QueryRowContext(ctx, `SELECT r.replica_id,r.display_name,r.base_url,r.public_key,r.role,r.capabilities_json,r.status,r.approved_at FROM replica_credentials c JOIN workspace_replicas r ON r.workspace_id=c.workspace_id AND r.replica_id=c.replica_id WHERE c.token_hash=? AND c.workspace_id=? AND c.revoked_at IS NULL AND r.status='active'`, hashToken(token), workspaceID).Scan(&replica.ID, &replica.DisplayName, &replica.BaseURL, &publicKey, &replica.Role, &capabilities, &replica.Status, &approved)
	if errors.Is(err, sql.ErrNoRows) {
		return replica, nil, ErrNotFound
	}
	if err != nil {
		return replica, nil, err
	}
	replica.WorkspaceID, replica.PublicKey, replica.ApprovedAt = workspaceID, base64.RawURLEncoding.EncodeToString(publicKey), parseTime(approved)
	var scopes []string
	_ = json.Unmarshal([]byte(capabilities), &scopes)
	return replica, scopes, nil
}

func hasCapability(items []string, desired string) bool {
	for _, item := range items {
		if item == desired || item == "manage" {
			return true
		}
	}
	return false
}

func (s *Store) EventsAfter(ctx context.Context, workspaceID string, have []domain.SyncHead, limit int) ([]domain.DomainEvent, bool, error) {
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	known := map[string]int64{}
	for _, head := range have {
		known[head.OriginReplicaID] = head.Sequence
	}
	var authorityID string
	if err := s.db.QueryRowContext(ctx, `SELECT authority_replica_id FROM workspaces WHERE id=?`, workspaceID).Scan(&authorityID); err != nil {
		return nil, false, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT event_id,workspace_id,COALESCE(project_id,''),origin_replica_id,origin_sequence,schema_version,type,entity_id,actor_id,COALESCE(run_id,''),causal_event_ids_json,payload_json,occurred_at,previous_hash,event_hash,signature FROM domain_events WHERE workspace_id=? ORDER BY CASE WHEN origin_replica_id=? THEN 0 ELSE 1 END,occurred_at,origin_replica_id,origin_sequence`, workspaceID, authorityID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	pending := []domain.DomainEvent{}
	for rows.Next() {
		var event domain.DomainEvent
		var causes, payload, occurred, signature string
		if err = rows.Scan(&event.EventID, &event.WorkspaceID, &event.ProjectID, &event.OriginReplicaID, &event.OriginSequence, &event.SchemaVersion, &event.Type, &event.EntityID, &event.ActorID, &event.RunID, &causes, &payload, &occurred, &event.PreviousHash, &event.EventHash, &signature); err != nil {
			return nil, false, err
		}
		if event.OriginSequence <= known[event.OriginReplicaID] {
			continue
		}
		_ = json.Unmarshal([]byte(causes), &event.CausalEventIDs)
		event.Payload, event.OccurredAt, event.Signature = json.RawMessage(payload), parseTime(occurred), signature
		if actorErr := s.db.QueryRowContext(ctx, `SELECT id,portable_actor_id,display_name,kind,origin_replica_id FROM principals WHERE portable_actor_id=? AND workspace_id=?`, event.ActorID, workspaceID).Scan(&event.Actor.PrincipalID, &event.Actor.PortableID, &event.Actor.DisplayName, &event.Actor.Kind, &event.Actor.OriginReplicaID); actorErr != nil {
			return nil, false, actorErr
		}
		pending = append(pending, event)
	}
	if err = rows.Err(); err != nil {
		return nil, false, err
	}

	// Merge origin streams without changing the order of any one hash chain.
	// Causal edges make authority-created projects and cross-origin targets arrive
	// before events that reference them.
	remaining := make(map[string]domain.DomainEvent, len(pending))
	chainDependency := map[string]string{}
	for _, event := range pending {
		remaining[event.EventID] = event
	}
	chainOrder := append([]domain.DomainEvent(nil), pending...)
	sort.SliceStable(chainOrder, func(i, j int) bool {
		if chainOrder[i].OriginReplicaID == chainOrder[j].OriginReplicaID {
			return chainOrder[i].OriginSequence < chainOrder[j].OriginSequence
		}
		return chainOrder[i].OriginReplicaID < chainOrder[j].OriginReplicaID
	})
	previousUnseen := map[string]string{}
	for _, event := range chainOrder {
		if previous := previousUnseen[event.OriginReplicaID]; previous != "" {
			chainDependency[event.EventID] = previous
		}
		previousUnseen[event.OriginReplicaID] = event.EventID
	}
	ordered := make([]domain.DomainEvent, 0, len(pending))
	for len(remaining) > 0 {
		progress := false
		for _, event := range pending {
			if _, exists := remaining[event.EventID]; !exists {
				continue
			}
			blocked := false
			if previous := chainDependency[event.EventID]; previous != "" {
				_, blocked = remaining[previous]
			}
			if !blocked {
				for _, cause := range event.CausalEventIDs {
					if _, exists := remaining[cause]; exists {
						blocked = true
						break
					}
				}
			}
			if blocked {
				continue
			}
			ordered = append(ordered, event)
			delete(remaining, event.EventID)
			progress = true
		}
		if !progress {
			return nil, false, errors.New("event dependency graph contains a cycle")
		}
	}
	more := len(ordered) > limit
	if more {
		ordered = ordered[:limit]
	}
	return ordered, more, nil
}

func (s *Store) RecordReplicaSuccess(ctx context.Context, workspaceID, replicaID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE workspace_replicas SET last_success_at=?,last_error='' WHERE workspace_id=? AND replica_id=?`, ts(now()), workspaceID, replicaID)
	return err
}

func (s *Store) RevokeReplica(ctx context.Context, membership domain.Membership, replicaID string) error {
	if membership.Role != "owner" {
		return errors.New("workspace owner required")
	}
	authority, err := s.IsWorkspaceAuthority(ctx, membership.WorkspaceID)
	if err != nil {
		return err
	}
	if !authority {
		return errors.New("replicas must be revoked on the workspace authority")
	}
	var lastSequence sql.NullInt64
	_ = s.db.QueryRowContext(ctx, `SELECT contiguous_sequence FROM sync_heads WHERE workspace_id=? AND origin_replica_id=?`, membership.WorkspaceID, replicaID).Scan(&lastSequence)
	t := now()
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE workspace_replicas SET status='revoked',revoked_at=?,accepted_through_sequence=? WHERE workspace_id=? AND replica_id=? AND role!='authority' AND status='active'`, ts(t), lastSequence, membership.WorkspaceID, replicaID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrNotFound
	}
	if _, err = tx.ExecContext(ctx, `UPDATE replica_credentials SET revoked_at=? WHERE workspace_id=? AND replica_id=? AND revoked_at IS NULL`, ts(t), membership.WorkspaceID, replicaID); err != nil {
		return err
	}
	return tx.Commit()
}

func verifyEvent(event domain.DomainEvent, publicKey ed25519.PublicKey) error {
	unsigned, err := unsignedEventBytes(event)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(unsigned)
	expectedHash := "sha256:" + hex.EncodeToString(hash[:])
	if event.EventHash != expectedHash {
		return errors.New("event hash mismatch")
	}
	signatureText := strings.TrimPrefix(event.Signature, "ed25519:")
	signature, err := base64.RawURLEncoding.DecodeString(signatureText)
	if err != nil || !ed25519.Verify(publicKey, []byte(event.EventHash), signature) {
		return errors.New("event signature is invalid")
	}
	return nil
}

func projectInWorkspaceTx(ctx context.Context, tx *sql.Tx, projectID, workspaceID string) error {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE id=? AND workspace_id=?`, projectID, workspaceID).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return errors.New("event project does not belong to its workspace")
	}
	return nil
}

func runInProjectTx(ctx context.Context, tx *sql.Tx, runID, projectID string) error {
	if runID == "" {
		return nil
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM runs WHERE id=? AND project_id=?`, runID, projectID).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return errors.New("event run does not belong to its project")
	}
	return nil
}

func validateEventEnvelopeTx(ctx context.Context, tx *sql.Tx, event domain.DomainEvent, originRole string) error {
	if event.EventID == "" || event.EntityID == "" || event.ActorID == "" || event.Actor.PortableID != event.ActorID || event.Actor.PrincipalID == "" {
		return errors.New("event envelope identity is incomplete")
	}
	if event.Actor.OriginReplicaID != "" && event.Actor.OriginReplicaID != event.OriginReplicaID {
		return errors.New("event actor is not owned by its origin replica")
	}
	if event.Type == "project.created" || event.Type == "repository.attached" {
		if originRole != "authority" {
			return errors.New("event type is reserved for the workspace authority")
		}
	}
	if event.Type != "project.created" {
		if event.ProjectID == "" {
			return errors.New("event project is required")
		}
		if err := projectInWorkspaceTx(ctx, tx, event.ProjectID, event.WorkspaceID); err != nil {
			return err
		}
	}

	switch event.Type {
	case "project.created":
		var item domain.Project
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			return err
		}
		if item.ID != event.EntityID || item.ID != event.ProjectID || item.WorkspaceID != event.WorkspaceID {
			return errors.New("project event payload does not match its envelope")
		}
		var existingWorkspace string
		err := tx.QueryRowContext(ctx, `SELECT workspace_id FROM projects WHERE id=?`, item.ID).Scan(&existingWorkspace)
		if err == nil && existingWorkspace != event.WorkspaceID {
			return errors.New("project ID belongs to another workspace")
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	case "run.started", "run.ended", "run.delivery_linked":
		var item domain.Run
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			return err
		}
		if item.ID != event.EntityID || item.ProjectID != event.ProjectID || item.PrincipalID != event.Actor.PrincipalID {
			return errors.New("run event payload does not match its envelope")
		}
		if err := validateJujutsuProvenance(item.VCS, item.JJWorkspace, item.JJChangeID, item.JJCommitID, item.JJBookmarks); err != nil {
			return err
		}
		if item.RepositoryID != "" {
			var repositoryCount int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM project_repositories WHERE project_id=? AND repository_id=?`, item.ProjectID, item.RepositoryID).Scan(&repositoryCount); err != nil || repositoryCount != 1 {
				return errors.New("run event repository is not attached to its project")
			}
		}
		if event.Type == "run.ended" || event.Type == "run.delivery_linked" {
			if err := runInProjectTx(ctx, tx, item.ID, event.ProjectID); err != nil {
				return err
			}
			if err := validateRunDeliveryTx(ctx, tx, item.ID, item.PrincipalID, domain.LinkRunDeliveryInput{RepositoryID: item.RepositoryID, VCS: item.VCS, DeliveryBranch: item.DeliveryBranch, HeadSHA: item.HeadSHA, DeliveryJJWorkspace: item.DeliveryJJWorkspace, DeliveryJJChangeID: item.DeliveryJJChangeID, DeliveryJJCommitID: item.DeliveryJJCommitID, DeliveryJJBookmarks: item.DeliveryJJBookmarks, PullRequestURL: item.PullRequestURL, PullRequestNumber: item.PullRequestNumber, PullRequestState: item.PullRequestState, MergeCommitSHA: item.MergeCommitSHA, MergedAt: item.MergedAt}); err != nil {
				return err
			}
		}
		var existingProject string
		err := tx.QueryRowContext(ctx, `SELECT project_id FROM runs WHERE id=?`, item.ID).Scan(&existingProject)
		if err == nil && existingProject != event.ProjectID {
			return errors.New("run ID belongs to another project")
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var existingPrincipal string
		err = tx.QueryRowContext(ctx, `SELECT principal_id FROM agents WHERE id=?`, item.AgentID).Scan(&existingPrincipal)
		if err == nil && existingPrincipal != event.Actor.PrincipalID {
			return errors.New("agent ID belongs to another actor")
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	case "note.recorded":
		var item domain.Note
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			return err
		}
		if item.ID != event.EntityID || item.ProjectID != event.ProjectID || item.PrincipalID != event.Actor.PrincipalID || item.RunID != event.RunID {
			return errors.New("note event payload does not match its envelope")
		}
		if err := runInProjectTx(ctx, tx, item.RunID, event.ProjectID); err != nil {
			return err
		}
		var existingProject string
		err := tx.QueryRowContext(ctx, `SELECT project_id FROM notes WHERE id=?`, item.ID).Scan(&existingProject)
		if err == nil && existingProject != event.ProjectID {
			return errors.New("note ID belongs to another project")
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	case "note.superseded":
		var item struct {
			Superseded  domain.Note `json:"superseded"`
			Replacement domain.Note `json:"replacement"`
		}
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			return err
		}
		if item.Superseded.ID != event.EntityID || item.Superseded.ProjectID != event.ProjectID {
			return errors.New("supersession target does not match its envelope")
		}
		var targetProject string
		if err := tx.QueryRowContext(ctx, `SELECT project_id FROM notes WHERE id=?`, item.Superseded.ID).Scan(&targetProject); err != nil || targetProject != event.ProjectID {
			return errors.New("supersession target does not belong to its project")
		}
		if item.Replacement.ID != "" {
			if item.Replacement.ProjectID != event.ProjectID || item.Replacement.PrincipalID != event.Actor.PrincipalID || item.Replacement.RunID != event.RunID {
				return errors.New("supersession replacement does not match its envelope")
			}
			if err := runInProjectTx(ctx, tx, item.Replacement.RunID, event.ProjectID); err != nil {
				return err
			}
		}
	case "trajectory.started":
		var item domain.Trajectory
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			return err
		}
		if item.ID != event.EntityID || item.ProjectID != event.ProjectID || item.PrincipalID != event.Actor.PrincipalID || item.RunID != event.RunID {
			return errors.New("trajectory event payload does not match its envelope")
		}
		if err := validateJujutsuProvenance(item.VCS, item.JJWorkspace, item.JJChangeID, item.JJCommitID, item.JJBookmarks); err != nil {
			return err
		}
		return runInProjectTx(ctx, tx, item.RunID, event.ProjectID)
	case "repository.attached":
		var item domain.Repository
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			return err
		}
		if item.ID != event.EntityID || item.WorkspaceID != event.WorkspaceID {
			return errors.New("repository event payload does not match its envelope")
		}
		var existingWorkspace string
		err := tx.QueryRowContext(ctx, `SELECT workspace_id FROM repositories WHERE id=?`, item.ID).Scan(&existingWorkspace)
		if err == nil && existingWorkspace != event.WorkspaceID {
			return errors.New("repository ID belongs to another workspace")
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	default:
		return fmt.Errorf("unsupported domain event %q", event.Type)
	}
	return nil
}

func (s *Store) ImportEvents(ctx context.Context, workspaceID string, events []domain.DomainEvent) (int, error) {
	if len(events) > 500 {
		return 0, errors.New("event batch exceeds 500 events")
	}
	totalBytes := 0
	for _, event := range events {
		totalBytes += len(event.Payload)
	}
	if totalBytes > 1<<20 {
		return 0, errors.New("event batch exceeds 1 MiB")
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	imported := 0
	for _, event := range events {
		if event.WorkspaceID != workspaceID || event.SchemaVersion < 1 {
			return 0, errors.New("event workspace or schema is invalid")
		}
		var publicKey []byte
		var status, role string
		var acceptedThrough sql.NullInt64
		if err = tx.QueryRowContext(ctx, `SELECT public_key,status,role,accepted_through_sequence FROM workspace_replicas WHERE workspace_id=? AND replica_id=?`, workspaceID, event.OriginReplicaID).Scan(&publicKey, &status, &role, &acceptedThrough); err != nil {
			return 0, err
		}
		if status != "active" && (!acceptedThrough.Valid || event.OriginSequence > acceptedThrough.Int64) {
			return 0, errors.New("event origin is revoked")
		}
		if err = verifyEvent(event, ed25519.PublicKey(publicKey)); err != nil {
			return 0, err
		}
		var existingHash sql.NullString
		existingErr := tx.QueryRowContext(ctx, `SELECT event_hash FROM domain_events WHERE event_id=?`, event.EventID).Scan(&existingHash)
		if existingErr != nil && !errors.Is(existingErr, sql.ErrNoRows) {
			return 0, existingErr
		}
		if existingHash.Valid && existingHash.String != event.EventHash {
			return 0, errors.New("event ID was reused with different content")
		}
		var existing int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM domain_events WHERE event_id=?`, event.EventID).Scan(&existing); err != nil {
			return 0, err
		}
		if existing > 0 {
			continue
		}
		var sequence int64
		previousHash := genesisHash
		headErr := tx.QueryRowContext(ctx, `SELECT contiguous_sequence,event_hash FROM sync_heads WHERE workspace_id=? AND origin_replica_id=?`, workspaceID, event.OriginReplicaID).Scan(&sequence, &previousHash)
		if headErr != nil && !errors.Is(headErr, sql.ErrNoRows) {
			return 0, headErr
		}
		if event.OriginSequence != sequence+1 || event.PreviousHash != previousHash {
			return 0, fmt.Errorf("event gap for %s: have %d, received %d", event.OriginReplicaID, sequence, event.OriginSequence)
		}
		if event.SchemaVersion == 1 {
			if err = validateEventEnvelopeTx(ctx, tx, event, role); err != nil {
				return 0, err
			}
			if err = s.projectDomainEventTx(ctx, tx, event); err != nil {
				return 0, err
			}
		}
		causes, _ := json.Marshal(event.CausalEventIDs)
		if _, err = tx.ExecContext(ctx, `INSERT INTO domain_events(event_id,workspace_id,project_id,origin_replica_id,origin_sequence,schema_version,type,entity_id,actor_id,run_id,causal_event_ids_json,payload_json,occurred_at,previous_hash,event_hash,signature,ingested_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, event.EventID, event.WorkspaceID, nullable(event.ProjectID), event.OriginReplicaID, event.OriginSequence, event.SchemaVersion, event.Type, event.EntityID, event.ActorID, nullable(event.RunID), string(causes), string(event.Payload), ts(event.OccurredAt), event.PreviousHash, event.EventHash, []byte(event.Signature), ts(now())); err != nil {
			return 0, err
		}
		if event.SchemaVersion != 1 {
			reason := fmt.Sprintf("schema version %d is not supported by this binary", event.SchemaVersion)
			if _, err = tx.ExecContext(ctx, `INSERT OR REPLACE INTO projection_blockers(event_id,workspace_id,schema_version,reason,created_at) VALUES(?,?,?,?,?)`, event.EventID, workspaceID, event.SchemaVersion, reason, ts(now())); err != nil {
				return 0, err
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO sync_heads(workspace_id,origin_replica_id,contiguous_sequence,event_hash,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(workspace_id,origin_replica_id) DO UPDATE SET contiguous_sequence=excluded.contiguous_sequence,event_hash=excluded.event_hash,updated_at=excluded.updated_at`, workspaceID, event.OriginReplicaID, event.OriginSequence, event.EventHash, ts(now())); err != nil {
			return 0, err
		}
		imported++
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return imported, nil
}

func (s *Store) projectDomainEventTx(ctx context.Context, tx *sql.Tx, event domain.DomainEvent) error {
	actor := event.Actor
	if actor.PrincipalID == "" {
		actor.PrincipalID = newID("principal")
	}
	if actor.OriginReplicaID == "" {
		actor.OriginReplicaID = event.OriginReplicaID
	}
	if actor.OriginReplicaID != event.OriginReplicaID {
		return errors.New("event actor is not owned by its origin replica")
	}
	var existingWorkspace, existingPortable, existingOrigin string
	existingErr := tx.QueryRowContext(ctx, `SELECT workspace_id,portable_actor_id,origin_replica_id FROM principals WHERE id=?`, actor.PrincipalID).Scan(&existingWorkspace, &existingPortable, &existingOrigin)
	if existingErr == nil && (existingWorkspace != event.WorkspaceID || existingPortable != actor.PortableID || (existingOrigin != "" && existingOrigin != actor.OriginReplicaID)) {
		return errors.New("event attempts to replace an actor owned by another origin")
	}
	if existingErr != nil && !errors.Is(existingErr, sql.ErrNoRows) {
		return existingErr
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO principals(id,workspace_id,display_name,kind,created_at,portable_actor_id,origin_replica_id) VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET display_name=excluded.display_name,origin_replica_id=CASE WHEN principals.origin_replica_id='' THEN excluded.origin_replica_id ELSE principals.origin_replica_id END`, actor.PrincipalID, event.WorkspaceID, actor.DisplayName, actor.Kind, ts(event.OccurredAt), actor.PortableID, actor.OriginReplicaID); err != nil {
		return err
	}
	switch event.Type {
	case "project.created":
		var item domain.Project
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO projects(id,workspace_id,slug,name,description,created_at) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET slug=excluded.slug,name=excluded.name,description=excluded.description`, item.ID, event.WorkspaceID, item.Slug, item.Name, item.Description, ts(item.CreatedAt))
		return err
	case "run.started", "run.ended", "run.delivery_linked":
		var item domain.Run
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO agents(id,principal_id,name,created_at) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name`, item.AgentID, actor.PrincipalID, item.AgentName, ts(item.StartedAt)); err != nil {
			return err
		}
		profile, _ := json.Marshal(item.InstructionProfile)
		jjBookmarks := marshalProvenanceLabels(item.JJBookmarks)
		deliveryJJBookmarks := marshalProvenanceLabels(item.DeliveryJJBookmarks)
		_, err := tx.ExecContext(ctx, `INSERT INTO runs(id,project_id,agent_id,principal_id,harness,harness_version,provider,model,reasoning,role,parent_run_id,root_run_id,run_type,permission_mode,interaction_mode,repository_id,vcs,branch,worktree,base_sha,head_sha,jj_workspace,jj_change_id,jj_commit_id,jj_bookmarks_json,delivery_branch,delivery_jj_workspace,delivery_jj_change_id,delivery_jj_commit_id,delivery_jj_bookmarks_json,pull_request_url,pull_request_number,pull_request_state,merge_commit_sha,merged_at,objective,instruction_profile_json,started_at,ended_at,outcome,verification) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET vcs=CASE WHEN excluded.vcs='' THEN runs.vcs ELSE excluded.vcs END,head_sha=CASE WHEN excluded.head_sha='' THEN runs.head_sha ELSE excluded.head_sha END,delivery_branch=CASE WHEN excluded.delivery_branch='' THEN runs.delivery_branch ELSE excluded.delivery_branch END,delivery_jj_workspace=CASE WHEN excluded.delivery_jj_workspace='' THEN runs.delivery_jj_workspace ELSE excluded.delivery_jj_workspace END,delivery_jj_change_id=CASE WHEN excluded.delivery_jj_change_id='' THEN runs.delivery_jj_change_id ELSE excluded.delivery_jj_change_id END,delivery_jj_commit_id=CASE WHEN excluded.delivery_jj_commit_id='' THEN runs.delivery_jj_commit_id ELSE excluded.delivery_jj_commit_id END,delivery_jj_bookmarks_json=CASE WHEN excluded.delivery_jj_bookmarks_json='[]' THEN runs.delivery_jj_bookmarks_json ELSE excluded.delivery_jj_bookmarks_json END,pull_request_url=CASE WHEN excluded.pull_request_url='' THEN runs.pull_request_url ELSE excluded.pull_request_url END,pull_request_number=CASE WHEN excluded.pull_request_number=0 THEN runs.pull_request_number ELSE excluded.pull_request_number END,pull_request_state=CASE WHEN excluded.pull_request_state='' THEN runs.pull_request_state ELSE excluded.pull_request_state END,merge_commit_sha=CASE WHEN excluded.merge_commit_sha='' THEN runs.merge_commit_sha ELSE excluded.merge_commit_sha END,merged_at=COALESCE(excluded.merged_at,runs.merged_at),ended_at=COALESCE(excluded.ended_at,runs.ended_at),outcome=CASE WHEN excluded.outcome='' THEN runs.outcome ELSE excluded.outcome END,verification=CASE WHEN excluded.verification='' THEN runs.verification ELSE excluded.verification END`, item.ID, item.ProjectID, item.AgentID, actor.PrincipalID, item.Harness, item.HarnessVersion, item.Provider, item.Model, item.Reasoning, item.Role, nullable(item.ParentRunID), nullable(item.RootRunID), item.RunType, item.PermissionMode, item.InteractionMode, nullable(item.RepositoryID), item.VCS, item.Branch, item.Worktree, item.BaseSHA, item.HeadSHA, item.JJWorkspace, item.JJChangeID, item.JJCommitID, jjBookmarks, item.DeliveryBranch, item.DeliveryJJWorkspace, item.DeliveryJJChangeID, item.DeliveryJJCommitID, deliveryJJBookmarks, item.PullRequestURL, item.PullRequestNumber, item.PullRequestState, item.MergeCommitSHA, nullableTime(item.MergedAt), item.Objective, string(profile), ts(item.StartedAt), nullableTime(item.EndedAt), item.Outcome, item.Verification)
		if err == nil && item.RepositoryID != "" {
			_, err = tx.ExecContext(ctx, `UPDATE runs SET repository_id=? WHERE id=? AND repository_id IS NULL`, item.RepositoryID, item.ID)
		}
		if err == nil && event.Type == "run.ended" {
			_, err = tx.ExecContext(ctx, `UPDATE trajectories SET status='closed',updated_at=? WHERE run_id=? AND status='active'`, ts(event.OccurredAt), item.ID)
		}
		return err
	case "note.recorded":
		var item domain.Note
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			return err
		}
		return upsertNoteTx(ctx, tx, item)
	case "note.superseded":
		var item struct {
			Superseded  domain.Note `json:"superseded"`
			Replacement domain.Note `json:"replacement"`
		}
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			return err
		}
		if err := upsertNoteTx(ctx, tx, item.Superseded); err != nil {
			return err
		}
		if item.Replacement.ID != "" {
			if err := upsertNoteTx(ctx, tx, item.Replacement); err != nil {
				return err
			}
		}
		return recordLifecycleEdgeTx(ctx, tx, event)
	case "trajectory.started":
		var item domain.Trajectory
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			return err
		}
		paths, _ := json.Marshal(item.Paths)
		jjBookmarks := marshalProvenanceLabels(item.JJBookmarks)
		_, err := tx.ExecContext(ctx, `INSERT INTO trajectories(id,project_id,run_id,principal_id,objective,rationale,status,paths_json,repository_id,vcs,branch,base_sha,head_sha,jj_workspace,jj_change_id,jj_commit_id,jj_bookmarks_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET status=excluded.status,updated_at=excluded.updated_at`, item.ID, item.ProjectID, item.RunID, actor.PrincipalID, item.Objective, item.Rationale, item.Status, string(paths), nullable(item.RepositoryID), item.VCS, item.Branch, item.BaseSHA, item.HeadSHA, item.JJWorkspace, item.JJChangeID, item.JJCommitID, jjBookmarks, ts(item.CreatedAt), ts(item.UpdatedAt))
		return err
	case "repository.attached":
		var item domain.Repository
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO repositories(id,workspace_id,url,host,owner,name,visibility,description,default_branch,stars) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET url=excluded.url,description=excluded.description,default_branch=excluded.default_branch,stars=excluded.stars`, item.ID, event.WorkspaceID, item.URL, item.Host, item.Owner, item.Name, item.Visibility, item.Description, item.Default, item.Stars); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO project_repositories(project_id,repository_id,is_primary,path_scope) VALUES(?,?,0,'')`, event.ProjectID, item.ID)
		return err
	default:
		return fmt.Errorf("unsupported domain event %q", event.Type)
	}
}

func upsertNoteTx(ctx context.Context, tx *sql.Tx, item domain.Note) error {
	paths, _ := json.Marshal(item.Paths)
	if _, err := tx.ExecContext(ctx, `INSERT INTO notes(id,project_id,run_id,principal_id,kind,title,summary,rationale,status,led_by,direction_basis,confidence,verification,source_ref,paths_json,repository_id,pull_request_url,revision,superseded_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET status=excluded.status,revision=excluded.revision,superseded_by=excluded.superseded_by,updated_at=excluded.updated_at`, item.ID, item.ProjectID, nullable(item.RunID), item.PrincipalID, item.Kind, item.Title, item.Summary, item.Rationale, item.Status, item.LedBy, item.DirectionBasis, item.Confidence, item.Verification, item.SourceRef, string(paths), nullable(item.RepositoryID), item.PullRequestURL, item.Revision, nullable(item.SupersededBy), ts(item.CreatedAt), ts(item.UpdatedAt)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM notes_fts WHERE note_id=?`, item.ID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO notes_fts(note_id,project_id,title,summary,rationale) VALUES(?,?,?,?,?)`, item.ID, item.ProjectID, item.Title, item.Summary, item.Rationale)
	return err
}

func recordLifecycleEdgeTx(ctx context.Context, tx *sql.Tx, event domain.DomainEvent) error {
	var payload struct {
		Superseded  domain.Note `json:"superseded"`
		Replacement domain.Note `json:"replacement"`
		Reason      string      `json:"reason"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}
	if payload.Superseded.ID == "" {
		return errors.New("supersession event has no target note")
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO note_lifecycle_edges(event_id,project_id,target_note_id,replacement_note_id,reason,origin_replica_id,occurred_at) VALUES(?,?,?,?,?,?,?)`, event.EventID, payload.Superseded.ProjectID, payload.Superseded.ID, nullable(payload.Replacement.ID), payload.Reason, event.OriginReplicaID, ts(event.OccurredAt)); err != nil {
		return err
	}
	var successors int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(DISTINCT replacement_note_id) FROM note_lifecycle_edges WHERE target_note_id=? AND replacement_note_id IS NOT NULL AND replacement_note_id!=''`, payload.Superseded.ID).Scan(&successors); err != nil {
		return err
	}
	if successors > 1 {
		_, err := tx.ExecContext(ctx, `UPDATE notes SET status='contested',superseded_by=NULL,updated_at=? WHERE id=?`, ts(event.OccurredAt), payload.Superseded.ID)
		return err
	}
	return nil
}

func (s *Store) ListLifecycleEdges(ctx context.Context, projectID string) ([]domain.NoteLifecycleEdge, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT event_id,project_id,target_note_id,COALESCE(replacement_note_id,''),reason,origin_replica_id,occurred_at FROM note_lifecycle_edges WHERE project_id=? ORDER BY occurred_at,event_id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.NoteLifecycleEdge{}
	for rows.Next() {
		var item domain.NoteLifecycleEdge
		var occurred string
		if err = rows.Scan(&item.EventID, &item.ProjectID, &item.TargetNoteID, &item.ReplacementNoteID, &item.Reason, &item.OriginReplicaID, &occurred); err != nil {
			return nil, err
		}
		item.OccurredAt = parseTime(occurred)
		items = append(items, item)
	}
	return items, rows.Err()
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return ts(*value)
}

type ReplicaLink struct {
	ID              string
	WorkspaceID     string
	RemoteReplicaID string
	RemoteURL       string
	Direction       string
	Credential      string
	IntervalSeconds int
	LastSuccessAt   *time.Time
	LastError       string
}

func verifySnapshot(snapshot domain.WorkspaceSnapshot, authority domain.Replica) error {
	publicKey, err := base64.RawURLEncoding.DecodeString(authority.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return errors.New("authority public key is invalid")
	}
	unsigned := snapshot
	unsigned.SnapshotHash, unsigned.Signature = "", ""
	body, err := json.Marshal(unsigned)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(body)
	if snapshot.SnapshotHash != "sha256:"+hex.EncodeToString(hash[:]) {
		return errors.New("snapshot hash mismatch")
	}
	signature, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(snapshot.Signature, "ed25519:"))
	if err != nil || !ed25519.Verify(ed25519.PublicKey(publicKey), []byte(snapshot.SnapshotHash), signature) {
		return errors.New("snapshot signature is invalid")
	}
	return nil
}

func (s *Store) ImportWorkspaceSnapshot(ctx context.Context, snapshot domain.WorkspaceSnapshot, authority domain.Replica, localUserID string) error {
	if snapshot.SchemaVersion != 1 || snapshot.Workspace.ID == "" || snapshot.Workspace.AuthorityReplicaID != authority.ID {
		return errors.New("snapshot workspace or authority is invalid")
	}
	if err := verifySnapshot(snapshot, authority); err != nil {
		return err
	}
	localID, localName, localBaseURL, localPublic, _, ok := s.localReplica()
	if !ok {
		return errors.New("installation identity is not initialized")
	}
	authorityPublic, _ := base64.RawURLEncoding.DecodeString(authority.PublicKey)
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	workspace := snapshot.Workspace
	var slugOwner string
	err = tx.QueryRowContext(ctx, `SELECT id FROM workspaces WHERE slug=?`, workspace.Slug).Scan(&slugOwner)
	if err == nil && slugOwner != workspace.ID {
		workspace.Slug += "-" + workspace.ID[len(workspace.ID)-6:]
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if workspace.Status == "" {
		workspace.Status = "active"
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspaces(id,name,created_at,slug,authority_replica_id,status) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,slug=excluded.slug,authority_replica_id=excluded.authority_replica_id`, workspace.ID, workspace.Name, ts(workspace.CreatedAt), workspace.Slug, authority.ID, workspace.Status); err != nil {
		return err
	}
	authorityCapabilities, _ := json.Marshal(authority.Capabilities)
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_replicas(workspace_id,replica_id,display_name,base_url,public_key,role,capabilities_json,status,approved_at) VALUES(?,?,?,?,?,'authority',?,'active',?) ON CONFLICT(workspace_id,replica_id) DO UPDATE SET display_name=excluded.display_name,base_url=excluded.base_url,public_key=excluded.public_key,role='authority',status='active'`, workspace.ID, authority.ID, authority.DisplayName, authority.BaseURL, authorityPublic, string(authorityCapabilities), ts(authority.ApprovedAt)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_replicas(workspace_id,replica_id,display_name,base_url,public_key,role,capabilities_json,status,approved_at) VALUES(?,?,?,?,?,'replica','["pull","push"]','active',?) ON CONFLICT(workspace_id,replica_id) DO UPDATE SET display_name=excluded.display_name,base_url=excluded.base_url,public_key=excluded.public_key,status='active'`, workspace.ID, localID, localName, localBaseURL, localPublic, ts(now())); err != nil {
		return err
	}
	for _, replica := range snapshot.Replicas {
		key, keyErr := base64.RawURLEncoding.DecodeString(replica.PublicKey)
		if keyErr != nil || len(key) != ed25519.PublicKeySize {
			return errors.New("snapshot replica public key is invalid")
		}
		capabilities, _ := json.Marshal(replica.Capabilities)
		if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_replicas(workspace_id,replica_id,display_name,base_url,public_key,role,capabilities_json,status,accepted_through_sequence,approved_at,revoked_at) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(workspace_id,replica_id) DO UPDATE SET display_name=excluded.display_name,base_url=excluded.base_url,public_key=excluded.public_key,role=excluded.role,capabilities_json=excluded.capabilities_json,status=excluded.status,accepted_through_sequence=excluded.accepted_through_sequence,revoked_at=excluded.revoked_at`, workspace.ID, replica.ID, replica.DisplayName, replica.BaseURL, key, replica.Role, string(capabilities), replica.Status, replica.AcceptedThroughSequence, ts(replica.ApprovedAt), nullableTime(replica.RevokedAt)); err != nil {
			return err
		}
	}
	for _, actor := range snapshot.Actors {
		if _, err = tx.ExecContext(ctx, `INSERT INTO principals(id,workspace_id,display_name,kind,created_at,portable_actor_id,origin_replica_id) VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET display_name=excluded.display_name,portable_actor_id=excluded.portable_actor_id,origin_replica_id=excluded.origin_replica_id`, actor.PrincipalID, workspace.ID, actor.DisplayName, actor.Kind, ts(snapshot.CreatedAt), actor.PortableID, actor.OriginReplicaID); err != nil {
			return err
		}
	}
	if localUserID != "" {
		var count int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_memberships WHERE workspace_id=? AND user_id=?`, workspace.ID, localUserID).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			var displayName string
			if err = tx.QueryRowContext(ctx, `SELECT display_name FROM users WHERE id=? AND status='active'`, localUserID).Scan(&displayName); err != nil {
				return err
			}
			principalID := newID("principal")
			if _, err = tx.ExecContext(ctx, `INSERT INTO principals(id,workspace_id,display_name,kind,created_at,user_id,portable_actor_id,origin_replica_id) VALUES(?,?,?,?,?,?,?,?)`, principalID, workspace.ID, displayName, "human", ts(now()), localUserID, newID("actor"), localID); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_memberships(id,workspace_id,user_id,principal_id,role,status,created_at) VALUES(?,?,?,?,?,'active',?)`, newID("membership"), workspace.ID, localUserID, principalID, "owner", ts(now())); err != nil {
				return err
			}
		}
	}
	for _, agent := range snapshot.Agents {
		if _, err = tx.ExecContext(ctx, `INSERT INTO agents(id,principal_id,name,created_at) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name`, agent.ID, agent.PrincipalID, agent.Name, ts(agent.CreatedAt)); err != nil {
			return err
		}
	}
	for _, projectSnapshot := range snapshot.Projects {
		project := projectSnapshot.Project
		if _, err = tx.ExecContext(ctx, `INSERT INTO projects(id,workspace_id,slug,name,description,created_at) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,description=excluded.description`, project.ID, workspace.ID, project.Slug, project.Name, project.Description, ts(project.CreatedAt)); err != nil {
			return err
		}
		for _, run := range projectSnapshot.Runs {
			if err = validateJujutsuProvenance(run.VCS, run.JJWorkspace, run.JJChangeID, run.JJCommitID, run.JJBookmarks); err != nil {
				return err
			}
			if err = validateJujutsuProvenance(run.VCS, run.DeliveryJJWorkspace, run.DeliveryJJChangeID, run.DeliveryJJCommitID, run.DeliveryJJBookmarks); err != nil {
				return err
			}
			profile, _ := json.Marshal(run.InstructionProfile)
			jjBookmarks := marshalProvenanceLabels(run.JJBookmarks)
			deliveryJJBookmarks := marshalProvenanceLabels(run.DeliveryJJBookmarks)
			if _, err = tx.ExecContext(ctx, `INSERT INTO runs(id,project_id,agent_id,principal_id,harness,harness_version,provider,model,reasoning,role,parent_run_id,root_run_id,run_type,permission_mode,interaction_mode,repository_id,vcs,branch,worktree,base_sha,head_sha,jj_workspace,jj_change_id,jj_commit_id,jj_bookmarks_json,delivery_branch,delivery_jj_workspace,delivery_jj_change_id,delivery_jj_commit_id,delivery_jj_bookmarks_json,pull_request_url,pull_request_number,pull_request_state,merge_commit_sha,merged_at,objective,instruction_profile_json,started_at,ended_at,outcome,verification) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET vcs=CASE WHEN excluded.vcs='' THEN runs.vcs ELSE excluded.vcs END,head_sha=CASE WHEN excluded.head_sha='' THEN runs.head_sha ELSE excluded.head_sha END,delivery_branch=CASE WHEN excluded.delivery_branch='' THEN runs.delivery_branch ELSE excluded.delivery_branch END,delivery_jj_workspace=CASE WHEN excluded.delivery_jj_workspace='' THEN runs.delivery_jj_workspace ELSE excluded.delivery_jj_workspace END,delivery_jj_change_id=CASE WHEN excluded.delivery_jj_change_id='' THEN runs.delivery_jj_change_id ELSE excluded.delivery_jj_change_id END,delivery_jj_commit_id=CASE WHEN excluded.delivery_jj_commit_id='' THEN runs.delivery_jj_commit_id ELSE excluded.delivery_jj_commit_id END,delivery_jj_bookmarks_json=CASE WHEN excluded.delivery_jj_bookmarks_json='[]' THEN runs.delivery_jj_bookmarks_json ELSE excluded.delivery_jj_bookmarks_json END,pull_request_url=CASE WHEN excluded.pull_request_url='' THEN runs.pull_request_url ELSE excluded.pull_request_url END,pull_request_number=CASE WHEN excluded.pull_request_number=0 THEN runs.pull_request_number ELSE excluded.pull_request_number END,pull_request_state=CASE WHEN excluded.pull_request_state='' THEN runs.pull_request_state ELSE excluded.pull_request_state END,merge_commit_sha=CASE WHEN excluded.merge_commit_sha='' THEN runs.merge_commit_sha ELSE excluded.merge_commit_sha END,merged_at=COALESCE(excluded.merged_at,runs.merged_at),ended_at=COALESCE(excluded.ended_at,runs.ended_at),outcome=CASE WHEN excluded.outcome='' THEN runs.outcome ELSE excluded.outcome END,verification=CASE WHEN excluded.verification='' THEN runs.verification ELSE excluded.verification END`, run.ID, project.ID, run.AgentID, run.PrincipalID, run.Harness, run.HarnessVersion, run.Provider, run.Model, run.Reasoning, run.Role, nullable(run.ParentRunID), nullable(run.RootRunID), run.RunType, run.PermissionMode, run.InteractionMode, nullable(run.RepositoryID), run.VCS, run.Branch, run.Worktree, run.BaseSHA, run.HeadSHA, run.JJWorkspace, run.JJChangeID, run.JJCommitID, jjBookmarks, run.DeliveryBranch, run.DeliveryJJWorkspace, run.DeliveryJJChangeID, run.DeliveryJJCommitID, deliveryJJBookmarks, run.PullRequestURL, run.PullRequestNumber, run.PullRequestState, run.MergeCommitSHA, nullableTime(run.MergedAt), run.Objective, string(profile), ts(run.StartedAt), nullableTime(run.EndedAt), run.Outcome, run.Verification); err != nil {
				return err
			}
			if run.RepositoryID != "" {
				if _, err = tx.ExecContext(ctx, `UPDATE runs SET repository_id=? WHERE id=? AND repository_id IS NULL`, run.RepositoryID, run.ID); err != nil {
					return err
				}
			}
		}
		for _, note := range projectSnapshot.Notes {
			if err = upsertNoteTx(ctx, tx, note); err != nil {
				return err
			}
		}
		for _, trajectory := range projectSnapshot.Trajectories {
			if err = validateJujutsuProvenance(trajectory.VCS, trajectory.JJWorkspace, trajectory.JJChangeID, trajectory.JJCommitID, trajectory.JJBookmarks); err != nil {
				return err
			}
			paths, _ := json.Marshal(trajectory.Paths)
			jjBookmarks := marshalProvenanceLabels(trajectory.JJBookmarks)
			if _, err = tx.ExecContext(ctx, `INSERT INTO trajectories(id,project_id,run_id,principal_id,objective,rationale,status,paths_json,repository_id,vcs,branch,base_sha,head_sha,jj_workspace,jj_change_id,jj_commit_id,jj_bookmarks_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET status=excluded.status,updated_at=excluded.updated_at`, trajectory.ID, project.ID, trajectory.RunID, trajectory.PrincipalID, trajectory.Objective, trajectory.Rationale, trajectory.Status, string(paths), nullable(trajectory.RepositoryID), trajectory.VCS, trajectory.Branch, trajectory.BaseSHA, trajectory.HeadSHA, trajectory.JJWorkspace, trajectory.JJChangeID, trajectory.JJCommitID, jjBookmarks, ts(trajectory.CreatedAt), ts(trajectory.UpdatedAt)); err != nil {
				return err
			}
		}
		for _, repository := range projectSnapshot.Repositories {
			if _, err = tx.ExecContext(ctx, `INSERT INTO repositories(id,workspace_id,url,host,owner,name,visibility,description,default_branch,stars) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET url=excluded.url,description=excluded.description,default_branch=excluded.default_branch,stars=excluded.stars`, repository.ID, workspace.ID, repository.URL, repository.Host, repository.Owner, repository.Name, repository.Visibility, repository.Description, repository.Default, repository.Stars); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO project_repositories(project_id,repository_id,is_primary,path_scope) VALUES(?,?,0,'')`, project.ID, repository.ID); err != nil {
				return err
			}
		}
		for _, edge := range projectSnapshot.LifecycleEdges {
			if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO note_lifecycle_edges(event_id,project_id,target_note_id,replacement_note_id,reason,origin_replica_id,occurred_at) VALUES(?,?,?,?,?,?,?)`, edge.EventID, project.ID, edge.TargetNoteID, nullable(edge.ReplacementNoteID), edge.Reason, edge.OriginReplicaID, ts(edge.OccurredAt)); err != nil {
				return err
			}
		}
	}
	for _, head := range snapshot.Heads {
		if _, err = tx.ExecContext(ctx, `INSERT INTO sync_heads(workspace_id,origin_replica_id,contiguous_sequence,event_hash,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(workspace_id,origin_replica_id) DO UPDATE SET contiguous_sequence=excluded.contiguous_sequence,event_hash=excluded.event_hash,updated_at=excluded.updated_at`, workspace.ID, head.OriginReplicaID, head.Sequence, head.EventHash, ts(now())); err != nil {
			return err
		}
	}
	payload, _ := json.Marshal(snapshot)
	heads, _ := json.Marshal(snapshot.Heads)
	if _, err = tx.ExecContext(ctx, `INSERT INTO sync_snapshots(id,workspace_id,covers_vector_json,payload_json,snapshot_hash,signature,created_at) VALUES(?,?,?,?,?,?,?)`, newID("snapshot"), workspace.ID, string(heads), string(payload), snapshot.SnapshotHash, []byte(snapshot.Signature), ts(snapshot.CreatedAt)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SaveReplicaLink(ctx context.Context, workspaceID, remoteReplicaID, remoteURL, credential string) error {
	encrypted, err := s.seal([]byte(credential))
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO replica_links(id,workspace_id,remote_replica_id,remote_url,direction,credential_ciphertext,interval_seconds) VALUES(?,?,?,?,'push_pull',?,5) ON CONFLICT(workspace_id,remote_replica_id) DO UPDATE SET remote_url=excluded.remote_url,credential_ciphertext=excluded.credential_ciphertext,direction='push_pull',enabled=1`, newID("link"), workspaceID, remoteReplicaID, strings.TrimRight(remoteURL, "/"), encrypted)
	return err
}

func (s *Store) ListReplicaLinks(ctx context.Context) ([]ReplicaLink, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,workspace_id,remote_replica_id,remote_url,direction,credential_ciphertext,interval_seconds,last_success_at,last_error FROM replica_links WHERE enabled=1 ORDER BY workspace_id,remote_replica_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ReplicaLink{}
	for rows.Next() {
		var item ReplicaLink
		var encrypted []byte
		var success sql.NullString
		if err = rows.Scan(&item.ID, &item.WorkspaceID, &item.RemoteReplicaID, &item.RemoteURL, &item.Direction, &encrypted, &item.IntervalSeconds, &success, &item.LastError); err != nil {
			return nil, err
		}
		credential, openErr := s.openSealed(encrypted)
		if openErr != nil {
			return nil, openErr
		}
		item.Credential = string(credential)
		if success.Valid {
			value := parseTime(success.String)
			item.LastSuccessAt = &value
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) RecordLinkResult(ctx context.Context, linkID string, syncErr error) error {
	if syncErr == nil {
		_, err := s.db.ExecContext(ctx, `UPDATE replica_links SET last_attempt_at=?,last_success_at=?,last_error='' WHERE id=?`, ts(now()), ts(now()), linkID)
		return err
	}
	message := syncErr.Error()
	if len(message) > 500 {
		message = message[:500]
	}
	_, err := s.db.ExecContext(ctx, `UPDATE replica_links SET last_attempt_at=?,last_error=? WHERE id=?`, ts(now()), message, linkID)
	return err
}

func (s *Store) LocalReplicaID() string {
	id, _, _, _, _, _ := s.localReplica()
	return id
}

func (s *Store) ProjectionBlockerCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projection_blockers`).Scan(&count)
	return count, err
}

func (s *Store) EventsForOriginAfter(ctx context.Context, workspaceID, originID string, after int64, limit int) ([]domain.DomainEvent, error) {
	heads, err := s.syncHeads(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	found := false
	for index := range heads {
		if heads[index].OriginReplicaID == originID {
			heads[index].Sequence = after
			found = true
		}
	}
	if !found {
		heads = append(heads, domain.SyncHead{OriginReplicaID: originID, Sequence: after})
	}
	items, _, err := s.EventsAfter(ctx, workspaceID, heads, limit)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.DomainEvent, 0, len(items))
	for _, item := range items {
		if item.OriginReplicaID == originID {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (s *Store) ImportReplicaRoster(ctx context.Context, workspaceID, authorityID string, replicas []domain.Replica) error {
	var actualAuthority string
	if err := s.db.QueryRowContext(ctx, `SELECT authority_replica_id FROM workspaces WHERE id=?`, workspaceID).Scan(&actualAuthority); err != nil {
		return err
	}
	if authorityID != actualAuthority {
		return errors.New("replica roster did not come from workspace authority")
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, replica := range replicas {
		key, keyErr := base64.RawURLEncoding.DecodeString(replica.PublicKey)
		if keyErr != nil || len(key) != ed25519.PublicKeySize {
			return errors.New("replica roster contains an invalid public key")
		}
		capabilities, _ := json.Marshal(replica.Capabilities)
		if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_replicas(workspace_id,replica_id,display_name,base_url,public_key,role,capabilities_json,status,accepted_through_sequence,approved_at,revoked_at,last_success_at,last_error) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(workspace_id,replica_id) DO UPDATE SET display_name=excluded.display_name,base_url=excluded.base_url,public_key=excluded.public_key,role=excluded.role,capabilities_json=excluded.capabilities_json,status=excluded.status,accepted_through_sequence=excluded.accepted_through_sequence,revoked_at=excluded.revoked_at,last_success_at=excluded.last_success_at,last_error=excluded.last_error`, workspaceID, replica.ID, replica.DisplayName, replica.BaseURL, key, replica.Role, string(capabilities), replica.Status, replica.AcceptedThroughSequence, ts(replica.ApprovedAt), nullableTime(replica.RevokedAt), nullableTime(replica.LastSuccessAt), replica.LastError); err != nil {
			return err
		}
	}
	return tx.Commit()
}
