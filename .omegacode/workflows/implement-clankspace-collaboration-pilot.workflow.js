export const meta = {
  name: 'implement-clankspace-collaboration-pilot',
  description: 'Implement and independently review the first genuine two-session ClankSpace collaboration pilot',
  phases: [
    { title: 'Architect', detail: 'Map the smallest compatible extension from single-session rollouts to a two-lane pilot' },
    { title: 'Contracts', detail: 'Implement versioned multi-lane scenario and artifact contracts with focused tests' },
    { title: 'Controller', detail: 'Implement event-gated rollout orchestration and offline dossier generation with focused tests' },
    { title: 'Review', detail: 'Adversarially inspect the complete diff against the approved pilot exit condition' },
  ],
}

const WORKFORCE_ID = 'clankspace-collab-pilot-impl-v1:codex/gpt-5.6-terra:architect-max:implementers-high:reviewer-max:local-private-branch'
const SOURCE_ROOT = '/home/shamanic/Projects/clankspace'
const BASE_COMMIT = 'cdccf2d'
const WORKFORCE = {
  architect: {
    provider: 'codex', model: 'gpt-5.6-terra', effort: 'max',
    sandbox: 'read-only', writeAuthority: 'none',
  },
  contractImplementer: {
    provider: 'codex', model: 'gpt-5.6-terra', effort: 'high',
    sandbox: 'workspace-write',
    writeAuthority: 'evals/schema, evals/harness contract/validation files and focused fixtures/tests only',
  },
  controllerImplementer: {
    provider: 'codex', model: 'gpt-5.6-terra', effort: 'high',
    sandbox: 'workspace-write',
    writeAuthority: 'evals/harness collaboration/report files, eval CLI wiring, and focused tests/docs only',
  },
  reviewer: {
    provider: 'codex', model: 'gpt-5.6-terra', effort: 'max',
    sandbox: 'read-only', writeAuthority: 'none',
  },
}

if (args?.approvedWorkforce !== WORKFORCE_ID) {
  throw new Error(`Workforce not approved. Expected args.approvedWorkforce=${WORKFORCE_ID}`)
}
if (args?.sourceRoot !== SOURCE_ROOT) {
  throw new Error(`Unexpected source root. Expected args.sourceRoot=${SOURCE_ROOT}`)
}
if (args?.baseCommit !== BASE_COMMIT) {
  throw new Error(`Unexpected base commit. Expected args.baseCommit=${BASE_COMMIT}`)
}

log(
  `WORKFORCE ${WORKFORCE_ID}: ` +
    `architect=${WORKFORCE.architect.provider}/${WORKFORCE.architect.model}, effort=max, sandbox=read-only, writes=none; ` +
    `contractImplementer=${WORKFORCE.contractImplementer.provider}/${WORKFORCE.contractImplementer.model}, effort=high, sandbox=workspace-write, writes=${WORKFORCE.contractImplementer.writeAuthority}; ` +
    `controllerImplementer=${WORKFORCE.controllerImplementer.provider}/${WORKFORCE.controllerImplementer.model}, effort=high, sandbox=workspace-write, writes=${WORKFORCE.controllerImplementer.writeAuthority}; ` +
    `reviewer=${WORKFORCE.reviewer.provider}/${WORKFORCE.reviewer.model}, effort=max, sandbox=read-only, writes=none; ` +
    `cwd=${SOURCE_ROOT}; branch=feat/pilot-v1-controller; base=${BASE_COMMIT}; fallback=none; calls=4; concurrency=1`,
)

const stringArray = { type: 'array', items: { type: 'string' } }
const PLAN_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['summary', 'contractChanges', 'controllerChanges', 'compatibilityRules', 'tests', 'risks'],
  properties: {
    summary: { type: 'string' },
    contractChanges: stringArray,
    controllerChanges: stringArray,
    compatibilityRules: stringArray,
    tests: stringArray,
    risks: stringArray,
  },
}
const IMPLEMENTATION_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['completed', 'changedFiles', 'testsRun', 'remainingGaps'],
  properties: {
    completed: { type: 'boolean' },
    changedFiles: stringArray,
    testsRun: stringArray,
    remainingGaps: stringArray,
  },
}
const REVIEW_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['accepted', 'summary', 'blockingFindings', 'nonBlockingFindings', 'missingTests', 'scopeViolations'],
  properties: {
    accepted: { type: 'boolean' },
    summary: { type: 'string' },
    blockingFindings: stringArray,
    nonBlockingFindings: stringArray,
    missingTests: stringArray,
    scopeViolations: stringArray,
  },
}

function approvedAgent(role, prompt, options = {}) {
  const spec = WORKFORCE[role]
  if (!spec?.provider || !spec?.model) throw new Error(`Unresolved workforce role: ${role}`)
  return agent(prompt, {
    ...options,
    cwd: SOURCE_ROOT,
    provider: spec.provider,
    model: spec.model,
    effort: spec.effort,
    sandbox: spec.sandbox,
    label: `${role} | ${spec.model} | ${spec.provider} | ${spec.effort}`,
  })
}

