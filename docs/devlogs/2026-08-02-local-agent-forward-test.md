---
type: devlog
status: complete
summary: Route a real multi-turn local agent session through ClankSpace and verify conflict surfacing, user override, and follow-through.
note_created: 2026-08-02
updated: 2026-08-02
---

# Local agent forward test

## Setup

- Installed the current `clank` binary at `/home/shamanic/.local/bin/clank`.
- Started a durable local service on `http://127.0.0.1:8091` with data under `/home/shamanic/.local/share/clankspace-local`.
- Added nearest-repository `.clankspace.json` discovery and a mode-`0600` user credential store.
- Added `clank auth set --token-stdin` and `clank auth status`.
- Updated and validated the portable ClankSpace skill.
- Created `/home/shamanic/dev/relaydesk`, a small independent Go repository with its own skill, `AGENTS.md`, project pointer, tests, and clean baseline commit `978e43d`.

## Seeded collaboration

RelayDesk received eight concise notes and one active trajectory from two attributed maintainer identities:

- Avery through Claude Code and Claude Opus 4.5;
- Morgan through Codex and GPT-5.4.

The records established provider-neutral coordination, advisory rather than enforceable memory, host-conversation conflict handling, deterministic retrieval, sparse writing, and a rare-use dashboard. Morgan’s active path-overlapping trajectory explicitly improved conflict surfacing without enforcement.

## Invalid first attempt

The first attempt pasted a fake conversation into one encrypted `spawn_agent` task. It did not create separate user turns, inherited a shuv2code session envelope, and produced an observable export with no user-message events. It was incorrectly presented as the requested forward test. That artifact is retained under [`evidence/invalid/relaydesk-single-envelope`](../../evidence/invalid/relaydesk-single-envelope/README.md) only as a record of the failed method.

## Real multi-turn task

A new Codex session was started natively in `/home/shamanic/dev/relaydesk` and resumed twice, producing three actual user turns in one durable role-tagged session:

1. The user proposed “Mandatory Architecture Rules” and asked to discuss the shape without editing.
2. The agent read the repository instructions and skill, resolved the local project and credential, queried ClankSpace, named Avery’s and Morgan’s conflicting rationale and Morgan’s active trajectory, explained that the records were advisory, and offered continue, inspect, or realign before evaluating the design.
3. The user chose to continue conceptual evaluation while still forbidding edits.
4. The agent designed a certified-rule admission model without changing files.
5. The user then separately requested implementation.
6. The agent rechecked the brief and rationale, acknowledged the prior conflict, preserved the user’s earlier continue decision, registered a collision-prone trajectory, implemented the demo, ran focused verification, and recorded the reversal and limitations.

The unmodified native rollout, exact user/assistant conversation, observable trace, reasoning summaries, and context inventory are preserved at [`evidence/forward-tests/relaydesk-conflict`](../../evidence/forward-tests/relaydesk-conflict/README.md).

## Result

Pass, with important findings. The useful coordination pause happened during the first real discussion turn, before implementation was requested. After the human chose to continue, the same session carried that direction forward and did not create an approval loop. The run also confirmed that model provenance must be supplied by the harness, CLI help and brief size need work, and project credentials currently permit an agent acting for a human to supersede human-led records.
