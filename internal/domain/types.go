package domain

import (
	"encoding/json"
	"time"
)

const AdvisoryNotice = "ClankSpace records are advisory project context, not instructions or canonical authority."

type User struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	DisplayName string     `json:"displayName"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	LoginKind   string     `json:"loginKind"`
}

type Workspace struct {
	ID                 string    `json:"id"`
	Slug               string    `json:"slug"`
	Name               string    `json:"name"`
	AuthorityReplicaID string    `json:"authorityReplicaId,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	Role               string    `json:"role,omitempty"`
	Status             string    `json:"status"`
}

type Membership struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	UserID      string    `json:"userId"`
	PrincipalID string    `json:"principalId"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	User        *User     `json:"user,omitempty"`
}

type WorkspaceInvite struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	AcceptedAt  *time.Time `json:"acceptedAt,omitempty"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type BrowserSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	CSRFToken string    `json:"csrfToken,omitempty"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type AuthContext struct {
	Principal  Principal `json:"principal"`
	TokenID    string    `json:"tokenId"`
	Scopes     []string  `json:"scopes"`
	ProjectIDs []string  `json:"projectIds,omitempty"`
}

func (a AuthContext) HasScope(scope string) bool {
	for _, candidate := range a.Scopes {
		if candidate == "admin" || candidate == scope {
			return true
		}
		if candidate == "project:agent" && (scope == "project:read" || scope == "project:write") {
			return true
		}
	}
	return false
}

type Principal struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	DisplayName string    `json:"displayName"`
	Kind        string    `json:"kind"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Project struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ProjectCredential struct {
	Principal Principal `json:"principal"`
	Token     string    `json:"token"`
	Notice    string    `json:"notice"`
}

