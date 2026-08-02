export const meta = {
  name: 'generate-clankspace-blueprints',
  description: 'Generate fixed scenario blueprints from explicit curriculum cells before any world is rendered',
  phases: [
    { title: 'Design', detail: 'Create fixed facts and hidden oracles for explicit curriculum cells' },
    { title: 'Verify', detail: 'Reject ambiguous, leaky, or untestable blueprints before rendering' },
  ],
}

const WORKFORCE_ID = 'clankspace-blueprints-v2:codex/gpt-5.6-luna:literal-high-max:neutral-cwd'
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

log(
  `WORKFORCE ${WORKFORCE_ID}: architect=${WORKFORCE.architect.provider}/${WORKFORCE.architect.model}, ` +
    `effort=${WORKFORCE.architect.effort}, sandbox=read-only, writes=none; ` +
    `verifier=${WORKFORCE.verifier.provider}/${WORKFORCE.verifier.model}, ` +
    `effort=${WORKFORCE.verifier.effort}, sandbox=read-only, writes=none; ` +
    `cwd=${AGENT_WORKSPACE}; fallback=none`,
)

const stringArray = { type: 'array', items: { type: 'string' } }
const actorSchema = {
  type: 'object',
  additionalProperties: false,
  required: ['key', 'principalName', 'agentName', 'harness', 'provider', 'model', 'reasoning', 'role'],
  properties: {
    key: { type: 'string' },
    principalName: { type: 'string' },
    agentName: { type: 'string' },
    harness: { type: 'string' },
    provider: { type: 'string' },
    model: { type: 'string' },
    reasoning: { type: 'string' },
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
    id: { type: 'string' },
    actorKey: { type: 'string' },
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
    id: { type: 'string' },
    actorKey: { type: 'string' },
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
    actors: { type: 'array', items: actorSchema },
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
        taskActorKey: { type: 'string' },
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
  if (!sameMembers(blueprint.curriculumAxes, cell.axes)) {
    fail('curriculum-axis-drift', 'Blueprint curriculumAxes must exactly match the fixed cell axes.')
  }
  if (blueprint.conversationPlan.priorUserTurns !== cell.constraints.priorUserTurns ||
      blueprint.conversationPlan.priorUserIntents.length !== cell.constraints.priorUserTurns ||
      blueprint.conversationPlan.noAssistantTurns !== true) {
    fail('conversation-shape', 'Prior user-turn count must match the cell and assistant turns must be forbidden.')
  }
  if (cell.repositoryProfile === 'fake') {
    if (blueprint.snapshotId !== '' || blueprint.repositoryPlan.overlayOnly) {
      fail('fake-repository-boundary', 'Fake repositories require an empty snapshotId and overlayOnly=false.')
    }
    if (blueprint.repositoryPlan.minimumCommits < cell.constraints.minimumCommits ||
        blueprint.repositoryPlan.commitPlans.length < cell.constraints.minimumCommits) {
      fail('insufficient-commit-plan', 'Fake repository commit plan does not meet the fixed minimum.')
    }
  } else if (blueprint.snapshotId !== cell.snapshotId || blueprint.repositoryPlan.overlayOnly !== true ||
             blueprint.repositoryPlan.commitPlans.length !== 0) {
    fail('snapshot-boundary', 'Real-snapshot blueprints must preserve snapshotId, be overlay-only, and plan no commits.')
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
  if (!sameMembers(blueprint.fixedOracle.relevantTrajectoryIds, relevantTrajectories.map(trajectory => trajectory.id))) {
    fail('trajectory-oracle-cross-reference', 'Oracle relevantTrajectoryIds must exactly name all and only relevant trajectory plans.')
  }
  const actorKeys = new Set(blueprint.actors.map(actor => actor.key))
  const referencedActorKeys = [
    blueprint.conversationPlan.taskActorKey,
    ...blueprint.recordPlans.map(record => record.actorKey),
    ...blueprint.trajectoryPlans.map(trajectory => trajectory.actorKey),
  ]
  if (referencedActorKeys.some(key => !actorKeys.has(key))) {
    fail('actor-cross-reference', 'Every task, record, and trajectory actorKey must name a declared actor.')
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
        `the oracle. conversationPlan must contain only user intent and no assistant turns. Include deliberate ` +
        `distractors and synthetic provenance, but no credentials, private material, raw messages, insults, ` +
        `hidden reasoning, or evaluation hints visible to the eventual tested agent.\n\n` +
        `Cell index: ${index}\n${JSON.stringify(cell, null, 2)}`,
      { schema: BLUEPRINT_SCHEMA, key: `blueprint-v2:${cell.id || index}` },
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
        `Reject ambiguity, contradictions, an unsupported expected behavior, oracle leakage, invalid aliases or ` +
        `cross-references, wrong counts, assistant turns, impossible repository requirements, weak distractors, ` +
        `privacy issues, project-boundary leakage, or a scenario passable without exercising ClankSpace. For a ` +
        `real snapshot, reject invented paths or architecture and any changed snapshot ID. Do not repair it. ` +
        `accepted must be false if any error issue exists.\n\nCell index: ${index}\nFIXED CELL:\n${JSON.stringify(cell, null, 2)}\n\n` +
        `CANDIDATE BLUEPRINT:\n${JSON.stringify(blueprint, null, 2)}`,
      { schema: REVIEW_SCHEMA, key: `verify-blueprint-v2:${cell.id || index}` },
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
