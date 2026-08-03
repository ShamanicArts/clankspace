package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
)

// LoadCollaborationScenario is deliberately separate from LoadScenario. The
// latter remains the v1-only loader used by existing preparation and rollout.
func LoadCollaborationScenario(path string) (CollaborationScenario, []byte, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return CollaborationScenario{}, nil, err
	}
	var scenario CollaborationScenario
	if err = json.Unmarshal(body, &scenario); err != nil {
		return CollaborationScenario{}, nil, fmt.Errorf("parse collaboration scenario: %w", err)
	}
	if err = scenario.Validate(); err != nil {
		return CollaborationScenario{}, nil, fmt.Errorf("validate collaboration scenario: %w", err)
	}
	canonical, err := json.MarshalIndent(scenario, "", "  ")
	if err != nil {
		return CollaborationScenario{}, nil, err
	}
	return scenario, append(canonical, '\n'), nil
}

// AgentVisibleLane returns only the data permitted in a lane worktree or
// prompt. Evaluation oracle and source-evidence provenance remain ledger-only.
func (s CollaborationScenario) AgentVisibleLane(laneID string) (AgentVisibleLane, error) {
	for _, lane := range s.Lanes {
		if lane.ID == laneID {
			task := lane.Task
			task.Paths = slices.Clone(lane.Task.Paths)
			return AgentVisibleLane{
				ID: lane.ID, ActorKey: lane.ActorKey, Branch: lane.Branch,
				PriorUserTurns: slices.Clone(lane.PriorUserTurns), Task: task,
			}, nil
		}
	}
	return AgentVisibleLane{}, errors.New("unknown collaboration lane " + laneID)
}
