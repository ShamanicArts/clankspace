package httpapi_test

import (
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
	for _, path := range []string{"/", "/healthz"} {
		r := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s returned %d", path, w.Code)
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
