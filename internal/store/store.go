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
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrConflict            = errors.New("revision conflict")
	ErrIdempotencyKeyReuse = errors.New("idempotency key reused with different request")
)

var fullGitSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}([0-9a-fA-F]{24})?$`)

type Store struct {
	db             *sql.DB
	writeMu        sync.Mutex
	secretKey      [32]byte
	replicaMu      sync.RWMutex
	replicaID      string
	replicaName    string
	replicaBaseURL string
	replicaPublic  ed25519.PublicKey
	replicaPrivate ed25519.PrivateKey
}

type scanQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func Open(path string) (*Store, error) {
	secret, err := installationSecret(path)
	if err != nil {
		return nil, err
	}
	return OpenWithSecret(path, secret)
}

func OpenWithSecret(path, secret string) (*Store, error) {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	dsn := path + separator + "_txlock=immediate&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	if err := prepareMigrationBackup(db, path); err != nil {
		db.Close()
		return nil, fmt.Errorf("prepare migration backup: %w", err)
	}
	if err := applyMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	return &Store{db: db, secretKey: sha256.Sum256([]byte(secret))}, nil
}

func prepareMigrationBackup(db *sql.DB, path string) error {
	if path == ":memory:" || strings.Contains(path, "mode=memory") {
		return nil
	}
	var version int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		return err
	}
	if version == 0 || version >= latestSchemaVersion {
		return nil
	}
	var integrity string
	if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return err
	}
	if integrity != "ok" {
		return fmt.Errorf("database integrity check failed: %s", integrity)
	}
	cleanPath := strings.Split(path, "?")[0]
	backupPath := fmt.Sprintf("%s.pre-migration-v%d-%d", cleanPath, version, time.Now().UTC().UnixNano())
	if _, err := db.Exec(`VACUUM INTO ?`, backupPath); err != nil {
		return err
	}
	if err := os.Chmod(backupPath, 0600); err != nil {
		return err
	}
	return nil
}

func installationSecret(path string) (string, error) {
	if value := strings.TrimSpace(os.Getenv("CLANKSPACE_INSTALLATION_SECRET")); value != "" {
		return value, nil
	}
	secretPath := filepath.Join(filepath.Dir(path), "clankspace.secret")
	body, err := os.ReadFile(secretPath)
	if err == nil {
		if len(body) < 32 {
			return "", errors.New("installation secret file is invalid")
		}
		return string(body), nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	random := make([]byte, 32)
	if _, err = rand.Read(random); err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(random)
	if err = os.MkdirAll(filepath.Dir(secretPath), 0700); err != nil {
		return "", err
	}
	if err = os.WriteFile(secretPath, []byte(encoded), 0600); err != nil {
		return "", err
	}
	return encoded, nil
}

func (s *Store) Close() error                   { return s.db.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func newID(prefix string) string {
	id, err := uuid.NewV7()
	if err != nil {
		id = uuid.New()
	}
	return prefix + "_" + strings.ReplaceAll(id.String(), "-", "")
}

func now() time.Time        { return time.Now().UTC().Truncate(time.Millisecond) }
func ts(t time.Time) string { return t.Format(time.RFC3339Nano) }

func parseTime(v string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, v)
	return t
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Store) EnsureBootstrap(ctx context.Context, token, workspaceName, ownerName string) (domain.Principal, error) {
	var existing int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM principals`).Scan(&existing); err != nil {
		return domain.Principal{}, err
	}
	if existing > 0 {
		return s.Authenticate(ctx, token)
	}
	if token == "" {
		return domain.Principal{}, errors.New("bootstrap token is required for an empty database")
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	t := now()
	workspaceID, principalID := newID("ws"), newID("principal")
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Principal{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspaces(id,name,created_at,slug) VALUES(?,?,?,?)`, workspaceID, workspaceName, ts(t), slugForWorkspace(workspaceName, workspaceID)); err != nil {
		return domain.Principal{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO principals(id,workspace_id,display_name,kind,created_at,portable_actor_id) VALUES(?,?,?,?,?,?)`, principalID, workspaceID, ownerName, "human", ts(t), newID("actor")); err != nil {
		return domain.Principal{}, err
	}
	prefix := token
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO api_tokens(id,principal_id,token_hash,token_prefix,scopes_json,created_at) VALUES(?,?,?,?,?,?)`, newID("token"), principalID, hashToken(token), prefix, `["admin"]`, ts(t)); err != nil {
		return domain.Principal{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.Principal{}, err
	}
	return domain.Principal{ID: principalID, WorkspaceID: workspaceID, DisplayName: ownerName, Kind: "human", CreatedAt: t}, nil
}

func (s *Store) Authenticate(ctx context.Context, token string) (domain.Principal, error) {
	auth, err := s.AuthenticateContext(ctx, token)
	return auth.Principal, err
}

func (s *Store) AuthenticateContext(ctx context.Context, token string) (domain.AuthContext, error) {
	var p domain.Principal
	var created, tokenID, scopesJSON, workspaceStatus string
	var expires, issuerMembershipStatus, issuerUserStatus sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT p.id,p.workspace_id,p.display_name,p.kind,p.created_at,t.id,t.scopes_json,t.expires_at,w.status,m.status,u.status FROM api_tokens t JOIN principals p ON p.id=t.principal_id JOIN workspaces w ON w.id=p.workspace_id LEFT JOIN workspace_memberships m ON m.id=t.issued_by_membership_id LEFT JOIN users u ON u.id=m.user_id WHERE t.token_hash=? AND t.revoked_at IS NULL`, hashToken(token)).Scan(&p.ID, &p.WorkspaceID, &p.DisplayName, &p.Kind, &created, &tokenID, &scopesJSON, &expires, &workspaceStatus, &issuerMembershipStatus, &issuerUserStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AuthContext{}, ErrNotFound
	}
	if err != nil {
		return domain.AuthContext{}, err
	}
	if expires.Valid && !parseTime(expires.String).After(now()) {
		return domain.AuthContext{}, ErrNotFound
	}
	if (issuerMembershipStatus.Valid && issuerMembershipStatus.String != "active") || (issuerUserStatus.Valid && issuerUserStatus.String != "active") {
		return domain.AuthContext{}, ErrNotFound
	}
	p.CreatedAt = parseTime(created)
	var scopes []string
	if err = json.Unmarshal([]byte(scopesJSON), &scopes); err != nil {
		return domain.AuthContext{}, fmt.Errorf("parse token scopes: %w", err)
	}
	if workspaceStatus != "active" {
		isAdmin := false
		for _, scope := range scopes {
			isAdmin = isAdmin || scope == "admin"
		}
		if !isAdmin {
			return domain.AuthContext{}, ErrNotFound
		}
	}
	projectIDs := []string{}
	if p.Kind != "human" {
		rows, queryErr := s.db.QueryContext(ctx, `SELECT project_id FROM project_principals WHERE principal_id=? ORDER BY project_id`, p.ID)
		if queryErr != nil {
			return domain.AuthContext{}, queryErr
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if queryErr = rows.Scan(&id); queryErr != nil {
				return domain.AuthContext{}, queryErr
			}
			projectIDs = append(projectIDs, id)
		}
		if queryErr = rows.Err(); queryErr != nil {
			return domain.AuthContext{}, queryErr
		}
	}
	return domain.AuthContext{Principal: p, TokenID: tokenID, Scopes: scopes, ProjectIDs: projectIDs}, nil
}

