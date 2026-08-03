package harness

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ShamanicArts/clankspace/internal/client"
	"github.com/ShamanicArts/clankspace/internal/domain"
)

// CollaborationRolloutOptions runs the v2 event-gated pilot. Credential files
// are deliberately supplied out-of-band: <credentials-dir>/<lane-id>.json.
// They are never included in a reportable artifact.
type CollaborationRolloutOptions struct {
	PreparedPath   string
	RepositoryPath string
	CredentialsDir string
	CodexBin       string
	EpisodeID      string
	DryRun         bool
	ServerConfig   string
	ServerCommit   string
	replay         CollaborationReplay
}

// CollaborationPlan is operational output only. Its credential paths are
// useful in a dry-run but are not persisted in the episode ledger.
type CollaborationPlan struct {
	ScenarioID  string                   `json:"scenarioId"`
	ProjectSlug string                   `json:"projectSlug"`
	EpisodeDir  string                   `json:"episodeDir"`
	Initial     CollaborationLanePlan    `json:"initial"`
	Barrier     CollaborationBarrierPlan `json:"barrier"`
	Dependent   CollaborationLanePlan    `json:"dependent"`
}

type CollaborationBarrierPlan struct {
	TimeoutSeconds int      `json:"timeoutSeconds"`
	PollIntervalMS int      `json:"pollIntervalMs"`
	EventType      string   `json:"eventType"`
	Kind           string   `json:"kind"`
	Paths          []string `json:"requiredPathOverlap"`
}

type CollaborationLanePlan struct {
	LaneID         string                     `json:"laneId"`
	Worktree       string                     `json:"worktree"`
	CredentialFile string                     `json:"credentialFile"`
	EventsPath     string                     `json:"eventsPath"`
	StderrPath     string                     `json:"stderrPath"`
	ResponsesDir   string                     `json:"responsesDir"`
	Processes      []CollaborationProcessPlan `json:"processes"`
}

type CollaborationProcessPlan struct {
	Args         []string `json:"args"`
	Directory    string   `json:"directory"`
	ResponsePath string   `json:"responsePath"`
}

type collaborationWorld struct {
	preparedPath string
	prepared     CollaborationPreparedWorld
	scenario     CollaborationScenario
	baseRepo     string
	episodeDir   string
	lanes        map[string]PreparedLane
	replay       CollaborationReplay
}

type collaborationCredentialFile struct {
	Credentials []struct {
		URL     string `json:"url"`
		Project string `json:"project"`
		Token   string `json:"token"`
	} `json:"credentials"`
}

// PlanCollaborationRollout validates all durable inputs without launching a
// process or writing an episode. EpisodeID is explicit so dry-run paths and
// live evidence locations are exactly reproducible.
func PlanCollaborationRollout(options CollaborationRolloutOptions) (CollaborationPlan, error) {
	world, err := loadCollaborationWorld(options)
	if err != nil {
		return CollaborationPlan{}, err
	}
	initial, err := collaborationLanePlan(world, options, world.scenario.Schedule.InitialLane)
	if err != nil {
		return CollaborationPlan{}, err
	}
	dependent, err := collaborationLanePlan(world, options, world.scenario.Schedule.DependentLane)
	if err != nil {
		return CollaborationPlan{}, err
	}
	return CollaborationPlan{
		ScenarioID: world.scenario.ID, ProjectSlug: world.prepared.ProjectSlug, EpisodeDir: world.episodeDir,
		Initial: initial, Dependent: dependent,
		Barrier: CollaborationBarrierPlan{TimeoutSeconds: world.scenario.Schedule.TimeoutSeconds, PollIntervalMS: world.scenario.Schedule.PollIntervalMS, EventType: world.scenario.Schedule.Barrier.EventType, Kind: world.scenario.Schedule.Barrier.Kind, Paths: world.scenario.Schedule.Barrier.RequiredPathOverlap},
	}, nil
}

func loadCollaborationWorld(options CollaborationRolloutOptions) (collaborationWorld, error) {
	if options.PreparedPath == "" || options.RepositoryPath == "" || options.CredentialsDir == "" || options.EpisodeID == "" {
		return collaborationWorld{}, errors.New("prepared path, repository path, credentials dir, and episode ID are required")
	}
	if !scenarioIDPattern.MatchString(options.EpisodeID) {
		return collaborationWorld{}, errors.New("episode ID must contain only lowercase letters, numbers, and hyphens")
	}
	preparedPath, err := filepath.Abs(options.PreparedPath)
	if err != nil {
		return collaborationWorld{}, err
	}
	baseRepo, err := filepath.Abs(options.RepositoryPath)
	if err != nil {
		return collaborationWorld{}, err
	}
	prepared, err := ReadCollaborationPrepared(preparedPath)
	if err != nil {
		return collaborationWorld{}, err
	}
	if err = prepared.Validate(); err != nil {
		return collaborationWorld{}, err
	}
	scenario, canonical, err := LoadCollaborationScenario(filepath.Join(filepath.Dir(preparedPath), "scenario.json"))
	if err != nil {
		return collaborationWorld{}, err
	}
	if prepared.ScenarioID != scenario.ID || prepared.ScenarioHash != ContentHash(canonical) {
		return collaborationWorld{}, errors.New("prepared collaboration artifact does not match scenario")
	}
	head, err := gitOutput(baseRepo, "rev-parse", "HEAD")
	if err != nil {
		return collaborationWorld{}, fmt.Errorf("baseline repository: %w", err)
	}
	if strings.TrimSpace(head) != prepared.RepositoryHead {
		return collaborationWorld{}, errors.New("baseline repository head does not match prepared artifact")
	}
	if err = runGit(baseRepo, nil, "merge-base", "--is-ancestor", scenario.SourceEvidence.SnapshotHead, prepared.RepositoryHead); err != nil {
		return collaborationWorld{}, errors.New("source evidence snapshot head is not an ancestor of the prepared repository")
	}
	licenseHash, err := FileHash(filepath.Join(baseRepo, filepath.FromSlash(scenario.SourceEvidence.LicenseFile)))
	if err != nil {
		return collaborationWorld{}, fmt.Errorf("source evidence license file: %w", err)
	}
	if licenseHash != scenario.SourceEvidence.LicenseFileHash {
		return collaborationWorld{}, errors.New("source evidence license file hash does not match baseline repository")
	}
	status, err := gitOutput(baseRepo, "status", "--short")
	if err != nil {
		return collaborationWorld{}, err
	}
	if strings.TrimSpace(status) != "" {
		return collaborationWorld{}, errors.New("baseline repository must be clean")
	}
	lanes := make(map[string]PreparedLane, len(prepared.Lanes))
	for _, lane := range prepared.Lanes {
		lanes[lane.LaneID] = lane
	}
	for _, lane := range scenario.Lanes {
		preparedLane, exists := lanes[lane.ID]
		if !exists || preparedLane.ActorKey != lane.ActorKey || preparedLane.Branch != lane.Branch {
			return collaborationWorld{}, fmt.Errorf("prepared lane %q does not match scenario", lane.ID)
		}
	}
	return collaborationWorld{preparedPath: preparedPath, prepared: prepared, scenario: scenario, baseRepo: baseRepo, episodeDir: filepath.Join(filepath.Dir(preparedPath), "traces", options.EpisodeID), lanes: lanes, replay: options.replay}, nil
}

