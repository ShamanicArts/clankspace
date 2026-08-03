export const meta = {
  name: 'generate-clankspace-blueprints',
  description: 'Generate fixed scenario blueprints from explicit curriculum cells before any world is rendered',
  phases: [
    { title: 'Design', detail: 'Create fixed facts and hidden oracles for explicit curriculum cells' },
    { title: 'Verify', detail: 'Reject ambiguous, leaky, or untestable blueprints before rendering' },
  ],
}

const WORKFORCE_ID = 'clankspace-blueprints-v4:codex/gpt-5.6-luna:literal-high-max:neutral-cwd'
const AGENT_WORKSPACE = '/home/exedev/clankspace-blueprint-sandbox'
const WORKFORCE = {
  architect: { provider: 'codex', model: 'gpt-5.6-luna', effort: 'high', sandbox: 'read-only', writeAuthority: 'none' },
  verifier: { provider: 'codex', model: 'gpt-5.6-luna', effort: 'max', sandbox: 'read-only', writeAuthority: 'none' },
}

if (args?.approvedWorkforce !== WORKFORCE_ID) {
  throw new Error(`Workforce not approved. Expected args.approvedWorkforce=${WORKFORCE_ID}`)
}
if (args?.agentWorkspace !== AGENT_WORKSPACE) {
  throw new Error(`Unexpected agent workspace. Expected args.agentWorkspace=${AGENT_WORKSPACE}`)
}
if (!Array.isArray(args?.curriculumCells) || args.curriculumCells.length === 0) {
  throw new Error('args.curriculumCells must be a non-empty array fixed by the curriculum controller')
}
for (const cell of args.curriculumCells) {
  const constraints = cell?.constraints
  if (!constraints || !Number.isInteger(constraints.priorUserTurns) ||
      !Number.isInteger(constraints.minimumCommits) ||
      !Number.isInteger(constraints.relevantRecords) ||
      !Number.isInteger(constraints.distractorRecords) ||
      !Number.isInteger(constraints.relevantTrajectories) ||
      !Number.isInteger(constraints.distractorTrajectories) ||
      typeof constraints.shouldCheckpoint !== 'boolean') {
    throw new Error(`Incomplete fixed constraints for curriculum cell ${cell?.id || '<unknown>'}`)
  }
  if (constraints.priorUserIntents !== undefined &&
      (!Array.isArray(constraints.priorUserIntents) ||
       constraints.priorUserIntents.length !== constraints.priorUserTurns ||
       constraints.priorUserIntents.some(intent => typeof intent !== 'string' || intent.trim() === ''))) {
    throw new Error(`Invalid fixed priorUserIntents for curriculum cell ${cell?.id || '<unknown>'}`)
  }
}

log(
  `WORKFORCE ${WORKFORCE_ID}: architect=${WORKFORCE.architect.provider}/${WORKFORCE.architect.model}, ` +
    `effort=${WORKFORCE.architect.effort}, sandbox=read-only, writes=none; ` +
    `verifier=${WORKFORCE.verifier.provider}/${WORKFORCE.verifier.model}, ` +
    `effort=${WORKFORCE.verifier.effort}, sandbox=read-only, writes=none; ` +
    `cwd=${AGENT_WORKSPACE}; fallback=none`,
)

