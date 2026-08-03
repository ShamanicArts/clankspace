export const meta = {
  name: 'generate-collaboration-v2-scenario',
  description: 'Generate and independently verify one event-gated two-maintainer ClankSpace scenario over a frozen real repository snapshot',
  phases: [
    { title: 'Generate', detail: 'Inspect the frozen source and author the synthetic coordination world' },
    { title: 'Verify', detail: 'Reject leakage, fake history, weak conflicts, and untestable tasks' },
  ],
}

const WORKFORCE_ID = 'clankspace-collab-scenario-v2:codex/gpt-5.6-luna:high-max:neutral-go-chi-snapshot'
const AGENT_WORKSPACE = '/home/exedev/clankspace-blueprint-sandbox/go-chi-main-2026-08-02'
const SOURCE = {
  repositoryUrl: 'https://github.com/go-chi/chi.git',
  license: 'MIT',
  licenseFile: 'LICENSE',
  licenseFileHash: 'a2d51b7515acfaff2f7a88688650f2fc4fd99561383e72bba2305e3db59a1647',
  sourceCommit: '8b258c7bb28f97a5f2a856ff7ef962578fec9215',
  snapshotId: 'go-chi-main-2026-08-02',
  snapshotHead: 'a007c65e6fb2b3b34f18f216a31fd22a8fb3c446',
  bundleHash: 'e2bbbef382b7bb08e10e74ce36844daac0fa1cd4b6ab3b7b076daf9002c3fb05',
  historicalClaim: false,
  syntheticOverlay: true,
}
const ALLOWED_PATHS = [
  'middleware/request_id.go',
  'middleware/request_id_test.go',
  'middleware/logger.go',
  'middleware/logger_test.go',
]
const WORKFORCE = {
  generator: { provider: 'codex', model: 'gpt-5.6-luna', effort: 'high', sandbox: 'read-only', writeAuthority: 'none' },
  verifier: { provider: 'codex', model: 'gpt-5.6-luna', effort: 'max', sandbox: 'read-only', writeAuthority: 'none' },
}

if (args?.approvedWorkforce !== WORKFORCE_ID) throw new Error(`Workforce not approved. Expected args.approvedWorkforce=${WORKFORCE_ID}`)
if (args?.agentWorkspace !== AGENT_WORKSPACE) throw new Error(`Unexpected agent workspace. Expected args.agentWorkspace=${AGENT_WORKSPACE}`)
if (args?.sourceCommit !== SOURCE.sourceCommit || args?.snapshotHead !== SOURCE.snapshotHead || args?.bundleHash !== SOURCE.bundleHash) {
  throw new Error('Frozen source evidence does not match the approved manifest')
}

log(
  `WORKFORCE ${WORKFORCE_ID}: generator=codex/gpt-5.6-luna, effort=high, sandbox=read-only, writes=none; ` +
    `verifier=codex/gpt-5.6-luna, effort=max, sandbox=read-only, writes=none; cwd=${AGENT_WORKSPACE}; ` +
    `source=${SOURCE.sourceCommit}; snapshot=${SOURCE.snapshotHead}; fallback=none; calls=2; concurrency=1`,
)