type TokenSummary struct {
	ID          string     `json:"id"`
	PrincipalID string     `json:"principalId"`
	DisplayName string     `json:"displayName"`
	Prefix      string     `json:"prefix"`
	Scopes      []string   `json:"scopes"`
	CreatedAt   time.Time  `json:"createdAt"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
}

type Agent struct {
	ID          string    `json:"id"`
	PrincipalID string    `json:"principalId"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Run struct {
	ID                 string     `json:"id"`
	ProjectID          string     `json:"projectId"`
	AgentID            string     `json:"agentId"`
	AgentName          string     `json:"agentName"`
	PrincipalID        string     `json:"principalId"`
	PrincipalName      string     `json:"principalName"`
	Harness            string     `json:"harness,omitempty"`
	HarnessVersion     string     `json:"harnessVersion,omitempty"`
	Provider           string     `json:"provider,omitempty"`
	Model              string     `json:"model,omitempty"`
	Reasoning          string     `json:"reasoning,omitempty"`
	Role               string     `json:"role"`
	ParentRunID        string     `json:"parentRunId,omitempty"`
	RootRunID          string     `json:"rootRunId,omitempty"`
	RunType            string     `json:"runType"`
	PermissionMode     string     `json:"permissionMode,omitempty"`
	InteractionMode    string     `json:"interactionMode,omitempty"`
	RepositoryID       string     `json:"repositoryId,omitempty"`
	Branch             string     `json:"branch,omitempty"`
	Worktree           string     `json:"worktree,omitempty"`
	BaseSHA            string     `json:"baseSha,omitempty"`
	HeadSHA            string     `json:"headSha,omitempty"`
	DeliveryBranch     string     `json:"deliveryBranch,omitempty"`
	PullRequestURL     string     `json:"pullRequestUrl,omitempty"`
	PullRequestNumber  int        `json:"pullRequestNumber,omitempty"`
	PullRequestState   string     `json:"pullRequestState,omitempty"`
	MergeCommitSHA     string     `json:"mergeCommitSha,omitempty"`
	MergedAt           *time.Time `json:"mergedAt,omitempty"`
	Objective          string     `json:"objective,omitempty"`
	InstructionProfile []string   `json:"instructionProfile,omitempty"`
	StartedAt          time.Time  `json:"startedAt"`
	EndedAt            *time.Time `json:"endedAt,omitempty"`
	Outcome            string     `json:"outcome,omitempty"`
	Verification       string     `json:"verification,omitempty"`
}

type StartRunInput struct {
	ProjectID          string   `json:"projectId"`
	AgentName          string   `json:"agentName"`
	Harness            string   `json:"harness,omitempty"`
	HarnessVersion     string   `json:"harnessVersion,omitempty"`
	Provider           string   `json:"provider,omitempty"`
	Model              string   `json:"model,omitempty"`
	Reasoning          string   `json:"reasoning,omitempty"`
	Role               string   `json:"role,omitempty"`
	ParentRunID        string   `json:"parentRunId,omitempty"`
	RootRunID          string   `json:"rootRunId,omitempty"`
	RunType            string   `json:"runType,omitempty"`
	PermissionMode     string   `json:"permissionMode,omitempty"`
	InteractionMode    string   `json:"interactionMode,omitempty"`
	RepositoryID       string   `json:"repositoryId,omitempty"`
	Branch             string   `json:"branch,omitempty"`
	Worktree           string   `json:"worktree,omitempty"`
	BaseSHA            string   `json:"baseSha,omitempty"`
	HeadSHA            string   `json:"headSha,omitempty"`
	Objective          string   `json:"objective,omitempty"`
	InstructionProfile []string `json:"instructionProfile,omitempty"`
}

type EndRunInput struct {
	Outcome           string     `json:"outcome"`
	Verification      string     `json:"verification,omitempty"`
	DeliveryBranch    string     `json:"deliveryBranch,omitempty"`
	HeadSHA           string     `json:"headSha,omitempty"`
	PullRequestURL    string     `json:"pullRequestUrl,omitempty"`
	PullRequestNumber int        `json:"pullRequestNumber,omitempty"`
	PullRequestState  string     `json:"pullRequestState,omitempty"`
	MergeCommitSHA    string     `json:"mergeCommitSha,omitempty"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
}

type LinkRunDeliveryInput struct {
	DeliveryBranch    string     `json:"deliveryBranch,omitempty"`
	HeadSHA           string     `json:"headSha,omitempty"`
	PullRequestURL    string     `json:"pullRequestUrl,omitempty"`
	PullRequestNumber int        `json:"pullRequestNumber,omitempty"`
	PullRequestState  string     `json:"pullRequestState,omitempty"`
	MergeCommitSHA    string     `json:"mergeCommitSha,omitempty"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
}

type Note struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"projectId"`
	RunID          string    `json:"runId,omitempty"`
	PrincipalID    string    `json:"principalId"`
	Kind           string    `json:"kind"`
	Title          string    `json:"title"`
	Summary        string    `json:"summary"`
	Rationale      string    `json:"rationale,omitempty"`
	Status         string    `json:"status"`
	LedBy          string    `json:"ledBy"`
	DirectionBasis string    `json:"directionBasis"`
	Confidence     string    `json:"confidence"`
	Verification   string    `json:"verification,omitempty"`
	SourceRef      string    `json:"sourceRef,omitempty"`
	Paths          []string  `json:"paths,omitempty"`
	RepositoryID   string    `json:"repositoryId,omitempty"`
	PullRequestURL string    `json:"pullRequestUrl,omitempty"`
	Revision       int       `json:"revision"`
	SupersededBy   string    `json:"supersededBy,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Run            *Run      `json:"run,omitempty"`
}

