export const meta = {
  name: 'generate-clankspace-blueprints',
  description: 'Generate fixed scenario blueprints from explicit curriculum cells before any world is rendered',
  phases: [
    { title: 'Design', detail: 'Create fixed facts and hidden oracles for explicit curriculum cells' },
    { title: 'Verify', detail: 'Reject ambiguous, leaky, or untestable blueprints before rendering' },
  ],
}

const WORKFORCE_ID = 'clankspace-blueprints-v1:codex/gpt-5.6-luna:literal-high-max'
const WORKFORCE = {
  architect: { provider: 'codex', model: 'gpt-5.6-luna', effort: 'high', sandbox: 'read-only', writeAuthority: 'none' },
  verifier: { provider: 'codex', model: 'gpt-5.6-luna', effort: 'max', sandbox: 'read-only', writeAuthority: 'none' },
}

if (args?.approvedWorkforce !== WORKFORCE_ID) {
  throw new Error(`Workforce not approved. Expected args.approvedWorkforce=${WORKFORCE_ID}`)
}
if (!Array.isArray(args?.curriculumCells) || args.curriculumCells.length === 0) {
  throw new Error('args.curriculumCells must be a non-empty array fixed by the curriculum controller')
}

log(
  `WORKFORCE ${WORKFORCE_ID}: architect=${WORKFORCE.architect.provider}/${WORKFORCE.architect.model}, ` +
    `effort=${WORKFORCE.architect.effort}, sandbox=read-only, writes=none; ` +
    `verifier=${WORKFORCE.verifier.provider}/${WORKFORCE.verifier.model}, ` +
    `effort=${WORKFORCE.verifier.effort}, sandbox=read-only, writes=none; fallback=none`,
)

const BLUEPRINT_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: [
    'schemaVersion', 'id', 'split', 'category', 'repositoryProfile', 'actors',
    'fixedFacts', 'fixedOracle', 'repositoryRequirements', 'conversationRequirements',
    'taskConstraints', 'curriculumAxes', 'generation',
  ],
  properties: {
    schemaVersion: { const: 1 },
    id: { type: 'string' },
    split: { enum: ['train', 'dev', 'holdout'] },
    category: { type: 'string' },
    repositoryProfile: { enum: ['fake', 'real-snapshot'] },
    actors: { type: 'array', items: { type: 'object' } },
    fixedFacts: { type: 'object' },
    fixedOracle: { type: 'object' },
    repositoryRequirements: { type: 'object' },
    conversationRequirements: { type: 'object' },
    taskConstraints: { type: 'object' },
    curriculumAxes: { type: 'array', items: { type: 'string' } },
    generation: { type: 'object' },
  },
}

const REVIEW_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['blueprintId', 'accepted', 'blueprint', 'issues'],
  properties: {
    blueprintId: { type: 'string' },
    accepted: { type: 'boolean' },
    blueprint: BLUEPRINT_SCHEMA,
    issues: {
      type: 'array',
      items: {
        type: 'object',
        additionalProperties: false,
        required: ['severity', 'code', 'detail'],
        properties: {
          severity: { enum: ['warning', 'error'] },
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
    provider: spec.provider,
    model: spec.model,
    effort: spec.effort,
    sandbox: spec.sandbox,
    label: `${role} | ${spec.model} | ${spec.provider} | ${spec.effort}`,
  })
}

const reviewed = await pipeline(
  args.curriculumCells,
  (cell, _original, index) => {
    phase('Design')
    return approvedAgent(
      'architect',
      `Design one fixed ClankSpace evaluation blueprint for the explicit curriculum cell below.\n\n` +
        `Choose all material facts and the hidden oracle now, before any repository, record prose, prior ` +
        `user turns, or final task is rendered. Make the oracle decidable and identify exact relevant aliases. ` +
        `The tested agent must later be able to succeed from repository evidence plus ClankSpace, but the ` +
        `agent-visible world must not contain the answer. Include deliberate distractors and provenance. ` +
        `For real-snapshot cells, preserve the supplied snapshot ID and constrain synthetic overlays to its ` +
        `actual architecture. Do not include credentials, private personal material, raw messages, insults, ` +
        `or hidden reasoning.\n\nCell index: ${index}\n${JSON.stringify(cell, null, 2)}`,
      { schema: BLUEPRINT_SCHEMA, key: `blueprint:${cell.id || index}` },
    )
  },
  (blueprint, cell, index) => {
    phase('Verify')
    return approvedAgent(
      'verifier',
      `Try to reject this blueprint before it enters the immutable corpus.\n\n` +
        `Reject ambiguous or self-contradictory truth, an expected behavior not supported by fixed facts, ` +
        `agent-visible oracle leakage, impossible repository requirements, weak distractors, privacy issues, ` +
        `project-boundary leakage, or a scenario that can be passed without exercising ClankSpace. Do not ` +
        `repair it. Return the blueprint unchanged with precise issues.\n\n` +
        `Cell index: ${index}\nFIXED CELL:\n${JSON.stringify(cell, null, 2)}\n\n` +
        `CANDIDATE BLUEPRINT:\n${JSON.stringify(blueprint, null, 2)}`,
      { schema: REVIEW_SCHEMA, key: `verify-blueprint:${cell.id || index}` },
    )
  },
)

const completed = reviewed.filter(Boolean)
return {
  workforceId: WORKFORCE_ID,
  accepted: completed.filter(result => result.accepted).map(result => result.blueprint),
  rejected: completed.filter(result => !result.accepted),
  missing: reviewed.length - completed.length,
}
