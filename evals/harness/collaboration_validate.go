package harness

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var (
	gitCommitPattern = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	sha256Pattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Validate keeps collaboration-v2 rules separate from Scenario.Validate so
// v1 loading and rollout semantics cannot be weakened accidentally.
func (s CollaborationScenario) Validate() error {
	if s.SchemaVersion != 2 {
		return fmt.Errorf("schemaVersion must be 2, got %d", s.SchemaVersion)
	}
	if !scenarioIDPattern.MatchString(s.ID) {
		return errors.New("id must contain only lowercase letters, numbers, and hyphens")
	}
	if !slices.Contains([]string{"train", "dev", "holdout"}, s.Split) {
		return fmt.Errorf("unsupported split %q", s.Split)
	}
	if strings.TrimSpace(s.Category) == "" {
		return errors.New("category is required")
	}
	if !scenarioIDPattern.MatchString(s.Project.Slug) {
		return errors.New("project.slug must be a lowercase slug")
	}
	if s.Project.RepositoryProfile != "real-snapshot" {
		return errors.New("collaboration scenarios require project.repositoryProfile real-snapshot")
	}
	if err := validatePaths(s.Project.Paths); err != nil {
		return fmt.Errorf("project paths: %w", err)
	}
	if err := s.validateSourceEvidence(); err != nil {
		return err
	}
	if strings.TrimSpace(s.Repository.SnapshotID) == "" {
		return errors.New("real-snapshot repositories require repository.snapshotId")
	}
	if s.Repository.SnapshotID != s.SourceEvidence.SnapshotID {
		return errors.New("repository.snapshotId must match sourceEvidence.snapshotId")
	}
	if err := (Scenario{Repository: s.Repository}).validateRepository(); err != nil {
		return err
	}

	actors, err := validateCollaborationActors(s.Actors)
	if err != nil {
		return err
	}
	records, err := validateCollaborationRecords(s.Records, actors)
	if err != nil {
		return err
	}
	trajectories, err := validateCollaborationTrajectories(s.Trajectories, actors)
	if err != nil {
		return err
	}
	if len(s.Lanes) != 2 {
		return fmt.Errorf("collaboration scenarios require exactly two lanes, got %d", len(s.Lanes))
	}
	laneIDs := map[string]struct{}{}
	laneActors := map[string]struct{}{}
	for i, lane := range s.Lanes {
		if !aliasPattern.MatchString(lane.ID) {
			return fmt.Errorf("lanes[%d].id is invalid", i)
		}
		if _, exists := laneIDs[lane.ID]; exists {
			return fmt.Errorf("duplicate lane id %q", lane.ID)
		}
		laneIDs[lane.ID] = struct{}{}
		if _, exists := actors[lane.ActorKey]; !exists {
			return fmt.Errorf("lane %q references unknown actor %q", lane.ID, lane.ActorKey)
		}
		if _, exists := laneActors[lane.ActorKey]; exists {
			return fmt.Errorf("lanes must use distinct actors; %q is repeated", lane.ActorKey)
		}
		laneActors[lane.ActorKey] = struct{}{}
		if strings.TrimSpace(lane.Branch) == "" {
			return fmt.Errorf("lane %q requires a branch", lane.ID)
		}
		if err := validatePriorUserTurns(lane.PriorUserTurns); err != nil {
			return fmt.Errorf("lane %q: %w", lane.ID, err)
		}
		if err := validateLaneTask(lane.Task); err != nil {
			return fmt.Errorf("lane %q: %w", lane.ID, err)
		}
		if err := validateLedgerOracle(lane.LedgerOracle, records, trajectories); err != nil {
			return fmt.Errorf("lane %q ledgerOracle: %w", lane.ID, err)
		}
	}
	if err := s.Schedule.Validate(laneIDs); err != nil {
		return err
	}
	if s.Generation.CurriculumVersion == "" || s.Generation.Seed == "" || s.Generation.GeneratorProvider == "" || s.Generation.GeneratorModel == "" {
		return errors.New("generation provenance is incomplete")
	}
	return nil
}

func (s CollaborationScenario) validateSourceEvidence() error {
	if parsed, err := url.ParseRequestURI(s.SourceEvidence.RepositoryURL); err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return errors.New("sourceEvidence.repositoryUrl must be an https URL")
	}
	if s.SourceEvidence.License != "MIT" {
		return fmt.Errorf("sourceEvidence.license must be MIT, got %q", s.SourceEvidence.License)
	}
	if err := validateRelativePath(s.SourceEvidence.LicenseFile); err != nil {
		return fmt.Errorf("sourceEvidence.licenseFile: %w", err)
	}
	if !sha256Pattern.MatchString(s.SourceEvidence.LicenseFileHash) {
		return errors.New("sourceEvidence.licenseFileHash must be a lowercase SHA-256")
	}
	if !gitCommitPattern.MatchString(s.SourceEvidence.SourceCommit) {
		return errors.New("sourceEvidence.sourceCommit must be a lowercase Git commit hash")
	}
	if strings.TrimSpace(s.SourceEvidence.SnapshotID) == "" {
		return errors.New("sourceEvidence.snapshotId is required")
	}
	if !gitCommitPattern.MatchString(s.SourceEvidence.SnapshotHead) {
		return errors.New("sourceEvidence.snapshotHead must be a lowercase Git commit hash")
	}
	if !sha256Pattern.MatchString(s.SourceEvidence.BundleHash) {
		return errors.New("sourceEvidence.bundleHash must be a lowercase SHA-256")
	}
	if s.SourceEvidence.HistoricalClaim {
		return errors.New("sourceEvidence.historicalClaim must be false")
	}
	if !s.SourceEvidence.SyntheticOverlay {
		return errors.New("sourceEvidence.syntheticOverlay must be true")
	}
	return nil
}

