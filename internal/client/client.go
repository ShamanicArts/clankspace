package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/google/uuid"
)

type Client struct {
	BaseURL, Token string
	HTTP           *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Token: token, HTTP: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) Do(ctx context.Context, method, path string, in, out any) error {
	key := ""
	if method != http.MethodGet {
		key = uuid.NewString()
	}
	return c.DoWithKey(ctx, method, path, key, in, out)
}

func (c *Client) RequestMagicLink(ctx context.Context, email string) error {
	var out map[string]string
	return c.Do(ctx, http.MethodPost, "/auth/magic-link", map[string]string{"email": email}, &out)
}

func (c *Client) DoWithKey(ctx context.Context, method, path, idempotencyKey string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+"/api/v1"+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	if method != http.MethodGet {
		if strings.TrimSpace(idempotencyKey) == "" {
			return fmt.Errorf("idempotency key is required for %s %s", method, path)
		}
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return fmt.Errorf("clankspace: %s", e.Error)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) ListProjects(ctx context.Context) ([]domain.Project, error) {
	var o struct {
		Projects []domain.Project `json:"projects"`
	}
	err := c.Do(ctx, "GET", "/projects", nil, &o)
	return o.Projects, err
}

func (c *Client) ListWorkspaces(ctx context.Context) ([]domain.Workspace, error) {
	var out struct {
		Workspaces []domain.Workspace `json:"workspaces"`
	}
	err := c.Do(ctx, http.MethodGet, "/admin/workspaces", nil, &out)
	return out.Workspaces, err
}

func (c *Client) CreateWorkspace(ctx context.Context, slug, name string) (domain.Workspace, error) {
	var out struct {
		Workspace domain.Workspace `json:"workspace"`
	}
	err := c.Do(ctx, http.MethodPost, "/admin/workspaces", map[string]string{"slug": slug, "name": name}, &out)
	return out.Workspace, err
}
func (c *Client) CreateProject(ctx context.Context, slug, name, description string) (domain.Project, error) {
	var o struct {
		Project domain.Project `json:"project"`
	}
	err := c.Do(ctx, "POST", "/projects", map[string]string{"slug": slug, "name": name, "description": description}, &o)
	return o.Project, err
}

func (c *Client) ExportProject(ctx context.Context, project string) (map[string]any, error) {
	var out map[string]any
	err := c.Do(ctx, http.MethodGet, "/projects/"+project+"/export", nil, &out)
	return out, err
}

func (c *Client) IssueProjectToken(ctx context.Context, project, displayName string) (domain.ProjectCredential, error) {
	var out domain.ProjectCredential
	err := c.Do(ctx, http.MethodPost, "/projects/"+project+"/tokens", map[string]string{"displayName": displayName}, &out)
	return out, err
}
func (c *Client) StartRun(ctx context.Context, in domain.StartRunInput) (domain.Run, error) {
	var o struct {
		Run domain.Run `json:"run"`
	}
	err := c.Do(ctx, "POST", "/runs", in, &o)
	return o.Run, err
}
func (c *Client) EndRun(ctx context.Context, id string, in domain.EndRunInput) (domain.Run, error) {
	var o struct {
		Run domain.Run `json:"run"`
	}
	err := c.Do(ctx, "POST", "/runs/"+id+"/end", in, &o)
	return o.Run, err
}
func (c *Client) ListRuns(ctx context.Context, project string, limit int) ([]domain.Run, error) {
	var out struct {
		Runs []domain.Run `json:"runs"`
	}
	err := c.Do(ctx, http.MethodGet, fmt.Sprintf("/projects/%s/runs?limit=%d", project, limit), nil, &out)
	return out.Runs, err
}
func (c *Client) CreateNote(ctx context.Context, project string, in domain.CreateNoteInput) (domain.Note, error) {
	var o struct {
		Note domain.Note `json:"note"`
	}
	err := c.Do(ctx, "POST", "/projects/"+project+"/notes", in, &o)
	return o.Note, err
}
func (c *Client) SupersedeNote(ctx context.Context, project, id string, in domain.SupersedeNoteInput) (domain.Note, error) {
	var o struct {
		Note domain.Note `json:"note"`
	}
	err := c.Do(ctx, "POST", "/projects/"+project+"/notes/"+id+"/supersede", in, &o)
	return o.Note, err
}
func (c *Client) CreateTrajectory(ctx context.Context, project string, in domain.CreateTrajectoryInput) (domain.Trajectory, error) {
	var o struct {
		Trajectory domain.Trajectory `json:"trajectory"`
	}
	err := c.Do(ctx, "POST", "/projects/"+project+"/trajectories", in, &o)
	return o.Trajectory, err
}
func (c *Client) Brief(ctx context.Context, project string, in domain.BriefInput) (domain.Brief, error) {
	var o domain.Brief
	err := c.Do(ctx, "POST", "/projects/"+project+"/brief", in, &o)
	return o, err
}
func (c *Client) AttachRepository(ctx context.Context, project, url string) (domain.Repository, error) {
	var o struct {
		Repository domain.Repository `json:"repository"`
	}
	err := c.Do(ctx, "POST", "/projects/"+project+"/repositories", map[string]string{"url": url}, &o)
	return o.Repository, err
}

func (c *Client) JoinReplica(ctx context.Context, remoteURL, code string) (domain.Workspace, error) {
	var out struct {
		Workspace domain.Workspace `json:"workspace"`
	}
	err := c.Do(ctx, http.MethodPost, "/admin/replica/join", map[string]string{"remoteUrl": remoteURL, "code": code}, &out)
	return out.Workspace, err
}

func (c *Client) MirrorReplica(ctx context.Context, workspaceID, remoteURL, code string) (domain.Workspace, error) {
	var out struct {
		Workspace domain.Workspace `json:"workspace"`
	}
	err := c.Do(ctx, http.MethodPost, "/admin/replica/mirror", map[string]string{"workspaceId": workspaceID, "remoteUrl": remoteURL, "code": code}, &out)
	return out.Workspace, err
}

func (c *Client) SyncOnce(ctx context.Context) error {
	var out map[string]string
	return c.Do(ctx, http.MethodPost, "/admin/sync", map[string]string{}, &out)
}

func (c *Client) ExportWorkspaceBundle(ctx context.Context, workspaceID string) (domain.WorkspaceBundle, error) {
	var out domain.WorkspaceBundle
	err := c.Do(ctx, http.MethodGet, "/admin/workspaces/"+workspaceID+"/bundle", nil, &out)
	return out, err
}

func (c *Client) ImportWorkspaceBundle(ctx context.Context, bundle domain.WorkspaceBundle) (domain.Workspace, error) {
	var out struct {
		Workspace domain.Workspace `json:"workspace"`
	}
	err := c.Do(ctx, http.MethodPost, "/admin/bundles/import", bundle, &out)
	return out.Workspace, err
}
