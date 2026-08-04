package httpapi

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/githubsync"
	"github.com/ShamanicArts/clankspace/internal/service"
	"github.com/ShamanicArts/clankspace/internal/store"
)

//go:embed web/*
var webFiles embed.FS

type Server struct {
	Store       *store.Store
	Core        *service.Service
	GitHub      *githubsync.Client
	Log         *slog.Logger
	BaseURL     string
	AuthMode    string
	SyncEnabled bool
	ReplicaName string
}

type principalKey struct{}
type authContextKey struct{}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.Store.Ping(r.Context()); err != nil {
			writeError(w, err)
			return
		}
		if blockers, err := s.Store.ProjectionBlockerCount(r.Context()); err != nil {
			writeError(w, err)
			return
		} else if blockers > 0 {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "projection-blocked", "blockedEvents": blockers})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/v1/meta", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"authMode": s.AuthMode, "notice": domain.AdvisoryNotice})
	})
	mux.HandleFunc("GET /clankspace-skill.md", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://raw.githubusercontent.com/ShamanicArts/clankspace/main/.agents/skills/clankspace/SKILL.md", http.StatusTemporaryRedirect)
	})
	mux.Handle("POST /api/v1/admin/claim", s.auth(http.HandlerFunc(s.claimBootstrapOwner)))
	mux.Handle("GET /api/v1/admin/users", s.auth(http.HandlerFunc(s.adminUsers)))
	mux.Handle("POST /api/v1/admin/users/{user}/status", s.auth(http.HandlerFunc(s.adminUserStatus)))
	mux.Handle("GET /api/v1/admin/workspaces", s.auth(http.HandlerFunc(s.adminWorkspaces)))
	mux.Handle("POST /api/v1/admin/workspaces", s.auth(http.HandlerFunc(s.adminWorkspaces)))
	mux.Handle("POST /api/v1/admin/workspaces/{workspace}/status", s.auth(http.HandlerFunc(s.adminWorkspaceStatus)))
	mux.HandleFunc("POST /api/v1/auth/magic-link", s.magicLink)
	mux.HandleFunc("POST /api/v1/auth/consume", s.consumeAuth)
	mux.HandleFunc("POST /api/v1/setup/start", s.setupStart)
	mux.HandleFunc("GET /api/v1/setup/requests/{code}", s.setupRequest)
	mux.HandleFunc("POST /api/v1/setup/exchange", s.setupExchange)
	mux.Handle("POST /api/v1/setup/approve", s.auth(http.HandlerFunc(s.setupApproveToken)))
	mux.HandleFunc("GET /api/v1/session-status", s.sessionStatus)
	mux.Handle("GET /api/v1/account", s.session(false, http.HandlerFunc(s.account)))
	mux.Handle("POST /api/v1/auth/logout", s.session(true, http.HandlerFunc(s.logout)))
	mux.Handle("POST /api/v1/account/setup/approve", s.session(true, http.HandlerFunc(s.setupApproveSession)))
	mux.Handle("GET /api/v1/account/workspaces", s.session(false, http.HandlerFunc(s.accountWorkspaces)))
	mux.Handle("POST /api/v1/account/workspaces", s.session(true, http.HandlerFunc(s.accountWorkspaces)))
	mux.Handle("GET /api/v1/account/workspaces/{workspace}/members", s.session(false, http.HandlerFunc(s.accountMembers)))
	mux.Handle("POST /api/v1/account/workspaces/{workspace}/members/{membership}", s.session(true, http.HandlerFunc(s.accountMemberUpdate)))
	mux.Handle("GET /api/v1/account/workspaces/{workspace}/invites", s.session(false, http.HandlerFunc(s.accountInvites)))
	mux.Handle("POST /api/v1/account/workspaces/{workspace}/invites", s.session(true, http.HandlerFunc(s.accountInvites)))
	mux.Handle("POST /api/v1/account/workspaces/{workspace}/invites/{invite}/revoke", s.session(true, http.HandlerFunc(s.accountInviteRevoke)))
	mux.Handle("GET /api/v1/account/workspaces/{workspace}/projects", s.session(false, http.HandlerFunc(s.accountProjects)))
	mux.Handle("POST /api/v1/account/workspaces/{workspace}/projects", s.session(true, http.HandlerFunc(s.accountProjects)))
	mux.Handle("GET /api/v1/account/workspaces/{workspace}/projects/{project}", s.session(false, http.HandlerFunc(s.accountProject)))
	mux.Handle("POST /api/v1/account/workspaces/{workspace}/projects/{project}/notes", s.session(true, http.HandlerFunc(s.accountNote)))
	mux.Handle("POST /api/v1/account/workspaces/{workspace}/projects/{project}/notes/{note}/supersede", s.session(true, http.HandlerFunc(s.accountSupersede)))
	mux.Handle("GET /api/v1/account/workspaces/{workspace}/projects/{project}/tokens", s.session(false, http.HandlerFunc(s.accountTokens)))
	mux.Handle("POST /api/v1/account/workspaces/{workspace}/projects/{project}/tokens", s.session(true, http.HandlerFunc(s.accountTokens)))
	mux.Handle("POST /api/v1/account/workspaces/{workspace}/projects/{project}/tokens/{token}/revoke", s.session(true, http.HandlerFunc(s.accountTokenRevoke)))
	mux.Handle("GET /api/v1/account/workspaces/{workspace}/projects/{project}/export", s.session(false, http.HandlerFunc(s.accountProjectExport)))
	mux.Handle("POST /api/v1/account/workspaces/{workspace}/projects/{project}/repositories", s.session(true, http.HandlerFunc(s.accountRepositoryAttach)))
	mux.Handle("GET /api/v1/account/workspaces/{workspace}/replicas", s.session(false, http.HandlerFunc(s.accountReplicas)))
	mux.Handle("POST /api/v1/account/workspaces/{workspace}/replicas/offers", s.session(true, http.HandlerFunc(s.accountReplicaOffer)))
	mux.Handle("POST /api/v1/account/workspaces/{workspace}/replicas/{replica}/revoke", s.session(true, http.HandlerFunc(s.accountReplicaRevoke)))
	mux.HandleFunc("POST /api/v1/sync/pair/claim", s.syncPairClaim)
	mux.HandleFunc("POST /api/v1/sync/workspaces/{workspace}/pull", s.syncPull)
	mux.HandleFunc("POST /api/v1/sync/workspaces/{workspace}/push", s.syncPush)
	mux.Handle("POST /api/v1/admin/replica/join", s.auth(http.HandlerFunc(s.adminReplicaJoin)))
	mux.Handle("POST /api/v1/admin/sync", s.auth(http.HandlerFunc(s.adminSyncOnce)))
	mux.Handle("POST /api/v1/admin/replica/mirror", s.auth(http.HandlerFunc(s.adminReplicaMirror)))
	mux.Handle("POST /api/v1/account/mirror-offers", s.session(true, http.HandlerFunc(s.accountMirrorOffer)))
	mux.HandleFunc("POST /api/v1/sync/mirror/claim", s.syncMirrorClaim)
	mux.Handle("GET /api/v1/admin/workspaces/{workspace}/bundle", s.auth(http.HandlerFunc(s.adminBundleExport)))
	mux.Handle("POST /api/v1/admin/bundles/import", s.auth(http.HandlerFunc(s.adminBundleImport)))
	mux.Handle("GET /api/v1/whoami", s.auth(http.HandlerFunc(s.whoami)))
	mux.Handle("GET /api/v1/projects", s.auth(http.HandlerFunc(s.projects)))
	mux.Handle("POST /api/v1/projects", s.auth(http.HandlerFunc(s.projects)))
	mux.Handle("GET /api/v1/projects/{project}", s.auth(http.HandlerFunc(s.project)))
	mux.Handle("GET /api/v1/projects/{project}/export", s.auth(http.HandlerFunc(s.exportProject)))
	mux.Handle("POST /api/v1/projects/{project}/tokens", s.auth(http.HandlerFunc(s.projectToken)))
	mux.Handle("GET /api/v1/projects/{project}/notes", s.auth(http.HandlerFunc(s.notes)))
	mux.Handle("POST /api/v1/projects/{project}/notes", s.auth(http.HandlerFunc(s.notes)))
	mux.Handle("POST /api/v1/projects/{project}/notes/{note}/supersede", s.auth(http.HandlerFunc(s.supersede)))
	mux.Handle("GET /api/v1/projects/{project}/trajectories", s.auth(http.HandlerFunc(s.trajectories)))
	mux.Handle("POST /api/v1/projects/{project}/trajectories", s.auth(http.HandlerFunc(s.trajectories)))
	mux.Handle("POST /api/v1/projects/{project}/brief", s.auth(http.HandlerFunc(s.brief)))
	mux.Handle("GET /api/v1/projects/{project}/runs", s.auth(http.HandlerFunc(s.projectRuns)))
	mux.Handle("GET /api/v1/projects/{project}/repositories", s.auth(http.HandlerFunc(s.repositories)))
	mux.Handle("POST /api/v1/projects/{project}/repositories", s.auth(http.HandlerFunc(s.repositories)))
	mux.Handle("POST /api/v1/projects/{project}/repositories/{repo}/refresh", s.auth(http.HandlerFunc(s.refreshRepository)))
	mux.Handle("POST /api/v1/runs", s.auth(http.HandlerFunc(s.runs)))
	mux.Handle("POST /api/v1/runs/{run}/end", s.auth(http.HandlerFunc(s.endRun)))
	assets, _ := fs.Sub(webFiles, "web")
	mux.Handle("GET /", http.FileServer(http.FS(assets)))
	return requestLog(s.Log, mux)
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "bearer token required"})
			return
		}
		auth, err := s.Store.AuthenticateContext(r.Context(), token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, auth)
		ctx = context.WithValue(ctx, principalKey{}, auth.Principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func principal(r *http.Request) domain.Principal {
	return r.Context().Value(principalKey{}).(domain.Principal)
}
func authentication(r *http.Request) domain.AuthContext {
	return r.Context().Value(authContextKey{}).(domain.AuthContext)
}
func requireScope(w http.ResponseWriter, r *http.Request, scope string) bool {
	if authentication(r).HasScope(scope) {
		return true
	}
	writeJSON(w, http.StatusForbidden, map[string]string{"error": "credential does not allow " + scope})
	return false
}
func idempotency(r *http.Request) string { return strings.TrimSpace(r.Header.Get("Idempotency-Key")) }

func (s *Server) whoami(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"principal": principal(r), "scopes": authentication(r).Scopes, "notice": domain.AdvisoryNotice})
}

