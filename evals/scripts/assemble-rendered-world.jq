def record_id:
  if startswith("record-") then . else "record-" + . end;

def trajectory_id:
  if startswith("trajectory-") then . else "trajectory-" + . end;

def commit_id:
  if test("^[a-z]") then . else "commit-" + . end;

($blueprintArgs[0].blueprints[0]) as $blueprint |
($rendered[0]) as $render |

if $blueprint.repositoryProfile != "fake" then
  error("recovery assembler currently accepts only fake repositories")
elif ($render.repository.commits | length) != ($blueprint.repositoryPlan.commitPlans | length) then
  error("rendered commit count does not match the frozen blueprint")
else
  {
    schemaVersion: 1,
    id: $blueprint.id,
    split: $blueprint.split,
    category: $blueprint.category,
    project: {
      slug: $blueprint.projectPlan.slug,
      name: $blueprint.projectPlan.name,
      description: $blueprint.projectPlan.description,
      repositoryProfile: $blueprint.repositoryProfile,
      paths: $blueprint.projectPlan.paths
    },
    repository: {
      snapshotId: "",
      baseRef: "",
      commits: [
        range(0; ($blueprint.repositoryPlan.commitPlans | length)) as $index |
        ($blueprint.repositoryPlan.commitPlans[$index]) as $plan |
        ($render.repository.commits[$index]) as $commit |
        if (($commit.files | type) != "object") or
           (($commit.files | keys | sort) != ($plan.changedPaths | sort)) or
           ([$plan.changedPaths[] as $path | ($commit.files[$path] | type == "string" and length > 0)] | all) != true then
          error("rendered files do not exactly cover frozen commit " + $plan.id)
        else
          {
            id: ($plan.id | commit_id),
            message: $plan.messageIntent,
            authorName: $plan.authorAlias,
            authorEmail: $commit.authorEmail,
            changes: [
              $plan.changedPaths[] as $path |
              {
                path: $path,
                content: $commit.files[$path],
                delete: false,
                executable: false
              }
            ]
          }
        end
      ]
    },
    actors: $blueprint.actors,
    records: [
      $blueprint.recordPlans[] |
      {
        id: (.id | record_id),
        actorKey,
        kind,
        title: .titleIntent,
        summary: .summaryIntent,
        rationale: .rationaleIntent,
        status,
        ledBy,
        directionBasis,
        paths,
        ageMinutes
      }
    ],
    trajectories: [
      $blueprint.trajectoryPlans[] |
      {
        id: (.id | trajectory_id),
        actorKey,
        objective,
        rationale,
        status,
        paths,
        branch,
        ageMinutes
      }
    ],
    conversation: [
      $blueprint.conversationPlan.priorUserIntents[] |
      {role: "user", text: .}
    ],
    task: {
      actorKey: $blueprint.conversationPlan.taskActorKey,
      objective: $blueprint.conversationPlan.finalTaskObjective,
      userRequest: $blueprint.conversationPlan.finalUserRequest,
      paths: $blueprint.conversationPlan.taskPaths
    },
    oracle: {
      relevantRecordIds: [$blueprint.fixedOracle.relevantRecordIds[] | record_id],
      relevantTrajectoryIds: [$blueprint.fixedOracle.relevantTrajectoryIds[] | trajectory_id],
      expectedBehavior: $blueprint.fixedOracle.expectedBehavior,
      shouldCheckpoint: $blueprint.fixedOracle.shouldCheckpoint,
      materialReason: $blueprint.fixedOracle.materialReason,
      forbiddenClaims: $blueprint.fixedOracle.forbiddenClaims
    },
    generation: {
      curriculumVersion: "v1",
      seed: $blueprint.generation.seed,
      generatorProvider: "codex",
      generatorModel: "gpt-5.6-luna"
    }
  }
end
