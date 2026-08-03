package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/service"
	"github.com/ShamanicArts/clankspace/internal/store"
)

func setup(t *testing.T) (*store.Store, *service.Service, domain.Principal, domain.Project) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "clankspace.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	p, err := db.EnsureBootstrap(context.Background(), "test-token", "Workshop", "Maintainers")
	if err != nil {
		t.Fatal(err)
	}
	svc := service.New(db)
	project, _, err := svc.CreateProject(context.Background(), p, "project-1", "shuv2code", "shuv2code", "Cross-provider coding sessions")
	if err != nil {
		t.Fatal(err)
	}
	return db, svc, p, project
}

func TestOriginalCoordinationScenario(t *testing.T) {
	db, svc, principal, project := setup(t)
	ctx := context.Background()
	shuvRun, _, err := svc.StartRun(ctx, principal, "run-shuv", domain.StartRunInput{ProjectID: project.ID, AgentName: "Shuv's agent", Harness: "Codex", HarnessVersion: "2026.08", Provider: "openai", Model: "gpt-5.6", Reasoning: "high", Role: "primary", RunType: "interactive", Branch: "session-control", Objective: "Make any session able to control any other session across backend providers", InstructionProfile: []string{"AGENTS.md:sha256:example"}})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.CreateNote(ctx, principal, project.ID, "note-shuv", domain.CreateNoteInput{RunID: shuvRun.ID, Kind: "intent", Title: "Unify cross-provider session control", Summary: "Voice, permissions, interruption handling, and startup behavior are parts of one provider-neutral session-control trajectory.", Rationale: "The voice work requires shared lifecycle and permission infrastructure rather than a voice-only patch.", LedBy: "human", DirectionBasis: "explicit_human_direction", Confidence: "confirmed", Paths: []string{"apps/server/session", "apps/web/permissions"}})
	if err != nil {
		t.Fatal(err)
	}
	trajectory, _, err := svc.CreateTrajectory(ctx, principal, project.ID, "trajectory-shuv", domain.CreateTrajectoryInput{RunID: shuvRun.ID, Objective: "Standardize permissions and lifecycle for provider-neutral cross-session control", Rationale: "Support control between sessions regardless of backend provider.", Paths: []string{"apps/web/permissions", "apps/server/session"}, Branch: "session-control"})
	if err != nil {
		t.Fatal(err)
	}
	shamanicRun, _, err := svc.StartRun(ctx, principal, "run-shamanic", domain.StartRunInput{ProjectID: project.ID, AgentName: "Shamanic's agent", Harness: "shuv2code", HarnessVersion: "nightly", Provider: "openai", Model: "gpt-5.6-sol", Reasoning: "high", Role: "primary", RunType: "interactive", Branch: "permissions-fix", Objective: "Remove the new permission behavior because it appears broken"})
	if err != nil {
		t.Fatal(err)
	}
	brief, err := svc.Brief(ctx, principal, project.ID, domain.BriefInput{RunID: shamanicRun.ID, Query: "permissions session control", Objective: "Remove the new permission layer", Paths: []string{"apps/web/permissions"}, CheckConflicts: true})
	if err != nil {
		t.Fatal(err)
	}
	if brief.Notice != domain.AdvisoryNotice || !strings.Contains(strings.ToLower(brief.Notice), "not instructions") {
		t.Fatalf("brief must explicitly be advisory: %q", brief.Notice)
	}
	if len(brief.Warnings) != 1 {
		t.Fatalf("wanted one possible overlap warning, got %#v", brief.Warnings)
	}
	warning := brief.Warnings[0]
	if warning.Kind != "possible-overlap" || !strings.Contains(warning.Summary, "not a conflict determination") {
		t.Fatalf("warning presented a heuristic match as a conflict: %#v", warning)
	}
	if warning.Trajectory == nil || warning.Trajectory.ID != trajectory.ID {
		t.Fatalf("warning did not identify the intersecting trajectory: %#v", warning)
	}
	if !strings.Contains(warning.Summary, "concurrent edits would collide") {
		t.Fatalf("warning omitted the execution-collision axis: %#v", warning)
	}
	if strings.Join(warning.Options, ",") != "compare,continue-if-compatible-and-independent,pause-if-incompatible-or-collision-prone" {
		t.Fatalf("unexpected options: %#v", warning.Options)
	}
	if len(warning.RelatedNotes) == 0 || warning.RelatedNotes[0].Run == nil {
		t.Fatal("brief lost the rationale or runtime provenance")
	}
	if warning.RelatedNotes[0].Run.Harness != "Codex" || warning.RelatedNotes[0].Run.Role != "primary" {
		t.Fatalf("wrong provenance: %#v", warning.RelatedNotes[0].Run)
	}
	if warning.RelatedNotes[0].Run.AgentName != "Shuv's agent" || warning.RelatedNotes[0].Run.PrincipalName != "Maintainers" {
		t.Fatalf("human-readable attribution was lost: %#v", warning.RelatedNotes[0].Run)
	}
	_ = db
}