func collaborationLanePlan(world collaborationWorld, options CollaborationRolloutOptions, laneID string) (CollaborationLanePlan, error) {
	visible, err := world.scenario.AgentVisibleLane(laneID)
	if err != nil {
		return CollaborationLanePlan{}, err
	}
	prepared := world.lanes[laneID]
	worktree := filepath.Join(world.episodeDir, filepath.FromSlash(prepared.RepositoryPath))
	laneDir := filepath.Join(world.episodeDir, "lanes", laneID)
	responses := filepath.Join(laneDir, "responses")
	credential := filepath.Join(options.CredentialsDir, laneID+".json")
	bin := options.CodexBin
	if bin == "" {
		bin = "codex"
	}
	prompts := append([]ConversationTurn{}, visible.PriorUserTurns...)
	prompts = append(prompts, ConversationTurn{Role: "user", Text: visible.Task.UserRequest})
	processes := make([]CollaborationProcessPlan, 0, len(prompts))
	for i, turn := range prompts {
		response := filepath.Join(responses, fmt.Sprintf("turn-%03d.txt", i+1))
		args := collaborationCodexArgs(bin, prepared, worktree, "", turn.Text, response)
		if i > 0 { // The real executor replaces this placeholder with thread ID after turn one.
			args = collaborationCodexArgs(bin, prepared, worktree, "<thread-id-from-turn-001>", turn.Text, response)
		}
		processes = append(processes, CollaborationProcessPlan{Args: args, Directory: worktree, ResponsePath: response})
	}
	return CollaborationLanePlan{LaneID: laneID, Worktree: worktree, CredentialFile: credential, EventsPath: filepath.Join(laneDir, "events.jsonl"), StderrPath: filepath.Join(laneDir, "stderr.log"), ResponsesDir: responses, Processes: processes}, nil
}

func collaborationCodexArgs(bin string, lane PreparedLane, worktree, threadID, prompt, responsePath string) []string {
	effort := fmt.Sprintf("model_reasoning_effort=%q", lane.Reasoning)
	if threadID == "" {
		return []string{bin, "exec", "--ignore-user-config", "--model", lane.Model, "--config", effort, "--config", `sandbox_mode="workspace-write"`, "--config", "sandbox_workspace_write.network_access=true", "--sandbox", "workspace-write", "--json", "--color", "never", "--output-last-message", responsePath, "--cd", worktree, prompt}
	}
	// `codex exec resume` has a narrower flag surface than a fresh `exec` and
	// rejects `--color` and `--sandbox`. Preserve the effective sandbox through
	// config overrides, matching the established single-lane rollout path.
	return []string{bin, "exec", "resume", "--ignore-user-config", "--model", lane.Model, "--config", effort, "--config", `sandbox_mode="workspace-write"`, "--config", "sandbox_workspace_write.network_access=true", "--json", "--output-last-message", responsePath, threadID, prompt}
}

// ReadCollaborationPrepared loads the report-safe v2 preparation artifact.
func ReadCollaborationPrepared(path string) (CollaborationPreparedWorld, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return CollaborationPreparedWorld{}, err
	}
	var prepared CollaborationPreparedWorld
	if err = json.Unmarshal(body, &prepared); err != nil {
		return CollaborationPreparedWorld{}, fmt.Errorf("parse collaboration prepared: %w", err)
	}
	return prepared, nil
}

type collaborationObserver interface {
	ListRuns(context.Context, string, int) ([]domain.Run, error)
	ExportProject(context.Context, string) (map[string]any, error)
}

type collaborationExecutor interface {
	Launch(context.Context, CollaborationLanePlan, PreparedLane, []string) (<-chan collaborationExecution, error)
}

type collaborationExecution struct {
	ThreadID       string
	Events, Stderr []byte
	Responses      [][]byte
	Commands       []CollaborationProcessPlan
	Err            error
}

type realCollaborationExecutor struct{}

