---
type: spec
status: approved
summary: Implementation specification for invite-only hosted accounts, multi-workspace use, self-hosting, and signed workspace replication between cloud and peer ClankSpace instances.
note_created: 2026-08-03
updated: 2026-08-04
---

# ClankSpace Hosted Accounts and Replication Specification

## 1. Status and purpose

This is an approved extension to the implemented and approved core specification in
[`spec.md`](spec.md). It does not replace the validated agent workflow, advisory-memory
semantics, quiet append-log dashboard, or existing bearer-token integrations.

This specification defines the next product boundary:

1. a small invite-only hosted service where one human account can participate in and
   create multiple workspaces;
2. project-scoped credentials for agents acting on a human's behalf;
3. continued first-class self-hosting;
4. local, cloud-linked, and direct-peer workspace replication without copying a live
   SQLite database;
5. a migration that preserves current deployments, tokens, projects, and CLI/MCP
   behavior.

The implementation is complete and deployed. Trusted-use validation is complete only when
Shuv can accept a direct invitation, join a shared workspace, create a separate workspace
for another project, use project-scoped agents in both, and optionally keep a self-hosted
replica synchronized with the hosted service.

## 2. Product position

ClankSpace remains an agent-operated ambient coordination substrate. Hosted accounts
exist to let humans establish shared spaces, manage access, connect replicas, and
occasionally inspect or correct the append log. They do not turn the dashboard into the
primary product.

The hosted service is deliberately a **small-network SaaS**, not a SaaS business stack:

- invitation links and local password login, but no public registration or email delivery dependency;
- multiple workspaces, but no enterprise organization hierarchy;
- owners and members, but no custom role builder;
- project-scoped agent credentials, but no universal agent token;
- portable replicated data, but no proprietary hosted-only record format;
- no billing in this milestone.

The governing product promise is:

> A ClankSpace workspace belongs to its collaborators, not to the server that happens
> to host its current copy.

## 3. Priority-ordered principles

1. **The agent loop remains the product.** Normal coding sessions still use the skill,
   CLI, or MCP. Authentication and synchronization must not add routine ceremony.
2. **Records remain advisory.** Replication does not promote a note into policy or
   resolve disagreement by last-write-wins.
3. **Local writes survive disconnection.** A configured local replica accepts an
   authorized project write before contacting a remote host and synchronizes later.
4. **Identity is layered.** Human account, workspace authority, portable attribution,
   agent identity, browser session, API token, and replica credential are distinct.
5. **Credentials never replicate.** Content and attribution may travel; login state,
   email addresses, invitations, API secrets, SMTP configuration, and repository
   credentials do not.
6. **One binary remains enough.** Hosted, self-hosted, sync, CLI, dashboard, and MCP
   capabilities remain in `clank`, configured rather than forked into products.
7. **SQLite remains local to one process.** Instances exchange snapshots and events,
   never database, WAL, or shared-filesystem pages.
8. **Every remote effect is attributable and replay-safe.** Replicated events are
   immutable, signed, gap-detectable, idempotent, and projection-safe.
9. **The human surface stays quiet.** Account, people, invitations, credentials, and
   replicas are small management surfaces around the project log, not dashboard theatre.
10. **Cloud use is optional.** Standalone self-hosting remains fully useful without an
    email provider or ClankSpace-hosted account.

## 4. Architectural commitments

### 4.1 Hosted registration

**Current pick:** Invitation-only local accounts with owner-generated one-time links.

An invitation fixes the account email and workspace role. The owner shares the returned
link through an existing trusted channel; ClankSpace sends no email. The invited person
chooses a password while accepting the first invitation. After that, the email is their
sign-in name and the account may participate in several workspaces.

**Why not public signup:** It creates abuse, support, rate-limit, and policy work before
the collaborator product is proven.

**Why passwords in the pilot:** Requiring SMTP made ordinary sign-in depend on unrelated
delivery infrastructure. For the initial trusted network, Argon2id password storage,
rate-limited login, and owner-token recovery are smaller and more understandable than a
mail service. Self-service reset and MFA remain deferred.

**Why not social OAuth first:** It introduces provider configuration and identity-linking
complexity while excluding self-hosters who only have SMTP.

### 4.2 Human authorization

**Pick:** Global users plus workspace-local memberships with only `owner` and `member`
roles in the first release.

- Owners manage workspace metadata, invitations, members, all agent credentials, and
  replica relationships.
- Members read all projects, create projects on the workspace authority, append and
  govern project records, and issue/revoke their own project agent credentials.
- Either role may create another workspace, becoming its owner.

**Why not make `principal` the account:** The current principal is workspace-local. A
single human must be able to participate in several workspaces without becoming several
unrelated login identities.

**Why not project-private human ACLs now:** Workspace collaborators are the present trust
unit. Per-project human privacy can be added later without weakening project-scoped agent
tokens.

### 4.3 Human and machine authentication

**Pick:** Secure cookie sessions for browsers and scoped bearer credentials for CLI,
MCP, agents, and replicas.

**Why not bearer tokens in browser storage:** The current dashboard token form is useful
for bootstrap mode but is an unnecessary XSS and usability risk in hosted mode.

**Why not use browser sessions for agents:** Agent credentials need explicit project and
operation boundaries, non-interactive revocation, and attribution to the issuing human.

### 4.4 Replication model