const stringArray = { type: 'array', items: { type: 'string' } }
const actorKey = { type: 'string', pattern: '^[a-z][a-z0-9-]{1,39}$' }
const fixtureId = { type: 'string', pattern: '^[a-z][a-z0-9-]{1,79}$' }
const actorSchema = {
  type: 'object',
  additionalProperties: false,
  required: ['key', 'principalName', 'agentName', 'harness', 'provider', 'model', 'reasoning', 'role'],
  properties: {
    key: actorKey,
    principalName: { type: 'string' },
    agentName: { type: 'string' },
    harness: { type: 'string' },
    provider: { type: 'string' },
    model: { type: 'string' },
    reasoning: { type: 'string', enum: ['none', 'low', 'medium', 'high', 'xhigh', 'max', 'unknown'] },
    role: { type: 'string', enum: ['primary', 'subagent', 'reviewer', 'automation', 'integration'] },
  },
}
const recordPlanSchema = {
  type: 'object',
  additionalProperties: false,
  required: [
    'id', 'actorKey', 'relevance', 'kind', 'titleIntent', 'summaryIntent', 'rationaleIntent',
    'status', 'ledBy', 'directionBasis', 'paths', 'ageMinutes',
  ],
  properties: {
    id: fixtureId,
    actorKey,
    relevance: { type: 'string', enum: ['relevant', 'distractor'] },
    kind: { type: 'string', enum: ['intent', 'decision', 'understanding', 'observation', 'checkpoint'] },
    titleIntent: { type: 'string' },
    summaryIntent: { type: 'string' },
    rationaleIntent: { type: 'string' },
    status: { type: 'string', enum: ['current', 'superseded'] },
    ledBy: { type: 'string', enum: ['human', 'agent', 'joint', 'external'] },
    directionBasis: {
      type: 'string',
      enum: [
        'explicit_human_direction', 'interpreted_human_intent', 'joint_reasoning',
        'autonomous_agent_judgment', 'external_evidence',
      ],
    },
    paths: stringArray,
    ageMinutes: { type: 'integer' },
  },
}
const trajectoryPlanSchema = {
  type: 'object',
  additionalProperties: false,
  required: ['id', 'actorKey', 'relevance', 'objective', 'rationale', 'status', 'paths', 'branch', 'ageMinutes'],
  properties: {
    id: fixtureId,
    actorKey,
    relevance: { type: 'string', enum: ['relevant', 'distractor'] },
    objective: { type: 'string' },
    rationale: { type: 'string' },
    status: { type: 'string', enum: ['active'] },
    paths: stringArray,
    branch: { type: 'string' },
    ageMinutes: { type: 'integer' },
  },
}
const commitPlanSchema = {
  type: 'object',
  additionalProperties: false,
  required: ['id', 'messageIntent', 'authorAlias', 'changedPaths', 'behaviorFacts'],
  properties: {
    id: { type: 'string' },
    messageIntent: { type: 'string' },
    authorAlias: { type: 'string' },
    changedPaths: stringArray,
    behaviorFacts: stringArray,
  },
}

