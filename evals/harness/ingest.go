package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type IngestResult struct {
	SchemaVersion int      `json:"schemaVersion"`
	WorkflowRun   string   `json:"workflowRun"`
	RawArtifact   string   `json:"rawArtifact"`
	Accepted      []string `json:"accepted"`
}

func IngestWorldWorkflow(inputPath, ledgerRoot string) (IngestResult, error) {
	body, err := os.ReadFile(inputPath)
	if err != nil {
		return IngestResult{}, err
	}
	var envelope struct {
		RunID  string `json:"runId"`
		Status string `json:"status"`
		Result struct {
			Accepted []json.RawMessage `json:"accepted"`
		} `json:"result"`
	}
	if err = json.Unmarshal(body, &envelope); err != nil {
		return IngestResult{}, fmt.Errorf("parse workflow output: %w", err)
	}
	if envelope.RunID == "" || envelope.Status != "completed" {
		return IngestResult{}, errors.New("workflow output must have a completed runId")
	}
	if len(envelope.Result.Accepted) == 0 {
		return IngestResult{}, errors.New("workflow completed without accepted worlds")
	}
	runDir := filepath.Join(ledgerRoot, "generation-runs", envelope.RunID)
	if err = os.MkdirAll(runDir, 0700); err != nil {
		return IngestResult{}, err
	}
	rawPath := filepath.Join(runDir, "omegacode-output.json")
	if err = writeExclusive(rawPath, body, 0600); err != nil {
		return IngestResult{}, err
	}
	result := IngestResult{SchemaVersion: 1, WorkflowRun: envelope.RunID, RawArtifact: rawPath, Accepted: []string{}}
	for _, raw := range envelope.Result.Accepted {
		var scenario Scenario
		if err = json.Unmarshal(raw, &scenario); err != nil {
			return IngestResult{}, fmt.Errorf("decode accepted scenario: %w", err)
		}
		if scenario.Generation.WorkflowRun == "" {
			scenario.Generation.WorkflowRun = envelope.RunID
		} else if scenario.Generation.WorkflowRun != envelope.RunID {
			return IngestResult{}, fmt.Errorf("scenario %s claims workflow run %q, expected %q", scenario.ID, scenario.Generation.WorkflowRun, envelope.RunID)
		}
		if err = scenario.Validate(); err != nil {
			return IngestResult{}, fmt.Errorf("accepted scenario %s is invalid: %w", scenario.ID, err)
		}
		canonical, marshalErr := json.MarshalIndent(scenario, "", "  ")
		if marshalErr != nil {
			return IngestResult{}, marshalErr
		}
		canonical = append(canonical, '\n')
		name := strings.TrimSuffix(scenario.ID, ".json") + "-" + ContentHash(canonical)[:12] + ".json"
		path := filepath.Join(runDir, "accepted", name)
		if err = writeExclusive(path, canonical, 0600); err != nil {
			return IngestResult{}, err
		}
		result.Accepted = append(result.Accepted, path)
	}
	if err = writeJSONExclusive(filepath.Join(runDir, "ingest.json"), result, 0600); err != nil {
		return IngestResult{}, err
	}
	return result, nil
}