func (s *Store) ListProjects(ctx context.Context, workspaceID string) ([]domain.Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,workspace_id,slug,name,description,created_at FROM projects WHERE workspace_id=? ORDER BY name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Project, 0)
	for rows.Next() {
		var p domain.Project
		var created string
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Slug, &p.Name, &p.Description, &created); err != nil {
			return nil, err
		}
		p.CreatedAt = parseTime(created)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ListProjectsForPrincipal(ctx context.Context, p domain.Principal) ([]domain.Project, error) {
	if p.Kind == "human" {
		return s.ListProjects(ctx, p.WorkspaceID)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT p.id,p.workspace_id,p.slug,p.name,p.description,p.created_at FROM projects p JOIN project_principals pp ON pp.project_id=p.id WHERE pp.principal_id=? ORDER BY p.name`, p.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Project, 0)
	for rows.Next() {
		var p domain.Project
		var created string
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Slug, &p.Name, &p.Description, &created); err != nil {
			return nil, err
		}
		p.CreatedAt = parseTime(created)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) CanAccessProject(ctx context.Context, p domain.Principal, projectID string) (bool, error) {
	if p.Kind == "human" {
		var n int
		err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE id=? AND workspace_id=?`, projectID, p.WorkspaceID).Scan(&n)
		return n > 0, err
	}
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM project_principals pp JOIN projects pr ON pr.id=pp.project_id WHERE pp.project_id=? AND pp.principal_id=? AND pr.workspace_id=?`, projectID, p.ID, p.WorkspaceID).Scan(&n)
	return n > 0, err
}

func (s *Store) IssueProjectToken(ctx context.Context, owner domain.Principal, projectID, key, displayName string) (domain.ProjectCredential, domain.Receipt, error) {
	if owner.Kind != "human" {
		return domain.ProjectCredential{}, domain.Receipt{}, errors.New("only a human workspace owner may issue project credentials")
	}
	project, err := s.GetProject(ctx, owner.WorkspaceID, projectID)
	if err != nil {
		return domain.ProjectCredential{}, domain.Receipt{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = project.Name + " agents"
	}
	request := struct {
		DisplayName string `json:"displayName"`
	}{DisplayName: displayName}
	body, receipt, err := s.mutate(ctx, owner.WorkspaceID, project.ID, owner.ID, owner.ID, "", key, request, func(tx *sql.Tx) (string, string, any, error) {
		random := make([]byte, 32)
		if _, err = rand.Read(random); err != nil {
			return "", "", nil, err
		}
		token := "clank_" + base64.RawURLEncoding.EncodeToString(random)
		t := now()
		p := domain.Principal{ID: newID("principal"), WorkspaceID: owner.WorkspaceID, DisplayName: displayName, Kind: "project", CreatedAt: t}
		var issuerMembership string
		queryErr := tx.QueryRowContext(ctx, `SELECT id FROM workspace_memberships WHERE principal_id=? AND status='active'`, owner.ID).Scan(&issuerMembership)
		if queryErr != nil && !errors.Is(queryErr, sql.ErrNoRows) {
			return "", "", nil, queryErr
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO principals(id,workspace_id,display_name,kind,created_at,portable_actor_id,created_by_membership_id,origin_replica_id) VALUES(?,?,?,?,?,?,?,?)`, p.ID, p.WorkspaceID, p.DisplayName, p.Kind, ts(t), newID("actor"), nullable(issuerMembership), s.LocalReplicaID()); err != nil {
			return "", "", nil, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO project_principals(project_id,principal_id,role,created_at) VALUES(?,?,?,?)`, project.ID, p.ID, "agent", ts(t)); err != nil {
			return "", "", nil, err
		}
		prefix := token
		if len(prefix) > 12 {
			prefix = prefix[:12]
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO api_tokens(id,principal_id,token_hash,token_prefix,scopes_json,created_at,issued_by_membership_id) VALUES(?,?,?,?,?,?,?)`, newID("token"), p.ID, hashToken(token), prefix, `["project:agent"]`, ts(t), nullable(issuerMembership)); err != nil {
			return "", "", nil, err
		}
		credential := domain.ProjectCredential{Principal: p, Token: token, Notice: "Copy this project agent token now. It is stored only as a hash and cannot be shown again."}
		return p.ID, "project.token.issued", credential, nil
	})
	if err != nil {
		return domain.ProjectCredential{}, receipt, err
	}
	var credential domain.ProjectCredential
	err = json.Unmarshal(body, &credential)
	return credential, receipt, err
}

