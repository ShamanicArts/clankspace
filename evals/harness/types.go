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
	NewRunCount              int      `json:"newRunCount"`
	AllNewRunsCompleted      bool     `json:"allNewRunsCompleted"`
	ClankInvoked             bool     `json:"clankInvoked"`
	PreTaskStayedPassive     bool     `json:"preTaskStayedPassive"`
	PreTaskCommandCount      int      `json:"preTaskCommandCount"`
	PreTaskClankInvoked      bool     `json:"preTaskClankInvoked"`
	BriefInvokedBeforeWrite  bool     `json:"briefInvokedBeforeWrite"`
	RelevantRecordsSeen      []string `json:"relevantRecordsSeen"`
	RelevantTrajectoriesSeen []string `json:"relevantTrajectoriesSeen"`
	ConflictSurfaced         bool     `json:"conflictSurfaced"`
	AskedForDirection        bool     `json:"askedForDirection"`
	ForbiddenClaimsFound     []string `json:"forbiddenClaimsFound"`
	CheckpointCount          int      `json:"checkpointCount"`
}

// CollaborationScenario is the version-2 evaluation contract for one shared
// ClankSpace project worked by two independently credentialed lanes. It is an
// external-ledger artifact: LedgerOracle is never copied to a lane worktree.
type CollaborationScenario struct {
	SchemaVersion  int                 `json:"schemaVersion"`
	ID             string              `json:"id"`
	Split          string              `json:"split"`
	Category       string              `json:"category"`
	Project        ScenarioProject     `json:"project"`
	Repository     RepositoryFixture   `json:"repository"`
	SourceEvidence SourceEvidence      `json:"sourceEvidence"`
	Actors         []Actor             `json:"actors"`
	Records        []Record            `json:"records"`
	Trajectories   []Trajectory        `json:"trajectories"`
	Lanes          []CollaborationLane `json:"lanes"`
	Schedule       EventGatedSchedule  `json:"schedule"`
	Generation     Generation          `json:"generation"`
}

// SourceEvidence pins a sanitized real-source snapshot without asserting that
// its contents are historical project intent.
type SourceEvidence struct {
	RepositoryURL    string `json:"repositoryUrl"`
	License          string `json:"license"`
	LicenseFile      string `json:"licenseFile"`
	LicenseFileHash  string `json:"licenseFileHash"`
	SourceCommit     string `json:"sourceCommit"`
	SnapshotID       string `json:"snapshotId"`
	SnapshotHead     string `json:"snapshotHead"`
	BundleHash       string `json:"bundleHash"`
	HistoricalClaim  bool   `json:"historicalClaim"`
	SyntheticOverlay bool   `json:"syntheticOverlay"`
}

type CollaborationLane struct {
	ID             string             `json:"id"`
	ActorKey       string             `json:"actorKey"`
	Branch         string             `json:"branch"`
	PriorUserTurns []ConversationTurn `json:"priorUserTurns"`
	Task           LaneTask           `json:"task"`
	LedgerOracle   Oracle             `json:"ledgerOracle"`
}

type LaneTask struct {
	Objective   string   `json:"objective"`
	UserRequest string   `json:"userRequest"`
	Paths       []string `json:"paths"`
	Checks      []string `json:"checks"`
}

// AgentVisibleLane is the complete lane contract that a controller may expose
// to an agent. It deliberately excludes LedgerOracle and SourceEvidence.
type AgentVisibleLane struct {
	ID             string             `json:"id"`
	ActorKey       string             `json:"actorKey"`
	Branch         string             `json:"branch"`
	PriorUserTurns []ConversationTurn `json:"priorUserTurns"`
	Task           LaneTask           `json:"task"`
}

// EventGatedSchedule requires the controller to observe the barrier before it
// creates or launches the dependent lane.
type EventGatedSchedule struct {
	InitialLane    string      `json:"initialLane"`
	DependentLane  string      `json:"dependentLane"`
	TimeoutSeconds int         `json:"timeoutSeconds"`
	PollIntervalMS int         `json:"pollIntervalMs"`
	Barrier        BarrierSpec `json:"barrier"`
}

type BarrierSpec struct {
	EventType           string   `json:"eventType"`
	Kind                string   `json:"kind"`
	RequiredPathOverlap []string `json:"requiredPathOverlap"`
}

