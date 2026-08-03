package httpapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/githubsync"
	"github.com/ShamanicArts/clankspace/internal/store"
)

const (
	sessionCookieName = "clank_session"
	csrfCookieName    = "clank_csrf"
)

type userKey struct{}

func currentUser(r *http.Request) domain.User {
	return r.Context().Value(userKey{}).(domain.User)
}

func (s *Server) cookieSecure() bool {
	base, err := url.Parse(s.BaseURL)
	return err == nil && base.Scheme == "https"
}

func (s *Server) session(requireCSRF bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie(sessionCookieName)
		if err != nil || strings.TrimSpace(sessionCookie.Value) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "sign in required"})
			return
		}
		csrf := ""
		if requireCSRF {
			if !s.sameOrigin(r) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "same-origin request required"})
				return
			}
			csrf = r.Header.Get("X-CSRF-Token")
		}
		user, err := s.Store.AuthenticateSession(r.Context(), sessionCookie.Value, csrf, requireCSRF)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "session expired"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	})
}

func (s *Server) sameOrigin(r *http.Request) bool {
	expected, err := url.Parse(s.BaseURL)
	if err != nil {
		return false
	}
	origin, err := url.Parse(r.Header.Get("Origin"))
	if err != nil || origin.Scheme == "" || origin.Host == "" {
		return false
	}
	return strings.EqualFold(origin.Scheme, expected.Scheme) && strings.EqualFold(origin.Host, expected.Host)
}

func (s *Server) claimBootstrapOwner(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "workspace:manage") {
		return
	}
	var in struct {
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
	}
	if !decode(w, r, &in) {
		return
	}
	user, membership, err := s.Store.ClaimBootstrapOwner(r.Context(), principal(r).ID, in.Email, in.DisplayName)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": user, "membership": membership})
}

func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "workspace:manage") {
		return
	}
	items, err := s.Store.ListUsers(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": items})
}

func (s *Server) adminUserStatus(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "workspace:manage") {
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.Store.SetUserStatus(r.Context(), r.PathValue("user"), in.Status); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": in.Status})
}

func (s *Server) adminWorkspaces(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "workspace:manage") {
		return
	}
	if r.Method == http.MethodGet {
		items, err := s.Store.ListAllWorkspaces(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"workspaces": items})
		return
	}
	var in struct{ Slug, Name string }
	if !decode(w, r, &in) {
		return
	}
	userID, err := s.Store.UserIDForPrincipal(r.Context(), principal(r).ID)
	if err != nil {
		userID, err = s.Store.EnsureLocalUserForPrincipal(r.Context(), principal(r))
	}
	if err != nil {
		writeError(w, err)
		return
	}
	workspace, membership, err := s.Store.CreateWorkspaceForUser(r.Context(), userID, in.Slug, in.Name)
	if err == nil && s.SyncEnabled {
		_, err = s.Store.EnsureWorkspaceAuthority(r.Context(), workspace.ID)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"workspace": workspace, "membership": membership})
}

func (s *Server) adminWorkspaceStatus(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "workspace:manage") {
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.Store.SetWorkspaceStatus(r.Context(), r.PathValue("workspace"), in.Status); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": in.Status})
}

func (s *Server) magicLink(w http.ResponseWriter, r *http.Request) {
	if s.AuthMode == "bootstrap" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "email login is not enabled"})
		return
	}
	var in struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &in) {
		return
	}
	source := r.RemoteAddr
	if host, _, splitErr := net.SplitHostPort(r.RemoteAddr); splitErr == nil {
		source = host
	}
	fingerprint := source + "|" + r.UserAgent()
	if err := s.Store.RequestMagicLink(r.Context(), in.Email, fingerprint, s.BaseURL); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "If that address can sign in, a one-time link has been sent."})
}

func (s *Server) consumeAuth(w http.ResponseWriter, r *http.Request) {
	if s.AuthMode == "bootstrap" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "email login is not enabled"})
		return
	}
	var in struct {
		Token       string `json:"token"`
		DisplayName string `json:"displayName"`
	}
	if !decode(w, r, &in) {
		return
	}
	result, err := s.Store.ConsumeAuthToken(r.Context(), in.Token, in.DisplayName)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "link is invalid or expired"})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: result.SessionToken, Path: "/", Expires: result.Session.ExpiresAt, MaxAge: 30 * 24 * 60 * 60, HttpOnly: true, Secure: s.cookieSecure(), SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: result.CSRFToken, Path: "/", Expires: result.Session.ExpiresAt, MaxAge: 30 * 24 * 60 * 60, HttpOnly: false, Secure: s.cookieSecure(), SameSite: http.SameSiteLaxMode})
	writeJSON(w, http.StatusOK, map[string]any{"user": result.User, "expiresAt": result.Session.ExpiresAt})
}

