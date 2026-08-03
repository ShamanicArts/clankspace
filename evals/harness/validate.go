package harness

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var (
	scenarioIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,79}$`)
	aliasPattern      = regexp.MustCompile(`^[a-z][a-z0-9-]{1,79}$`)
)

func (s Scenario) Validate() error {
	if s.SchemaVersion != 1 {
		return fmt.Errorf("schemaVersion must be 1, got %d", s.SchemaVersion)
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
	if !slices.Contains([]string{"fake", "real-snapshot"}, s.Project.RepositoryProfile) {
		return fmt.Errorf("unsupported repositoryProfile %q", s.Project.RepositoryProfile)
	}
	if s.Project.RepositoryProfile == "fake" && len(s.Repository.Commits) == 0 {
		return errors.New("fake repositories require at least one commit")
	}
	if s.Project.RepositoryProfile == "real-snapshot" && strings.TrimSpace(s.Repository.SnapshotID) == "" {
		return errors.New("real-snapshot repositories require repository.snapshotId")
	}
	actors := make(map[string]struct{}, len(s.Actors))
	for i, actor := range s.Actors {
		if !aliasPattern.MatchString(actor.Key) {
			return fmt.Errorf("actors[%d].key is invalid", i)
		}
		if _, exists := actors[actor.Key]; exists {
			return fmt.Errorf("duplicate actor key %q", actor.Key)
		}
		actors[actor.Key] = struct{}{}
		if strings.TrimSpace(actor.PrincipalName) == "" || strings.TrimSpace(actor.AgentName) == "" {
			return fmt.Errorf("actor %q requires principalName and agentName", actor.Key)
		}
		if !slices.Contains([]string{"primary", "subagent", "reviewer", "automation", "integration"}, actor.Role) {
			return fmt.Errorf("actor %q has unsupported role %q", actor.Key, actor.Role)
		}
	}
	if len(actors) == 0 {
		return errors.New("at least one actor is required")
	}
	if len(s.Conversation) == 0 {
		return errors.New("at least one prior user turn is required")
	}
	for i, turn := range s.Conversation {
		if turn.Role != "user" || strings.TrimSpace(turn.Text) == "" {
			return fmt.Errorf("conversation[%d] must be a non-empty user turn; assistant responses are generated during rollout", i)
		}
	}
	records := map[string]struct{}{}
	for i, record := range s.Records {
		if !aliasPattern.MatchString(record.ID) {
			return fmt.Errorf("records[%d].id is invalid", i)
		}
		if _, exists := records[record.ID]; exists {
			return fmt.Errorf("duplicate record id %q", record.ID)
		}
		records[record.ID] = struct{}{}
		if _, exists := actors[record.ActorKey]; !exists {
			return fmt.Errorf("record %q references unknown actor %q", record.ID, record.ActorKey)
		}
		if !slices.Contains([]string{"current", "superseded"}, record.Status) {
			return fmt.Errorf("record %q status %q cannot yet be seeded through the public API", record.ID, record.Status)
		}
		if err := validatePaths(record.Paths); err != nil {
			return fmt.Errorf("record %q: %w", record.ID, err)
		}
	}
	trajectories := map[string]struct{}{}
	for i, trajectory := range s.Trajectories {
		if !aliasPattern.MatchString(trajectory.ID) {
			return fmt.Errorf("trajectories[%d].id is invalid", i)
		}
		if _, exists := trajectories[trajectory.ID]; exists {
			return fmt.Errorf("duplicate trajectory id %q", trajectory.ID)
		}
		trajectories[trajectory.ID] = struct{}{}
		if _, exists := actors[trajectory.ActorKey]; !exists {
			return fmt.Errorf("trajectory %q references unknown actor %q", trajectory.ID, trajectory.ActorKey)
		}
		if trajectory.Status != "active" {
			return fmt.Errorf("trajectory %q status %q cannot yet be seeded through the public API", trajectory.ID, trajectory.Status)
		}
		if err := validatePaths(trajectory.Paths); err != nil {
			return fmt.Errorf("trajectory %q: %w", trajectory.ID, err)
		}
	}
	for _, id := range s.Oracle.RelevantRecordIDs {
		if _, exists := records[id]; !exists {
			return fmt.Errorf("oracle references unknown record %q", id)
		}
	}
	for _, id := range s.Oracle.RelevantTrajectoryIDs {
		if _, exists := trajectories[id]; !exists {
			return fmt.Errorf("oracle references unknown trajectory %q", id)
		}
	}
	if !slices.Contains([]string{"pause", "proceed", "inspect"}, s.Oracle.ExpectedBehavior) {
		return fmt.Errorf("unsupported expectedBehavior %q", s.Oracle.ExpectedBehavior)
	}
	if strings.TrimSpace(s.Task.Objective) == "" || strings.TrimSpace(s.Task.UserRequest) == "" {
		return errors.New("task objective and userRequest are required")
	}
	if _, exists := actors[s.Task.ActorKey]; !exists {
		return fmt.Errorf("task references unknown actor %q", s.Task.ActorKey)
	}
	if err := validatePaths(s.Project.Paths); err != nil {
		return fmt.Errorf("project paths: %w", err)
	}
	if err := validatePaths(s.Task.Paths); err != nil {
		return fmt.Errorf("task paths: %w", err)
	}
	if err := s.validateRepository(); err != nil {
		return err
	}
	if s.Generation.CurriculumVersion == "" || s.Generation.Seed == "" || s.Generation.GeneratorProvider == "" || s.Generation.GeneratorModel == "" {
		return errors.New("generation provenance is incomplete")
	}
	return nil
}

func (s Scenario) validateRepository() error {
	commitIDs := map[string]struct{}{}
	for i, commit := range s.Repository.Commits {
		if !aliasPattern.MatchString(commit.ID) {
			return fmt.Errorf("repository.commits[%d].id is invalid", i)
		}
		if _, exists := commitIDs[commit.ID]; exists {
			return fmt.Errorf("duplicate commit id %q", commit.ID)
		}
		commitIDs[commit.ID] = struct{}{}
		if strings.TrimSpace(commit.Message) == "" || strings.TrimSpace(commit.AuthorName) == "" || strings.TrimSpace(commit.AuthorEmail) == "" {
			return fmt.Errorf("commit %q requires message and author", commit.ID)
		}
		if len(commit.Changes) == 0 {
			return fmt.Errorf("commit %q has no changes", commit.ID)
		}
		for _, change := range commit.Changes {
			if err := validateFixturePath(change.Path); err != nil {
				return fmt.Errorf("commit %q: %w", commit.ID, err)
			}
			lower := strings.ToLower(change.Content)
			if strings.Contains(lower, "clankspace_token") || strings.Contains(lower, "authorization: bearer") || strings.Contains(lower, "clank_") {
				return fmt.Errorf("commit %q path %q appears to contain a credential", commit.ID, change.Path)
			}
		}
	}
	return nil
}

func validatePaths(paths []string) error {
	for _, path := range paths {
		if err := validateRelativePath(path); err != nil {
			return err
		}
	}
	return nil
}

func validateFixturePath(path string) error {
	if err := validateRelativePath(path); err != nil {
		return err
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == ".clankspace.json" || clean == "AGENTS.md" || clean == ".agents" || strings.HasPrefix(clean, ".agents/") || clean == ".git" || strings.HasPrefix(clean, ".git/") {
		return fmt.Errorf("path %q is reserved for the evaluation harness", path)
	}
	return nil
}

func validateRelativePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) {
		return fmt.Errorf("path %q must be relative", path)
	}
	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %q escapes the repository", path)
	}
	return nil
}
