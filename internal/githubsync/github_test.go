package githubsync_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ShamanicArts/clankspace/internal/githubsync"
)

func TestParseAndSyncPublicRepository(t *testing.T) {
	repo, err := githubsync.ParseRepository("github.com/ShamanicArts/shuv2code.git")
	if err != nil {
		t.Fatal(err)
	}
	if repo.Owner != "ShamanicArts" || repo.Name != "shuv2code" {
		t.Fatalf("wrong repository: %#v", repo)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/ShamanicArts/shuv2code" {
			w.Write([]byte(`{"description":"sessions","default_branch":"main","stargazers_count":3,"private":false}`))
			return
		}
		if r.URL.Path == "/repos/ShamanicArts/shuv2code/pulls" {
			w.Write([]byte(`[{"number":42,"title":"Session control","html_url":"https://github.com/ShamanicArts/shuv2code/pull/42","state":"open","updated_at":"2026-08-02T00:00:00Z","user":{"login":"shuv"}}]`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	c := githubsync.New("")
	c.BaseURL = server.URL
	repo, pulls, err := c.Sync(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	if repo.Default != "main" || len(pulls) != 1 || pulls[0].ExternalID != "42" {
		t.Fatalf("incomplete sync: %#v %#v", repo, pulls)
	}
}

func TestParseRepositoryAcceptsGitSSHRemote(t *testing.T) {
	repo, err := githubsync.ParseRepository("git@github.com:ShamanicArts/clankspace.git")
	if err != nil {
		t.Fatal(err)
	}
	if repo.URL != "https://github.com/ShamanicArts/clankspace" {
		t.Fatalf("unexpected canonical URL %q", repo.URL)
	}
}

func TestRejectsNonGitHubAndPrivateShape(t *testing.T) {
	if _, err := githubsync.ParseRepository("https://example.com/a/b"); err == nil {
		t.Fatal("accepted non-GitHub URL")
	}
	if _, err := githubsync.ParseRepository("https://github.com/a/b/issues/1"); err == nil {
		t.Fatal("accepted non-repository URL")
	}
}
