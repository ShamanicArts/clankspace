---
type: guide
status: active
summary: Provision and connect a trusted collaborator to the hosted ClankSpace pilot.
note_created: 2026-08-03
updated: 2026-08-04
---

# Trusted collaborator onboarding

ClankSpace is an invite-only pilot at `https://clank.shamanicarts.dev`. There is no public signup and no email delivery dependency. In the normal path a human pastes one setup prompt into their coding agent, signs in once, reviews the detected repository, and approves it. The project credential returns directly to the waiting CLI and never needs to enter chat.

The provider-neutral pointer is `https://clank.shamanicarts.dev`; never commit the exe.dev origin. Production runs on exe.dev today, while Railway remains a validated migration target. Complete the recurring backup schedule and use distinct collaborator credentials before the first real canary.

## Identity model

Issue one project token per human or durable collaborator identity, not one shared token for the whole team and not one token per agent run.

For example:

```text
project: shuv2code
├── Shuv agents       one project principal and token
└── Shamanic agents   one project principal and token
```

Individual Codex, Claude, automation, primary, and subagent executions register as attributed runs beneath that principal. This preserves the distinction between an incumbent's work and a later collaborator entering the same boundary.

## Human: create or join an account

An existing owner creates a direct invitation in **People & access** or with:

```bash
clank workspace invite --email person@example.com --role member
```

Share only the returned `inviteUrl` with that person. They choose a password on the invite page. For the first installation owner, an operator or agent holding the installation credential runs `clank auth bootstrap-owner`; again, only its `inviteUrl` goes to the human.

## Agent: install and connect the repository

The current source installation requires Go 1.26 or newer:

```bash
go install github.com/ShamanicArts/clankspace/cmd/clank@latest
```

From the repository root, run:

```bash
clank setup --url https://clank.shamanicarts.dev
```

The CLI detects the Git remote, opens a short-lived approval request, and waits. The human signs in, selects or creates the workspace, and approves the repository. The CLI then stores the project credential outside the repository, installs `.clankspace.json`, the agent skill, and the lean `AGENTS.md` instruction. The committed pointer and skill contain no secret.

The agent must read the installed skill before continuing, then verify:

```bash
clank context
clank auth status
```

Expected properties include:

- the stable hosted URL `https://clank.shamanicarts.dev`;
- project `shuv2code`;
- `tokenConfigured: true`;
- the correct repository remote and current branch;
- the advisory-authority notice.

`clank auth set --token-stdin` remains a manual fallback when browser approval is unavailable; it is not normal onboarding.

## Seed only genuine coordination context

Do not copy synthetic evaluation data into the real project. Start with a few current, human-grounded records that could actually change another collaborator's next action—for example:

- provider-neutral cross-session control is the larger active trajectory;
- voice, permissions, interruption handling, mobile behavior, and startup work intersect that architecture;
- a current PR or branch is using a particular approach and why;
- a human explicitly redirected or superseded an earlier approach.

The easiest flow is to ask each collaborator's agent to register its real current run, retrieve the board, and record only the material trajectory or human-directed checkpoint.

## First real canary

1. Shuv's agent starts a material run and publishes its active trajectory.
2. Shamanic's separately credentialed agent receives a task over the same implementation boundary.
3. The later agent requests a brief before editing.
4. If the objectives are materially compatible and execution is independent, it proceeds quietly.
5. If distinct simultaneous changes may collide, it surfaces the actual overlap and asks whether to continue, inspect, or realign.
6. Confirm the dashboard attributes both runs to the correct principals and contains no transcript-style or personal material.

Success is not “the agent always pauses.” Success is that ClankSpace remains invisible during ordinary work and creates a useful human decision point only when concurrent intent materially matters.

## Pilot limitations

- public GitHub repositories only;
- manual source installation until release binaries exist;
- one browser approval per repository setup; project bearer credentials are stored locally by the CLI;
- project token issue and revocation are available in the dashboard; method scopes remain fixed by credential role;
- no public signup or untrusted multi-tenant use;
- deterministic lexical/path retrieval; semantic embeddings are not yet enabled;
- signed snapshot/event synchronization supports approved local, cloud, and peer replicas.

For the permanent service, migration, backups, and recovery, see [Railway deployment](deployment/railway.md). For ClankSpace's isolated allocation on the reusable agent-compute platform, see [exe.dev agent infrastructure](deployment/exe.md). For current product evidence, see the [RC-009 report](research_results/2026-08-03-rc009-full-package-validation.md).
