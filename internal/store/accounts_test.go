package store

import (
	"context"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func authTokenFromBody(t *testing.T, body string) string {
	t.Helper()
	marker := "/?token="
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatalf("auth link missing from mail: %q", body)
	}
	value := body[start+len(marker):]
	if newline := strings.IndexByte(value, '\n'); newline >= 0 {
		value = value[:newline]
	}
	decoded, err := url.QueryUnescape(strings.TrimSpace(value))
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func inviteTokenFromURL(t *testing.T, address string) string {
	t.Helper()
	parsed, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}
	token := parsed.Query().Get("invite")
	if token == "" {
		t.Fatalf("invite token missing from %q", address)
	}
	return token
}

func TestDirectInviteAndPasswordLoginDoNotNeedEmail(t *testing.T) {
	ctx := context.Background()
	db, err := OpenWithSecret(filepath.Join(t.TempDir(), "clankspace.db"), "direct-invite-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ownerPrincipal, err := db.EnsureBootstrap(ctx, "bootstrap-token", "Shared Studio", "Shamanic")
	if err != nil {
		t.Fatal(err)
	}
	owner, membership, err := db.ClaimBootstrapOwner(ctx, ownerPrincipal.ID, "owner@example.test", "Owner")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.SetUserPassword(ctx, owner.ID, "correct horse battery staple"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.AuthenticatePassword(ctx, owner.Email, "wrong password", "test-source"); err == nil {
		t.Fatal("wrong password was accepted")
	}
	ownerSession, err := db.AuthenticatePassword(ctx, owner.Email, "correct horse battery staple", "test-source")
	if err != nil || ownerSession.SessionToken == "" {
		t.Fatalf("owner password login = %#v, %v", ownerSession, err)
	}
	link, err := db.CreateWorkspaceInviteLink(ctx, membership, "friend@example.test", "member", "https://clank.test")
	if err != nil {
		t.Fatal(err)
	}
	if messages, claimErr := db.ClaimOutbox(ctx, 10); claimErr != nil || len(messages) != 0 {
		t.Fatalf("direct invite queued email: %#v, %v", messages, claimErr)
	}
	token := inviteTokenFromURL(t, link.URL)
	preview, err := db.WorkspaceInvitePreview(ctx, token)
	if err != nil || preview.Email != "friend@example.test" || preview.WorkspaceName != "Shared Studio" {
		t.Fatalf("invite preview = %#v, %v", preview, err)
	}
	accepted, err := db.ConsumeInviteWithPassword(ctx, token, "Friend", "friend password")
	if err != nil || accepted.User.Email != "friend@example.test" {
		t.Fatalf("accept invite = %#v, %v", accepted, err)
	}
	if _, err = db.ConsumeInviteWithPassword(ctx, token, "Friend", "friend password"); err == nil {
		t.Fatal("one-time direct invite was accepted twice")
	}
	if _, err = db.AuthenticatePassword(ctx, "friend@example.test", "friend password", "friend-source"); err != nil {
		t.Fatalf("invited user password login: %v", err)
	}
}

func TestHostedInviteLoginAndMultiWorkspaceFlow(t *testing.T) {
	ctx := context.Background()
	db, err := OpenWithSecret(filepath.Join(t.TempDir(), "clankspace.db"), "test-installation-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ownerPrincipal, err := db.EnsureBootstrap(ctx, "bootstrap-token", "Shared Studio", "Shamanic")
	if err != nil {
		t.Fatal(err)
	}
	owner, ownerMembership, err := db.ClaimBootstrapOwner(ctx, ownerPrincipal.ID, "shamanic@example.test", "Shamanic")
	if err != nil {
		t.Fatal(err)
	}
	if ownerMembership.Role != "owner" {
		t.Fatalf("owner membership = %#v", ownerMembership)
	}
	personal, _, err := db.CreateWorkspaceForUser(ctx, owner.ID, "private-lab", "Private Lab")
	if err != nil {
		t.Fatal(err)
	}
	workspaces, err := db.ListWorkspacesForUser(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 2 || personal.Slug != "private-lab" {
		t.Fatalf("workspaces = %#v", workspaces)
	}
	invite, err := db.CreateWorkspaceInvite(ctx, ownerMembership, "shuv@example.test", "member", "http://clank.test")
	if err != nil {
		t.Fatal(err)
	}
	if invite.Email != "shuv@example.test" {
		t.Fatalf("invite = %#v", invite)
	}
	messages, err := db.ClaimOutbox(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Template != "workspace_invite" {
		t.Fatalf("messages = %#v", messages)
	}
	result, err := db.ConsumeAuthToken(ctx, authTokenFromBody(t, messages[0].Body), "Shuv")
	if err != nil {
		t.Fatal(err)
	}
	if result.User.DisplayName != "Shuv" || result.SessionToken == "" || result.CSRFToken == "" {
		t.Fatalf("session result = %#v", result)
	}
	if _, err = db.AuthenticateSession(ctx, result.SessionToken, result.CSRFToken, true); err != nil {
		t.Fatal(err)
	}
	if _, err = db.AuthenticateSession(ctx, result.SessionToken, "wrong", true); err == nil {
		t.Fatal("wrong CSRF token was accepted")
	}
	shuvWorkspaces, err := db.ListWorkspacesForUser(ctx, result.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(shuvWorkspaces) != 1 || shuvWorkspaces[0].ID != ownerMembership.WorkspaceID {
		t.Fatalf("invited workspace leakage: %#v", shuvWorkspaces)
	}
	shuvMembership, err := db.Membership(ctx, result.User.ID, ownerMembership.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	shuvPrincipal, err := db.PrincipalForMembership(ctx, shuvMembership)
	if err != nil {
		t.Fatal(err)
	}
	project, _, err := db.CreateProject(ctx, ownerPrincipal, "suspension-project", "suspension", "Suspension", "")
	if err != nil {
		t.Fatal(err)
	}
	credential, _, err := db.IssueProjectToken(ctx, shuvPrincipal, project.ID, "shuv-token", "Shuv agent")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.AuthenticateContext(ctx, credential.Token); err != nil {
		t.Fatalf("active member token failed: %v", err)
	}
	if err = db.UpdateMembership(ctx, ownerMembership, shuvMembership.ID, "member", "suspended"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.AuthenticateContext(ctx, credential.Token); err == nil {
		t.Fatal("suspended member's project token remained active")
	}
	if _, err = db.ConsumeAuthToken(ctx, authTokenFromBody(t, messages[0].Body), "Shuv"); err == nil {
		t.Fatal("one-time invite was accepted twice")
	}
}

func TestTokenScopesAreLoadedAndExpiredTokensFail(t *testing.T) {
	ctx := context.Background()
	db, err := OpenWithSecret(filepath.Join(t.TempDir(), "clankspace.db"), "test-installation-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.EnsureBootstrap(ctx, "bootstrap-token", "Workspace", "Owner"); err != nil {
		t.Fatal(err)
	}
	auth, err := db.AuthenticateContext(ctx, "bootstrap-token")
	if err != nil {
		t.Fatal(err)
	}
	if !auth.HasScope("workspace:manage") || auth.TokenID == "" {
		t.Fatalf("auth context = %#v", auth)
	}
	if _, err = db.db.Exec(`UPDATE api_tokens SET expires_at='2000-01-01T00:00:00Z' WHERE id=?`, auth.TokenID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.AuthenticateContext(ctx, "bootstrap-token"); err == nil {
		t.Fatal("expired token was accepted")
	}
}
