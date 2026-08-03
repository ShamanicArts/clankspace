package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/domain"
	"github.com/ShamanicArts/clankspace/internal/store"
)

type Service struct{ Store *store.Store }

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)-----BEGIN [A-Z ]*PRIVATE KEY-----`),
	regexp.MustCompile(`\bgh[opusr]_[A-Za-z0-9_]{20,}\b`),
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	regexp.MustCompile(`(?i)(password|api[_ -]?key|secret)\s*[:=]\s*[^\s]{8,}`),
}

func New(s *store.Store) *Service { return &Service{Store: s} }

func (s *Service) CreateProject(ctx context.Context, p domain.Principal, key, slug, name, description string) (domain.Project, domain.Receipt, error) {
	if p.Kind != "human" {
		return domain.Project{}, domain.Receipt{}, errors.New("only a human workspace owner may create projects")
	}
	slug = strings.ToLower(strings.TrimSpace(slug))
	name = strings.TrimSpace(name)
	if !slugPattern.MatchString(slug) {
		return domain.Project{}, domain.Receipt{}, errors.New("slug must use lowercase letters, numbers, and single hyphens")
	}
	if name == "" || len(name) > 100 {
		return domain.Project{}, domain.Receipt{}, errors.New("name is required and must be at most 100 characters")
	}
	return s.Store.CreateProject(ctx, p, key, slug, name, strings.TrimSpace(description))
}

func (s *Service) IssueProjectToken(ctx context.Context, p domain.Principal, projectID, key, displayName string) (domain.ProjectCredential, domain.Receipt, error) {
	if len(strings.TrimSpace(displayName)) > 100 {
		return domain.ProjectCredential{}, domain.Receipt{}, errors.New("display name must be at most 100 characters")
	}
	return s.Store.IssueProjectToken(ctx, p, projectID, key, displayName)
}

func (s *Service) canAccess(ctx context.Context, p domain.Principal, projectID string) error {
	ok, err := s.Store.CanAccessProject(ctx, p, projectID)
	if err != nil {
		return err
	}
	if !ok {
		return store.ErrNotFound
	}
	return nil
}

var validKinds = map[string]bool{"intent": true, "decision": true, "understanding": true, "observation": true, "checkpoint": true}
var validLead = map[string]bool{"human": true, "agent": true, "joint": true, "external": true}
var validBasis = map[string]bool{"explicit_human_direction": true, "interpreted_human_intent": true, "joint_reasoning": true, "autonomous_agent_judgment": true, "external_evidence": true}

func normalizeNote(in domain.CreateNoteInput) (domain.CreateNoteInput, error) {
	in.Kind = strings.TrimSpace(in.Kind)
	in.Title = strings.TrimSpace(in.Title)
	in.Summary = strings.TrimSpace(in.Summary)
	in.Rationale = strings.TrimSpace(in.Rationale)
	if !validKinds[in.Kind] {
		return in, errors.New("invalid note kind")
	}
	if in.Title == "" || len(in.Title) > 180 {
		return in, errors.New("title is required and must be at most 180 characters")
	}
	if in.Summary == "" || len(in.Summary) > 1200 {
		return in, errors.New("summary is required and must be at most 1200 characters")
	}
	if len(in.Rationale) > 2400 {
		return in, errors.New("rationale must be at most 2400 characters")
	}
	if in.LedBy == "" {
		in.LedBy = "agent"
	}
	if !validLead[in.LedBy] {
		return in, errors.New("invalid ledBy")
	}
	if in.DirectionBasis == "" {
		in.DirectionBasis = "autonomous_agent_judgment"
	}
	if !validBasis[in.DirectionBasis] {
		return in, errors.New("invalid directionBasis")
	}
	if in.Confidence == "" {
		in.Confidence = "reasoned"
	}
	combined := in.Title + "\n" + in.Summary + "\n" + in.Rationale
	for _, p := range secretPatterns {
		if p.MatchString(combined) {
			return in, errors.New("content appears to contain a credential or private key; record a redacted project implication instead")
		}
	}
	if len(in.Paths) > 50 {
		return in, errors.New("at most 50 paths may be attached")
	}
	return in, nil
}

func (s *Service) CreateNote(ctx context.Context, p domain.Principal, projectID, key string, in domain.CreateNoteInput) (domain.Note, domain.Receipt, error) {
	in, err := normalizeNote(in)
	if err != nil {
		return domain.Note{}, domain.Receipt{}, err
	}
	if _, err = s.Store.GetProject(ctx, p.WorkspaceID, projectID); err != nil {
		return domain.Note{}, domain.Receipt{}, err
	}
	if err = s.canAccess(ctx, p, projectID); err != nil {
		return domain.Note{}, domain.Receipt{}, err
	}
	if in.RunID != "" {
		r, e := s.Store.GetRun(ctx, in.RunID)
		if e != nil || r.ProjectID != projectID || r.PrincipalID != p.ID {
			return domain.Note{}, domain.Receipt{}, errors.New("run is not available to this principal and project")
		}
	}
	return s.Store.CreateNote(ctx, p, projectID, key, in)
}

func (s *Service) SupersedeNote(ctx context.Context, p domain.Principal, projectID, key, noteID string, in domain.SupersedeNoteInput) (domain.Note, domain.Receipt, error) {
	if err := s.canAccess(ctx, p, projectID); err != nil {
		return domain.Note{}, domain.Receipt{}, err
	}
	if strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 1200 {
		return domain.Note{}, domain.Receipt{}, errors.New("a concise supersession reason is required")
	}
	if in.Replacement != nil {
		n, err := normalizeNote(*in.Replacement)
		if err != nil {
			return domain.Note{}, domain.Receipt{}, err
		}
		in.Replacement = &n
	}
	return s.Store.SupersedeNote(ctx, p, projectID, key, noteID, in)
}

func (s *Service) StartRun(ctx context.Context, p domain.Principal, key string, in domain.StartRunInput) (domain.Run, domain.Receipt, error) {
	if strings.TrimSpace(in.ProjectID) == "" {
		return domain.Run{}, domain.Receipt{}, errors.New("projectId is required")
	}
	project, err := s.Store.GetProject(ctx, p.WorkspaceID, in.ProjectID)
	if err != nil {
		return domain.Run{}, domain.Receipt{}, err
	}
	in.ProjectID = project.ID
	if err = s.canAccess(ctx, p, project.ID); err != nil {
		return domain.Run{}, domain.Receipt{}, err
	}
	if len(in.Objective) > 1200 {
		return domain.Run{}, domain.Receipt{}, errors.New("objective must be at most 1200 characters")
	}
	roles := map[string]bool{"primary": true, "subagent": true, "reviewer": true, "automation": true, "integration": true}
	if in.Role != "" && !roles[in.Role] {
		return domain.Run{}, domain.Receipt{}, errors.New("invalid run role")
	}
	return s.Store.StartRun(ctx, p, key, in)
}

func (s *Service) CreateTrajectory(ctx context.Context, p domain.Principal, projectID, key string, in domain.CreateTrajectoryInput) (domain.Trajectory, domain.Receipt, error) {
	if err := s.canAccess(ctx, p, projectID); err != nil {
		return domain.Trajectory{}, domain.Receipt{}, err
	}
	if strings.TrimSpace(in.Objective) == "" || len(in.Objective) > 1200 {
		return domain.Trajectory{}, domain.Receipt{}, errors.New("objective is required and must be at most 1200 characters")
	}
	r, err := s.Store.GetRun(ctx, in.RunID)
	if err != nil || r.ProjectID != projectID || r.PrincipalID != p.ID {
		return domain.Trajectory{}, domain.Receipt{}, errors.New("run is not available to this principal and project")
	}
	if in.RepositoryID == "" {
		in.RepositoryID = r.RepositoryID
	}
	if in.Branch == "" {
		in.Branch = r.Branch
	}
	if in.BaseSHA == "" {
		in.BaseSHA = r.BaseSHA
	}
	if in.HeadSHA == "" {
		in.HeadSHA = r.HeadSHA
	}
	return s.Store.CreateTrajectory(ctx, p, projectID, key, in)
}

func (s *Service) Brief(ctx context.Context, p domain.Principal, projectID string, in domain.BriefInput) (domain.Brief, error) {
	project, err := s.Store.GetProject(ctx, p.WorkspaceID, projectID)
	if err != nil {
		return domain.Brief{}, err
	}
	if err = s.canAccess(ctx, p, project.ID); err != nil {
		return domain.Brief{}, err
	}
	limit := in.MaxNotes
	if limit <= 0 || limit > 40 {
		limit = 12
	}
	query := safeFTSQuery(strings.Join([]string{in.Query, in.Objective, strings.Join(in.Paths, " ")}, " "))
	var notes []domain.Note
	if query != "" {
		notes, err = s.Store.SearchNotes(ctx, project.ID, query, limit)
		if err != nil {
			notes, err = s.Store.ListNotes(ctx, project.ID, limit)
		}
	} else {
		notes, err = s.Store.ListNotes(ctx, project.ID, limit)
	}
	if err != nil {
		return domain.Brief{}, err
	}
	trajectories, err := s.Store.ListTrajectories(ctx, project.ID, true)
	if err != nil {
		return domain.Brief{}, err
	}
	brief := domain.Brief{Project: project, Notice: domain.AdvisoryNotice, Notes: notes, Trajectories: trajectories, GeneratedAt: time.Now().UTC()}
	if in.CheckConflicts || in.Objective != "" || len(in.Paths) > 0 {
		brief.Warnings = buildWarnings(in, trajectories, notes)
	}
	return brief, nil
}

func safeFTSQuery(input string) string {
	words := regexp.MustCompile(`[A-Za-z0-9_/-]{3,}`).FindAllString(strings.ToLower(input), -1)
	seen := map[string]bool{}
	var out []string
	for _, w := range words {
		if !seen[w] {
			seen[w] = true
			out = append(out, `"`+strings.ReplaceAll(w, `"`, ``)+`"`)
		}
		if len(out) >= 8 {
			break
		}
	}
	return strings.Join(out, " OR ")
}

