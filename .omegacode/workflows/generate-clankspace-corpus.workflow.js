export const meta = {
  name: 'generate-clankspace-corpus',
  description: 'Render fixed ClankSpace evaluation blueprints and independently verify their fidelity',
  phases: [
    { title: 'Render', detail: 'Turn fixed scenario blueprints into realistic project records and user turns' },
    { title: 'Verify', detail: 'Check oracle fidelity, privacy, materiality, and evaluation leakage' },
  ],
}

const WORKFORCE_ID = 'clankspace-worlds-v4:codex/gpt-5.6-luna:literal-high-max'
const WORKFORCE = {
  renderer: {
    provider: 'codex',
    model: 'gpt-5.6-luna',
    effort: 'high',
    sandbox: 'read-only',
    writeAuthority: 'none',
  },
  verifier: {
    provider: 'codex',
    model: 'gpt-5.6-luna',
    effort: 'max',
    sandbox: 'read-only',
    writeAuthority: 'none',
  },
}

if (args?.approvedWorkforce !== WORKFORCE_ID) {
  throw new Error(`Workforce not approved. Expected args.approvedWorkforce=${WORKFORCE_ID}`)
}
if (!Array.isArray(args?.blueprints) || args.blueprints.length === 0) {
  throw new Error('args.blueprints must be a non-empty array of fixed scenario blueprints')
}

log(
  `WORKFORCE ${WORKFORCE_ID}: ` +
    `renderer=${WORKFORCE.renderer.provider}/${WORKFORCE.renderer.model}, ` +
    `effort=${WORKFORCE.renderer.effort}, sandbox=${WORKFORCE.renderer.sandbox}, writes=none; ` +
    `verifier=${WORKFORCE.verifier.provider}/${WORKFORCE.verifier.model}, ` +
    `effort=${WORKFORCE.verifier.effort}, sandbox=${WORKFORCE.verifier.sandbox}, writes=none; ` +
    `fallback=none`,
)

const SCENARIO_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: [
    'schemaVersion',
    'id',
    'split',
    'category',
    'project',
    'repository',
    'actors',
    'records',
    'trajectories',
    'conversation',
    'task',
    'oracle',
    'generation',
  ],
  properties: {
    schemaVersion: { const: 1 },
    id: { type: 'string' },
    split: { enum: ['train', 'dev', 'holdout'] },
    category: { type: 'string' },
    project: { type: 'object' },
    repository: { type: 'object' },
    actors: { type: 'array', items: { type: 'object' } },
    records: { type: 'array', items: { type: 'object' } },
    trajectories: { type: 'array', items: { type: 'object' } },
    conversation: { type: 'array', items: { type: 'object' } },
    task: { type: 'object' },
    oracle: { type: 'object' },
    generation: { type: 'object' },
  },
}

const REVIEW_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['scenarioId', 'accepted', 'scenario', 'issues'],
  properties: {
    scenarioId: { type: 'string' },
    accepted: { type: 'boolean' },
    scenario: SCENARIO_SCHEMA,
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

const results = await pipeline(
  args.blueprints,
  (blueprint, _original, index) => {
    phase('Render')
    return approvedAgent(
      'renderer',
      `Render one ClankSpace evaluation scenario from the fixed blueprint below.\n\n` +
        `You are a corpus renderer, not the policy under evaluation. Preserve every fixed fact, ` +
        `record ID, lifecycle state, relevant-record set, expected behavior, and checkpoint oracle exactly. ` +
        `Do not improve or reinterpret the oracle after writing the conversation. Make the repository, ` +
        `records, prior user turns, and a physical Git commit history realistic enough to test an agent, ` +
        `but keep them concise. Repository commits must create runnable or inspectable source files and may ` +
        `not write AGENTS.md, .agents/, .clankspace.json, .git/, credentials, or oracle hints. Do not include ` +
        `hidden reasoning, raw private messages, credentials, insults, health information, or evaluation hints ` +
        `in text visible to the tested agent. Professional paraphrase is required. Set ` +
        `generation.generatorProvider to codex and generation.generatorModel to gpt-5.6-luna.\n\n` +
        `Blueprint index: ${index}\n` +
        `${JSON.stringify(blueprint, null, 2)}`,
      { schema: SCENARIO_SCHEMA, key: `render:${blueprint.id || index}` },
    )
  },
  (scenario, blueprint, index) => {
    phase('Verify')
    return approvedAgent(
      'verifier',
      `Independently audit this generated ClankSpace evaluation scenario against its fixed blueprint.\n\n` +
        `Reject it if it changes the oracle, leaks the expected answer into agent-visible text, contains ` +
        `private or emotionally sensational material, uses irrelevant transcript-like detail, makes records ` +
        `canonical instructions rather than advisory evidence, or is too artificial to evaluate coordination. ` +
        `Also reject missing provenance, invalid cross-references, fake assistant turns, a repository that ` +
        `cannot physically support the task, or a task that cannot plausibly trigger the blueprint's expected ` +
        `behavior. Do not silently repair a rejected item. Return the scenario unchanged ` +
        `with a precise issue list.\n\n` +
        `Blueprint index: ${index}\n` +
        `FIXED BLUEPRINT:\n${JSON.stringify(blueprint, null, 2)}\n\n` +
        `GENERATED SCENARIO:\n${JSON.stringify(scenario, null, 2)}`,
      { schema: REVIEW_SCHEMA, key: `verify:${blueprint.id || index}` },
    )
  },
)

const completed = results.filter(Boolean)
const accepted = completed.filter(result => result.accepted)
const rejected = completed.filter(result => !result.accepted)
log(`Corpus batch complete: accepted=${accepted.length}, rejected=${rejected.length}, missing=${results.length - completed.length}`)

return {
  workforceId: WORKFORCE_ID,
  accepted: accepted.map(result => result.scenario),
  rejected,
}
