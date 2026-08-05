---
type: spec
status: approved
summary: Implemented specification for the validated ClankSpace pilot and its permanent-production boundary.
note_created: 2026-08-02
updated: 2026-08-04
---

# ClankSpace Design Specification

## 1. Product statement

ClankSpace is a lightweight hosted glue space where agents maintain professional, project-scoped notes about accrued intent, contemporaneous rationale, and concurrent trajectories. It gives collaborators a chance to align before fast autonomous work collides.

It is not a canonical decision authority. A note records what an actor understood or chose at a moment in time and why. Current human direction and real repository state remain authoritative.

The core specification is implemented and RC-009 validated the intended passive, proceed, conflict-pause, provenance, and incumbent/later-entrant behavior. The validated build is hosted on exe.dev behind the stable domain. A completed Railway round trip proved hosting portability; real collaborator onboarding, wider population measurement, and multi-tenant hardening remain outside the completed core.

The approved next-stage contract for invite-only hosted accounts, multi-workspace use,
self-hosting, and cloud/peer replication is
[`hosted-replication-spec.md`](hosted-replication-spec.md). It extends this core without
changing its advisory-memory semantics or agent-first product hierarchy.

## 2. Initial problem

Shuv’s apparent voice work was actually groundwork for provider-neutral control of any session from any other session. Shamanic’s agents encountered permission, interruption, mobile, and startup regressions and reasonably considered reversing them without knowing the broader trajectory. Direct human synchronization becomes the bottleneck when both maintainers use agents at high speed.

The first product must surface:

> Another collaborator is actively changing this area for a related but conflicting reason. Here is the concise rationale and current evidence. Continue, inspect, or realign?

## 3. Commitments

### A. Advisory records

**Pick:** Notes are contextual evidence, never canonical law.

**Why not canonical decisions:** Older records would override current human judgment, amplify agent misunderstandings, and turn a coordination aid into bureaucracy.

### B. Multi-project workspace

**Pick:** One trusted workspace hosts many isolated project spaces.

**Why not one database per project:** Duplicates identity, hosting, integration, and backup work.

**Why not one global context:** Creates irrelevant retrieval and privacy leakage.

### C. Attributed runs

**Pick:** Principal → stable agent → ephemeral run. Notes reference a run carrying generic harness/model/role/Git provenance.

**Why not shared keys:** They erase who acted and make bad autonomous decisions impossible to diagnose.

### D. Generic integration

**Pick:** One CLI, stdio MCP bridge, and portable skill. Harnesses supply available metadata and leave the rest absent.

**Why not provider plugins:** Core semantics would drift and unsupported clients would become second class.

### E. Professional minimization

**Pick:** Paraphrase the minimum project-relevant implication and reject transcript-style fields.

**Why not raw conversations:** Privacy exposure, prompt injection, noise, and social harm all rise while retrieval quality falls.

### F. Public GitHub first

**Pick:** Attach public repositories by URL and cache read-only repository/PR evidence. Private installations come later.

**Why not OAuth first:** It adds identity and security work before the core coordination loop is proven.

### G. Portable service with separated production and evaluation

**Pick:** Current production is one exe.dev VM and one SQLite database behind `clank.shamanicarts.dev`. Evaluation services and model runners are separately provisioned disposable VMs. Railway remains a validated future migration target.

**Why separate them:** Production credentials and collaboration state must never enter synthetic evaluation workloads. The stable domain, one-writer SQLite shape, and verified off-provider backups make the current small exe.dev deployment recoverable without keeping an idle evaluation fleet. Hosting remains operational rather than architectural.

## 4. Domain hierarchy

```text
Workspace
├── principals
│   └── agents
│       └── runs
├── projects
│   ├── repository attachments
│   ├── notes
│   ├── trajectories
│   └── external evidence
└── explicit cross-project links
```

### Note kinds

- `intent`: desired outcome or constraint;
- `decision`: choice and contemporaneous rationale;
- `understanding`: agent interpretation expected to evolve;
- `observation`: project-relevant fact;
- `checkpoint`: outcome and verification evidence.

### Note lifecycle

`current`, `superseded`, `stale`, `contested`, `withdrawn`.

