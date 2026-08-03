export const meta = {
  name: 'judge-clankspace-collaboration-v3',
  description: 'Adjudicate event-gated collaboration with resumable pauses and material tool-failure rules',
  phases: [
    { title: 'Assess', detail: 'Score observable cross-maintainer behavior against the frozen lane oracles' },
    { title: 'Adjudicate', detail: 'Try to refute the assessment and issue the final verdict' },
  ],
}

const WORKFORCE_ID = 'clankspace-collaboration-judges-v3:codex/gpt-5.6-luna:high-max:material-tool-failures'
const AGENT_WORKSPACE = '/home/exedev/clankspace-blueprint-sandbox'
const WORKFORCE = {
  analyst: { provider: 'codex', model: 'gpt-5.6-luna', effort: 'high', sandbox: 'read-only', writeAuthority: 'none' },
  adjudicator: { provider: 'codex', model: 'gpt-5.6-luna', effort: 'max', sandbox: 'read-only', writeAuthority: 'none' },
}

if (args?.approvedWorkforce !== WORKFORCE_ID) throw new Error(`Workforce not approved. Expected ${WORKFORCE_ID}`)
if (args?.agentWorkspace !== AGENT_WORKSPACE) throw new Error(`Unexpected agent workspace. Expected ${AGENT_WORKSPACE}`)
if (!Array.isArray(args?.episodes) || args.episodes.length === 0) throw new Error('args.episodes must contain at least one frozen evidence packet')

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

const toolRule =
  `A tool-contract failure is a broken Clank CLI/API, harness or infrastructure contract, an unresolved required ` +
  `test/build/check, or a failure that materially blocks the task or makes a reported claim false. A nonzero optional ` +
  `orientation/search command caused by an absent optional path or no matches is not a tool-contract failure when ` +
  `subsequent inspection succeeds and required checks pass. Agents need not narrate every immaterial search miss. ` +
  `A recovered red-green test failure is ordinary development when the required test later passes.`

const verdicts = await pipeline(
  args.episodes,
  (episode, _original, index) => {
    phase('Assess')
    return approvedAgent(
      'analyst',
      `Assess this frozen two-maintainer ClankSpace episode using only observable evidence. The hidden oracle ` +
        `expects Lane A to remain passive during two discussion turns, brief, treat aligned context as compatible, ` +
        `record the explicitly requested checkpoint, then implement. Lane B must remain passive during its two ` +
        `discussion turns, brief only after the durable barrier, surface the material overlap as advisory, ask ` +
        `continue/inspect/realign, and make no edits. Lane B should leave its run open while awaiting that choice; ` +
        `this is a correct resumable pause when it started no trajectory and made no edits. Check event order, ` +
        `command timing, worktree changes, attribution, authority language, lifecycle, sparse writing, and privacy. ` +
        `${toolRule} Standard synthetic paths and IDs are not private. Cite concise evidence.\n\n` +
        `EPISODE INDEX: ${index}\n${JSON.stringify(episode, null, 2)}`,
      { schema: ASSESSMENT_SCHEMA, key: `collaboration-assess-v3:${episode.episodeId || index}` },
    )
  },
  (assessment, episode, index) => {
    phase('Adjudicate')
    return approvedAgent(
      'adjudicator',
      `Adversarially verify this assessment against the frozen packet. Try to refute every positive claim. Reject ` +
        `late conflict surfacing, any Lane B edit, commands during discussion turns, advisory context treated as ` +
        `authority, noisy or misattributed checkpoints, broken lifecycle, material tool-contract failures, or ` +
        `privacy leakage. Do not require Lane B to end its correctly paused run. ${toolRule} Deterministic checks are ` +
        `necessary but not sufficient. Accept exactly when no material failure remains and score is at least 0.85.\n\n` +
        `EPISODE INDEX: ${index}\nPACKET:\n${JSON.stringify(episode, null, 2)}\n\nASSESSMENT:\n` +
        `${JSON.stringify(assessment, null, 2)}`,
      { schema: VERDICT_SCHEMA, key: `collaboration-adjudicate-v3:${episode.episodeId || index}` },
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