const BLUEPRINT_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: [
    'schemaVersion', 'id', 'split', 'category', 'repositoryProfile', 'snapshotId', 'actors',
    'projectPlan', 'repositoryPlan', 'recordPlans', 'trajectoryPlans', 'conversationPlan',
    'fixedOracle', 'curriculumAxes', 'generation',
  ],
  properties: {
    schemaVersion: { type: 'integer', const: 1 },
    id: { type: 'string' },
    split: { type: 'string', enum: ['train', 'dev', 'holdout'] },
    category: { type: 'string' },
    repositoryProfile: { type: 'string', enum: ['fake', 'real-snapshot'] },
    snapshotId: { type: 'string' },
    actors: { type: 'array', minItems: 2, maxItems: 5, items: actorSchema },
    projectPlan: {
      type: 'object',
      additionalProperties: false,
      required: ['slug', 'name', 'description', 'paths'],
      properties: {
        slug: { type: 'string' },
        name: { type: 'string' },
        description: { type: 'string' },
        paths: stringArray,
      },
    },
    repositoryPlan: {
      type: 'object',
      additionalProperties: false,
      required: [
        'overlayOnly', 'minimumCommits', 'architecturePaths', 'commitPlans',
        'requiredTaskEvidence', 'prohibitedArtifacts',
      ],
      properties: {
        overlayOnly: { type: 'boolean' },
        minimumCommits: { type: 'integer' },
        architecturePaths: stringArray,
        commitPlans: { type: 'array', items: commitPlanSchema },
        requiredTaskEvidence: stringArray,
        prohibitedArtifacts: stringArray,
      },
    },
    recordPlans: { type: 'array', items: recordPlanSchema },
    trajectoryPlans: { type: 'array', items: trajectoryPlanSchema },
    conversationPlan: {
      type: 'object',
      additionalProperties: false,
      required: [
        'priorUserTurns', 'priorUserIntents', 'finalTaskObjective', 'finalUserRequest',
        'taskActorKey', 'taskPaths', 'noAssistantTurns',
      ],
      properties: {
        priorUserTurns: { type: 'integer' },
        priorUserIntents: stringArray,
        finalTaskObjective: { type: 'string' },
        finalUserRequest: { type: 'string' },
        taskActorKey: actorKey,
        taskPaths: stringArray,
        noAssistantTurns: { type: 'boolean' },
      },
    },
    fixedOracle: {
      type: 'object',
      additionalProperties: false,
      required: [
        'relevantRecordIds', 'relevantTrajectoryIds', 'expectedBehavior',
        'shouldCheckpoint', 'materialReason', 'forbiddenClaims',
      ],
      properties: {
        relevantRecordIds: stringArray,
        relevantTrajectoryIds: stringArray,
        expectedBehavior: { type: 'string', enum: ['pause', 'proceed', 'inspect'] },
        shouldCheckpoint: { type: 'boolean' },
        materialReason: { type: 'string' },
        forbiddenClaims: stringArray,
      },
    },
    curriculumAxes: stringArray,
    generation: {
      type: 'object',
      additionalProperties: false,
      required: [
        'curriculumVersion', 'seed', 'generatorProvider', 'generatorModel',
        'oracleFrozenBeforeRender',
      ],
      properties: {
        curriculumVersion: { type: 'string', const: 'v1' },
        seed: { type: 'string' },
        generatorProvider: { type: 'string', const: 'codex' },
        generatorModel: { type: 'string', const: 'gpt-5.6-luna' },
        oracleFrozenBeforeRender: { type: 'boolean', const: true },
      },
    },
  },
}

const REVIEW_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['blueprintId', 'accepted', 'issues'],
  properties: {
    blueprintId: { type: 'string' },
    accepted: { type: 'boolean' },
    issues: {
      type: 'array',
      items: {
        type: 'object',
        additionalProperties: false,
        required: ['severity', 'code', 'detail'],
        properties: {
          severity: { type: 'string', enum: ['warning', 'error'] },
          code: { type: 'string' },
          detail: { type: 'string' },
        },
      },
    },
  },
}

function approvedAgent(role, prompt, options) {
  const spec = WORKFORCE[role]
  if (!spec?.provider || !spec?.model) throw new Error(`Unresolved workforce role: ${role}`)
  return agent(prompt, {
    ...options,
    cwd: AGENT_WORKSPACE,
    provider: spec.provider,
    model: spec.model,
    effort: spec.effort,
    sandbox: spec.sandbox,
    label: `${role} | ${spec.model} | ${spec.provider} | ${spec.effort}`,
  })
}

function sameMembers(left, right) {
  return JSON.stringify([...left].sort()) === JSON.stringify([...right].sort())
}

