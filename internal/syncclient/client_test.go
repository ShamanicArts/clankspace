package syncclient_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/githubsync"
	"github.com/ShamanicArts/clankspace/internal/httpapi"
	"github.com/ShamanicArts/clankspace/internal/service"
	"github.com/ShamanicArts/clankspace/internal/store"
	"github.com/ShamanicArts/clankspace/internal/syncclient"
)

func testInstance(t *testing.T, name string) (*store.Store, domain.Principal, domain.User, domain.Membership) {
	t.Helper()
	ctx := context.Background()
	db, err := store.OpenWithSecret(filepath.Join(t.TempDir(), "clankspace.db"), name+"-installation-secret")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	principal, err := db.EnsureBootstrap(ctx, name+"-bootstrap", name+" Workspace", name)
	if err != nil {
		t.Fatal(err)
	}
	user, membership, err := db.ClaimBootstrapOwner(ctx, principal.ID, name+"@example.test", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.EnsureInstallationIdentity(ctx, name+" replica", "http://"+name+".test"); err != nil {
		t.Fatal(err)
	}
	if err = db.EnsureAllWorkspaceAuthorities(ctx); err != nil {
		t.Fatal(err)
	}
	return db, principal, user, membership
}

func TestTwoInstancesPairPullPushAndRevoke(t *testing.T) {
	ctx := context.Background()
	authorityDB, authorityPrincipal, _, authorityMembership := testInstance(t, "authority")
	core := service.New(authorityDB)
	project, _, err := core.CreateProject(ctx, authorityPrincipal, "project-create", "shared-project", "Shared project", "A replicated project")
	if err != nil {
		t.Fatal(err)
	}
	repository, _, err := authorityDB.UpsertRepository(ctx, authorityPrincipal, project.ID, "shared-repository", domain.Repository{URL: "https://github.com/example/repo", Host: "github.com", Owner: "example", Name: "repo", Visibility: "public"})
	if err != nil {
		t.Fatal(err)
	}
	initial, _, err := core.CreateNote(ctx, authorityPrincipal, project.ID, "initial-note", domain.CreateNoteInput{Kind: "intent", Title: "Keep the API small", Summary: "The first client stays deliberately narrow.", Rationale: "Agents should not gain unrelated administration tools.", LedBy: "human", DirectionBasis: "explicit_human_direction", Confidence: "high"})
	if err != nil {
		t.Fatal(err)
	}
	authorityServer := httptest.NewServer((&httpapi.Server{Store: authorityDB, Core: core, GitHub: githubsync.New(""), Log: slog.Default(), BaseURL: "http://authority.test", AuthMode: "hybrid", SyncEnabled: true, ReplicaName: "authority replica"}).Handler())
	t.Cleanup(authorityServer.Close)
	code, _, err := authorityDB.CreateReplicaOffer(ctx, authorityMembership, []string{"pull", "push"})
	if err != nil {
		t.Fatal(err)
	}

	replicaDB, _, replicaUser, _ := testInstance(t, "replica")
	client := syncclient.New()
	joined, err := client.Join(ctx, replicaDB, authorityServer.URL, code, "replica node", "http://replica.test", replicaUser.ID)
	if err != nil {
		t.Fatal(err)
	}
	if joined.ID != authorityMembership.WorkspaceID {
		t.Fatalf("joined workspace = %#v", joined)
	}
	replicaProject, err := replicaDB.GetProject(ctx, joined.ID, project.ID)
	if err != nil || replicaProject.Name != project.Name {
		t.Fatalf("replica project = %#v, %v", replicaProject, err)
	}
	replicaNotes, err := replicaDB.ListNotes(ctx, project.ID, -1)
	if err != nil || len(replicaNotes) != 1 || replicaNotes[0].ID != initial.ID {
		t.Fatalf("initial snapshot notes = %#v, %v", replicaNotes, err)
	}

	later, _, err := core.CreateNote(ctx, authorityPrincipal, project.ID, "later-note", domain.CreateNoteInput{Kind: "decision", Title: "Keep local writes available", Summary: "A disconnected replica may append before reconnecting.", Rationale: "The coordination layer must not block coding when the host is unavailable.", LedBy: "human", DirectionBasis: "explicit_human_direction", Confidence: "high"})
	if err != nil {
		t.Fatal(err)
	}
	replicaHeads, err := replicaDB.SyncHeads(ctx, joined.ID)
	if err != nil {
		t.Fatal(err)
	}
	pending, _, err := authorityDB.EventsAfter(ctx, joined.ID, replicaHeads, 10)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending events = %#v, %v", pending, err)
	}
	tampered := append([]domain.DomainEvent(nil), pending...)
	tampered[0].Payload = []byte(`{"title":"forged"}`)
	if _, err = replicaDB.ImportEvents(ctx, joined.ID, tampered); err == nil {
		t.Fatal("tampered event was accepted")
	}
	if err = client.SyncAll(ctx, replicaDB); err != nil {
		t.Fatal(err)
	}
	replicaNotes, _ = replicaDB.ListNotes(ctx, project.ID, -1)
	if len(replicaNotes) != 2 || replicaNotes[0].ID != later.ID {
		t.Fatalf("pulled notes = %#v", replicaNotes)
	}

	replicaMembership, err := replicaDB.Membership(ctx, replicaUser.ID, joined.ID)
	if err != nil {
		t.Fatal(err)
	}
	replicaPrincipal, err := replicaDB.PrincipalForMembership(ctx, replicaMembership)
	if err != nil {
		t.Fatal(err)
	}
	replicaCore := service.New(replicaDB)
	run, _, err := replicaCore.StartRun(ctx, replicaPrincipal, "replica-run", domain.StartRunInput{ProjectID: project.ID, RepositoryID: repository.ID, AgentName: "Local agent", Harness: "codex", Provider: "openai", Model: "test-model", Reasoning: "high", Role: "primary", RunType: "interactive", Branch: "offline/provenance", BaseSHA: "1111111111111111111111111111111111111111", HeadSHA: "1111111111111111111111111111111111111111", Objective: "Record offline context"})
	if err != nil {
		t.Fatal(err)
	}
	offline, _, err := replicaCore.CreateNote(ctx, replicaPrincipal, project.ID, "offline-note", domain.CreateNoteInput{RunID: run.ID, Kind: "checkpoint", Title: "Local validation passed", Summary: "The local replica completed its focused check while disconnected.", Rationale: "This proves local writes do not wait for the authority.", LedBy: "agent", DirectionBasis: "autonomous_agent_judgment", Confidence: "high", Verification: "focused test passed"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = replicaDB.EndRun(ctx, replicaPrincipal, "replica-run-end", run.ID, domain.EndRunInput{Outcome: "completed", HeadSHA: "2222222222222222222222222222222222222222", PullRequestURL: "https://github.com/example/repo/pull/7", PullRequestNumber: 7, PullRequestState: "open"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = replicaDB.LinkRunDelivery(ctx, replicaPrincipal, "replica-run-merged", run.ID, domain.LinkRunDeliveryInput{PullRequestState: "merged", MergeCommitSHA: "3333333333333333333333333333333333333333"}); err != nil {
		t.Fatal(err)
	}
	if err = client.SyncAll(ctx, replicaDB); err != nil {
		t.Fatal(err)
	}
	authorityNotes, err := authorityDB.ListNotes(ctx, project.ID, -1)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, note := range authorityNotes {
		found = found || note.ID == offline.ID
	}
	if !found {
		t.Fatalf("pushed note missing: %#v", authorityNotes)
	}
	for _, note := range authorityNotes {
		if note.ID == offline.ID && (note.Run == nil || note.Run.PullRequestNumber != 7 || note.Run.PullRequestState != "merged" || note.Run.MergeCommitSHA == "") {
			t.Fatalf("delivery provenance did not converge: %#v", note.Run)
		}
	}
	_, _, err = core.SupersedeNote(ctx, authorityPrincipal, project.ID, "authority-supersede", initial.ID, domain.SupersedeNoteInput{ExpectedRevision: 1, Reason: "Authority learned a new constraint.", Replacement: &domain.CreateNoteInput{Kind: "intent", Title: "Authority successor", Summary: "The authority recorded one new account.", LedBy: "human", DirectionBasis: "explicit_human_direction", Confidence: "high"}})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = replicaCore.SupersedeNote(ctx, replicaPrincipal, project.ID, "replica-supersede", initial.ID, domain.SupersedeNoteInput{RunID: run.ID, ExpectedRevision: 1, Reason: "Replica learned a different constraint while offline.", Replacement: &domain.CreateNoteInput{Kind: "intent", Title: "Replica successor", Summary: "The replica recorded a different new account.", LedBy: "agent", DirectionBasis: "interpreted_human_intent", Confidence: "medium"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = client.SyncAll(ctx, replicaDB); err != nil {
		t.Fatal(err)
	}
	for label, database := range map[string]*store.Store{"authority": authorityDB, "replica": replicaDB} {
		notes, listErr := database.ListNotes(ctx, project.ID, -1)
		if listErr != nil {
			t.Fatal(listErr)
		}
		var target domain.Note
		for _, note := range notes {
			if note.ID == initial.ID {
				target = note
			}
		}
		if target.Status != "contested" || target.SupersededBy != "" {
			t.Fatalf("%s concurrent supersession = %#v", label, target)
		}
		edges, edgeErr := database.ListLifecycleEdges(ctx, project.ID)
		if edgeErr != nil || len(edges) != 2 {
			t.Fatalf("%s lifecycle edges = %#v, %v", label, edges, edgeErr)
		}
	}

	localReplicaID := replicaDB.LocalReplicaID()
	if err = authorityDB.RevokeReplica(ctx, authorityMembership, localReplicaID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = replicaCore.CreateNote(ctx, replicaPrincipal, project.ID, "post-revoke-note", domain.CreateNoteInput{RunID: run.ID, Kind: "observation", Title: "Created after revocation", Summary: "This record must remain local.", LedBy: "agent", DirectionBasis: "autonomous_agent_judgment", Confidence: "high"}); err != nil {
		t.Fatal(err)
	}
	if err = client.SyncAll(ctx, replicaDB); err == nil {
		t.Fatal("revoked replica synchronized successfully")
	}
	authorityNotes, _ = authorityDB.ListNotes(ctx, project.ID, -1)
	for _, note := range authorityNotes {
		if note.Title == "Created after revocation" {
			t.Fatal("authority accepted a post-revocation event")
		}
	}
}

func TestSelfHostedAuthorityMirrorsToCloudAndStaysAuthority(t *testing.T) {
	ctx := context.Background()
	selfDB, selfPrincipal, _, selfMembership := testInstance(t, "selfhost")
	selfCore := service.New(selfDB)
	project, _, err := selfCore.CreateProject(ctx, selfPrincipal, "self-project", "portable-space", "Portable Space", "Created on the self-host")
	if err != nil {
		t.Fatal(err)
	}
	seed, _, err := selfCore.CreateNote(ctx, selfPrincipal, project.ID, "self-seed", domain.CreateNoteInput{Kind: "intent", Title: "Self-host remains authority", Summary: "Cloud is a synchronized copy, not the owner of this workspace.", Rationale: "The collaborators choose where workspace authority lives.", LedBy: "human", DirectionBasis: "explicit_human_direction", Confidence: "high"})
	if err != nil {
		t.Fatal(err)
	}

	cloudDB, _, cloudUser, _ := testInstance(t, "cloud")
	cloudCore := service.New(cloudDB)
	cloudServer := httptest.NewServer((&httpapi.Server{Store: cloudDB, Core: cloudCore, GitHub: githubsync.New(""), Log: slog.Default(), BaseURL: "http://cloud.test", AuthMode: "hybrid", SyncEnabled: true, ReplicaName: "cloud replica"}).Handler())
	t.Cleanup(cloudServer.Close)
	code, _, err := cloudDB.CreateMirrorOffer(ctx, cloudUser.ID)
	if err != nil {
		t.Fatal(err)
	}
	client := syncclient.New()
	mirrored, err := client.Mirror(ctx, selfDB, selfMembership.WorkspaceID, cloudServer.URL, code)
	if err != nil {
		t.Fatal(err)
	}
	if mirrored.ID != selfMembership.WorkspaceID {
		t.Fatalf("mirrored workspace = %#v", mirrored)
	}
	cloudWorkspace, err := cloudDB.GetWorkspace(ctx, mirrored.ID)
	if err != nil || cloudWorkspace.AuthorityReplicaID != selfDB.LocalReplicaID() {
		t.Fatalf("cloud workspace = %#v, %v", cloudWorkspace, err)
	}
	cloudNotes, err := cloudDB.ListNotes(ctx, project.ID, -1)
	if err != nil || len(cloudNotes) != 1 || cloudNotes[0].ID != seed.ID {
		t.Fatalf("cloud snapshot notes = %#v, %v", cloudNotes, err)
	}
	if allowed, err := cloudDB.CanShareHumans(ctx, mirrored.ID); err != nil || allowed {
		t.Fatalf("cloud mirror human sharing = %v, %v", allowed, err)
	}
	if allowed, err := selfDB.CanShareHumans(ctx, mirrored.ID); err != nil || !allowed {
		t.Fatalf("self-host authority human sharing = %v, %v", allowed, err)
	}

	cloudMembership, err := cloudDB.Membership(ctx, cloudUser.ID, mirrored.ID)
	if err != nil {
		t.Fatal(err)
	}
	cloudPrincipal, err := cloudDB.PrincipalForMembership(ctx, cloudMembership)
	if err != nil {
		t.Fatal(err)
	}
	cloudNote, _, err := cloudCore.CreateNote(ctx, cloudPrincipal, project.ID, "cloud-note", domain.CreateNoteInput{Kind: "understanding", Title: "Cloud collaborator context", Summary: "A cloud-side human can append while the self-host is unavailable.", LedBy: "human", DirectionBasis: "explicit_human_direction", Confidence: "medium"})
	if err != nil {
		t.Fatal(err)
	}
	if err = client.SyncAll(ctx, selfDB); err != nil {
		t.Fatal(err)
	}
	selfNotes, err := selfDB.ListNotes(ctx, project.ID, -1)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, note := range selfNotes {
		found = found || note.ID == cloudNote.ID
	}
	if !found {
		t.Fatalf("cloud-origin note did not reach self-host: %#v", selfNotes)
	}
}

func TestWorkspaceSnapshotDoesNotTruncateRunHistory(t *testing.T) {
	ctx := context.Background()
	db, principal, _, _ := testInstance(t, "snapshot")
	core := service.New(db)
	project, _, err := core.CreateProject(ctx, principal, "snapshot-project", "complete-history", "Complete history", "Snapshot coverage test")
	if err != nil {
		t.Fatal(err)
	}
	repository, _, err := db.UpsertRepository(ctx, principal, project.ID, "snapshot-repository", domain.Repository{URL: "https://github.com/example/repo", Host: "github.com", Owner: "example", Name: "repo", Visibility: "public"})
	if err != nil {
		t.Fatal(err)
	}
	var deliveryRun domain.Run
	for index := 0; index < 501; index++ {
		input := domain.StartRunInput{ProjectID: project.ID, AgentName: "archive-agent", Role: "automation", Objective: fmt.Sprintf("Archived run %03d", index)}
		if index == 0 {
			input.RepositoryID, input.Branch, input.BaseSHA, input.HeadSHA = repository.ID, "feature/snapshot", "1111111111111111111111111111111111111111", "1111111111111111111111111111111111111111"
		}
		run, _, startErr := core.StartRun(ctx, principal, fmt.Sprintf("snapshot-run-%03d", index), input)
		err = startErr
		if err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			merged := time.Date(2026, time.August, 4, 2, 0, 0, 0, time.UTC)
			deliveryRun, _, err = db.LinkRunDelivery(ctx, principal, "snapshot-delivery", run.ID, domain.LinkRunDeliveryInput{DeliveryBranch: "release/snapshot", HeadSHA: "2222222222222222222222222222222222222222", PullRequestURL: "https://github.com/example/repo/pull/5", PullRequestNumber: 5, PullRequestState: "merged", MergeCommitSHA: "3333333333333333333333333333333333333333", MergedAt: &merged})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	snapshot, err := db.BuildWorkspaceSnapshot(ctx, principal.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Projects) != 1 || len(snapshot.Projects[0].Runs) != 501 {
		t.Fatalf("snapshot truncated run history: projects=%d runs=%d", len(snapshot.Projects), len(snapshot.Projects[0].Runs))
	}
	found := false
	for _, run := range snapshot.Projects[0].Runs {
		if run.ID == deliveryRun.ID {
			found = run.Branch == "feature/snapshot" && run.DeliveryBranch == "release/snapshot" && run.PullRequestNumber == 5 && run.MergeCommitSHA != "" && run.MergedAt != nil
		}
	}
	if !found {
		t.Fatal("snapshot lost run delivery provenance")
	}
	replicas, err := db.ListReplicas(ctx, principal.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	var authority domain.Replica
	for _, replica := range replicas {
		if replica.Role == "authority" {
			authority = replica
		}
	}
	if authority.ID == "" {
		t.Fatal("snapshot source authority is missing")
	}
	importedDB, _, importedUser, _ := testInstance(t, "snapshot-import")
	if err = importedDB.ImportWorkspaceSnapshot(ctx, snapshot, authority, importedUser.ID); err != nil {
		t.Fatal(err)
	}
	imported, err := importedDB.GetRun(ctx, deliveryRun.ID)
	if err != nil || imported.Branch != "feature/snapshot" || imported.BaseSHA != "1111111111111111111111111111111111111111" || imported.DeliveryBranch != "release/snapshot" || imported.HeadSHA != "2222222222222222222222222222222222222222" || imported.PullRequestURL != "https://github.com/example/repo/pull/5" || imported.PullRequestNumber != 5 || imported.PullRequestState != "merged" || imported.MergeCommitSHA == "" || imported.MergedAt == nil {
		t.Fatalf("snapshot import lost run delivery provenance: %#v, %v", imported, err)
	}
}