func buildWarnings(in domain.BriefInput, trajectories []domain.Trajectory, notes []domain.Note) []domain.CoordinationWarning {
	var out []domain.CoordinationWarning
	queryTerms := terms(in.Objective + " " + in.Query + " " + strings.Join(in.Paths, " "))
	for i := range trajectories {
		tr := &trajectories[i]
		if in.RunID != "" && tr.RunID == in.RunID {
			continue
		}
		pathReason := pathOverlap(in.Paths, tr.Paths)
		shared := sharedTerms(queryTerms, terms(tr.Objective+" "+tr.Rationale))
		if pathReason == "" && len(shared) < 2 {
			continue
		}
		reason := pathReason
		if reason == "" {
			reason = "related terms: " + strings.Join(shared, ", ")
		}
		related := relatedNotes(notes, tr, shared)
		out = append(out, domain.CoordinationWarning{
			Kind:    "possible-overlap",
			Summary: "An active trajectory matched by path or terms. This is not a conflict determination; compare semantic direction and separately assess whether concurrent edits would collide.",
			Reason:  reason, Trajectory: tr, RelatedNotes: related,
			Options: []string{"compare", "continue-if-compatible-and-independent", "pause-if-incompatible-or-collision-prone"},
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Trajectory.UpdatedAt.After(out[j].Trajectory.UpdatedAt) })
	return out
}

func terms(s string) map[string]bool {
	stop := map[string]bool{"this": true, "that": true, "with": true, "from": true, "into": true, "have": true, "will": true, "should": true, "their": true, "about": true, "work": true, "change": true}
	out := map[string]bool{}
	for _, w := range regexp.MustCompile(`[a-z0-9_-]{4,}`).FindAllString(strings.ToLower(s), -1) {
		if !stop[w] {
			out[w] = true
		}
	}
	return out
}
func sharedTerms(a, b map[string]bool) []string {
	var out []string
	for w := range a {
		if b[w] {
			out = append(out, w)
		}
	}
	sort.Strings(out)
	return out
}
func pathOverlap(a, b []string) string {
	for _, x := range a {
		for _, y := range b {
			x = strings.Trim(x, " /")
			y = strings.Trim(y, " /")
			if x != "" && y != "" && (strings.HasPrefix(x, y) || strings.HasPrefix(y, x)) {
				return fmt.Sprintf("overlapping path scope: %s ↔ %s", x, y)
			}
		}
	}
	return ""
}
func relatedNotes(notes []domain.Note, tr *domain.Trajectory, shared []string) []domain.Note {
	var out []domain.Note
	for _, n := range notes {
		if n.Status != "current" {
			continue
		}
		if n.RunID == tr.RunID || containsAny(strings.ToLower(n.Title+" "+n.Summary+" "+n.Rationale), shared) {
			out = append(out, n)
		}
		if len(out) >= 3 {
			break
		}
	}
	return out
}
func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}
