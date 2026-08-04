package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/githubsync"
	"github.com/ShamanicArts/clankspace/internal/store"
)

func (s *Server) setupStart(w http.ResponseWriter, r *http.Request) {
	allowed, err := s.Store.AllowSetupRequest(r.Context(), r.RemoteAddr+"|"+r.UserAgent())
	if err != nil {
		writeError(w, err)
		return
	}
	if !allowed {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many setup requests; try again later"})
		return
	}
	var in struct {
		Challenge     string `json:"challenge"`
		ProjectSlug   string `json:"projectSlug"`
		ProjectName   string `json:"projectName"`
		RepositoryURL string `json:"repositoryUrl"`
		AgentName     string `json:"agentName"`
	}
	if !decode(w, r, &in) {
		return
	}
	request, deviceCode, err := s.Store.CreateSetupRequest(r.Context(), in.Challenge, in.ProjectSlug, in.ProjectName, in.RepositoryURL, in.AgentName)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"deviceCode":      deviceCode,
		"userCode":        request.UserCode,
		"verificationUrl": strings.TrimRight(s.BaseURL, "/") + "/?setup=" + request.UserCode,
		"expiresAt":       request.ExpiresAt,
	})
}

func (s *Server) setupRequest(w http.ResponseWriter, r *http.Request) {
	request, err := s.Store.SetupRequestByUserCode(r.Context(), r.PathValue("code"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"request": request, "notice": domain.AdvisoryNotice})
}

func (s *Server) setupExchange(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DeviceCode string `json:"deviceCode"`
		Verifier   string `json:"verifier"`
	}
	if !decode(w, r, &in) {
		return
	}
	exchange, err := s.Store.ExchangeSetupRequest(r.Context(), in.DeviceCode, in.Verifier)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "pending"})
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "approved", "project": exchange.Project, "token": exchange.Token})
}

func (s *Server) setupApproveToken(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	if p.Kind != "human" || !authentication(r).HasScope("workspace:manage") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "human workspace owner access is required to approve setup"})
		return
	}
	userID, err := s.Store.UserIDForPrincipal(r.Context(), p.ID)
	if errors.Is(err, store.ErrNotFound) {
		userID, err = s.Store.EnsureLocalUserForPrincipal(r.Context(), p)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	s.setupApprove(w, r, userID, p.WorkspaceID)
}

func (s *Server) setupApproveSession(w http.ResponseWriter, r *http.Request) {
	s.setupApprove(w, r, currentUser(r).ID, "")
}

func (s *Server) setupApprove(w http.ResponseWriter, r *http.Request, userID, forcedWorkspaceID string) {
	var in struct {
		UserCode    string `json:"userCode"`
		WorkspaceID string `json:"workspaceId"`
	}
	if !decode(w, r, &in) {
		return
	}
	request, err := s.Store.SetupRequestByUserCode(r.Context(), in.UserCode)
	if err != nil {
		writeError(w, err)
		return
	}
	if request.Approved {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "setup request is already approved"})
		return
	}
	workspaceID := forcedWorkspaceID
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(in.WorkspaceID)
	}
	workspaces, err := s.Store.ListWorkspacesForUser(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}
	if workspaceID == "" && len(workspaces) == 1 {
		workspaceID = workspaces[0].ID
	}
	if workspaceID == "" && len(workspaces) == 0 {
		workspace, _, createErr := s.Store.CreateWorkspaceForUser(r.Context(), userID, request.ProjectSlug+"-workspace", request.ProjectName+" workspace")
		if createErr != nil {
			writeError(w, createErr)
			return
		}
		if s.SyncEnabled {
			if _, createErr = s.Store.EnsureWorkspaceAuthority(r.Context(), workspace.ID); createErr != nil {
				writeError(w, createErr)
				return
			}
		}
		workspaceID = workspace.ID
	}
	if workspaceID == "" {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "choose a workspace", "workspaces": workspaces})
		return
	}
	membership, err := s.Store.Membership(r.Context(), userID, workspaceID)
	if err != nil {
		writeError(w, err)
		return
	}
	p, err := s.Store.PrincipalForMembership(r.Context(), membership)
	if err != nil {
		writeError(w, err)
		return
	}
	project, err := s.Store.GetProject(r.Context(), workspaceID, request.ProjectSlug)
	if errors.Is(err, store.ErrNotFound) {
		project, _, err = s.Core.CreateProject(r.Context(), p, "setup-project-"+request.ID, request.ProjectSlug, request.ProjectName, "Connected by clank setup")
	}
	if err != nil {
		writeError(w, err)
		return
	}
	if request.RepositoryURL != "" {
		repository, parseErr := githubsync.ParseRepository(request.RepositoryURL)
		if parseErr == nil {
			pulls := []domain.ExternalArtifact{}
			syncError := ""
			if synced, syncedPulls, syncErr := s.GitHub.Sync(r.Context(), repository); syncErr == nil {
				repository, pulls = synced, syncedPulls
			} else {
				syncError = syncErr.Error()
			}
			if attached, _, attachErr := s.Store.UpsertRepository(r.Context(), p, project.ID, "setup-repository-"+request.ID, repository); attachErr == nil {
				for index := range pulls {
					pulls[index].RepositoryID = attached.ID
				}
				_ = s.Store.UpdateRepositorySync(r.Context(), attached, "", pulls, syncError)
				attached.Pulls = pulls
				now := time.Now().UTC()
				attached.SyncedAt = &now
			}
		}
	}
	credential, _, err := s.Core.IssueProjectToken(r.Context(), p, project.ID, "setup-token-"+request.ID, request.AgentName)
	if err != nil {
		writeError(w, err)
		return
	}
	if err = s.Store.ApproveSetupRequest(r.Context(), request.UserCode, membership, project, credential.Token); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "approved", "project": project, "notice": domain.AdvisoryNotice})
}