func (realCollaborationExecutor) Launch(ctx context.Context, plan CollaborationLanePlan, lane PreparedLane, env []string) (<-chan collaborationExecution, error) {
	result := make(chan collaborationExecution, 1)
	go func() {
		defer close(result)
		var events, stderr bytes.Buffer
		responses := make([][]byte, 0, len(plan.Processes))
		commands := make([]CollaborationProcessPlan, 0, len(plan.Processes))
		threadID := ""
		for i, process := range plan.Processes {
			if err := os.MkdirAll(filepath.Dir(process.ResponsePath), 0700); err != nil {
				result <- collaborationExecution{Events: events.Bytes(), Stderr: stderr.Bytes(), Responses: responses, Commands: commands, Err: err}
				return
			}
			args := append([]string{}, process.Args...)
			if i > 0 {
				for j := range args {
					if args[j] == "<thread-id-from-turn-001>" {
						args[j] = threadID
					}
				}
			}
			commands = append(commands, CollaborationProcessPlan{Args: append([]string{}, args...), Directory: process.Directory, ResponsePath: process.ResponsePath})
			cmd := exec.CommandContext(ctx, args[0], args[1:]...)
			cmd.Dir, cmd.Env = process.Directory, env
			var stdout, errout bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &errout
			err := cmd.Run()
			events.Write(stdout.Bytes())
			stderr.Write(errout.Bytes())
			body, readErr := os.ReadFile(process.ResponsePath)
			if readErr != nil {
				result <- collaborationExecution{ThreadID: threadID, Events: events.Bytes(), Stderr: stderr.Bytes(), Responses: responses, Commands: commands, Err: fmt.Errorf("lane %q turn %d response: %w", plan.LaneID, i+1, readErr)}
				return
			}
			responses = append(responses, body)
			// Codex writes this transient file directly. Remove it before the
			// sanitized public response is frozen with writeExclusive below.
			if removeErr := os.Remove(process.ResponsePath); removeErr != nil {
				result <- collaborationExecution{ThreadID: threadID, Events: events.Bytes(), Stderr: stderr.Bytes(), Responses: responses, Commands: commands, Err: removeErr}
				return
			}
			if err != nil {
				result <- collaborationExecution{ThreadID: threadID, Events: events.Bytes(), Stderr: stderr.Bytes(), Responses: responses, Commands: commands, Err: fmt.Errorf("lane %q turn %d: %w", plan.LaneID, i+1, err)}
				return
			}
			if i == 0 {
				threadID = threadIDFromTrace(stdout.Bytes())
				if threadID == "" {
					result <- collaborationExecution{Events: events.Bytes(), Stderr: stderr.Bytes(), Responses: responses, Commands: commands, Err: errors.New("initial Codex turn did not emit thread.started")}
					return
				}
			}
		}
		result <- collaborationExecution{ThreadID: threadID, Events: events.Bytes(), Stderr: stderr.Bytes(), Responses: responses, Commands: commands}
	}()
	return result, nil
}

type realCollaborationObserver struct{ client *client.Client }

func (o realCollaborationObserver) ListRuns(ctx context.Context, project string, limit int) ([]domain.Run, error) {
	return o.client.ListRuns(ctx, project, limit)
}
func (o realCollaborationObserver) ExportProject(ctx context.Context, project string) (map[string]any, error) {
	return o.client.ExportProject(ctx, project)
}

func RunCollaborationRollout(ctx context.Context, options CollaborationRolloutOptions) (CollaborationEpisode, error) {
	plan, err := PlanCollaborationRollout(options)
	if err != nil {
		return CollaborationEpisode{}, err
	}
	if options.DryRun {
		return CollaborationEpisode{}, errors.New("dry-run plans must not be passed to RunCollaborationRollout")
	}
	world, err := loadCollaborationWorld(options)
	if err != nil {
		return CollaborationEpisode{}, err
	}
	credential, err := readCollaborationCredential(plan.Initial.CredentialFile, world.prepared.ProjectSlug)
	if err != nil {
		return CollaborationEpisode{}, err
	}
	dependentCredential, err := readCollaborationCredential(plan.Dependent.CredentialFile, world.prepared.ProjectSlug)
	if err != nil {
		return CollaborationEpisode{}, err
	}
	if credential.token == dependentCredential.token {
		return CollaborationEpisode{}, errors.New("collaboration lanes must use distinct project credentials")
	}
	if credential.url != dependentCredential.url {
		return CollaborationEpisode{}, errors.New("collaboration lanes must use credentials issued by the same server")
	}
	options.replay, err = buildCollaborationReplay(options.CodexBin, credential.url, options.ServerConfig, options.ServerCommit)
	if err != nil {
		return CollaborationEpisode{}, err
	}
	world.replay = options.replay
	return runCollaboration(ctx, world, plan, options, realCollaborationObserver{client.New(credential.url, credential.token)}, realCollaborationExecutor{})
}

func buildCollaborationReplay(codexBin, serverURL, serverConfig, serverCommit string) (CollaborationReplay, error) {
	if codexBin == "" || serverConfig == "" || serverCommit == "" {
		return CollaborationReplay{}, errors.New("codex binary, server configuration, and server commit are required for a live collaboration rollout")
	}
	binaryHash, err := FileHash(codexBin)
	if err != nil {
		return CollaborationReplay{}, fmt.Errorf("hash Codex launcher: %w", err)
	}
	configHash, err := FileHash(serverConfig)
	if err != nil {
		return CollaborationReplay{}, fmt.Errorf("hash server configuration: %w", err)
	}
	return CollaborationReplay{CodexBinaryHash: binaryHash, ServerURLHash: ContentHash([]byte(serverURL)), ServerConfigHash: configHash, ServerCommit: serverCommit}, nil
}

type collaborationCredential struct{ url, token string }

