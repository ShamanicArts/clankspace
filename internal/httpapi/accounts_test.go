package httpapi_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/githubsync"
	"github.com/ShamanicArts/clankspace/internal/httpapi"
	"github.com/ShamanicArts/clankspace/internal/service"
	"github.com/ShamanicArts/clankspace/internal/store"
)

func tokenFromMail(t *testing.T, body string) string {
	t.Helper()
	marker := "/?token="
	index := strings.Index(body, marker)
	if index < 0 {
		t.Fatalf("mail has no auth link: %q", body)
	}
	value := body[index+len(marker):]
	if end := strings.IndexByte(value, '\n'); end >= 0 {
		value = value[:end]
	}
	decoded, err := url.QueryUnescape(strings.TrimSpace(value))
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func jsonRequest(t *testing.T, client *http.Client, method, address string, input any, origin, csrf string) (*http.Response, map[string]any) {
	t.Helper()
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(method, address, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var output map[string]any
	_ = json.NewDecoder(response.Body).Decode(&output)
	response.Body.Close()
	return response, output
}

func csrfFromJar(t *testing.T, jar http.CookieJar, address string) string {
	t.Helper()
	parsed, _ := url.Parse(address)
	for _, cookie := range jar.Cookies(parsed) {
		if cookie.Name == "clank_csrf" {
			return cookie.Value
		}
	}
	t.Fatal("CSRF cookie missing")
	return ""
}

func TestHostedHTTPInviteSessionAndWorkspaceFlow(t *testing.T) {
	ctx := t.Context()
	db, err := store.OpenWithSecret(filepath.Join(t.TempDir(), "clankspace.db"), "hosted-http-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ownerPrincipal, err := db.EnsureBootstrap(ctx, "bootstrap-token", "Shared", "Shamanic")
	if err != nil {
		t.Fatal(err)
	}
	owner, ownerMembership, err := db.ClaimBootstrapOwner(ctx, ownerPrincipal.ID, "shamanic@example.test", "Shamanic")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.EnsureInstallationIdentity(ctx, "Hosted", "http://hosted.test"); err != nil {
		t.Fatal(err)
	}
	if err = db.EnsureAllWorkspaceAuthorities(ctx); err != nil {
		t.Fatal(err)
	}
	serverConfig := &httpapi.Server{Store: db, Core: service.New(db), GitHub: githubsync.New(""), Log: slog.Default(), AuthMode: "hybrid", SyncEnabled: true, ReplicaName: "Hosted"}
	server := httptest.NewServer(serverConfig.Handler())
	defer server.Close()
	serverConfig.BaseURL = server.URL

	ownerJar, _ := cookiejar.New(nil)
	ownerClient := &http.Client{Jar: ownerJar}
	response, _ := jsonRequest(t, ownerClient, http.MethodPost, server.URL+"/api/v1/auth/magic-link", map[string]string{"email": owner.Email}, "", "")
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("magic link status = %d", response.StatusCode)
	}
	messages, err := db.ClaimOutbox(ctx, 10)
	if err != nil || len(messages) != 1 {
		t.Fatalf("outbox = %#v, %v", messages, err)
	}
	response, _ = jsonRequest(t, ownerClient, http.MethodPost, server.URL+"/api/v1/auth/consume", map[string]string{"token": tokenFromMail(t, messages[0].Body)}, "", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("consume status = %d", response.StatusCode)
	}
	csrf := csrfFromJar(t, ownerJar, server.URL)
	response, _ = jsonRequest(t, ownerClient, http.MethodPost, server.URL+"/api/v1/account/workspaces", map[string]string{"slug": "second-space", "name": "Second Space"}, server.URL, "")
	if response.StatusCode != http.StatusUnauthorized && response.StatusCode != http.StatusForbidden {
		t.Fatalf("missing CSRF status = %d", response.StatusCode)
	}
	response, created := jsonRequest(t, ownerClient, http.MethodPost, server.URL+"/api/v1/account/workspaces", map[string]string{"slug": "second-space", "name": "Second Space"}, server.URL, csrf)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("workspace create = %d %#v", response.StatusCode, created)
	}
	workspace := created["workspace"].(map[string]any)
	if workspace["slug"] != "second-space" {
		t.Fatalf("workspace = %#v", workspace)
	}

	invite, err := db.CreateWorkspaceInvite(ctx, ownerMembership, "shuv@example.test", "member", server.URL)
	if err != nil || invite.Email != "shuv@example.test" {
		t.Fatal(err)
	}
	messages, err = db.ClaimOutbox(ctx, 10)
	if err != nil || len(messages) < 2 {
		t.Fatalf("invite outbox = %#v, %v", messages, err)
	}
	var inviteBody string
	for _, message := range messages {
		if message.Template == "workspace_invite" {
			inviteBody = message.Body
		}
	}
	shuvJar, _ := cookiejar.New(nil)
	shuvClient := &http.Client{Jar: shuvJar}
	response, result := jsonRequest(t, shuvClient, http.MethodPost, server.URL+"/api/v1/auth/consume", map[string]string{"token": tokenFromMail(t, inviteBody), "displayName": "Shuv"}, "", "")
	if response.StatusCode != http.StatusOK || result["user"].(map[string]any)["displayName"] != "Shuv" {
		t.Fatalf("invite consume = %d %#v", response.StatusCode, result)
	}
	request, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/account", nil)
	response, err = shuvClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var account struct {
		User       domain.User        `json:"user"`
		Workspaces []domain.Workspace `json:"workspaces"`
	}
	if err = json.NewDecoder(response.Body).Decode(&account); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if len(account.Workspaces) != 1 || account.Workspaces[0].ID != ownerMembership.WorkspaceID {
		t.Fatalf("invited account leaked workspaces: %#v", account)
	}
}