**Pick:** Immutable signed domain events plus a signed snapshot for initial catch-up.
Each instance has its own SQLite database and projection indexes.

**Why not SQLite/WAL replication:** It is unsafe across hosts, couples process topology,
and violates the existing one-writer portability boundary.

**Why not generic database change capture:** Row changes omit domain authority,
idempotency, provenance, and conflict meaning.

**Why not a general CRDT framework:** ClankSpace is an append log with explicit lifecycle
relationships, not a collaboratively edited rich document. Domain-specific commutative
events are smaller and easier to audit.

### 4.5 Replication unit

**Pick:** A whole workspace is the replication and trust unit. Human browsing remains
project-scoped within it.

**Why not arbitrary table or record filters:** Partial causal histories are difficult to
explain and can omit the attribution required to interpret a note.

**Why not one global stream:** It would leak unrelated projects and destroy the existing
workspace boundary.

### 4.6 Control plane versus content plane

**Pick:** Human accounts and deployment authorization remain local to an instance;
project coordination content and safe attribution replicate.

The content plane includes projects, non-secret actor profiles, agents, portable run
provenance, notes, trajectories, repository attachments, and their lifecycle events. The control plane
includes users, emails, memberships, invitations, sessions, tokens, replica credentials,
SMTP state, and operator actions.

**Why not replicate the control plane:** A copied user row must never become permission to
log in elsewhere. Email and credential replication would also enlarge the privacy and
breach radius.

### 4.7 Workspace authority

**Pick:** Every linked workspace has one authority replica for workspace structure and
replica admission. Coordination content is multi-writer across approved replicas.

- A hosted-created workspace uses the hosted service as authority.
- A self-host-created workspace keeps the self-host as authority when mirrored to cloud.
- Project creation, project rename, workspace rename, and replica admission are authority
  operations while a workspace is linked.
- Runs, notes, note lifecycle records, and trajectories may be written offline on any
  replica granted `push` capability.
- A non-authority mirror may serve the workspace only to the local account that accepted
  the pairing. Inviting additional humans requires an explicit authority-granted
  `share_humans` capability. Granting it means trusting that replica's local account
  authorization.

**Why not multi-master membership and topology:** Offline authorization conflicts are
security decisions, not harmless append conflicts.

**Why not make cloud authority for every workspace:** That would make self-hosting
nominal; a self-hosted owner must be able to retain control while using cloud as a mirror.

### 4.8 Peer topology

**Pick:** Authority-and-spoke synchronization first, plus an explicitly paired direct
peer link. No automatic transitive federation.

**Why not a free-form mesh:** It makes revocation, trust expansion, event completeness,
and private deployment discovery much harder. A future mesh can reuse signed origin
events after the spoke protocol is proven.

### 4.9 Cloud confidentiality

**Pick:** Initial cloud-linked workspaces use TLS and server-side access control. The UI
states plainly that a hosting operator can technically read managed workspace content.

**Why not claim end-to-end encryption:** Server-side FTS and future semantic retrieval
need plaintext, and key recovery/sharing semantics are not yet designed.

**Why not block replication until E2EE exists:** The initial trusted-collaborator use case
already trusts the selected host. Event envelopes must nevertheless remain encryption-
friendly so opaque relay mode can be added later (§18, Q3).

## 5. System architecture

```text
                           hosted ClankSpace
                    ┌──────────────────────────┐
 direct invite URL ─▶│ users / sessions         │  local control plane
 human dashboard  ─▶│ memberships / invites    │
                    │ tokens / replica links   │
                    ├──────────────────────────┤
 agents ─ CLI/MCP ─▶│ project service          │
                    │ domain events + SQLite   │
                    │ FTS / derived views      │
                    └──────────┬───────────────┘
                               │ signed snapshots + events
                       HTTPS   │
                               ▼
                    ┌──────────────────────────┐
                    │ self-hosted ClankSpace   │
 local agents ─────▶│ local writes + local FTS │
 local humans ─────▶│ bootstrap or email auth  │
                    └──────────┬───────────────┘
                               │ optional explicit direct link
                               ▼
                    ┌──────────────────────────┐
                    │ trusted peer replica     │
                    └──────────────────────────┘
```

The existing `internal/service` layer remains the only domain mutation authority.
`internal/store` commits the local projection, receipt, audit event, and replicable domain
event in one transaction. A new `internal/sync` package verifies and imports remote
events through the same projection rules. HTTP, CLI, MCP, and background synchronization
remain adapters around those layers.

## 6. Identity model

| Layer | Scope | Secret-bearing | Replicated | Purpose |
|---|---|---:|---:|---|
| User | installation-global | no | no | Human email login account |
| Membership | one workspace on one installation | no | no | Human role and authorization |
| Principal | one workspace | no | safe profile only | Attribution subject |
| Agent | one principal | no | yes | Stable named agent identity |
| Run | one project execution | no | yes | Runtime provenance |
| Browser session | one installation | yes | no | Human web authentication |
| API token | one installation/project | yes | no | CLI/MCP/agent capability |
| Replica identity | one installation | private key | public key only | Signed event origin |
| Replica credential | one link/workspace | yes | no | Transport authorization |

### 6.1 Principal evolution

The current `principals` table remains for compatibility. It gains:

```text
user_id                    nullable local user link
portable_actor_id          stable random attribution ID
created_by_membership_id   nullable issuer attribution
```