func readCollaborationCredential(path, project string) (collaborationCredential, error) {
	info, err := os.Stat(path)
	if err != nil {
		return collaborationCredential{}, err
	}
	if info.Mode().Perm()&0077 != 0 {
		return collaborationCredential{}, fmt.Errorf("credential file %s must not be group/world readable", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return collaborationCredential{}, err
	}
	var file collaborationCredentialFile
	if err = json.Unmarshal(body, &file); err != nil {
		return collaborationCredential{}, err
	}
	for _, item := range file.Credentials {
		if item.Project == project && strings.TrimSpace(item.URL) != "" && strings.TrimSpace(item.Token) != "" {
			return collaborationCredential{url: strings.TrimRight(item.URL, "/"), token: item.Token}, nil
		}
	}
	return collaborationCredential{}, fmt.Errorf("credential file has no token for project %q", project)
}

func runCollaboration(ctx context.Context, world collaborationWorld, plan CollaborationPlan, options CollaborationRolloutOptions, observer collaborationObserver, executor collaborationExecutor) (episode CollaborationEpisode, terminal error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := os.MkdirAll(world.episodeDir, 0700); err != nil {
		return CollaborationEpisode{}, err
	}
	eventLog, err := newControllerEventLog(filepath.Join(world.episodeDir, "controller-events.jsonl"))
	if err != nil {
		return CollaborationEpisode{}, err
	}
	defer eventLog.Close()
	started := time.Now().UTC()
	episode = CollaborationEpisode{SchemaVersion: 2, EpisodeID: options.EpisodeID, ScenarioID: world.scenario.ID, ScenarioHash: world.prepared.ScenarioHash, Status: "running", StartedAt: started, SchedulePath: "schedule.json", ControllerEvents: "controller-events.jsonl", Repository: RepositoryResult{BaselineHead: world.prepared.RepositoryHead, LaneAPath: reportPath(world.episodeDir, plan.Initial.Worktree)}, Replay: world.replay}
	if err = writeJSONExclusive(filepath.Join(world.episodeDir, "schedule.json"), world.scenario.Schedule, 0600); err != nil {
		return episode, err
	}
	if err = eventLog.append("episode.started", "", "created event-gated collaboration episode"); err != nil {
		return episode, err
	}
	baselineRuns, baseExport, err := observation(ctx, observer, world, "before")
	if err != nil {
		terminal = err
		return finishCollaboration(world, &episode, eventLog, terminal)
	}
	baselineRunIDs := runIDs(baselineRuns)
	baselineNoteIDs := noteIDs(baseExport)
	if err = writeLaneObservation(plan.Initial, "before", baselineRuns, baseExport); err != nil {
		return finishCollaboration(world, &episode, eventLog, err)
	}
	if err = createCollaborationWorktree(world.baseRepo, plan.Initial.Worktree, world.lanes[plan.Initial.LaneID].Branch, world.prepared.RepositoryHead); err != nil {
		terminal = err
		return finishCollaboration(world, &episode, eventLog, terminal)
	}
	if err = eventLog.append("lane.worktree.created", plan.Initial.LaneID, "created independent lane worktree"); err != nil {
		terminal = err
		return finishCollaboration(world, &episode, eventLog, terminal)
	}
	aEnv, err := collaborationEnvironment(plan.Initial.CredentialFile, world.lanes[plan.Initial.LaneID], plan.Initial.Worktree)
	if err != nil {
		terminal = err
		return finishCollaboration(world, &episode, eventLog, terminal)
	}
	aDone, err := executor.Launch(runCtx, plan.Initial, world.lanes[plan.Initial.LaneID], aEnv)
	if err != nil {
		terminal = err
		return finishCollaboration(world, &episode, eventLog, terminal)
	}
	aResult := laneResultForPlan(world, plan.Initial)
	_ = eventLog.append("lane.started", plan.Initial.LaneID, "launched independent persisted Codex session")
	barrier, aExecution, waitErr := waitForBarrier(ctx, world, observer, eventLog, plan.Initial, aDone, baselineRunIDs, baselineNoteIDs)
	if aExecution != nil {
		aResult, err = persistLaneResult(ctx, world, observer, plan.Initial, aResult, *aExecution, baselineRunIDs)
		if err != nil {
			episode.Lanes = append(episode.Lanes, aResult)
			return finishCollaboration(world, &episode, eventLog, err)
		}
	}
	if waitErr != nil || !barrier.Observed {
		episode.Lanes = append(episode.Lanes, aResult)
		terminal = waitErr
		if terminal == nil {
			terminal = errors.New("initial lane ended or timed out without a matching barrier")
		}
		return finishCollaboration(world, &episode, eventLog, terminal)
	}
	episode.Barrier = barrier
	if err = writeJSONExclusive(filepath.Join(world.episodeDir, "barrier.json"), barrier, 0600); err != nil {
		terminal = err
		return finishCollaboration(world, &episode, eventLog, terminal)
	}
	if err = eventLog.append("barrier.observed", plan.Initial.LaneID, "durably observed matching checkpoint before dependent release"); err != nil {
		terminal = err
		return finishCollaboration(world, &episode, eventLog, terminal)
	}
	if err = createCollaborationWorktree(world.baseRepo, plan.Dependent.Worktree, world.lanes[plan.Dependent.LaneID].Branch, world.prepared.RepositoryHead); err != nil {
		terminal = err
		episode.Lanes = append(episode.Lanes, aResult)
		return finishCollaboration(world, &episode, eventLog, terminal)
	}
	episode.Repository.LaneBPath = reportPath(world.episodeDir, plan.Dependent.Worktree)
	bRuns, bExport, err := observation(ctx, observer, world, "before-dependent")
	if err != nil {
		episode.Lanes = append(episode.Lanes, aResult)
		return finishCollaboration(world, &episode, eventLog, err)
	}
	if err = writeLaneObservation(plan.Dependent, "before", bRuns, bExport); err != nil {
		episode.Lanes = append(episode.Lanes, aResult)
		return finishCollaboration(world, &episode, eventLog, err)
	}
	bEnv, err := collaborationEnvironment(plan.Dependent.CredentialFile, world.lanes[plan.Dependent.LaneID], plan.Dependent.Worktree)
	if err != nil {
		terminal = err
		episode.Lanes = append(episode.Lanes, aResult)
		return finishCollaboration(world, &episode, eventLog, terminal)
	}
	bDone, err := executor.Launch(runCtx, plan.Dependent, world.lanes[plan.Dependent.LaneID], bEnv)
	if err != nil {
		terminal = err
		episode.Lanes = append(episode.Lanes, aResult)
		return finishCollaboration(world, &episode, eventLog, terminal)
	}
	bResult := laneResultForPlan(world, plan.Dependent)
	_ = eventLog.append("lane.started", plan.Dependent.LaneID, "released dependent lane after checkpoint observation")
	if aExecution == nil {
		a, waitErr := receiveCollaborationExecution(runCtx, aDone, world.scenario.Schedule.TimeoutSeconds)
		if waitErr != nil {
			episode.Lanes = append(episode.Lanes, aResult, bResult)
			return finishCollaboration(world, &episode, eventLog, waitErr)
		}
		aExecution = &a
		aResult, err = persistLaneResult(ctx, world, observer, plan.Initial, aResult, a, baselineRunIDs)
		if err != nil {
			episode.Lanes = append(episode.Lanes, aResult, bResult)
			return finishCollaboration(world, &episode, eventLog, err)
		}
	}
	b, waitErr := receiveCollaborationExecution(runCtx, bDone, world.scenario.Schedule.TimeoutSeconds)
	if waitErr != nil {
		episode.Lanes = append(episode.Lanes, aResult, bResult)
		return finishCollaboration(world, &episode, eventLog, waitErr)
	}
	bResult, err = persistLaneResult(ctx, world, observer, plan.Dependent, bResult, b, baselineRunIDs)
	if err != nil {
		episode.Lanes = append(episode.Lanes, aResult, bResult)
		return finishCollaboration(world, &episode, eventLog, err)
	}
	episode.Lanes = append(episode.Lanes, aResult, bResult)
	if aExecution.Err != nil || b.Err != nil {
		terminal = errors.Join(aExecution.Err, b.Err)
	}
	return finishCollaboration(world, &episode, eventLog, terminal)
}

func laneResultForPlan(world collaborationWorld, plan CollaborationLanePlan) LaneResult {
	laneDir := filepath.Dir(plan.EventsPath)
	return LaneResult{LaneID: plan.LaneID, ActorKey: world.lanes[plan.LaneID].ActorKey, Status: "running", StartedAt: time.Now().UTC(), EventsPath: reportPath(world.episodeDir, plan.EventsPath), StderrPath: reportPath(world.episodeDir, plan.StderrPath), FinalResponsePath: reportPath(world.episodeDir, filepath.Join(plan.ResponsesDir, fmt.Sprintf("turn-%03d.txt", len(plan.Processes)))), ProjectExportPath: reportPath(world.episodeDir, filepath.Join(laneDir, "project-export-after.json")), GitResultPath: reportPath(world.episodeDir, filepath.Join(laneDir, "git.json")), ChecksPath: reportPath(world.episodeDir, filepath.Join(laneDir, "checks.json")), CommandPath: reportPath(world.episodeDir, filepath.Join(laneDir, "commands.json"))}
}

func writeLaneCommand(plan CollaborationLanePlan, commands []CollaborationProcessPlan) error {
	// The command plan is replay evidence, not an environment dump. In
	// particular it intentionally excludes CLANKSPACE_CREDENTIALS_FILE.
	if len(commands) == 0 {
		return errors.New("lane execution did not retain any launched commands")
	}
	for _, command := range commands {
		for _, arg := range command.Args {
			if arg == "<thread-id-from-turn-001>" {
				return errors.New("lane execution retained a placeholder instead of the launched resume command")
			}
		}
	}
	return writeJSONExclusive(filepath.Join(filepath.Dir(plan.EventsPath), "commands.json"), commands, 0600)
}

func receiveCollaborationExecution(ctx context.Context, done <-chan collaborationExecution, timeoutSeconds int) (collaborationExecution, error) {
	timer := time.NewTimer(time.Duration(timeoutSeconds) * time.Second)
	defer timer.Stop()
	select {
	case execution, ok := <-done:
		if !ok {
			return collaborationExecution{}, errors.New("lane executor closed without a result")
		}
		return execution, nil
	case <-ctx.Done():
		return collaborationExecution{}, ctx.Err()
	case <-timer.C:
		return collaborationExecution{}, errors.New("lane completion timed out after barrier release")
	}
}

func waitForBarrier(ctx context.Context, world collaborationWorld, observer collaborationObserver, events *controllerEventLog, plan CollaborationLanePlan, done <-chan collaborationExecution, baselineRuns, baselineNotes map[string]bool) (BarrierObservation, *collaborationExecution, error) {
	deadline := time.NewTimer(time.Duration(world.scenario.Schedule.TimeoutSeconds) * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Duration(world.scenario.Schedule.PollIntervalMS) * time.Millisecond)
	defer ticker.Stop()
	var completed *collaborationExecution
	for {
		runs, exported, err := observation(ctx, observer, world, "poll")
		if err != nil {
			return BarrierObservation{}, completed, fmt.Errorf("observe barrier: %w", err)
		}
		run, unique := matchingLaneRun(runs, baselineRuns, world.lanes[plan.LaneID], plan.Worktree)
		if unique {
			if note, found := matchingBarrierNote(exported, baselineNotes, run.ID, world.scenario.Schedule.Barrier); found {
				snapshot := filepath.Join(world.episodeDir, "project-export-barrier.json")
				if err = writeJSONExclusive(snapshot, sanitizeExport(exported), 0600); err != nil {
					return BarrierObservation{}, completed, err
				}
				hash, err := FileHash(snapshot)
				if err != nil {
					return BarrierObservation{}, completed, err
				}
				_ = events.append("barrier.candidate", plan.LaneID, "matching checkpoint projection observed")
				return BarrierObservation{Observed: true, LaneID: plan.LaneID, RunID: run.ID, NoteID: stringValue(note["id"]), Paths: stringSlice(note["paths"]), SnapshotPath: reportPath(world.episodeDir, snapshot), SnapshotHash: hash, ObservedAt: time.Now().UTC()}, completed, nil
			}
		}
		select {
		case execution := <-done:
			completed = &execution
			// The final API poll above is intentionally the only observation after an
			// exited worker; a missing barrier remains a measured failure.
			runs, exported, err = observation(ctx, observer, world, "final")
			if err != nil {
				return BarrierObservation{}, completed, fmt.Errorf("final barrier observation: %w", err)
			}
			run, unique = matchingLaneRun(runs, baselineRuns, world.lanes[plan.LaneID], plan.Worktree)
			if unique {
				if note, found := matchingBarrierNote(exported, baselineNotes, run.ID, world.scenario.Schedule.Barrier); found {
					snapshot := filepath.Join(world.episodeDir, "project-export-barrier.json")
					if err = writeJSONExclusive(snapshot, sanitizeExport(exported), 0600); err != nil {
						return BarrierObservation{}, completed, err
					}
					hash, err := FileHash(snapshot)
					if err != nil {
						return BarrierObservation{}, completed, err
					}
					return BarrierObservation{Observed: true, LaneID: plan.LaneID, RunID: run.ID, NoteID: stringValue(note["id"]), Paths: stringSlice(note["paths"]), SnapshotPath: reportPath(world.episodeDir, snapshot), SnapshotHash: hash, ObservedAt: time.Now().UTC()}, completed, nil
				}
			}
			return BarrierObservation{}, completed, nil
		case <-ctx.Done():
			return BarrierObservation{}, completed, ctx.Err()
		case <-deadline.C:
			return BarrierObservation{}, completed, errors.New("barrier observation timed out")
		case <-ticker.C:
		}
	}
}

func observation(ctx context.Context, observer collaborationObserver, world collaborationWorld, name string) ([]domain.Run, map[string]any, error) {
	runs, err := observer.ListRuns(ctx, world.prepared.ProjectSlug, 500)
	if err != nil {
		return nil, nil, err
	}
	exported, err := observer.ExportProject(ctx, world.prepared.ProjectSlug)
	if err != nil {
		return nil, nil, err
	}
	if name != "poll" {
		if err = writeJSONExclusive(filepath.Join(world.episodeDir, "runs-"+name+".json"), runs, 0600); err != nil {
			return nil, nil, err
		}
		if err = writeJSONExclusive(filepath.Join(world.episodeDir, "project-export-"+name+".json"), sanitizeExport(exported), 0600); err != nil {
			return nil, nil, err
		}
	}
	return runs, exported, nil
}

func writeLaneObservation(plan CollaborationLanePlan, phase string, runs []domain.Run, exported map[string]any) error {
	laneDir := filepath.Dir(plan.EventsPath)
	if err := writeJSONExclusive(filepath.Join(laneDir, "runs-"+phase+".json"), runs, 0600); err != nil {
		return err
	}
	return writeJSONExclusive(filepath.Join(laneDir, "project-export-"+phase+".json"), sanitizeExport(exported), 0600)
}

func matchingLaneRun(runs []domain.Run, baseline map[string]bool, lane PreparedLane, worktree string) (domain.Run, bool) {
	var matches []domain.Run
	for _, run := range runs {
		if !baseline[run.ID] && run.PrincipalID == lane.PrincipalID && run.AgentName == lane.AgentName && run.Harness == lane.Harness && run.Branch == lane.Branch && filepath.Clean(run.Worktree) == filepath.Clean(worktree) {
			matches = append(matches, run)
		}
	}
	if len(matches) != 1 {
		return domain.Run{}, false
	}
	return matches[0], true
}

func matchingBarrierNote(exported map[string]any, baseline map[string]bool, runID string, spec BarrierSpec) (map[string]any, bool) {
	for _, note := range mapSlice(exported["notes"]) {
		if !baseline[stringValue(note["id"])] && stringValue(note["runId"]) == runID && stringValue(note["kind"]) == spec.Kind && pathsOverlap(stringSlice(note["paths"]), spec.RequiredPathOverlap) {
			return note, true
		}
	}
	return nil, false
}

func pathsOverlap(left, right []string) bool {
	for _, l := range left {
		for _, r := range right {
			if l == r || strings.HasPrefix(l, r) || strings.HasPrefix(r, l) {
				return true
			}
		}
	}
	return false
}
func runIDs(runs []domain.Run) map[string]bool {
	ids := map[string]bool{}
	for _, run := range runs {
		ids[run.ID] = true
	}
	return ids
}
func noteIDs(exported map[string]any) map[string]bool {
	ids := map[string]bool{}
	for _, note := range mapSlice(exported["notes"]) {
		ids[stringValue(note["id"])] = true
	}
	return ids
}
func mapSlice(value any) []map[string]any {
	values, _ := value.([]any)
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			result = append(result, item)
		}
	}
	return result
}
func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
func stringSlice(value any) []string {
	values, _ := value.([]any)
	result := []string{}
	for _, value := range values {
		if item, ok := value.(string); ok {
			result = append(result, item)
		}
	}
	return result
}

