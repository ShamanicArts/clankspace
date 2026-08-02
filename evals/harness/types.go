package harness

import "time"

type Scenario struct {
	SchemaVersion int                `json:"schemaVersion"`
	ID            string             `json:"id"`
	Split         string             `json:"split"`
	Category      string             `json:"category"`
	Project       ScenarioProject    `json:"project"`
	Repository    RepositoryFixture  `json:"repository"`
	Actors        []Actor            `json:"actors"`
	Records       []Record           `json:"records"`
	Trajectories  []Trajectory       `json:"trajectories"`
	Conversation  []ConversationTurn `json:"conversation"`
	Task          Task               `json:"task"`
	Oracle        Oracle             `json:"oracle"`
	Generation    Generation         `json:"generation"`
}

type ScenarioProject struct {
	Slug              string   `json:"slug"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	RepositoryProfile string   `json:"repositoryProfile"`
	Paths             []string `json:"paths"`
}

type RepositoryFixture struct {
	SnapshotID string       `json:"snapshotId,omitempty"`
	BaseRef    string       `json:"baseRef,omitempty"`
	Commits    []CommitSpec `json:"commits,omitempty"`
}

type CommitSpec struct {
	ID          string       `json:"id"`
	Message     string       `json:"message"`
	AuthorName  string       `json:"authorName"`
	AuthorEmail string       `json:"authorEmail"`
	Changes     []FileChange `json:"changes"`
}

type FileChange struct {
	Path       string `json:"path"`
	Content    string `json:"content,omitempty"`
	Delete     bool   `json:"delete,omitempty"`
	Executable bool   `json:"executable,omitempty"`
}

type Actor struct {
	Key           string `json:"key"`
	PrincipalName string `json:"principalName"`
	AgentName     string `json:"agentName"`
	Harness       string `json:"harness"`
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	Reasoning     string `json:"reasoning,omitempty"`
	Role          string `json:"role"`
}

type Record struct {
	ID             string   `json:"id"`
	ActorKey       string   `json:"actorKey"`
	Kind           string   `json:"kind"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	Rationale      string   `json:"rationale"`
	Status         string   `json:"status"`
	LedBy          string   `json:"ledBy"`
	DirectionBasis string   `json:"directionBasis"`
	Paths          []string `json:"paths"`
	AgeMinutes     int      `json:"ageMinutes"`
}

type Trajectory struct {
	ID         string   `json:"id"`
	ActorKey   string   `json:"actorKey"`
	Objective  string   `json:"objective"`
	Rationale  string   `json:"rationale"`
	Status     string   `json:"status"`
	Paths      []string `json:"paths"`
	Branch     string   `json:"branch"`
	AgeMinutes int      `json:"ageMinutes"`
}

type ConversationTurn struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type Task struct {
	ActorKey    string   `json:"actorKey"`
	Objective   string   `json:"objective"`
	UserRequest string   `json:"userRequest"`
	Paths       []string `json:"paths"`
}

type Oracle struct {
	RelevantRecordIDs     []string `json:"relevantRecordIds"`
	RelevantTrajectoryIDs []string `json:"relevantTrajectoryIds"`
	ExpectedBehavior      string   `json:"expectedBehavior"`
	ShouldCheckpoint      bool     `json:"shouldCheckpoint"`
	MaterialReason        string   `json:"materialReason"`
	ForbiddenClaims       []string `json:"forbiddenClaims"`
}

type Generation struct {
	CurriculumVersion string `json:"curriculumVersion"`
	Seed              string `json:"seed"`
	GeneratorProvider string `json:"generatorProvider"`
	GeneratorModel    string `json:"generatorModel"`
	WorkflowRun       string `json:"workflowRun,omitempty"`
}

type PreparedWorld struct {
	SchemaVersion  int               `json:"schemaVersion"`
	ScenarioID     string            `json:"scenarioId"`
	ScenarioHash   string            `json:"scenarioHash"`
	SkillHash      string            `json:"skillHash"`
	CorpusVersion  string            `json:"corpusVersion"`
	Split          string            `json:"split"`
	ProjectSlug    string            `json:"projectSlug"`
	ProjectID      string            `json:"projectId,omitempty"`
	RepositoryPath string            `json:"repositoryPath"`
	RepositoryHead string            `json:"repositoryHead"`
	RecordIDs      map[string]string `json:"recordIds,omitempty"`
	TrajectoryIDs  map[string]string `json:"trajectoryIds,omitempty"`
	RunIDs         map[string]string `json:"runIds,omitempty"`
	CredentialFile string            `json:"credentialFile"`
	CreatedAt      time.Time         `json:"createdAt"`
}

type RolloutResult struct {
	SchemaVersion int                `json:"schemaVersion"`
	EpisodeID     string             `json:"episodeId"`
	ScenarioID    string             `json:"scenarioId"`
	ScenarioHash  string             `json:"scenarioHash"`
	ThreadID      string             `json:"threadId"`
	ClankRunID    string             `json:"clankRunId"`
	Model         string             `json:"model"`
	Reasoning     string             `json:"reasoning"`
	StartedAt     time.Time          `json:"startedAt"`
	EndedAt       time.Time          `json:"endedAt"`
	Turns         []TurnArtifact     `json:"turns"`
	FinalResponse string             `json:"finalResponse"`
	ProjectExport string             `json:"projectExport"`
	Deterministic DeterministicScore `json:"deterministicScore"`
}

type TurnArtifact struct {
	Index     int    `json:"index"`
	Role      string `json:"role"`
	Prompt    string `json:"prompt"`
	TracePath string `json:"tracePath"`
	Response  string `json:"response"`
}

type DeterministicScore struct {
	ExpectedBehavior         string   `json:"expectedBehavior"`
	RunRegistered            bool     `json:"runRegistered"`
	ClankInvoked             bool     `json:"clankInvoked"`
	BriefInvokedBeforeWrite  bool     `json:"briefInvokedBeforeWrite"`
	RelevantRecordsSeen      []string `json:"relevantRecordsSeen"`
	RelevantTrajectoriesSeen []string `json:"relevantTrajectoriesSeen"`
	ConflictSurfaced         bool     `json:"conflictSurfaced"`
	AskedForDirection        bool     `json:"askedForDirection"`
	ForbiddenClaimsFound     []string `json:"forbiddenClaimsFound"`
	CheckpointCount          int      `json:"checkpointCount"`
}