func (s *Server) projects(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	if r.Method == http.MethodGet {
		projects, err := s.Store.ListProjectsForPrincipal(r.Context(), p)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"projects": projects, "notice": domain.AdvisoryNotice})
		return
	}
	if !requireScope(w, r, "workspace:manage") {
		return
	}
	if s.SyncEnabled {
		authority, err := s.Store.IsWorkspaceAuthority(r.Context(), p.WorkspaceID)
		if err != nil || !authority {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "projects must be created on the workspace authority replica"})
			return
		}
	}
	var in struct{ Slug, Name, Description string }
	if !decode(w, r, &in) {
		return
	}
	project, receipt, err := s.Core.CreateProject(r.Context(), p, idempotency(r), in.Slug, in.Name, in.Description)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"project": project, "receipt": receipt})
}

func (s *Server) project(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	id := r.PathValue("project")
	project, err := s.getProject(r, p, id)
	if err != nil {
		writeError(w, err)
		return
	}
	notes, _ := s.Store.ListNotes(r.Context(), project.ID, 40)
	trajectories, _ := s.Store.ListTrajectories(r.Context(), project.ID, false)
	repos, _ := s.Store.ListRepositories(r.Context(), project.ID)
	writeJSON(w, http.StatusOK, map[string]any{"project": project, "notice": domain.AdvisoryNotice, "notes": notes, "trajectories": trajectories, "repositories": repos})
}

