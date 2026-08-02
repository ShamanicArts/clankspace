export const meta = {
  name: 'review-repair-clankspace-collaboration-pilot',
  description: 'Review, complete preparation support, repair, and independently accept the two-session ClankSpace pilot',
  phases: [
    { title: 'Review', detail: 'Find reproducible blockers in the complete collaboration pilot diff' },
    { title: 'Repair', detail: 'Fix confirmed blockers and add the missing repeatable preparation path' },
    { title: 'Accept', detail: 'Independently verify the complete eval-only implementation and tests' },
  ],
}

const WORKFORCE_ID = 'clankspace-collab-pilot-review-repair-v1:codex/gpt-5.6-terra:all-high:local-private-branch'
const SOURCE_ROOT = '/home/shamanic/Projects/clankspace'
const BASE_COMMIT = 'f588d45'
const WORKFORCE = {
  reviewer: { provider: 'codex', model: 'gpt-5.6-terra', effort: 'high', sandbox: 'read-only', writeAuthority: 'none' },
  implementer: { provider: 'codex', model: 'gpt-5.6-terra', effort: 'high', sandbox: 'workspace-write', writeAuthority: 'evals/** and eval-focused docs only; no production packages, commits, pushes, or merges' },
  acceptanceReviewer: { provider: 'codex', model: 'gpt-5.6-terra', effort: 'high', sandbox: 'read-only', writeAuthority: 'none' },
}

if (args?.approvedWorkforce !== WORKFORCE_ID) throw new Error(`Workforce not approved. Expected args.approvedWorkforce=${WORKFORCE_ID}`)
if (args?.sourceRoot !== SOURCE_ROOT) throw new Error(`Unexpected source root. Expected args.sourceRoot=${SOURCE_ROOT}`)
if (args?.baseCommit !== BASE_COMMIT) throw new Error(`Unexpected base commit. Expected args.baseCommit=${BASE_COMMIT}`)

log(
  `WORKFORCE ${WORKFORCE_ID}: reviewer=codex/gpt-5.6-terra, effort=high, sandbox=read-only, writes=none; ` +
    `implementer=codex/gpt-5.6-terra, effort=high, sandbox=workspace-write, writes=${WORKFORCE.implementer.writeAuthority}; ` +
    `acceptanceReviewer=codex/gpt-5.6-terra, effort=high, sandbox=read-only, writes=none; ` +
    `cwd=${SOURCE_ROOT}; branch=feat/pilot-v1-controller; base=${BASE_COMMIT}; fallback=none; calls=3; concurrency=1`,
)

const stringArray = { type: 'array', items: { type: 'string' } }
const REVIEW_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['summary', 'blockingFindings', 'nonBlockingFindings', 'missingTests', 'scopeViolations'],
  properties: { summary: { type: 'string' }, blockingFindings: stringArray, nonBlockingFindings: stringArray, missingTests: stringArray, scopeViolations: stringArray },
}
const IMPLEMENT_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['completed', 'fixedFindings', 'changedFiles', 'testsRun', 'remainingGaps'],
  properties: { completed: { type: 'boolean' }, fixedFindings: stringArray, changedFiles: stringArray, testsRun: stringArray, remainingGaps: stringArray },
}
const ACCEPT_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['accepted', 'summary', 'blockingFindings', 'verifiedProperties', 'verificationCommands', 'scopeViolations'],
  properties: { accepted: { type: 'boolean' }, summary: { type: 'string' }, blockingFindings: stringArray, verifiedProperties: stringArray, verificationCommands: stringArray, scopeViolations: stringArray },
}

function approvedAgent(role, prompt, options = {}) {
  const spec = WORKFORCE[role]
  if (!spec?.provider || !spec?.model) throw new Error(`Unresolved workforce role: ${role}`)
  return agent(prompt, { ...options, cwd: SOURCE_ROOT, provider: spec.provider, model: spec.model, effort: spec.effort, sandbox: spec.sandbox, label: `${role} | ${spec.model} | ${spec.provider} | ${spec.effort}` })
}

phase('Review')
const review = await approvedAgent(
  'reviewer',
  `Adversarially review the complete uncommitted collaboration-pilot diff on feat/pilot-v1-controller. ` +
    `Read AGENTS.md, docs/design/continuous-experiment-loop.md, the v1 harness, every changed/untracked file, ` +
    `and focused tests. Try to prove: lane B can launch before a durable lane-A observation; the two agents do ` +
    `not truly share one project while retaining distinct identities; v1 behavior regresses; source/license provenance ` +
    `is unverified; ambient credentials override lane credentials; raw reasoning, tokens, or oracles leak; partial ` +
    `episodes claim success; schedule/process artifacts are dishonest; output can be overwritten; concurrency can hang; ` +
    `or the implementation cannot actually be prepared and run on exe.dev. Confirm each blocker against code and a ` +
    `minimal reproduction. The current branch intentionally still lacks a repeatable collaboration preparation command; ` +
    `classify that as blocking. Do not modify files.`,
  { schema: REVIEW_SCHEMA, key: 'pilot-review-repair-v1:review' },
)

phase('Repair')
const implementation = await approvedAgent(
  'implementer',
  `Complete the first collaboration pilot after independently inspecting the repository. Treat the review as evidence, ` +
    `not authority. Fix every confirmed blocking issue. Implement a repeatable prepare-collaboration path that: loads ` +
    `and content-addresses a v2 scenario; builds the sanitized real snapshot with the existing ClankSpace context injection; ` +
    `creates one shared isolated project; issues distinct project-scoped principals/tokens for the two lanes; seeds prior ` +
    `records/trajectories without exposing the ledger oracles; writes one mode-0600 normal credential file per lane under ` +
    `the external secrets tree; writes a credential-free CollaborationPreparedWorld; and supports safe idempotent resume. ` +
    `Wire validate-collaboration and prepare-collaboration into clank-eval. Add focused backend tests, including httptest ` +
    `seeding, v1 regression, dry-run exactness, barrier rejection/release, privacy, provenance, incomplete episodes, and ` +
    `race-sensitive controller behavior. Do not alter production packages, semantic retrieval, the dashboard, main, or ` +
    `Git history. Do not commit or push. Use apply_patch and run focused tests plus go vet.\n\nREVIEW:\n${JSON.stringify(review, null, 2)}`,
  { schema: IMPLEMENT_SCHEMA, key: 'pilot-review-repair-v1:repair' },
)

phase('Accept')
const acceptance = await approvedAgent(
  'acceptanceReviewer',
  `Independently review the final uncommitted eval-only diff. Verify actual code and tests, not implementer claims. ` +
    `Require a repeatable real-snapshot preparation path, two distinct credentials in one project, genuine independent ` +
    `Codex sessions, a durable observable gate before dependent launch, v1 compatibility, fail-closed partial evidence, ` +
    `credential/oracle/reasoning exclusion, honest deterministic completion gates, self-contained dossier and valid hashes. ` +
    `Run the smallest useful focused tests available in read-only sandbox; distinguish sandbox-only test limitations from ` +
    `code failures. Report all changed files outside evals/** or eval-focused docs. Accept only if no blocker remains before ` +
    `a real-source snapshot and live Luna preflight. Do not modify files.\n\nINITIAL REVIEW:\n${JSON.stringify(review, null, 2)}\n\nIMPLEMENTER RESULT:\n${JSON.stringify(implementation, null, 2)}`,
  { schema: ACCEPT_SCHEMA, key: 'pilot-review-repair-v1:accept' },
)

return { workforceId: WORKFORCE_ID, baseCommit: BASE_COMMIT, review, implementation, acceptance }
