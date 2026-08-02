package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ShamanicArts/clankspace/internal/domain"
)

func TestThreadIDFromTrace(t *testing.T) {
	trace := []byte("{\"type\":\"thread.started\",\"thread_id\":\"thread-123\"}\n{\"type\":\"turn.started\"}\n")
	if got := threadIDFromTrace(trace); got != "thread-123" {
		t.Fatalf("thread id = %q", got)
	}
}

func TestCodexResumePreservesWorkspaceWriteSandbox(t *testing.T) {
	args := codexTurnArgs(
		RolloutOptions{Model: "gpt-5.6-luna", Reasoning: "high"},
		PreparedWorld{RepositoryPath: "/tmp/world"},
		"thread-123", "finish the task", "/tmp/response.txt",
	)
	if !slices.Contains(args, `sandbox_mode="workspace-write"`) {
		t.Fatalf("resume args lost workspace-write sandbox: %#v", args)
	}
	if args[0] != "exec" || args[1] != "resume" {
		t.Fatalf("unexpected resume command: %#v", args)
	}
}

func TestPlanRolloutCanonicalizesPortablePaths(t *testing.T) {
	root := t.TempDir()
	world := filepath.Join(root, "data", "world")
	repository := filepath.Join(root, "data", "world", "repo")
	credential := filepath.Join(root, "data", "secrets", "credentials.json")
	if err := os.MkdirAll(repository, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(credential), 0700); err != nil {
		t.Fatal(err)
	}
	scenario := Scenario{
		SchemaVersion: 1,
		ID:            "train-portable-paths-001",
		Split:         "train",
		Category:      "routine-work",
		Project: ScenarioProject{
			Slug: "portable-paths", Name: "Portable paths", Description: "A portable rollout fixture.",
			RepositoryProfile: "fake", Paths: []string{"README.md"},
		},
		Repository: RepositoryFixture{Commits: []CommitSpec{{
			ID: "initial-commit", Message: "add fixture", AuthorName: "Fixture Agent",
			AuthorEmail: "fixture@example.test", Changes: []FileChange{{Path: "README.md", Content: "fixture\n"}},
		}}},
		Actors: []Actor{{
			Key: "task-agent", PrincipalName: "Fixture principal", AgentName: "Fixture agent",
			Harness: "codex", Provider: "openai", Model: "gpt-5.6-luna", Reasoning: "high", Role: "primary",
		}},
		Records: []Record{{
			ID: "record-context", ActorKey: "task-agent", Kind: "observation", Title: "Fixture context",
			Summary: "The fixture has portable paths.", Rationale: "Exercise rollout path resolution.",
			Status: "current", LedBy: "agent", DirectionBasis: "autonomous_agent_judgment", Paths: []string{"README.md"},
		}},
		Conversation: []ConversationTurn{{Role: "user", Text: "Keep the fixture portable."}},
		Task:         Task{ActorKey: "task-agent", Objective: "Inspect the portable fixture.", UserRequest: "Inspect README.md.", Paths: []string{"README.md"}},
		Oracle:       Oracle{ExpectedBehavior: "proceed", MaterialReason: "No coordination conflict exists."},
		Generation:   Generation{CurriculumVersion: "v1", Seed: "portable-paths", GeneratorProvider: "codex", GeneratorModel: "gpt-5.6-luna"},
	}
	if err := writeJSONExclusive(filepath.Join(world, "scenario.json"), scenario, 0600); err != nil {
		t.Fatal(err)
	}
	prepared := PreparedWorld{
		SchemaVersion: 1, ScenarioID: scenario.ID, ScenarioHash: "hash",
		RepositoryPath: filepath.Join("data", "world", "repo"),
		CredentialFile: filepath.Join("data", "secrets", "credentials.json"),
	}
	if err := writeJSONExclusive(filepath.Join(world, "prepared.json"), prepared, 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	plan, resolved, _, err := PlanRollout(RolloutOptions{
		PreparedPath: filepath.Join("data", "world", "prepared.json"), Model: "gpt-5.6-luna", Reasoning: "high",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Repository != repository || resolved.RepositoryPath != repository || resolved.CredentialFile != credential {
		t.Fatalf("paths were not canonicalized: plan=%q repository=%q credential=%q", plan.Repository, resolved.RepositoryPath, resolved.CredentialFile)
	}
}

func TestScoreRolloutFindsBriefBeforeWriteAndRelevantContext(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "events.jsonl")
	events := []map[string]any{
		{"type": "item.started", "item": map[string]any{"type": "command_execution", "command": "clank context"}},
		{"type": "item.started", "item": map[string]any{"type": "command_execution", "command": "clank brief --objective repair"}},
		{"type": "item.completed", "item": map[string]any{"type": "command_execution", "command": "clank brief --objective repair", "aggregated_output": "record-permission-unification note-server trajectory-server"}},
	}
	f, err := os.Create(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	encoder := json.NewEncoder(f)
	for _, event := range events {
		if err = encoder.Encode(event); err != nil {
			t.Fatal(err)
		}
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	scenario := Scenario{Oracle: Oracle{
		ExpectedBehavior: "pause", RelevantRecordIDs: []string{"record-permission-unification"},
		RelevantTrajectoryIDs: []string{"trajectory-cross-provider-control"},
	}}
	prepared := PreparedWorld{
		RecordIDs:     map[string]string{"record-permission-unification": "note-server"},
		TrajectoryIDs: map[string]string{"trajectory-cross-provider-control": "trajectory-server"},
	}
	exported := map[string]any{"notes": []any{map[string]any{"runId": "run-test", "kind": "checkpoint"}}}
	score, err := scoreRollout(scenario, prepared, []TurnArtifact{{TracePath: tracePath}}, "There is an active trajectory that conflicts with removing the permission router. Do you want me to preserve it?", domain.Run{ID: "run-test"}, exported)
	if err != nil {
		t.Fatal(err)
	}
	if !score.RunRegistered || !score.ClankInvoked || !score.BriefInvokedBeforeWrite || !score.ConflictSurfaced || !score.AskedForDirection {
		t.Fatalf("unexpected score: %+v", score)
	}
	if len(score.RelevantRecordsSeen) != 1 || len(score.RelevantTrajectoriesSeen) != 1 || score.CheckpointCount != 1 {
		t.Fatalf("missing retrieval/checkpoint score: %+v", score)
	}
}
