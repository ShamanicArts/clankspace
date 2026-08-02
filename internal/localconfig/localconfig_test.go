package localconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveProjectAndStoredCredential(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "src", "feature")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	projectBody, _ := json.Marshal(ProjectFile{URL: "http://127.0.0.1:8091/", Project: "demo"})
	if err := os.WriteFile(filepath.Join(root, ".clankspace.json"), projectBody, 0600); err != nil {
		t.Fatal(err)
	}
	credentials := filepath.Join(t.TempDir(), "credentials.json")
	t.Setenv("CLANKSPACE_CREDENTIALS_FILE", credentials)
	t.Setenv("CLANKSPACE_URL", "")
	t.Setenv("CLANKSPACE_PROJECT", "")
	t.Setenv("CLANKSPACE_TOKEN", "")
	if err := StoreCredential(credentials, "http://127.0.0.1:8091", "demo", "project-token"); err != nil {
		t.Fatal(err)
	}

	got, err := Resolve(nested)
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "http://127.0.0.1:8091/" || got.Project != "demo" || got.Token != "project-token" {
		t.Fatalf("unexpected resolution: %#v", got)
	}
	if got.TokenSource != "credential_store" || got.ProjectFilePath != filepath.Join(root, ".clankspace.json") {
		t.Fatalf("missing sources: %#v", got)
	}
	info, err := os.Stat(credentials)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("credential mode is %o", info.Mode().Perm())
	}
}

func TestEnvironmentOverridesProjectFileAndCredential(t *testing.T) {
	root := t.TempDir()
	projectBody, _ := json.Marshal(ProjectFile{URL: "http://file", Project: "file-project"})
	if err := os.WriteFile(filepath.Join(root, ".clankspace.json"), projectBody, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLANKSPACE_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("CLANKSPACE_URL", "http://environment")
	t.Setenv("CLANKSPACE_PROJECT", "environment-project")
	t.Setenv("CLANKSPACE_TOKEN", "environment-token")

	got, err := Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "http://environment" || got.Project != "environment-project" || got.Token != "environment-token" || got.TokenSource != "environment" {
		t.Fatalf("unexpected resolution: %#v", got)
	}
}

func TestStoreCredentialUpdatesOnlyMatchingProject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	if err := StoreCredential(path, "http://local", "one", "first"); err != nil {
		t.Fatal(err)
	}
	if err := StoreCredential(path, "http://local", "two", "second"); err != nil {
		t.Fatal(err)
	}
	if err := StoreCredential(path, "http://local/", "one", "updated"); err != nil {
		t.Fatal(err)
	}
	file, err := readCredentials(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Credentials) != 2 {
		t.Fatalf("wanted two credentials, got %d", len(file.Credentials))
	}
	one, _ := loadCredential(path, "http://local", "one")
	two, _ := loadCredential(path, "http://local", "two")
	if one != "updated" || two != "second" {
		t.Fatalf("unexpected credentials: one=%q two=%q", one, two)
	}
}
