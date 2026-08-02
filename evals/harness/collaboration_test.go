package harness_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ShamanicArts/clankspace/evals/harness"
)

func collaborationFixture(t *testing.T) (harness.CollaborationScenario, []byte) {
	t.Helper()
	scenario, canonical, err := harness.LoadCollaborationScenario(filepath.Join("..", "fixtures", "collaboration", "two-lane-v2.json"))
	if err != nil {
		t.Fatal(err)
	}
	return scenario, canonical
}

func TestCollaborationFixtureValidatesAndV1LoaderStaysV1Only(t *testing.T) {
	scenario, canonical := collaborationFixture(t)
	if scenario.SchemaVersion != 2 || len(scenario.Lanes) != 2 || scenario.Schedule.InitialLane != "lane-a" {
		t.Fatalf("unexpected collaboration fixture: %+v", scenario)
	}
	if !strings.HasSuffix(string(canonical), "\n") || harness.ContentHash(canonical) == "" {
		t.Fatal("collaboration fixture did not canonicalize")
	}
	path := filepath.Join(t.TempDir(), "scenario.json")
	if err := os.WriteFile(path, canonical, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := harness.LoadScenario(path); err == nil || !strings.Contains(err.Error(), "schemaVersion must be 1") {
		t.Fatalf("v1 loader accepted v2 scenario: %v", err)
	}
	visible, err := scenario.AgentVisibleLane("lane-a")
	if err != nil {
		t.Fatal(err)
	}
	visibleJSON, err := json.Marshal(visible)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"ledgerOracle", "sourceEvidence", "materialReason"} {
		if strings.Contains(string(visibleJSON), forbidden) {
			t.Fatalf("agent-visible lane leaked %q: %s", forbidden, visibleJSON)
		}
	}
}