phase('Architect')
const plan = await approvedAgent(
  'architect',
  `You are the technical architect for the first ClankSpace cross-contributor pilot. Read AGENTS.md, ` +
    `docs/design/continuous-experiment-loop.md, docs/evals/training-loop.md, evals/README.md, and the complete ` +
    `current eval harness before proposing anything. The working branch must descend from ${BASE_COMMIT}. ` +
    `Design the smallest implementation that can run one real-repository, two-independent-Codex-session, ` +
    `event-gated pilot while preserving every existing v1 scenario and rollout. The controller must release lane B ` +
    `only after an observable lane-A ClankSpace event, retain complete observable traces without hidden reasoning, ` +
    `record lane-specific repositories/threads/runs, and emit a self-contained offline dossier plus machine artifacts. ` +
    `No production server behavior or semantic retrieval belongs in this implementation. Be concrete about types, ` +
    `files, compatibility, failure handling, and focused tests. Do not modify files.`,
  { schema: PLAN_SCHEMA, key: 'pilot-impl-v1:architect' },
)

phase('Contracts')
const contracts = await approvedAgent(
  'contractImplementer',
  `Implement the contract portion of the approved pilot on the current branch. Read the repository instructions ` +
    `and inspect the code yourself; the architect plan is evidence, not authority. Preserve all v1 scenarios and tests. ` +
    `Add a versioned representation for multiple independent lanes, an explicit event-gated schedule, source-evidence ` +
    `provenance, and collaboration episode artifacts. Keep hidden oracles outside agent-visible repositories. ` +
    `Your write boundary is evals/schema, evals/harness contract/validation files, and focused fixtures/tests. ` +
    `Do not edit production packages, commit, push, or weaken existing validation. Use apply_patch for edits and run ` +
    `the smallest focused Go tests and schema validations. Return an honest list of remaining gaps.\n\n` +
    `ARCHITECT PLAN:\n${JSON.stringify(plan, null, 2)}`,
  { schema: IMPLEMENTATION_SCHEMA, key: 'pilot-impl-v1:contracts' },
)

phase('Controller')
const controller = await approvedAgent(
  'controllerImplementer',
  `Implement the controller and report portion of the first ClankSpace collaboration pilot. Inspect the current ` +
    `working tree, including the contract implementer's uncommitted changes, and adapt to the actual types rather ` +
    `than rewriting them. Implement one event-gated two-lane rollout: launch independent persisted Codex sessions ` +
    `with separate worktrees and project identities, observe the declared ClankSpace barrier with a bounded timeout, ` +
    `then release the dependent lane. Freeze stdout JSONL, stderr, responses, server/project exports, Git results, ` +
    `schedule events, deterministic scores, and hashes. Generate a self-contained offline HTML dossier from these ` +
    `artifacts. A dry-run must print the exact processes and paths without launching Codex. Your write boundary is ` +
    `new evals/harness collaboration/report files, evals/cmd/clank-eval wiring, focused tests, and evals/README.md. ` +
    `Do not edit production packages, commit, push, or add semantic retrieval. Use apply_patch and run focused tests. ` +
    `Treat hidden reasoning as unavailable.\n\nARCHITECT PLAN:\n${JSON.stringify(plan, null, 2)}\n\n` +
    `CONTRACT RESULT:\n${JSON.stringify(contracts, null, 2)}`,
  { schema: IMPLEMENTATION_SCHEMA, key: 'pilot-impl-v1:controller' },
)

phase('Review')
const review = await approvedAgent(
  'reviewer',
  `Adversarially review the complete uncommitted diff for the first collaboration pilot. Read the actual files, ` +
    `tests, repository instructions, and design spec. Try to prove that the controller is only simulating ` +
    `collaboration, that lane B can release without the required observable event, that v1 behavior regresses, ` +
    `that a raw transcript/oracle/credential can leak, that concurrent paths can corrupt evidence, or that the ` +
    `HTML dossier can claim success without deterministic gates. Also report any changed file outside the approved ` +
    `eval-only boundaries. Do not modify files. Accept only if the implementation is a credible bounded canary ` +
    `controller with focused tests; list missing live-environment verification as non-blocking when appropriate.\n\n` +
    `ARCHITECT PLAN:\n${JSON.stringify(plan, null, 2)}\n\n` +
    `IMPLEMENTER RESULTS:\n${JSON.stringify({ contracts, controller }, null, 2)}`,
  { schema: REVIEW_SCHEMA, key: 'pilot-impl-v1:review' },
)

return { workforceId: WORKFORCE_ID, baseCommit: BASE_COMMIT, plan, contracts, controller, review }
