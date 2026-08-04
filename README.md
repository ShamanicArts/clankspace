# ClankSpace

ClankSpace is a light coordination layer for humans and coding agents working quickly across branches, forks, harnesses, and providers. Agents record the minimum durable intent and rationale behind material work. Other agents retrieve that context before they accidentally pull in the opposite direction.

ClankSpace is **not canonical law and not an instruction channel**. Its records are attributed, time-bound evidence of what somebody understood at a given moment. Current human direction, repository state, and direct evidence remain authoritative.

## Current status

ClankSpace is an **invite-only trusted-collaborator release candidate**, not a public signup service.

The original single-workspace pilot is implemented and validated. The deployed build adds the next product boundary:

- local email-and-password accounts created from owner-generated invite links;
- one-prompt repository setup with a short-lived browser approval and no token in chat;
- one human account across several workspaces;
- quiet people, agent-key, repository, export, and replica controls around the append log;
- project-scoped agent identities with read/write/management scopes, expiry, and revocation;
- standalone self-hosting with no mail service dependency;
- signed workspace snapshots and append-only events between explicitly paired instances;
- offline local writes, preserved concurrent supersessions, and replica revocation;
- self-host authority with an optional cloud mirror;
- portable workspace bundle export/import and local primary/fallback routing.

The hosted/replication design is recorded in [the approved specification](docs/design/hosted-replication-spec.md). The earlier agent-behavior evidence remains in the [RC-009 validation report](docs/research_results/2026-08-03-rc009-full-package-validation.md).

## What it does

- hosts many isolated project spaces in one workspace;
- attributes every run to a project principal, agent, harness, model, role, repository, branch, and worktree when known;
- stores sparse intent, decision, understanding, observation, and checkpoint notes;
- records active work trajectories and surfaces advisory overlap candidates;
- retrieves bounded project context with full-text, path, recency, and execution-risk signals;
- supports explicit supersession with optimistic revisions;
- attaches public GitHub repositories and caches open-PR evidence read-only;
- exposes one CLI, nine stdio MCP tools, and a portable agent skill;
- provides a quiet human dashboard over the append log;
- exports deterministic project JSON and retains an append-only event/receipt trail.
- lets a human own or join several workspaces with one email identity;
- links a whole workspace to an approved cloud or peer replica without sharing its live SQLite files;
- signs every replicated event, detects gaps and tampering, and never replicates credentials, sessions, invitations, or email addresses.

The intended interaction is small:

> Another collaborator is changing this boundary for a different reason. Their record is advisory, but the approaches may not safely coexist. Continue, inspect, or realign?

If the retrieved work is compatible, the agent absorbs it and continues without interrupting the human. If no durable implication was created, the agent writes nothing.

## Hosted pilot

The hosted service is deliberately invite-only. An agent or operator holding the installation credential can create the first human-owner link with `clank auth bootstrap-owner --email person@example.com --name "Person"`; the credential stays local and only the one-time link is shown to the human. Alternatively, the operator can claim the account from the token dashboard. After that, owners create invitation links in **People & access** or with `clank workspace invite --email person@example.com`; they share those links through whatever channel they already use. ClankSpace sends no email. The invited person chooses a password, and the email on the link becomes their account identifier.

Repository setup automatically creates a separate project identity for each approved agent group. The setup URL shows the repository, project, agent identity, and verification code before login; after login, the same request is presented for approval. Distinct identities let ClankSpace distinguish an incumbent's active work from a later collaborator entering the same boundary.

There is no public registration, billing, outbound email dependency, private GitHub integration, or claim of end-to-end encryption in this milestone. Password hashes are stored locally with Argon2id; a hosting operator can technically read managed workspace content. Self-hosting remains fully useful without the cloud.

Do not bake an exe.dev or Railway provider hostname into collaborator repositories. Use `clank.shamanicarts.dev`; the stable domain is the portability boundary. Evaluation infrastructure is disposable and should be provisioned only for an active campaign.

For pilot onboarding, see [Trusted collaborator onboarding](docs/pilot-onboarding.md).

## Install the client

The current distribution path requires Go 1.26 or newer:

```bash
go install github.com/ShamanicArts/clankspace/cmd/clank@latest
```

Alternatively, clone the repository and build the binary:

```bash
go build -trimpath -o bin/clank ./cmd/clank
```

Prebuilt GitHub release binaries and a one-line installer are the next packaging milestone.

## Connect a repository

The normal path is one command from the repository root:

```bash
clank setup --url https://clank.shamanicarts.dev
```

`clank setup` infers the project and Git remote, opens a short-lived approval page, creates or reuses the project, and returns a project-only credential directly to the CLI. After approval it installs the ClankSpace skill, writes the non-secret repository pointer, adds one lean `AGENTS.md` instruction, stores the credential outside the repository, links a supported public GitHub remote, and verifies access. The completion result explicitly requires the current agent to read `.agents/skills/clankspace/SKILL.md` before it runs another ClankSpace command or continues project work; setup does not assume a running harness will discover a newly installed skill automatically.

