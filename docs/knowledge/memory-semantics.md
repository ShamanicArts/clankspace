---
type: knowledge
keywords: [advisory, intent, decision-note, rationale, conflict, precedence]
related: [docs/design/spec.md, docs/knowledge/content-boundary.md]
summary: Defines ClankSpace as accrued intent and coordination evidence rather than canonical authority.
last_verified: 2026-08-02
note_created: 2026-08-02
updated: 2026-08-03
---

# Memory semantics

ClankSpace stores contemporaneous claims about project intent, decisions, rationale, understandings, and active work. A record answers “why did this actor do this then?” It does not command a future agent.

Agents evaluate retrieved work on two axes. Direction may be aligned, adjacent, ambiguous, or incompatible; execution may independently be safe or collision-prone. Compatible intent does not make simultaneous edits to the same implementation boundary safe. When two distinct runs overlap materially, the later entrant pauses while an incumbent that already published its trajectory continues unless new evidence creates a genuine contradiction.

## Precedence

1. Current explicit human and system instructions.
2. Repository-enforced contracts and authoritative external state.
3. Current code, PR, test, and operational evidence.
4. ClankSpace notes as advisory context.

When sources conflict, surface the divergence and request direction. Do not silently obey an older note or discard it.

## Lifecycle

Notes may be `current`, `superseded`, `stale`, `contested`, or `withdrawn`. “Current” means newest known account, not canonical law. Human confirmation validates the account of past intent; it does not make the note permanent.

## Retrieval language

Say “related context,” “recorded rationale,” and “coordination warning.” Avoid “policy requires,” “binding decision,” and “must obey” unless referring to a real repository or security policy outside ClankSpace.
