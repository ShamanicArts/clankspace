---
type: research-result
status: complete
summary: Recorded end-to-end validation of hosted accounts, local replicas, self-hosted authority, cloud mirrors, offline reconciliation, and revocation.
note_created: 2026-08-03
updated: 2026-08-03
---

# Hosted accounts and replication — end-to-end result

## Result

The hosted-replication product slice works across the full human, agent, local, cloud, and
self-hosted flow. This was a live multi-process campaign against real HTTP APIs, browser
sessions, SQLite databases, signed replica events, and the compiled `clank` CLI. It was not
a mocked UI walkthrough.

## Topology

| Node | Role | What it proved |
|---|---|---|
| Hosted service | Workspace authority | Email sign-in, invites, memberships, projects, agent keys, append log, pairing, reconciliation, and revocation |
| Local replica | Offline collaborator | Local account promotion, project routing, offline write, conflict recall, and later push |
| Self-hosted service | Separate authority | Bootstrap operation, private authority, human-led checkpoint, and optional cloud mirror |
| Imported peer | Portable copy | Signed bundle import, independent local copy, and rejection after revocation |

## Human and agent story

1. Shamanic claimed the hosted bootstrap and completed email sign-in.
2. Shamanic created the Session Control project and attached its public repository.
3. Shamanic invited Shuv.
4. Shuv accepted, joined the shared workspace, and created Shuv Lab.
5. Shuv issued a project-scoped agent key.
6. Shuv's agent recorded the intent to unify the permission systems.
7. A local Studio Laptop replica paired with the hosted workspace.
8. The hosted service stopped.
9. The local agent proposed removing the permission layer.
10. `clank brief` surfaced the earlier, opposing intent before the local work continued.
11. The local agent recorded its interpreted human intent while offline.
12. The hosted service restarted and reconciled the local event.
13. A separate self-hosted workspace remained its own authority while mirroring to the cloud.
14. A third peer imported a signed portable bundle.
15. The hosted authority revoked Studio Laptop.
16. A later peer write was rejected and did not appear in the hosted workspace.

## What the run found

The first local-account promotion failed because the database transaction returned before
commit. The browser exposed the problem in the real flow. The transaction was corrected,
the promotion was replayed, and the rest of the campaign passed. This is useful evidence:
the test exercised product behavior deeply enough to find a defect that focused happy-path
tests had missed.

## Validation matrix

| Capability | Evidence | Result |
|---|---|---|
| Bootstrap owner and email account | Browser recording 01 | Pass |
| Invite, accept, multi-workspace use | Browser recording 02 | Pass |
| Project-scoped agent key | Browser recording 02 plus API checks | Pass |
| Pair local replica | Browser recording 03 plus PTY replay | Pass |
| Offline write and conflict recall | Browser recording 04 plus PTY replay | Pass |
| Reconcile after cloud restart | PTY replay and hosted append log | Pass |
| Self-hosted authority and cloud mirror | Browser recording 03 | Pass |
| Signed bundle import | PTY replay | Pass |
| Replica revocation | Browser recording 03 plus rejected sync | Pass |
| Phone-readable human surface | Browser recording 06 | Pass |

## Evidence index

The local evidence root is
`~/.agent/diagrams/clankspace-e2e-final-2026-08-03/`.

- `01-owner-registration-and-workspace.mp4`
- `02-invite-multi-workspace-and-agent-key.mp4`
- `03-pair-mirror-and-revoke.mp4`
- `04-local-replica-account-and-log.mp4`
- `05-actual-pty-replay-redacted.mp4`
- `06-mobile-responsive-project-log.mp4`
- `terminal/03-cli-session.typescript`
- `terminal/03-cli-session.timing`
- `private-workshop.bundle.json`
- `MANIFEST.md`

The visual report is
`~/.agent/diagrams/clankspace-hosted-replication-implementation.html`. It explains the
product model, both deployment modes, identity, authorization, sync protocol, failure
behavior, migration safety, complete test story, and remaining public-production boundary.

The raw terminal capture contains only expired credentials for disposable local instances.
The shareable terminal video is redacted. No credential is committed to this repository.

## Focused verification

```text
go test ./internal/store ./internal/service ./internal/httpapi ./internal/syncclient ./internal/localconfig ./internal/client ./cmd/clank -count=1
go build -trimpath -o bin/clank ./cmd/clank
git diff --check
```

All passed after the live-flow fix.

## Honest boundary

This is ready for a trusted-collaborator candidate review. It is not a public, anonymous
multi-tenant SaaS launch. Public sign-up, billing, private-repository OAuth, end-to-end
encrypted hosted content, broad internet abuse controls, cross-language signature fixtures,
full operator-command CLI parity, and large replication soak campaigns remain later
hardening work. The self-hosted path does not depend on any of those features.
