package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/domain"
)

const setupCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type SetupRequest struct {
	ID            string    `json:"id"`
	UserCode      string    `json:"userCode,omitempty"`
	ProjectSlug   string    `json:"projectSlug"`
	ProjectName   string    `json:"projectName"`
	RepositoryURL string    `json:"repositoryUrl,omitempty"`
	AgentName     string    `json:"agentName"`
	Approved      bool      `json:"approved"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

type SetupExchange struct {
	Project domain.Project
	Token   string
}

func (s *Store) AllowSetupRequest(ctx context.Context, fingerprint string) (bool, error) {
	return s.allowAuthRequest(ctx, "setup|"+fingerprint, 20)
}

func setupUserCode() (string, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for i := range random {
		random[i] = setupCodeAlphabet[int(random[i])%len(setupCodeAlphabet)]
	}
	return string(random[:4]) + "-" + string(random[4:]), nil
}

func normalizeSetupCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func (s *Store) CreateSetupRequest(ctx context.Context, challenge, projectSlug, projectName, repositoryURL, agentName string) (SetupRequest, string, error) {
	challenge = strings.ToLower(strings.TrimSpace(challenge))
	if decoded, err := hex.DecodeString(challenge); err != nil || len(decoded) != sha256.Size {
		return SetupRequest{}, "", errors.New("code challenge must be a SHA-256 hex digest")
	}
	deviceCode, err := randomToken("setup_")
	if err != nil {
		return SetupRequest{}, "", err
	}
	userCode, err := setupUserCode()
	if err != nil {
		return SetupRequest{}, "", err
	}
	t := now()
	item := SetupRequest{
		ID:            newID("setup"),
		UserCode:      userCode,
		ProjectSlug:   strings.TrimSpace(projectSlug),
		ProjectName:   strings.TrimSpace(projectName),
		RepositoryURL: strings.TrimSpace(repositoryURL),
		AgentName:     strings.TrimSpace(agentName),
		ExpiresAt:     t.Add(10 * time.Minute),
	}
	item.ProjectSlug = strings.ToLower(item.ProjectSlug)
	if !workspaceSlugPattern.MatchString(item.ProjectSlug) {
		return SetupRequest{}, "", errors.New("project slug must use lowercase letters, numbers, and single hyphens")
	}
	if item.ProjectName == "" || item.AgentName == "" {
		return SetupRequest{}, "", errors.New("project slug, project name, and agent name are required")
	}
	if len(item.ProjectName) > 100 || len(item.AgentName) > 100 || len(item.RepositoryURL) > 2048 {
		return SetupRequest{}, "", errors.New("setup request fields are too long")
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO setup_requests(id,device_code_hash,user_code_hash,code_challenge,project_slug,project_name,repository_url,agent_name,expires_at,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, item.ID, hashToken(deviceCode), hashToken(userCode), challenge, item.ProjectSlug, item.ProjectName, item.RepositoryURL, item.AgentName, ts(item.ExpiresAt), ts(t))
	return item, deviceCode, err
}

func (s *Store) SetupRequestByUserCode(ctx context.Context, userCode string) (SetupRequest, error) {
	var item SetupRequest
	var approved sql.NullString
	var expires string
	err := s.db.QueryRowContext(ctx, `SELECT id,project_slug,project_name,repository_url,agent_name,approved_at,expires_at FROM setup_requests WHERE user_code_hash=? AND consumed_at IS NULL AND expires_at>?`, hashToken(normalizeSetupCode(userCode)), ts(now())).Scan(&item.ID, &item.ProjectSlug, &item.ProjectName, &item.RepositoryURL, &item.AgentName, &approved, &expires)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return item, ErrNotFound
		}
		return item, err
	}
	item.UserCode = normalizeSetupCode(userCode)
	item.Approved = approved.Valid
	item.ExpiresAt = parseTime(expires)
	return item, nil
}

func (s *Store) ApproveSetupRequest(ctx context.Context, userCode string, membership domain.Membership, project domain.Project, token string) error {
	sealed, err := s.seal([]byte(strings.TrimSpace(token)))
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE setup_requests SET membership_id=?,project_id=?,credential_ciphertext=?,approved_at=? WHERE user_code_hash=? AND approved_at IS NULL AND consumed_at IS NULL AND expires_at>?`, membership.ID, project.ID, sealed, ts(now()), hashToken(normalizeSetupCode(userCode)), ts(now()))
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return ErrConflict
	}
	return nil
}

func (s *Store) ExchangeSetupRequest(ctx context.Context, deviceCode, verifier string) (SetupExchange, error) {
	var projectID, challenge string
	var sealed []byte
	err := s.db.QueryRowContext(ctx, `SELECT project_id,code_challenge,credential_ciphertext FROM setup_requests WHERE device_code_hash=? AND approved_at IS NOT NULL AND consumed_at IS NULL AND expires_at>?`, hashToken(strings.TrimSpace(deviceCode)), ts(now())).Scan(&projectID, &challenge, &sealed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SetupExchange{}, ErrNotFound
		}
		return SetupExchange{}, err
	}
	digest := sha256.Sum256([]byte(verifier))
	if hex.EncodeToString(digest[:]) != challenge {
		return SetupExchange{}, errors.New("invalid setup verifier")
	}
	plaintext, err := s.openSealed(sealed)
	if err != nil {
		return SetupExchange{}, err
	}
	var project domain.Project
	var created string
	if err = s.db.QueryRowContext(ctx, `SELECT id,workspace_id,slug,name,description,created_at FROM projects WHERE id=?`, projectID).Scan(&project.ID, &project.WorkspaceID, &project.Slug, &project.Name, &project.Description, &created); err != nil {
		return SetupExchange{}, err
	}
	project.CreatedAt = parseTime(created)
	result, err := s.db.ExecContext(ctx, `UPDATE setup_requests SET consumed_at=?,credential_ciphertext=x'' WHERE device_code_hash=? AND consumed_at IS NULL`, ts(now()), hashToken(strings.TrimSpace(deviceCode)))
	if err != nil {
		return SetupExchange{}, err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return SetupExchange{}, ErrConflict
	}
	return SetupExchange{Project: project, Token: string(plaintext)}, nil
}