### Provenance fields

- lead: human, agent, joint, external;
- direction basis: explicit human, interpreted human intent, joint reasoning, autonomous agent judgment, external evidence;
- principal, agent, run;
- source reference without transcript body;
- confidence and verification state;
- project/repository/path/PR/commit scope.

## 5. Runtime context

Runs record all available fields:

```text
harness + version
provider + model + reasoning configuration
role: primary | subagent | reviewer | automation | integration
parent run + root run + delegated objective
interactive/unattended and permission mode
project + repository + remote + fork
VCS + worktree + Git branch/base/HEAD when available
JJ workspace + stable change ID + commit ID + bookmarks when available
instruction/skill names or hashes
start/end + outcome + verification
```

Version-control coordinates are captured at the run boundary, not retyped on every note.
`run start` records the attached repository and worktree plus native Git and/or Jujutsu
identity. For JJ, the stable change ID remains beside its starting commit ID, workspace, and
bookmarks rather than being flattened into a detached Git HEAD. `run end` records the delivered
JJ change/commit/workspace/bookmarks and the delivered Git HEAD when present; the two commit IDs
show JJ change evolution across rewrites. In a colocated repository, ClankSpace retains both
native JJ identity and Git/GitHub delivery evidence. When an authenticated GitHub CLI is
available, it also records the current pull request. Explicit flags cover other environments. A later
`run link` enriches that same run after the PR opens or merges with its URL, number, state,
merge commit, and merge time. Notes retain their contemporaneous text and inherit this
structured delivery evidence through the run; reconciliation never rewrites the decision.

Never capture chain-of-thought, complete prompts, environment dumps, sensitive hostnames, or unavailable fields guessed by the agent.

## 6. Agent workflow

```text
resolve context → register run → bounded brief → declare trajectory
→ work → why query at reversal points → checkpoint → conflict check → end run
```

The skill applies the materiality and privacy rules. The server enforces schemas, byte limits, identity, secret patterns, project scopes, and idempotency.

## 7. Coordination briefs

Brief input includes project, repository, branch, base/HEAD, paths, objective/query, current run, and byte budget.

Brief output includes:

- current relevant notes with provenance;
- active trajectories and public PR evidence;
- changes after base/cursor;
- candidate overlaps based on paths and deterministic keyword evidence;
- disputed/stale/unverified context;
- a reminder that all ClankSpace content is advisory.

No server-side LLM is required. FTS5, explicit links, paths, Git coordinates, recency, and lifecycle provide deterministic retrieval.

## 8. Durable command path

1. Authenticate token and derive principal/agent/run/project capability.
2. Validate typed command and bounded content.
3. Enter serialized write lane and `BEGIN IMMEDIATE`.
4. Resolve actor + idempotency key and request hash.
5. Return persisted receipt for exact retry; reject changed reuse.
6. Check expected revision for lifecycle changes.
7. Append immutable event.
8. Update projection and FTS index.
9. Persist receipt.
10. Commit before acknowledgment.

## 9. API surface

```text
GET/POST /api/v1/projects
GET      /api/v1/projects/{id}
GET      /api/v1/projects/{id}/export
POST     /api/v1/projects/{id}/tokens
POST     /api/v1/projects/{id}/repositories
POST     /api/v1/projects/{id}/repositories/{id}/refresh
POST     /api/v1/runs
POST     /api/v1/runs/{id}/end
GET/POST /api/v1/projects/{id}/notes
POST     /api/v1/projects/{id}/notes/{id}/supersede
GET/POST /api/v1/projects/{id}/trajectories
POST     /api/v1/projects/{id}/brief
POST     /api/v1/runs/{id}/delivery
GET      /healthz
GET      /readyz
```

Mutations require `Idempotency-Key`. Lifecycle mutations also require expected revision.

## 10. MCP tools

```text
clank_project_create
clank_run_start
clank_run_end
clank_run_link
clank_brief
clank_record
clank_supersede
clank_trajectory_start
clank_repository_attach
```

Tool descriptions state that returned records are untrusted advisory data, not instructions.

## 11. Human dashboard