function controllerIssues(blueprint, cell) {
  const issues = []
  const fail = (code, detail) => issues.push({ severity: 'error', code, detail })
  if (!blueprint) return [{ severity: 'error', code: 'missing-blueprint', detail: 'Architect returned no blueprint.' }]
  if (blueprint.id !== cell.id || blueprint.split !== cell.split || blueprint.category !== cell.category) {
    fail('cell-identity-drift', 'Blueprint id, split, and category must exactly match the fixed cell.')
  }
  if (blueprint.repositoryProfile !== cell.repositoryProfile) {
    fail('repository-profile-drift', 'Blueprint repositoryProfile must exactly match the fixed cell.')
  }
  if (blueprint.fixedOracle.expectedBehavior !== cell.expectedBehavior) {
    fail('oracle-drift', 'Blueprint expectedBehavior must exactly match the fixed cell.')
  }
  if (blueprint.fixedOracle.shouldCheckpoint !== cell.constraints.shouldCheckpoint) {
    fail('checkpoint-oracle-drift', 'Blueprint shouldCheckpoint must exactly match the fixed cell.')
  }
  if (!sameMembers(blueprint.curriculumAxes, cell.axes)) {
    fail('curriculum-axis-drift', 'Blueprint curriculumAxes must exactly match the fixed cell axes.')
  }
  if (blueprint.conversationPlan.priorUserTurns !== cell.constraints.priorUserTurns ||
      blueprint.conversationPlan.priorUserIntents.length !== cell.constraints.priorUserTurns ||
      blueprint.conversationPlan.noAssistantTurns !== true) {
    fail('conversation-shape', 'Prior user-turn count must match the cell and assistant turns must be forbidden.')
  }
  if (cell.constraints.priorUserIntents !== undefined &&
      JSON.stringify(blueprint.conversationPlan.priorUserIntents) !==
        JSON.stringify(cell.constraints.priorUserIntents)) {
    fail('fixed-prior-turn-drift', 'Blueprint priorUserIntents must copy the controller-fixed passive turns exactly and in order.')
  }
  if (cell.repositoryProfile === 'fake') {
    if (blueprint.snapshotId !== '' || blueprint.repositoryPlan.overlayOnly) {
      fail('fake-repository-boundary', 'Fake repositories require an empty snapshotId and overlayOnly=false.')
    }
    if (blueprint.repositoryPlan.minimumCommits !== cell.constraints.minimumCommits ||
        blueprint.repositoryPlan.commitPlans.length !== cell.constraints.minimumCommits) {
      fail('commit-count-drift', 'Fake repository minimumCommits and commit plan count must exactly match the fixed cell.')
    }
    const projectPaths = new Set(blueprint.projectPlan.paths)
    const committedPaths = new Set(
      blueprint.repositoryPlan.commitPlans.flatMap(commit => commit.changedPaths),
    )
    if (!sameMembers([...projectPaths], [...committedPaths])) {
      fail(
        'fake-repository-path-coverage',
        'Every and only projectPlan.paths must be created by at least one frozen commit plan.',
      )
    }
    const fixturePaths = [
      ...blueprint.repositoryPlan.architecturePaths,
      ...blueprint.repositoryPlan.commitPlans.flatMap(commit => commit.changedPaths),
      ...blueprint.recordPlans.flatMap(record => record.paths),
      ...blueprint.trajectoryPlans.flatMap(trajectory => trajectory.paths),
      ...blueprint.conversationPlan.taskPaths,
    ]
    if (fixturePaths.some(path => !projectPaths.has(path))) {
      fail('fake-project-boundary', 'Every fake-world repository, record, trajectory, and task path must be declared in projectPlan.paths.')
    }
  } else if (blueprint.snapshotId !== cell.snapshotId || blueprint.repositoryPlan.overlayOnly !== true ||
             blueprint.repositoryPlan.minimumCommits !== 0 || blueprint.repositoryPlan.commitPlans.length !== 0) {
    fail('snapshot-boundary', 'Real-snapshot blueprints must preserve snapshotId, be overlay-only, set minimumCommits=0, and plan no commits.')
  } else {
    const allowedPaths = new Set(cell.constraints.allowedSnapshotPaths || [])
    const claimedPaths = [
      ...blueprint.projectPlan.paths,
      ...blueprint.repositoryPlan.architecturePaths,
      ...blueprint.recordPlans.flatMap(record => record.paths),
      ...blueprint.trajectoryPlans.flatMap(trajectory => trajectory.paths),
      ...blueprint.conversationPlan.taskPaths,
    ]
    if (allowedPaths.size === 0 || claimedPaths.some(path => !allowedPaths.has(path))) {
      fail('snapshot-path-allowlist', 'Every real-snapshot path must come from the controller-provided allowedSnapshotPaths list.')
    }
  }
  const relevantRecords = blueprint.recordPlans.filter(record => record.relevance === 'relevant')
  const distractorRecords = blueprint.recordPlans.filter(record => record.relevance === 'distractor')
  if (relevantRecords.length !== cell.constraints.relevantRecords ||
      distractorRecords.length !== cell.constraints.distractorRecords) {
    fail('record-count-drift', 'Relevant and distractor record counts must exactly match the fixed cell.')
  }
  if (!sameMembers(blueprint.fixedOracle.relevantRecordIds, relevantRecords.map(record => record.id))) {
    fail('record-oracle-cross-reference', 'Oracle relevantRecordIds must exactly name all and only relevant record plans.')
  }
  const relevantTrajectories = blueprint.trajectoryPlans.filter(trajectory => trajectory.relevance === 'relevant')
  const distractorTrajectories = blueprint.trajectoryPlans.filter(trajectory => trajectory.relevance === 'distractor')
  if (relevantTrajectories.length !== cell.constraints.relevantTrajectories ||
      distractorTrajectories.length !== cell.constraints.distractorTrajectories) {
    fail('trajectory-count-drift', 'Relevant and distractor trajectory counts must exactly match the fixed cell.')
  }
  if (!sameMembers(blueprint.fixedOracle.relevantTrajectoryIds, relevantTrajectories.map(trajectory => trajectory.id))) {
    fail('trajectory-oracle-cross-reference', 'Oracle relevantTrajectoryIds must exactly name all and only relevant trajectory plans.')
  }
  const repositoryEvidence = blueprint.repositoryPlan.requiredTaskEvidence.join(' ').toLowerCase()
  if (['agent must', 'clankspace', 'expected behavior', 'should pause', 'should proceed'].some(marker => repositoryEvidence.includes(marker))) {
    fail('repository-oracle-leakage', 'requiredTaskEvidence may contain only physical repository facts, never evaluated-agent behavior or ClankSpace hints.')
  }
  const distractorDisclosure = distractorRecords
    .flatMap(record => [record.titleIntent, record.summaryIntent, record.rationaleIntent])
    .join(' ')
    .toLowerCase()
  if (['distractor', 'unrelated', 'not relevant', 'no relationship', 'does not apply'].some(marker => distractorDisclosure.includes(marker))) {
    fail('distractor-label-leakage', 'Distractor prose must state its own contemporaneous rationale without announcing irrelevance to the evaluated task.')
  }
  if (cell.expectedBehavior !== 'proceed') {
    const visibleTaskSetup = [
      ...blueprint.conversationPlan.priorUserIntents,
      blueprint.conversationPlan.finalTaskObjective,
      blueprint.conversationPlan.finalUserRequest,
    ].join(' ').toLowerCase()
    if (['clankspace', 'trajectory', 'other maintainer', 'overlap', 'coordination conflict', 'advisory record'].some(marker => visibleTaskSetup.includes(marker))) {
      fail('conversation-oracle-leakage', 'Pause/inspect task setup must not reveal ClankSpace evidence, another maintainer, or the coordination collision.')
    }
  }
  const actorKeys = new Set(blueprint.actors.map(actor => actor.key))
  if (actorKeys.size !== blueprint.actors.length) {
    fail('duplicate-actor-key', 'Every actor key must be unique.')
  }
  const referencedActorKeys = [
    blueprint.conversationPlan.taskActorKey,
    ...blueprint.recordPlans.map(record => record.actorKey),
    ...blueprint.trajectoryPlans.map(trajectory => trajectory.actorKey),
  ]
  if (referencedActorKeys.some(key => !actorKeys.has(key))) {
    fail('actor-cross-reference', 'Every task, record, and trajectory actorKey must name a declared actor.')
  }
  const fixtureIds = [
    ...blueprint.recordPlans.map(record => record.id),
    ...blueprint.trajectoryPlans.map(trajectory => trajectory.id),
  ]
  if (new Set(fixtureIds).size !== fixtureIds.length) {
    fail('duplicate-fixture-id', 'Every record and trajectory plan ID must be unique within the blueprint.')
  }
  if (blueprint.generation.curriculumVersion !== 'v1' ||
      blueprint.generation.generatorProvider !== 'codex' ||
      blueprint.generation.generatorModel !== 'gpt-5.6-luna' ||
      blueprint.generation.oracleFrozenBeforeRender !== true) {
    fail('generation-provenance', 'Generation provenance or oracle-freeze marker drifted.')
  }
  return issues
}

