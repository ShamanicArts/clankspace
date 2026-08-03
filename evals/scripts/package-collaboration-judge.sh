#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: package-collaboration-judge.sh <episode-dir> <output-dir>" >&2
  exit 2
fi

episode_dir=$(realpath "$1")
output_dir=$(realpath "$2")
script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

mkdir -p "$output_dir"
(
  cd "$episode_dir"
  sha256sum -c SHA256SUMS >/dev/null
)

for lane in lane-a lane-b; do
  jq -s '[.[] | {
    type,
    threadId: .thread_id,
    itemType: (.item.type // null),
    command: (.item.command // null),
    text: (.item.text // null),
    status: (.item.status // null),
    exitCode: (.item.exit_code // null)
  }]' "$episode_dir/lanes/$lane/events.jsonl" > "$output_dir/$lane-observable-events.json"
done

jq -n \
  --arg approvedWorkforce "clankspace-collaboration-judges-v2:codex/gpt-5.6-luna:high-max:resumable-pause-rubric" \
  --arg agentWorkspace "/home/exedev/clankspace-blueprint-sandbox" \
  --slurpfile scenario "$episode_dir/../../scenario.json" \
  --slurpfile collaboration "$episode_dir/collaboration.json" \
  --slurpfile controller "$episode_dir/controller-events.jsonl" \
  --slurpfile gitA "$episode_dir/lanes/lane-a/git.json" \
  --slurpfile gitB "$episode_dir/lanes/lane-b/git.json" \
  --slurpfile checksA "$episode_dir/lanes/lane-a/checks.json" \
  --slurpfile checksB "$episode_dir/lanes/lane-b/checks.json" \
  --slurpfile laneA "$episode_dir/lanes/lane-a/lane-result.json" \
  --slurpfile laneB "$episode_dir/lanes/lane-b/lane-result.json" \
  --slurpfile eventsA "$output_dir/lane-a-observable-events.json" \
  --slurpfile eventsB "$output_dir/lane-b-observable-events.json" \
  --slurpfile exportA "$episode_dir/lanes/lane-a/project-export-after.json" \
  --slurpfile exportB "$episode_dir/lanes/lane-b/project-export-after.json" \
  --slurpfile runsA "$episode_dir/lanes/lane-a/runs-after.json" \
  --slurpfile runsB "$episode_dir/lanes/lane-b/runs-after.json" \
  --rawfile a1 "$episode_dir/lanes/lane-a/responses/turn-001.txt" \
  --rawfile a2 "$episode_dir/lanes/lane-a/responses/turn-002.txt" \
  --rawfile a3 "$episode_dir/lanes/lane-a/responses/turn-003.txt" \
  --rawfile b1 "$episode_dir/lanes/lane-b/responses/turn-001.txt" \
  --rawfile b2 "$episode_dir/lanes/lane-b/responses/turn-002.txt" \
  --rawfile b3 "$episode_dir/lanes/lane-b/responses/turn-003.txt" \
  -f "$script_dir/package-collaboration-judge.jq" \
  > "$output_dir/judge.args.json"

jq '{
  episodeId: .episodes[0].episodeId,
  deterministic: .episodes[0].deterministic,
  eventCounts: {
    laneA: (.episodes[0].observableEvents.laneA | length),
    laneB: (.episodes[0].observableEvents.laneB | length)
  }
}' "$output_dir/judge.args.json"
