package harness

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ShamanicArts/clankspace/internal/client"
	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/localconfig"
)

type SeedOptions struct {
	URL            string
	AdminToken     string
	ControlProject string
	Layout         Layout
	ScenarioHash   string
	CorpusVersion  string
}

type seedCredential struct {
	PrincipalID string `json:"principalId"`
	Token       string `json:"token"`
}

type seedSecrets struct {
	Version int                       `json:"version"`
	Actors  map[string]seedCredential `json:"actors"`
}

func LoadAdminEnvironment(path string) (string, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	values := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if found {
			values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	if err = scanner.Err(); err != nil {
		return "", "", err
	}
	url, token := values["CLANKSPACE_URL"], values["CLANKSPACE_TOKEN"]
	if url == "" || token == "" {
		return "", "", errors.New("admin environment requires CLANKSPACE_URL and CLANKSPACE_TOKEN")
	}
	return url, token, nil
}

func SeedScenario(ctx context.Context, scenario Scenario, options SeedOptions) (PreparedWorld, error) {
	if options.URL == "" || options.AdminToken == "" || options.ScenarioHash == "" || options.CorpusVersion == "" {
		return PreparedWorld{}, errors.New("seed URL, admin token, scenario hash, and corpus version are required")
	}
	admin := client.New(options.URL, options.AdminToken)
	seedKey := options.ScenarioHash + ":" + options.CorpusVersion
	projectSlug := ProjectSlugForScenario(scenario.ID, options.CorpusVersion, options.ScenarioHash)
	project, err := ensureProject(ctx, admin, projectSlug, scenario.Project.Name, "Isolated evaluation world for "+scenario.ID, key(seedKey, "project"))
	if err != nil {
		return PreparedWorld{}, err
	}
	secrets := seedSecrets{Version: 1, Actors: map[string]seedCredential{}}
	runIDs := map[string]string{}
	actorClients := map[string]*client.Client{}
	for _, actor := range scenario.Actors {
		var credential domain.ProjectCredential
		err = admin.DoWithKey(ctx, http.MethodPost, "/projects/"+project.ID+"/tokens", key(seedKey, "token", actor.Key), map[string]string{"displayName": actor.PrincipalName}, &credential)
		if err != nil {
			return PreparedWorld{}, fmt.Errorf("issue token for %s: %w", actor.Key, err)
		}
		secrets.Actors[actor.Key] = seedCredential{PrincipalID: credential.Principal.ID, Token: credential.Token}
		actorClient := client.New(options.URL, credential.Token)
		actorClients[actor.Key] = actorClient
		var out struct {
			Run domain.Run `json:"run"`
		}
		in := domain.StartRunInput{
			ProjectID: project.ID, AgentName: actor.AgentName, Harness: actor.Harness,
			Provider: actor.Provider, Model: actor.Model, Reasoning: actor.Reasoning,
			Role: actor.Role, RunType: "automation", Objective: "Synthetic prior project activity",
			Branch: "main", Worktree: scenario.ID, InstructionProfile: []string{"clankspace-eval-v1"},
		}
		if err = actorClient.DoWithKey(ctx, http.MethodPost, "/runs", key(seedKey, "run", actor.Key), in, &out); err != nil {
			return PreparedWorld{}, fmt.Errorf("start run for %s: %w", actor.Key, err)
		}
		runIDs[actor.Key] = out.Run.ID
	}
	recordIDs := map[string]string{}
	for _, record := range scenario.Records {
		actorClient := actorClients[record.ActorKey]
		var out struct {
			Note domain.Note `json:"note"`
		}
		in := domain.CreateNoteInput{
			RunID: runIDs[record.ActorKey], Kind: record.Kind, Title: record.Title,
			Summary: record.Summary, Rationale: record.Rationale, LedBy: record.LedBy,
			DirectionBasis: record.DirectionBasis, Paths: record.Paths,
			SourceRef: fmt.Sprintf("eval:%s:%s", scenario.ID, record.ID),
		}
		if err = actorClient.DoWithKey(ctx, http.MethodPost, "/projects/"+project.ID+"/notes", key(seedKey, "note", record.ID), in, &out); err != nil {
			return PreparedWorld{}, fmt.Errorf("create note %s: %w", record.ID, err)
		}
		recordIDs[record.ID] = out.Note.ID
		if record.Status == "superseded" {
			supersede := domain.SupersedeNoteInput{RunID: runIDs[record.ActorKey], ExpectedRevision: out.Note.Revision, Reason: "Synthetic lifecycle state from the fixed scenario blueprint."}
			var superseded struct {
				Note domain.Note `json:"note"`
			}
			if err = actorClient.DoWithKey(ctx, http.MethodPost, "/projects/"+project.ID+"/notes/"+out.Note.ID+"/supersede", key(seedKey, "supersede", record.ID), supersede, &superseded); err != nil {
				return PreparedWorld{}, fmt.Errorf("supersede note %s: %w", record.ID, err)
			}
		}
	}
	trajectoryIDs := map[string]string{}
	for _, trajectory := range scenario.Trajectories {
		actorClient := actorClients[trajectory.ActorKey]
		var out struct {
			Trajectory domain.Trajectory `json:"trajectory"`
		}
		in := domain.CreateTrajectoryInput{
			RunID: runIDs[trajectory.ActorKey], Objective: trajectory.Objective,
			Rationale: trajectory.Rationale, Paths: trajectory.Paths, Branch: trajectory.Branch,
		}
		if err = actorClient.DoWithKey(ctx, http.MethodPost, "/projects/"+project.ID+"/trajectories", key(seedKey, "trajectory", trajectory.ID), in, &out); err != nil {
			return PreparedWorld{}, fmt.Errorf("create trajectory %s: %w", trajectory.ID, err)
		}
		trajectoryIDs[trajectory.ID] = out.Trajectory.ID
	}
	actorsWithActiveTrajectories := map[string]bool{}
	for _, trajectory := range scenario.Trajectories {
		if trajectory.Status == "active" {
			actorsWithActiveTrajectories[trajectory.ActorKey] = true
		}
	}
	for actorKey, actorClient := range actorClients {
		if actorsWithActiveTrajectories[actorKey] {
			continue
		}
		var out struct {
			Run domain.Run `json:"run"`
		}
		_ = actorClient.DoWithKey(ctx, http.MethodPost, "/runs/"+runIDs[actorKey]+"/end", key(seedKey, "end", actorKey), domain.EndRunInput{Outcome: "completed", Verification: "Seeded from a validated immutable scenario."}, &out)
	}
	secretsPath := filepath.Join(options.Layout.SecretsPath, "actor-tokens.json")
	if err = writeJSONExclusive(secretsPath, secrets, 0600); err != nil {
		return PreparedWorld{}, err
	}
	credentialPath := filepath.Join(options.Layout.SecretsPath, "credentials.json")
	testCredential, exists := secrets.Actors[scenario.Task.ActorKey]
	if !exists {
		return PreparedWorld{}, fmt.Errorf("missing test actor credential for %q", scenario.Task.ActorKey)
	}
	if err = localconfig.StoreCredential(credentialPath, options.URL, projectSlug, testCredential.Token); err != nil {
		return PreparedWorld{}, err
	}
	return PreparedWorld{
		SchemaVersion: 1, ScenarioID: scenario.ID, ScenarioHash: options.ScenarioHash,
		CorpusVersion: options.CorpusVersion, Split: scenario.Split, ProjectSlug: projectSlug,
		ProjectID: project.ID, RecordIDs: recordIDs, TrajectoryIDs: trajectoryIDs, RunIDs: runIDs,
		CredentialFile: credentialPath,
	}, nil
}

func TrackPrepared(ctx context.Context, prepared PreparedWorld, options SeedOptions) error {
	if options.ControlProject == "" {
		return nil
	}
	admin := client.New(options.URL, options.AdminToken)
	seedKey := prepared.ScenarioHash + ":" + prepared.CorpusVersion
	var runOut struct {
		Run domain.Run `json:"run"`
	}
	runIn := domain.StartRunInput{ProjectID: options.ControlProject, AgentName: "clank-eval", Harness: "clank-eval", Provider: "deterministic", Model: "none", Role: "automation", RunType: "automation", Objective: "Prepare immutable evaluation worlds"}
	if err := admin.DoWithKey(ctx, http.MethodPost, "/runs", key(seedKey, "control-run"), runIn, &runOut); err != nil {
		return err
	}
	note := domain.CreateNoteInput{
		RunID: runOut.Run.ID, Kind: "checkpoint", Title: "Prepared evaluation world " + prepared.ScenarioID,
		Summary:   fmt.Sprintf("Prepared corpus %s/%s scenario %s as isolated project %s at repository head %s.", prepared.CorpusVersion, prepared.Split, prepared.ScenarioID, prepared.ProjectSlug, prepared.RepositoryHead),
		Rationale: "The world passed deterministic schema, repository, identity, and seeding checks. Hidden oracle and raw traces remain in the external evaluation ledger.",
		LedBy:     "agent", DirectionBasis: "autonomous_agent_judgment", Verification: "deterministic harness", SourceRef: "eval-ledger:" + prepared.ScenarioHash,
	}
	var noteOut struct {
		Note domain.Note `json:"note"`
	}
	if err := admin.DoWithKey(ctx, http.MethodPost, "/projects/"+options.ControlProject+"/notes", key(seedKey, "control-note"), note, &noteOut); err != nil {
		return err
	}
	var endOut struct {
		Run domain.Run `json:"run"`
	}
	return admin.DoWithKey(ctx, http.MethodPost, "/runs/"+runOut.Run.ID+"/end", key(seedKey, "control-end"), domain.EndRunInput{Outcome: "completed", Verification: "prepared artifact recorded"}, &endOut)
}

func ensureProject(ctx context.Context, c *client.Client, slug, name, description, idempotencyKey string) (domain.Project, error) {
	projects, err := c.ListProjects(ctx)
	if err != nil {
		return domain.Project{}, err
	}
	for _, project := range projects {
		if project.Slug == slug {
			return project, nil
		}
	}
	var out struct {
		Project domain.Project `json:"project"`
	}
	err = c.DoWithKey(ctx, http.MethodPost, "/projects", idempotencyKey, map[string]string{"slug": slug, "name": name, "description": description}, &out)
	return out.Project, err
}

func ProjectSlugForScenario(id, corpusVersion, hash string) string {
	suffix := hash[:8]
	prefix := "eval-" + corpusVersion + "-"
	maximumID := 63 - len(prefix) - 1 - len(suffix)
	if len(id) > maximumID {
		id = strings.TrimRight(id[:maximumID], "-")
	}
	return prefix + id + "-" + suffix
}

func key(parts ...string) string {
	joined := strings.Join(parts, ":")
	if len(joined) > 180 {
		return joined[:180]
	}
	return joined
}

func ReadPrepared(path string) (PreparedWorld, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return PreparedWorld{}, err
	}
	var prepared PreparedWorld
	err = json.Unmarshal(body, &prepared)
	return prepared, err
}
