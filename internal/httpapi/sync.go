package httpapi

import (
	"net/http"
	"strings"

	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/store"
	"github.com/ShamanicArts/clankspace/internal/syncclient"
)

func (s *Server) adminReplicaJoin(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "workspace:manage") {
		return
	}
	if !s.SyncEnabled {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "synchronization is not enabled on this server"})
		return
	}
	var in struct {
		RemoteURL string `json:"remoteUrl"`
		Code      string `json:"code"`
	}
	if !decode(w, r, &in) {
		return
	}
	userID, err := s.Store.UserIDForPrincipal(r.Context(), principal(r).ID)
	if err != nil {
		userID, err = s.Store.EnsureLocalUserForPrincipal(r.Context(), principal(r))
		if err != nil {
			writeError(w, err)
			return
		}
	}
	workspace, err := syncclient.New().Join(r.Context(), s.Store, in.RemoteURL, in.Code, s.ReplicaName, s.BaseURL, userID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"workspace": workspace})
}

func (s *Server) adminReplicaMirror(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "workspace:manage") {
		return
	}
	if !s.SyncEnabled {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "synchronization is not enabled on this server"})
		return
	}
	var in struct {
		WorkspaceID string `json:"workspaceId"`
		RemoteURL   string `json:"remoteUrl"`
		Code        string `json:"code"`
	}
	if !decode(w, r, &in) {
		return
	}
	workspace, err := syncclient.New().Mirror(r.Context(), s.Store, in.WorkspaceID, in.RemoteURL, in.Code)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"workspace": workspace})
}

func (s *Server) accountMirrorOffer(w http.ResponseWriter, r *http.Request) {
	if !s.SyncEnabled {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "synchronization is not enabled on this server"})
		return
	}
	code, expires, err := s.Store.CreateMirrorOffer(r.Context(), currentUser(r).ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"code": code, "expiresAt": expires, "cloudUrl": s.BaseURL, "notice": "Run clank replica mirror from the self-hosted authority before this code expires."})
}

func (s *Server) adminSyncOnce(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "workspace:manage") {
		return
	}
	if err := syncclient.New().SyncAll(r.Context(), s.Store); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "synchronized"})
}

func (s *Server) adminBundleExport(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "workspace:manage") {
		return
	}
	snapshot, err := s.Store.BuildWorkspaceSnapshot(r.Context(), r.PathValue("workspace"))
	if err != nil {
		writeError(w, err)
		return
	}
	var authority domain.Replica
	for _, replica := range snapshot.Replicas {
		if replica.Role == "authority" {
			authority = replica
			break
		}
	}
	if authority.ID == "" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "workspace authority is missing"})
		return
	}
	if authority.ID != s.Store.LocalReplicaID() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "portable bundles must be exported from the workspace authority"})
		return
	}
	writeJSON(w, http.StatusOK, domain.WorkspaceBundle{SchemaVersion: 1, Authority: authority, Snapshot: snapshot})
}

func (s *Server) adminBundleImport(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "workspace:manage") {
		return
	}
	var bundle domain.WorkspaceBundle
	if !decodeLarge(w, r, &bundle, 32<<20) {
		return
	}
	if bundle.SchemaVersion != 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported workspace bundle"})
		return
	}
	userID, err := s.Store.UserIDForPrincipal(r.Context(), principal(r).ID)
	if err != nil {
		userID, err = s.Store.EnsureLocalUserForPrincipal(r.Context(), principal(r))
		if err != nil {
			writeError(w, err)
			return
		}
	}
	if err = s.Store.ImportWorkspaceSnapshot(r.Context(), bundle.Snapshot, bundle.Authority, userID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"workspace": bundle.Snapshot.Workspace})
}

func (s *Server) accountReplicas(w http.ResponseWriter, r *http.Request) {
	membership, err := s.accountMembership(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := s.Store.ListReplicas(r.Context(), membership.WorkspaceID)
	if err != nil {
		writeError(w, err)
		return
	}
	authority, _ := s.Store.IsWorkspaceAuthority(r.Context(), membership.WorkspaceID)
	writeJSON(w, http.StatusOK, map[string]any{"replicas": items, "isAuthority": authority})
}

func (s *Server) accountReplicaOffer(w http.ResponseWriter, r *http.Request) {
	if !s.SyncEnabled {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "synchronization is not enabled on this server"})
		return
	}
	membership, err := s.accountMembership(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Capabilities []string `json:"capabilities"`
	}
	if !decode(w, r, &in) {
		return
	}
	code, expires, err := s.Store.CreateReplicaOffer(r.Context(), membership, in.Capabilities)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"code": code, "expiresAt": expires, "authorityUrl": s.BaseURL, "notice": "The pairing code is shown once and expires in 15 minutes."})
}