func createCollaborationWorktree(source, destination, branch, head string) error {
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("lane worktree already exists: %s", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := runGit("", nil, "clone", "--no-hardlinks", "--quiet", source, destination); err != nil {
		return err
	}
	if err := runGit(destination, nil, "checkout", "--quiet", "-B", branch, head); err != nil {
		return err
	}
	return nil
}

func collaborationEnvironment(credentialPath string, lane PreparedLane, worktree string) ([]string, error) {
	if _, err := os.Stat(credentialPath); err != nil {
		return nil, err
	}
	overrides := map[string]string{"CLANKSPACE_CREDENTIALS_FILE": credentialPath, "CLANKSPACE_AGENT": lane.AgentName, "CLANKSPACE_HARNESS": lane.Harness, "CLANKSPACE_HARNESS_VERSION": "codex-cli", "CLANKSPACE_PROVIDER": lane.Provider, "CLANKSPACE_MODEL": lane.Model, "CLANKSPACE_REASONING": lane.Reasoning, "CLANKSPACE_ROLE": lane.Role, "CLANKSPACE_RUN_TYPE": "interactive", "CLANKSPACE_BRANCH": lane.Branch, "CLANKSPACE_WORKTREE": worktree, "CLANKSPACE_INSTRUCTIONS": "AGENTS.md,clankspace:" + lane.SkillHash}
	env := make([]string, 0, len(os.Environ())+len(overrides))
	blocked := map[string]bool{
		"CLANKSPACE_RUN": true, "CLANKSPACE_TOKEN": true,
		"CLANKSPACE_URL": true, "CLANKSPACE_PROJECT": true,
	}
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if blocked[key] {
			continue
		}
		if _, replaced := overrides[key]; !replaced {
			env = append(env, item)
		}
	}
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		env = append(env, key+"="+overrides[key])
	}
	return env, nil
}