func (s *Server) sessionStatus(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
		return
	}
	user, err := s.Store.AuthenticateSession(r.Context(), cookie.Value, "", false)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
		return
	}
	workspaces, err := s.Store.ListWorkspacesForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "user": user, "workspaces": workspaces, "notice": domain.AdvisoryNotice})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		_ = s.Store.RevokeSession(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.cookieSecure(), SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: "", Path: "/", MaxAge: -1, Secure: s.cookieSecure(), SameSite: http.SameSiteLaxMode})
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed out"})
}

func (s *Server) account(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	workspaces, err := s.Store.ListWorkspacesForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "workspaces": workspaces, "notice": domain.AdvisoryNotice})
}

func (s *Server) accountWorkspaces(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if r.Method == http.MethodGet {
		items, err := s.Store.ListWorkspacesForUser(r.Context(), user.ID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"workspaces": items})
		return
	}
	var in struct{ Slug, Name string }
	if !decode(w, r, &in) {
		return
	}
	workspace, membership, err := s.Store.CreateWorkspaceForUser(r.Context(), user.ID, in.Slug, in.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	if s.SyncEnabled {
		if _, err = s.Store.EnsureWorkspaceAuthority(r.Context(), workspace.ID); err != nil {
			writeError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"workspace": workspace, "membership": membership})
}

func (s *Server) accountMembership(r *http.Request) (domain.Membership, error) {
	return s.Store.Membership(r.Context(), currentUser(r).ID, r.PathValue("workspace"))
}

func (s *Server) accountMembers(w http.ResponseWriter, r *http.Request) {
	membership, err := s.accountMembership(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := s.Store.ListMembers(r.Context(), membership)
	if err != nil {
		writeError(w, err)
		return
	}
	canShare := true
	if s.SyncEnabled {
		canShare, _ = s.Store.CanShareHumans(r.Context(), membership.WorkspaceID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": items, "canShareHumans": canShare})
}

func (s *Server) accountMemberUpdate(w http.ResponseWriter, r *http.Request) {
	membership, err := s.accountMembership(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if s.SyncEnabled {
		allowed, shareErr := s.Store.CanShareHumans(r.Context(), membership.WorkspaceID)
		if shareErr != nil || !allowed {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "this mirror may not change human access"})
			return
		}
	}
	var in struct{ Role, Status string }
	if !decode(w, r, &in) {
		return
	}
	if err = s.Store.UpdateMembership(r.Context(), membership, r.PathValue("membership"), in.Role, in.Status); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) accountInvites(w http.ResponseWriter, r *http.Request) {
	membership, err := s.accountMembership(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if r.Method == http.MethodGet {
		items, listErr := s.Store.ListInvites(r.Context(), membership)
		if listErr != nil {
			writeError(w, listErr)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"invites": items})
		return
	}
	if s.SyncEnabled {
		allowed, shareErr := s.Store.CanShareHumans(r.Context(), membership.WorkspaceID)
		if shareErr != nil || !allowed {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "this mirror may not invite additional humans"})
			return
		}
	}
	var in struct{ Email, Role string }
	if !decode(w, r, &in) {
		return
	}
	invite, err := s.Store.CreateWorkspaceInvite(r.Context(), membership, in.Email, in.Role, s.BaseURL)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"invite": invite})
}

func (s *Server) accountInviteRevoke(w http.ResponseWriter, r *http.Request) {
	membership, err := s.accountMembership(r)
	if err == nil {
		err = s.Store.RevokeInvite(r.Context(), membership, r.PathValue("invite"))
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (s *Server) accountProjects(w http.ResponseWriter, r *http.Request) {
	membership, err := s.accountMembership(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if r.Method == http.MethodGet {
		items, listErr := s.Store.ListProjects(r.Context(), membership.WorkspaceID)
		if listErr != nil {
			writeError(w, listErr)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"projects": items})
		return
	}
	var in struct{ Slug, Name, Description string }
	if !decode(w, r, &in) {
		return
	}
	if s.SyncEnabled {
		authority, authorityErr := s.Store.IsWorkspaceAuthority(r.Context(), membership.WorkspaceID)
		if authorityErr != nil || !authority {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "projects must be created on the workspace authority replica"})
			return
		}
	}
	p, err := s.Store.PrincipalForMembership(r.Context(), membership)
	if err != nil {
		writeError(w, err)
		return
	}
	project, receipt, err := s.Core.CreateProject(r.Context(), p, idempotency(r), in.Slug, in.Name, in.Description)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"project": project, "receipt": receipt})
}

func (s *Server) accountProjectContext(r *http.Request) (domain.Membership, domain.Principal, domain.Project, error) {
	membership, err := s.accountMembership(r)
	if err != nil {
		return membership, domain.Principal{}, domain.Project{}, err
	}
	p, err := s.Store.PrincipalForMembership(r.Context(), membership)
	if err != nil {
		return membership, p, domain.Project{}, err
	}
	project, err := s.Store.GetProject(r.Context(), membership.WorkspaceID, r.PathValue("project"))
	return membership, p, project, err
}

