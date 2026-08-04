package httpapi_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/githubsync"
	"github.com/ShamanicArts/clankspace/internal/httpapi"
	"github.com/ShamanicArts/clankspace/internal/service"
	"github.com/ShamanicArts/clankspace/internal/store"
)

func TestSetupApprovalReturnsProjectCredentialOnce(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.EnsureBootstrap(t.Context(), "owner-token", "Workspace", "Owner"); err != nil {
		t.Fatal(err)
	}
	h := (&httpapi.Server{Store: db, Core: service.New(db), GitHub: githubsync.New(""), BaseURL: "https://clank.example", Log: slog.New(slog.NewTextHandler(&strings.Builder{}, nil))}).Handler()
	verifier := "this-is-a-random-looking-test-verifier"
	digest := sha256.Sum256([]byte(verifier))
	startBody, _ := json.Marshal(map[string]string{
		"challenge": hex.EncodeToString(digest[:]), "projectSlug": "example-repo", "projectName": "Example Repo",
		"repositoryUrl": "https://github.com/example/repo", "agentName": "Example agents",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/setup/start", bytes.NewReader(startBody))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("start returned %d: %s", w.Code, w.Body.String())
	}
	var started struct {
		DeviceCode      string `json:"deviceCode"`
		UserCode        string `json:"userCode"`
		VerificationURL string `json:"verificationUrl"`
	}
	if err = json.NewDecoder(w.Body).Decode(&started); err != nil {
		t.Fatal(err)
	}
	if started.DeviceCode == "" || started.UserCode == "" || !strings.Contains(started.VerificationURL, started.UserCode) {
		t.Fatalf("incomplete setup start: %#v", started)
	}
	approveBody, _ := json.Marshal(map[string]string{"userCode": started.UserCode})
	r = httptest.NewRequest(http.MethodPost, "/api/v1/setup/approve", bytes.NewReader(approveBody))
	r.Header.Set("Authorization", "Bearer owner-token")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("approve returned %d: %s", w.Code, w.Body.String())
	}
	exchangeBody, _ := json.Marshal(map[string]string{"deviceCode": started.DeviceCode, "verifier": verifier})
	r = httptest.NewRequest(http.MethodPost, "/api/v1/setup/exchange", bytes.NewReader(exchangeBody))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("exchange returned %d: %s", w.Code, w.Body.String())
	}
	var exchanged struct {
		Status  string         `json:"status"`
		Project domain.Project `json:"project"`
		Token   string         `json:"token"`
	}
	if err = json.NewDecoder(w.Body).Decode(&exchanged); err != nil {
		t.Fatal(err)
	}
	if exchanged.Status != "approved" || exchanged.Project.Slug != "example-repo" || exchanged.Token == "" {
		t.Fatalf("unexpected exchange: %#v", exchanged)
	}
	r = httptest.NewRequest(http.MethodGet, "/api/v1/whoami", nil)
	r.Header.Set("Authorization", "Bearer "+exchanged.Token)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("issued token is unusable: %d %s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest(http.MethodGet, "/api/v1/projects/example-repo/repositories", nil)
	r.Header.Set("Authorization", "Bearer "+exchanged.Token)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "https://github.com/example/repo") {
		t.Fatalf("setup did not link the inferred repository: %d %s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest(http.MethodPost, "/api/v1/setup/exchange", bytes.NewReader(exchangeBody))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("credential should only be returned once, got %d", w.Code)
	}
}

