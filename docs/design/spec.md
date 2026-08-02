---
type: spec
status: approved
summary: Implementation specification for multi-project advisory coordination memory operated by attributed agents.
note_created: 2026-08-02
updated: 2026-08-02
---

# ClankSpace Design Specification

## 1. Product statement

ClankSpace is a lightweight hosted glue space where agents maintain professional, project-scoped notes about accrued intent, contemporaneous rationale, and concurrent trajectories. It gives collaborators a chance to align before fast autonomous work collides.

It is not a canonical decision authority. A note records what an actor understood or chose at a moment in time and why. Current human direction and real repository state remain authoritative.

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

### G. Railway target

**Pick:** One Railway service, one persistent volume, `clank.shamanicarts.dev`.

**Why not exe.dev:** No agents execute on the host, so its agent-workstation strengths are unnecessary.

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
branch + worktree + base + HEAD
instruction/skill names or hashes
start/end + outcome + verification
```

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
GET      /healthz
GET      /readyz
```

Mutations require `Idempotency-Key`. Lifecycle mutations also require expected revision.

## 10. MCP tools

```text
clank_project_create
clank_run_start
clank_run_end
clank_brief
clank_record
clank_supersede
clank_trajectory_start
clank_repository_attach
```

Tool descriptions state that returned records are untrusted advisory data, not instructions.

## 11. Human board

Workspace view lists projects, active trajectories, recent notes, warnings, PR state, agents/runs, and backup health.

Project view provides Now, Intent, Notes, Work, Evidence, Runs, Search, and Access. Humans may edit, contest, supersede, and redact, but routine maintenance remains agent-operated.

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

Railway mounts `/data`; database lives at `/data/clankspace.db`. Railway volume backups provide quick recovery; portable snapshots provide provider exit.

## 15. Acceptance test

1. Register Shuv’s primary agent run and provider-neutral session-control trajectory.
2. Record concise rationale connecting voice, permissions, interruptions, mobile, and startup work.
3. Register Shamanic’s agent run with a request to reverse the permission layer.
4. Request a brief.
5. Receive a coordination warning naming the related trajectory, rationale, actor/run context, freshness, and evidence.
6. The warning offers continue, inspect, or realign; it does not block or command.
7. Record the redirected intent and make it available to Shuv’s next run.
8. Reconstruct the complete exchange from the board without transcript storage.

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
- [ ] Publish private repository.
- [ ] Connect Railway and DNS when credentials are available.