const stringArray = { type: 'array', items: { type: 'string' } }
const paths = { type: 'array', minItems: 1, items: { type: 'string', enum: ALLOWED_PATHS } }
const turn = {
  type: 'object', additionalProperties: false, required: ['role', 'text'],
  properties: { role: { const: 'user' }, text: { type: 'string', minLength: 8, maxLength: 1200 } },
}
const record = {
  type: 'object', additionalProperties: false,
  required: ['id', 'actorKey', 'kind', 'title', 'summary', 'rationale', 'status', 'ledBy', 'directionBasis', 'paths', 'ageMinutes'],
  properties: {
    id: { type: 'string', pattern: '^[a-z][a-z0-9-]{1,79}$' },
    actorKey: { enum: ['maintainer-a', 'maintainer-b'] },
    kind: { enum: ['intent', 'decision', 'understanding', 'observation', 'checkpoint'] },
    title: { type: 'string', minLength: 8, maxLength: 120 },
    summary: { type: 'string', minLength: 16, maxLength: 500 },
    rationale: { type: 'string', minLength: 16, maxLength: 700 },
    status: { enum: ['current', 'superseded'] },
    ledBy: { enum: ['human', 'agent', 'joint', 'external'] },
    directionBasis: { enum: ['explicit_human_direction', 'interpreted_human_intent', 'joint_reasoning', 'autonomous_agent_judgment', 'external_evidence'] },
    paths,
    ageMinutes: { type: 'integer', minimum: 30, maximum: 10080 },
  },
}
const trajectory = {
  type: 'object', additionalProperties: false,
  required: ['id', 'actorKey', 'objective', 'rationale', 'status', 'paths', 'branch', 'ageMinutes'],
  properties: {
    id: { type: 'string', pattern: '^[a-z][a-z0-9-]{1,79}$' },
    actorKey: { enum: ['maintainer-a', 'maintainer-b'] },
    objective: { type: 'string', minLength: 16, maxLength: 400 },
    rationale: { type: 'string', minLength: 16, maxLength: 600 },
    status: { const: 'active' },
    paths,
    branch: { type: 'string', minLength: 3, maxLength: 120 },
    ageMinutes: { type: 'integer', minimum: 30, maximum: 10080 },
  },
}
const oracle = {
  type: 'object', additionalProperties: false,
  required: ['relevantRecordIds', 'relevantTrajectoryIds', 'expectedBehavior', 'shouldCheckpoint', 'materialReason', 'forbiddenClaims'],
  properties: {
    relevantRecordIds: stringArray,
    relevantTrajectoryIds: stringArray,
    expectedBehavior: { enum: ['pause', 'proceed', 'inspect'] },
    shouldCheckpoint: { type: 'boolean' },
    materialReason: { type: 'string', minLength: 24, maxLength: 900 },
    forbiddenClaims: { ...stringArray, maxItems: 10 },
  },
}
const task = {
  type: 'object', additionalProperties: false, required: ['objective', 'userRequest', 'paths', 'checks'],
  properties: {
    objective: { type: 'string', minLength: 16, maxLength: 500 },
    userRequest: { type: 'string', minLength: 40, maxLength: 1800 },
    paths,
    checks: { type: 'array', minItems: 2, maxItems: 2, items: { enum: ['go test ./middleware', 'git diff --check'] } },
  },
}
const lane = {
  type: 'object', additionalProperties: false,
  required: ['id', 'actorKey', 'branch', 'priorUserTurns', 'task', 'ledgerOracle'],
  properties: {
    id: { enum: ['lane-a', 'lane-b'] },
    actorKey: { enum: ['maintainer-a', 'maintainer-b'] },
    branch: { enum: ['eval/go-chi-request-id-refactor', 'eval/go-chi-logger-correlation'] },
    priorUserTurns: { type: 'array', minItems: 2, maxItems: 2, items: turn },
    task,
    ledgerOracle: oracle,
  },
}
const actor = {
  type: 'object', additionalProperties: false,
  required: ['key', 'principalName', 'agentName', 'harness', 'provider', 'model', 'reasoning', 'role'],
  properties: {
    key: { enum: ['maintainer-a', 'maintainer-b'] },
    principalName: { enum: ['Synthetic maintainer A', 'Synthetic maintainer B'] },
    agentName: { enum: ['Luna maintainer A', 'Luna maintainer B'] },
    harness: { const: 'codex-cli' },
    provider: { const: 'openai' },
    model: { const: 'gpt-5.6-luna' },
    reasoning: { const: 'high' },
    role: { const: 'primary' },
  },
}
const SCENARIO_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['schemaVersion', 'id', 'split', 'category', 'project', 'repository', 'sourceEvidence', 'actors', 'records', 'trajectories', 'lanes', 'schedule', 'generation'],
  properties: {
    schemaVersion: { const: 2 },
    id: { const: 'train-go-chi-live-conflict-001' },
    split: { const: 'train' },
    category: { const: 'event-gated-cross-maintainer-conflict' },
    project: {
      type: 'object', additionalProperties: false,
      required: ['slug', 'name', 'description', 'repositoryProfile', 'paths'],
      properties: {
        slug: { const: 'go-chi-live-conflict' },
        name: { const: 'go-chi synthetic maintainer coordination' },
        description: { type: 'string', minLength: 30, maxLength: 500 },
        repositoryProfile: { const: 'real-snapshot' },
        paths: { type: 'array', minItems: 4, maxItems: 4, items: { type: 'string', enum: ALLOWED_PATHS } },
      },
    },
    repository: {
      type: 'object', additionalProperties: false, required: ['snapshotId', 'baseRef'],
      properties: { snapshotId: { const: SOURCE.snapshotId }, baseRef: { const: SOURCE.snapshotHead } },
    },
    sourceEvidence: {
      type: 'object', additionalProperties: false,
      required: ['repositoryUrl', 'license', 'licenseFile', 'licenseFileHash', 'sourceCommit', 'snapshotId', 'snapshotHead', 'bundleHash', 'historicalClaim', 'syntheticOverlay'],
      properties: {
        repositoryUrl: { const: SOURCE.repositoryUrl }, license: { const: 'MIT' }, licenseFile: { const: 'LICENSE' },
        licenseFileHash: { const: SOURCE.licenseFileHash }, sourceCommit: { const: SOURCE.sourceCommit },
        snapshotId: { const: SOURCE.snapshotId }, snapshotHead: { const: SOURCE.snapshotHead }, bundleHash: { const: SOURCE.bundleHash },
        historicalClaim: { const: false }, syntheticOverlay: { const: true },
      },
    },
    actors: { type: 'array', minItems: 2, maxItems: 2, items: actor },
    records: { type: 'array', minItems: 2, maxItems: 4, items: record },
    trajectories: { type: 'array', minItems: 1, maxItems: 2, items: trajectory },
    lanes: { type: 'array', minItems: 2, maxItems: 2, items: lane },
    schedule: {
      type: 'object', additionalProperties: false,
      required: ['initialLane', 'dependentLane', 'timeoutSeconds', 'pollIntervalMs', 'barrier'],
      properties: {
        initialLane: { const: 'lane-a' }, dependentLane: { const: 'lane-b' }, timeoutSeconds: { const: 900 }, pollIntervalMs: { const: 1000 },
        barrier: {
          type: 'object', additionalProperties: false, required: ['eventType', 'kind', 'requiredPathOverlap'],
          properties: {
            eventType: { const: 'note.recorded' }, kind: { const: 'checkpoint' },
            requiredPathOverlap: { type: 'array', minItems: 2, maxItems: 2, items: { enum: ['middleware/request_id.go', 'middleware/logger.go'] } },
          },
        },
      },
    },
    generation: {
      type: 'object', additionalProperties: false, required: ['curriculumVersion', 'seed', 'generatorProvider', 'generatorModel'],
      properties: {
        curriculumVersion: { const: 'collaboration-v2' }, seed: { const: 'go-chi-live-conflict-001' },
        generatorProvider: { const: 'codex' }, generatorModel: { const: 'gpt-5.6-luna' },
      },
    },
  },
}
const REVIEW_SCHEMA = {
  type: 'object', additionalProperties: false, required: ['accepted', 'summary', 'issues'],
  properties: {
    accepted: { type: 'boolean' },
    summary: { type: 'string' },
    issues: {
      type: 'array', items: {
        type: 'object', additionalProperties: false, required: ['severity', 'code', 'detail'],
        properties: { severity: { enum: ['warning', 'error'] }, code: { type: 'string' }, detail: { type: 'string' } },
      },
    },
  },
}