func TestHealthStaticAndAuthentication(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.EnsureBootstrap(t.Context(), "token", "Workspace", "Owner"); err != nil {
		t.Fatal(err)
	}
	h := (&httpapi.Server{Store: db, Core: service.New(db), GitHub: githubsync.New(""), Log: slog.New(slog.NewTextHandler(&strings.Builder{}, nil))}).Handler()
	for _, path := range []string{"/", "/healthz", "/docs/", "/docs/docs.js"} {
		r := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s returned %d", path, w.Code)
		}
		if path == "/" {
			body := w.Body.String()
			if !strings.Contains(body, "Project log") {
				t.Fatal("dashboard should open around the project append log")
			}
			if !strings.Contains(body, "Copy agent setup prompt") || !strings.Contains(body, "Sign in") {
				t.Fatal("unauthenticated home should offer agent setup and a clear sign-in path")
			}
			if strings.Contains(body, "Before your agent changes the code") {
				t.Fatal("dashboard must not present the agent coordination check as a human workflow")
			}
		}
		if path == "/docs/" {
			body := w.Body.String()
			if !strings.Contains(body, "Connect a repository") || !strings.Contains(body, "Give an agent access") || !strings.Contains(body, "data-copy=\"agent-prompt\"") {
				t.Fatal("docs should lead humans through access, repository connection, and agent onboarding")
			}
			if strings.Contains(body, "Project token: clank_") {
				t.Fatal("docs must not embed a project credential")
			}
		}
		if path == "/docs/docs.js" && !strings.Contains(w.Body.String(), "Never ask me to paste a project token into chat") {
			t.Fatal("generated agent prompt should preserve the credential boundary")
		}
	}
	r := httptest.NewRequest("GET", "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wanted 401, got %d", w.Code)
	}
	r = httptest.NewRequest("GET", "/api/v1/projects", nil)
	r.Header.Set("Authorization", "Bearer token")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("wanted 200, got %d: %s", w.Code, w.Body.String())
	}
	var projects struct {
		Projects []domain.Project `json:"projects"`
	}
	if err := json.NewDecoder(w.Body).Decode(&projects); err != nil {
		t.Fatal(err)
	}
	if projects.Projects == nil {
		t.Fatal("empty projects must encode as [] rather than null")
	}
}

func TestProjectExportIsNotDashboardCapped(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	p, err := db.EnsureBootstrap(t.Context(), "token", "Workspace", "Owner")
	if err != nil {
		t.Fatal(err)
	}
	core := service.New(db)
	project, _, err := core.CreateProject(t.Context(), p, "project", "large", "Large", "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 105; i++ {
		_, _, err = core.CreateNote(t.Context(), p, project.ID, fmt.Sprintf("note-%d", i), domain.CreateNoteInput{Kind: "observation", Title: fmt.Sprintf("Note %d", i), Summary: "Material project context", LedBy: "agent", DirectionBasis: "autonomous_agent_judgment"})
		if err != nil {
			t.Fatal(err)
		}
	}
	h := (&httpapi.Server{Store: db, Core: core, GitHub: githubsync.New(""), Log: slog.New(slog.NewTextHandler(&strings.Builder{}, nil))}).Handler()
	r := httptest.NewRequest("GET", "/api/v1/projects/large/export", nil)
	r.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("wanted 200, got %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		Notes []domain.Note `json:"notes"`
	}
	if err = json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Notes) != 105 {
		t.Fatalf("export truncated notes: got %d", len(out.Notes))
	}
}

func TestProjectRunsAreVisibleToScopedClients(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	owner, err := db.EnsureBootstrap(t.Context(), "token", "Workspace", "Owner")
	if err != nil {
		t.Fatal(err)
	}
	core := service.New(db)
	project, _, err := core.CreateProject(t.Context(), owner, "project", "runs", "Runs", "")
	if err != nil {
		t.Fatal(err)
	}
	credential, _, err := core.IssueProjectToken(t.Context(), owner, project.ID, "credential", "Test agents")
	if err != nil {
		t.Fatal(err)
	}
	principal, err := db.Authenticate(t.Context(), credential.Token)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = core.StartRun(t.Context(), principal, "run", domain.StartRunInput{ProjectID: project.ID, AgentName: "Luna", Model: "gpt-5.6-luna"}); err != nil {
		t.Fatal(err)
	}
	h := (&httpapi.Server{Store: db, Core: core, GitHub: githubsync.New(""), Log: slog.New(slog.NewTextHandler(&strings.Builder{}, nil))}).Handler()
	r := httptest.NewRequest("GET", "/api/v1/projects/runs/runs?limit=10", nil)
	r.Header.Set("Authorization", "Bearer "+credential.Token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("wanted 200, got %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		Runs []domain.Run `json:"runs"`
	}
	if err = json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Runs) != 1 || out.Runs[0].AgentName != "Luna" {
		t.Fatalf("unexpected runs: %#v", out.Runs)
	}
}