func validateCollaborationActors(input []Actor) (map[string]struct{}, error) {
	actors := make(map[string]struct{}, len(input))
	for i, actor := range input {
		if !aliasPattern.MatchString(actor.Key) {
			return nil, fmt.Errorf("actors[%d].key is invalid", i)
		}
		if _, exists := actors[actor.Key]; exists {
			return nil, fmt.Errorf("duplicate actor key %q", actor.Key)
		}
		if strings.TrimSpace(actor.PrincipalName) == "" || strings.TrimSpace(actor.AgentName) == "" || strings.TrimSpace(actor.Harness) == "" || strings.TrimSpace(actor.Provider) == "" || strings.TrimSpace(actor.Model) == "" {
			return nil, fmt.Errorf("actor %q has incomplete provenance", actor.Key)
		}
		if !slices.Contains([]string{"primary", "subagent", "reviewer", "automation", "integration"}, actor.Role) {
			return nil, fmt.Errorf("actor %q has unsupported role %q", actor.Key, actor.Role)
		}
		actors[actor.Key] = struct{}{}
	}
	if len(actors) < 2 {
		return nil, errors.New("collaboration scenarios require at least two actors")
	}
	return actors, nil
}

func validateCollaborationRecords(input []Record, actors map[string]struct{}) (map[string]struct{}, error) {
	if len(input) == 0 {
		return nil, errors.New("at least one seeded record is required")
	}
	records := map[string]struct{}{}
	for i, record := range input {
		if !aliasPattern.MatchString(record.ID) {
			return nil, fmt.Errorf("records[%d].id is invalid", i)
		}
		if _, exists := records[record.ID]; exists {
			return nil, fmt.Errorf("duplicate record id %q", record.ID)
		}
		if _, exists := actors[record.ActorKey]; !exists {
			return nil, fmt.Errorf("record %q references unknown actor %q", record.ID, record.ActorKey)
		}
		if !slices.Contains([]string{"current", "superseded"}, record.Status) {
			return nil, fmt.Errorf("record %q status %q cannot yet be seeded through the public API", record.ID, record.Status)
		}
		if err := validatePaths(record.Paths); err != nil {
			return nil, fmt.Errorf("record %q: %w", record.ID, err)
		}
		records[record.ID] = struct{}{}
	}
	return records, nil
}

func validateCollaborationTrajectories(input []Trajectory, actors map[string]struct{}) (map[string]struct{}, error) {
	trajectories := map[string]struct{}{}
	for i, trajectory := range input {
		if !aliasPattern.MatchString(trajectory.ID) {
			return nil, fmt.Errorf("trajectories[%d].id is invalid", i)
		}
		if _, exists := trajectories[trajectory.ID]; exists {
			return nil, fmt.Errorf("duplicate trajectory id %q", trajectory.ID)
		}
		if _, exists := actors[trajectory.ActorKey]; !exists {
			return nil, fmt.Errorf("trajectory %q references unknown actor %q", trajectory.ID, trajectory.ActorKey)
		}
		if trajectory.Status != "active" {
			return nil, fmt.Errorf("trajectory %q status %q cannot yet be seeded through the public API", trajectory.ID, trajectory.Status)
		}
		if err := validatePaths(trajectory.Paths); err != nil {
			return nil, fmt.Errorf("trajectory %q: %w", trajectory.ID, err)
		}
		trajectories[trajectory.ID] = struct{}{}
	}
	return trajectories, nil
}