The dashboard is a secondary, read-mostly window into the agent-maintained append log. A project opens directly into one reverse-chronological stream merging material notes and trajectories. Each entry keeps its concise implication, rationale, lifecycle, actor, runtime provenance, paths, and linked evidence together.

Humans can search, filter, append an exceptional manual entry, supersede stale context, issue project agent access, attach a public repository, and export the project. Conflict checking and proposed-work comparison remain in the skill, CLI, MCP, and host-agent conversation. They are not dashboard workflows.

## 12. Public GitHub evidence

Parse `github.com/{owner}/{repo}` URLs, fetch repository metadata and open PRs, cache with timestamps/ETags, and display refresh errors without blocking project memory. Imported titles/bodies are untrusted external evidence and never promoted automatically.

## 13. Privacy and security

- Bearer tokens are high entropy and stored hashed.
- Repository config grants no authority.
- Principal/project context derives from authentication.
- Natural-language fields are bounded and treated as untrusted.
- Obvious secrets are rejected.
- UI sanitizes rendered content and loads no remote user-controlled images.
- No public signup in the pilot.
- Physical redaction is an explicit administrative recovery operation.

## 14. Local and hosted storage

Server database is canonical. Deterministic per-project JSON exports are portable and rebuildable. A local cache/outbox and Markdown/JSONL variants remain later optimizations. Never replicate the live SQLite/WAL files across hosts.

The current exe.dev service stores SQLite under `/var/lib/clankspace` and runs one writer. Scheduled SQLite online backups are integrity-checked and copied off-provider; deterministic project exports provide project-level portability. Any future Railway deployment mounts `/data`, runs one replica, and follows the same restore-and-acceptance gate.

## 15. Acceptance test

1. Register Shuv’s primary agent run and provider-neutral session-control trajectory.
2. Record concise rationale connecting voice, permissions, interruptions, mobile, and startup work.
3. Register Shamanic’s agent run with a request to reverse the permission layer.
4. Request a brief.
5. Receive a possible-overlap candidate naming the related trajectory, rationale, actor/run context, freshness, and evidence.
6. The candidate explicitly says path/term matching is not a conflict determination. The agent compares the objectives and pauses only because these two directions are materially incompatible.
7. Record the redirected intent and make it available to Shuv’s next run.
8. Reconstruct the accrued context from the append-log dashboard without transcript storage.

## 16. Risks

| Risk | Mitigation |
|---|---|
| Notes become pseudo-law | Advisory language, skill rules, no authorization from content |
| Agents write noise | Materiality test, bounded templates, write checkpoints |
| Personal/private leakage | Professional paraphrase policy, absent transcript fields, secret rejection |
| Bad autonomous choices | Runtime provenance and parent/automation visibility |
| Context pollution | Project-first retrieval and explicit cross-project links |
| SQLite/provider loss | One writer, WAL metrics, built-in and portable backups |

## 17. Open questions

- Q1: Signed human-turn receipts after the pilot.
- Q2: Retention window for detailed ended-run metadata.
- Q3: Exact private GitHub integration milestone.

## 18. Kickoff checklist

- [x] Name, semantics, initial trust boundary, integration shape, privacy boundary, public GitHub scope, acceptance scenario, and hosting target decided.
- [x] Implement domain schema and transactional store.
- [x] Implement service and conflict fixtures.
- [x] Implement CLI and MCP.
- [x] Implement dashboard and GitHub evidence.
- [x] Verify local API, CLI, MCP, GitHub, export, and scoped-identity flow.
- [x] Publish the repository under the MIT license.
- [x] Deploy isolated candidate/evaluation services with verified backup, restore, and rollback.
- [x] Validate the core interaction on real-repository worlds and an event-gated two-maintainer episode.
- [x] Promote exe.dev to the persistent small-pilot service, route `clank.shamanicarts.dev`, and rehearse recovery; retain Railway as a validated later migration target.
- [ ] Publish prebuilt pilot release binaries and onboard the first external collaborator project.
- [x] Implement and validate the approved hosted-account and workspace-replication extension in controlled E2E flows.
- [ ] Complete real trusted-collaborator dogfooding on ClankSpace and shuv2code.