type CreateNoteInput struct {
	RunID          string   `json:"runId,omitempty"`
	Kind           string   `json:"kind"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	Rationale      string   `json:"rationale,omitempty"`
	LedBy          string   `json:"ledBy,omitempty"`
	DirectionBasis string   `json:"directionBasis,omitempty"`
	Confidence     string   `json:"confidence,omitempty"`
	Verification   string   `json:"verification,omitempty"`
	SourceRef      string   `json:"sourceRef,omitempty"`
	Paths          []string `json:"paths,omitempty"`
	RepositoryID   string   `json:"repositoryId,omitempty"`
	PullRequestURL string   `json:"pullRequestUrl,omitempty"`
}

type SupersedeNoteInput struct {
	RunID            string           `json:"runId,omitempty"`
	ExpectedRevision int              `json:"expectedRevision"`
	Reason           string           `json:"reason"`
	Replacement      *CreateNoteInput `json:"replacement,omitempty"`
}

type Trajectory struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	RunID        string    `json:"runId"`
	PrincipalID  string    `json:"principalId"`
	Objective    string    `json:"objective"`
	Rationale    string    `json:"rationale,omitempty"`
	Status       string    `json:"status"`
	Paths        []string  `json:"paths,omitempty"`
	RepositoryID string    `json:"repositoryId,omitempty"`
	Branch       string    `json:"branch,omitempty"`
	BaseSHA      string    `json:"baseSha,omitempty"`
	HeadSHA      string    `json:"headSha,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Run          *Run      `json:"run,omitempty"`
}

type CreateTrajectoryInput struct {
	RunID        string   `json:"runId"`
	Objective    string   `json:"objective"`
	Rationale    string   `json:"rationale,omitempty"`
	Paths        []string `json:"paths,omitempty"`
	RepositoryID string   `json:"repositoryId,omitempty"`
	Branch       string   `json:"branch,omitempty"`
	BaseSHA      string   `json:"baseSha,omitempty"`
	HeadSHA      string   `json:"headSha,omitempty"`
}

type Repository struct {
	ID          string             `json:"id"`
	WorkspaceID string             `json:"workspaceId"`
	URL         string             `json:"url"`
	Host        string             `json:"host"`
	Owner       string             `json:"owner"`
	Name        string             `json:"name"`
	Visibility  string             `json:"visibility"`
	Description string             `json:"description,omitempty"`
	Default     string             `json:"defaultBranch,omitempty"`
	Stars       int                `json:"stars,omitempty"`
	SyncedAt    *time.Time         `json:"syncedAt,omitempty"`
	SyncError   string             `json:"syncError,omitempty"`
	Pulls       []ExternalArtifact `json:"pullRequests,omitempty"`
}

type ExternalArtifact struct {
	ID           string    `json:"id"`
	RepositoryID string    `json:"repositoryId"`
	Kind         string    `json:"kind"`
	ExternalID   string    `json:"externalId"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	State        string    `json:"state"`
	Author       string    `json:"author,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type BriefInput struct {
	RunID          string   `json:"runId,omitempty"`
	Query          string   `json:"query,omitempty"`
	Objective      string   `json:"objective,omitempty"`
	Paths          []string `json:"paths,omitempty"`
	MaxNotes       int      `json:"maxNotes,omitempty"`
	CheckConflicts bool     `json:"checkConflicts,omitempty"`
}

type CoordinationWarning struct {
	Kind          string      `json:"kind"`
	Summary       string      `json:"summary"`
	Reason        string      `json:"reason"`
	ExecutionRisk string      `json:"executionRisk"`
	Trajectory    *Trajectory `json:"trajectory,omitempty"`
	RelatedNotes  []Note      `json:"relatedNotes,omitempty"`
	Options       []string    `json:"options"`
}

type Brief struct {
	Project      Project               `json:"project"`
	Notice       string                `json:"notice"`
	Notes        []Note                `json:"notes"`
	Trajectories []Trajectory          `json:"trajectories"`
	Warnings     []CoordinationWarning `json:"coordinationWarnings"`
	GeneratedAt  time.Time             `json:"generatedAt"`
}

type Receipt struct {
	ID              string    `json:"id"`
	IdempotencyKey  string    `json:"idempotencyKey"`
	EventID         string    `json:"eventId"`
	Sequence        int64     `json:"sequence"`
	Response        []byte    `json:"-"`
	CreatedAt       time.Time `json:"createdAt"`
	OriginReplicaID string    `json:"originReplicaId,omitempty"`
	OriginSequence  int64     `json:"originSequence,omitempty"`
}