func validatePriorUserTurns(turns []ConversationTurn) error {
	if len(turns) == 0 {
		return errors.New("at least one prior user turn is required")
	}
	for i, turn := range turns {
		if turn.Role != "user" || strings.TrimSpace(turn.Text) == "" {
			return fmt.Errorf("priorUserTurns[%d] must be a non-empty user turn", i)
		}
	}
	return nil
}

func validateLaneTask(task LaneTask) error {
	if strings.TrimSpace(task.Objective) == "" || strings.TrimSpace(task.UserRequest) == "" {
		return errors.New("task objective and userRequest are required")
	}
	if err := validatePaths(task.Paths); err != nil {
		return fmt.Errorf("task paths: %w", err)
	}
	if len(task.Checks) == 0 {
		return errors.New("task requires at least one deterministic check")
	}
	for i, check := range task.Checks {
		if strings.TrimSpace(check) == "" || strings.ContainsRune(check, '\x00') {
			return fmt.Errorf("task check %d is invalid", i)
		}
	}
	return nil
}

func validateLedgerOracle(oracle Oracle, records, trajectories map[string]struct{}) error {
	for _, id := range oracle.RelevantRecordIDs {
		if _, exists := records[id]; !exists {
			return fmt.Errorf("references unknown record %q", id)
		}
	}
	for _, id := range oracle.RelevantTrajectoryIDs {
		if _, exists := trajectories[id]; !exists {
			return fmt.Errorf("references unknown trajectory %q", id)
		}
	}
	if !slices.Contains([]string{"pause", "proceed", "inspect"}, oracle.ExpectedBehavior) {
		return fmt.Errorf("unsupported expectedBehavior %q", oracle.ExpectedBehavior)
	}
	if strings.TrimSpace(oracle.MaterialReason) == "" {
		return errors.New("materialReason is required")
	}
	return nil
}

func (s EventGatedSchedule) Validate(laneIDs map[string]struct{}) error {
	if s.InitialLane == s.DependentLane || s.InitialLane == "" || s.DependentLane == "" {
		return errors.New("schedule requires distinct initialLane and dependentLane")
	}
	if _, exists := laneIDs[s.InitialLane]; !exists {
		return fmt.Errorf("schedule initialLane %q is not a lane", s.InitialLane)
	}
	if _, exists := laneIDs[s.DependentLane]; !exists {
		return fmt.Errorf("schedule dependentLane %q is not a lane", s.DependentLane)
	}
	if s.TimeoutSeconds < 1 || s.TimeoutSeconds > 3600 {
		return errors.New("schedule.timeoutSeconds must be between 1 and 3600")
	}
	if s.PollIntervalMS < 50 || s.PollIntervalMS > 60000 || s.PollIntervalMS > s.TimeoutSeconds*1000 {
		return errors.New("schedule.pollIntervalMs must be between 50 and the timeout")
	}
	if s.Barrier.EventType != "note.recorded" || s.Barrier.Kind != "checkpoint" {
		return errors.New("schedule barrier must be a note.recorded checkpoint")
	}
	if len(s.Barrier.RequiredPathOverlap) == 0 {
		return errors.New("schedule barrier requires at least one path overlap")
	}
	return validatePaths(s.Barrier.RequiredPathOverlap)
}

func (p CollaborationPreparedWorld) Validate() error {
	if p.SchemaVersion != 2 || p.ScenarioID == "" || p.ScenarioHash == "" || p.ProjectSlug == "" || p.ProjectID == "" || p.RepositoryHead == "" || p.SkillHash == "" {
		return errors.New("collaboration prepared world has incomplete provenance")
	}
	if len(p.Lanes) != 2 {
		return errors.New("collaboration prepared world requires two lanes")
	}
	seen := map[string]struct{}{}
	principals := map[string]struct{}{}
	repositories := map[string]struct{}{}
	for _, lane := range p.Lanes {
		if lane.LaneID == "" || lane.ActorKey == "" || lane.PrincipalID == "" || lane.Branch == "" || lane.AgentName == "" || lane.Harness == "" || lane.Provider == "" || lane.Model == "" || lane.Role == "" {
			return errors.New("prepared lane has incomplete principal provenance")
		}
		if lane.RepositoryHead != p.RepositoryHead || lane.SkillHash != p.SkillHash {
			return fmt.Errorf("prepared lane %q does not match shared repository or skill provenance", lane.LaneID)
		}
		if _, exists := seen[lane.LaneID]; exists {
			return fmt.Errorf("duplicate prepared lane %q", lane.LaneID)
		}
		seen[lane.LaneID] = struct{}{}
		if _, exists := principals[lane.PrincipalID]; exists {
			return fmt.Errorf("prepared lanes must use distinct principals; %q is repeated", lane.PrincipalID)
		}
		principals[lane.PrincipalID] = struct{}{}
		if err := validateReportPath(lane.RepositoryPath); err != nil {
			return fmt.Errorf("prepared lane %q repositoryPath: %w", lane.LaneID, err)
		}
		if _, exists := repositories[lane.RepositoryPath]; exists {
			return fmt.Errorf("prepared lanes must use distinct repository paths; %q is repeated", lane.RepositoryPath)
		}
		repositories[lane.RepositoryPath] = struct{}{}
		if err := validateReportPath(lane.ArtifactPath); err != nil {
			return fmt.Errorf("prepared lane %q artifactPath: %w", lane.LaneID, err)
		}
	}
	return nil
}

