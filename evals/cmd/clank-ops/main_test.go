package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCollectsBoundedOperationsStatus(t *testing.T) {
	root := t.TempDir()
	gateDir := filepath.Join(root, "evals", "gates")
	omegaDir := filepath.Join(root, "omega", "wf_demo")
	opsDir := filepath.Join(root, "data", "ops")
	for _, dir := range []string{gateDir, omegaDir, opsDir} {
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
