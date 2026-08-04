package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/domain"
)

var workspaceSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var workspaceSlugInvalid = regexp.MustCompile(`[^a-z0-9]+`)

func slugForWorkspace(name, id string) string {
	slug := strings.Trim(workspaceSlugInvalid.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-"), "-")
	if slug == "" {
		slug = "workspace"
	}
	if len(id) >= 6 {
		slug += "-" + id[len(id)-6:]
	}
	return slug
}

type SessionResult struct {
	User         domain.User
	Session      domain.BrowserSession
	SessionToken string
	CSRFToken    string
}

type OutboxMessage struct {
	ID        string
	Recipient string
	Template  string
	Subject   string
	Body      string
}

type emailPayload struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func randomToken(prefix string) (string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(random), nil
}

func normalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(value)
	if err != nil || strings.ToLower(parsed.Address) != value || !strings.Contains(value, "@") {
		return "", errors.New("valid email address is required")
	}
	return value, nil
}

func displayNameForEmail(email string) string {
	name := strings.TrimSpace(strings.SplitN(email, "@", 2)[0])
	if name == "" {
		return "ClankSpace user"
	}
	return name
}

func (s *Store) ClaimBootstrapOwner(ctx context.Context, principalID, email, displayName string) (domain.User, domain.Membership, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return domain.User{}, domain.Membership{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = displayNameForEmail(email)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, domain.Membership{}, err
	}
	defer tx.Rollback()
	var workspaceID, kind string
	if err = tx.QueryRowContext(ctx, `SELECT workspace_id,kind FROM principals WHERE id=?`, principalID).Scan(&workspaceID, &kind); err != nil {
		return domain.User{}, domain.Membership{}, err
	}
	if kind != "human" {
		return domain.User{}, domain.Membership{}, errors.New("bootstrap owner must be a human principal")
	}
	var existingUserID sql.NullString
	if err = tx.QueryRowContext(ctx, `SELECT user_id FROM principals WHERE id=?`, principalID).Scan(&existingUserID); err != nil {
		return domain.User{}, domain.Membership{}, err
	}
	if existingUserID.Valid && existingUserID.String != "" {
		var loginKind string
		if err = tx.QueryRowContext(ctx, `SELECT login_kind FROM users WHERE id=?`, existingUserID.String).Scan(&loginKind); err != nil {
			return domain.User{}, domain.Membership{}, err
		}
		if loginKind == "local" {
			if _, err = tx.ExecContext(ctx, `UPDATE users SET email_normalized=?,display_name=?,login_kind='email' WHERE id=?`, email, displayName, existingUserID.String); err != nil {
				return domain.User{}, domain.Membership{}, err
			}
			if _, err = tx.ExecContext(ctx, `UPDATE principals SET display_name=? WHERE id=?`, displayName, principalID); err != nil {
				return domain.User{}, domain.Membership{}, err
			}
		}
		user, membership, scanErr := s.scanUserMembershipTx(ctx, tx, existingUserID.String, workspaceID)
		if scanErr != nil {
			return domain.User{}, domain.Membership{}, scanErr
		}
		if err = tx.Commit(); err != nil {
			return domain.User{}, domain.Membership{}, err
		}
		return user, membership, nil
	}
	t := now()
	user := domain.User{ID: newID("user"), Email: email, DisplayName: displayName, Status: "active", LoginKind: "email", CreatedAt: t}
	if _, err = tx.ExecContext(ctx, `INSERT INTO users(id,email_normalized,display_name,status,created_at) VALUES(?,?,?,?,?)`, user.ID, user.Email, user.DisplayName, user.Status, ts(t)); err != nil {
		return domain.User{}, domain.Membership{}, err
	}
	membership := domain.Membership{ID: newID("membership"), WorkspaceID: workspaceID, UserID: user.ID, PrincipalID: principalID, Role: "owner", Status: "active", CreatedAt: t}
	if _, err = tx.ExecContext(ctx, `UPDATE principals SET user_id=?,display_name=? WHERE id=?`, user.ID, user.DisplayName, principalID); err != nil {
		return domain.User{}, domain.Membership{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_memberships(id,workspace_id,user_id,principal_id,role,status,created_at) VALUES(?,?,?,?,?,?,?)`, membership.ID, membership.WorkspaceID, membership.UserID, membership.PrincipalID, membership.Role, membership.Status, ts(t)); err != nil {
		return domain.User{}, domain.Membership{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE workspaces SET created_by_user_id=? WHERE id=?`, user.ID, workspaceID); err != nil {
		return domain.User{}, domain.Membership{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.User{}, domain.Membership{}, err
	}
	return user, membership, nil
}

func (s *Store) scanUserMembershipTx(ctx context.Context, tx *sql.Tx, userID, workspaceID string) (domain.User, domain.Membership, error) {
	var user domain.User
	var userCreated string
	var lastLogin sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT id,email_normalized,display_name,status,created_at,last_login_at,login_kind FROM users WHERE id=?`, userID).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &userCreated, &lastLogin, &user.LoginKind); err != nil {
		return user, domain.Membership{}, err
	}
	user.CreatedAt = parseTime(userCreated)
	if lastLogin.Valid {
		value := parseTime(lastLogin.String)
		user.LastLoginAt = &value
	}
	var membership domain.Membership
	var membershipCreated string
	if err := tx.QueryRowContext(ctx, `SELECT id,workspace_id,user_id,principal_id,role,status,created_at FROM workspace_memberships WHERE user_id=? AND workspace_id=?`, userID, workspaceID).Scan(&membership.ID, &membership.WorkspaceID, &membership.UserID, &membership.PrincipalID, &membership.Role, &membership.Status, &membershipCreated); err != nil {
		return user, membership, err
	}
	membership.CreatedAt = parseTime(membershipCreated)
	return user, membership, nil
}

func (s *Store) ListWorkspacesForUser(ctx context.Context, userID string) ([]domain.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT w.id,w.slug,w.name,w.authority_replica_id,w.created_at,m.role,w.status FROM workspaces w JOIN workspace_memberships m ON m.workspace_id=w.id WHERE m.user_id=? AND m.status='active' AND w.status='active' ORDER BY w.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Workspace{}
	for rows.Next() {
		var item domain.Workspace
		var created string
		if err = rows.Scan(&item.ID, &item.Slug, &item.Name, &item.AuthorityReplicaID, &created, &item.Role, &item.Status); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(created)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateWorkspaceForUser(ctx context.Context, userID, slug, name string) (domain.Workspace, domain.Membership, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	name = strings.TrimSpace(name)
	if !workspaceSlugPattern.MatchString(slug) {
		return domain.Workspace{}, domain.Membership{}, errors.New("workspace slug must contain lowercase letters, numbers, and single hyphens")
	}
	if name == "" {
		return domain.Workspace{}, domain.Membership{}, errors.New("workspace name is required")
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Workspace{}, domain.Membership{}, err
	}
	defer tx.Rollback()
	var displayName, status string
	if err = tx.QueryRowContext(ctx, `SELECT display_name,status FROM users WHERE id=?`, userID).Scan(&displayName, &status); err != nil {
		return domain.Workspace{}, domain.Membership{}, err
	}
	if status != "active" {
		return domain.Workspace{}, domain.Membership{}, errors.New("user is not active")
	}
	t := now()
	workspace := domain.Workspace{ID: newID("ws"), Slug: slug, Name: name, Status: "active", CreatedAt: t}
	principalID := newID("principal")
	membership := domain.Membership{ID: newID("membership"), WorkspaceID: workspace.ID, UserID: userID, PrincipalID: principalID, Role: "owner", Status: "active", CreatedAt: t}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspaces(id,name,created_at,slug,created_by_user_id) VALUES(?,?,?,?,?)`, workspace.ID, workspace.Name, ts(t), workspace.Slug, userID); err != nil {
		return domain.Workspace{}, domain.Membership{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO principals(id,workspace_id,display_name,kind,created_at,user_id,portable_actor_id,origin_replica_id) VALUES(?,?,?,?,?,?,?,?)`, principalID, workspace.ID, displayName, "human", ts(t), userID, newID("actor"), s.LocalReplicaID()); err != nil {
		return domain.Workspace{}, domain.Membership{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_memberships(id,workspace_id,user_id,principal_id,role,status,created_at) VALUES(?,?,?,?,?,?,?)`, membership.ID, membership.WorkspaceID, membership.UserID, membership.PrincipalID, membership.Role, membership.Status, ts(t)); err != nil {
		return domain.Workspace{}, domain.Membership{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.Workspace{}, domain.Membership{}, err
	}
	return workspace, membership, nil
}

func (s *Store) Membership(ctx context.Context, userID, workspaceID string) (domain.Membership, error) {
	var item domain.Membership
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT m.id,m.workspace_id,m.user_id,m.principal_id,m.role,m.status,m.created_at FROM workspace_memberships m JOIN workspaces w ON w.id=m.workspace_id WHERE m.user_id=? AND m.workspace_id=? AND m.status='active' AND w.status='active'`, userID, workspaceID).Scan(&item.ID, &item.WorkspaceID, &item.UserID, &item.PrincipalID, &item.Role, &item.Status, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	item.CreatedAt = parseTime(created)
	return item, nil
}

func (s *Store) PrincipalForMembership(ctx context.Context, membership domain.Membership) (domain.Principal, error) {
	var p domain.Principal
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT id,workspace_id,display_name,kind,created_at FROM principals WHERE id=? AND workspace_id=?`, membership.PrincipalID, membership.WorkspaceID).Scan(&p.ID, &p.WorkspaceID, &p.DisplayName, &p.Kind, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	p.CreatedAt = parseTime(created)
	return p, nil
}

func (s *Store) UserIDForPrincipal(ctx context.Context, principalID string) (string, error) {
	var userID sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM principals WHERE id=?`, principalID).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) || !userID.Valid || userID.String == "" {
		return "", ErrNotFound
	}
	return userID.String, err
}

func (s *Store) EnsureLocalUserForPrincipal(ctx context.Context, principal domain.Principal) (string, error) {
	if userID, err := s.UserIDForPrincipal(ctx, principal.ID); err == nil {
		return userID, nil
	}
	t := now()
	userID := newID("user")
	email := "local-" + principal.ID + "@clankspace.invalid"
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var existing sql.NullString
	if err = tx.QueryRowContext(ctx, `SELECT user_id FROM principals WHERE id=?`, principal.ID).Scan(&existing); err != nil {
		return "", err
	}
	if existing.Valid && existing.String != "" {
		return existing.String, nil
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO users(id,email_normalized,display_name,status,created_at,login_kind) VALUES(?,?,?,?,?,'local')`, userID, email, principal.DisplayName, "active", ts(t)); err != nil {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE principals SET user_id=? WHERE id=?`, userID, principal.ID); err != nil {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_memberships(id,workspace_id,user_id,principal_id,role,status,created_at) VALUES(?,?,?,?,?,'active',?)`, newID("membership"), principal.WorkspaceID, userID, principal.ID, "owner", ts(t)); err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return userID, nil
}

func (s *Store) CreateWorkspaceInvite(ctx context.Context, inviter domain.Membership, email, role, baseURL string) (domain.WorkspaceInvite, error) {
	invite, _, err := s.createWorkspaceInvite(ctx, inviter, email, role, baseURL, true)
	return invite, err
}

type WorkspaceInviteLink struct {
	Invite domain.WorkspaceInvite `json:"invite"`
	URL    string                 `json:"inviteUrl"`
}

type WorkspaceInvitePreview struct {
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	WorkspaceID   string    `json:"workspaceId"`
	WorkspaceName string    `json:"workspaceName"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

func (s *Store) CreateWorkspaceInviteLink(ctx context.Context, inviter domain.Membership, email, role, baseURL string) (WorkspaceInviteLink, error) {
	invite, link, err := s.createWorkspaceInvite(ctx, inviter, email, role, baseURL, false)
	return WorkspaceInviteLink{Invite: invite, URL: link}, err
}

func (s *Store) WorkspaceInvitePreview(ctx context.Context, token string) (WorkspaceInvitePreview, error) {
	var item WorkspaceInvitePreview
	var expires string
	err := s.db.QueryRowContext(ctx, `SELECT i.email_normalized,i.role,i.workspace_id,w.name,i.expires_at FROM workspace_invites i JOIN workspaces w ON w.id=i.workspace_id WHERE i.token_hash=? AND i.accepted_at IS NULL AND i.revoked_at IS NULL AND datetime(i.expires_at)>datetime(?) AND w.status='active'`, hashToken(strings.TrimSpace(token)), ts(now())).Scan(&item.Email, &item.Role, &item.WorkspaceID, &item.WorkspaceName, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	item.ExpiresAt = parseTime(expires)
	return item, nil
}

func (s *Store) createWorkspaceInvite(ctx context.Context, inviter domain.Membership, email, role, baseURL string, sendEmail bool) (domain.WorkspaceInvite, string, error) {
	if inviter.Role != "owner" || inviter.Status != "active" {
		return domain.WorkspaceInvite{}, "", errors.New("workspace owner required")
	}
	email, err := normalizeEmail(email)
	if err != nil {
		return domain.WorkspaceInvite{}, "", err
	}
	if role == "" {
		role = "member"
	}
	if role != "owner" && role != "member" {
		return domain.WorkspaceInvite{}, "", errors.New("invite role must be owner or member")
	}
	token, err := randomToken("invite_")
	if err != nil {
		return domain.WorkspaceInvite{}, "", err
	}
	t := now()
	invite := domain.WorkspaceInvite{ID: newID("invite"), WorkspaceID: inviter.WorkspaceID, Email: email, Role: role, ExpiresAt: t.Add(24 * time.Hour), CreatedAt: t}
	queryKey := "invite"
	if sendEmail {
		queryKey = "token"
	}
	link := strings.TrimRight(baseURL, "/") + "/?" + queryKey + "=" + url.QueryEscape(token)
	var sealed []byte
	if sendEmail {
		payload, _ := json.Marshal(emailPayload{Subject: "Join a ClankSpace workspace", Body: "Open this one-time link to join the workspace:\n\n" + link + "\n\nThis link expires in 24 hours."})
		sealed, err = s.seal(payload)
		if err != nil {
			return domain.WorkspaceInvite{}, "", err
		}
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.WorkspaceInvite{}, "", err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_invites(id,workspace_id,email_normalized,role,token_hash,invited_by_membership_id,expires_at,created_at) VALUES(?,?,?,?,?,?,?,?)`, invite.ID, invite.WorkspaceID, invite.Email, invite.Role, hashToken(token), inviter.ID, ts(invite.ExpiresAt), ts(t)); err != nil {
		return domain.WorkspaceInvite{}, "", err
	}
	if sendEmail {
		if _, err = tx.ExecContext(ctx, `INSERT INTO email_outbox(id,recipient,template,payload_ciphertext,next_attempt_at,created_at) VALUES(?,?,?,?,?,?)`, newID("mail"), email, "workspace_invite", sealed, ts(t), ts(t)); err != nil {
			return domain.WorkspaceInvite{}, "", err
		}
	}
	if err = tx.Commit(); err != nil {
		return domain.WorkspaceInvite{}, "", err
	}
	return invite, link, nil
}

func (s *Store) RequestMagicLink(ctx context.Context, email, fingerprint, baseURL string) error {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil
	}
	allowed, err := s.allowAuthRequest(ctx, "email:"+email, 5)
	if err != nil || !allowed {
		return err
	}
	allowed, err = s.allowAuthRequest(ctx, "source:"+fingerprint, 20)
	if err != nil || !allowed {
		return err
	}
	var eligible int
	if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE email_normalized=? AND status='active'`, email).Scan(&eligible); err != nil {
		return err
	}
	if eligible == 0 {
		return nil
	}
	token, err := randomToken("login_")
	if err != nil {
		return err
	}
	t := now()
	link := strings.TrimRight(baseURL, "/") + "/?token=" + url.QueryEscape(token)
	payload, _ := json.Marshal(emailPayload{Subject: "Your ClankSpace sign-in link", Body: "Open this one-time link to sign in:\n\n" + link + "\n\nThis link expires in 15 minutes."})
	sealed, err := s.seal(payload)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO login_challenges(id,email_normalized,token_hash,expires_at,request_fingerprint_hash,created_at) VALUES(?,?,?,?,?,?)`, newID("login"), email, hashToken(token), ts(t.Add(15*time.Minute)), hashToken(fingerprint), ts(t)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO email_outbox(id,recipient,template,payload_ciphertext,next_attempt_at,created_at) VALUES(?,?,?,?,?,?)`, newID("mail"), email, "magic_link", sealed, ts(t), ts(t)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) allowAuthRequest(ctx context.Context, key string, maximum int) (bool, error) {
	t := now()
	cutoff := t.Add(-10 * time.Minute)
	keyHash := hashToken(key)
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM auth_rate_limits WHERE datetime(window_started_at)<datetime(?)`, ts(cutoff)); err != nil {
		return false, err
	}
	var started string
	var count int
	err = tx.QueryRowContext(ctx, `SELECT window_started_at,request_count FROM auth_rate_limits WHERE key_hash=?`, keyHash).Scan(&started, &count)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && parseTime(started).Before(cutoff)) {
		count = 1
		_, err = tx.ExecContext(ctx, `INSERT INTO auth_rate_limits(key_hash,window_started_at,request_count) VALUES(?,?,1) ON CONFLICT(key_hash) DO UPDATE SET window_started_at=excluded.window_started_at,request_count=1`, keyHash, ts(t))
	} else if err == nil {
		count++
		_, err = tx.ExecContext(ctx, `UPDATE auth_rate_limits SET request_count=? WHERE key_hash=?`, count, keyHash)
	}
	if err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return count <= maximum, nil
}