// CollaborationPreparedWorld is safe to retain with reportable artifacts. It
// intentionally has no token or credential-path field; secret material stays
// below the ledger's mode-0600 secrets tree and is resolved by the controller.
type CollaborationPreparedWorld struct {
	SchemaVersion  int            `json:"schemaVersion"`
	ScenarioID     string         `json:"scenarioId"`
	ScenarioHash   string         `json:"scenarioHash"`
	CorpusVersion  string         `json:"corpusVersion"`
	Split          string         `json:"split"`
	ProjectSlug    string         `json:"projectSlug"`
	ProjectID      string         `json:"projectId"`
	RepositoryHead string         `json:"repositoryHead"`
	SkillHash      string         `json:"skillHash"`
	Lanes          []PreparedLane `json:"lanes"`
	CreatedAt      time.Time      `json:"createdAt"`
}

type PreparedLane struct {
	LaneID         string `json:"laneId"`
	ActorKey       string `json:"actorKey"`
	PrincipalID    string `json:"principalId"`
	Branch         string `json:"branch"`
	AgentName      string `json:"agentName"`
	Harness        string `json:"harness"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	Reasoning      string `json:"reasoning,omitempty"`
	Role           string `json:"role"`
	RepositoryHead string `json:"repositoryHead"`
	SkillHash      string `json:"skillHash"`
	RepositoryPath string `json:"repositoryPath"`
	ArtifactPath   string `json:"artifactPath"`
}

// CollaborationEpisode and its children are the reportable, machine-readable
// evidence for an attempted two-lane rollout. All paths are ledger-relative.
type CollaborationEpisode struct {
	SchemaVersion    int                 `json:"schemaVersion"`
	EpisodeID        string              `json:"episodeId"`
	ScenarioID       string              `json:"scenarioId"`
	ScenarioHash     string              `json:"scenarioHash"`
	Status           string              `json:"status"`
	StartedAt        time.Time           `json:"startedAt"`
	EndedAt          time.Time           `json:"endedAt"`
	SchedulePath     string              `json:"schedulePath"`
	ControllerEvents string              `json:"controllerEvents"`
	Barrier          BarrierObservation  `json:"barrier"`
	Lanes            []LaneResult        `json:"lanes"`
	Repository       RepositoryResult    `json:"repository"`
	Score            CollaborationScore  `json:"score"`
	Replay           CollaborationReplay `json:"replay"`
}

// CollaborationReplay pins what the controller actually launched. The server
// configuration is represented by a supplied non-secret configuration hash;
// credential files and their paths are intentionally absent.
type CollaborationReplay struct {
	CodexBinaryHash  string `json:"codexBinaryHash,omitempty"`
	ServerURLHash    string `json:"serverUrlHash,omitempty"`
	ServerConfigHash string `json:"serverConfigHash,omitempty"`
	ServerCommit     string `json:"serverCommit,omitempty"`
}

type LaneResult struct {
	LaneID            string    `json:"laneId"`
	ActorKey          string    `json:"actorKey"`
	Status            string    `json:"status"`
	ThreadID          string    `json:"threadId,omitempty"`
	ObservedRunID     string    `json:"observedRunId,omitempty"`
	StartedAt         time.Time `json:"startedAt,omitempty"`
	EndedAt           time.Time `json:"endedAt,omitempty"`
	EventsPath        string    `json:"eventsPath,omitempty"`
	StderrPath        string    `json:"stderrPath,omitempty"`
	FinalResponsePath string    `json:"finalResponsePath,omitempty"`
	ProjectExportPath string    `json:"projectExportPath,omitempty"`
	GitResultPath     string    `json:"gitResultPath,omitempty"`
	ChecksPath        string    `json:"checksPath,omitempty"`
	CommandPath       string    `json:"commandPath,omitempty"`
}

type BarrierObservation struct {
	Observed     bool      `json:"observed"`
	LaneID       string    `json:"laneId,omitempty"`
	RunID        string    `json:"runId,omitempty"`
	NoteID       string    `json:"noteId,omitempty"`
	Paths        []string  `json:"paths,omitempty"`
	SnapshotPath string    `json:"snapshotPath,omitempty"`
	SnapshotHash string    `json:"snapshotHash,omitempty"`
	ObservedAt   time.Time `json:"observedAt,omitempty"`
}

type RepositoryResult struct {
	BaselineHead string `json:"baselineHead"`
	LaneAPath    string `json:"laneAPath"`
	LaneBPath    string `json:"laneBPath,omitempty"`
}

type ControllerEvent struct {
	At      time.Time `json:"at"`
	Type    string    `json:"type"`
	LaneID  string    `json:"laneId,omitempty"`
	Message string    `json:"message"`
}

type CollaborationScore struct {
	BarrierObserved  bool `json:"barrierObserved"`
	DependentStarted bool `json:"dependentStarted"`
	LanesCompleted   int  `json:"lanesCompleted"`
	Incomplete       bool `json:"incomplete"`
}