func TestIdempotencyAndSecretBoundary(t *testing.T) {
	_, svc, p, project := setup(t)
	ctx := context.Background()
	in := domain.CreateNoteInput{Kind: "decision", Title: "Keep one binary", Summary: "Serve the API and board from one process.", LedBy: "joint", DirectionBasis: "joint_reasoning"}
	first, receipt, err := svc.CreateNote(ctx, p, project.ID, "same", in)
	if err != nil {
		t.Fatal(err)
	}
	second, replay, err := svc.CreateNote(ctx, p, project.ID, "same", in)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || receipt.EventID != replay.EventID {
		t.Fatalf("retry created a duplicate: %#v %#v", receipt, replay)
	}
	in.Summary = "Different request"
	if _, _, err = svc.CreateNote(ctx, p, project.ID, "same", in); !errors.Is(err, store.ErrIdempotencyKeyReuse) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
	in = domain.CreateNoteInput{Kind: "observation", Title: "Credential", Summary: "api_key=definitely-not-for-storage", LedBy: "agent", DirectionBasis: "autonomous_agent_judgment"}
	if _, _, err = svc.CreateNote(ctx, p, project.ID, "secret", in); err == nil {
		t.Fatal("credential-like content was accepted")
	}
}

func TestProjectAgentIdentityIsScoped(t *testing.T) {
	db, svc, owner, project := setup(t)
	ctx := context.Background()
	other, _, err := svc.CreateProject(ctx, owner, "other-project", "other", "Other", "")
	if err != nil {
		t.Fatal(err)
	}
	credential, receipt, err := svc.IssueProjectToken(ctx, owner, project.ID, "agent-token", "shuv2code agents")
	if err != nil {
		t.Fatal(err)
	}
	replayed, replayReceipt, err := svc.IssueProjectToken(ctx, owner, project.ID, "agent-token", "shuv2code agents")
	if err != nil {
		t.Fatal(err)
	}
	if credential.Principal.ID != replayed.Principal.ID || credential.Token != replayed.Token || receipt.EventID != replayReceipt.EventID {
		t.Fatal("project credential retry created a duplicate identity")
	}
	agentPrincipal, err := db.Authenticate(ctx, credential.Token)
	if err != nil {
		t.Fatal(err)
	}
	if agentPrincipal.Kind != "project" {
		t.Fatalf("wanted project principal, got %#v", agentPrincipal)
	}
	projects, err := db.ListProjectsForPrincipal(ctx, agentPrincipal)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].ID != project.ID {
		t.Fatalf("project token escaped scope: %#v", projects)
	}
	if _, _, err = svc.CreateProject(ctx, agentPrincipal, "forbidden", "bad", "Bad", ""); err == nil {
		t.Fatal("project agent created a project")
	}
	if _, err = svc.Brief(ctx, agentPrincipal, other.ID, domain.BriefInput{}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("project agent read another project: %v", err)
	}
	run, _, err := svc.StartRun(ctx, agentPrincipal, "agent-run", domain.StartRunInput{ProjectID: project.Slug, AgentName: "codex", Role: "automation", Objective: "Check project context"})
	if err != nil {
		t.Fatal(err)
	}
	if run.PrincipalID != agentPrincipal.ID {
		t.Fatal("run was not attributed to project agent identity")
	}
	if run.AgentName != "codex" || run.PrincipalName != "shuv2code agents" {
		t.Fatalf("run is not legible to collaborators: %#v", run)
	}
}
