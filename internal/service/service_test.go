package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if warning.ExecutionRisk != "live-interactive-overlap" || !strings.Contains(warning.Summary, "live collision candidate") {
		t.Fatalf("warning omitted live execution provenance: %#v", warning)
	}
	if strings.Join(warning.Options, ",") != "compare,continue-if-same-objective,pause-if-distinct-concurrent-objective" {
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
	in = domain.CreateNoteInput{Kind: "checkpoint", Title: "Contradictory provenance", Summary: "The team selected this direction.", LedBy: "joint", DirectionBasis: "autonomous_agent_judgment"}
	if _, _, err = svc.CreateNote(ctx, p, project.ID, "bad-provenance", in); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("incoherent lead/basis pairing was accepted: %v", err)
	}
}

func TestDecisionAndCheckpointInheritOriginAndDeliveryProvenance(t *testing.T) {
	db, svc, principal, project := setup(t)
	ctx := context.Background()
	repository, _, err := db.UpsertRepository(ctx, principal, project.ID, "delivery-repository", domain.Repository{URL: "https://github.com/example/repo", Host: "github.com", Owner: "example", Name: "repo", Visibility: "public"})
	if err != nil {
		t.Fatal(err)
	}
	run, _, err := svc.StartRun(ctx, principal, "delivery-run", domain.StartRunInput{
		ProjectID: project.ID, AgentName: "Codex", RepositoryID: repository.ID, VCS: "jj", Branch: "feature/provenance",
		BaseSHA: "1111111111111111111111111111111111111111", HeadSHA: "1111111111111111111111111111111111111111",
		JJWorkspace: "agent-workspace", JJChangeID: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", JJCommitID: "1111111111111111111111111111111111111111", JJBookmarks: []string{"agent/provenance"},
	})
	if err != nil {
		t.Fatal(err)
	}
	trajectory, _, err := svc.CreateTrajectory(ctx, principal, project.ID, "delivery-trajectory", domain.CreateTrajectoryInput{RunID: run.ID, Objective: "Preserve native Jujutsu provenance"})
	if err != nil {
		t.Fatal(err)
	}
	if trajectory.VCS != "jj" || trajectory.JJWorkspace != "agent-workspace" || trajectory.JJChangeID != run.JJChangeID || len(trajectory.JJBookmarks) != 1 {
		t.Fatalf("trajectory did not inherit Jujutsu provenance: %#v", trajectory)
	}
	for _, kind := range []string{"decision", "checkpoint"} {
		if _, _, err = svc.CreateNote(ctx, principal, project.ID, "delivery-note-"+kind, domain.CreateNoteInput{RunID: run.ID, Kind: kind, Title: "Delivery provenance " + kind, Summary: "This record inherits delivery evidence from its run.", LedBy: "agent", DirectionBasis: "autonomous_agent_judgment"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err = db.EndRun(ctx, principal, "delivery-end", run.ID, domain.EndRunInput{
		Outcome: "completed", VCS: "jj", DeliveryBranch: "release/provenance", HeadSHA: "2222222222222222222222222222222222222222",
		DeliveryJJWorkspace: "agent-workspace", DeliveryJJChangeID: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", DeliveryJJCommitID: "2222222222222222222222222222222222222222", DeliveryJJBookmarks: []string{"agent/provenance"},
		PullRequestURL: "https://github.com/example/repo/pull/42", PullRequestNumber: 42, PullRequestState: "open",
	}); err != nil {
		t.Fatal(err)
	}
	merged := time.Date(2026, time.August, 4, 3, 0, 0, 0, time.UTC)
	if _, _, err = db.LinkRunDelivery(ctx, principal, "delivery-merge", run.ID, domain.LinkRunDeliveryInput{PullRequestState: "merged", MergeCommitSHA: "3333333333333333333333333333333333333333", MergedAt: &merged}); err != nil {
		t.Fatal(err)
	}
	notes, err := db.ListNotes(ctx, project.ID, 10)
	if err != nil || len(notes) != 2 {
		t.Fatalf("notes = %#v, %v", notes, err)
	}
	for _, note := range notes {
		if note.Run == nil || note.Run.BaseSHA != "1111111111111111111111111111111111111111" || note.Run.HeadSHA != "2222222222222222222222222222222222222222" {
			t.Fatalf("origin/delivery coordinates lost: %#v", note.Run)
		}
		if note.Run.Branch != "feature/provenance" || note.Run.DeliveryBranch != "release/provenance" {
			t.Fatalf("origin branch was rewritten: %#v", note.Run)
		}
		if note.Run.VCS != "jj" || note.Run.JJWorkspace != "agent-workspace" || note.Run.JJChangeID != note.Run.DeliveryJJChangeID || note.Run.JJCommitID == note.Run.DeliveryJJCommitID || note.Run.DeliveryJJCommitID != "2222222222222222222222222222222222222222" {
			t.Fatalf("Jujutsu change evolution lost: %#v", note.Run)
		}
		if len(note.Run.JJBookmarks) != 1 || len(note.Run.DeliveryJJBookmarks) != 1 || note.Run.DeliveryJJBookmarks[0] != "agent/provenance" {
			t.Fatalf("Jujutsu bookmarks lost: %#v", note.Run)
		}
		if note.Run.PullRequestNumber != 42 || note.Run.PullRequestState != "merged" || note.Run.MergeCommitSHA != "3333333333333333333333333333333333333333" || note.Run.MergedAt == nil {
			t.Fatalf("pull request lifecycle lost: %#v", note.Run)
		}
	}
}

func TestRunDeliveryPreservesEvidenceAndRejectsForeignRepository(t *testing.T) {
	db, svc, principal, project := setup(t)
	ctx := context.Background()
	repository, _, err := db.UpsertRepository(ctx, principal, project.ID, "delivery-preserve-repository", domain.Repository{URL: "https://github.com/example/repo", Host: "github.com", Owner: "example", Name: "repo", Visibility: "public"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = svc.StartRun(ctx, principal, "foreign-repository-run", domain.StartRunInput{ProjectID: project.ID, AgentName: "Codex", RepositoryID: "repo-from-another-project"}); err == nil {
		t.Fatal("run accepted a repository that is not attached to its project")
	}
	if _, _, err = svc.StartRun(ctx, principal, "invalid-jj-run", domain.StartRunInput{ProjectID: project.ID, AgentName: "Codex", VCS: "jj", JJChangeID: "not/a/change"}); err == nil {
		t.Fatal("run accepted an invalid Jujutsu change ID")
	}
	run, _, err := svc.StartRun(ctx, principal, "preserve-run", domain.StartRunInput{ProjectID: project.ID, AgentName: "Codex", RepositoryID: repository.ID, Branch: "feature/origin", BaseSHA: "1111111111111111111111111111111111111111", HeadSHA: "1111111111111111111111111111111111111111"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = db.LinkRunDelivery(ctx, principal, "preserve-link", run.ID, domain.LinkRunDeliveryInput{DeliveryBranch: "feature/delivery", HeadSHA: "2222222222222222222222222222222222222222", PullRequestURL: "https://github.com/example/repo/pull/9", PullRequestNumber: 9, PullRequestState: "open"}); err != nil {
		t.Fatal(err)
	}
	ended, _, err := db.EndRun(ctx, principal, "preserve-end", run.ID, domain.EndRunInput{Outcome: "completed", Verification: "focused checks passed"})
	if err != nil {
		t.Fatal(err)
	}
	if ended.Branch != "feature/origin" || ended.DeliveryBranch != "feature/delivery" || ended.PullRequestNumber != 9 || ended.PullRequestState != "open" {
		t.Fatalf("partial end erased delivery evidence: %#v", ended)
	}
	if _, _, err = db.LinkRunDelivery(ctx, principal, "foreign-pr", run.ID, domain.LinkRunDeliveryInput{PullRequestURL: "https://github.com/other/repo/pull/9", PullRequestNumber: 9, PullRequestState: "open"}); err == nil {
		t.Fatal("run accepted a pull request from another repository")
	}
	if _, _, err = db.LinkRunDelivery(ctx, principal, "malformed-pr", run.ID, domain.LinkRunDeliveryInput{PullRequestURL: "%", PullRequestNumber: 9, PullRequestState: "open"}); err == nil {
		t.Fatal("run accepted a malformed pull request URL")
	}
	legacy, _, err := svc.StartRun(ctx, principal, "legacy-run", domain.StartRunInput{ProjectID: project.ID, AgentName: "Legacy agent", Branch: "main"})
	if err != nil {
		t.Fatal(err)
	}
	backfilled, _, err := db.LinkRunDelivery(ctx, principal, "legacy-backfill", legacy.ID, domain.LinkRunDeliveryInput{RepositoryID: repository.ID, DeliveryBranch: "feature/legacy", HeadSHA: "4444444444444444444444444444444444444444", PullRequestURL: "https://github.com/example/repo/pull/10", PullRequestNumber: 10, PullRequestState: "open"})
	if err != nil || backfilled.RepositoryID != repository.ID || backfilled.PullRequestNumber != 10 {
		t.Fatalf("legacy repository backfill = %#v, %v", backfilled, err)
	}
	other, _, err := db.UpsertRepository(ctx, principal, project.ID, "other-repository", domain.Repository{URL: "https://github.com/example/other", Host: "github.com", Owner: "example", Name: "other", Visibility: "public"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = db.LinkRunDelivery(ctx, principal, "replace-repository", legacy.ID, domain.LinkRunDeliveryInput{RepositoryID: other.ID}); err == nil {
		t.Fatal("legacy repository backfill replaced an existing repository")
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
	if _, _, err = svc.StartRun(ctx, agentPrincipal, "escaped-run", domain.StartRunInput{ProjectID: other.ID, AgentName: "codex", Role: "automation", Objective: "Escape the assigned project"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("project agent started a run in another project: %v", err)
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
