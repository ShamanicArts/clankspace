package harness_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/ShamanicArts/clankspace/internal/githubsync"
	"github.com/ShamanicArts/clankspace/internal/httpapi"
	"github.com/ShamanicArts/clankspace/internal/service"
	"github.com/ShamanicArts/clankspace/internal/store"
)

func TestPrepareCollaborationSeedsOneProjectWithPrivateLaneCredentials(t *testing.T) {
	server, adminToken := collaborationSeedServer(t)
	defer server.Close()
	scenario, snapshot := collaborationScenarioWithSnapshot(t)
	canonical, err := json.MarshalIndent(scenario, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	canonical = append(canonical, '\n')
	ledger := t.TempDir()
	layout, hash, err := harness.CreateCollaborationLayout(ledger, "v2", scenario, canonical)
	if err != nil {
		t.Fatal(err)
	}
	if err = harness.VerifyCollaborationSourceEvidence(scenario, snapshot.Repository); err != nil {
		t.Fatal(err)
	}
	skill := filepath.Join(t.TempDir(), "SKILL.md")
	if err = os.WriteFile(skill, []byte("# test skill\n"), 0600); err != nil {
		t.Fatal(err)
	}
	head, skillHash, err := harness.BuildRepository(harness.Scenario{Project: scenario.Project, Repository: scenario.Repository}, harness.RepositoryOptions{Destination: layout.RepositoryPath, ProjectURL: server.URL, ProjectSlug: harness.ProjectSlugForScenario(scenario.ID, "v2", hash), SkillPath: skill, SnapshotSources: map[string]string{snapshot.ID: snapshot.Repository}})
	if err != nil {
		t.Fatal(err)
	}
	options := harness.SeedOptions{URL: server.URL, AdminToken: adminToken, Layout: layout, ScenarioHash: hash, CorpusVersion: "v2"}
	prepared, err := harness.SeedCollaborationScenario(context.Background(), scenario, options)
	if err != nil {
		t.Fatal(err)
	}
	prepared.RepositoryHead, prepared.SkillHash = head, skillHash
	for i := range prepared.Lanes {
		prepared.Lanes[i].RepositoryHead, prepared.Lanes[i].SkillHash = head, skillHash
	}
	if err = prepared.Validate(); err != nil {
		t.Fatal(err)
	}
	if err = harness.WriteCollaborationPrepared(layout, prepared); err != nil {
		t.Fatal(err)
	}
	if len(prepared.Lanes) != 2 || prepared.Lanes[0].PrincipalID == prepared.Lanes[1].PrincipalID {
		t.Fatalf("lanes did not receive distinct principals: %+v", prepared.Lanes)
	}
	for _, lane := range scenario.Lanes {
		credential := filepath.Join(layout.SecretsPath, lane.ID+".json")
		if info, statErr := os.Stat(credential); statErr != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("credential %s is not mode 0600: info=%v err=%v", lane.ID, info, statErr)
		}
		artifact, readErr := os.ReadFile(filepath.Join(layout.Root, "lanes", lane.ID+".json"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if strings.Contains(string(artifact), "ledgerOracle") || strings.Contains(string(artifact), "materialReason") {
			t.Fatalf("lane artifact leaked ledger oracle: %s", artifact)
		}
	}
	body, err := os.ReadFile(layout.PreparedPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(body)), "token") || strings.Contains(strings.ToLower(string(body)), "credential") {
		t.Fatalf("prepared artifact leaked a credential reference: %s", body)
	}
	// The same scenario hash returns the same project and issued lane identities;
	// no resume operation may make a second project or rewrite credentials.
	resumed, err := harness.SeedCollaborationScenario(context.Background(), scenario, options)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.ProjectID != prepared.ProjectID || resumed.Lanes[0].PrincipalID != prepared.Lanes[0].PrincipalID || resumed.Lanes[1].PrincipalID != prepared.Lanes[1].PrincipalID {
		t.Fatalf("resume changed prepared identities: first=%+v resumed=%+v", prepared, resumed)
	}
	export, err := client.New(server.URL, adminToken).ExportProject(context.Background(), prepared.ProjectSlug)
	if err != nil {
		t.Fatal(err)
	}
	if len(export["notes"].([]any)) != len(scenario.Records) || len(export["trajectories"].([]any)) != len(scenario.Trajectories) {
		t.Fatalf("seeded project records are incomplete: %#v", export)
	}
}

func TestCollaborationSourceEvidenceRejectsManifestMismatches(t *testing.T) {
	scenario, snapshot := collaborationScenarioWithSnapshot(t)
	tests := []struct {
		name   string
		mutate func(*harness.CollaborationScenario)
	}{
		{"source commit", func(s *harness.CollaborationScenario) { s.SourceEvidence.SourceCommit = strings.Repeat("0", 40) }},
		{"snapshot head", func(s *harness.CollaborationScenario) { s.SourceEvidence.SnapshotHead = strings.Repeat("1", 40) }},
		{"bundle hash", func(s *harness.CollaborationScenario) { s.SourceEvidence.BundleHash = strings.Repeat("2", 64) }},
		{"license", func(s *harness.CollaborationScenario) { s.SourceEvidence.License = "Apache-2.0" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			copy := scenario
			test.mutate(&copy)
			if err := harness.VerifyCollaborationSourceEvidence(copy, snapshot.Repository); err == nil {
				t.Fatal("mismatched source evidence was accepted")
			}
		})
	}
}

func collaborationSeedServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err = db.EnsureBootstrap(context.Background(), "admin-token", "Eval", "Owner"); err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer((&httpapi.Server{Store: db, Core: service.New(db), GitHub: githubsync.New(""), Log: slog.New(slog.NewTextHandler(&strings.Builder{}, nil))}).Handler()), "admin-token"
}

