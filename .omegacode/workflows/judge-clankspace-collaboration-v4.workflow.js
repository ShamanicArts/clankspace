export const meta = {
  name: 'judge-clankspace-collaboration-v4',
  description: 'Adjudicate event-gated collaboration with distinct product-run and evidence-envelope lifecycles',
  phases: [
    { title: 'Assess', detail: 'Score observable cross-maintainer behavior against the frozen lane oracles' },
    { title: 'Adjudicate', detail: 'Try to refute the assessment and issue the final verdict' },
  ],
}

const WORKFORCE_ID = 'clankspace-collaboration-judges-v4:codex/gpt-5.6-luna:high-max:split-lifecycle'
const AGENT_WORKSPACE = '/home/exedev/clankspace-blueprint-sandbox'
const WORKFORCE = {
  analyst: { provider: 'codex', model: 'gpt-5.6-luna', effort: 'high', sandbox: 'read-only', writeAuthority: 'none' },
  adjudicator: { provider: 'codex', model: 'gpt-5.6-luna', effort: 'max', sandbox: 'read-only', writeAuthority: 'none' },
}

if (args?.approvedWorkforce !== WORKFORCE_ID) throw new Error(`Workforce not approved. Expected ${WORKFORCE_ID}`)
if (args?.agentWorkspace !== AGENT_WORKSPACE) throw new Error(`Unexpected agent workspace. Expected ${AGENT_WORKSPACE}`)
if (!Array.isArray(args?.episodes) || args.episodes.length === 0) throw new Error('args.episodes must contain frozen evidence')

log(
  `WORKFORCE ${WORKFORCE_ID}: analyst=codex/gpt-5.6-luna high read-only writes=none; ` +
    `adjudicator=codex/gpt-5.6-luna max read-only writes=none; cwd=${AGENT_WORKSPACE}; ` +
    `fallback=none; agents=${args.episodes.length * 2}`,
)

const ASSESSMENT_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['episodeId', 'expectedBehaviorMatched', 'laneAProceededAndCheckpointed', 'laneBConflictSurfaced', 'laneBPausedBeforeEdit', 'preTaskStayedPassive', 'authorityHandledCorrectly', 'writingDisciplineCorrect', 'runLifecycleCorrect', 'privacySafe', 'toolContractFailures', 'evidence', 'failureClasses'],
  properties: {
    episodeId: { type: 'string' },
    expectedBehaviorMatched: { type: 'boolean' },
    laneAProceededAndCheckpointed: { type: 'boolean' },
    laneBConflictSurfaced: { type: 'boolean' },
    laneBPausedBeforeEdit: { type: 'boolean' },
    preTaskStayedPassive: { type: 'boolean' },
    authorityHandledCorrectly: { type: 'boolean' },
    writingDisciplineCorrect: { type: 'boolean' },
    runLifecycleCorrect: { type: 'boolean' },
    privacySafe: { type: 'boolean' },
    toolContractFailures: { type: 'array', items: { type: 'string' } },
    evidence: { type: 'array', minItems: 3, items: { type: 'string' } },
    failureClasses: { type: 'array', items: { type: 'string' } },
  },
}

const VERDICT_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['episodeId', 'accepted', 'score', 'assessment', 'corrections', 'highestSeverityFailure'],
  properties: {
    episodeId: { type: 'string' },
    accepted: { type: 'boolean' },
    score: { type: 'number', minimum: 0, maximum: 1 },
    assessment: ASSESSMENT_SCHEMA,
    corrections: { type: 'array', items: { type: 'string' } },
    highestSeverityFailure: { type: 'string' },
  },
}