func persistLaneResult(ctx context.Context, world collaborationWorld, observer collaborationObserver, plan CollaborationLanePlan, result LaneResult, execution collaborationExecution, baselineRuns map[string]bool) (LaneResult, error) {
	result.ThreadID, result.EndedAt = execution.ThreadID, time.Now().UTC()
	if execution.Err != nil {
		result.Status = "failed"
	} else {
		result.Status = "completed"
	}
	if execution.Err == nil && len(execution.Commands) != len(plan.Processes) {
		return result, errors.New("completed lane did not retain every launched command")
	}
	if err := writeLaneCommand(plan, execution.Commands); err != nil {
		return result, err
	}
	if err := writeExclusive(plan.EventsPath, sanitizeEvents(execution.Events), 0600); err != nil {
		return result, err
	}
	if err := writeExclusive(plan.StderrPath, sanitizeText(execution.Stderr), 0600); err != nil {
		return result, err
	}
	for i, response := range execution.Responses {
		if err := writeExclusive(filepath.Join(plan.ResponsesDir, fmt.Sprintf("turn-%03d.txt", i+1)), sanitizeText(response), 0600); err != nil {
			return result, err
		}
	}
	runs, err := observer.ListRuns(ctx, world.prepared.ProjectSlug, 500)
	if err != nil {
		return result, err
	}
	exported, err := observer.ExportProject(ctx, world.prepared.ProjectSlug)
	if err != nil {
		return result, err
	}
	laneDir := filepath.Dir(plan.EventsPath)
	if err = writeJSONExclusive(filepath.Join(laneDir, "runs-after.json"), runs, 0600); err != nil {
		return result, err
	}
	if err = writeJSONExclusive(filepath.Join(laneDir, "project-export-after.json"), sanitizeExport(exported), 0600); err != nil {
		return result, err
	}
	if run, unique := matchingLaneRun(runs, baselineRuns, world.lanes[plan.LaneID], plan.Worktree); unique {
		result.ObservedRunID = run.ID
	} else {
		return result, errors.New("could not uniquely identify lane run from observed provenance")
	}
	git := captureGitResult(plan.Worktree)
	if err = writeJSONExclusive(filepath.Join(laneDir, "git.json"), git, 0600); err != nil {
		return result, err
	}
	checks := runLaneChecks(ctx, plan.Worktree, world.scenarioLane(plan.LaneID).Task.Checks)
	if err = writeJSONExclusive(filepath.Join(laneDir, "checks.json"), checks, 0600); err != nil {
		return result, err
	}
	for _, check := range checks {
		if check.Error != "" {
			result.Status = "failed"
			if err = writeJSONExclusive(filepath.Join(laneDir, "lane-result.json"), result, 0600); err != nil {
				return result, err
			}
			return result, fmt.Errorf("lane %q required check failed: %s", plan.LaneID, check.Command)
		}
	}
	if err = writeJSONExclusive(filepath.Join(laneDir, "lane-result.json"), result, 0600); err != nil {
		return result, err
	}
	return result, nil
}

