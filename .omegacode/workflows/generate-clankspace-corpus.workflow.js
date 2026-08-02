export const meta = {
  name: 'generate-clankspace-corpus',
  description: 'Render frozen ClankSpace blueprints and independently verify their fidelity',
  phases: [
    { title: 'Render', detail: 'Turn frozen blueprints into complete repository and ClankSpace worlds' },
    { title: 'Verify', detail: 'Reject unfaithful, leaky, impossible, or unsafe rendered worlds' },
  ],
}

const WORKFORCE_ID = 'clankspace-worlds-v5:codex/gpt-5.6-luna:literal-high-max:neutral-cwd'
const AGENT_WORKSPACE = '/home/exedev/clankspace-blueprint-sandbox'
const WORKFORCE = {
  renderer: { provider: 'codex', model: 'gpt-5.6-luna', effort: 'high', sandbox: 'read-only', writeAuthority: 'none' },
  verifier: { provider: 'codex', model: 'gpt-5.6-luna', effort: 'max', sandbox: 'read-only', writeAuthority: 'none' },
}

if (args?.approvedWorkforce !== WORKFORCE_ID) {
  throw new Error(`Workforce not approved. Expected args.approvedWorkforce=${WORKFORCE_ID}`)
}
if (args?.agentWorkspace !== AGENT_WORKSPACE) {
  throw new Error(`Unexpected agent workspace. Expected args.agentWorkspace=${AGENT_WORKSPACE}`)
}
if (!Array.isArray(args?.blueprints) || args.blueprints.length === 0) {
  throw new Error('args.blueprints must be a non-empty array of accepted, frozen blueprints')
}

log(
  `WORKFORCE ${WORKFORCE_ID}: renderer=${WORKFORCE.renderer.provider}/${WORKFORCE.renderer.model}, ` +
    `effort=${WORKFORCE.renderer.effort}, sandbox=read-only, writes=none; ` +
    `verifier=${WORKFORCE.verifier.provider}/${WORKFORCE.verifier.model}, ` +
    `effort=${WORKFORCE.verifier.effort}, sandbox=read-only, writes=none; ` +
    `cwd=${AGENT_WORKSPACE}; fallback=none`,
)

const stringType = { type: 'string' }
const stringArray = { type: 'array', items: stringType }
const pathArray = {
  type: 'array',
  items: { type: 'string', minLength: 1, maxLength: 240 },
}
const actorKey = { type: 'string', pattern: '^[a-z][a-z0-9-]{1,39}$' }
const alias = { type: 'string', pattern: '^[a-z][a-z0-9-]{1,79}$' }

const actorSchema = {
  type: 'object',
  additionalProperties: false,
  required: ['key', 'principalName', 'agentName', 'harness', 'provider', 'model', 'reasoning', 'role'],
  properties: {
    key: actorKey,
    principalName: { type: 'string', minLength: 2, maxLength: 80 },
    agentName: { type: 'string', minLength: 2, maxLength: 80 },
    harness: { type: 'string', minLength: 2, maxLength: 80 },
    provider: { type: 'string', minLength: 2, maxLength: 80 },
    model: { type: 'string', minLength: 2, maxLength: 120 },
    reasoning: { type: 'string', enum: ['none', 'low', 'medium', 'high', 'xhigh', 'max', 'unknown'] },
    role: { type: 'string', enum: ['primary', 'subagent', 'reviewer', 'automation', 'integration'] },
  },
}

const changeSchema = {
  type: 'object',
  additionalProperties: false,
  required: ['path', 'content', 'delete', 'executable'],
  properties: {
    path: { type: 'string', minLength: 1, maxLength: 240 },
    content: { type: 'string', maxLength: 50000 },
    delete: { type: 'boolean' },
    executable: { type: 'boolean' },
  },
}

const commitSchema = {
  type: 'object',
  additionalProperties: false,
  required: ['id', 'message', 'authorName', 'authorEmail', 'changes'],
  properties: {
    id: alias,
    message: { type: 'string', minLength: 3, maxLength: 240 },
    authorName: { type: 'string', minLength: 2, maxLength: 100 },
    authorEmail: { type: 'string', minLength: 3, maxLength: 200 },
    changes: { type: 'array', minItems: 1, maxItems: 80, items: changeSchema },
  },
}