func (s *Server) exportProject(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	project, err := s.getProject(r, p, r.PathValue("project"))
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

func (s *Server) projectToken(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "token:issue") {
		return
	}
	p := principal(r)
	project, err := s.getProject(r, p, r.PathValue("project"))
	if err != nil {
		writeError(w, err)
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

func (s *Server) getProject(r *http.Request, p domain.Principal, id string) (domain.Project, error) {
	project, err := s.Store.GetProject(r.Context(), p.WorkspaceID, id)
	if err != nil {
		return project, err
	}
	ok, err := s.Store.CanAccessProject(r.Context(), p, project.ID)
	if err != nil {
		return project, err
	}
	if !ok {
		return domain.Project{}, store.ErrNotFound
	}
	return project, nil
}

func (s *Server) notes(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	projectID := r.PathValue("project")
	project, err := s.getProject(r, p, projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	if r.Method == http.MethodGet {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		notes, err := s.Store.ListNotes(r.Context(), project.ID, limit)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"notes": notes, "notice": domain.AdvisoryNotice})
		return
	}
	if !requireScope(w, r, "project:write") {
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
	writeJSON(w, http.StatusCreated, map[string]any{"note": note, "receipt": receipt, "notice": domain.AdvisoryNotice})
}

func (s *Server) supersede(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "project:write") {
		return
	}
	p := principal(r)
	project, err := s.getProject(r, p, r.PathValue("project"))
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

func (s *Server) trajectories(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	project, err := s.getProject(r, p, r.PathValue("project"))
	if err != nil {
		writeError(w, err)
		return
	}
	if r.Method == http.MethodGet {
		items, err := s.Store.ListTrajectories(r.Context(), project.ID, r.URL.Query().Get("active") == "true")
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"trajectories": items})
		return
	}
	if !requireScope(w, r, "project:write") {
		return
	}
	var in domain.CreateTrajectoryInput
	if !decode(w, r, &in) {
		return
	}
	item, receipt, err := s.Core.CreateTrajectory(r.Context(), p, project.ID, idempotency(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"trajectory": item, "receipt": receipt})
}

