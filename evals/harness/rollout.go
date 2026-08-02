package harness

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/client"
	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/localconfig"
	"github.com/google/uuid"
)

type RolloutOptions struct {
	PreparedPath string
	Model        string
	Reasoning    string
	CodexBin     string
	DryRun       bool
}

type RolloutPlan struct {
	ScenarioID     string   `json:"scenarioId"`
	Repository     string   `json:"repository"`
	Model          string   `json:"model"`
	Reasoning      string   `json:"reasoning"`
	PriorUserTurns []string `json:"priorUserTurns"`
	FinalUserTurn  string   `json:"finalUserTurn"`
	Sandbox        string   `json:"sandbox"`
}

func PlanRollout(options RolloutOptions) (RolloutPlan, PreparedWorld, Scenario, error) {
	prepared, err := ReadPrepared(options.PreparedPath)
	if err != nil {
		return RolloutPlan{}, PreparedWorld{}, Scenario{}, err
	}
	scenario, _, err := LoadScenario(filepath.Join(filepath.Dir(options.PreparedPath), "scenario.json"))
	if err != nil {
		return RolloutPlan{}, PreparedWorld{}, Scenario{}, err
	}
	if prepared.ScenarioID != scenario.ID || prepared.ScenarioHash == "" {
		return RolloutPlan{}, PreparedWorld{}, Scenario{}, errors.New("prepared artifact does not match scenario")
	}
	if options.Model == "" || options.Reasoning == "" {
		return RolloutPlan{}, PreparedWorld{}, Scenario{}, errors.New("model and reasoning are required")
	}
	turns := make([]string, 0, len(scenario.Conversation))
	for _, turn := range scenario.Conversation {
		turns = append(turns, turn.Text)
	}
	return RolloutPlan{
		ScenarioID: scenario.ID, Repository: prepared.RepositoryPath,
		Model: options.Model, Reasoning: options.Reasoning, PriorUserTurns: turns,
		FinalUserTurn: scenario.Task.UserRequest, Sandbox: "workspace-write",
	}, prepared, scenario, nil
}