func (s *Store) ConsumeAuthToken(ctx context.Context, token, displayName string) (SessionResult, error) {
	return s.consumeAuthToken(ctx, token, displayName, "", "", false)
}

func (s *Store) ConsumeInviteWithPassword(ctx context.Context, token, displayName, password, fingerprint string) (SessionResult, error) {
	if err := validatePassword(password); err != nil {
		return SessionResult{}, err
	}
	preview, err := s.WorkspaceInvitePreview(ctx, token)
	if err != nil {
		return SessionResult{}, err
	}
	allowed, err := s.allowAuthRequest(ctx, "invite-email:"+preview.Email, 10)
	if err != nil || !allowed {
		return SessionResult{}, ErrNotFound
	}
	allowed, err = s.allowAuthRequest(ctx, "invite-source:"+fingerprint, 30)
	if err != nil || !allowed {
		return SessionResult{}, ErrNotFound
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return SessionResult{}, err
	}
	return s.consumeAuthToken(ctx, token, displayName, password, passwordHash, true)
}

func (s *Store) consumeAuthToken(ctx context.Context, token, displayName, password, passwordHash string, inviteOnly bool) (SessionResult, error) {
	tokenHash := hashToken(strings.TrimSpace(token))
	t := now()
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SessionResult{}, err
	}
	defer tx.Rollback()
	var email string
	var inviteID, workspaceID, role sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT id,workspace_id,email_normalized,role FROM workspace_invites WHERE token_hash=? AND accepted_at IS NULL AND revoked_at IS NULL AND datetime(expires_at)>datetime(?)`, tokenHash, ts(t)).Scan(&inviteID, &workspaceID, &email, &role)
	if errors.Is(err, sql.ErrNoRows) {
		if inviteOnly {
			return SessionResult{}, ErrNotFound
		}
		var challengeID string
		err = tx.QueryRowContext(ctx, `SELECT id,email_normalized FROM login_challenges WHERE token_hash=? AND consumed_at IS NULL AND datetime(expires_at)>datetime(?)`, tokenHash, ts(t)).Scan(&challengeID, &email)
		if errors.Is(err, sql.ErrNoRows) {
			return SessionResult{}, ErrNotFound
		}
		if err != nil {
			return SessionResult{}, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE login_challenges SET consumed_at=? WHERE id=? AND consumed_at IS NULL`, ts(t), challengeID); err != nil {
			return SessionResult{}, err
		}
	} else if err != nil {
		return SessionResult{}, err
	}
	var user domain.User
	var userCreated string
	var lastLogin sql.NullString
	var existingPassword string
	err = tx.QueryRowContext(ctx, `SELECT id,email_normalized,display_name,status,created_at,last_login_at,login_kind,password_hash FROM users WHERE email_normalized=?`, email).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &userCreated, &lastLogin, &user.LoginKind, &existingPassword)
	if errors.Is(err, sql.ErrNoRows) {
		if !inviteID.Valid {
			return SessionResult{}, ErrNotFound
		}
		displayName = strings.TrimSpace(displayName)
		if displayName == "" {
			displayName = displayNameForEmail(email)
		}
		user = domain.User{ID: newID("user"), Email: email, DisplayName: displayName, Status: "active", LoginKind: "email", CreatedAt: t}
		if _, err = tx.ExecContext(ctx, `INSERT INTO users(id,email_normalized,display_name,status,created_at,last_login_at,login_kind,password_hash) VALUES(?,?,?,?,?,?,?,?)`, user.ID, user.Email, user.DisplayName, user.Status, ts(t), ts(t), "email", passwordHash); err != nil {
			return SessionResult{}, err
		}
	} else if err != nil {
		return SessionResult{}, err
	} else {
		user.CreatedAt = parseTime(userCreated)
		if user.Status != "active" {
			return SessionResult{}, ErrNotFound
		}
		if inviteOnly {
			if existingPassword == "" {
				if _, err = tx.ExecContext(ctx, `UPDATE users SET password_hash=? WHERE id=?`, passwordHash, user.ID); err != nil {
					return SessionResult{}, err
				}
			} else if !verifyPassword(existingPassword, password) {
				return SessionResult{}, ErrNotFound
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE users SET last_login_at=? WHERE id=?`, ts(t), user.ID); err != nil {
			return SessionResult{}, err
		}
	}
	last := t
	user.LastLoginAt = &last
	if inviteID.Valid {
		var count int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_memberships WHERE workspace_id=? AND user_id=?`, workspaceID.String, user.ID).Scan(&count); err != nil {
			return SessionResult{}, err
		}
		if count == 0 {
			principalID := newID("principal")
			membershipID := newID("membership")
			if _, err = tx.ExecContext(ctx, `INSERT INTO principals(id,workspace_id,display_name,kind,created_at,user_id,portable_actor_id,origin_replica_id) VALUES(?,?,?,?,?,?,?,?)`, principalID, workspaceID.String, user.DisplayName, "human", ts(t), user.ID, newID("actor"), s.LocalReplicaID()); err != nil {
				return SessionResult{}, err
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_memberships(id,workspace_id,user_id,principal_id,role,status,created_at) VALUES(?,?,?,?,?,?,?)`, membershipID, workspaceID.String, user.ID, principalID, role.String, "active", ts(t)); err != nil {
				return SessionResult{}, err
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE workspace_invites SET accepted_by_user_id=?,accepted_at=? WHERE id=? AND accepted_at IS NULL`, user.ID, ts(t), inviteID.String); err != nil {
			return SessionResult{}, err
		}
	}
	sessionToken, err := randomToken("session_")
	if err != nil {
		return SessionResult{}, err
	}
	csrfToken, err := randomToken("csrf_")
	if err != nil {
		return SessionResult{}, err
	}
	session := domain.BrowserSession{ID: newID("session"), UserID: user.ID, ExpiresAt: t.Add(30 * 24 * time.Hour)}
	if _, err = tx.ExecContext(ctx, `INSERT INTO browser_sessions(id,user_id,token_hash,csrf_hash,expires_at,last_seen_at,created_at) VALUES(?,?,?,?,?,?,?)`, session.ID, session.UserID, hashToken(sessionToken), hashToken(csrfToken), ts(session.ExpiresAt), ts(t), ts(t)); err != nil {
		return SessionResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return SessionResult{}, err
	}
	session.CSRFToken = csrfToken
	return SessionResult{User: user, Session: session, SessionToken: sessionToken, CSRFToken: csrfToken}, nil
}

func (s *Store) SetUserPassword(ctx context.Context, userID, password string) error {
	encoded, err := hashPassword(password)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	result, err := s.db.ExecContext(ctx, `UPDATE users SET password_hash=? WHERE id=? AND status='active'`, encoded, userID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) AuthenticatePassword(ctx context.Context, email, password, fingerprint string) (SessionResult, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return SessionResult{}, ErrNotFound
	}
	allowed, err := s.allowAuthRequest(ctx, "password-email:"+email, 10)
	if err != nil || !allowed {
		return SessionResult{}, ErrNotFound
	}
	allowed, err = s.allowAuthRequest(ctx, "password-source:"+fingerprint, 30)
	if err != nil || !allowed {
		return SessionResult{}, ErrNotFound
	}
	var user domain.User
	var created string
	var lastLogin sql.NullString
	var passwordHash string
	err = s.db.QueryRowContext(ctx, `SELECT id,email_normalized,display_name,status,created_at,last_login_at,login_kind,password_hash FROM users WHERE email_normalized=?`, email).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &created, &lastLogin, &user.LoginKind, &passwordHash)
	if err != nil || user.Status != "active" || passwordHash == "" || !verifyPassword(passwordHash, password) {
		return SessionResult{}, ErrNotFound
	}
	t := now()
	user.CreatedAt = parseTime(created)
	user.LastLoginAt = &t
	sessionToken, err := randomToken("session_")
	if err != nil {
		return SessionResult{}, err
	}
	csrfToken, err := randomToken("csrf_")
	if err != nil {
		return SessionResult{}, err
	}
	session := domain.BrowserSession{ID: newID("session"), UserID: user.ID, ExpiresAt: t.Add(30 * 24 * time.Hour)}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SessionResult{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE users SET last_login_at=? WHERE id=?`, ts(t), user.ID); err != nil {
		return SessionResult{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO browser_sessions(id,user_id,token_hash,csrf_hash,expires_at,last_seen_at,created_at) VALUES(?,?,?,?,?,?,?)`, session.ID, session.UserID, hashToken(sessionToken), hashToken(csrfToken), ts(session.ExpiresAt), ts(t), ts(t)); err != nil {
		return SessionResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return SessionResult{}, err
	}
	session.CSRFToken = csrfToken
	return SessionResult{User: user, Session: session, SessionToken: sessionToken, CSRFToken: csrfToken}, nil
}