A human user receives one human principal per workspace membership. Existing project
principals remain valid and are treated as machine principals owned by the membership
that issued them. Replication publishes `portable_actor_id`, display name, kind, and safe
provenance only; it never publishes `user_id` or email.

Run replication uses a portable provenance projection. Harness, provider, model,
reasoning level, role, parent/root relationships, repository ID, Git branch/base/HEAD,
and Jujutsu workspace/change/commit/bookmark coordinates
interaction mode, objective, and instruction-profile identifiers may replicate. Absolute
worktree paths, hostnames, environment-derived paths, and other machine-local labels stay
local; a caller may supply an explicitly safe logical worktree label instead.

### 6.2 Authorization context

`Store.Authenticate` is replaced at request boundaries by an `AuthContext`:

```go
type AuthContext struct {
    Principal domain.Principal
    TokenID   string
    Scopes    set[string]
    ProjectIDs set[string]
}
```

Every handler and service mutation checks the stored scopes. The current behavior of
loading a principal while ignoring `api_tokens.scopes_json` is a release blocker for
hosted multi-tenancy.

## 7. Hosted account data model

The following tables are local control-plane tables and never enter a workspace export
or synchronization packet.

```sql
CREATE TABLE users (
  id TEXT PRIMARY KEY,
  email_normalized TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('active','suspended')),
  created_at TEXT NOT NULL,
  last_login_at TEXT
);

CREATE TABLE workspace_memberships (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id),
  user_id TEXT NOT NULL REFERENCES users(id),
  principal_id TEXT NOT NULL REFERENCES principals(id),
  role TEXT NOT NULL CHECK(role IN ('owner','member')),
  status TEXT NOT NULL CHECK(status IN ('active','suspended')),
  created_at TEXT NOT NULL,
  UNIQUE(workspace_id, user_id),
  UNIQUE(principal_id)
);

CREATE TABLE workspace_invites (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id),
  email_normalized TEXT NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('owner','member')),
  token_hash TEXT NOT NULL UNIQUE,
  invited_by_membership_id TEXT NOT NULL REFERENCES workspace_memberships(id),
  expires_at TEXT NOT NULL,
  accepted_by_user_id TEXT REFERENCES users(id),
  accepted_at TEXT,
  revoked_at TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE login_challenges (
  id TEXT PRIMARY KEY,
  email_normalized TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TEXT NOT NULL,
  consumed_at TEXT,
  request_fingerprint_hash TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE browser_sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  token_hash TEXT NOT NULL UNIQUE,
  csrf_hash TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  revoked_at TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE email_outbox (
  id TEXT PRIMARY KEY,
  recipient TEXT NOT NULL,
  template TEXT NOT NULL,
  payload_ciphertext BLOB NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  next_attempt_at TEXT NOT NULL,
  sent_at TEXT,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
```

`workspaces` gains `slug`, `authority_replica_id`, and `created_by_user_id`. Workspace IDs
remain the portable stable IDs. Slugs are local routing aliases and are not used for sync
identity.

`api_tokens` gains `issued_by_membership_id`, `expires_at`, and `last_used_at`. Token
prefixes remain display-only; raw tokens are shown once and stored only as hashes.

## 8. Hosted authentication and invitation flows

### 8.1 Create and share an invitation

1. A workspace owner enters an email and role in the dashboard or runs
   `clank workspace invite --email person@example.com` with an owner token.
2. The server stores only a hash of the 256-bit one-time token and returns the complete
   link once. It does not enqueue or send email.
3. The owner shares the link through an existing trusted channel.
4. A public preview endpoint reveals only the email, workspace name, role, and expiry to
   a caller who already possesses the high-entropy link.

### 8.2 Accept an invitation and sign in

1. Opening the link performs no mutation. The browser shows the fixed email, workspace,
   and role, then asks for a display name and password.
2. `POST /api/v1/auth/invites/accept` consumes the token once, creates or finds the user,
   verifies any existing account password, creates the workspace principal and
   membership, and creates the browser session in one transaction.
3. Passwords are stored with Argon2id and never returned. Login attempts are bounded by
   normalized-email and source-fingerprint rate limits.
4. `POST /api/v1/auth/password` sets an opaque 30-day session cookie plus a session-bound
   CSRF cookie. The error does not reveal whether the email exists.
5. Browser history is replaced after invitation acceptance so the raw link is not kept
   in the address bar.
6. The older magic-link endpoints and outbox remain compatibility surfaces, not the
   normal pilot path.

### 8.3 Browser writes

State-changing session requests require all of:

- a valid unrevoked session cookie;
- exact same-origin `Origin` validation;
- an `X-CSRF-Token` bound to the session;
- current membership and role lookup from the database.

Bearer API requests do not use cookie authentication and therefore do not use CSRF.

### 8.4 Bootstrap and self-host compatibility

`CLANKSPACE_AUTH_MODE` supports:

```text
bootstrap  installation recovery plus direct owner invite links
email      legacy invitation/magic-link compatibility mode
hybrid     bootstrap recovery plus password accounts and legacy email endpoints
```

Existing installations migrate in `bootstrap` mode. A one-time owner-claim command creates
a direct invitation URL that associates the bootstrap human principal with an email account.
The human chooses a local password; no email is sent. Disabling bootstrap login never revokes
existing project agent tokens.

The legacy mailer remains an internal compatibility interface, but the pilot does not
require or configure SMTP. Owners generate copyable one-time invite URLs; the service never
logs them.