func (s *Server) accountReplicaRevoke(w http.ResponseWriter, r *http.Request) {
	membership, err := s.accountMembership(r)
	if err == nil {
		err = s.Store.RevokeReplica(r.Context(), membership, r.PathValue("replica"))
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (s *Server) syncPairClaim(w http.ResponseWriter, r *http.Request) {
	if !s.SyncEnabled {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "synchronization is not enabled"})
		return
	}
	var in struct {
		Code  string             `json:"code"`
		Claim store.ReplicaClaim `json:"claim"`
	}
	if !decode(w, r, &in) {
		return
	}
	result, err := s.Store.ClaimReplicaOffer(r.Context(), in.Code, in.Claim)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) syncMirrorClaim(w http.ResponseWriter, r *http.Request) {
	if !s.SyncEnabled {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "synchronization is not enabled"})
		return
	}
	var in struct {
		Code      string                   `json:"code"`
		Authority domain.Replica           `json:"authority"`
		Snapshot  domain.WorkspaceSnapshot `json:"snapshot"`
	}
	if !decodeLarge(w, r, &in, 16<<20) {
		return
	}
	result, err := s.Store.ClaimMirrorOffer(r.Context(), in.Code, in.Snapshot, in.Authority)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) replicaAuthorization(r *http.Request, capability string) (domain.Replica, bool) {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	replica, scopes, err := s.Store.AuthenticateReplica(r.Context(), token, r.PathValue("workspace"))
	if err != nil || !contains(scopes, capability) {
		return domain.Replica{}, false
	}
	return replica, true
}

func contains(items []string, desired string) bool {
	for _, item := range items {
		if item == desired || item == "manage" {
			return true
		}
	}
	return false
}

func (s *Server) syncPull(w http.ResponseWriter, r *http.Request) {
	replica, ok := s.replicaAuthorization(r, "pull")
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid replica credential"})
		return
	}
	var in struct {
		Heads []domain.SyncHead `json:"heads"`
		Limit int               `json:"limit"`
	}
	if !decode(w, r, &in) {
		return
	}
	events, more, err := s.Store.EventsAfter(r.Context(), r.PathValue("workspace"), in.Heads, in.Limit)
	if err != nil {
		writeError(w, err)
		return
	}
	heads, err := s.Store.SyncHeads(r.Context(), r.PathValue("workspace"))
	if err != nil {
		writeError(w, err)
		return
	}
	replicas, err := s.Store.ListReplicas(r.Context(), r.PathValue("workspace"))
	if err != nil {
		writeError(w, err)
		return
	}
	_ = s.Store.RecordReplicaSuccess(r.Context(), r.PathValue("workspace"), replica.ID)
	writeJSON(w, http.StatusOK, map[string]any{"events": events, "heads": heads, "replicas": replicas, "more": more})
}

func (s *Server) syncPush(w http.ResponseWriter, r *http.Request) {
	replica, ok := s.replicaAuthorization(r, "push")
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid replica credential"})
		return
	}
	var in struct {
		Events []domain.DomainEvent `json:"events"`
	}
	if !decode(w, r, &in) {
		return
	}
	for _, event := range in.Events {
		if event.OriginReplicaID != replica.ID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "replica may push only its own origin events"})
			return
		}
	}
	imported, err := s.Store.ImportEvents(r.Context(), r.PathValue("workspace"), in.Events)
	if err != nil {
		writeError(w, err)
		return
	}
	heads, _ := s.Store.SyncHeads(r.Context(), r.PathValue("workspace"))
	_ = s.Store.RecordReplicaSuccess(r.Context(), r.PathValue("workspace"), replica.ID)
	writeJSON(w, http.StatusOK, map[string]any{"imported": imported, "heads": heads})
}
