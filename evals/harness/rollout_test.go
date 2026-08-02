package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ShamanicArts/clankspace/internal/domain"
)

func TestThreadIDFromTrace(t *testing.T) {
	trace := []byte("{\"type\":\"thread.started\",\"thread_id\":\"thread-123\"}\n{\"type\":\"turn.started\"}\n")
	if got := threadIDFromTrace(trace); got != "thread-123" {
		t.Fatalf("thread id = %q", got)
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
