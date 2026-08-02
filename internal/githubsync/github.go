package githubsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/domain"
)

type Client struct {
	HTTP    *http.Client
	BaseURL string
	Token   string
}

func New(token string) *Client {
	return &Client{HTTP: &http.Client{Timeout: 12 * time.Second}, BaseURL: "https://api.github.com", Token: token}
}

func ParseRepository(raw string) (domain.Repository, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Hostname(), "github.com") {
		return domain.Repository{}, errors.New("only public github.com repository URLs are supported in the pilot")
	}
	parts := strings.Split(strings.Trim(strings.TrimSuffix(u.Path, ".git"), "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return domain.Repository{}, errors.New("repository URL must look like https://github.com/owner/repo")
	}
	return domain.Repository{URL: "https://github.com/" + parts[0] + "/" + parts[1], Host: "github.com", Owner: parts[0], Name: parts[1], Visibility: "public"}, nil
}

func (c *Client) Sync(ctx context.Context, repo domain.Repository) (domain.Repository, []domain.ExternalArtifact, error) {
	var meta struct {
		Description   string `json:"description"`
		DefaultBranch string `json:"default_branch"`
		Stargazers    int    `json:"stargazers_count"`
		Private       bool   `json:"private"`
	}
	if err := c.get(ctx, fmt.Sprintf("/repos/%s/%s", repo.Owner, repo.Name), &meta); err != nil {
		return repo, nil, err
	}
	if meta.Private {
		return repo, nil, errors.New("private repositories are deferred from the pilot")
	}
	repo.Description, repo.Default, repo.Stars = meta.Description, meta.DefaultBranch, meta.Stargazers
	var pulls []struct {
		Number    int       `json:"number"`
		Title     string    `json:"title"`
		HTMLURL   string    `json:"html_url"`
		State     string    `json:"state"`
		UpdatedAt time.Time `json:"updated_at"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := c.get(ctx, fmt.Sprintf("/repos/%s/%s/pulls?state=open&per_page=50", repo.Owner, repo.Name), &pulls); err != nil {
		return repo, nil, err
	}
	artifacts := make([]domain.ExternalArtifact, 0, len(pulls))
	for _, p := range pulls {
		artifacts = append(artifacts, domain.ExternalArtifact{ID: fmt.Sprintf("github-pr-%s-%s-%d", repo.Owner, repo.Name, p.Number), RepositoryID: repo.ID, Kind: "pull_request", ExternalID: fmt.Sprint(p.Number), Title: p.Title, URL: p.HTMLURL, State: p.State, Author: p.User.Login, UpdatedAt: p.UpdatedAt})
	}
	return repo, artifacts, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "clankspace/0.1")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
