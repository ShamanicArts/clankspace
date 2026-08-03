package harness

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/client"
	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/localconfig"
)

// VerifyCollaborationSourceEvidence binds a v2 blueprint to the manifest made
// by CreateSanitizedSnapshot. Format validation alone is deliberately not
// enough: the source commit, sanitized head, bundle, license, and source URL
// must all agree with the local, immutable snapshot evidence.
func VerifyCollaborationSourceEvidence(s CollaborationScenario, snapshot string) error {
	if s.SourceEvidence.License != "MIT" {
		return errors.New("source evidence license must be MIT")
	}
	snapshot, err := filepath.Abs(snapshot)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(filepath.Dir(snapshot), s.SourceEvidence.SnapshotID+".snapshot.json")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read source snapshot manifest: %w", err)
	}
	var manifest SnapshotResult
	if err = json.Unmarshal(body, &manifest); err != nil {
		return fmt.Errorf("parse source snapshot manifest: %w", err)
	}
	manifestRepository, err := filepath.Abs(manifest.Repository)
	if err != nil || filepath.Clean(manifestRepository) != filepath.Clean(snapshot) {
		return errors.New("source snapshot manifest does not identify the configured snapshot repository")
	}
	if manifest.ID != s.SourceEvidence.SnapshotID || manifest.SourceHead != s.SourceEvidence.SourceCommit || manifest.SnapshotHead != s.SourceEvidence.SnapshotHead {
		return errors.New("source evidence does not match snapshot manifest")
	}
	if manifest.SourceURL == "" || strings.TrimRight(manifest.SourceURL, "/") != strings.TrimRight(s.SourceEvidence.RepositoryURL, "/") {
		return errors.New("source evidence repository URL does not match snapshot manifest")
	}
	head, err := gitOutput(snapshot, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(head) != s.SourceEvidence.SnapshotHead {
		return errors.New("source evidence snapshot head does not match configured snapshot repository")
	}
	bundleHash, err := FileHash(manifest.Bundle)
	if err != nil || bundleHash != s.SourceEvidence.BundleHash {
		return errors.New("source evidence bundle hash does not match snapshot manifest bundle")
	}
	licensePath := filepath.Join(snapshot, filepath.FromSlash(s.SourceEvidence.LicenseFile))
	license, err := os.ReadFile(licensePath)
	if err != nil {
		return fmt.Errorf("read source license: %w", err)
	}
	if ContentHash(license) != s.SourceEvidence.LicenseFileHash || !strings.Contains(strings.ToLower(string(license)), "mit license") {
		return errors.New("source evidence license is not the declared MIT license")
	}
	if s.Repository.BaseRef != s.SourceEvidence.SnapshotHead {
		return errors.New("repository baseRef must equal the verified sanitized snapshot head")
	}
	return nil
}

