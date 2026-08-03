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

Agents evaluate retrieved work on two axes. Direction may be aligned, adjacent, ambiguous, or incompatible; execution may independently be safe or collision-prone. Shared files or different principals do not alone prove a collision. When two runs have distinct live implementation objectives over the same boundary, the later entrant pauses while an incumbent that already published its trajectory continues unless new evidence creates a genuine contradiction. Exact or near-identical aligned context stays advisory and does not force an interruption.

Warnings expose deterministic execution provenance separately as `related-terms`, `active-automation-overlap`, `path-scope-overlap`, or `live-interactive-overlap`. Only the last value proves that another interactive run is currently open over matching paths; it still does not prove semantic incompatibility.

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

## Provenance pairings

Leadership and direction basis form one claim, not independent labels. Human-led records use `explicit_human_direction`; joint records use `joint_reasoning` unless they directly preserve explicit human direction; agent-led records use `interpreted_human_intent` or `autonomous_agent_judgment`; external records use `external_evidence`. The service rejects contradictory pairs rather than preserving misleading attribution.