func (e CollaborationEpisode) Validate() error {
	if e.SchemaVersion != 2 || e.EpisodeID == "" || e.ScenarioID == "" || e.ScenarioHash == "" {
		return errors.New("collaboration episode identity is incomplete")
	}
	if !slices.Contains([]string{"planned", "running", "completed", "incomplete", "failed", "cancelled"}, e.Status) {
		return fmt.Errorf("unsupported collaboration episode status %q", e.Status)
	}
	for name, path := range map[string]string{"schedulePath": e.SchedulePath, "controllerEvents": e.ControllerEvents, "repository.laneAPath": e.Repository.LaneAPath} {
		if err := validateReportPath(path); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if e.Repository.LaneBPath != "" {
		if err := validateReportPath(e.Repository.LaneBPath); err != nil {
			return fmt.Errorf("repository.laneBPath: %w", err)
		}
	}
	for _, lane := range e.Lanes {
		for name, path := range map[string]string{"eventsPath": lane.EventsPath, "stderrPath": lane.StderrPath, "finalResponsePath": lane.FinalResponsePath, "projectExportPath": lane.ProjectExportPath, "gitResultPath": lane.GitResultPath, "checksPath": lane.ChecksPath, "commandPath": lane.CommandPath} {
			if path != "" {
				if err := validateReportPath(path); err != nil {
					return fmt.Errorf("lane %q %s: %w", lane.LaneID, name, err)
				}
			}
		}
	}
	if e.Barrier.SnapshotPath != "" {
		if err := validateReportPath(e.Barrier.SnapshotPath); err != nil {
			return fmt.Errorf("barrier snapshotPath: %w", err)
		}
	}
	if e.Status == "completed" {
		if !e.Barrier.Observed || !e.Score.BarrierObserved || !e.Score.DependentStarted || e.Score.Incomplete || e.Score.LanesCompleted != 2 || len(e.Lanes) != 2 {
			return errors.New("completed collaboration episode does not satisfy deterministic completion gates")
		}
		seen := map[string]bool{}
		for _, lane := range e.Lanes {
			if lane.LaneID == "" || seen[lane.LaneID] || lane.Status != "completed" || lane.ThreadID == "" || lane.ObservedRunID == "" || lane.ChecksPath == "" || lane.CommandPath == "" {
				return errors.New("completed collaboration episode has invalid lane evidence")
			}
			seen[lane.LaneID] = true
		}
	}
	return nil
}

// Validate ensures controller-event JSONL can be retained as observable
// evidence without admitting a credential value or credential location.
func (e ControllerEvent) Validate() error {
	if e.At.IsZero() || strings.TrimSpace(e.Type) == "" || strings.TrimSpace(e.Message) == "" {
		return errors.New("controller event is incomplete")
	}
	lower := strings.ToLower(e.Message)
	for _, marker := range []string{"authorization: bearer", "clankspace_token", "token=", "credentials_file", "/secrets/", "secrets/"} {
		if strings.Contains(lower, marker) {
			return errors.New("controller event may not retain credential-like content")
		}
	}
	return nil
}

func validateReportPath(path string) error {
	if err := validateRelativePath(path); err != nil {
		return err
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	for _, part := range strings.Split(clean, "/") {
		lower := strings.ToLower(part)
		if lower == "secrets" || strings.Contains(lower, "credential") || strings.Contains(lower, "token") {
			return fmt.Errorf("path %q may not reference secrets or credentials", path)
		}
	}
	return nil
}