func (s *Store) GetProject(ctx context.Context, workspaceID, projectID string) (domain.Project, error) {
	var p domain.Project
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT id,workspace_id,slug,name,description,created_at FROM projects WHERE workspace_id=? AND (id=? OR slug=?)`, workspaceID, projectID, projectID).Scan(&p.ID, &p.WorkspaceID, &p.Slug, &p.Name, &p.Description, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	p.CreatedAt = parseTime(created)
	return p, nil
}

type mutationFunc func(*sql.Tx) (entityID, eventType string, response any, err error)

func (s *Store) mutate(ctx context.Context, workspaceID, projectID, principalID, actorID, runID, key string, request any, fn mutationFunc) ([]byte, domain.Receipt, error) {
	if strings.TrimSpace(key) == "" {
		return nil, domain.Receipt{}, errors.New("idempotency key is required")
	}
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, domain.Receipt{}, err
	}
	sum := sha256.Sum256(reqJSON)
	requestHash := hex.EncodeToString(sum[:])
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, domain.Receipt{}, err
	}
	defer tx.Rollback()
	var existingHash, responseJSON, rid, eventID, created string
	var sequence int64
	err = tx.QueryRowContext(ctx, `SELECT request_hash,response_json,id,event_id,sequence,created_at FROM receipts WHERE actor_id=? AND idempotency_key=?`, actorID, key).Scan(&existingHash, &responseJSON, &rid, &eventID, &sequence, &created)
	if err == nil {
		if existingHash != requestHash {
			return nil, domain.Receipt{}, ErrIdempotencyKeyReuse
		}
		originReplicaID, originSequence := s.domainEventInfoTx(ctx, tx, eventID)
		return []byte(responseJSON), domain.Receipt{ID: rid, IdempotencyKey: key, EventID: eventID, Sequence: sequence, Response: []byte(responseJSON), CreatedAt: parseTime(created), OriginReplicaID: originReplicaID, OriginSequence: originSequence}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, domain.Receipt{}, err
	}
	entityID, eventType, response, err := fn(tx)
	if err != nil {
		return nil, domain.Receipt{}, err
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return nil, domain.Receipt{}, err
	}
	eventID = newID("evt")
	t := now()
	result, err := tx.ExecContext(ctx, `INSERT INTO events(id,workspace_id,project_id,principal_id,actor_id,run_id,type,entity_id,payload_json,occurred_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, eventID, workspaceID, nullable(projectID), principalID, actorID, nullable(runID), eventType, entityID, string(reqJSON), ts(t))
	if err != nil {
		return nil, domain.Receipt{}, err
	}
	sequence, err = result.LastInsertId()
	if err != nil {
		return nil, domain.Receipt{}, err
	}
	originReplicaID, originSequence, err := s.appendDomainEventTx(ctx, tx, eventID, workspaceID, projectID, principalID, runID, eventType, entityID, responseBytes, t)
	if err != nil {
		return nil, domain.Receipt{}, err
	}
	rid = newID("receipt")
	if _, err = tx.ExecContext(ctx, `INSERT INTO receipts(id,actor_id,idempotency_key,request_hash,event_id,sequence,response_json,created_at) VALUES(?,?,?,?,?,?,?,?)`, rid, actorID, key, requestHash, eventID, sequence, string(responseBytes), ts(t)); err != nil {
		return nil, domain.Receipt{}, err
	}
	if err = tx.Commit(); err != nil {
		return nil, domain.Receipt{}, err
	}
	return responseBytes, domain.Receipt{ID: rid, IdempotencyKey: key, EventID: eventID, Sequence: sequence, Response: responseBytes, CreatedAt: t, OriginReplicaID: originReplicaID, OriginSequence: originSequence}, nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *Store) CreateProject(ctx context.Context, principal domain.Principal, key, slug, name, description string) (domain.Project, domain.Receipt, error) {
	req := map[string]any{"slug": slug, "name": name, "description": description}
	b, receipt, err := s.mutate(ctx, principal.WorkspaceID, "", principal.ID, principal.ID, "", key, req, func(tx *sql.Tx) (string, string, any, error) {
		t := now()
		p := domain.Project{ID: newID("project"), WorkspaceID: principal.WorkspaceID, Slug: slug, Name: name, Description: description, CreatedAt: t}
		_, err := tx.ExecContext(ctx, `INSERT INTO projects(id,workspace_id,slug,name,description,created_at) VALUES(?,?,?,?,?,?)`, p.ID, p.WorkspaceID, p.Slug, p.Name, p.Description, ts(t))
		return p.ID, "project.created", p, err
	})
	if err != nil {
		return domain.Project{}, receipt, err
	}
	var p domain.Project
	err = json.Unmarshal(b, &p)
	return p, receipt, err
}