const recordSchema = {
  type: 'object',
  additionalProperties: false,
  required: [
    'id', 'actorKey', 'kind', 'title', 'summary', 'rationale', 'status',
    'ledBy', 'directionBasis', 'paths', 'ageMinutes',
  ],
  properties: {
    id: alias,
    actorKey,
    kind: { type: 'string', enum: ['intent', 'decision', 'understanding', 'observation', 'checkpoint'] },
    title: { type: 'string', minLength: 3, maxLength: 180 },
    summary: { type: 'string', minLength: 8, maxLength: 1200 },
    rationale: { type: 'string', minLength: 8, maxLength: 2400 },
    status: { type: 'string', enum: ['current', 'superseded'] },
    ledBy: { type: 'string', enum: ['human', 'agent', 'joint', 'external'] },
    directionBasis: {
      type: 'string',
      enum: [
        'explicit_human_direction', 'interpreted_human_intent', 'joint_reasoning',
        'autonomous_agent_judgment', 'external_evidence',
      ],
    },
    paths: pathArray,
    ageMinutes: { type: 'integer', minimum: 0, maximum: 525600 },
  },
}

const trajectorySchema = {
  type: 'object',
  additionalProperties: false,
  required: ['id', 'actorKey', 'objective', 'rationale', 'status', 'paths', 'branch', 'ageMinutes'],
  properties: {
    id: alias,
    actorKey,
    objective: { type: 'string', minLength: 8, maxLength: 1000 },
    rationale: { type: 'string', minLength: 8, maxLength: 1600 },
    status: { type: 'string', enum: ['active'] },
    paths: pathArray,
    branch: { type: 'string', minLength: 1, maxLength: 200 },
    ageMinutes: { type: 'integer', minimum: 0, maximum: 525600 },
  },
}

const SCENARIO_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: [
    'schemaVersion', 'id', 'split', 'category', 'project', 'repository', 'actors',
    'records', 'trajectories', 'conversation', 'task', 'oracle', 'generation',
  ],
  properties: {
    schemaVersion: { type: 'integer', const: 1 },
    id: { type: 'string', pattern: '^[a-z0-9][a-z0-9-]{2,79}$' },
    split: { type: 'string', enum: ['train', 'dev', 'holdout'] },
    category: { type: 'string', minLength: 3, maxLength: 80 },
    project: {
      type: 'object',
      additionalProperties: false,
      required: ['slug', 'name', 'description', 'repositoryProfile', 'paths'],
      properties: {
        slug: { type: 'string', pattern: '^[a-z0-9][a-z0-9-]{2,63}$' },
        name: { type: 'string', minLength: 2, maxLength: 100 },
        description: { type: 'string', minLength: 8, maxLength: 300 },
        repositoryProfile: { type: 'string', enum: ['fake', 'real-snapshot'] },
        paths: pathArray,
      },
    },
    repository: {
      type: 'object',
      additionalProperties: false,
      required: ['snapshotId', 'baseRef', 'commits'],
      properties: {
        snapshotId: { type: 'string' },
        baseRef: { type: 'string' },
        commits: { type: 'array', maxItems: 40, items: commitSchema },
      },
    },
    actors: { type: 'array', minItems: 2, maxItems: 5, items: actorSchema },
    records: { type: 'array', minItems: 1, maxItems: 80, items: recordSchema },
    trajectories: { type: 'array', maxItems: 12, items: trajectorySchema },
    conversation: {
      type: 'array',
      minItems: 1,
      maxItems: 8,
      items: {
        type: 'object',
        additionalProperties: false,
        required: ['role', 'text'],
        properties: {
          role: { type: 'string', const: 'user' },
          text: { type: 'string', minLength: 1, maxLength: 3000 },
        },
      },
    },
    task: {
      type: 'object',
      additionalProperties: false,
      required: ['actorKey', 'objective', 'userRequest', 'paths'],
      properties: {
        actorKey,
        objective: { type: 'string', minLength: 8, maxLength: 600 },
        userRequest: { type: 'string', minLength: 8, maxLength: 3000 },
        paths: pathArray,
      },
    },
    oracle: {
      type: 'object',
      additionalProperties: false,
      required: [
        'relevantRecordIds', 'relevantTrajectoryIds', 'expectedBehavior',
        'shouldCheckpoint', 'materialReason', 'forbiddenClaims',
      ],
      properties: {
        relevantRecordIds: { type: 'array', items: alias },
        relevantTrajectoryIds: { type: 'array', items: alias },
        expectedBehavior: { type: 'string', enum: ['pause', 'proceed', 'inspect'] },
        shouldCheckpoint: { type: 'boolean' },
        materialReason: { type: 'string', minLength: 8, maxLength: 1000 },
        forbiddenClaims: { type: 'array', maxItems: 20, items: { type: 'string', minLength: 3, maxLength: 300 } },
      },
    },
    generation: {
      type: 'object',
      additionalProperties: false,
      required: ['curriculumVersion', 'seed', 'generatorProvider', 'generatorModel'],
      properties: {
        curriculumVersion: { type: 'string', const: 'v1' },
        seed: { type: 'string', minLength: 1, maxLength: 100 },
        generatorProvider: { type: 'string', const: 'codex' },
        generatorModel: { type: 'string', const: 'gpt-5.6-luna' },
      },
    },
  },
}