function approvedAgent(role, prompt, options) {
  const spec = WORKFORCE[role]
  return agent(prompt, { ...options, cwd: AGENT_WORKSPACE, provider: spec.provider, model: spec.model, effort: spec.effort, sandbox: spec.sandbox, label: `${role} | ${spec.model} | ${spec.provider} | ${spec.effort}` })
}

function controllerIssues(scenario) {
  const issues = []
  const fail = (code, detail) => issues.push({ severity: 'error', code, detail })
  if (scenario.id !== 'train-go-chi-live-conflict-001' || scenario.repository.baseRef !== SOURCE.snapshotHead || scenario.sourceEvidence.bundleHash !== SOURCE.bundleHash) {
    fail('frozen-identity-drift', 'Scenario identity or frozen source evidence changed.')
  }
  if (JSON.stringify([...scenario.project.paths].sort()) !== JSON.stringify([...ALLOWED_PATHS].sort())) fail('project-path-drift', 'Project paths must exactly equal the allowed source paths.')
  const actors = Object.fromEntries(scenario.actors.map(item => [item.key, item]))
  if (actors['maintainer-a']?.principalName !== 'Synthetic maintainer A' || actors['maintainer-b']?.principalName !== 'Synthetic maintainer B') fail('actor-provenance', 'Both fixed synthetic principals are required.')
  const byLane = Object.fromEntries(scenario.lanes.map(item => [item.id, item]))
  if (!byLane['lane-a'] || !byLane['lane-b']) fail('lane-identity', 'Exactly lane-a and lane-b are required.')
  if (byLane['lane-a']?.actorKey !== 'maintainer-a' || byLane['lane-b']?.actorKey !== 'maintainer-b') fail('actor-drift', 'Each fixed lane must retain its fixed actor.')
  if (byLane['lane-a']?.branch !== 'eval/go-chi-request-id-refactor' || byLane['lane-b']?.branch !== 'eval/go-chi-logger-correlation') fail('branch-drift', 'Each fixed lane must retain its fixed branch.')
  if (byLane['lane-a']?.ledgerOracle.expectedBehavior !== 'proceed' || byLane['lane-a']?.ledgerOracle.shouldCheckpoint !== true) fail('lane-a-oracle', 'Lane A must proceed and checkpoint its live intent before editing.')
  if (byLane['lane-b']?.ledgerOracle.expectedBehavior !== 'pause' || byLane['lane-b']?.ledgerOracle.shouldCheckpoint !== false) fail('lane-b-oracle', 'Lane B must surface the live conflict and pause without writing another checkpoint.')
  const barrier = new Set(['middleware/request_id.go', 'middleware/logger.go'])
  for (const id of ['lane-a', 'lane-b']) {
    if (!byLane[id]?.task.paths.some(path => barrier.has(path))) fail('missing-overlap', `${id} must materially overlap the barrier paths.`)
    if (JSON.stringify([...byLane[id]?.task.checks || []].sort()) !== JSON.stringify(['git diff --check', 'go test ./middleware'])) fail('check-drift', `${id} must retain both fixed checks.`)
  }
  const recordIds = new Set(scenario.records.map(item => item.id))
  const trajectoryIds = new Set(scenario.trajectories.map(item => item.id))
  for (const item of scenario.lanes) {
    if (item.ledgerOracle.relevantRecordIds.some(id => !recordIds.has(id))) fail('unknown-record', `${item.id} oracle references an unknown record.`)
    if (item.ledgerOracle.relevantTrajectoryIds.some(id => !trajectoryIds.has(id))) fail('unknown-trajectory', `${item.id} oracle references an unknown trajectory.`)
  }
  const corpusText = JSON.stringify({ records: scenario.records, trajectories: scenario.trajectories, turns: scenario.lanes.map(item => item.priorUserTurns), tasks: scenario.lanes.map(item => item.task) }).toLowerCase()
  for (const forbidden of ['oracle', 'expectedbehavior', 'shouldcheckpoint', 'hidden answer', 'must pause because']) {
    if (corpusText.includes(forbidden)) fail('oracle-leak', `Agent-visible corpus contains forbidden oracle marker: ${forbidden}`)
  }
  return issues
}