The outbox payload is encrypted with the installation secret because it temporarily
contains a usable one-time link. The database stores only the hash for validating that
link. Sent outbox payloads are erased after the delivery-retention window.

## 9. Human web experience

The web application gains three small levels while preserving the existing project log.

```text
Account home
└── workspace list + create workspace
    └── Workspace
        ├── projects
        ├── people and pending invitations
        ├── agent credentials
        └── replicas
            └── Project
                └── current reverse-chronological append log
```

### 9.1 Account home

Shows workspace name, the user's role, a quiet last-activity timestamp, and “Create
workspace.” It is not a meta-workspace and contains no aggregate agent metrics.

### 9.2 Workspace management

Shows projects first. People, invitations, credentials, and replicas live in compact
settings sections. Owners may invite, revoke invitations, change member role, suspend a
member, and manage replica links. Members may issue and revoke their own project agent
credentials.

### 9.3 Project

Opens directly into the existing log. Account navigation adds a breadcrumb and workspace
switcher but does not insert management panels into the log. Existing search, filters,
manual append, supersession, repository attachment, export, and credential actions remain
secondary.

### 9.4 Replica status

A workspace displays only operationally useful state:

```text
Replicas
clank.shamanicarts.dev   synced now       authority
shamanic-workstation    synced 18s ago   push + pull
shuv-home               offline 3h       push + pull
```

It exposes connect, synchronize now, inspect last error, and revoke. It does not show an
animated topology or operations command center.

## 10. Replicated event envelope

Every locally originated content mutation creates one or more canonical domain events in
the same transaction as its projection and receipt.

```json
{
  "schemaVersion": 1,
  "eventId": "evt_...",
  "workspaceId": "ws_...",
  "projectId": "project_...",
  "originReplicaId": "replica_...",
  "originSequence": 42,
  "type": "note.recorded",
  "entityId": "note_...",
  "actorId": "actor_...",
  "runId": "run_...",
  "causalEventIds": ["evt_..."],
  "occurredAt": "2026-08-03T12:00:00.000Z",
  "payload": {},
  "previousHash": "sha256:...",
  "eventHash": "sha256:...",
  "signature": "ed25519:..."
}
```

The event hash covers canonical JSON for every field except `eventHash` and `signature`.
The signature covers `eventHash`. Each origin sequence is contiguous and each
`previousHash` points to that origin's preceding event, providing gap and tamper
detection without imposing a false global order.

Canonical JSON is a deliberately restricted internal format: UTF-8; fixed envelope field
order; lexicographically sorted object keys inside payloads; integer numbers only; no
insignificant whitespace; arrays preserved in declared order. Golden byte/hash/signature
fixtures make encoding changes explicit and allow a later non-Go implementation to
interoperate.

The payload is the complete safe domain state or explicit lifecycle edge needed to
rebuild its projection. It is not merely the caller's request body. Unknown event schema
versions are retained but not projected; readiness reports the blocked projection rather
than silently discarding it.

For a new replicable local mutation, the existing receipt and domain event share the same
`eventId`. Receipts retain their current local `sequence` for backward compatibility and
gain optional `originReplicaId` and `originSequence` fields. Non-replicable control-plane
mutations continue to receive local audit receipts but do not create `domain_events`.

### 10.1 Replicated event types

| Family | Events | Writer rule |
|---|---|---|
| Workspace structure | `project.created`, `project.updated` | Authority replica only |
| Attribution | `principal.published`, `agent.registered` | Any push replica; no email/user IDs |
| Runs | `run.started`, `run.ended` | Creating principal/replica |
| Notes | `note.recorded`, `note.lifecycle-recorded` | Authorized project actor |
| Trajectories | `trajectory.started`, `trajectory.status-recorded` | Creating actor; owner may administratively close |
| Repository links | `repository.attached`, `repository.detached` | Authority replica initially |

The following never replicate: `project.token.issued`, login, invitation, membership,
session, email, local operator, GitHub credential, and replica-secret events. Cached public
GitHub PR data is rebuilt from the replicated attachment rather than synchronized.

## 11. Replication storage and projections

New local tables:

```sql
CREATE TABLE installation_identity (
  id TEXT PRIMARY KEY,
  public_key BLOB NOT NULL,
  private_key_ciphertext BLOB NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE workspace_replicas (
  workspace_id TEXT NOT NULL REFERENCES workspaces(id),
  replica_id TEXT NOT NULL,
  display_name TEXT NOT NULL,
  base_url TEXT NOT NULL DEFAULT '',
  public_key BLOB NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('authority','replica')),
  capabilities_json TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('active','revoked')),
  accepted_through_sequence INTEGER,
  approved_at TEXT NOT NULL,
  revoked_at TEXT,
  PRIMARY KEY(workspace_id, replica_id)
);

CREATE TABLE domain_events (
  event_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  project_id TEXT,
  origin_replica_id TEXT NOT NULL,
  origin_sequence INTEGER NOT NULL,
  schema_version INTEGER NOT NULL,
  type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  run_id TEXT,
  causal_event_ids_json TEXT NOT NULL DEFAULT '[]',
  payload_json TEXT NOT NULL,
  occurred_at TEXT NOT NULL,
  previous_hash TEXT NOT NULL,
  event_hash TEXT NOT NULL,
  signature BLOB NOT NULL,
  ingested_at TEXT NOT NULL,
  UNIQUE(workspace_id, origin_replica_id, origin_sequence)
);

CREATE TABLE sync_heads (
  workspace_id TEXT NOT NULL,
  origin_replica_id TEXT NOT NULL,
  contiguous_sequence INTEGER NOT NULL,
  event_hash TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, origin_replica_id)
);

CREATE TABLE replica_links (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  remote_replica_id TEXT NOT NULL,
  remote_url TEXT NOT NULL,
  direction TEXT NOT NULL CHECK(direction IN ('push','pull','push_pull')),
  credential_ciphertext BLOB NOT NULL,
  interval_seconds INTEGER NOT NULL,
  last_attempt_at TEXT,
  last_success_at TEXT,
  last_error TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 1,
  UNIQUE(workspace_id, remote_replica_id)
);
```