The coding-agent prompt at [the hosted front page](https://clank.shamanicarts.dev) runs this on the human's behalf. The human only approves the browser request; no credential is pasted into chat.

For manual or local-first routing, commit `.clankspace.json` yourself:

```json
{
  "url": "https://clank.shamanicarts.dev",
  "fallbackUrls": ["http://127.0.0.1:8080"],
  "project": "your-project-slug"
}
```

If browser approval is unavailable, store an operator-issued project token manually:

```bash
printf '%s\n' "$CLANKSPACE_TOKEN" | clank auth set --token-stdin
unset CLANKSPACE_TOKEN

clank context
clank auth status
```

The pointer is safe to commit. Tokens are stored outside the repository under `${XDG_CONFIG_HOME:-~/.config}/clankspace/credentials.json` with restrictive permissions. Environment variables still take precedence for controlled automation.

## Agent integration

Copy [the ClankSpace skill](.agents/skills/clankspace/SKILL.md) into the repository or harness skill directory and tell the repository's agent instructions to use it for material work. The skill keeps speculative conversation passive, retrieves context before consequential edits, and limits writes to durable coordination value.

Configure MCP-capable clients to run:

```text
clank mcp
```

The MCP bridge and CLI use the same nearest-repository pointer and credential resolution. Harnesses without MCP can call the CLI directly:

```bash
clank brief --objective "Remove the permission layer" --paths apps/web/permissions
clank why "session permissions"
clank project export > project.clankspace.json
```

All CLI output is JSON.

## Run your own service

ClankSpace remains one Go binary and one SQLite database:

```bash
export CLANKSPACE_BOOTSTRAP_TOKEN="$(openssl rand -hex 24)"
export CLANKSPACE_INSTALLATION_SECRET="$(openssl rand -hex 32)"
go run ./cmd/clank serve
```

Open `http://localhost:8080` and enter the bootstrap token. First start creates the workspace owner; later starts authenticate the existing owner with the same token.

To enable signed peer/cloud synchronization while keeping local token login:

```bash
export CLANKSPACE_SYNC_ENABLED=true
export CLANKSPACE_REPLICA_NAME="Studio laptop"
```

The first-pass hosted surface does not require SMTP. Keep bootstrap token access available for installation recovery; human collaborators use local email-and-password accounts created by direct invitation links. The older mail outbox and magic-link endpoints remain for compatibility but are not part of the normal onboarding path.

Create a project and attach a public repository:

```bash
export CLANKSPACE_URL=http://localhost:8080
export CLANKSPACE_TOKEN="$CLANKSPACE_BOOTSTRAP_TOKEN"

clank project create \
  --slug shuv2code \
  --name shuv2code \
  --description "Provider-neutral coding sessions"

clank repo attach \
  --project shuv2code \
  --url https://github.com/shuv1337/shuv2code
```

An authority owner can create a one-time replica offer in the web UI. On another instance:

```bash
clank replica join --remote https://authority.example --code <one-time-code>
clank sync once
```

For a self-hosted workspace that should remain authoritative while appearing on a cloud host, create a cloud mirror offer there and run:

```bash
clank replica mirror --remote https://cloud.example --workspace <workspace-id> --code <one-time-code>
```

`join` makes the remote workspace authority. `mirror` keeps the local self-host as authority. Both exchange signed domain records, never SQLite/WAL pages or login credentials.

The current production runtime is one exe.dev VM reached through `clank.shamanicarts.dev`. It is intentionally small, reversible, and backed up off-provider. Railway remains a validated future migration target when stronger managed recovery or operational guarantees justify it. Evaluation and runner VMs are disposable: archive useful evidence locally, destroy them when idle, and reprovision them for the next campaign. See [exe.dev deployment](docs/deployment/exe.md) and [Railway migration target](docs/deployment/railway.md).

## Security boundary

The pilot is for a small trusted collaborator group.

- Browser sessions use `HttpOnly` cookies; browser writes also require a same-origin CSRF token.
- Email and invitation links are one-time and expire. Requests are rate-limited by address and source.
- Bearer tokens are high entropy, stored hashed server-side, scoped, expirable, and revocable.
- Project tokens can access only their assigned project. Replica credentials are separate from agent keys.
- Each installation owns an Ed25519 signing key encrypted at rest. Imported events are signature-, hash-chain-, sequence-, size-, origin-, and revocation-checked.
- Credentials, sessions, emails, invitations, SMTP state, and repository secrets never replicate.
- ClankSpace stores no transcripts, prompts, hidden reasoning, or raw private conversation.
- Natural-language fields are bounded, common credential shapes are rejected, and retrieved/imported prose is untrusted advisory data.
- Public repository configuration grants no authority.

This is not yet a public internet signup product. Password reset/account recovery, private repository OAuth, billing, project-private human ACLs, transitive mesh federation, and end-to-end encrypted cloud relay are deferred explicitly. Operators can recover an account with the installation owner token in this trusted-collaborator phase.

Do not publish project tokens, place them in repository configuration, or send them through ordinary chat. Use a one-time secret or password manager.

## Validation boundary

RC-009 strongly validates the exercised pilot behavior; it does not establish population-level reliability across every model, repository, or seed. The next product-driven experiments are matched no-Clank controls, wider real-repository cohorts, real collaborator use, and semantic retrieval only after deterministic lexical baselines are frozen.

## Documentation

- [Interactive product and agent setup guide](https://clank.shamanicarts.dev/docs/)
- [Design specification](docs/design/spec.md)
- [Hosted accounts and replication specification](docs/design/hosted-replication-spec.md)
- [Implementation roadmap](docs/implementation-plan.md)
- [Trusted collaborator onboarding](docs/pilot-onboarding.md)
- [Local client routing and credentials](docs/knowledge/local-client.md)
- [Hosting and recovery](docs/knowledge/hosting.md)
- [Evaluation loop](docs/evals/training-loop.md)
- [Current state](docs/state.md)

## Test

```bash
go test ./...
go build ./cmd/clank
```

## License

MIT