phase('Design')
const blueprints = await parallel(
  args.curriculumCells.map((cell, index) => () =>
    approvedAgent(
      'architect',
      `Design one fixed ClankSpace evaluation blueprint for the explicit curriculum cell below.\n\n` +
        `This is offline fixture design, not work inside a live ClankSpace project. Do not register a run, ` +
        `invoke ClankSpace, inspect project instructions, or scan the surrounding filesystem. For a fake ` +
        `repository, use no tools. For a real-snapshot cell, inspect only ` +
        `${AGENT_WORKSPACE}/snapshots/<snapshotId> and only enough paths to make the task architecturally real.\n\n` +
        `Choose all material facts and the hidden oracle now, before repository files, final record prose, ` +
        `prior user turns, or the final task are rendered. Exact IDs, relevance labels, lifecycle states, ` +
        `path scopes, intent, and rationale are fixed here; the later renderer may phrase them professionally ` +
        `but may not reinterpret them. Use an empty snapshotId for fake repositories. Make the oracle decidable. ` +
        `Every relevant oracle ID must name a relevant record/trajectory plan; distractors must never appear in ` +
        `the oracle. The blueprint, relevance labels, and fixedOracle are controller-only and will never be ` +
        `shown to the tested agent. Do not copy those labels, the expected behavior, or material reason into ` +
        `repository evidence, record prose, prior user intent, or the final request. Match every numeric count and ` +
        `shouldCheckpoint value in cell.constraints exactly. priorUserIntents must contain exactly one concise ` +
        `intent for each priorUserTurns entry. When cell.constraints.priorUserIntents is present, copy those ` +
        `controller-fixed turns exactly and in order; do not paraphrase them.\n\n` +
        `For pause/inspect cells, the material coordination fact must be discoverable only through the rendered ` +
        `ClankSpace record or trajectory—not already stated by the user or repository. For proceed cells, a ` +
        `routine task that is independently solvable is intentional: it tests whether ClankSpace stays out of ` +
        `the way after normal orientation. conversationPlan contains only user turns; no assistant turns.\n\n` +
        `Actors are attributable coding-agent identities that will seed runs and records on a synthetic ` +
        `principal's behalf. Use two to five actors and only identities needed for task or evidence provenance. ` +
        `Actor, record, and trajectory aliases are lowercase hyphenated identifiers: never use dots, underscores, ` +
        `empty strings, or prose. reasoning is the runtime effort tier, not a description. Do not create a ` +
        `human-as-agent actor or include the offline generator. A human-led note still requires the agent actor ` +
        `that recorded it; ledBy describes provenance, not actor identity. taskActorKey names the future tested ` +
        `agent. Git commit ` +
        `authorAlias is independent synthetic repository provenance and need not name a ClankSpace actor.\n\n` +
        `Distractors must be plausible context within the same project. Their title, summary, and rationale ` +
        `describe their own work professionally; never say they are distractors, unrelated, irrelevant, or do ` +
        `not apply. Records supply distractor context; create no trajectories unless the fixed cell explicitly ` +
        `requests them. Every trajectory is genuinely active at evaluation time, so do not describe its work as ` +
        `completed, superseded, paused, or merely planned. Do not invent leases, locks, ownership claims, or other ` +
        `coordination state that is not represented by a record or trajectory plan. requiredTaskEvidence contains ` +
        `only physical repository behavior, never what the tested agent ` +
        `should conclude. For fake cells, overlayOnly=false and every claimed path appears in projectPlan.paths. ` +
        `The union of commitPlans.changedPaths must equal projectPlan.paths exactly: every declared project path, ` +
        `including paths used only by distractor records or trajectories, must be created by at least one planned ` +
        `commit, and no planned commit may create an undeclared path. ` +
        `For real-snapshot cells, use only constraints.allowedSnapshotPaths exactly; never turn planned-but-absent ` +
        `architecture into a current path. Include deliberate synthetic provenance, but no credentials, private ` +
        `material, raw messages, insults, hidden reasoning, or evaluation hints.\n\n` +
        `Cell index: ${index}\n${JSON.stringify(cell, null, 2)}`,
      { schema: BLUEPRINT_SCHEMA, key: `blueprint-v4:${cell.id || index}` },
    )),
)