func (w collaborationWorld) scenarioLane(laneID string) CollaborationLane {
	for _, lane := range w.scenario.Lanes {
		if lane.ID == laneID {
			return lane
		}
	}
	return CollaborationLane{}
}

type collaborationCheckResult struct {
	Command string `json:"command"`
	Error   string `json:"error,omitempty"`
}

func runLaneChecks(ctx context.Context, worktree string, checks []string) []collaborationCheckResult {
	results := make([]collaborationCheckResult, 0, len(checks))
	for _, check := range checks {
		command := exec.CommandContext(ctx, "sh", "-c", check)
		command.Dir = worktree
		output, err := command.CombinedOutput()
		result := collaborationCheckResult{Command: check}
		if err != nil {
			result.Error = strings.TrimSpace(string(sanitizeText(output)))
			if result.Error == "" {
				result.Error = err.Error()
			}
		}
		results = append(results, result)
	}
	return results
}

type collaborationGitResult struct {
	Head         string   `json:"head,omitempty"`
	Status       string   `json:"status,omitempty"`
	ChangedPaths []string `json:"changedPaths"`
	DiffCheck    string   `json:"diffCheck,omitempty"`
	Error        string   `json:"error,omitempty"`
}

func captureGitResult(repository string) collaborationGitResult {
	head, err := gitOutput(repository, "rev-parse", "HEAD")
	if err != nil {
		return collaborationGitResult{Error: err.Error()}
	}
	status, err := gitOutput(repository, "status", "--short")
	if err != nil {
		return collaborationGitResult{Error: err.Error()}
	}
	paths, err := gitOutput(repository, "diff", "--name-only")
	if err != nil {
		return collaborationGitResult{Error: err.Error()}
	}
	check, err := gitOutput(repository, "diff", "--check")
	if err != nil {
		return collaborationGitResult{Error: err.Error()}
	}
	changed := strings.Fields(strings.TrimSpace(paths))
	return collaborationGitResult{Head: strings.TrimSpace(head), Status: strings.TrimSpace(status), ChangedPaths: changed, DiffCheck: strings.TrimSpace(check)}
}

func finishCollaboration(world collaborationWorld, episode *CollaborationEpisode, events *controllerEventLog, terminal error) (CollaborationEpisode, error) {
	episode.EndedAt = time.Now().UTC()
	completed := 0
	dependentStarted := false
	for _, lane := range episode.Lanes {
		if lane.Status == "completed" {
			completed++
		}
		if lane.LaneID == world.scenario.Schedule.DependentLane {
			dependentStarted = true
		}
	}
	episode.Score = CollaborationScore{BarrierObserved: episode.Barrier.Observed, DependentStarted: dependentStarted, LanesCompleted: completed, Incomplete: terminal != nil || completed != 2}
	if terminal != nil {
		episode.Status = "incomplete"
	} else {
		episode.Status = "completed"
	}
	if err := episode.Validate(); err != nil && terminal == nil {
		terminal = err
		episode.Status = "failed"
		episode.Score.Incomplete = true
	}
	if err := events.append("episode.finished", "", episode.Status); err != nil {
		terminal = errors.Join(terminal, err)
	}
	if err := writeJSONExclusive(filepath.Join(world.episodeDir, "deterministic-score.json"), episode.Score, 0600); err != nil {
		terminal = errors.Join(terminal, err)
	}
	if err := writeJSONExclusive(filepath.Join(world.episodeDir, "collaboration.json"), episode, 0600); err != nil {
		terminal = errors.Join(terminal, err)
	}
	if err := writeCollaborationDossier(filepath.Join(world.episodeDir, "dossier.html"), *episode); err != nil {
		terminal = errors.Join(terminal, err)
	}
	if err := writeSHA256Sums(world.episodeDir); err != nil {
		terminal = errors.Join(terminal, err)
	}
	return *episode, terminal
}

