package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCollectsBoundedOperationsStatus(t *testing.T) {
	root := t.TempDir()
	gateDir := filepath.Join(root, "evals", "gates")
	omegaDir := filepath.Join(root, "omega", "wf_demo")
	episodeDir := filepath.Join(root, "data", "corpora", "v1", "train", "scenario-one", "hash", "traces", "episode-one")
	rolloutDir := filepath.Join(root, "data", "corpora", "v1", "holdout", "scenario-two", "hash", "traces", "episode_two")
	opsDir := filepath.Join(root, "data", "ops")
	for _, dir := range []string{gateDir, omegaDir, episodeDir, filepath.Join(rolloutDir, "turn-001"), opsDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	gate := `{"gateId":"product-rc-999","status":"passed","productVerdict":"promote","productionWasTouched":false,"claimsSupported":["one"],"claimsNotYetEstablished":["two"],"alignedOverlapRegression":{"fixedEpisode":{"score":0.97}},"projectIsolationProbe":{"passed":true}}`
	if err := os.WriteFile(filepath.Join(gateDir, "product-rc-999.result.json"), []byte(gate), 0o600); err != nil {
		t.Fatal(err)
	}
	events := "{\"type\":\"run\",\"status\":\"started\",\"runId\":\"wf_demo\",\"workflowFile\":\"/safe/judge.workflow.js\",\"t\":1000}\n" +
		"{\"type\":\"agent\",\"label\":\"judge | gpt-test | codex | high\",\"state\":\"done\",\"t\":2000}\n"
	if err := os.WriteFile(filepath.Join(omegaDir, "events.jsonl"), []byte(events), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(omegaDir, "result.json"), []byte(`{"verdicts":[{"accepted":true,"score":0.97}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	controllerEvents := fmt.Sprintf("{\"at\":%q,\"type\":\"episode.started\"}\n", now.Add(-3*time.Minute).Format(time.RFC3339Nano)) +
		fmt.Sprintf("{\"at\":%q,\"type\":\"lane.started\",\"laneId\":\"lane-a\"}\n", now.Add(-2*time.Minute).Format(time.RFC3339Nano)) +
		fmt.Sprintf("{\"at\":%q,\"type\":\"barrier.observed\",\"laneId\":\"lane-a\"}\n", now.Add(-time.Minute).Format(time.RFC3339Nano)) +
		fmt.Sprintf("{\"at\":%q,\"type\":\"lane.started\",\"laneId\":\"lane-b\"}\n", now.Format(time.RFC3339Nano))
	if err := os.WriteFile(filepath.Join(episodeDir, "controller-events.jsonl"), []byte(controllerEvents), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rolloutDir, "turn-001", "events.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	journal := filepath.Join(opsDir, "shift.jsonl")
	entry := journalEntry{ID: "ops_test", At: time.Unix(100, 0).UTC(), Kind: "validation", State: "active", Title: "Seed passed"}
	if err := appendJournal(journal, entry); err != nil {
		t.Fatal(err)
	}
	deployPath := filepath.Join(opsDir, "deployments.json")
	deployBytes, _ := json.Marshal(deploymentFile{Deployments: []deployment{{Name: "eval", Health: "passed", Readiness: "passed", ObservedAt: time.Now()}}})
	if err := os.WriteFile(deployPath, deployBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	got := collect(config{Root: root, OmegaRoot: filepath.Join(root, "omega"), JournalPath: journal, DeployPath: deployPath})
	if got.Shift.State != "active" || got.Shift.Headline != "Seed passed" {
		t.Fatalf("unexpected shift: %#v", got.Shift)
	}
	if len(got.Gates) != 1 || got.Gates[0].ID != "product-rc-999" || got.Gates[0].PrimaryScore == nil || *got.Gates[0].PrimaryScore != 0.97 {
		t.Fatalf("unexpected gates: %#v", got.Gates)
	}
	if len(got.Runs) != 1 || got.Runs[0].Outcome != "accepted" || got.Runs[0].AgentCounts["done"] != 1 {
		t.Fatalf("unexpected runs: %#v", got.Runs)
	}
	byID := map[string]episodeSummary{}
	for _, episode := range got.Episodes {
		byID[episode.ID] = episode
	}
	rollout := byID["episode_two"]
	collaboration := byID["episode-one"]
	if len(got.Episodes) != 2 || rollout.Kind != "single-agent" || rollout.Stage != "turn 1 running" {
		t.Fatalf("unexpected rollout episode: %#v", got.Episodes)
	}
	if collaboration.Kind != "collaboration" || !collaboration.BarrierObserved || !collaboration.LaneBStarted || collaboration.Stage != "lane B running" {
		t.Fatalf("unexpected episodes: %#v", got.Episodes)
	}
	if len(got.Deployments) != 1 || len(got.ClaimsProven) != 1 || len(got.ClaimsOpen) != 1 {
		t.Fatalf("unexpected aggregate: %#v", got)
	}
}

func TestPostRequiresBoundedFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shift.jsonl")
	if err := post([]string{"--journal", path, "--title", "Night shift started", "--state", "active", "--evidence", "commit=abc123"}); err != nil {
		t.Fatal(err)
	}
	entries, err := readJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Evidence["commit"] != "abc123" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}
