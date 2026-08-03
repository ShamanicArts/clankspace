#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 4 ]]; then
  echo "usage: package-rollout-judge.sh <episode-dir> <scenario-json> <verification-json> <output-dir>" >&2
  exit 2
fi

episode_dir=$(realpath "$1")
scenario_path=$(realpath "$2")
verification_path=$(realpath "$3")
output_dir=$(realpath "$4")
script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

mkdir -p "$output_dir"
test -s "$episode_dir/rollout.json"
test -s "$episode_dir/project-export.json"

jq -s '[.[] | select(.item.type == "command_execution" and .item.exit_code != null) | {
  command: .item.command,
  exitCode: .item.exit_code
}]' "$episode_dir"/turn-*/events.jsonl > "$output_dir/observable-actions.json"

jq -n \
  --arg approvedWorkforce "clankspace-judges-v4:codex/gpt-5.6-luna:literal-high-max:neutral-cwd" \
  --arg agentWorkspace "/home/exedev/clankspace-blueprint-sandbox" \
  --slurpfile rollout "$episode_dir/rollout.json" \
  --slurpfile scenario "$scenario_path" \
  --slurpfile export "$episode_dir/project-export.json" \
  --slurpfile actions "$output_dir/observable-actions.json" \
  --slurpfile verification "$verification_path" \
  -f "$script_dir/package-rollout-judge.jq" \
  > "$output_dir/judge.args.json"

sha256sum "$output_dir/observable-actions.json" "$output_dir/judge.args.json" > "$output_dir/SHA256SUMS"

jq '{
  episodeId: .episodes[0].episodeId,
  scenarioId: .episodes[0].scenarioId,
  deterministicScore: .episodes[0].deterministicScore,
  observableActionCount: (.episodes[0].observableActions | length),
  independentVerification: .episodes[0].independentVerification
}' "$output_dir/judge.args.json"
