package httpapi_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

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
