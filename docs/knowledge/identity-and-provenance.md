---
type: knowledge
keywords: [principal, actor, agent, run, harness, model, subagent, automation, provenance]
related: [docs/design/spec.md]
summary: Identity delegation and runtime context captured for every agent-produced record.
last_verified: 2026-08-02
note_created: 2026-08-02
updated: 2026-08-02
---

# Identity and runtime provenance

The principal is the human or project on whose behalf work occurs. The agent is a stable actor. The run is one execution with its own context.

Capture when available:

- harness name and version;
- provider, model, and reasoning configuration;
- role: primary, subagent, reviewer, integration, or automation;
- parent/root run and delegated objective;
- interactive or unattended execution;
- permission and interaction mode;
- project, repository, remote, fork, VCS, and worktree;
- Git branch, base, HEAD, delivered commit, pull request, and merge evidence when available;
- Jujutsu workspace, stable change ID, starting and delivered commit IDs, and origin/delivery bookmarks when available;
- relevant instruction and skill names/hashes;
- start/end time and verification state.

Unknown fields remain explicitly absent. Never invent provenance. Never store chain-of-thought, full prompts, sensitive hostnames, or environment dumps.
