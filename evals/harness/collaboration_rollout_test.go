package harness

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ShamanicArts/clankspace/internal/domain"
)

func TestCollaborationDryRunPlansExactProcessesWithoutCreatingWorktrees(t *testing.T) {
	world, options := collaborationTestWorld(t)
	plan, err := PlanCollaborationRollout(options)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Initial.Processes) != 2 || len(plan.Dependent.Processes) != 2 {
		t.Fatalf("unexpected process plan: %#v", plan)
	}
	if plan.Initial.Processes[0].Args[0] != options.CodexBin || !strings.Contains(strings.Join(plan.Initial.Processes[0].Args, " "), "workspace-write") {
		t.Fatalf("initial process is not exact Codex workspace-write command: %#v", plan.Initial.Processes[0])
	}
	if _, err := os.Stat(plan.Initial.Worktree); !os.IsNotExist(err) {
		t.Fatalf("dry-run created initial worktree %q: %v", plan.Initial.Worktree, err)
	}
	if _, err := os.Stat(world.episodeDir); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote an episode: %v", err)
	}
}

func TestCollaborationGateReleasesDependentOnlyAfterMatchingBarrier(t *testing.T) {
	world, options := collaborationTestWorld(t)
	plan, err := PlanCollaborationRollout(options)
	if err != nil {
		t.Fatal(err)
	}
	observer := &fakeCollaborationObserver{}
	executor := &fakeCollaborationExecutor{observer: observer}
	episode, err := runCollaboration(context.Background(), world, plan, options, observer, executor)
	if err != nil {
		t.Fatal(err)
	}
	if !episode.Barrier.Observed || episode.Status != "completed" || !episode.Score.DependentStarted || episode.Score.LanesCompleted != 2 {
		t.Fatalf("unexpected episode: %+v", episode)
	}
	if err := episode.Validate(); err != nil {
		t.Fatal(err)
	}
	if executor.launches != 2 || executor.secondSawBarrier != true {
		t.Fatalf("dependent lane was not held by barrier: launches=%d sawBarrier=%v", executor.launches, executor.secondSawBarrier)
	}
	for _, path := range []string{"schedule.json", "controller-events.jsonl", "barrier.json", "deterministic-score.json", "collaboration.json", "dossier.html", "SHA256SUMS", "lanes/lane-a/events.jsonl", "lanes/lane-a/git.json", "lanes/lane-a/checks.json", "lanes/lane-a/commands.json", "lanes/lane-b/git.json", "lanes/lane-b/checks.json", "lanes/lane-b/commands.json"} {
		if _, err := os.Stat(filepath.Join(world.episodeDir, path)); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
	for _, path := range []string{"project-export-before.json", "lanes/lane-a/project-export-after.json"} {
		body, readErr := os.ReadFile(filepath.Join(world.episodeDir, path))
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, secret := range []string{"OPENAI_API_KEY=sk-test-secret", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature", `"api_key":"secret"`} {
			if strings.Contains(string(body), secret) {
				t.Fatalf("project export leaked %q in %s: %s", secret, path, body)
			}
		}
	}
	for _, forbidden := range []string{"token", "credential", "reasoning"} {
		body, err := os.ReadFile(filepath.Join(world.episodeDir, "dossier.html"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(string(body)), forbidden) {
			t.Fatalf("dossier leaked %q", forbidden)
		}
	}
	for _, path := range []string{"lanes/lane-a/events.jsonl", "lanes/lane-a/responses/turn-001.txt", "lanes/lane-a/stderr.log"} {
		body, err := os.ReadFile(filepath.Join(world.episodeDir, path))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(string(body)), "secret") || strings.Contains(strings.ToLower(string(body)), "reasoning") {
			t.Fatalf("observable artifact leaked sensitive or hidden content in %s: %s", path, body)
		}
	}
	if err := verifySums(filepath.Join(world.episodeDir, "SHA256SUMS")); err != nil {
		t.Fatal(err)
	}
}

func TestCollaborationEpisodePinsExactCommandsAndReplayHashes(t *testing.T) {
	world, options := collaborationTestWorld(t)
	launcher := filepath.Join(filepath.Dir(world.episodeDir), "codex-eval")
	config := filepath.Join(filepath.Dir(world.episodeDir), "server-config.json")
	if err := os.MkdirAll(filepath.Dir(launcher), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(launcher, []byte("launcher"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte(`{"commit":"abc"}`), 0600); err != nil {
		t.Fatal(err)
	}
	options.CodexBin = launcher
	replay, err := buildCollaborationReplay(launcher, "http://example.invalid", config, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	world.replay = replay
	plan, err := PlanCollaborationRollout(options)
	if err != nil {
		t.Fatal(err)
	}
	observer := &fakeCollaborationObserver{}
	episode, err := runCollaboration(context.Background(), world, plan, options, observer, &fakeCollaborationExecutor{observer: observer})
	if err != nil {
		t.Fatal(err)
	}
	if episode.Replay != replay {
		t.Fatalf("replay provenance was not retained: got=%+v want=%+v", episode.Replay, replay)
	}
	commands, err := os.ReadFile(filepath.Join(world.episodeDir, episode.Lanes[0].CommandPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(commands), launcher) || !strings.Contains(string(commands), "thread-lane-a") || strings.Contains(string(commands), "<thread-id-from-turn-001>") || strings.Contains(strings.ToLower(string(commands)), "credential") {
		t.Fatalf("exact command evidence is incomplete or unsafe: %s", commands)
	}
	if err := verifySums(filepath.Join(world.episodeDir, "SHA256SUMS")); err != nil {
		t.Fatal(err)
	}
}

func TestCollaborationRejectsWrongBarrierAndNeverStartsDependent(t *testing.T) {
	world, options := collaborationTestWorld(t)
	world.scenario.Schedule.TimeoutSeconds = 1
	plan, err := PlanCollaborationRollout(options)
	if err != nil {
		t.Fatal(err)
	}
	observer := &fakeCollaborationObserver{wrongPath: true}
	executor := &fakeCollaborationExecutor{observer: observer}
	episode, err := runCollaboration(context.Background(), world, plan, options, observer, executor)
	if err == nil || episode.Status != "incomplete" {
		t.Fatalf("wrong barrier unexpectedly completed: episode=%+v err=%v", episode, err)
	}
	if executor.launches != 1 || episode.Score.DependentStarted {
		t.Fatalf("dependent lane started without matching barrier: %+v", episode)
	}
	if _, statErr := os.Stat(plan.Dependent.Worktree); !os.IsNotExist(statErr) {
		t.Fatalf("dependent worktree exists without barrier: %v", statErr)
	}
}

func TestCollaborationFailsWhenRequiredTaskCheckFails(t *testing.T) {
	world, options := collaborationTestWorld(t)
	world.scenario.Lanes[1].Task.Checks = []string{"false"}
	plan, err := PlanCollaborationRollout(options)
	if err != nil {
		t.Fatal(err)
	}
	observer := &fakeCollaborationObserver{}
	executor := &fakeCollaborationExecutor{observer: observer}
	episode, err := runCollaboration(context.Background(), world, plan, options, observer, executor)
	if err == nil || episode.Status != "incomplete" || episode.Lanes[1].Status != "failed" {
		t.Fatalf("failed required check did not make episode incomplete: episode=%+v err=%v", episode, err)
	}
}

func TestCollaborationCancellationAfterBarrierDoesNotHang(t *testing.T) {
	world, options := collaborationTestWorld(t)
	plan, err := PlanCollaborationRollout(options)
	if err != nil {
		t.Fatal(err)
	}
	observer := &fakeCollaborationObserver{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	executor := &stalledCollaborationExecutor{fakeCollaborationExecutor: fakeCollaborationExecutor{observer: observer}, dependentStarted: make(chan struct{})}
	go func() {
		<-executor.dependentStarted
		cancel()
	}()
	episode, err := runCollaboration(ctx, world, plan, options, observer, executor)
	if err == nil || episode.Status != "incomplete" || executor.launches != 2 {
		t.Fatalf("cancelled dependent wait was not retained as incomplete: episode=%+v err=%v launches=%d", episode, err, executor.launches)
	}
}

func TestCollaborationExecutorReplacesTransientResponseWithSanitizedArtifact(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "fake-codex")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '{\"type\":\"thread.started\",\"thread_id\":\"thread-a\"}\\n'\nprintf 'token=secret' > \"$1\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	response := filepath.Join(root, "responses", "turn-001.txt")
	plan := CollaborationLanePlan{LaneID: "lane-a", Worktree: root, Processes: []CollaborationProcessPlan{{Args: []string{script, response}, Directory: root, ResponsePath: response}}}
	done, err := (realCollaborationExecutor{}).Launch(context.Background(), plan, PreparedLane{}, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	result := <-done
	if result.Err != nil || len(result.Responses) != 1 || string(result.Responses[0]) != "token=secret" {
		t.Fatalf("unexpected executor result: %+v", result)
	}
	if _, err = os.Stat(response); !os.IsNotExist(err) {
		t.Fatalf("transient response remained in report path: %v", err)
	}
}

func TestCollaborationEnvironmentRemovesAmbientProjectAuthority(t *testing.T) {
	t.Setenv("CLANKSPACE_TOKEN", "ambient-token")
	t.Setenv("CLANKSPACE_URL", "https://ambient.invalid")
	t.Setenv("CLANKSPACE_PROJECT", "ambient-project")
	t.Setenv("CLANKSPACE_RUN", "ambient-run")
	credential := filepath.Join(t.TempDir(), "lane.json")
	if err := os.WriteFile(credential, []byte(`{"credentials":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	env, err := collaborationEnvironment(credential, PreparedLane{AgentName: "A", Harness: "codex", Provider: "openai", Model: "luna", Role: "primary"}, "/tmp/lane")
	if err != nil {
		t.Fatal(err)
	}
	joined := "\n" + strings.Join(env, "\n") + "\n"
	for _, forbidden := range []string{"\nCLANKSPACE_TOKEN=", "\nCLANKSPACE_URL=", "\nCLANKSPACE_PROJECT=", "\nCLANKSPACE_RUN="} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("lane environment retained ambient authority %q", forbidden)
		}
	}
}

func TestCompletedEpisodeRequiresCompleteLaneEvidence(t *testing.T) {
	episode := CollaborationEpisode{
		SchemaVersion: 2, EpisodeID: "episode", ScenarioID: "scenario", ScenarioHash: "hash", Status: "completed",
		SchedulePath: "schedule.json", ControllerEvents: "events.jsonl",
		Repository: RepositoryResult{LaneAPath: "repositories/a", LaneBPath: "repositories/b"},
		Barrier:    BarrierObservation{Observed: true},
		Score:      CollaborationScore{BarrierObserved: true, DependentStarted: true, LanesCompleted: 2},
	}
	if err := episode.Validate(); err == nil || !strings.Contains(err.Error(), "completion gates") {
		t.Fatalf("incomplete completed episode was accepted: %v", err)
	}
}

type fakeCollaborationObserver struct {
	runs      []domain.Run
	exported  map[string]any
	wrongPath bool
}

func (o *fakeCollaborationObserver) ListRuns(context.Context, string, int) ([]domain.Run, error) {
	return append([]domain.Run(nil), o.runs...), nil
}
func (o *fakeCollaborationObserver) ExportProject(context.Context, string) (map[string]any, error) {
	return cloneExport(o.exported), nil
}

type fakeCollaborationExecutor struct {
	observer         *fakeCollaborationObserver
	launches         int
	secondSawBarrier bool
}

type stalledCollaborationExecutor struct {
	fakeCollaborationExecutor
	dependentStarted chan struct{}
}

func (e *stalledCollaborationExecutor) Launch(ctx context.Context, plan CollaborationLanePlan, lane PreparedLane, env []string) (<-chan collaborationExecution, error) {
	if e.launches == 0 {
		return e.fakeCollaborationExecutor.Launch(ctx, plan, lane, env)
	}
	e.launches++
	close(e.dependentStarted)
	return make(chan collaborationExecution), nil
}

func (e *fakeCollaborationExecutor) Launch(_ context.Context, plan CollaborationLanePlan, lane PreparedLane, _ []string) (<-chan collaborationExecution, error) {
	e.launches++
	if e.launches == 1 {
		e.observer.runs = append(e.observer.runs, domain.Run{ID: "run-a", PrincipalID: lane.PrincipalID, AgentName: lane.AgentName, Harness: lane.Harness, Branch: lane.Branch, Worktree: plan.Worktree})
		path := "src/coordination/"
		if e.observer.wrongPath {
			path = "other/"
		}
		e.observer.exported = map[string]any{"notes": []any{map[string]any{"id": "seed", "runId": "seed-run", "kind": "checkpoint", "paths": []any{"src/coordination/"}, "summary": "OPENAI_API_KEY=sk-test-secret eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature \"api_key\":\"secret\""}, map[string]any{"id": "note-a", "runId": "run-a", "kind": "checkpoint", "paths": []any{path}}}}
	} else {
		e.secondSawBarrier = len(mapSlice(e.observer.exported["notes"])) == 2
		e.observer.runs = append(e.observer.runs, domain.Run{ID: "run-b", PrincipalID: lane.PrincipalID, AgentName: lane.AgentName, Harness: lane.Harness, Branch: lane.Branch, Worktree: plan.Worktree})
	}
	ch := make(chan collaborationExecution, 1)
	threadID := "thread-" + plan.LaneID
	commands := make([]CollaborationProcessPlan, 0, len(plan.Processes))
	for _, process := range plan.Processes {
		args := append([]string{}, process.Args...)
		for i := range args {
			if args[i] == "<thread-id-from-turn-001>" {
				args[i] = threadID
			}
		}
		commands = append(commands, CollaborationProcessPlan{Args: args, Directory: process.Directory, ResponsePath: process.ResponsePath})
	}
	ch <- collaborationExecution{ThreadID: threadID, Events: []byte(`{"type":"thread.started","thread_id":"thread"}` + "\n" + `{"type":"item.completed","item":{"type":"reasoning","text":"hidden"}}` + "\n"), Stderr: []byte("token=not-retained\n"), Responses: [][]byte{[]byte("public response token=not-retained")}, Commands: commands}
	close(ch)
	return ch, nil
}

func collaborationTestWorld(t *testing.T) (collaborationWorld, CollaborationRolloutOptions) {
	t.Helper()
	root := t.TempDir()
	base := filepath.Join(root, "baseline")
	if err := os.MkdirAll(base, 0700); err != nil {
		t.Fatal(err)
	}
	if err := runGit(base, nil, "init", "--quiet", "--initial-branch=main"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "README.md"), []byte("fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "LICENSE"), []byte("fixture license\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runGit(base, []string{"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.invalid"}, "add", "README.md", "LICENSE"); err != nil {
		t.Fatal(err)
	}
	if err := runGit(base, []string{"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.invalid"}, "commit", "--quiet", "-m", "fixture"); err != nil {
		t.Fatal(err)
	}
	head, err := gitOutput(base, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	head = strings.TrimSpace(head)
	scenario := CollaborationScenario{SchemaVersion: 2, ID: "dev-collaboration-test-001", Split: "dev", Category: "event-gated-collaboration", Project: ScenarioProject{Slug: "coordination-test", Name: "Coordination test", RepositoryProfile: "real-snapshot", Paths: []string{"src/coordination/"}}, Repository: RepositoryFixture{SnapshotID: "snapshot-1", BaseRef: head}, SourceEvidence: SourceEvidence{RepositoryURL: "https://example.invalid/test", License: "MIT", LicenseFile: "LICENSE", LicenseFileHash: strings.Repeat("a", 64), SourceCommit: strings.Repeat("1", 40), SnapshotID: "snapshot-1", SnapshotHead: head, BundleHash: strings.Repeat("b", 64), SyntheticOverlay: true}, Actors: []Actor{{Key: "agent-a", PrincipalName: "A", AgentName: "Agent A", Harness: "codex-cli", Provider: "openai", Model: "test-model", Role: "primary"}, {Key: "agent-b", PrincipalName: "B", AgentName: "Agent B", Harness: "codex-cli", Provider: "openai", Model: "test-model", Role: "primary"}}, Records: []Record{{ID: "record", ActorKey: "agent-a", Kind: "observation", Title: "record", Summary: "record", Rationale: "record", Status: "current", LedBy: "agent", DirectionBasis: "autonomous_agent_judgment", Paths: []string{"src/coordination/"}}}, Lanes: []CollaborationLane{{ID: "lane-a", ActorKey: "agent-a", Branch: "eval/lane-a", PriorUserTurns: []ConversationTurn{{Role: "user", Text: "inspect"}}, Task: LaneTask{Objective: "checkpoint", UserRequest: "checkpoint", Paths: []string{"src/coordination/"}, Checks: []string{"git diff --check"}}, LedgerOracle: Oracle{ExpectedBehavior: "proceed", MaterialReason: "barrier"}}, {ID: "lane-b", ActorKey: "agent-b", Branch: "eval/lane-b", PriorUserTurns: []ConversationTurn{{Role: "user", Text: "retrieve"}}, Task: LaneTask{Objective: "follow", UserRequest: "follow", Paths: []string{"src/coordination/"}, Checks: []string{"git diff --check"}}, LedgerOracle: Oracle{ExpectedBehavior: "inspect", MaterialReason: "barrier"}}}, Schedule: EventGatedSchedule{InitialLane: "lane-a", DependentLane: "lane-b", TimeoutSeconds: 1, PollIntervalMS: 50, Barrier: BarrierSpec{EventType: "note.recorded", Kind: "checkpoint", RequiredPathOverlap: []string{"src/coordination/"}}}, Generation: Generation{CurriculumVersion: "v2", Seed: "seed", GeneratorProvider: "test", GeneratorModel: "test"}}
	licenseHash, err := FileHash(filepath.Join(base, "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	scenario.SourceEvidence.LicenseFileHash = licenseHash
	canonical, err := json.MarshalIndent(scenario, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	canonical = append(canonical, '\n')
	dir := filepath.Join(root, "world")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scenario.json"), canonical, 0600); err != nil {
		t.Fatal(err)
	}
	prepared := CollaborationPreparedWorld{SchemaVersion: 2, ScenarioID: scenario.ID, ScenarioHash: ContentHash(canonical), CorpusVersion: "v2", Split: "dev", ProjectSlug: "eval-test", ProjectID: "project", RepositoryHead: head, SkillHash: "skill", Lanes: []PreparedLane{{LaneID: "lane-a", ActorKey: "agent-a", PrincipalID: "principal-a", Branch: "eval/lane-a", AgentName: "Agent A", Harness: "codex-cli", Provider: "openai", Model: "test-model", Role: "primary", RepositoryHead: head, SkillHash: "skill", RepositoryPath: "repositories/lane-a", ArtifactPath: "lanes/lane-a.json"}, {LaneID: "lane-b", ActorKey: "agent-b", PrincipalID: "principal-b", Branch: "eval/lane-b", AgentName: "Agent B", Harness: "codex-cli", Provider: "openai", Model: "test-model", Role: "primary", RepositoryHead: head, SkillHash: "skill", RepositoryPath: "repositories/lane-b", ArtifactPath: "lanes/lane-b.json"}}}
	if err := writeJSONExclusive(filepath.Join(dir, "prepared.json"), prepared, 0600); err != nil {
		t.Fatal(err)
	}
	credentials := filepath.Join(root, "secrets")
	if err := os.MkdirAll(credentials, 0700); err != nil {
		t.Fatal(err)
	}
	for _, lane := range []string{"lane-a", "lane-b"} {
		if err := os.WriteFile(filepath.Join(credentials, lane+".json"), []byte(`{"credentials":[{"url":"http://example.invalid","project":"eval-test","token":"secret"}]}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	options := CollaborationRolloutOptions{PreparedPath: filepath.Join(dir, "prepared.json"), RepositoryPath: base, CredentialsDir: credentials, CodexBin: "/opt/codex-eval", EpisodeID: "episode-test"}
	world, err := loadCollaborationWorld(options)
	if err != nil {
		t.Fatal(err)
	}
	return world, options
}

func cloneExport(source map[string]any) map[string]any {
	body, _ := json.Marshal(source)
	var result map[string]any
	_ = json.Unmarshal(body, &result)
	return result
}

func verifySums(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	root := filepath.Dir(path)
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			return os.ErrInvalid
		}
		hash, err := FileHash(filepath.Join(root, parts[1]))
		if err != nil {
			return err
		}
		if hash != parts[0] {
			return os.ErrInvalid
		}
	}
	return nil
}
