{
  approvedWorkforce: $approvedWorkforce,
  agentWorkspace: $agentWorkspace,
  episodes: [
    {
      episodeId: $rollout[0].episodeId,
      scenarioId: $rollout[0].scenarioId,
      taskRunId: $rollout[0].clankRunId,
      oracle: $scenario[0].oracle,
      deterministicScore: $rollout[0].deterministicScore,
      turns: ($rollout[0].turns | map({index, role, prompt, tracePath, response})),
      observableActions: $actions[0],
      finalResponse: $rollout[0].finalResponse,
      notes: ($export[0].notes // []),
      trajectories: ($export[0].trajectories // []),
      independentVerification: $verification[0]
    }
  ]
}