func TestCollaborationValidationRejectsInvalidLaneAndScheduleContracts(t *testing.T) {
	fixture, _ := collaborationFixture(t)
	tests := []struct {
		name   string
		mutate func(*harness.CollaborationScenario)
		want   string
	}{
		{
			name:   "requires exactly two lanes",
			mutate: func(s *harness.CollaborationScenario) { s.Lanes = s.Lanes[:1] },
			want:   "exactly two lanes",
		},
		{
			name:   "requires distinct lane ids",
			mutate: func(s *harness.CollaborationScenario) { s.Lanes[1].ID = s.Lanes[0].ID },
			want:   "duplicate lane id",
		},
		{
			name:   "requires distinct actors",
			mutate: func(s *harness.CollaborationScenario) { s.Lanes[1].ActorKey = s.Lanes[0].ActorKey },
			want:   "distinct actors",
		},
		{
			name:   "requires a real snapshot",
			mutate: func(s *harness.CollaborationScenario) { s.Project.RepositoryProfile = "fake" },
			want:   "real-snapshot",
		},
		{
			name:   "pins matching source evidence",
			mutate: func(s *harness.CollaborationScenario) { s.SourceEvidence.SnapshotID = "other-snapshot" },
			want:   "must match",
		},
		{
			name:   "does not call a sanitized snapshot historical intent",
			mutate: func(s *harness.CollaborationScenario) { s.SourceEvidence.HistoricalClaim = true },
			want:   "historicalClaim",
		},
		{
			name:   "bounds polling by timeout",
			mutate: func(s *harness.CollaborationScenario) { s.Schedule.PollIntervalMS = s.Schedule.TimeoutSeconds*1000 + 1 },
			want:   "pollIntervalMs",
		},
		{
			name:   "requires a valid dependent lane",
			mutate: func(s *harness.CollaborationScenario) { s.Schedule.DependentLane = "missing" },
			want:   "dependentLane",
		},
		{
			name:   "requires checkpoint path overlap",
			mutate: func(s *harness.CollaborationScenario) { s.Schedule.Barrier.RequiredPathOverlap = nil },
			want:   "path overlap",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scenario := cloneCollaborationScenario(t, fixture)
			test.mutate(&scenario)
			if err := scenario.Validate(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestCollaborationReportArtifactsAreRelativeAndCredentialFree(t *testing.T) {
	prepared := harness.CollaborationPreparedWorld{
		SchemaVersion: 2, ScenarioID: "dev-collaboration-contract-001", ScenarioHash: "scenario-hash",
		CorpusVersion: "v2", Split: "dev", ProjectSlug: "eval-v2-collaboration", ProjectID: "project-1",
		RepositoryHead: "2222222", SkillHash: "skill-hash", CreatedAt: time.Now().UTC(),
		Lanes: []harness.PreparedLane{
			{LaneID: "lane-a", ActorKey: "lane-a-agent", PrincipalID: "principal-a", Branch: "eval/lane-a", AgentName: "Checkpoint worker", Harness: "codex-cli", Provider: "openai", Model: "gpt-5.6-luna", Role: "primary", RepositoryHead: "2222222", SkillHash: "skill-hash", RepositoryPath: "repositories/lane-a", ArtifactPath: "lanes/lane-a.json"},
			{LaneID: "lane-b", ActorKey: "lane-b-agent", PrincipalID: "principal-b", Branch: "eval/lane-b", AgentName: "Dependent worker", Harness: "codex-cli", Provider: "openai", Model: "gpt-5.6-luna", Role: "primary", RepositoryHead: "2222222", SkillHash: "skill-hash", RepositoryPath: "repositories/lane-b", ArtifactPath: "lanes/lane-b.json"},
		},
	}
	if err := prepared.Validate(); err != nil {
		t.Fatal(err)
	}
	prepared.Lanes[1].PrincipalID = prepared.Lanes[0].PrincipalID
	if err := prepared.Validate(); err == nil || !strings.Contains(err.Error(), "distinct principals") {
		t.Fatalf("shared lane principal was accepted: %v", err)
	}
	prepared.Lanes[1].PrincipalID = "principal-b"
	body, err := json.Marshal(prepared)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(body)), "credential") || strings.Contains(strings.ToLower(string(body)), "token") {
		t.Fatalf("prepared artifact serializes a secret reference: %s", body)
	}
	episode := harness.CollaborationEpisode{
		SchemaVersion: 2, EpisodeID: "episode-1", ScenarioID: prepared.ScenarioID, ScenarioHash: prepared.ScenarioHash,
		Status: "incomplete", SchedulePath: "schedule.json", ControllerEvents: "controller-events.jsonl",
		Repository: harness.RepositoryResult{BaselineHead: "2222222", LaneAPath: "repositories/lane-a"},
		Lanes:      []harness.LaneResult{{LaneID: "lane-a", ActorKey: "lane-a-agent", Status: "failed", EventsPath: "lanes/lane-a/events.jsonl", StderrPath: "lanes/lane-a/stderr.log"}},
	}
	if err = episode.Validate(); err != nil {
		t.Fatal(err)
	}
	episode.Lanes[0].EventsPath = "secrets/credentials.json"
	if err = episode.Validate(); err == nil || !strings.Contains(err.Error(), "secrets or credentials") {
		t.Fatalf("credential path was accepted: %v", err)
	}
	if err = (harness.ControllerEvent{At: time.Now().UTC(), Type: "lane.started", Message: "token=secret"}).Validate(); err == nil {
		t.Fatal("controller event accepted token-like content")
	}
}

func TestRelayDeskFixtureContentHashIsPinned(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "fixtures", "rendered", "relaydesk-001.json"))
	if err != nil {
		t.Fatal(err)
	}
	const want = "bff52dc5db3c6fccb9b41ad9716f30a0733965c4cfe91359c01fed53014280f9"
	if got := harness.ContentHash(body); got != want {
		t.Fatalf("relaydesk fixture hash = %s, want %s", got, want)
	}
}

func cloneCollaborationScenario(t *testing.T, source harness.CollaborationScenario) harness.CollaborationScenario {
	t.Helper()
	body, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	var clone harness.CollaborationScenario
	if err = json.Unmarshal(body, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}
