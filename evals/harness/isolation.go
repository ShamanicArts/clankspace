package harness

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/ShamanicArts/clankspace/internal/client"
	"github.com/ShamanicArts/clankspace/internal/domain"
)

var isolationProbeID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,39}$`)

type IsolationProbeOptions struct {
	AdminEnvironment string
	ProbeID          string
}

type IsolationProbeChecks struct {
	ProjectListScoped     bool `json:"projectListScoped"`
	LocalBriefClean       bool `json:"localBriefClean"`
	ForeignBriefDenied    bool `json:"foreignBriefDenied"`
	ForeignExportDenied   bool `json:"foreignExportDenied"`
	ForeignRunsDenied     bool `json:"foreignRunsDenied"`
	ForeignMutationDenied bool `json:"foreignMutationDenied"`
}

type IsolationProbeResult struct {
	SchemaVersion   int                  `json:"schemaVersion"`
	ProbeID         string               `json:"probeId"`
	ProjectA        string               `json:"projectA"`
	ProjectB        string               `json:"projectB"`
	VisibleProjects []string             `json:"visibleProjects"`
	DecoyMarker     string               `json:"decoyMarker"`
	Checks          IsolationProbeChecks `json:"checks"`
	Leaked          bool                 `json:"leaked"`
	Passed          bool                 `json:"passed"`
}

func RunIsolationProbe(ctx context.Context, options IsolationProbeOptions) (IsolationProbeResult, error) {
	result := IsolationProbeResult{SchemaVersion: 1, ProbeID: options.ProbeID}
	if options.AdminEnvironment == "" || !isolationProbeID.MatchString(options.ProbeID) {
		return result, errors.New("admin environment and a lowercase 3-40 character probe ID are required")
	}
	url, token, err := LoadAdminEnvironment(options.AdminEnvironment)
	if err != nil {
		return result, err
	}
	admin := client.New(url, token)
	result.ProjectA = "isolation-a-" + options.ProbeID
	result.ProjectB = "isolation-b-" + options.ProbeID
	result.DecoyMarker = "orbital-needle-" + options.ProbeID
	projectA, err := ensureProject(ctx, admin, result.ProjectA, "Isolation probe A", "Agent-visible isolation probe project", key(options.ProbeID, "project-a"))
	if err != nil {
		return result, err
	}
	projectB, err := ensureProject(ctx, admin, result.ProjectB, "Isolation probe B", "Foreign decoy project", key(options.ProbeID, "project-b"))
	if err != nil {
		return result, err
	}
	var credential domain.ProjectCredential
	if err = admin.DoWithKey(ctx, http.MethodPost, "/projects/"+projectA.ID+"/tokens", key(options.ProbeID, "token-a"), map[string]string{"displayName": "isolation probe agent"}, &credential); err != nil {
		return result, err
	}
	var seeded struct {
		Run domain.Run `json:"run"`
	}
	if err = admin.DoWithKey(ctx, http.MethodPost, "/runs", key(options.ProbeID, "run-b"), domain.StartRunInput{
		ProjectID: projectB.ID, AgentName: "decoy-seeder", Harness: "clank-eval", Provider: "deterministic",
		Model: "none", Role: "automation", RunType: "automation", Objective: "Seed foreign-project homonym evidence",
	}, &seeded); err != nil {
		return result, err
	}
	var noteOut struct {
		Note domain.Note `json:"note"`
	}
	if err = admin.DoWithKey(ctx, http.MethodPost, "/projects/"+projectB.ID+"/notes", key(options.ProbeID, "note-b"), domain.CreateNoteInput{
		RunID: seeded.Run.ID, Kind: "decision", Title: "Request ID coverage direction",
		Summary:   "Foreign-project decoy marker " + result.DecoyMarker + " applies only to project B.",
		Rationale: "An attractive matching phrase verifies that retrieval remains project-scoped.",
		LedBy:     "agent", DirectionBasis: "autonomous_agent_judgment", Paths: []string{"middleware/request_id_test.go"},
	}, &noteOut); err != nil {
		return result, err
	}
	var ended struct {
		Run domain.Run `json:"run"`
	}
	if err = admin.DoWithKey(ctx, http.MethodPost, "/runs/"+seeded.Run.ID+"/end", key(options.ProbeID, "end-b"), domain.EndRunInput{
		Outcome: "completed", Verification: "Foreign-project decoy seeded for the isolation probe.",
	}, &ended); err != nil {
		return result, err
	}
	agent := client.New(url, credential.Token)
	projects, listErr := agent.ListProjects(ctx)
	for _, project := range projects {
		result.VisibleProjects = append(result.VisibleProjects, project.Slug)
	}
	result.Checks.ProjectListScoped = listErr == nil && len(projects) == 1 && projects[0].ID == projectA.ID
	brief, localBriefErr := agent.Brief(ctx, projectA.Slug, domain.BriefInput{
		Query: result.DecoyMarker, Objective: "Check request ID coverage", Paths: []string{"middleware/request_id_test.go"}, CheckConflicts: true,
	})
	briefBody, _ := json.Marshal(brief)
	lowerBrief := strings.ToLower(string(briefBody))
	result.Checks.LocalBriefClean = localBriefErr == nil &&
		!strings.Contains(lowerBrief, strings.ToLower(result.DecoyMarker)) &&
		!strings.Contains(lowerBrief, strings.ToLower(projectB.Slug))
	_, foreignBriefErr := agent.Brief(ctx, projectB.Slug, domain.BriefInput{Query: result.DecoyMarker, CheckConflicts: true})
	result.Checks.ForeignBriefDenied = foreignBriefErr != nil
	_, foreignExportErr := agent.ExportProject(ctx, projectB.Slug)
	result.Checks.ForeignExportDenied = foreignExportErr != nil
	_, foreignRunsErr := agent.ListRuns(ctx, projectB.Slug, 10)
	result.Checks.ForeignRunsDenied = foreignRunsErr != nil
	_, foreignMutationErr := agent.CreateNote(ctx, projectB.Slug, domain.CreateNoteInput{
		RunID: seeded.Run.ID, Kind: "observation", Title: "forbidden probe write",
		Summary: "This write must be denied by project scope.", LedBy: "agent", DirectionBasis: "autonomous_agent_judgment",
	})
	result.Checks.ForeignMutationDenied = foreignMutationErr != nil
	result.Leaked = !result.Checks.ProjectListScoped || !result.Checks.LocalBriefClean ||
		!result.Checks.ForeignBriefDenied || !result.Checks.ForeignExportDenied ||
		!result.Checks.ForeignRunsDenied || !result.Checks.ForeignMutationDenied
	result.Passed = !result.Leaked
	if !result.Passed {
		return result, fmt.Errorf("project isolation probe %s failed", options.ProbeID)
	}
	return result, nil
}