func (s *Server) accountProject(w http.ResponseWriter, r *http.Request) {
	_, _, project, err := s.accountProjectContext(r)
	if err != nil {
		writeError(w, err)
		return
	}
	notes, _ := s.Store.ListNotes(r.Context(), project.ID, 100)
	trajectories, _ := s.Store.ListTrajectories(r.Context(), project.ID, false)
	repositories, _ := s.Store.ListRepositories(r.Context(), project.ID)
	writeJSON(w, http.StatusOK, map[string]any{"project": project, "notes": notes, "trajectories": trajectories, "repositories": repositories, "notice": domain.AdvisoryNotice})
}

func (s *Server) accountNote(w http.ResponseWriter, r *http.Request) {
	_, p, project, err := s.accountProjectContext(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in domain.CreateNoteInput
	if !decode(w, r, &in) {
		return
	}
	note, receipt, err := s.Core.CreateNote(r.Context(), p, project.ID, idempotency(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"note": note, "receipt": receipt})
}

func (s *Server) accountSupersede(w http.ResponseWriter, r *http.Request) {
	_, p, project, err := s.accountProjectContext(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in domain.SupersedeNoteInput
	if !decode(w, r, &in) {
		return
	}
	note, receipt, err := s.Core.SupersedeNote(r.Context(), p, project.ID, idempotency(r), r.PathValue("note"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"note": note, "receipt": receipt})
}

func (s *Server) accountTokens(w http.ResponseWriter, r *http.Request) {
	membership, p, project, err := s.accountProjectContext(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if r.Method == http.MethodGet {
		items, listErr := s.Store.ListProjectTokens(r.Context(), membership, project.ID)
		if listErr != nil {
			writeError(w, listErr)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"tokens": items})
		return
	}
	var in struct {
		DisplayName string `json:"displayName"`
	}
	if !decode(w, r, &in) {
		return
	}
	credential, receipt, err := s.Core.IssueProjectToken(r.Context(), p, project.ID, idempotency(r), in.DisplayName)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"principal": credential.Principal, "token": credential.Token, "notice": credential.Notice, "receipt": receipt})
}

func (s *Server) accountTokenRevoke(w http.ResponseWriter, r *http.Request) {
	membership, _, project, err := s.accountProjectContext(r)
	if err == nil {
		err = s.Store.RevokeProjectToken(r.Context(), membership, project.ID, r.PathValue("token"))
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (s *Server) accountProjectExport(w http.ResponseWriter, r *http.Request) {
	_, _, project, err := s.accountProjectContext(r)
	if err != nil {
		writeError(w, err)
		return
	}
	notes, err := s.Store.ListNotes(r.Context(), project.ID, -1)
	if err != nil {
		writeError(w, err)
		return
	}
	trajectories, err := s.Store.ListTrajectories(r.Context(), project.ID, false)
	if err != nil {
		writeError(w, err)
		return
	}
	repositories, err := s.Store.ListRepositories(r.Context(), project.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+project.Slug+`.clankspace.json"`)
	writeJSON(w, http.StatusOK, map[string]any{"schemaVersion": 1, "exportedAt": time.Now().UTC(), "notice": domain.AdvisoryNotice, "project": project, "notes": notes, "trajectories": trajectories, "repositories": repositories})
}

func (s *Server) accountRepositoryAttach(w http.ResponseWriter, r *http.Request) {
	membership, p, project, err := s.accountProjectContext(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if s.SyncEnabled {
		authority, authorityErr := s.Store.IsWorkspaceAuthority(r.Context(), membership.WorkspaceID)
		if authorityErr != nil || !authority {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "repository links must be changed on the workspace authority replica"})
			return
		}
	}
	var in struct {
		URL string `json:"url"`
	}
	if !decode(w, r, &in) {
		return
	}
	repository, err := githubsync.ParseRepository(in.URL)
	if err != nil {
		writeError(w, err)
		return
	}
	repository, pulls, err := s.GitHub.Sync(r.Context(), repository)
	if err != nil {
		writeError(w, err)
		return
	}
	repository, receipt, err := s.Store.UpsertRepository(r.Context(), p, project.ID, idempotency(r), repository)
	if err != nil {
		writeError(w, err)
		return
	}
	for index := range pulls {
		pulls[index].RepositoryID = repository.ID
	}
	if err = s.Store.UpdateRepositorySync(r.Context(), repository, "", pulls, ""); err != nil {
		writeError(w, err)
		return
	}
	repository.Pulls = pulls
	now := time.Now().UTC()
	repository.SyncedAt = &now
	writeJSON(w, http.StatusCreated, map[string]any{"repository": repository, "receipt": receipt})
}

func isNotFound(err error) bool { return errors.Is(err, store.ErrNotFound) }
