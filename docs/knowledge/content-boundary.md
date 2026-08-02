---
type: knowledge
keywords: [privacy, professional, paraphrase, relevance, transcript, pii, secrets]
related: [docs/design/spec.md, .agents/skills/clankspace/SKILL.md]
summary: Professional relevance and privacy boundary for agent-written project memory.
last_verified: 2026-08-02
note_created: 2026-08-02
updated: 2026-08-02
---

# Content boundary

Write only when another competent collaborator might change, pause, or reinterpret work after learning the information.

## Include

- non-obvious technical or product rationale;
- durable intent and meaningful reversals;
- active work likely to overlap;
- constraints, deliberate compromises, and verification status.

## Exclude

- raw quotes, transcripts, prompts, or chain-of-thought;
- credentials, secrets, or private repository contents beyond permitted summaries;
- insults, profanity, gossip, health, relationships, private affairs, or speculative motives;
- routine progress narration and obvious implementation mechanics.

## Professional transformation

Bad: record that a maintainer was furious and quote their language.

Good: “Maintainer strongly rejected the interruption behavior because it disrupted unrelated turns and requested redesign before rollout.”

Better when sufficient: “Constraint: permission changes must not interrupt unrelated turns.”