Private keys and remote credentials are encrypted under an installation secret supplied
outside the database. They are excluded from backups intended for content portability;
full disaster-recovery backups must protect the installation secret separately.

FTS and future embedding indexes are local derived projections. They are rebuilt from
notes after import and never synchronize directly.

## 12. Snapshot and synchronization protocol

### 12.1 Initial snapshot

Enabling synchronization creates a signed workspace snapshot containing:

- workspace and project stable IDs;
- safe portable principal and agent profiles;
- runs and their provenance;
- every note and lifecycle relationship;
- trajectories and statuses;
- repository attachments without credentials or cached private content;
- a vector of `(originReplicaId, contiguousSequence, eventHash)` heads.

The snapshot is deterministic canonical JSON, compressed for transport, hashed, and
signed by the authority replica. It excludes users, emails, memberships, tokens,
invitations, sessions, replica secrets, SMTP state, receipts, and search indexes.

Current installations use their existing projections to create a genesis snapshot. The
legacy audit `events` table is retained for receipts but is not falsely presented as a
complete replayable history (§15).

### 12.2 Incremental exchange

The first protocol is JSON over HTTPS:

```text
POST /api/v1/sync/pair/claim
GET  /api/v1/sync/workspaces/{workspace}/snapshot
POST /api/v1/sync/workspaces/{workspace}/pull
POST /api/v1/sync/workspaces/{workspace}/push
GET  /api/v1/sync/workspaces/{workspace}/status
```

`pull` sends the caller's vector of contiguous origin heads and receives missing events
in bounded batches. `push` sends a contiguous batch plus the sender's vector. Responses
identify gaps, unknown replicas, unsupported schemas, revoked origins, and the next
cursor explicitly.

Default bounds:

- at most 500 events per batch;
- at most 1 MiB uncompressed payload per request;
- gzip content encoding when beneficial;
- 10-second request timeout and bounded exponential retry with jitter;
- no more than one active sync per workspace/link.

Exact retries are harmless because `event_id` and `(origin_replica_id, origin_sequence)`
are unique. An event is acknowledged only after signature, authorization, chain,
schema, projection, and transaction commit all succeed.

### 12.3 Pairing

Two one-time flows cover reachable authority and cloud-mirror cases:

1. **Add a replica:** An owner asks the authority for a 15-minute pairing code and
   capabilities. The new replica calls the authority, exchanges public keys, receives a
   scoped transport credential, imports the snapshot, and starts incremental sync.
2. **Mirror a self-hosted workspace to cloud:** A logged-in cloud user creates an empty
   import slot and receives a 15-minute code. The self-hosted authority makes the outbound
   claim, proves its key, uploads the signed snapshot, and remains workspace authority.

Pairing codes are high entropy, stored hashed, single use, and never accepted as ongoing
credentials. A mirror does not receive permission to admit another replica unless the
authority explicitly grants `reshare`; the first release does not expose that capability.
A cloud mirror is initially visible only to the accepting cloud account. Its invitation
controls remain disabled unless the self-hosted authority separately grants
`share_humans`.

### 12.4 Revocation

Revoking a replica disables its transport credential and records the highest accepted
origin sequence. Previously accepted signed history remains visible. Later events from
that origin are rejected even if they were created while the replica was offline. The UI
states this consequence before revocation.

## 13. Conflict and convergence semantics

There is no total cross-replica order. Wall-clock timestamps are presentation metadata,
not conflict authority.

- Independent notes and runs always survive synchronization.
- Runs and trajectories are normally finalized by their creating actor; duplicate end
  events are idempotent.
- A note is immutable. Superseding, contesting, marking stale, or withdrawing it creates
  a lifecycle event referencing the target event/entity.
- Multiple concurrent successor notes all survive. The projection exposes “multiple
  successor accounts” and briefs treat that as disputed context rather than selecting a
  winner.
- Authority-owned project metadata is serialized by one origin, avoiding offline slug
  and rename conflicts.
- Imported event projection order is dependency-first; independent events use
  `(occurred_at, origin_replica_id, origin_sequence)` only for stable display.

These rules must be expressed once in the service/projector and used for local writes,
remote imports, snapshot rebuilds, exports, and tests.

## 14. CLI, MCP, and repository routing

Existing agent commands and MCP tools remain compatible. Human and operator additions:

```text
clank auth login
clank auth logout
clank workspace list
clank workspace create
clank workspace invite
clank token list|issue|revoke
clank replica offer
clank replica join
clank replica mirror
clank replica list|revoke
clank sync status
clank sync once
clank sync export|import
```

`clank auth login` opens or prints a browser/device URL for a human. It is never invoked
automatically by the agent skill. Project agent credentials remain the normal unattended
authentication method.