func (s *Server) brief(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	project, err := s.getProject(r, p, r.PathValue("project"))
	if err != nil {
		writeError(w, err)
		return
	}
	var in domain.BriefInput
	if !decode(w, r, &in) {
		return
	}
	b, err := s.Core.Brief(r.Context(), p, project.ID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) runs(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "project:write") {
		return
	}
	var in domain.StartRunInput
	if !decode(w, r, &in) {
		return
	}
	run, receipt, err := s.Core.StartRun(r.Context(), principal(r), idempotency(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"run": run, "receipt": receipt, "notice": domain.AdvisoryNotice})
}

func (s *Server) projectRuns(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	project, err := s.getProject(r, p, r.PathValue("project"))
	if err != nil {
		writeError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	runs, err := s.Store.ListRuns(r.Context(), project.ID, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (s *Server) endRun(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "project:write") {
		return
	}
	var in domain.EndRunInput
	if !decode(w, r, &in) {
		return
	}
	run, receipt, err := s.Store.EndRun(r.Context(), principal(r), idempotency(r), r.PathValue("run"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"run": run, "receipt": receipt})
}

func (s *Server) repositories(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	project, err := s.getProject(r, p, r.PathValue("project"))
	if err != nil {
		writeError(w, err)
		return
	}
	if r.Method == http.MethodGet {
		repos, err := s.Store.ListRepositories(r.Context(), project.ID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"repositories": repos})
		return
	}
	if !requireScope(w, r, "repository:manage") {
		return
	}
	if s.SyncEnabled {
		authority, authorityErr := s.Store.IsWorkspaceAuthority(r.Context(), p.WorkspaceID)
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
	repo, err := githubsync.ParseRepository(in.URL)
	if err != nil {
		writeError(w, err)
		return
	}
	repo, pulls, err := s.GitHub.Sync(r.Context(), repo)
	if err != nil {
		writeError(w, err)
		return
	}
	repo, receipt, err := s.Store.UpsertRepository(r.Context(), p, project.ID, idempotency(r), repo)
	if err != nil {
		writeError(w, err)
		return
	}
	for i := range pulls {
		pulls[i].RepositoryID = repo.ID
	}
	if err = s.Store.UpdateRepositorySync(r.Context(), repo, "", pulls, ""); err != nil {
		writeError(w, err)
		return
	}
	repo.Pulls, repo.SyncedAt = pulls, pointer(time.Now().UTC())
	writeJSON(w, http.StatusCreated, map[string]any{"repository": repo, "receipt": receipt})
}

func (s *Server) refreshRepository(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "repository:manage") {
		return
	}
	p := principal(r)
	project, err := s.getProject(r, p, r.PathValue("project"))
	if err != nil {
		writeError(w, err)
		return
	}
	repos, err := s.Store.ListRepositories(r.Context(), project.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	var repo domain.Repository
	for _, candidate := range repos {
		if candidate.ID == r.PathValue("repo") {
			repo = candidate
			break
		}
	}
	if repo.ID == "" {
		writeError(w, store.ErrNotFound)
		return
	}
	updated, pulls, err := s.GitHub.Sync(r.Context(), repo)
	if err != nil {
		_ = s.Store.UpdateRepositorySync(r.Context(), repo, "", nil, err.Error())
		writeError(w, err)
		return
	}
	for i := range pulls {
		pulls[i].RepositoryID = repo.ID
	}
	if err = s.Store.UpdateRepositorySync(r.Context(), updated, "", pulls, ""); err != nil {
		writeError(w, err)
		return
	}
	updated.Pulls, updated.SyncedAt = pulls, pointer(time.Now().UTC())
	writeJSON(w, http.StatusOK, map[string]any{"repository": updated})
}

func pointer[T any](v T) *T { return &v }

func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	return decodeLarge(w, r, out, 1<<20)
}

func decodeLarge(w http.ResponseWriter, r *http.Request, out any, limit int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, store.ErrNotFound) {
		status = http.StatusNotFound
	}
	if errors.Is(err, store.ErrConflict) || errors.Is(err, store.ErrIdempotencyKeyReuse) {
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func requestLog(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