func (s *Store) StartRun(ctx context.Context, principal domain.Principal, key string, in domain.StartRunInput) (domain.Run, domain.Receipt, error) {
	b, receipt, err := s.mutate(ctx, principal.WorkspaceID, in.ProjectID, principal.ID, principal.ID, "", key, in, func(tx *sql.Tx) (string, string, any, error) {
		var projectCount int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE id=? AND workspace_id=?`, in.ProjectID, principal.WorkspaceID).Scan(&projectCount); err != nil || projectCount == 0 {
			if err == nil {
				err = ErrNotFound
			}
			return "", "", nil, err
		}
		if len(in.Branch) > 255 || len(in.Worktree) > 4096 {
			return "", "", nil, errors.New("run branch or worktree is too long")
		}
		for label, sha := range map[string]string{"base SHA": in.BaseSHA, "head SHA": in.HeadSHA} {
			if sha != "" && !fullGitSHA.MatchString(sha) {
				return "", "", nil, fmt.Errorf("%s must be a full 40- or 64-character hexadecimal object ID", label)
			}
		}
		if in.RepositoryID != "" {
			var repositoryCount int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM project_repositories WHERE project_id=? AND repository_id=?`, in.ProjectID, in.RepositoryID).Scan(&repositoryCount); err != nil || repositoryCount != 1 {
				if err == nil {
					err = errors.New("repository is not attached to this project")
				}
				return "", "", nil, err
			}
		}
		agentName := strings.TrimSpace(in.AgentName)
		if agentName == "" {
			agentName = "agent"
		}
		var agentID string
		err := tx.QueryRowContext(ctx, `SELECT id FROM agents WHERE principal_id=? AND name=?`, principal.ID, agentName).Scan(&agentID)
		if errors.Is(err, sql.ErrNoRows) {
			agentID = newID("agent")
			_, err = tx.ExecContext(ctx, `INSERT INTO agents(id,principal_id,name,created_at) VALUES(?,?,?,?)`, agentID, principal.ID, agentName, ts(now()))
		}
		if err != nil {
			return "", "", nil, err
		}
		t := now()
		role := in.Role
		if role == "" {
			role = "primary"
		}
		runType := in.RunType
		if runType == "" {
			runType = "interactive"
		}
		profile, _ := json.Marshal(in.InstructionProfile)
		r := domain.Run{ID: newID("run"), ProjectID: in.ProjectID, AgentID: agentID, AgentName: agentName, PrincipalID: principal.ID, PrincipalName: principal.DisplayName, Harness: in.Harness, HarnessVersion: in.HarnessVersion, Provider: in.Provider, Model: in.Model, Reasoning: in.Reasoning, Role: role, ParentRunID: in.ParentRunID, RootRunID: in.RootRunID, RunType: runType, PermissionMode: in.PermissionMode, InteractionMode: in.InteractionMode, RepositoryID: in.RepositoryID, Branch: in.Branch, Worktree: in.Worktree, BaseSHA: in.BaseSHA, HeadSHA: in.HeadSHA, Objective: in.Objective, InstructionProfile: in.InstructionProfile, StartedAt: t}
		_, err = tx.ExecContext(ctx, `INSERT INTO runs(id,project_id,agent_id,principal_id,harness,harness_version,provider,model,reasoning,role,parent_run_id,root_run_id,run_type,permission_mode,interaction_mode,repository_id,branch,worktree,base_sha,head_sha,objective,instruction_profile_json,started_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, r.ID, r.ProjectID, r.AgentID, r.PrincipalID, r.Harness, r.HarnessVersion, r.Provider, r.Model, r.Reasoning, r.Role, nullable(r.ParentRunID), nullable(r.RootRunID), r.RunType, r.PermissionMode, r.InteractionMode, nullable(r.RepositoryID), r.Branch, r.Worktree, r.BaseSHA, r.HeadSHA, r.Objective, string(profile), ts(t))
		return r.ID, "run.started", r, err
	})
	if err != nil {
		return domain.Run{}, receipt, err
	}
	var r domain.Run
	err = json.Unmarshal(b, &r)
	return r, receipt, err
}

func (s *Store) EndRun(ctx context.Context, principal domain.Principal, key, runID string, in domain.EndRunInput) (domain.Run, domain.Receipt, error) {
	b, receipt, err := s.mutate(ctx, principal.WorkspaceID, "", principal.ID, principal.ID, runID, key, in, func(tx *sql.Tx) (string, string, any, error) {
		if err := validateRunDeliveryTx(ctx, tx, runID, principal.ID, domain.LinkRunDeliveryInput{DeliveryBranch: in.DeliveryBranch, HeadSHA: in.HeadSHA, PullRequestURL: in.PullRequestURL, PullRequestNumber: in.PullRequestNumber, PullRequestState: in.PullRequestState, MergeCommitSHA: in.MergeCommitSHA, MergedAt: in.MergedAt}); err != nil {
			return "", "", nil, err
		}
		t := now()
		result, err := tx.ExecContext(ctx, `UPDATE runs SET ended_at=?,outcome=?,verification=?,delivery_branch=CASE WHEN ?='' THEN delivery_branch ELSE ? END,head_sha=CASE WHEN ?='' THEN head_sha ELSE ? END,pull_request_url=CASE WHEN ?='' THEN pull_request_url ELSE ? END,pull_request_number=CASE WHEN ?=0 THEN pull_request_number ELSE ? END,pull_request_state=CASE WHEN ?='' THEN pull_request_state ELSE ? END,merge_commit_sha=CASE WHEN ?='' THEN merge_commit_sha ELSE ? END,merged_at=COALESCE(?,merged_at) WHERE id=? AND principal_id=? AND ended_at IS NULL`, ts(t), in.Outcome, in.Verification, in.DeliveryBranch, in.DeliveryBranch, in.HeadSHA, in.HeadSHA, in.PullRequestURL, in.PullRequestURL, in.PullRequestNumber, in.PullRequestNumber, in.PullRequestState, in.PullRequestState, in.MergeCommitSHA, in.MergeCommitSHA, nullableTime(in.MergedAt), runID, principal.ID)
		if err != nil {
			return "", "", nil, err
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			return "", "", nil, ErrNotFound
		}
		if _, err = tx.ExecContext(ctx, `UPDATE trajectories SET status='closed',updated_at=? WHERE run_id=? AND status='active'`, ts(t), runID); err != nil {
			return "", "", nil, err
		}
		r, err := scanRun(tx.QueryRowContext(ctx, runSelect+` WHERE r.id=?`, runID))
		return runID, "run.ended", r, err
	})
	if err != nil {
		return domain.Run{}, receipt, err
	}
	var r domain.Run
	err = json.Unmarshal(b, &r)
	if err == nil && (in.DeliveryBranch != "" || in.HeadSHA != "" || in.PullRequestURL != "" || in.PullRequestNumber != 0 || in.PullRequestState != "" || in.MergeCommitSHA != "" || in.MergedAt != nil) {
		linked, _, linkErr := s.LinkRunDelivery(ctx, principal, key+":delivery", runID, domain.LinkRunDeliveryInput{DeliveryBranch: in.DeliveryBranch, HeadSHA: in.HeadSHA, PullRequestURL: in.PullRequestURL, PullRequestNumber: in.PullRequestNumber, PullRequestState: in.PullRequestState, MergeCommitSHA: in.MergeCommitSHA, MergedAt: in.MergedAt})
		if linkErr != nil {
			return domain.Run{}, receipt, linkErr
		}
		r = linked
	}
	return r, receipt, err
}

func (s *Store) LinkRunDelivery(ctx context.Context, principal domain.Principal, key, runID string, in domain.LinkRunDeliveryInput) (domain.Run, domain.Receipt, error) {
	b, receipt, err := s.mutate(ctx, principal.WorkspaceID, "", principal.ID, principal.ID, runID, key, in, func(tx *sql.Tx) (string, string, any, error) {
		if err := validateRunDeliveryTx(ctx, tx, runID, principal.ID, in); err != nil {
			return "", "", nil, err
		}
		result, err := tx.ExecContext(ctx, `UPDATE runs SET delivery_branch=CASE WHEN ?='' THEN delivery_branch ELSE ? END,head_sha=CASE WHEN ?='' THEN head_sha ELSE ? END,pull_request_url=CASE WHEN ?='' THEN pull_request_url ELSE ? END,pull_request_number=CASE WHEN ?=0 THEN pull_request_number ELSE ? END,pull_request_state=CASE WHEN ?='' THEN pull_request_state ELSE ? END,merge_commit_sha=CASE WHEN ?='' THEN merge_commit_sha ELSE ? END,merged_at=COALESCE(?,merged_at) WHERE id=? AND principal_id=?`, in.DeliveryBranch, in.DeliveryBranch, in.HeadSHA, in.HeadSHA, in.PullRequestURL, in.PullRequestURL, in.PullRequestNumber, in.PullRequestNumber, in.PullRequestState, in.PullRequestState, in.MergeCommitSHA, in.MergeCommitSHA, nullableTime(in.MergedAt), runID, principal.ID)
		if err != nil {
			return "", "", nil, err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return "", "", nil, ErrNotFound
		}
		r, err := scanRun(tx.QueryRowContext(ctx, runSelect+` WHERE r.id=?`, runID))
		return runID, "run.delivery_linked", r, err
	})
	if err != nil {
		return domain.Run{}, receipt, err
	}
	var r domain.Run
	err = json.Unmarshal(b, &r)
	return r, receipt, err
}

func validateRunDeliveryTx(ctx context.Context, tx *sql.Tx, runID, principalID string, in domain.LinkRunDeliveryInput) error {
	if len(in.DeliveryBranch) > 255 || len(in.PullRequestURL) > 2048 {
		return errors.New("delivery branch or pull request URL is too long")
	}
	for label, sha := range map[string]string{"head SHA": in.HeadSHA, "merge commit SHA": in.MergeCommitSHA} {
		if sha != "" && !fullGitSHA.MatchString(sha) {
			return fmt.Errorf("%s must be a full 40- or 64-character hexadecimal object ID", label)
		}
	}
	if in.PullRequestNumber < 0 {
		return errors.New("pull request number must be positive")
	}
	if in.PullRequestState != "" && in.PullRequestState != "open" && in.PullRequestState != "closed" && in.PullRequestState != "merged" {
		return errors.New("pull request state must be open, closed, or merged")
	}
	var repositoryID, host, owner, name, currentURL, currentState string
	var currentNumber int
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(r.repository_id,''),COALESCE(repo.host,''),COALESCE(repo.owner,''),COALESCE(repo.name,''),r.pull_request_url,r.pull_request_number,r.pull_request_state FROM runs r LEFT JOIN repositories repo ON repo.id=r.repository_id WHERE r.id=? AND r.principal_id=?`, runID, principalID).Scan(&repositoryID, &host, &owner, &name, &currentURL, &currentNumber, &currentState)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	pullURL := in.PullRequestURL
	if pullURL == "" {
		pullURL = currentURL
	}
	pullNumber := in.PullRequestNumber
	if pullNumber == 0 {
		pullNumber = currentNumber
	}
	state := in.PullRequestState
	if state == "" {
		state = currentState
	}
	if pullURL != "" {
		if repositoryID == "" {
			return errors.New("a pull request cannot be linked until the run has an attached repository")
		}
		parsed, parseErr := url.Parse(pullURL)
		if parseErr != nil || parsed == nil {
			return errors.New("pull request URL is invalid")
		}
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), host) || len(parts) != 4 || !strings.EqualFold(parts[0], owner) || !strings.EqualFold(parts[1], name) || parts[2] != "pull" {
			return errors.New("pull request URL must belong to the run repository")
		}
		number, numberErr := strconv.Atoi(parts[3])
		if numberErr != nil || number <= 0 || (pullNumber != 0 && pullNumber != number) {
			return errors.New("pull request URL and number do not match")
		}
	}
	if (in.MergeCommitSHA != "" || in.MergedAt != nil) && state != "merged" {
		return errors.New("merge evidence requires pull request state merged")
	}
	return nil
}