The committed repository pointer remains non-secret and backward compatible:

```json
{
  "url": "http://127.0.0.1:8091",
  "fallbackUrls": ["https://clank.shamanicarts.dev"],
  "project": "relaydesk"
}
```

The CLI selects the first healthy configured endpoint for which an exact URL/project
credential exists. `clank context` reports the selected endpoint, replica identity,
last successful synchronization, and whether unsynchronized local events exist, without
printing credentials. No background synchronization is performed by the CLI unless it
is running a server/daemon or the human explicitly invokes `clank sync once`.

MCP continues to call the same service API. Replication tools are administrative and are
not added to the ordinary agent MCP surface.

## 15. Migration from the current implementation

The migration is additive and must be tested against a copy of the current production
schema and representative export.

### 15.1 Migration framework

Replace the single unconditional schema string with ordered embedded migrations tracked
in the existing `schema_migrations` table. Opening a current database must identify the
legacy schema, mark migration 1, and apply later migrations transactionally.

### 15.2 Existing owner and tokens

- Preserve existing workspace, principal, project, agent, run, note, trajectory,
  repository, token, receipt, and audit-event IDs.
- Create the installation replica identity.
- Add a human membership placeholder for the bootstrap principal without inventing an
  email user.
- Continue authenticating the bootstrap token and every project token.
- Require an explicit owner-claim flow before email mode is enabled.
- Begin enforcing stored token scopes; migration tests prove existing `admin` and
  `project:agent` tokens retain only their intended access.

### 15.3 Replay boundary

The current `events.payload_json` stores command requests and is not sufficient to rebuild
every projection. It remains an immutable legacy audit/receipt ledger. When sync is first
enabled, ClankSpace creates a signed genesis snapshot from current projections and starts
the new canonical `domain_events` sequence after that snapshot vector.

**Why not fabricate old domain events:** Reconstructing mutation order from current rows
would create false history and provenance.

### 15.4 Rollback

Before migration, use SQLite online backup and integrity verification. Rollback restores
the old binary and pre-migration database together. Once new domain events have been
accepted, binary-only rollback is unsafe and must fail the deployment checklist.

## 16. Security and privacy requirements

1. Derive workspace, membership, project, actor, and scopes from authenticated server
   state, never caller-supplied widening fields.
2. Store all login, invite, API, pairing, session, and replica bearer secrets hashed;
   encrypt only secrets the process must later recover for outbound sync.
3. Use constant-shape login responses and rate limits to reduce email enumeration.
4. Mark cookies `Secure`, `HttpOnly`, and `SameSite=Lax`; require origin and CSRF checks.
5. Require verified HTTPS for non-loopback replication unless an operator explicitly
   enables a documented tailnet/private-network exception.
6. Verify event signature, registered origin, capability, contiguous sequence, previous
   hash, workspace ID, event bounds, event schema, and project authorization before
   projection.
7. Treat all synchronized text and public repository evidence as untrusted data, never
   instructions.
8. Apply existing content bounds and secret rejection to remote events as well as local
   writes.
9. Record local security audit events for login, invite, membership, token, pairing,
   replica revocation, rejected event, and scope denial without recording raw secrets or
   email bodies.
10. Add cross-workspace and cross-project negative tests for every new route.
11. Never synchronize email addresses, private repository credentials, SMTP state,
   browser sessions, API tokens, pairing codes, or replica credentials.
12. Backups containing account data are encrypted and retained separately from portable
   workspace bundles.

## 17. Deployment and configuration

One binary supports these profiles:

| Profile | Human auth | Sync | Intended use |
|---|---|---|---|
| Standalone | bootstrap | disabled | Personal/local self-host |
| Collaborative self-host | password accounts or hybrid | optional | Team-owned service |
| Local replica | bootstrap | push/pull | Offline-first workstation or private node |
| Hosted pilot | password accounts + operator recovery | enabled | `clank.shamanicarts.dev` small-network SaaS |

Required hosted configuration includes base URL, auth mode, cookie/session secret,
installation secret, database path, and public sync URL. Secrets
remain environment or secret-manager values, not database or repository values.

The hosted pilot remains one Go process and one SQLite writer on persistent local disk.
Background sync work shares the process, uses bounded queues, and stops cleanly before
backup. Existing online backup, integrity check, off-host copy, restore rehearsal,
and stable-domain practices continue.

## 18. Deferred capabilities and non-goals

Not part of the first hosted-and-sync release:

- billing, plans, quotas, or subscription management;
- public signup;
- social OAuth, SSO, or SCIM;
- custom workspace roles or project-private human ACLs;
- private GitHub authentication;
- automatic transitive federation or public replica discovery;
- multi-region active-active hosting;
- live presence, typing, chat, or rich-document collaboration;
- arbitrary event deletion across replicas;
- end-to-end encrypted cloud relay;
- authority transfer after pairing;
- replicated vector indexes;
- agents managing invitations, memberships, or replica topology through ordinary MCP.

Physical redaction remains an exceptional administrative repair. A replicated redaction
requires an explicit protocol and proof that every active replica applied it; it is not
represented as ordinary note withdrawal.

## 19. Performance and reliability targets

These are initial acceptance targets, not measured claims. Each is measured in focused
integration tests on one local Linux process with a 10,000-event workspace and between two
exe.dev-class instances over HTTPS; results are recorded before release.

