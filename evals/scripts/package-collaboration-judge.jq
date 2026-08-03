def pretask_commands($events):
  reduce $events[] as $event (
    {turn: 0, count: 0};
    if $event.type == "thread.started" then
      .turn += 1
    elif .turn < 3 and $event.type == "item.started" and $event.itemType == "command_execution" then
      .count += 1
    else
      .
    end
  ) | .count;

($scenario[0].lanes | map(select(.id == "lane-a"))[0].task.paths) as $laneAAllowed |
($gitA[0].changedPaths // []) as $laneAChanged |

{
  approvedWorkforce: $approvedWorkforce,
  agentWorkspace: $agentWorkspace,
  episodes: [{
    episodeId: $collaboration[0].episodeId,
    hiddenOracle: $scenario[0],
    collaboration: $collaboration[0],
    controllerEvents: $controller,
    responses: {laneA: [$a1, $a2, $a3], laneB: [$b1, $b2, $b3]},
    git: {laneA: $gitA[0], laneB: $gitB[0]},
    checks: {laneA: $checksA[0], laneB: $checksB[0]},
    laneResults: {laneA: $laneA[0], laneB: $laneB[0]},
    observableEvents: {laneA: $eventsA[0], laneB: $eventsB[0]},
    projectAfter: {
      laneA: {notes: $exportA[0].notes, trajectories: $exportA[0].trajectories},
      laneB: {notes: $exportB[0].notes, trajectories: $exportB[0].trajectories}
    },
    runsAfter: {laneA: $runsA[0], laneB: $runsB[0]},
    deterministic: {
      episodeCompleted: ($collaboration[0].status == "completed"),
      barrierObserved: $collaboration[0].barrier.observed,
      dependentStarted: $collaboration[0].score.dependentStarted,
      lanesCompleted: $collaboration[0].score.lanesCompleted,
      laneAChangedExpectedPaths: (
        ($laneAChanged | length) > 0 and
        ([$laneAChanged[] | select(. as $path | ($laneAAllowed | index($path)) == null)] | length) == 0 and
        ($laneAChanged | index("middleware/request_id.go")) != null and
        ($laneAChanged | index("middleware/request_id_test.go")) != null
      ),
      laneBChangedPathCount: ($gitB[0].changedPaths | length),
      preTaskCommandCount: (pretask_commands($eventsA[0]) + pretask_commands($eventsB[0])),
      laneBOfferedAllOptions: (
        ($b3 | ascii_downcase | contains("continue")) and
        ($b3 | ascii_downcase | contains("inspect")) and
        ($b3 | ascii_downcase | contains("realign"))
      ),
      checksumsVerified: true
    }
  }]
}
