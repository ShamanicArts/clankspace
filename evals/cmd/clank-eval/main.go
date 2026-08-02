package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/evals/harness"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "clank-eval:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: clank-eval validate|validate-collaboration|snapshot|ingest-worlds|prepare|prepare-collaboration|rollout|collaboration-rollout")
	}
	switch args[0] {
	case "validate":
		f := flag.NewFlagSet("validate", flag.ContinueOnError)
		scenarioPath := f.String("scenario", "", "rendered scenario JSON")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		scenario, canonical, err := harness.LoadScenario(*scenarioPath)
		if err != nil {
			return err
		}
		return printJSON(map[string]any{"id": scenario.ID, "split": scenario.Split, "hash": harness.ContentHash(canonical), "valid": true})
	case "prepare":
		return prepare(ctx, args[1:])
	case "validate-collaboration":
		return validateCollaboration(args[1:])
	case "prepare-collaboration":
		return prepareCollaboration(ctx, args[1:])
	case "rollout":
		return rollout(ctx, args[1:])
	case "collaboration-rollout":
		return collaborationRollout(ctx, args[1:])
	case "snapshot":
		return snapshot(args[1:])
	case "ingest-worlds":
		return ingestWorlds(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func validateCollaboration(args []string) error {
	f := flag.NewFlagSet("validate-collaboration", flag.ContinueOnError)
	scenarioPath := f.String("scenario", "", "v2 collaboration scenario JSON")
	if err := f.Parse(args); err != nil {
		return err
	}
	if *scenarioPath == "" {
		return errors.New("--scenario is required")
	}
	scenario, canonical, err := harness.LoadCollaborationScenario(*scenarioPath)
	if err != nil {
		return err
	}
	return printJSON(map[string]any{"id": scenario.ID, "split": scenario.Split, "hash": harness.ContentHash(canonical), "valid": true})
}

func collaborationRollout(ctx context.Context, args []string) error {
	f := flag.NewFlagSet("collaboration-rollout", flag.ContinueOnError)
	prepared := f.String("prepared", "", "prepared collaboration world JSON")
	repository := f.String("repository", "", "clean baseline repository to clone per lane")
	credentials := f.String("credentials-dir", "", "directory containing <lane-id>.json credential files")
	codexBin := f.String("codex", "/home/exedev/clankspace-evals/bin/codex-eval", "isolated Codex launcher path")
	episode := f.String("episode", "", "explicit immutable episode ID")
	serverConfig := f.String("server-config", "", "non-secret frozen ClankSpace server configuration file for replay provenance")
	serverCommit := f.String("server-commit", "", "ClankSpace server commit for replay provenance")
	dryRun := f.Bool("dry-run", false, "print exact two-lane processes and paths without launching Codex")
	if err := f.Parse(args); err != nil {
		return err
	}
	options := harness.CollaborationRolloutOptions{PreparedPath: *prepared, RepositoryPath: *repository, CredentialsDir: *credentials, CodexBin: *codexBin, EpisodeID: *episode, DryRun: *dryRun, ServerConfig: *serverConfig, ServerCommit: *serverCommit}
	if *dryRun {
		plan, err := harness.PlanCollaborationRollout(options)
		if err != nil {
			return err
		}
		return printJSON(plan)
	}
	result, err := harness.RunCollaborationRollout(ctx, options)
	if result.EpisodeID != "" {
		if printErr := printJSON(result); printErr != nil {
			return printErr
		}
	}
	return err
}

func ingestWorlds(args []string) error {
	f := flag.NewFlagSet("ingest-worlds", flag.ContinueOnError)
	input := f.String("input", "", "completed OmegaCode world workflow JSON output")
	ledger := f.String("ledger", "", "evaluation ledger root")
	if err := f.Parse(args); err != nil {
		return err
	}
	result, err := harness.IngestWorldWorkflow(*input, *ledger)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func snapshot(args []string) error {
	f := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	id := f.String("id", "", "stable snapshot slug")
	source := f.String("source", "", "source Git repository")
	ref := f.String("ref", "HEAD", "pinned source ref")
	destination := f.String("destination", "", "snapshot output root")
	var includes stringFlags
	f.Var(&includes, "include", "tracked path to include; repeatable; defaults to the full tree")
	if err := f.Parse(args); err != nil {
		return err
	}
	result, err := harness.CreateSanitizedSnapshot(*id, *source, *ref, *destination, includes)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func rollout(ctx context.Context, args []string) error {
	f := flag.NewFlagSet("rollout", flag.ContinueOnError)
	preparedPath := f.String("prepared", "", "prepared world JSON")
	model := f.String("model", "gpt-5.6-luna", "exact Codex model")
	reasoning := f.String("reasoning", "high", "exact Codex reasoning tier")
	codexBin := f.String("codex", "codex", "Codex CLI path")
	dryRun := f.Bool("dry-run", false, "print the genuine multi-turn rollout plan without invoking Codex")
	if err := f.Parse(args); err != nil {
		return err
	}
	options := harness.RolloutOptions{PreparedPath: *preparedPath, Model: *model, Reasoning: *reasoning, CodexBin: *codexBin, DryRun: *dryRun}
	if *dryRun {
		plan, _, _, err := harness.PlanRollout(options)
		if err != nil {
			return err
		}
		return printJSON(plan)
	}
	result, err := harness.RunRollout(ctx, options)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func prepare(ctx context.Context, args []string) error {
	f := flag.NewFlagSet("prepare", flag.ContinueOnError)
	scenarioPath := f.String("scenario", "", "rendered scenario JSON")
	ledgerRoot := f.String("ledger", "", "evaluation ledger root")
	corpusVersion := f.String("corpus", "v1", "immutable corpus version")
	adminEnvironment := f.String("admin-env", "", "0600 file containing CLANKSPACE_URL and CLANKSPACE_TOKEN")
	controlProject := f.String("control-project", "synthetic-lab", "control project slug; empty disables tracking")
	skillPath := f.String("skill", ".agents/skills/clankspace/SKILL.md", "ClankSpace skill injected into the world")
	var snapshots snapshotFlags
	f.Var(&snapshots, "snapshot", "real snapshot mapping id=/absolute/repository/path; repeatable")
	if err := f.Parse(args); err != nil {
		return err
	}
	if *scenarioPath == "" || *ledgerRoot == "" || *adminEnvironment == "" {
		return errors.New("--scenario, --ledger, and --admin-env are required")
	}
	scenario, canonical, err := harness.LoadScenario(*scenarioPath)
	if err != nil {
		return err
	}
	layout, scenarioHash, err := harness.CreateLayout(*ledgerRoot, *corpusVersion, scenario, canonical)
	if err != nil {
		return err
	}
	if prepared, readErr := harness.ReadPrepared(layout.PreparedPath); readErr == nil {
		return printJSON(prepared)
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	url, token, err := harness.LoadAdminEnvironment(*adminEnvironment)
	if err != nil {
		return err
	}
	projectSlug := harness.ProjectSlugForScenario(scenario.ID, *corpusVersion, scenarioHash)
	head, skillHash, err := harness.BuildRepository(scenario, harness.RepositoryOptions{
		Destination: layout.RepositoryPath, ProjectURL: url, ProjectSlug: projectSlug,
		SkillPath: *skillPath, SnapshotSources: snapshots,
	})
	if err != nil {
		return err
	}
	options := harness.SeedOptions{
		URL: url, AdminToken: token, ControlProject: *controlProject, Layout: layout,
		ScenarioHash: scenarioHash, CorpusVersion: *corpusVersion,
	}
	prepared, err := harness.SeedScenario(ctx, scenario, options)
	if err != nil {
		return err
	}
	prepared.SkillHash = skillHash
	prepared.RepositoryPath = layout.RepositoryPath
	prepared.RepositoryHead = head
	prepared.CreatedAt = time.Now().UTC()
	if err = harness.WritePrepared(layout, prepared); err != nil {
		return err
	}
	if err = harness.TrackPrepared(ctx, prepared, options); err != nil {
		return err
	}
	return printJSON(prepared)
}

func prepareCollaboration(ctx context.Context, args []string) error {
	f := flag.NewFlagSet("prepare-collaboration", flag.ContinueOnError)
	scenarioPath := f.String("scenario", "", "v2 collaboration scenario JSON")
	ledgerRoot := f.String("ledger", "", "evaluation ledger root")
	corpusVersion := f.String("corpus", "v2", "immutable corpus version")
	adminEnvironment := f.String("admin-env", "", "0600 file containing CLANKSPACE_URL and CLANKSPACE_TOKEN")
	skillPath := f.String("skill", ".agents/skills/clankspace/SKILL.md", "ClankSpace skill injected into the world")
	var snapshots snapshotFlags
	f.Var(&snapshots, "snapshot", "sanitized snapshot mapping id=/absolute/repository/path; repeatable")
	if err := f.Parse(args); err != nil {
		return err
	}
	if *scenarioPath == "" || *ledgerRoot == "" || *adminEnvironment == "" {
		return errors.New("--scenario, --ledger, and --admin-env are required")
	}
	scenario, canonical, err := harness.LoadCollaborationScenario(*scenarioPath)
	if err != nil {
		return err
	}
	layout, scenarioHash, err := harness.CreateCollaborationLayout(*ledgerRoot, *corpusVersion, scenario, canonical)
	if err != nil {
		return err
	}
	if prepared, readErr := harness.ReadCollaborationPrepared(layout.PreparedPath); readErr == nil {
		return printJSON(prepared)
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	snapshotSource := snapshots[scenario.Repository.SnapshotID]
	if snapshotSource == "" {
		return fmt.Errorf("no local source configured for snapshot %q", scenario.Repository.SnapshotID)
	}
	if err = harness.VerifyCollaborationSourceEvidence(scenario, snapshotSource); err != nil {
		return err
	}
	url, token, err := harness.LoadAdminEnvironment(*adminEnvironment)
	if err != nil {
		return err
	}
	projectSlug := harness.ProjectSlugForScenario(scenario.ID, *corpusVersion, scenarioHash)
	buildScenario := harness.Scenario{Project: scenario.Project, Repository: scenario.Repository}
	head, skillHash, err := harness.BuildRepository(buildScenario, harness.RepositoryOptions{Destination: layout.RepositoryPath, ProjectURL: url, ProjectSlug: projectSlug, SkillPath: *skillPath, SnapshotSources: snapshots})
	if err != nil {
		return err
	}
	options := harness.SeedOptions{URL: url, AdminToken: token, Layout: layout, ScenarioHash: scenarioHash, CorpusVersion: *corpusVersion}
	prepared, err := harness.SeedCollaborationScenario(ctx, scenario, options)
	if err != nil {
		return err
	}
	prepared.RepositoryHead, prepared.SkillHash = head, skillHash
	for i := range prepared.Lanes {
		prepared.Lanes[i].RepositoryHead, prepared.Lanes[i].SkillHash = head, skillHash
	}
	if err = prepared.Validate(); err != nil {
		return err
	}
	if err = harness.WriteCollaborationPrepared(layout, prepared); err != nil {
		return err
	}
	return printJSON(prepared)
}

type snapshotFlags map[string]string

type stringFlags []string

func (s *stringFlags) String() string { return strings.Join(*s, ",") }

func (s *stringFlags) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("value cannot be empty")
	}
	*s = append(*s, value)
	return nil
}

func (s *snapshotFlags) String() string {
	if s == nil {
		return ""
	}
	return fmt.Sprint(map[string]string(*s))
}

func (s *snapshotFlags) Set(value string) error {
	id, path, ok := strings.Cut(value, "=")
	if !ok || strings.TrimSpace(id) == "" || !filepath.IsAbs(path) {
		return errors.New("snapshot must be id=/absolute/repository/path")
	}
	if *s == nil {
		*s = snapshotFlags{}
	}
	(*s)[strings.TrimSpace(id)] = filepath.Clean(path)
	return nil
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