// SeedCollaborationScenario creates one isolated project and one attributable
// project credential per lane. Credential files live only under Layout.SecretsPath;
// neither the prepared artifact nor an agent-visible lane artifact can reveal
// them or the ledger oracle.
func SeedCollaborationScenario(ctx context.Context, scenario CollaborationScenario, options SeedOptions) (CollaborationPreparedWorld, error) {
	if options.URL == "" || options.AdminToken == "" || options.ScenarioHash == "" || options.CorpusVersion == "" {
		return CollaborationPreparedWorld{}, errors.New("seed URL, admin token, scenario hash, and corpus version are required")
	}
	admin := client.New(options.URL, options.AdminToken)
	seedKey := options.ScenarioHash + ":" + options.CorpusVersion
	projectSlug := ProjectSlugForScenario(scenario.ID, options.CorpusVersion, options.ScenarioHash)
	project, err := ensureProject(ctx, admin, projectSlug, scenario.Project.Name, "Isolated collaboration evaluation world for "+scenario.ID, key(seedKey, "project"))
	if err != nil {
		return CollaborationPreparedWorld{}, err
	}
	actorByKey := make(map[string]Actor, len(scenario.Actors))
	actorClients := make(map[string]*client.Client, len(scenario.Actors))
	runIDs := make(map[string]string, len(scenario.Actors))
	credentials := make(map[string]seedCredential, len(scenario.Actors))
	for _, actor := range scenario.Actors {
		actorByKey[actor.Key] = actor
		var issued domain.ProjectCredential
		if err := admin.DoWithKey(ctx, http.MethodPost, "/projects/"+project.ID+"/tokens", key(seedKey, "token", actor.Key), map[string]string{"displayName": actor.PrincipalName}, &issued); err != nil {
			return CollaborationPreparedWorld{}, fmt.Errorf("issue token for %s: %w", actor.Key, err)
		}
		credentials[actor.Key] = seedCredential{PrincipalID: issued.Principal.ID, Token: issued.Token}
		actorClients[actor.Key] = client.New(options.URL, issued.Token)
		var out struct {
			Run domain.Run `json:"run"`
		}
		in := domain.StartRunInput{ProjectID: project.ID, AgentName: actor.AgentName, Harness: actor.Harness, Provider: actor.Provider, Model: actor.Model, Reasoning: actor.Reasoning, Role: actor.Role, RunType: "automation", Objective: "Synthetic prior collaboration activity", Branch: "main", Worktree: scenario.ID, InstructionProfile: []string{"clankspace-eval-v2"}}
		if err := actorClients[actor.Key].DoWithKey(ctx, http.MethodPost, "/runs", key(seedKey, "run", actor.Key), in, &out); err != nil {
			return CollaborationPreparedWorld{}, fmt.Errorf("start seed run for %s: %w", actor.Key, err)
		}
		runIDs[actor.Key] = out.Run.ID
	}
	for _, record := range scenario.Records {
		in := domain.CreateNoteInput{RunID: runIDs[record.ActorKey], Kind: record.Kind, Title: record.Title, Summary: record.Summary, Rationale: record.Rationale, LedBy: record.LedBy, DirectionBasis: record.DirectionBasis, Paths: record.Paths, SourceRef: fmt.Sprintf("eval:%s:%s", scenario.ID, record.ID)}
		var out struct {
			Note domain.Note `json:"note"`
		}
		if err := actorClients[record.ActorKey].DoWithKey(ctx, http.MethodPost, "/projects/"+project.ID+"/notes", key(seedKey, "note", record.ID), in, &out); err != nil {
			return CollaborationPreparedWorld{}, fmt.Errorf("create seeded note %s: %w", record.ID, err)
		}
		if record.Status == "superseded" {
			var superseded struct {
				Note domain.Note `json:"note"`
			}
			in := domain.SupersedeNoteInput{RunID: runIDs[record.ActorKey], ExpectedRevision: out.Note.Revision, Reason: "Synthetic lifecycle state from the fixed collaboration scenario."}
			if err := actorClients[record.ActorKey].DoWithKey(ctx, http.MethodPost, "/projects/"+project.ID+"/notes/"+out.Note.ID+"/supersede", key(seedKey, "supersede", record.ID), in, &superseded); err != nil {
				return CollaborationPreparedWorld{}, fmt.Errorf("supersede seeded note %s: %w", record.ID, err)
			}
		}
	}
	for _, trajectory := range scenario.Trajectories {
		in := domain.CreateTrajectoryInput{RunID: runIDs[trajectory.ActorKey], Objective: trajectory.Objective, Rationale: trajectory.Rationale, Paths: trajectory.Paths, Branch: trajectory.Branch}
		var out struct {
			Trajectory domain.Trajectory `json:"trajectory"`
		}
		if err := actorClients[trajectory.ActorKey].DoWithKey(ctx, http.MethodPost, "/projects/"+project.ID+"/trajectories", key(seedKey, "trajectory", trajectory.ID), in, &out); err != nil {
			return CollaborationPreparedWorld{}, fmt.Errorf("create seeded trajectory %s: %w", trajectory.ID, err)
		}
	}
	prepared := CollaborationPreparedWorld{SchemaVersion: 2, ScenarioID: scenario.ID, ScenarioHash: options.ScenarioHash, CorpusVersion: options.CorpusVersion, Split: scenario.Split, ProjectSlug: projectSlug, ProjectID: project.ID, Lanes: make([]PreparedLane, 0, len(scenario.Lanes)), CreatedAt: time.Now().UTC()}
	for _, lane := range scenario.Lanes {
		actor := actorByKey[lane.ActorKey]
		credential := credentials[lane.ActorKey]
		credentialPath := filepath.Join(options.Layout.SecretsPath, lane.ID+".json")
		if err := storeLaneCredential(credentialPath, options.URL, projectSlug, credential.Token); err != nil {
			return CollaborationPreparedWorld{}, fmt.Errorf("store credential for %s: %w", lane.ID, err)
		}
		visible, err := scenario.AgentVisibleLane(lane.ID)
		if err != nil {
			return CollaborationPreparedWorld{}, err
		}
		artifactPath := filepath.Join(options.Layout.Root, "lanes", lane.ID+".json")
		if err = writeLaneArtifact(artifactPath, visible); err != nil {
			return CollaborationPreparedWorld{}, err
		}
		prepared.Lanes = append(prepared.Lanes, PreparedLane{LaneID: lane.ID, ActorKey: lane.ActorKey, PrincipalID: credential.PrincipalID, Branch: lane.Branch, AgentName: actor.AgentName, Harness: actor.Harness, Provider: actor.Provider, Model: actor.Model, Reasoning: actor.Reasoning, Role: actor.Role, RepositoryPath: filepath.ToSlash(filepath.Join("repositories", lane.ID)), ArtifactPath: filepath.ToSlash(filepath.Join("lanes", lane.ID+".json"))})
	}
	return prepared, nil
}

func writeLaneArtifact(path string, visible AgentVisibleLane) error {
	if existing, err := os.ReadFile(path); err == nil {
		body, marshalErr := json.MarshalIndent(visible, "", "  ")
		if marshalErr != nil || string(existing) != string(append(body, '\n')) {
			return fmt.Errorf("lane artifact already exists with different content: %s", path)
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeJSONExclusive(path, visible, 0600)
}

func storeLaneCredential(path, url, project, token string) error {
	if existing, err := readCollaborationCredential(path, project); err == nil {
		if existing.url == strings.TrimRight(url, "/") && existing.token == token {
			return nil
		}
		return errors.New("existing lane credential does not match idempotent issued credential")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return localconfig.StoreCredential(path, url, project, token)
}