func collaborationScenarioWithSnapshot(t *testing.T) (harness.CollaborationScenario, harness.SnapshotResult) {
	t.Helper()
	source := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	git(t, source, "init", "--quiet", "--initial-branch=main")
	git(t, source, "remote", "add", "origin", "https://example.test/acme/collaboration.git")
	if err := os.WriteFile(filepath.Join(source, "LICENSE"), []byte("MIT License\n\nPermission is hereby granted\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "src", "coordination"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "src", "coordination", "main.go"), []byte("package coordination\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, source, "add", ".")
	git(t, source, "-c", "user.name=fixture", "-c", "user.email=fixture@example.test", "commit", "--quiet", "-m", "fixture")
	snapshot, err := harness.CreateSanitizedSnapshot("collaboration-source-001", source, "HEAD", t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	license, err := os.ReadFile(filepath.Join(snapshot.Repository, "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(snapshot.Bundle)
	if err != nil {
		t.Fatal(err)
	}
	scenario := harness.CollaborationScenario{SchemaVersion: 2, ID: "dev-collaboration-source-001", Split: "dev", Category: "event-gated-collaboration", Project: harness.ScenarioProject{Slug: "coordination-source", Name: "Coordination source", Description: "Isolated real snapshot for collaboration preparation tests.", RepositoryProfile: "real-snapshot", Paths: []string{"src/coordination/"}}, Repository: harness.RepositoryFixture{SnapshotID: snapshot.ID, BaseRef: snapshot.SnapshotHead}, SourceEvidence: harness.SourceEvidence{RepositoryURL: "https://example.test/acme/collaboration.git", License: "MIT", LicenseFile: "LICENSE", LicenseFileHash: hashBytes(license), SourceCommit: snapshot.SourceHead, SnapshotID: snapshot.ID, SnapshotHead: snapshot.SnapshotHead, BundleHash: hashBytes(bundle), SyntheticOverlay: true}, Actors: []harness.Actor{{Key: "agent-a", PrincipalName: "Lane A", AgentName: "Lane A agent", Harness: "codex-cli", Provider: "openai", Model: "test", Role: "primary"}, {Key: "agent-b", PrincipalName: "Lane B", AgentName: "Lane B agent", Harness: "codex-cli", Provider: "openai", Model: "test", Role: "primary"}}, Records: []harness.Record{{ID: "prior-record", ActorKey: "agent-a", Kind: "understanding", Title: "Prior context", Summary: "Context is shared through the project.", Rationale: "The dependent lane must retrieve it.", Status: "current", LedBy: "agent", DirectionBasis: "autonomous_agent_judgment", Paths: []string{"src/coordination/"}}}, Trajectories: []harness.Trajectory{{ID: "prior-trajectory", ActorKey: "agent-b", Objective: "Preserve coordination context.", Rationale: "The seeded direction is visible to later runs.", Status: "active", Paths: []string{"src/coordination/"}, Branch: "main"}}, Lanes: []harness.CollaborationLane{{ID: "lane-a", ActorKey: "agent-a", Branch: "eval/lane-a", PriorUserTurns: []harness.ConversationTurn{{Role: "user", Text: "Inspect first."}}, Task: harness.LaneTask{Objective: "Create the first checkpoint.", UserRequest: "Inspect and checkpoint.", Paths: []string{"src/coordination/"}, Checks: []string{"git diff --check"}}, LedgerOracle: harness.Oracle{ExpectedBehavior: "proceed", MaterialReason: "The barrier is a durable checkpoint."}}, {ID: "lane-b", ActorKey: "agent-b", Branch: "eval/lane-b", PriorUserTurns: []harness.ConversationTurn{{Role: "user", Text: "Inspect after the handoff."}}, Task: harness.LaneTask{Objective: "Complete dependent work.", UserRequest: "Retrieve and continue.", Paths: []string{"src/coordination/"}, Checks: []string{"git diff --check"}}, LedgerOracle: harness.Oracle{ExpectedBehavior: "inspect", MaterialReason: "The barrier must be observed."}}}, Schedule: harness.EventGatedSchedule{InitialLane: "lane-a", DependentLane: "lane-b", TimeoutSeconds: 1, PollIntervalMS: 50, Barrier: harness.BarrierSpec{EventType: "note.recorded", Kind: "checkpoint", RequiredPathOverlap: []string{"src/coordination/"}}}, Generation: harness.Generation{CurriculumVersion: "v2", Seed: "seed", GeneratorProvider: "test", GeneratorModel: "test"}}
	if err = scenario.Validate(); err != nil {
		t.Fatal(err)
	}
	return scenario, snapshot
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func hashBytes(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