func (s *Store) AuthenticateSession(ctx context.Context, sessionToken, csrfToken string, requireCSRF bool) (domain.User, error) {
	var user domain.User
	var created string
	var lastLogin sql.NullString
	var storedCSRF string
	err := s.db.QueryRowContext(ctx, `SELECT u.id,u.email_normalized,u.display_name,u.status,u.created_at,u.last_login_at,u.login_kind,s.csrf_hash FROM browser_sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=? AND s.revoked_at IS NULL AND s.expires_at>?`, hashToken(sessionToken), ts(now())).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &created, &lastLogin, &user.LoginKind, &storedCSRF)
	if errors.Is(err, sql.ErrNoRows) || user.Status != "active" {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	if requireCSRF && (csrfToken == "" || storedCSRF != hashToken(csrfToken)) {
		return domain.User{}, errors.New("invalid CSRF token")
	}
	user.CreatedAt = parseTime(created)
	if lastLogin.Valid {
		value := parseTime(lastLogin.String)
		user.LastLoginAt = &value
	}
	return user, nil
}

func (s *Store) RevokeSession(ctx context.Context, sessionToken string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE browser_sessions SET revoked_at=? WHERE token_hash=? AND revoked_at IS NULL`, ts(now()), hashToken(sessionToken))
	return err
}

func (s *Store) ListMembers(ctx context.Context, requester domain.Membership) ([]domain.Membership, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT m.id,m.workspace_id,m.user_id,m.principal_id,m.role,m.status,m.created_at,u.id,u.email_normalized,u.display_name,u.status,u.created_at,u.last_login_at,u.login_kind FROM workspace_memberships m JOIN users u ON u.id=m.user_id WHERE m.workspace_id=? ORDER BY u.display_name`, requester.WorkspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Membership{}
	for rows.Next() {
		var item domain.Membership
		var user domain.User
		var membershipCreated, userCreated string
		var lastLogin sql.NullString
		if err = rows.Scan(&item.ID, &item.WorkspaceID, &item.UserID, &item.PrincipalID, &item.Role, &item.Status, &membershipCreated, &user.ID, &user.Email, &user.DisplayName, &user.Status, &userCreated, &lastLogin, &user.LoginKind); err != nil {
			return nil, err
		}
		item.CreatedAt, user.CreatedAt = parseTime(membershipCreated), parseTime(userCreated)
		if lastLogin.Valid {
			value := parseTime(lastLogin.String)
			user.LastLoginAt = &value
		}
		item.User = &user
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListInvites(ctx context.Context, requester domain.Membership) ([]domain.WorkspaceInvite, error) {
	if requester.Role != "owner" {
		return nil, errors.New("workspace owner required")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,workspace_id,email_normalized,role,expires_at,accepted_at,revoked_at,created_at FROM workspace_invites WHERE workspace_id=? ORDER BY created_at DESC`, requester.WorkspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.WorkspaceInvite{}
	for rows.Next() {
		var item domain.WorkspaceInvite
		var expiresAt, createdAt string
		var acceptedAt, revokedAt sql.NullString
		if err = rows.Scan(&item.ID, &item.WorkspaceID, &item.Email, &item.Role, &expiresAt, &acceptedAt, &revokedAt, &createdAt); err != nil {
			return nil, err
		}
		item.ExpiresAt, item.CreatedAt = parseTime(expiresAt), parseTime(createdAt)
		if acceptedAt.Valid {
			value := parseTime(acceptedAt.String)
			item.AcceptedAt = &value
		}
		if revokedAt.Valid {
			value := parseTime(revokedAt.String)
			item.RevokedAt = &value
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) RevokeInvite(ctx context.Context, requester domain.Membership, inviteID string) error {
	if requester.Role != "owner" {
		return errors.New("workspace owner required")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE workspace_invites SET revoked_at=? WHERE id=? AND workspace_id=? AND accepted_at IS NULL AND revoked_at IS NULL`, ts(now()), inviteID, requester.WorkspaceID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateMembership(ctx context.Context, requester domain.Membership, membershipID, role, status string) error {
	if requester.Role != "owner" {
		return errors.New("workspace owner required")
	}
	if role != "owner" && role != "member" {
		return errors.New("role must be owner or member")
	}
	if status != "active" && status != "suspended" {
		return errors.New("status must be active or suspended")
	}
	var targetRole, targetStatus string
	if err := s.db.QueryRowContext(ctx, `SELECT role,status FROM workspace_memberships WHERE id=? AND workspace_id=?`, membershipID, requester.WorkspaceID).Scan(&targetRole, &targetStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if targetRole == "owner" && targetStatus == "active" && (role != "owner" || status != "active") {
		var owners int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_memberships WHERE workspace_id=? AND role='owner' AND status='active'`, requester.WorkspaceID).Scan(&owners); err != nil {
			return err
		}
		if owners <= 1 {
			return errors.New("workspace must retain at least one active owner")
		}
	}
	_, err := s.db.ExecContext(ctx, `UPDATE workspace_memberships SET role=?,status=? WHERE id=? AND workspace_id=?`, role, status, membershipID, requester.WorkspaceID)
	return err
}

func (s *Store) ListProjectTokens(ctx context.Context, requester domain.Membership, projectID string) ([]domain.TokenSummary, error) {
	if _, err := s.GetProject(ctx, requester.WorkspaceID, projectID); err != nil {
		return nil, err
	}
	query := `SELECT t.id,p.id,p.display_name,t.token_prefix,t.scopes_json,t.created_at,t.expires_at,t.revoked_at FROM api_tokens t JOIN principals p ON p.id=t.principal_id JOIN project_principals pp ON pp.principal_id=p.id WHERE pp.project_id=?`
	args := []any{projectID}
	if requester.Role != "owner" {
		query += ` AND p.created_by_membership_id=?`
		args = append(args, requester.ID)
	}
	query += ` ORDER BY t.created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.TokenSummary{}
	for rows.Next() {
		var item domain.TokenSummary
		var scopesJSON, created string
		var expires, revoked sql.NullString
		if err = rows.Scan(&item.ID, &item.PrincipalID, &item.DisplayName, &item.Prefix, &scopesJSON, &created, &expires, &revoked); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(created)
		_ = json.Unmarshal([]byte(scopesJSON), &item.Scopes)
		if expires.Valid {
			value := parseTime(expires.String)
			item.ExpiresAt = &value
		}
		if revoked.Valid {
			value := parseTime(revoked.String)
			item.RevokedAt = &value
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) RevokeProjectToken(ctx context.Context, requester domain.Membership, projectID, tokenID string) error {
	if _, err := s.GetProject(ctx, requester.WorkspaceID, projectID); err != nil {
		return err
	}
	query := `UPDATE api_tokens SET revoked_at=? WHERE id=? AND revoked_at IS NULL AND principal_id IN (SELECT principal_id FROM project_principals WHERE project_id=?)`
	args := []any{ts(now()), tokenID, projectID}
	if requester.Role != "owner" {
		query += ` AND principal_id IN (SELECT id FROM principals WHERE created_by_membership_id=?)`
		args = append(args, requester.ID)
	}
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ClaimOutbox(ctx context.Context, limit int) ([]OutboxMessage, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,recipient,template,payload_ciphertext FROM email_outbox WHERE sent_at IS NULL AND datetime(next_attempt_at)<=datetime(?) ORDER BY created_at LIMIT ?`, ts(now()), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []OutboxMessage{}
	for rows.Next() {
		var item OutboxMessage
		var encrypted []byte
		if err = rows.Scan(&item.ID, &item.Recipient, &item.Template, &encrypted); err != nil {
			return nil, err
		}
		plaintext, openErr := s.openSealed(encrypted)
		if openErr != nil {
			return nil, fmt.Errorf("decrypt outbox %s: %w", item.ID, openErr)
		}
		var payload emailPayload
		if err = json.Unmarshal(plaintext, &payload); err != nil {
			return nil, err
		}
		item.Subject, item.Body = payload.Subject, payload.Body
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) MarkOutboxSent(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE email_outbox SET sent_at=?,payload_ciphertext=x'',last_error='' WHERE id=? AND sent_at IS NULL`, ts(now()), id)
	return err
}

func (s *Store) MarkOutboxFailed(ctx context.Context, id string, sendErr error) error {
	message := "delivery failed"
	if sendErr != nil {
		message = sendErr.Error()
	}
	if len(message) > 500 {
		message = message[:500]
	}
	_, err := s.db.ExecContext(ctx, `UPDATE email_outbox SET attempts=attempts+1,next_attempt_at=?,last_error=? WHERE id=? AND sent_at IS NULL`, ts(now().Add(time.Minute)), message, id)
	return err
}

func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,email_normalized,display_name,status,created_at,last_login_at,login_kind FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.User{}
	for rows.Next() {
		var item domain.User
		var created string
		var lastLogin sql.NullString
		if err = rows.Scan(&item.ID, &item.Email, &item.DisplayName, &item.Status, &created, &lastLogin, &item.LoginKind); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(created)
		if lastLogin.Valid {
			value := parseTime(lastLogin.String)
			item.LastLoginAt = &value
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SetUserStatus(ctx context.Context, userID, status string) error {
	if status != "active" && status != "suspended" {
		return errors.New("user status must be active or suspended")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE users SET status=? WHERE id=?`, status, userID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrNotFound
	}
	if status == "suspended" {
		_, err = s.db.ExecContext(ctx, `UPDATE browser_sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL`, ts(now()), userID)
	}
	return err
}

func (s *Store) ListAllWorkspaces(ctx context.Context) ([]domain.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,slug,name,authority_replica_id,created_at,status FROM workspaces ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Workspace{}
	for rows.Next() {
		var item domain.Workspace
		var created string
		if err = rows.Scan(&item.ID, &item.Slug, &item.Name, &item.AuthorityReplicaID, &created, &item.Status); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(created)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SetWorkspaceStatus(ctx context.Context, workspaceID, status string) error {
	if status != "active" && status != "suspended" {
		return errors.New("workspace status must be active or suspended")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE workspaces SET status=? WHERE id=?`, status, workspaceID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrNotFound
	}
	return nil
}
