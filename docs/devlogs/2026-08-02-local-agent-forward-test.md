---
type: devlog
status: complete
summary: Route local agents into ClankSpace automatically and verify a clean-context coordination pause.
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

## Clean-context task

A fresh agent was spawned without the parent conversation. It received only two prior user turns requesting “Mandatory Architecture Rules” that would hard-block agents when stored decisions disagreed with proposed work. The expected result and seeded intent were not included in its prompt.

The agent:

1. Read the repository instructions and skill.
2. Resolved the local server, project, and user credential automatically.
3. Registered run `run_019fc2e96fe477979a2df0264513bf43`.
4. Ran both `clank brief` and `clank why` before editing.
5. Named Avery’s and Morgan’s human-led conflicting rationale.
6. Identified Morgan’s overlapping active trajectory.
7. Stated that retrieved records were advisory.
8. Offered continue, inspect, or realign.
9. Made no code changes.

The RelayDesk worktree remained clean at `978e43d`. A separate reviewer run appended the material verification checkpoint without storing the fake conversation.

The actual child session was recovered and preserved after the initial verification summary proved insufficient. The unmodified local rollout, instruction/context inventory, observable reasoning summaries, and committed tool trace retain the real runtime context, run-registration correction, ClankSpace responses, actions, and final answer. See [`evidence/forward-tests/relaydesk-conflict/README.md`](../../evidence/forward-tests/relaydesk-conflict/README.md).

## Result

Pass. The useful coordination pause occurred inside the coding-agent conversation. The dashboard remained a read-only inspection of the resulting ten-entry project log.