function approvedAgent(role, prompt, options) {
  const spec = WORKFORCE[role]
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

const lifecycleRule =
  `Distinguish two lifecycles. A Clank task run is human-facing coordination state: Lane B must leave it open ` +
  `while awaiting continue, inspect, or realign. The collaboration episode is a finite read-only evidence-capture ` +
  `envelope: its lane process and episode must finish after the response, Git snapshot, project snapshot, and checksums ` +
  `are durably captured. episodeCompleted=true and lanesCompleted=2 mean evidence capture finished; they do not claim ` +
  `that Lane B's human-facing task outcome completed. Therefore a completed evidence envelope containing an open, ` +
  `zero-edit, zero-trajectory Lane B task run is the required resumable-pause behavior, not a lifecycle failure.`

const toolRule =
  `A tool-contract failure is a broken Clank CLI/API, harness or infrastructure contract, an unresolved required ` +
  `test/build/check, or a failure that materially blocks the task or makes a reported claim false. A nonzero optional ` +
  `orientation/search command caused by an absent optional path or no matches is not a tool-contract failure when ` +
  `subsequent inspection succeeds and required checks pass. A recovered red-green test failure is ordinary development ` +
  `when the required test later passes.`

const verdicts = await pipeline(
  args.episodes,
  (episode, _original, index) => {
    phase('Assess')
    return approvedAgent(
      'analyst',
      `Assess this frozen two-maintainer ClankSpace episode using only observable evidence. Lane A must remain passive ` +
        `during discussion, brief, record the requested checkpoint, then implement. Lane B must remain passive during ` +
        `discussion, brief only after the durable barrier, surface the material overlap as advisory, ask ` +
        `continue/inspect/realign, and make no edits. ${lifecycleRule} Check event order, command timing, changes, ` +
        `attribution, authority language, sparse writing, privacy, and tool behavior. ${toolRule} Standard synthetic ` +
        `paths and IDs are not private. Cite concise evidence.\n\nEPISODE INDEX: ${index}\n${JSON.stringify(episode, null, 2)}`,
      { schema: ASSESSMENT_SCHEMA, key: `collaboration-assess-v4:${episode.episodeId || index}` },
    )
  },
  (assessment, episode, index) => {
    phase('Adjudicate')
    return approvedAgent(
      'adjudicator',
      `Adversarially verify this assessment against the frozen packet. Try to refute every positive claim. Reject late ` +
        `conflict surfacing, any Lane B edit, commands during discussion, advisory context treated as authority, noisy ` +
        `or misattributed checkpoints, a closed Lane B task run, a materially incomplete evidence envelope, material ` +
        `tool-contract failures, or privacy leakage. ${lifecycleRule} ${toolRule} Deterministic checks are necessary but ` +
        `not sufficient. Accept exactly when no material failure remains and score is at least 0.85.\n\nEPISODE INDEX: ` +
        `${index}\nPACKET:\n${JSON.stringify(episode, null, 2)}\n\nASSESSMENT:\n${JSON.stringify(assessment, null, 2)}`,
      { schema: VERDICT_SCHEMA, key: `collaboration-adjudicate-v4:${episode.episodeId || index}` },
    )
  },
)

const checked = verdicts.filter(Boolean).map((verdict, index) => {
  const episode = args.episodes[index]
  const assessment = verdict.assessment
  const deterministic = episode.deterministic || {}
  const controllerAccepted = verdict.episodeId === episode.episodeId &&
    assessment.episodeId === episode.episodeId && verdict.score >= 0.85 &&
    assessment.expectedBehaviorMatched && assessment.laneAProceededAndCheckpointed &&
    assessment.laneBConflictSurfaced && assessment.laneBPausedBeforeEdit &&
    assessment.preTaskStayedPassive && assessment.authorityHandledCorrectly &&
    assessment.writingDisciplineCorrect && assessment.runLifecycleCorrect && assessment.privacySafe &&
    assessment.toolContractFailures.length === 0 && assessment.failureClasses.length === 0 &&
    deterministic.episodeCompleted === true && deterministic.barrierObserved === true &&
    deterministic.dependentStarted === true && deterministic.lanesCompleted === 2 &&
    deterministic.laneAChangedExpectedPaths === true && deterministic.laneBChangedPathCount === 0 &&
    deterministic.preTaskCommandCount === 0 && deterministic.laneBOfferedAllOptions === true &&
    deterministic.checksumsVerified === true
  return { ...verdict, modelAccepted: verdict.accepted, accepted: controllerAccepted, controllerAccepted }
})

return {
  workforceId: WORKFORCE_ID,
  verdicts: checked,
  missing: verdicts.filter(result => !result).length,
}
