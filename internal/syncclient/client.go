package syncclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/store"
)

type Client struct {
	HTTP *http.Client
}

func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func validateRemoteURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return errors.New("replica URL is invalid")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme != "http" {
		return errors.New("replica URL must use HTTPS")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" {
		return nil
	}
	addresses, err := net.LookupIP(host)
	if err != nil {
		return errors.New("plain HTTP replica URLs are allowed only on loopback or private networks")
	}
	tailscaleRange := netip.MustParsePrefix("100.64.0.0/10")
	for _, address := range addresses {
		parsedAddress, ok := netip.AddrFromSlice(address)
		if ok && (parsedAddress.IsLoopback() || parsedAddress.IsPrivate() || tailscaleRange.Contains(parsedAddress.Unmap())) {
			return nil
		}
	}
	return errors.New("replica credentials require HTTPS outside loopback or private networks")
}

func (c *Client) post(ctx context.Context, baseURL, path, token string, input, output any) error {
	if err := validateRemoteURL(baseURL); err != nil {
		return err
	}
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/v1"+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var problem struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(response.Body).Decode(&problem)
		if problem.Error == "" {
			problem.Error = response.Status
		}
		return errors.New(problem.Error)
	}
	return json.NewDecoder(response.Body).Decode(output)
}

func (c *Client) Join(ctx context.Context, db *store.Store, remoteURL, code, replicaName, localBaseURL, localUserID string) (domain.Workspace, error) {
	local, err := db.EnsureInstallationIdentity(ctx, replicaName, localBaseURL)
	if err != nil {
		return domain.Workspace{}, err
	}
	claim := store.ReplicaClaim{ReplicaID: local.ID, DisplayName: local.DisplayName, BaseURL: local.BaseURL, PublicKey: local.PublicKey, Capabilities: []string{"pull", "push"}}
	var result store.ReplicaClaimResult
	if err = c.post(ctx, remoteURL, "/sync/pair/claim", "", map[string]any{"code": code, "claim": claim}, &result); err != nil {
		return domain.Workspace{}, err
	}
	if err = db.ImportWorkspaceSnapshot(ctx, result.Snapshot, result.Authority, localUserID); err != nil {
		return domain.Workspace{}, fmt.Errorf("import workspace snapshot: %w", err)
	}
	if err = db.SaveReplicaLink(ctx, result.Workspace.ID, result.Authority.ID, remoteURL, result.Credential); err != nil {
		return domain.Workspace{}, err
	}
	return result.Workspace, nil
}

func (c *Client) Mirror(ctx context.Context, db *store.Store, workspaceID, cloudURL, code string) (domain.Workspace, error) {
	authority, err := db.IsWorkspaceAuthority(ctx, workspaceID)
	if err != nil || !authority {
		if err == nil {
			err = errors.New("workspace mirror must be started on its authority replica")
		}
		return domain.Workspace{}, err
	}
	snapshot, err := db.BuildWorkspaceSnapshot(ctx, workspaceID)
	if err != nil {
		return domain.Workspace{}, err
	}
	replicas, err := db.ListReplicas(ctx, workspaceID)
	if err != nil {
		return domain.Workspace{}, err
	}
	var localAuthority domain.Replica
	for _, replica := range replicas {
		if replica.Role == "authority" {
			localAuthority = replica
			break
		}
	}
	if localAuthority.ID == "" {
		return domain.Workspace{}, errors.New("workspace authority identity is missing")
	}
	var result store.MirrorClaimResult
	if err = c.post(ctx, cloudURL, "/sync/mirror/claim", "", map[string]any{"code": code, "authority": localAuthority, "snapshot": snapshot}, &result); err != nil {
		return domain.Workspace{}, err
	}
	if err = db.ImportReplicaRoster(ctx, workspaceID, localAuthority.ID, []domain.Replica{localAuthority, result.Replica}); err != nil {
		return domain.Workspace{}, err
	}
	if err = db.SaveReplicaLink(ctx, workspaceID, result.Replica.ID, cloudURL, result.Credential); err != nil {
		return domain.Workspace{}, err
	}
	return result.Workspace, nil
}

type pullResponse struct {
	Events   []domain.DomainEvent `json:"events"`
	Heads    []domain.SyncHead    `json:"heads"`
	Replicas []domain.Replica     `json:"replicas"`
	More     bool                 `json:"more"`
}

type pushResponse struct {
	Imported int               `json:"imported"`
	Heads    []domain.SyncHead `json:"heads"`
}

func sequenceFor(heads []domain.SyncHead, originID string) int64 {
	for _, head := range heads {
		if head.OriginReplicaID == originID {
			return head.Sequence
		}
	}
	return 0
}

func (c *Client) SyncLink(ctx context.Context, db *store.Store, link store.ReplicaLink) error {
	for {
		localHeads, err := db.SyncHeads(ctx, link.WorkspaceID)
		if err != nil {
			return err
		}
		var pulled pullResponse
		if err = c.post(ctx, link.RemoteURL, "/sync/workspaces/"+link.WorkspaceID+"/pull", link.Credential, map[string]any{"heads": localHeads, "limit": 500}, &pulled); err != nil {
			return err
		}
		authorityID, authorityErr := db.WorkspaceAuthorityID(ctx, link.WorkspaceID)
		if authorityErr != nil {
			return authorityErr
		}
		if link.RemoteReplicaID == authorityID {
			if err = db.ImportReplicaRoster(ctx, link.WorkspaceID, link.RemoteReplicaID, pulled.Replicas); err != nil {
				return err
			}
		}
		if _, err = db.ImportEvents(ctx, link.WorkspaceID, pulled.Events); err != nil {
			return fmt.Errorf("import pulled events: %w", err)
		}
		if !pulled.More {
			remoteHeads := pulled.Heads
			localOrigin := db.LocalReplicaID()
			for {
				events, eventErr := db.EventsForOriginAfter(ctx, link.WorkspaceID, localOrigin, sequenceFor(remoteHeads, localOrigin), 500)
				if eventErr != nil {
					return eventErr
				}
				if len(events) == 0 {
					return nil
				}
				var pushed pushResponse
				if eventErr = c.post(ctx, link.RemoteURL, "/sync/workspaces/"+link.WorkspaceID+"/push", link.Credential, map[string]any{"events": events}, &pushed); eventErr != nil {
					return eventErr
				}
				remoteHeads = pushed.Heads
				if len(events) < 500 {
					return nil
				}
			}
		}
	}
}

func (c *Client) SyncAll(ctx context.Context, db *store.Store) error {
	links, err := db.ListReplicaLinks(ctx)
	if err != nil {
		return err
	}
	var failures []string
	for _, link := range links {
		syncErr := c.SyncLink(ctx, db, link)
		_ = db.RecordLinkResult(ctx, link.ID, syncErr)
		if syncErr != nil {
			failures = append(failures, link.RemoteURL+": "+syncErr.Error())
		}
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

func Run(ctx context.Context, db *store.Store, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	client := New()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		_ = client.SyncAll(ctx, db)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
