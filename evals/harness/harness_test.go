package harness_test

import (
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ShamanicArts/clankspace/evals/harness"
	"github.com/ShamanicArts/clankspace/internal/client"
	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/githubsync"
	"github.com/ShamanicArts/clankspace/internal/httpapi"
	"github.com/ShamanicArts/clankspace/internal/localconfig"
	"github.com/ShamanicArts/clankspace/internal/service"
	"github.com/ShamanicArts/clankspace/internal/store"
)

func fixture(t *testing.T) (harness.Scenario, []byte) {
	t.Helper()
	path := filepath.Join("..", "fixtures", "rendered", "relaydesk-001.json")
	scenario, canonical, err := harness.LoadScenario(path)
	if err != nil {
		t.Fatal(err)
	}
	return scenario, canonical
}

func TestRelayDeskFixtureValidatesAndBuildsReproducibleRepository(t *testing.T) {
	scenario, _ := fixture(t)
	skill := filepath.Join(t.TempDir(), "SKILL.md")
	if err := os.WriteFile(skill, []byte("# ClankSpace\n\nCheck project intent.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	build := func(destination string) string {
		head, _, err := harness.BuildRepository(scenario, harness.RepositoryOptions{
			Destination: destination, ProjectURL: "https://eval.example.test", ProjectSlug: "eval-relaydesk",
			SkillPath: skill,
		})
		if err != nil {
			t.Fatal(err)
		}
		return head
	}
	first, second := build(filepath.Join(t.TempDir(), "repo")), build(filepath.Join(t.TempDir(), "repo"))
	if first != second {
		t.Fatalf("repository heads differ: %s != %s", first, second)
	}
	if first == "" {
		t.Fatal("repository head is empty")
	}
	resumeDir := filepath.Join(t.TempDir(), "repo")
	resumeHead := build(resumeDir)
	agentInstructions, err := os.ReadFile(filepath.Join(resumeDir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"without reading the skill", "Future-tense constraints", "passive until the human asks you to act"} {
		if !strings.Contains(string(agentInstructions), required) {
			t.Fatalf("generated AGENTS.md omitted passive bootstrap rule %q: %s", required, agentInstructions)
		}
	}
	if got := build(resumeDir); got != resumeHead {
		t.Fatalf("resumed repository head = %s, want %s", got, resumeHead)
	}
}

func TestSanitizedSnapshotContainsTrackedHeadWithoutSourceHistory(t *testing.T) {
	scenario, _ := fixture(t)
	skill := filepath.Join(t.TempDir(), "SKILL.md")
	if err := os.WriteFile(skill, []byte("# ClankSpace\n"), 0600); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "source")
	if _, _, err := harness.BuildRepository(scenario, harness.RepositoryOptions{
		Destination: source, ProjectURL: "https://eval.example.test", ProjectSlug: "source", SkillPath: skill,
	}); err != nil {
		t.Fatal(err)
	}
	result, err := harness.CreateSanitizedSnapshot("relaydesk-main-2026-08-02", source, "HEAD", t.TempDir(), []string{"src", "README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceHead == "" || result.SnapshotHead == "" || result.Bundle == "" {
		t.Fatalf("incomplete snapshot: %+v", result)
	}
	count, err := exec.Command("git", "-C", result.Repository, "rev-list", "--count", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(count)) != "1" {
		t.Fatalf("sanitized snapshot retained source history: %s", count)
	}
}

func TestBuildRealSnapshotAppliesOverlayAndStartsClean(t *testing.T) {
	scenario, _ := fixture(t)
	skill := filepath.Join(t.TempDir(), "SKILL.md")
	if err := os.WriteFile(skill, []byte("# ClankSpace\n"), 0600); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "source")
	if _, _, err := harness.BuildRepository(scenario, harness.RepositoryOptions{
		Destination: source, ProjectURL: "https://eval.example.test", ProjectSlug: "source", SkillPath: skill,
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := harness.CreateSanitizedSnapshot("relaydesk-main-2026-08-02", source, "HEAD", t.TempDir(), []string{"src", "README.md"})
	if err != nil {
		t.Fatal(err)
	}
	real := scenario
	real.Project.RepositoryProfile = "real-snapshot"
	real.Repository.SnapshotID = snapshot.ID
	real.Repository.Commits = []harness.CommitSpec{{
		ID: "synthetic-overlay", Message: "test: add overlay", AuthorName: "Morgan", AuthorEmail: "morgan@example.test",
		Changes: []harness.FileChange{{Path: "src/overlay.js", Content: "export const overlay = true\n"}},
	}}
	destination := filepath.Join(t.TempDir(), "world")
	if _, _, err = harness.BuildRepository(real, harness.RepositoryOptions{
		Destination: destination, ProjectURL: "https://eval.example.test", ProjectSlug: "eval-real", SkillPath: skill,
		SnapshotSources: map[string]string{snapshot.ID: snapshot.Repository},
	}); err != nil {
		t.Fatal(err)
	}
	status, err := exec.Command("git", "-C", destination, "status", "--short").Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(status)) != "" {
		t.Fatalf("real snapshot world is dirty: %s", status)
	}
	if _, err = os.Stat(filepath.Join(destination, "src", "overlay.js")); err != nil {
		t.Fatal("synthetic overlay was not applied")
	}
}

func TestScenarioRejectsHarnessOwnedPaths(t *testing.T) {
	scenario, _ := fixture(t)
	scenario.Repository.Commits[0].Changes[0].Path = "AGENTS.md"
	if err := scenario.Validate(); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("expected reserved path failure, got %v", err)
	}
}

func TestCreateLayoutIsContentAddressedAndImmutable(t *testing.T) {
	scenario, canonical := fixture(t)
	root := t.TempDir()
	first, hash, err := harness.CreateLayout(root, "v1", scenario, canonical)
	if err != nil {
		t.Fatal(err)
	}
	second, secondHash, err := harness.CreateLayout(root, "v1", scenario, canonical)
	if err != nil {
		t.Fatal(err)
	}
	if first.Root != second.Root || hash != secondHash {
		t.Fatal("identical scenario did not resolve to the same immutable layout")
	}
	if err = harness.WritePrepared(first, harness.PreparedWorld{SchemaVersion: 1, ScenarioID: scenario.ID}); err != nil {
		t.Fatal(err)
	}
	if err = harness.WritePrepared(first, harness.PreparedWorld{SchemaVersion: 1, ScenarioID: scenario.ID}); err == nil {
		t.Fatal("prepared artifact was overwritten")
	}
}

func TestIngestWorldWorkflowPinsRunProvenanceAndAcceptedScenario(t *testing.T) {
	scenario, _ := fixture(t)
	envelope := map[string]any{
		"runId": "wf_test123", "status": "completed",
		"result": map[string]any{"accepted": []harness.Scenario{scenario}},
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(t.TempDir(), "workflow.json")
	if err = os.WriteFile(input, body, 0600); err != nil {
		t.Fatal(err)
	}
	result, err := harness.IngestWorldWorkflow(input, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkflowRun != "wf_test123" || len(result.Accepted) != 1 {
		t.Fatalf("unexpected ingest result: %+v", result)
	}
	ingested, _, err := harness.LoadScenario(result.Accepted[0])
	if err != nil {
		t.Fatal(err)
	}
	if ingested.Generation.WorkflowRun != "wf_test123" {
		t.Fatalf("workflow provenance missing: %+v", ingested.Generation)
	}
}

func TestSeedScenarioCreatesIsolatedProjectAndAliasMap(t *testing.T) {
	scenario, canonical := fixture(t)
	db, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.EnsureBootstrap(t.Context(), "admin-token", "Eval", "Owner"); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer((&httpapi.Server{
		Store: db, Core: service.New(db), GitHub: githubsync.New(""),
		Log: slog.New(slog.NewTextHandler(&strings.Builder{}, nil)),
	}).Handler())
	defer server.Close()
	admin := client.New(server.URL, "admin-token")
	if _, err = admin.CreateProject(t.Context(), "synthetic-lab", "Synthetic lab", "Evaluation control log"); err != nil {
		t.Fatal(err)
	}
	layout, hash, err := harness.CreateLayout(t.TempDir(), "v1", scenario, canonical)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := harness.SeedScenario(t.Context(), scenario, harness.SeedOptions{
		URL: server.URL, AdminToken: "admin-token", ControlProject: "synthetic-lab",
		Layout: layout, ScenarioHash: hash, CorpusVersion: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.ProjectSlug == "synthetic-lab" || prepared.ProjectID == "" {
		t.Fatalf("scenario was not isolated: %+v", prepared)
	}
	if len(prepared.RecordIDs) != len(scenario.Records) || len(prepared.TrajectoryIDs) != len(scenario.Trajectories) {
		t.Fatalf("alias maps incomplete: %+v", prepared)
	}
	if info, err := os.Stat(prepared.CredentialFile); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("credential file is not private: info=%v err=%v", info, err)
	}
	t.Setenv("CLANKSPACE_CREDENTIALS_FILE", prepared.CredentialFile)
	t.Setenv("CLANKSPACE_URL", server.URL)
	t.Setenv("CLANKSPACE_PROJECT", prepared.ProjectSlug)
	resolved, err := localconfig.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	brief, err := client.New(server.URL, resolved.Token).Brief(t.Context(), prepared.ProjectSlug, domain.BriefInput{Objective: scenario.Task.Objective, Paths: scenario.Task.Paths, CheckConflicts: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(brief.Trajectories) != 1 || len(brief.Warnings) != 1 {
		t.Fatalf("active trajectory was not preserved: trajectories=%d warnings=%d", len(brief.Trajectories), len(brief.Warnings))
	}
	prepared.RepositoryHead = "deadbeef"
	if err = harness.TrackPrepared(t.Context(), prepared, harness.SeedOptions{
		URL: server.URL, AdminToken: "admin-token", ControlProject: "synthetic-lab", ScenarioHash: hash,
	}); err != nil {
		t.Fatal(err)
	}
	export, err := admin.ExportProject(t.Context(), "synthetic-lab")
	if err != nil {
		t.Fatal(err)
	}
	notes, ok := export["notes"].([]any)
	if !ok || len(notes) != 1 {
		t.Fatalf("control project checkpoint missing: %#v", export["notes"])
	}
}

func TestIsolationProbeDeniesForeignProjectAccess(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.EnsureBootstrap(t.Context(), "admin-token", "Eval", "Owner"); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer((&httpapi.Server{
		Store: db, Core: service.New(db), GitHub: githubsync.New(""),
		Log: slog.New(slog.NewTextHandler(&strings.Builder{}, nil)),
	}).Handler())
	defer server.Close()
	adminEnvironment := filepath.Join(t.TempDir(), "admin.env")
	if err = os.WriteFile(adminEnvironment, []byte("CLANKSPACE_URL="+server.URL+"\nCLANKSPACE_TOKEN=admin-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := harness.RunIsolationProbe(t.Context(), harness.IsolationProbeOptions{
		AdminEnvironment: adminEnvironment,
		ProbeID:          "unit-isolation",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.Leaked || len(result.VisibleProjects) != 1 || result.VisibleProjects[0] != result.ProjectA {
		t.Fatalf("unexpected isolation result: %+v", result)
	}
}