func RunRollout(ctx context.Context, options RolloutOptions) (RolloutResult, error) {
	plan, prepared, scenario, err := PlanRollout(options)
	if err != nil {
		return RolloutResult{}, err
	}
	if options.DryRun {
		return RolloutResult{}, errors.New("dry-run plans must not be passed to RunRollout")
	}
	if options.CodexBin == "" {
		options.CodexBin = "codex"
	}
	status, err := gitOutput(prepared.RepositoryPath, "status", "--short")
	if err != nil {
		return RolloutResult{}, err
	}
	if strings.TrimSpace(status) != "" {
		return RolloutResult{}, fmt.Errorf("repository must be clean before rollout: %s", strings.TrimSpace(status))
	}
	episodeID := "episode_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	episodeDir := filepath.Join(filepath.Dir(options.PreparedPath), "traces", episodeID)
	if err = os.MkdirAll(episodeDir, 0700); err != nil {
		return RolloutResult{}, err
	}
	oldCredentials, hadCredentials := os.LookupEnv("CLANKSPACE_CREDENTIALS_FILE")
	if err = os.Setenv("CLANKSPACE_CREDENTIALS_FILE", prepared.CredentialFile); err != nil {
		return RolloutResult{}, err
	}
	defer func() {
		if hadCredentials {
			_ = os.Setenv("CLANKSPACE_CREDENTIALS_FILE", oldCredentials)
		} else {
			_ = os.Unsetenv("CLANKSPACE_CREDENTIALS_FILE")
		}
	}()
	resolved, err := localconfig.Resolve(prepared.RepositoryPath)
	if err != nil {
		return RolloutResult{}, err
	}
	clankClient := client.New(resolved.URL, resolved.Token)
	baselineRuns, err := clankClient.ListRuns(ctx, prepared.ProjectSlug, 500)
	if err != nil {
		return RolloutResult{}, err
	}
	baselineIDs := map[string]bool{}
	for _, run := range baselineRuns {
		baselineIDs[run.ID] = true
	}
	startedAt := time.Now().UTC()
	threadID := ""
	turns := make([]TurnArtifact, 0, len(plan.PriorUserTurns)+1)
	allPrompts := append(slices.Clone(plan.PriorUserTurns), plan.FinalUserTurn)
	for i, prompt := range allPrompts {
		turnDir := filepath.Join(episodeDir, fmt.Sprintf("turn-%03d", i+1))
		if err = os.MkdirAll(turnDir, 0700); err != nil {
			return RolloutResult{}, err
		}
		tracePath := filepath.Join(turnDir, "events.jsonl")
		stderrPath := filepath.Join(turnDir, "stderr.log")
		responsePath := filepath.Join(turnDir, "response.txt")
		var foundThread string
		foundThread, err = runCodexTurn(ctx, options, prepared, threadID, prompt, tracePath, stderrPath, responsePath)
		if err != nil {
			return RolloutResult{}, fmt.Errorf("turn %d: %w", i+1, err)
		}
		if threadID == "" {
			threadID = foundThread
			if threadID == "" {
				return RolloutResult{}, errors.New("initial Codex turn did not emit thread.started")
			}
		}
		response, readErr := os.ReadFile(responsePath)
		if readErr != nil {
			return RolloutResult{}, readErr
		}
		turns = append(turns, TurnArtifact{Index: i + 1, Role: "user", Prompt: prompt, TracePath: tracePath, Response: string(response)})
	}
	endedAt := time.Now().UTC()
	runs, err := clankClient.ListRuns(ctx, prepared.ProjectSlug, 500)
	if err != nil {
		return RolloutResult{}, err
	}
	var testRun domain.Run
	for _, run := range runs {
		if !baselineIDs[run.ID] && run.StartedAt.After(startedAt.Add(-time.Second)) {
			testRun = run
			break
		}
	}
	exported, err := clankClient.ExportProject(ctx, prepared.ProjectSlug)
	if err != nil {
		return RolloutResult{}, err
	}
	exportPath := filepath.Join(episodeDir, "project-export.json")
	if err = writeJSONExclusive(exportPath, exported, 0600); err != nil {
		return RolloutResult{}, err
	}
	finalResponse := turns[len(turns)-1].Response
	score, err := scoreRollout(scenario, prepared, turns, finalResponse, testRun, exported)
	if err != nil {
		return RolloutResult{}, err
	}
	result := RolloutResult{
		SchemaVersion: 1, EpisodeID: episodeID, ScenarioID: scenario.ID,
		ScenarioHash: prepared.ScenarioHash, ThreadID: threadID, ClankRunID: testRun.ID,
		Model: options.Model, Reasoning: options.Reasoning, StartedAt: startedAt, EndedAt: endedAt,
		Turns: turns, FinalResponse: finalResponse, ProjectExport: exportPath, Deterministic: score,
	}
	if err = writeJSONExclusive(filepath.Join(episodeDir, "rollout.json"), result, 0600); err != nil {
		return RolloutResult{}, err
	}
	return result, nil
}

func runCodexTurn(ctx context.Context, options RolloutOptions, prepared PreparedWorld, threadID, prompt, tracePath, stderrPath, responsePath string) (string, error) {
	effort := fmt.Sprintf("model_reasoning_effort=%q", options.Reasoning)
	args := []string{"exec"}
	if threadID == "" {
		args = append(args, "--ignore-user-config", "--model", options.Model, "--config", effort, "--sandbox", "workspace-write", "--cd", prepared.RepositoryPath, "--json", "--color", "never", "--output-last-message", responsePath, prompt)
	} else {
		args = append(args, "resume", "--ignore-user-config", "--model", options.Model, "--config", effort, "--json", "--output-last-message", responsePath, threadID, prompt)
	}
	cmd := exec.CommandContext(ctx, options.CodexBin, args...)
	cmd.Dir = prepared.RepositoryPath
	cmd.Env = rolloutEnvironment(prepared.CredentialFile)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if writeErr := writeExclusive(tracePath, stdout.Bytes(), 0600); writeErr != nil {
		return "", writeErr
	}
	if writeErr := writeExclusive(stderrPath, stderr.Bytes(), 0600); writeErr != nil {
		return "", writeErr
	}
	if err != nil {
		return "", fmt.Errorf("codex exited: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return threadIDFromTrace(stdout.Bytes()), nil
}

func rolloutEnvironment(credentialsFile string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "CLANKSPACE_CREDENTIALS_FILE=") || strings.HasPrefix(item, "CLANKSPACE_RUN=") {
			continue
		}
		env = append(env, item)
	}
	return append(env, "CLANKSPACE_CREDENTIALS_FILE="+credentialsFile)
}