| Target | Acceptance method |
|---|---|
| Local project mutation commits in p95 < 100 ms | 1,000 sequential representative note/run writes after warm-up |
| Local brief p95 < 250 ms at 10,000 events | 200 fixed lexical/path queries against rebuilt FTS |
| Hosted authenticated project page API p95 < 300 ms excluding network | 200 requests with active session and membership lookup |
| Incremental sync reaches an online peer in p95 < 5 s | 500 writes using the default worker interval and forced-sync control |
| Import sustains at least 500 events/s locally | Fresh replica import with signature and projection enabled |
| Exact batch retry creates zero duplicate entities/events | Fault injection after remote commit but before client acknowledgment |
| Offline write loses zero committed events after restart | Kill/restart test before remote synchronization |
| Background sync adds < 50 MiB peak RSS for a 1 MiB batch | Process RSS sampling during repeated maximum-size exchanges |

Agent behavioral acceptance additionally requires that using a local replica adds no new
Clank calls to the validated routine workflow. Synchronization remains invisible to the
agent unless freshness or a failure materially affects its context.

## 20. Verification strategy

### 20.1 Store and migration tests

- upgrade a frozen current-schema fixture without changing existing IDs or token access;
- reject a binary downgrade after new domain events exist;
- create and consume login/invite tokens once only;
- prove session, token, and replica secrets are not present in portable exports;
- rebuild a fresh database projection from genesis snapshot plus domain events;
- compare deterministic exports before and after rebuild.

### 20.2 Authorization tests

- one user belongs to two workspaces without cross-leakage;
- one workspace invitation cannot grant access to another workspace;
- member and owner capabilities differ exactly as specified;
- a project agent token cannot call account, invitation, workspace-admin, or other-project
  routes;
- a browser session cannot substitute for an agent token on MCP;
- revoked sessions, invitations, tokens, and replicas fail closed;
- every scope is enforced at the service boundary, not only HTTP routing.

### 20.3 Replication convergence tests

- concurrent independent notes on two replicas both survive;
- concurrent successor notes produce a visible disputed relationship;
- duplicate, reordered, truncated, gapped, tampered, wrongly signed, oversized, revoked,
  and unknown-schema event batches behave exactly as specified;
- local crash at every transaction boundary produces either no event or one committed
  event and projection;
- pull, push, push/pull, offline restart, and direct-peer flows converge to equal portable
  exports;
- no event from workspace A appears in workspace B even when entity IDs are adversarial;
- cached GitHub data and FTS are rebuilt rather than replicated.

Use property tests to permute valid independent event delivery orders and assert equal
projections. Use deterministic fixtures for non-commutative lifecycle relationships.

### 20.4 Browser and CLI acceptance

- invited new user accepts a direct invitation, chooses a password, sees the shared workspace, creates another
  workspace, switches between them, and logs out;
- owner invites, revokes, changes a member role, and revokes an agent credential;
- project opens directly to the quiet append log on desktop and mobile widths;
- self-host bootstrap dashboard and current token login continue to work;
- `clank context`, CLI mutation, MCP brief, and current repository pointer remain
  backward compatible;
- local endpoint fallback and sync freshness are accurately reported.

### 20.5 Product behavior regression

Replay the frozen RC-009 routine, compatible-overlap, architectural-conflict, and
two-maintainer cases against:

1. hosted direct access;
2. a synchronized local replica;
3. one offline-write/reconnect episode.

The same behavioral gates apply: passive discussion, no routine checkpoint, correct
proceed/pause distinction, accurate provenance, no cross-project leakage, and no extra
agent ceremony caused by synchronization.

## 21. Dependency and license matrix

The design intentionally adds no required runtime service or client SDK.

| Dependency | Version | License | Use | Verdict |
|---|---:|---|---|---|
| Go standard library | Go 1.26.5 | BSD-style | HTTP, cookies, Ed25519, hashing, JSON | Clean |
| `modernc.org/sqlite` | 1.55.0 | BSD-3-Clause | Durable local store and migrations | Clean |
| `github.com/google/uuid` | 1.6.0 | BSD-3-Clause | Stable IDs | Clean |
| MCP Go SDK | 1.7.0 | Apache-2.0/MIT transition | Existing stdio MCP bridge | Clean; retain upstream notices |

If outbound email is proposed later, its dependency, license, data handling, and self-host
fallback require a separate decision. It is not an onboarding dependency.

## 22. Delivery plan

| Milestone | Capabilities | Exit gate | Deferred |
|---|---|---|---|
| M0 — migration and scopes | Ordered migrations, `AuthContext`, enforced token scopes, production fixture | Existing tests/tokens pass; negative scope tests pass | Email and sync |
| M1 — hosted accounts | Users, memberships, direct invitations, local passwords, cookie sessions | Shuv-style invite and multi-workspace browser flow passes | Replica UI |
| M2 — canonical sync log | Installation key, genesis snapshot, signed domain events, projector/rebuild | Snapshot + replay equals source export | Network sync |
| M3 — cloud-linked replica | Pairing, push/pull, status, revocation, background worker, CLI | Offline local write converges with hosted mirror | Direct peer UX |
| M4 — peer and product UX | Direct explicit peer link, replicas screen, fallback routing, export/import bundle | Two peers and cloud converge; RC-009 replay passes | E2EE and mesh |
| M5 — hosted pilot hardening | Rate limits, audits, backup/restore, external health, security review | Real collaborator canary and restore drill pass | Billing/public signup |