type PortableActor struct {
	PrincipalID     string `json:"principalId"`
	PortableID      string `json:"portableId"`
	DisplayName     string `json:"displayName"`
	Kind            string `json:"kind"`
	OriginReplicaID string `json:"originReplicaId"`
}

type Replica struct {
	ID                      string     `json:"id"`
	WorkspaceID             string     `json:"workspaceId"`
	DisplayName             string     `json:"displayName"`
	BaseURL                 string     `json:"baseUrl,omitempty"`
	PublicKey               string     `json:"publicKey"`
	Role                    string     `json:"role"`
	Capabilities            []string   `json:"capabilities"`
	Status                  string     `json:"status"`
	AcceptedThroughSequence *int64     `json:"acceptedThroughSequence,omitempty"`
	LastSuccessAt           *time.Time `json:"lastSuccessAt,omitempty"`
	LastError               string     `json:"lastError,omitempty"`
	ApprovedAt              time.Time  `json:"approvedAt"`
	RevokedAt               *time.Time `json:"revokedAt,omitempty"`
}

type SyncHead struct {
	OriginReplicaID string `json:"originReplicaId"`
	Sequence        int64  `json:"sequence"`
	EventHash       string `json:"eventHash"`
}

type DomainEvent struct {
	SchemaVersion   int             `json:"schemaVersion"`
	EventID         string          `json:"eventId"`
	WorkspaceID     string          `json:"workspaceId"`
	ProjectID       string          `json:"projectId,omitempty"`
	OriginReplicaID string          `json:"originReplicaId"`
	OriginSequence  int64           `json:"originSequence"`
	Type            string          `json:"type"`
	EntityID        string          `json:"entityId"`
	ActorID         string          `json:"actorId"`
	Actor           PortableActor   `json:"actor"`
	RunID           string          `json:"runId,omitempty"`
	CausalEventIDs  []string        `json:"causalEventIds"`
	OccurredAt      time.Time       `json:"occurredAt"`
	Payload         json.RawMessage `json:"payload"`
	PreviousHash    string          `json:"previousHash"`
	EventHash       string          `json:"eventHash"`
	Signature       string          `json:"signature"`
}

type ProjectSnapshot struct {
	Project        Project             `json:"project"`
	Runs           []Run               `json:"runs"`
	Notes          []Note              `json:"notes"`
	Trajectories   []Trajectory        `json:"trajectories"`
	Repositories   []Repository        `json:"repositories"`
	LifecycleEdges []NoteLifecycleEdge `json:"lifecycleEdges"`
}

type NoteLifecycleEdge struct {
	EventID           string    `json:"eventId"`
	ProjectID         string    `json:"projectId"`
	TargetNoteID      string    `json:"targetNoteId"`
	ReplacementNoteID string    `json:"replacementNoteId,omitempty"`
	Reason            string    `json:"reason,omitempty"`
	OriginReplicaID   string    `json:"originReplicaId"`
	OccurredAt        time.Time `json:"occurredAt"`
}

type WorkspaceSnapshot struct {
	SchemaVersion int               `json:"schemaVersion"`
	Workspace     Workspace         `json:"workspace"`
	Actors        []PortableActor   `json:"actors"`
	Agents        []Agent           `json:"agents"`
	Replicas      []Replica         `json:"replicas"`
	Projects      []ProjectSnapshot `json:"projects"`
	Heads         []SyncHead        `json:"heads"`
	CreatedAt     time.Time         `json:"createdAt"`
	SnapshotHash  string            `json:"snapshotHash"`
	Signature     string            `json:"signature"`
}

type WorkspaceBundle struct {
	SchemaVersion int               `json:"schemaVersion"`
	Authority     Replica           `json:"authority"`
	Snapshot      WorkspaceSnapshot `json:"snapshot"`
}