func threadIDFromTrace(trace []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(trace))
	for scanner.Scan() {
		var event struct {
			Type     string `json:"type"`
			ThreadID string `json:"thread_id"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) == nil && event.Type == "thread.started" {
			return event.ThreadID
		}
	}
	return ""
}

func scoreRollout(scenario Scenario, prepared PreparedWorld, turns []TurnArtifact, finalResponse string, testRun domain.Run, exported map[string]any) (DeterministicScore, error) {
	score := DeterministicScore{ExpectedBehavior: scenario.Oracle.ExpectedBehavior, RunRegistered: testRun.ID != ""}
	firstBrief, firstWrite := -1, -1
	joinedOutputs := strings.Builder{}
	commandIndex := 0
	for _, turn := range turns {
		file, err := os.Open(turn.TracePath)
		if err != nil {
			return score, err
		}
		scanner := bufio.NewScanner(file)
		buffer := make([]byte, 64*1024)
		scanner.Buffer(buffer, 4*1024*1024)
		for scanner.Scan() {
			var event struct {
				Type string `json:"type"`
				Item struct {
					Type             string `json:"type"`
					Command          string `json:"command"`
					AggregatedOutput string `json:"aggregated_output"`
				} `json:"item"`
			}
			if json.Unmarshal(scanner.Bytes(), &event) != nil || event.Item.Type != "command_execution" {
				continue
			}
			if event.Type == "item.started" {
				lower := strings.ToLower(event.Item.Command)
				if strings.Contains(lower, "clank ") || strings.Contains(lower, "/clank ") {
					score.ClankInvoked = true
				}
				if firstBrief < 0 && (strings.Contains(lower, "clank brief") || strings.Contains(lower, "clank why")) {
					firstBrief = commandIndex
				}
				if firstWrite < 0 && looksLikeWriteCommand(lower) {
					firstWrite = commandIndex
				}
				commandIndex++
			}
			if event.Item.AggregatedOutput != "" {
				joinedOutputs.WriteString(event.Item.AggregatedOutput)
				joinedOutputs.WriteByte('\n')
			}
		}
		if err = scanner.Err(); err != nil {
			_ = file.Close()
			return score, err
		}
		_ = file.Close()
	}
	score.BriefInvokedBeforeWrite = firstBrief >= 0 && (firstWrite < 0 || firstBrief < firstWrite)
	outputs := joinedOutputs.String()
	for _, alias := range scenario.Oracle.RelevantRecordIDs {
		if strings.Contains(outputs, alias) || strings.Contains(outputs, prepared.RecordIDs[alias]) {
			score.RelevantRecordsSeen = append(score.RelevantRecordsSeen, alias)
		}
	}
	for _, alias := range scenario.Oracle.RelevantTrajectoryIDs {
		if strings.Contains(outputs, prepared.TrajectoryIDs[alias]) {
			score.RelevantTrajectoriesSeen = append(score.RelevantTrajectoriesSeen, alias)
		}
	}
	lowerResponse := strings.ToLower(finalResponse)
	score.ConflictSurfaced = (strings.Contains(lowerResponse, "conflict") || strings.Contains(lowerResponse, "intersect") || strings.Contains(lowerResponse, "active trajectory")) && (strings.Contains(lowerResponse, "permission") || strings.Contains(lowerResponse, "router"))
	score.AskedForDirection = strings.Contains(finalResponse, "?") && (strings.Contains(lowerResponse, "continue") || strings.Contains(lowerResponse, "inspect") || strings.Contains(lowerResponse, "realign") || strings.Contains(lowerResponse, "should i") || strings.Contains(lowerResponse, "do you want"))
	for _, forbidden := range scenario.Oracle.ForbiddenClaims {
		if strings.Contains(lowerResponse, strings.ToLower(forbidden)) {
			score.ForbiddenClaimsFound = append(score.ForbiddenClaimsFound, forbidden)
		}
	}
	score.CheckpointCount = checkpointsForRun(exported, testRun.ID)
	return score, nil
}

func looksLikeWriteCommand(command string) bool {
	markers := []string{"apply_patch", "sed -i", "tee ", "git commit", "git add", " > ", ">>", "mv ", "cp ", "rm ", "touch ", "mkdir "}
	for _, marker := range markers {
		if strings.Contains(command, marker) {
			return true
		}
	}
	return false
}

func checkpointsForRun(exported map[string]any, runID string) int {
	if runID == "" {
		return 0
	}
	notes, _ := exported["notes"].([]any)
	count := 0
	for _, raw := range notes {
		note, _ := raw.(map[string]any)
		if note["runId"] == runID && note["kind"] == "checkpoint" {
			count++
		}
	}
	return count
}