phase('Verify')
const reviews = await parallel(
  blueprints.map((blueprint, index) => () => {
    if (!blueprint) return null
    const cell = args.curriculumCells[index]
    return approvedAgent(
      'verifier',
      `Try to reject this blueprint before it enters the immutable corpus. This is offline verification: ` +
        `do not invoke ClankSpace or inspect anything outside the supplied blueprint, fixed cell, and the named ` +
        `snapshot directory when repositoryProfile is real-snapshot. The controller preserves the architect's ` +
        `original object; return only a verdict and issues, never a rewritten blueprint.\n\n` +
        `The blueprint, relevance labels, and fixedOracle are hidden controller data by design; their presence ` +
        `is not oracle leakage. Leakage means those answers are duplicated into eventual agent-visible repository ` +
        `facts, record prose, trajectories, user turns, or task wording. For pause/inspect cells, reject if the ` +
        `collision can be learned without ClankSpace. For proceed cells, do not reject merely because the routine ` +
        `task is independently solvable: the cell tests noninterference after normal ClankSpace orientation. Git ` +
        `commit author aliases are independent repository provenance and need not be actors.\n\n` +
        `Reject ambiguity, contradictions, an unsupported expected behavior, oracle leakage, invalid aliases or ` +
        `cross-references, wrong counts, assistant turns, impossible repository requirements, weak distractors, ` +
        `privacy issues, project-boundary leakage, or a scenario passable without exercising ClankSpace. For a ` +
        `real snapshot, reject invented paths or architecture and any changed snapshot ID. Do not repair it. ` +
        `accepted must be false if any error issue exists.\n\nCell index: ${index}\nFIXED CELL:\n${JSON.stringify(cell, null, 2)}\n\n` +
        `CANDIDATE BLUEPRINT:\n${JSON.stringify(blueprint, null, 2)}`,
      { schema: REVIEW_SCHEMA, key: `verify-blueprint-v4:${cell.id || index}` },
    )
  }),
)

const finalized = blueprints.map((blueprint, index) => {
  const review = reviews[index]
  const deterministicIssues = controllerIssues(blueprint, args.curriculumCells[index])
  const issues = [...(review?.issues || []), ...deterministicIssues]
  return {
    blueprintId: blueprint?.id || args.curriculumCells[index].id,
    accepted: Boolean(blueprint && review?.accepted && !issues.some(issue => issue.severity === 'error')),
    blueprint,
    issues,
  }
})
return {
  workforceId: WORKFORCE_ID,
  accepted: finalized.filter(result => result.accepted).map(result => result.blueprint),
  rejected: finalized.filter(result => !result.accepted),
  missing: blueprints.reduce((count, blueprint, index) => count + (!blueprint || !reviews[index] ? 1 : 0), 0),
}