const runSelect = `SELECT r.id,r.project_id,r.agent_id,a.name,r.principal_id,p.display_name,r.harness,r.harness_version,r.provider,r.model,r.reasoning,r.role,COALESCE(r.parent_run_id,''),COALESCE(r.root_run_id,''),r.run_type,r.permission_mode,r.interaction_mode,COALESCE(r.repository_id,''),r.branch,r.worktree,r.base_sha,r.head_sha,r.delivery_branch,r.pull_request_url,r.pull_request_number,r.pull_request_state,r.merge_commit_sha,COALESCE(r.merged_at,''),r.objective,r.instruction_profile_json,r.started_at,COALESCE(r.ended_at,''),r.outcome,r.verification FROM runs r JOIN agents a ON a.id=r.agent_id JOIN principals p ON p.id=r.principal_id`

func scanRun(row *sql.Row) (domain.Run, error) {
	var r domain.Run
	var profile, started, ended, merged string
	err := row.Scan(&r.ID, &r.ProjectID, &r.AgentID, &r.AgentName, &r.PrincipalID, &r.PrincipalName, &r.Harness, &r.HarnessVersion, &r.Provider, &r.Model, &r.Reasoning, &r.Role, &r.ParentRunID, &r.RootRunID, &r.RunType, &r.PermissionMode, &r.InteractionMode, &r.RepositoryID, &r.Branch, &r.Worktree, &r.BaseSHA, &r.HeadSHA, &r.DeliveryBranch, &r.PullRequestURL, &r.PullRequestNumber, &r.PullRequestState, &r.MergeCommitSHA, &merged, &r.Objective, &profile, &started, &ended, &r.Outcome, &r.Verification)
	if err != nil {
		return r, err
	}
	_ = json.Unmarshal([]byte(profile), &r.InstructionProfile)
	r.StartedAt = parseTime(started)
	if ended != "" {
		t := parseTime(ended)
		r.EndedAt = &t
	}
	if merged != "" {
		t := parseTime(merged)
		r.MergedAt = &t
	}
	return r, nil
}