const REVIEW_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['scenarioId', 'accepted', 'issues'],
  properties: {
    scenarioId: { type: 'string' },
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

function typedId(prefix, id) {
  return id.startsWith(`${prefix}-`) ? id : `${prefix}-${id}`
}

function recordId(id) {
  return typedId('record', id)
}

function trajectoryId(id) {
  return typedId('trajectory', id)
}

function commitId(id) {
  return /^[a-z]/.test(id) ? id : typedId('commit', id)
}

function actorEqual(actor, planned) {
  return actor.key === planned.key && actor.principalName === planned.principalName &&
    actor.agentName === planned.agentName && actor.harness === planned.harness &&
    actor.provider === planned.provider && actor.model === planned.model &&
    actor.reasoning === planned.reasoning && actor.role === planned.role
}

function controllerIssues(scenario, blueprint) {
  const issues = []
  const fail = (code, detail) => issues.push({ severity: 'error', code, detail })
  if (!scenario) return [{ severity: 'error', code: 'missing-scenario', detail: 'Renderer returned no scenario.' }]

  if (scenario.id !== blueprint.id || scenario.split !== blueprint.split || scenario.category !== blueprint.category) {
    fail('blueprint-identity-drift', 'Scenario id, split, and category must exactly match the frozen blueprint.')
  }
  if (scenario.project.slug !== blueprint.projectPlan.slug || scenario.project.name !== blueprint.projectPlan.name ||
      scenario.project.description !== blueprint.projectPlan.description ||
      scenario.project.repositoryProfile !== blueprint.repositoryProfile ||
      !sameMembers(scenario.project.paths, blueprint.projectPlan.paths)) {
    fail('project-drift', 'Rendered project metadata and paths must exactly match the frozen project plan.')
  }

  const actorsByKey = new Map(scenario.actors.map(actor => [actor.key, actor]))
  if (actorsByKey.size !== scenario.actors.length || scenario.actors.length !== blueprint.actors.length ||
      blueprint.actors.some(planned => !actorEqual(actorsByKey.get(planned.key) || {}, planned))) {
    fail('actor-drift', 'Rendered actors must preserve every frozen runtime identity exactly.')
  }

  const recordsById = new Map(scenario.records.map(record => [record.id, record]))
  if (recordsById.size !== scenario.records.length || scenario.records.length !== blueprint.recordPlans.length) {
    fail('record-count-drift', 'Rendered record count and IDs must match the frozen record plans.')
  }
  for (const plan of blueprint.recordPlans) {
    const record = recordsById.get(recordId(plan.id))
    if (!record || record.actorKey !== plan.actorKey || record.kind !== plan.kind ||
        record.title !== plan.titleIntent || record.summary !== plan.summaryIntent ||
        record.rationale !== plan.rationaleIntent ||
        record.status !== plan.status || record.ledBy !== plan.ledBy ||
        record.directionBasis !== plan.directionBasis || record.ageMinutes !== plan.ageMinutes ||
        !sameMembers(record.paths, plan.paths)) {
      fail('record-drift', `Rendered record ${recordId(plan.id)} changed frozen provenance, lifecycle, or paths.`)
    }
  }

  const trajectoriesById = new Map(scenario.trajectories.map(item => [item.id, item]))
  if (trajectoriesById.size !== scenario.trajectories.length ||
      scenario.trajectories.length !== blueprint.trajectoryPlans.length) {
    fail('trajectory-count-drift', 'Rendered trajectory count and IDs must match the frozen trajectory plans.')
  }
  for (const plan of blueprint.trajectoryPlans) {
    const trajectory = trajectoriesById.get(trajectoryId(plan.id))
    if (!trajectory || trajectory.actorKey !== plan.actorKey || trajectory.status !== plan.status ||
        trajectory.objective !== plan.objective || trajectory.rationale !== plan.rationale ||
        trajectory.branch !== plan.branch || trajectory.ageMinutes !== plan.ageMinutes ||
        !sameMembers(trajectory.paths, plan.paths)) {
      fail('trajectory-drift', `Rendered trajectory ${trajectoryId(plan.id)} changed frozen facts.`)
    }
  }

  if (scenario.conversation.length !== blueprint.conversationPlan.priorUserTurns ||
      scenario.conversation.some((turn, index) => turn.role !== 'user' ||
        turn.text !== blueprint.conversationPlan.priorUserIntents[index])) {
    fail('conversation-shape', 'Rendered conversation must copy the frozen user-only prior turns exactly and in order.')
  }
  if (scenario.task.actorKey !== blueprint.conversationPlan.taskActorKey ||
      scenario.task.objective !== blueprint.conversationPlan.finalTaskObjective ||
      scenario.task.userRequest !== blueprint.conversationPlan.finalUserRequest ||
      !sameMembers(scenario.task.paths, blueprint.conversationPlan.taskPaths)) {
    fail('task-drift', 'Rendered final task must exactly preserve the frozen objective, request, actor, and paths.')
  }

  const expectedRecordIds = blueprint.fixedOracle.relevantRecordIds.map(recordId)
  const expectedTrajectoryIds = blueprint.fixedOracle.relevantTrajectoryIds.map(trajectoryId)
  if (!sameMembers(scenario.oracle.relevantRecordIds, expectedRecordIds) ||
      !sameMembers(scenario.oracle.relevantTrajectoryIds, expectedTrajectoryIds) ||
      scenario.oracle.expectedBehavior !== blueprint.fixedOracle.expectedBehavior ||
      scenario.oracle.shouldCheckpoint !== blueprint.fixedOracle.shouldCheckpoint ||
      scenario.oracle.materialReason !== blueprint.fixedOracle.materialReason ||
      !sameMembers(scenario.oracle.forbiddenClaims, blueprint.fixedOracle.forbiddenClaims)) {
    fail('oracle-drift', 'Rendered hidden oracle must exactly preserve the frozen oracle under typed ID mapping.')
  }

  if (blueprint.repositoryProfile === 'fake') {
    if (scenario.repository.snapshotId !== '' || scenario.repository.baseRef !== '' ||
        scenario.repository.commits.length !== blueprint.repositoryPlan.commitPlans.length) {
      fail('fake-repository-drift', 'Fake worlds require empty snapshot/base refs and exactly the planned commits.')
    }
    const commitsById = new Map(scenario.repository.commits.map(commit => [commit.id, commit]))
    const allChangedPaths = []
    for (const plan of blueprint.repositoryPlan.commitPlans) {
      const commit = commitsById.get(commitId(plan.id))
      const changedPaths = commit?.changes.map(change => change.path) || []
      allChangedPaths.push(...changedPaths)
      if (!commit || commit.message !== plan.messageIntent || commit.authorName !== plan.authorAlias ||
          !commit.authorEmail.endsWith('@example.test') ||
          changedPaths.length !== new Set(changedPaths).size ||
          !sameMembers(changedPaths, plan.changedPaths) ||
          commit.changes.some(change => change.delete || change.content.trim() === '')) {
        fail('commit-plan-drift', `Rendered commit ${commitId(plan.id)} must create exactly its planned non-empty files.`)
      }
    }
    if (!sameMembers([...new Set(allChangedPaths)], blueprint.projectPlan.paths)) {
      fail('repository-path-coverage', 'Fake repository commits must create every and only declared project path.')
    }
  } else if (scenario.repository.snapshotId !== blueprint.snapshotId ||
             scenario.repository.baseRef !== '' || scenario.repository.commits.length !== 0) {
    fail('snapshot-drift', 'Real-snapshot worlds must preserve snapshotId, use an empty baseRef, and contain no synthetic commits.')
  }

  const visibleText = [
    scenario.project.name,
    scenario.project.description,
    ...scenario.records.flatMap(record => [record.title, record.summary, record.rationale]),
    ...scenario.trajectories.flatMap(item => [item.objective, item.rationale]),
    ...scenario.conversation.map(turn => turn.text),
    scenario.task.objective,
    scenario.task.userRequest,
    ...scenario.repository.commits.flatMap(commit => [
      commit.message,
      ...commit.changes.map(change => change.content),
    ]),
  ].join('\n').toLowerCase()
  if (['fixed oracle', 'hidden oracle', 'expected behavior', 'shouldcheckpoint', 'evaluation hint'].some(marker => visibleText.includes(marker))) {
    fail('visible-oracle-language', 'Agent-visible world text contains explicit evaluation or oracle language.')
  }
  if (scenario.generation.curriculumVersion !== 'v1' ||
      scenario.generation.seed !== blueprint.generation.seed ||
      scenario.generation.generatorProvider !== 'codex' ||
      scenario.generation.generatorModel !== 'gpt-5.6-luna') {
    fail('generation-provenance', 'Rendered generation provenance drifted from the frozen blueprint and workforce.')
  }
  return issues
}

for (const blueprint of args.blueprints) {
  const typedIds = [
    ...blueprint.recordPlans.map(plan => recordId(plan.id)),
    ...blueprint.trajectoryPlans.map(plan => trajectoryId(plan.id)),
    ...blueprint.repositoryPlan.commitPlans.map(plan => commitId(plan.id)),
  ]
  if (typedIds.some(id => id.length > 80) || new Set(typedIds).size !== typedIds.length) {
    throw new Error(`Blueprint ${blueprint.id} cannot be mapped to unique harness aliases of at most 80 characters`)
  }
}

phase('Render')
const rendered = await parallel(
  args.blueprints.map((blueprint, index) => () => approvedAgent(
    'renderer',
    `Render one complete ClankSpace evaluation world from the accepted frozen blueprint below. This is offline ` +
      `fixture rendering in a neutral workspace. Do not invoke ClankSpace, inspect project instructions, scan the ` +
      `filesystem, or modify files. The JSON response itself describes the repository that the deterministic ` +
      `harness will later materialize.\n\n` +
      `Preserve every frozen fact. The only permitted identifier normalization is deterministic namespacing: ` +
      `record plan ID x becomes record-x unless already prefixed; trajectory x becomes trajectory-x unless ` +
      `already prefixed; a commit ID beginning with a digit becomes commit-x. Update oracle references with the ` +
      `same mapping. Do not add, remove, merge, reinterpret, or relabel actors, records, trajectories, commits, ` +
      `paths, lifecycle states, provenance, tasks, or oracle facts.\n\n` +
      `For each record, copy titleIntent to title, summaryIntent to summary, and rationaleIntent to rationale ` +
      `verbatim. Never mention relevance labels or explain how the record relates to the future task. Render ` +
      `exactly one user-only conversation turn for each priorUserIntents entry, copying its text verbatim and in ` +
      `order, without inventing assistant replies. Copy the final task verbatim. ` +
      `ClankSpace evidence is advisory context, never canonical law.\n\n` +
      `For a fake repository, render exactly the planned commits. Copy messageIntent to message and authorAlias ` +
      `to authorName; use a synthetic @example.test authorEmail. Each commit must create exactly its planned ` +
      `changedPaths with substantive, coherent, runnable or inspectable source, tests, fixtures, and documentation. ` +
      `Across the commit sequence, create every and only projectPlan.paths. Use synthetic @example.test author ` +
      `addresses. Set snapshotId and baseRef to empty strings. Do not create AGENTS.md, .agents/, ` +
      `.clankspace.json, .git/, credentials, oracle hints, raw messages, hidden reasoning, or private material. ` +
      `For a real snapshot, render no commits, preserve snapshotId, set baseRef to an empty string, and do not ` +
      `copy repository contents.\n\n` +
      `The hidden oracle remains in the controller-side scenario object for scoring but must not be duplicated into ` +
      `any repository file, record, trajectory, conversation turn, or task. generation must preserve the blueprint ` +
      `seed and identify codex/gpt-5.6-luna.\n\nBlueprint index: ${index}\n` +
      `${JSON.stringify(blueprint, null, 2)}`,
    { schema: SCENARIO_SCHEMA, key: `render-v5:${blueprint.id || index}` },
  )),
)

phase('Verify')
const reviews = await parallel(
  rendered.map((scenario, index) => () => {
    if (!scenario) return null
    const blueprint = args.blueprints[index]
    return approvedAgent(
      'verifier',
      `Try to reject this rendered ClankSpace evaluation world before it is materialized. This is offline ` +
        `verification in a neutral workspace. Do not invoke ClankSpace, inspect project instructions, scan the ` +
        `filesystem, or rewrite the scenario. Return only a verdict and issues; the controller retains the ` +
        `renderer object unchanged.\n\n` +
        `The blueprint and oracle are hidden controller data. Their presence below is not leakage. Reject when ` +
        `their conclusions appear in agent-visible repository content, records, trajectories, conversation, or ` +
        `task wording. Reject changed facts or provenance, missing paths, impossible behavior, hollow placeholder ` +
        `code, incoherent commit history, a task unsupported by the materialized repository, privacy violations, ` +
        `assistant turns, canonical-instruction framing, or distractor prose that announces its irrelevance. ` +
        `For proceed scenarios, ordinary independently solvable work is intentional and should remain out of the ` +
        `way. Do not repair defects. accepted must be false if any error exists.\n\n` +
        `Blueprint index: ${index}\nFIXED BLUEPRINT:\n${JSON.stringify(blueprint, null, 2)}\n\n` +
        `RENDERED SCENARIO:\n${JSON.stringify(scenario, null, 2)}`,
      { schema: REVIEW_SCHEMA, key: `verify-render-v5:${blueprint.id || index}` },
    )
  }),
)

const finalized = rendered.map((scenario, index) => {
  const review = reviews[index]
  const deterministicIssues = controllerIssues(scenario, args.blueprints[index])
  const issues = [...(review?.issues || []), ...deterministicIssues]
  return {
    scenarioId: scenario?.id || args.blueprints[index].id,
    accepted: Boolean(scenario && review?.accepted && !issues.some(issue => issue.severity === 'error')),
    scenario,
    issues,
  }
})
const accepted = finalized.filter(result => result.accepted).map(result => result.scenario)
const rejected = finalized.filter(result => !result.accepted)
const missing = rendered.reduce((count, scenario, index) => count + (!scenario || !reviews[index] ? 1 : 0), 0)
log(`Corpus render complete: accepted=${accepted.length}, rejected=${rejected.length}, missing=${missing}`)

return { workforceId: WORKFORCE_ID, accepted, rejected, missing }