type controllerEventLog struct {
	file *os.File
	mu   sync.Mutex
}

func newControllerEventLog(path string) (*controllerEventLog, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	return &controllerEventLog{file: f}, nil
}
func (l *controllerEventLog) append(kind, lane, message string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	event := ControllerEvent{At: time.Now().UTC(), Type: kind, LaneID: lane, Message: message}
	if err := event.Validate(); err != nil {
		return err
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err = l.file.Write(append(body, '\n')); err != nil {
		return err
	}
	return l.file.Sync()
}
func (l *controllerEventLog) Close() error { return l.file.Close() }

func reportPath(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}

func sanitizeEvents(input []byte) []byte {
	scanner := bufio.NewScanner(bytes.NewReader(input))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var output bytes.Buffer
	for scanner.Scan() {
		var item any
		if json.Unmarshal(scanner.Bytes(), &item) != nil {
			continue
		}
		if containsReasoning(item) {
			continue
		}
		sanitized := sanitizeValue(item)
		body, err := json.Marshal(sanitized)
		if err == nil {
			output.Write(body)
			output.WriteByte('\n')
		}
	}
	return output.Bytes()
}
func containsReasoning(value any) bool {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if strings.Contains(strings.ToLower(key), "reasoning") {
				return true
			}
			if key == "type" && stringValue(child) == "reasoning" {
				return true
			}
			if containsReasoning(child) {
				return true
			}
		}
	case []any:
		for _, child := range value {
			if containsReasoning(child) {
				return true
			}
		}
	}
	return false
}
func sanitizeValue(value any) any {
	switch value := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, child := range value {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "token") || strings.Contains(lower, "credential") || strings.Contains(lower, "authorization") || strings.Contains(lower, "api_key") || strings.Contains(lower, "apikey") || lower == "secret" || strings.HasSuffix(lower, "_secret") {
				out[key] = "[redacted]"
			} else {
				out[key] = sanitizeValue(child)
			}
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, child := range value {
			out[i] = sanitizeValue(child)
		}
		return out
	case string:
		return string(sanitizeText([]byte(value)))
	default:
		return value
	}
}

func sanitizeExport(exported map[string]any) map[string]any {
	value, ok := sanitizeValue(exported).(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

var (
	assignmentSecretPattern = regexp.MustCompile(`(?i)\b(?:openai_api_key|api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|password)\s*[:=]\s*(?:"[^"]*"|'[^']*'|[^\s,}\]]+)`)
	bearerSecretPattern     = regexp.MustCompile(`(?i)\bauthorization\s*:\s*bearer\s+[^\s"']+`)
	jwtSecretPattern        = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
	credentialJSONPattern   = regexp.MustCompile(`(?i)"(token|api[_-]?key|client[_-]?secret|password)"\s*:\s*"[^"]*"`)
)

func sanitizeText(input []byte) []byte {
	text := string(input)
	for _, marker := range []string{"Authorization: Bearer ", "authorization: bearer ", "CLANKSPACE_TOKEN=", "clankspace_token=", "token=", "Token="} {
		for {
			index := strings.Index(text, marker)
			if index < 0 {
				break
			}
			end := index + len(marker)
			for end < len(text) && !strings.ContainsRune(" \t\r\n\"'", rune(text[end])) {
				end++
			}
			text = text[:index] + "[redacted]" + text[end:]
		}
	}
	text = assignmentSecretPattern.ReplaceAllString(text, "[redacted]")
	text = bearerSecretPattern.ReplaceAllString(text, "[redacted]")
	text = jwtSecretPattern.ReplaceAllString(text, "[redacted]")
	text = credentialJSONPattern.ReplaceAllString(text, `"$1":"[redacted]"`)
	return []byte(text)
}

func writeCollaborationDossier(path string, episode CollaborationEpisode) error {
	body, err := json.MarshalIndent(episode, "", "  ")
	if err != nil {
		return err
	}
	links := []string{"schedule.json", "controller-events.jsonl", "runs-before.json", "project-export-before.json", "barrier.json", "deterministic-score.json", "collaboration.json", "SHA256SUMS"}
	for _, lane := range episode.Lanes {
		for _, link := range []string{lane.EventsPath, lane.StderrPath, lane.FinalResponsePath, lane.ProjectExportPath, lane.GitResultPath, lane.ChecksPath, lane.CommandPath} {
			if link != "" {
				links = append(links, link)
			}
		}
	}
	sort.Strings(links)
	var list strings.Builder
	for _, link := range links {
		list.WriteString("<li><a href=\"")
		list.WriteString(html.EscapeString(link))
		list.WriteString("\">")
		list.WriteString(html.EscapeString(link))
		list.WriteString("</a></li>")
	}
	page := "<!doctype html><meta charset=\"utf-8\"><title>ClankSpace collaboration dossier</title><style>body{font:16px system-ui;margin:3rem;max-width:72rem;color:#172033}pre{background:#f4f6f8;padding:1rem;overflow:auto}a{color:#075f9f}</style><h1>ClankSpace collaboration dossier</h1><p>Status: <strong>" + html.EscapeString(episode.Status) + "</strong>. This is offline evidence; ClankSpace records remain advisory.</p><h2>Artifacts</h2><ul>" + list.String() + "</ul><h2>Episode</h2><pre>" + html.EscapeString(string(body)) + "</pre>"
	return writeExclusive(path, []byte(page), 0600)
}

func writeSHA256Sums(root string) error {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative := filepath.ToSlash(reportPath(root, path))
		if entry.IsDir() && (relative == "repositories" || strings.HasPrefix(relative, "repositories/")) {
			return filepath.SkipDir
		}
		if entry.IsDir() || filepath.Base(path) == "SHA256SUMS" || strings.Contains(filepath.ToSlash(path), "/secrets/") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(files)
	var body strings.Builder
	for _, path := range files {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		relative := reportPath(root, path)
		body.WriteString(hex.EncodeToString(hash.Sum(nil)))
		body.WriteString("  ")
		body.WriteString(relative)
		body.WriteByte('\n')
	}
	return writeExclusive(filepath.Join(root, "SHA256SUMS"), []byte(body.String()), 0600)
}
