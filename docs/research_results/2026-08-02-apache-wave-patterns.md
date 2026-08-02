---
type: research-result
status: active
summary: Reusable Apache Wave patterns for ClankSpace sync, history, agent capabilities, and prompt-independent authorization.
keywords: [apache-wave, google-wave, append-log, sync, capabilities, token-router, guardrails]
note_created: 2026-08-02
updated: 2026-08-02
---

# Apache Wave patterns worth stealing

## Finding

Do not fork Apache Wave into ClankSpace. Its valuable contribution is a small set of substrate patterns, while its mutable collaborative-document model, Operational Transformation engine, GWT client, gadgets, XMPP federation, and broad product surface solve a different problem.

The final code is the retired Apache-2.0 repository [`apache/incubator-retired-wave`](https://github.com/apache/incubator-retired-wave), archived at commit `e4cb87f`. Apache also preserves the [0.4 distribution](https://archive.apache.org/dist/incubator/wave/) and [federation documentation](https://cwiki.apache.org/confluence/display/WAVE/Federation).

## Patterns in the source

| Wave pattern | Source shape | ClankSpace translation |
|---|---|---|
| Durable append plus projection | `DeltaAndSnapshotStore` appends contiguous deltas and a resulting snapshot durably | Keep events authoritative and projections disposable; commit event, receipt, search projection, and current view together |
| Versioned, tamper-evident history | `HashedVersion` combines a monotonic version with a hash of preceding deltas | Add a per-project event cursor and optional previous-event/hash chain to exports and sync receipts |
| Snapshot plus contiguous tail | `WaveletAndDeltas` and `DeltaSequence` reject gaps and rebuild state from a snapshot plus ordered deltas | `clank sync`: fetch a bounded project snapshot, then events after a cursor; reject gaps and rebuild the disposable local cache |
| Attributable operations | Every transformed delta retains author, application time, applied version, and resulting version | Preserve principal, agent, run, receipt, time, and previous cursor adjacent to every mutation |
| Explicit robot capabilities | `RobotCapabilities` maps subscribed event types to filters and hashes the declaration | Store a server-issued capability profile for each agent credential/run; do not infer authority from model prose |
| Robots as participants | Robots receive only subscribed events for wavelets where they participate, then return typed operations | Agents operate as attributable project participants through bounded typed tools, not anonymous omniscient automations |
| Access checked at data boundary | Snapshot/history access is checked against the participant; invalid version boundaries fail closed | Derive project and operation authority from authenticated server state on every endpoint; reject caller-supplied identity or scope |
| Commit notification distinct from update | Subscribers receive both live update and durable-commit notifications | Acknowledge ClankSpace writes only after the event and projections commit; later streaming can distinguish observed from durable |

These are visible in the archived source around `DeltaAndSnapshotStore`, `WaveletDeltaRecord`, `HashedVersion`, `DeltaSequence`, `WaveletAndDeltas`, `RobotCapabilities`, `OperationContext`, and `WaveletNotificationSubscriber`.

## Prompt-independent token router

The useful interpretation of “hide it from the system prompt” is not secrecy from the model. It is separation of policy from prompt text:

1. A repository contains only the small ClankSpace skill and project pointer.
2. Records are never bulk-injected into system instructions. They arrive only as bounded, explicitly requested, untrusted tool output.
3. A long-lived project credential can exchange for a short-lived run token.
4. The server binds that run token to principal, project, agent, run, allowed operations, repository, optional path envelope, expiry, and rate/content ceilings.
5. Middleware checks the token capability against the endpoint and stored resource before the CLI/MCP handler runs.
6. Caller-provided principal, project, run, role, or scope fields cannot widen the token.
7. Writes still require idempotency receipts and content/privacy validation; denials are auditable without storing prompts.

Suggested first capability set:

```text
context:read
brief:read
why:read
run:start|end
trajectory:start|end
note:create|supersede
project:export
project:admin
```

Ordinary agent tokens should omit `project:admin`. A rollout/test token can be further restricted to one synthetic project and expiry. The simplest implementation is internal Go authentication/authorization middleware over the existing `api_tokens.scopes_json`, not a new proxy service.

MCP tool descriptions may still occupy harness instruction space, depending on the provider. Deferred tool discovery or CLI-only use can reduce that overhead, but the security property must remain server-side even when the model sees every tool description.

## What to build from this

### Near term

- Add method-level scope enforcement to existing API tokens.
- Add short-lived run-token exchange with fixed project and actor provenance.
- Add monotonic per-project event cursors to exports and receipts.
- Design `clank sync` around snapshot plus contiguous events after a cursor; keep its local SQLite cache disposable.
- Include a capability/instruction-profile hash in run provenance so surprising autonomous decisions can be traced to the actual tool surface.

### Later, only if demanded

- Cursor-based SSE or long polling for active trajectories and material notes.
- Explicit cross-host project links or federation after one hosted workspace proves insufficient.
- Hash-chained portable exports if tamper evidence becomes materially useful.

### Explicitly avoid

- OT or CRDT infrastructure: ClankSpace is an append log with explicit supersession, not a concurrently edited rich document.
- A Wave-like all-purpose human workspace: the dashboard remains a quiet window into the agent-maintained log.
- XMPP federation, gadgets, live typing, presence, or nested conversation UI in the first product.
- Broad robot callbacks that send the whole project stream to an external agent.

## Product lesson

Wave's strongest substrate survived, but its product mixed conversation, documents, synchronous editing, federation, bots, and applications into one unfamiliar surface. ClankSpace should do the inverse: preserve a narrow coordination job, keep humans out of the dashboard during normal work, and make the advanced history/capability machinery almost invisible.

Contemporary Hacker News recollections reinforce that split: project-sized collaborative history was useful, while large amorphous spaces became hard to navigate and the lack of a focused use case hurt adoption. See the [Apache Wave discussion](https://news.ycombinator.com/item?id=7532059) and [Google Wave post-mortem discussion](https://news.ycombinator.com/item?id=22815713).