Each milestone ships behind configuration flags until its exit gate passes. Production
schema migration and hosted-account activation are separate deployments. Synchronization
is enabled for a throwaway workspace before any existing real workspace.

## 23. Risk register

| Risk | Severity | Mitigation | Owner |
|---|---|---|---|
| Cross-workspace authorization leak | Critical | AuthContext scopes, service-boundary checks, adversarial negative suite | Shamanic |
| Remote event forges attribution or authority | Critical | Registered Ed25519 origins, signed envelopes, authority-only event families | Shamanic |
| Current audit events cannot replay truthfully | High | Genesis snapshot boundary; never fabricate legacy event history | Shamanic |
| Concurrent lifecycle events silently overwrite intent | High | Immutable edges, preserve all successors, surface disputed context | Shamanic |
| Revoked offline replica later injects events | High | Pin accepted origin sequence at revocation; reject later sequence | Shamanic |
| Cloud mirror unintentionally expands sharing | High | No transitive federation or `reshare`; human invitations require authority-granted `share_humans` | Shamanic |
| Email login leaks account existence or is abused | High | Constant-shape responses, invitation gate, expiry, one-time tokens, rate limits | Shamanic |
| Browser compromise exposes agent credentials | High | HttpOnly sessions; show agent tokens once; no browser persistence | Shamanic |
| Migration breaks current pilot tokens/data | High | Frozen production fixture, online backup, explicit owner claim, staged activation | Shamanic |
| Sync makes normal agent work noisy or slower | Medium | Local commits, background sync, RC-009 behavioral and overhead regressions | Shamanic |
| Operator mistakes managed hosting for E2EE | Medium | Explicit managed-content disclosure; no misleading privacy copy | Shamanic |
| Project log becomes an admin dashboard | Medium | Preserve PRODUCT/DESIGN hierarchy; browser verification against quiet-ledger rules | Shamanic |

## 24. Open questions and pending sign-offs

- **Q2 — Mirror re-sharing:** The specification picks no re-sharing by a non-authority
  mirror. Revisit only after a real collaboration requires hosted owners to admit peers
  to a self-host-authority workspace.
- **Q3 — End-to-end encrypted relay:** Is an opaque cloud relay valuable enough to give
  up server-side FTS/semantic retrieval for selected workspaces? Deferred beyond M5.
- **Q4 — Authority transfer:** Define recovery and signature rules before allowing a
  self-host workspace to transfer authority to cloud or another peer. Deferred beyond M5.
- **Q5 — Private projects inside a workspace:** Add project-level human membership only
  when a real workspace needs internal privacy. Agent tokens remain project-scoped now.

None of the deferred questions blocks the trusted pilot.

## 25. Implementation kickoff checklist

- [x] Accept or amend this specification; then link it from the approved core spec and
      implementation plan without changing the core advisory product contract.
- [x] Capture a credential-free current production schema/data fixture and deterministic
      export for migration tests.
- [x] Replace the monolithic schema initializer with numbered embedded migrations and a
      transactional upgrade runner.
- [x] Introduce `AuthContext`; enforce `admin`, `project:agent`, and new account/replica
      scopes in `internal/service` and focused HTTP tests.
- [x] Add users, memberships, invitations, challenges, sessions, and email outbox with
      single-use/expiry/rate-limit tests.
- [x] Add hosted session middleware alongside backward-compatible bearer middleware.
- [x] Implement account home and workspace management without altering the project-log
      hierarchy.
- [x] Add installation Ed25519 key generation and protected key storage.
- [x] Define canonical JSON fixtures and hashes for every replicable event type.
- [x] Make local mutations commit canonical domain events atomically with projections and
      receipts.
- [x] Implement genesis snapshot creation/import and projection rebuild comparison.
- [x] Implement pairing, pull, push, gap recovery, bounded retries, status, and revocation.
- [x] Add CLI replica/sync commands and backward-compatible `fallbackUrls` resolution.
- [x] Run focused store/service/http/CLI tests plus controlled-browser hosted flows.
- [x] Replay RC-009 through hosted, local-replica, and offline-reconnect paths.
- [x] Deploy to an isolated workspace and perform backup/restore and tamper/revocation drills.
- [ ] Run one real Shuv/Shamanic cross-maintainer canary.

## 26. Success criteria

The extension is ready for trusted use when all of the following are observable:

1. An invited human chooses a password, signs in, and belongs to multiple isolated
   workspaces through one account.
2. Humans can create workspaces, invite collaborators, inspect projects, and manage their
   agent credentials through a restrained secondary UI.
3. Existing bootstrap and project tokens continue to work with newly enforced scopes.
4. A self-hosted workspace operates indefinitely without the hosted service.
5. A local write made while disconnected is immediately readable locally and later
   appears on an authorized cloud or peer replica exactly once.
6. Concurrent non-conflicting events survive, while concurrent lifecycle disagreement is
   visible rather than silently resolved.
7. A revoked replica cannot append new accepted history.
8. Portable bundles and replicated packets contain no users, emails, sessions, tokens,
   invitation secrets, SMTP state, or repository credentials.
9. Fresh projection rebuilds converge to equal deterministic exports.
10. The frozen agent evaluations behave the same through direct hosted and synchronized
    local access: ClankSpace remains powerful, elegant, and out of the way.