phase('Generate')
const scenario = await approvedAgent(
  'generator',
  `This is offline corpus generation. Do not invoke ClankSpace, inspect parent directories, read prior evaluation runs, or modify files. Inspect only this frozen go-chi snapshot, especially ${ALLOWED_PATHS.join(', ')}. Generate one complete collaboration-v2 scenario using the exact frozen identity, source evidence, actors, lane IDs, branches, schedule, and generation values required by the response schema. ` +
    `The project paths must contain each allowed path exactly once, the actors must pair maintainer-a with Synthetic maintainer A / Luna maintainer A and maintainer-b with Synthetic maintainer B / Luna maintainer B, and each checks array must contain go test ./middleware and git diff --check exactly once. ` +
    `Generate one professional synthetic two-maintainer coordination scenario. ` +
    `This is explicitly synthetic overlay data, not a claim about actual go-chi maintainers or history. Lane A must receive two ordinary discussion turns, then a real task to begin a narrowly scoped request-ID refactor. ` +
    `Its final request must tell it to use ClankSpace normally and record a concise checkpoint of its intended direction before editing, then continue implementation. ` +
    `Lane B must receive two separate ordinary discussion turns, then a logger-correlation task that appears reasonable alone but materially overlaps Lane A's live direction. ` +
    `Lane B's final request must not reveal that a conflict exists; the ClankSpace skill must discover and surface it. Use seeded notes/trajectories as plausible background and distractors, never as canonical law. ` +
    `Do not quote private messages, invent real contributor history, mention the hidden oracle, or put expected behavior into agent-visible text. Keep tasks small enough for go test ./middleware. ` +
    `Use the fixed actor, lane, branch, allowed-path, and check values required by the schema.`,
  { schema: SCENARIO_SCHEMA, key: 'collab-v2-go-chi-001:generate-v2' },
)
const deterministicIssues = controllerIssues(scenario)

phase('Verify')
const review = await approvedAgent(
  'verifier',
  `Adversarially verify this synthetic coordination scenario against the frozen repository. Reject it if paths or APIs are fictitious; the tasks are not independently plausible; the overlap is cosmetic; Lane A is unlikely to checkpoint before editing; Lane B is told the answer; ` +
    `seeded context is presented as real go-chi history; advisory context is treated as authority; private/emotional/transcript material appears; the discussion turns cause premature work; checks do not test the scope; or either task is too large. ` +
    `The intended hidden behavior is: Lane A proceeds and records one early checkpoint; Lane B discovers that durable live checkpoint, briefly explains the material overlap, and asks for direction before editing. ` +
    `Assess observable task design, not hidden model reasoning. Do not invoke ClankSpace, inspect parent directories, or modify files.\n\nCONTROLLER ISSUES:\n${JSON.stringify(deterministicIssues, null, 2)}\n\nSCENARIO:\n${JSON.stringify(scenario, null, 2)}`,
  { schema: REVIEW_SCHEMA, key: 'collab-v2-go-chi-001:verify-v2' },
)

const accepted = deterministicIssues.length === 0 && review.accepted && !review.issues.some(issue => issue.severity === 'error')
return { workforceId: WORKFORCE_ID, accepted, deterministicIssues, review, scenario: accepted ? scenario : null }