func (s *Store) GetRun(ctx context.Context, runID string) (domain.Run, error) {
	r, err := scanRun(s.db.QueryRowContext(ctx, runSelect+` WHERE r.id=?`, runID))
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

func (s *Store) ListRuns(ctx context.Context, projectID string, limit int) ([]domain.Run, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, runSelect+` WHERE r.project_id=? ORDER BY r.started_at DESC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := make([]domain.Run, 0)
	for rows.Next() {
		var run domain.Run
		var profile, started, ended, merged string
		if err = rows.Scan(&run.ID, &run.ProjectID, &run.AgentID, &run.AgentName, &run.PrincipalID, &run.PrincipalName, &run.Harness, &run.HarnessVersion, &run.Provider, &run.Model, &run.Reasoning, &run.Role, &run.ParentRunID, &run.RootRunID, &run.RunType, &run.PermissionMode, &run.InteractionMode, &run.RepositoryID, &run.Branch, &run.Worktree, &run.BaseSHA, &run.HeadSHA, &run.DeliveryBranch, &run.PullRequestURL, &run.PullRequestNumber, &run.PullRequestState, &run.MergeCommitSHA, &merged, &run.Objective, &profile, &started, &ended, &run.Outcome, &run.Verification); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(profile), &run.InstructionProfile)
		run.StartedAt = parseTime(started)
		if ended != "" {
			t := parseTime(ended)
			run.EndedAt = &t
		}
		if merged != "" {
			t := parseTime(merged)
			run.MergedAt = &t
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

// ListAllRuns is reserved for portable snapshots and exports. Interactive API
// callers should use ListRuns so a single request cannot accidentally return an
// unbounded history.
func (s *Store) ListAllRuns(ctx context.Context, projectID string) ([]domain.Run, error) {
	rows, err := s.db.QueryContext(ctx, runSelect+` WHERE r.project_id=? ORDER BY r.started_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := make([]domain.Run, 0)
	for rows.Next() {
		var run domain.Run
		var profile, started, ended, merged string
		if err = rows.Scan(&run.ID, &run.ProjectID, &run.AgentID, &run.AgentName, &run.PrincipalID, &run.PrincipalName, &run.Harness, &run.HarnessVersion, &run.Provider, &run.Model, &run.Reasoning, &run.Role, &run.ParentRunID, &run.RootRunID, &run.RunType, &run.PermissionMode, &run.InteractionMode, &run.RepositoryID, &run.Branch, &run.Worktree, &run.BaseSHA, &run.HeadSHA, &run.DeliveryBranch, &run.PullRequestURL, &run.PullRequestNumber, &run.PullRequestState, &run.MergeCommitSHA, &merged, &run.Objective, &profile, &started, &ended, &run.Outcome, &run.Verification); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(profile), &run.InstructionProfile)
		run.StartedAt = parseTime(started)
		if ended != "" {
			t := parseTime(ended)
			run.EndedAt = &t
		}
		if merged != "" {
			t := parseTime(merged)
			run.MergedAt = &t
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (s *Store) CreateNote(ctx context.Context, principal domain.Principal, projectID, key string, in domain.CreateNoteInput) (domain.Note, domain.Receipt, error) {
	b, receipt, err := s.mutate(ctx, principal.WorkspaceID, projectID, principal.ID, principal.ID, in.RunID, key, in, func(tx *sql.Tx) (string, string, any, error) {
		t := now()
		paths, _ := json.Marshal(in.Paths)
		n := domain.Note{ID: newID("note"), ProjectID: projectID, RunID: in.RunID, PrincipalID: principal.ID, Kind: in.Kind, Title: in.Title, Summary: in.Summary, Rationale: in.Rationale, Status: "current", LedBy: in.LedBy, DirectionBasis: in.DirectionBasis, Confidence: in.Confidence, Verification: in.Verification, SourceRef: in.SourceRef, Paths: in.Paths, RepositoryID: in.RepositoryID, PullRequestURL: in.PullRequestURL, Revision: 1, CreatedAt: t, UpdatedAt: t}
		_, err := tx.ExecContext(ctx, `INSERT INTO notes(id,project_id,run_id,principal_id,kind,title,summary,rationale,status,led_by,direction_basis,confidence,verification,source_ref,paths_json,repository_id,pull_request_url,revision,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, n.ID, n.ProjectID, nullable(n.RunID), n.PrincipalID, n.Kind, n.Title, n.Summary, n.Rationale, n.Status, n.LedBy, n.DirectionBasis, n.Confidence, n.Verification, n.SourceRef, string(paths), nullable(n.RepositoryID), n.PullRequestURL, n.Revision, ts(t), ts(t))
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO notes_fts(note_id,project_id,title,summary,rationale) VALUES(?,?,?,?,?)`, n.ID, n.ProjectID, n.Title, n.Summary, n.Rationale)
		}
		return n.ID, "note.recorded", n, err
	})
	if err != nil {
		return domain.Note{}, receipt, err
	}
	var n domain.Note
	err = json.Unmarshal(b, &n)
	return n, receipt, err
}

func (s *Store) SupersedeNote(ctx context.Context, principal domain.Principal, projectID, key, noteID string, in domain.SupersedeNoteInput) (domain.Note, domain.Receipt, error) {
	b, receipt, err := s.mutate(ctx, principal.WorkspaceID, projectID, principal.ID, principal.ID, in.RunID, key, in, func(tx *sql.Tx) (string, string, any, error) {
		current, err := scanNote(tx.QueryRowContext(ctx, noteSelect+` WHERE n.id=? AND n.project_id=?`, noteID, projectID))
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", nil, ErrNotFound
		}
		if err != nil {
			return "", "", nil, err
		}
		if current.Revision != in.ExpectedRevision || current.Status != "current" {
			return "", "", nil, ErrConflict
		}
		replacementID := ""
		var replacement domain.Note
		if in.Replacement != nil {
			r := *in.Replacement
			if r.RunID == "" {
				r.RunID = in.RunID
			}
			t := now()
			paths, _ := json.Marshal(r.Paths)
			replacement = domain.Note{ID: newID("note"), ProjectID: projectID, RunID: r.RunID, PrincipalID: principal.ID, Kind: r.Kind, Title: r.Title, Summary: r.Summary, Rationale: r.Rationale, Status: "current", LedBy: r.LedBy, DirectionBasis: r.DirectionBasis, Confidence: r.Confidence, Verification: r.Verification, SourceRef: r.SourceRef, Paths: r.Paths, RepositoryID: r.RepositoryID, PullRequestURL: r.PullRequestURL, Revision: 1, CreatedAt: t, UpdatedAt: t}
			replacementID = replacement.ID
			_, err = tx.ExecContext(ctx, `INSERT INTO notes(id,project_id,run_id,principal_id,kind,title,summary,rationale,status,led_by,direction_basis,confidence,verification,source_ref,paths_json,repository_id,pull_request_url,revision,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, replacement.ID, projectID, nullable(replacement.RunID), principal.ID, replacement.Kind, replacement.Title, replacement.Summary, replacement.Rationale, "current", replacement.LedBy, replacement.DirectionBasis, replacement.Confidence, replacement.Verification, replacement.SourceRef, string(paths), nullable(replacement.RepositoryID), replacement.PullRequestURL, 1, ts(t), ts(t))
			if err != nil {
				return "", "", nil, err
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO notes_fts(note_id,project_id,title,summary,rationale) VALUES(?,?,?,?,?)`, replacement.ID, projectID, replacement.Title, replacement.Summary, replacement.Rationale)
			if err != nil {
				return "", "", nil, err
			}
		}
		t := now()
		_, err = tx.ExecContext(ctx, `UPDATE notes SET status='superseded',superseded_by=?,revision=revision+1,updated_at=? WHERE id=?`, nullable(replacementID), ts(t), noteID)
		if err != nil {
			return "", "", nil, err
		}
		current.Status = "superseded"
		current.SupersededBy = replacementID
		current.Revision++
		current.UpdatedAt = t
		return noteID, "note.superseded", map[string]any{"superseded": current, "replacement": replacement, "reason": in.Reason}, nil
	})
	if err != nil {
		return domain.Note{}, receipt, err
	}
	var result struct {
		Superseded  domain.Note `json:"superseded"`
		Replacement domain.Note `json:"replacement"`
	}
	err = json.Unmarshal(b, &result)
	if result.Replacement.ID != "" {
		return result.Replacement, receipt, err
	}
	return result.Superseded, receipt, err
}

const noteSelect = `SELECT n.id,n.project_id,COALESCE(n.run_id,''),n.principal_id,n.kind,n.title,n.summary,n.rationale,n.status,n.led_by,n.direction_basis,n.confidence,n.verification,n.source_ref,n.paths_json,COALESCE(n.repository_id,''),n.pull_request_url,n.revision,COALESCE(n.superseded_by,''),n.created_at,n.updated_at FROM notes n`

func scanNote(row *sql.Row) (domain.Note, error) {
	var n domain.Note
	var paths, created, updated string
	err := row.Scan(&n.ID, &n.ProjectID, &n.RunID, &n.PrincipalID, &n.Kind, &n.Title, &n.Summary, &n.Rationale, &n.Status, &n.LedBy, &n.DirectionBasis, &n.Confidence, &n.Verification, &n.SourceRef, &paths, &n.RepositoryID, &n.PullRequestURL, &n.Revision, &n.SupersededBy, &created, &updated)
	if err != nil {
		return n, err
	}
	_ = json.Unmarshal([]byte(paths), &n.Paths)
	n.CreatedAt = parseTime(created)
	n.UpdatedAt = parseTime(updated)
	return n, nil
}

func (s *Store) ListNotes(ctx context.Context, projectID string, limit int) ([]domain.Note, error) {
	query := noteSelect + ` WHERE n.project_id=? ORDER BY n.updated_at DESC`
	args := []any{projectID}
	if limit >= 0 {
		if limit == 0 || limit > 100 {
			limit = 40
		}
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Note
	for rows.Next() {
		var n domain.Note
		var paths, created, updated string
		if err := rows.Scan(&n.ID, &n.ProjectID, &n.RunID, &n.PrincipalID, &n.Kind, &n.Title, &n.Summary, &n.Rationale, &n.Status, &n.LedBy, &n.DirectionBasis, &n.Confidence, &n.Verification, &n.SourceRef, &paths, &n.RepositoryID, &n.PullRequestURL, &n.Revision, &n.SupersededBy, &created, &updated); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(paths), &n.Paths)
		n.CreatedAt = parseTime(created)
		n.UpdatedAt = parseTime(updated)
		if n.RunID != "" {
			if r, e := s.GetRun(ctx, n.RunID); e == nil {
				n.Run = &r
			}
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) SearchNotes(ctx context.Context, projectID, query string, limit int) ([]domain.Note, error) {
	if strings.TrimSpace(query) == "" {
		return s.ListNotes(ctx, projectID, limit)
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT n.id FROM notes_fts f JOIN notes n ON n.id=f.note_id WHERE f.project_id=? AND notes_fts MATCH ? ORDER BY rank LIMIT ?`, projectID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	var out []domain.Note
	for _, id := range ids {
		n, err := scanNote(s.db.QueryRowContext(ctx, noteSelect+` WHERE n.id=?`, id))
		if err != nil {
			return nil, err
		}
		if n.RunID != "" {
			if r, e := s.GetRun(ctx, n.RunID); e == nil {
				n.Run = &r
			}
		}
		out = append(out, n)
	}
	return out, nil
}

func (s *Store) CreateTrajectory(ctx context.Context, principal domain.Principal, projectID, key string, in domain.CreateTrajectoryInput) (domain.Trajectory, domain.Receipt, error) {
	b, receipt, err := s.mutate(ctx, principal.WorkspaceID, projectID, principal.ID, principal.ID, in.RunID, key, in, func(tx *sql.Tx) (string, string, any, error) {
		t := now()
		paths, _ := json.Marshal(in.Paths)
		tr := domain.Trajectory{ID: newID("trajectory"), ProjectID: projectID, RunID: in.RunID, PrincipalID: principal.ID, Objective: in.Objective, Rationale: in.Rationale, Status: "active", Paths: in.Paths, RepositoryID: in.RepositoryID, Branch: in.Branch, BaseSHA: in.BaseSHA, HeadSHA: in.HeadSHA, CreatedAt: t, UpdatedAt: t}
		_, err := tx.ExecContext(ctx, `INSERT INTO trajectories(id,project_id,run_id,principal_id,objective,rationale,status,paths_json,repository_id,branch,base_sha,head_sha,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, tr.ID, projectID, tr.RunID, principal.ID, tr.Objective, tr.Rationale, tr.Status, string(paths), nullable(tr.RepositoryID), tr.Branch, tr.BaseSHA, tr.HeadSHA, ts(t), ts(t))
		return tr.ID, "trajectory.started", tr, err
	})
	if err != nil {
		return domain.Trajectory{}, receipt, err
	}
	var tr domain.Trajectory
	err = json.Unmarshal(b, &tr)
	return tr, receipt, err
}

func (s *Store) ListTrajectories(ctx context.Context, projectID string, activeOnly bool) ([]domain.Trajectory, error) {
	q := `SELECT id,project_id,run_id,principal_id,objective,rationale,status,paths_json,COALESCE(repository_id,''),branch,base_sha,head_sha,created_at,updated_at FROM trajectories WHERE project_id=?`
	if activeOnly {
		q += ` AND status='active'`
	}
	q += ` ORDER BY updated_at DESC`
	rows, err := s.db.QueryContext(ctx, q, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Trajectory
	for rows.Next() {
		var tr domain.Trajectory
		var paths, created, updated string
		if err := rows.Scan(&tr.ID, &tr.ProjectID, &tr.RunID, &tr.PrincipalID, &tr.Objective, &tr.Rationale, &tr.Status, &paths, &tr.RepositoryID, &tr.Branch, &tr.BaseSHA, &tr.HeadSHA, &created, &updated); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(paths), &tr.Paths)
		tr.CreatedAt = parseTime(created)
		tr.UpdatedAt = parseTime(updated)
		if r, e := s.GetRun(ctx, tr.RunID); e == nil {
			tr.Run = &r
		}
		out = append(out, tr)
	}
	return out, rows.Err()
}

func (s *Store) UpsertRepository(ctx context.Context, principal domain.Principal, projectID, key string, repo domain.Repository) (domain.Repository, domain.Receipt, error) {
	b, receipt, err := s.mutate(ctx, principal.WorkspaceID, projectID, principal.ID, principal.ID, "", key, repo, func(tx *sql.Tx) (string, string, any, error) {
		repo.ID = newID("repo")
		repo.WorkspaceID = principal.WorkspaceID
		_, err := tx.ExecContext(ctx, `INSERT INTO repositories(id,workspace_id,url,host,owner,name,visibility,description,default_branch,stars) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(workspace_id,host,owner,name) DO UPDATE SET url=excluded.url`, repo.ID, repo.WorkspaceID, repo.URL, repo.Host, repo.Owner, repo.Name, repo.Visibility, repo.Description, repo.Default, repo.Stars)
		if err != nil {
			return "", "", nil, err
		}
		var actualID string
		err = tx.QueryRowContext(ctx, `SELECT id FROM repositories WHERE workspace_id=? AND host=? AND owner=? AND name=?`, repo.WorkspaceID, repo.Host, repo.Owner, repo.Name).Scan(&actualID)
		if err != nil {
			return "", "", nil, err
		}
		repo.ID = actualID
		_, err = tx.ExecContext(ctx, `INSERT INTO project_repositories(project_id,repository_id,is_primary) VALUES(?,?,1) ON CONFLICT(project_id,repository_id) DO NOTHING`, projectID, actualID)
		return actualID, "repository.attached", repo, err
	})
	if err != nil {
		return domain.Repository{}, receipt, err
	}
	var out domain.Repository
	err = json.Unmarshal(b, &out)
	return out, receipt, err
}

func (s *Store) UpdateRepositorySync(ctx context.Context, repo domain.Repository, etag string, pulls []domain.ExternalArtifact, syncErr string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	t := now()
	_, err = tx.ExecContext(ctx, `UPDATE repositories SET description=?,default_branch=?,stars=?,etag=?,synced_at=?,sync_error=? WHERE id=?`, repo.Description, repo.Default, repo.Stars, etag, ts(t), syncErr, repo.ID)
	if err != nil {
		return err
	}
	for _, p := range pulls {
		_, err = tx.ExecContext(ctx, `INSERT INTO external_artifacts(id,repository_id,kind,external_id,title,url,state,author,updated_at) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(repository_id,kind,external_id) DO UPDATE SET title=excluded.title,url=excluded.url,state=excluded.state,author=excluded.author,updated_at=excluded.updated_at`, p.ID, repo.ID, p.Kind, p.ExternalID, p.Title, p.URL, p.State, p.Author, ts(p.UpdatedAt))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListRepositories(ctx context.Context, projectID string) ([]domain.Repository, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT r.id,r.workspace_id,r.url,r.host,r.owner,r.name,r.visibility,r.description,r.default_branch,r.stars,COALESCE(r.synced_at,''),r.sync_error FROM repositories r JOIN project_repositories pr ON pr.repository_id=r.id WHERE pr.project_id=? ORDER BY r.owner,r.name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Repository
	for rows.Next() {
		var r domain.Repository
		var synced string
		if err := rows.Scan(&r.ID, &r.WorkspaceID, &r.URL, &r.Host, &r.Owner, &r.Name, &r.Visibility, &r.Description, &r.Default, &r.Stars, &synced, &r.SyncError); err != nil {
			return nil, err
		}
		if synced != "" {
			t := parseTime(synced)
			r.SyncedAt = &t
		}
		p, e := s.ListArtifacts(ctx, r.ID)
		if e != nil {
			return nil, e
		}
		r.Pulls = p
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetRepository(ctx context.Context, repoID string) (domain.Repository, error) {
	var r domain.Repository
	var synced string
	err := s.db.QueryRowContext(ctx, `SELECT id,workspace_id,url,host,owner,name,visibility,description,default_branch,stars,COALESCE(synced_at,''),sync_error FROM repositories WHERE id=?`, repoID).Scan(&r.ID, &r.WorkspaceID, &r.URL, &r.Host, &r.Owner, &r.Name, &r.Visibility, &r.Description, &r.Default, &r.Stars, &synced, &r.SyncError)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if synced != "" {
		t := parseTime(synced)
		r.SyncedAt = &t
	}
	return r, err
}

func (s *Store) ListArtifacts(ctx context.Context, repoID string) ([]domain.ExternalArtifact, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,repository_id,kind,external_id,title,url,state,author,updated_at FROM external_artifacts WHERE repository_id=? ORDER BY updated_at DESC`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ExternalArtifact
	for rows.Next() {
		var a domain.ExternalArtifact
		var updated string
		if err := rows.Scan(&a.ID, &a.RepositoryID, &a.Kind, &a.ExternalID, &a.Title, &a.URL, &a.State, &a.Author, &updated); err != nil {
			return nil, err
		}
		a.UpdatedAt = parseTime(updated)
		out = append(out, a)
	}
	return out, rows.Err()
}
