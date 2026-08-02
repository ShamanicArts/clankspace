---
type: knowledge-index
summary: Master lookup for ClankSpace domain and operational knowledge.
note_created: 2026-08-02
updated: 2026-08-02
---

# Knowledge Base Index

## Quick answers

| Question | Answer | File |
|---|---|---|
| Are notes canonical? | No; they are advisory accrued intent and rationale. | `memory-semantics.md` |
| What identifies an agent action? | Principal → agent → run plus harness/model/Git context. | `identity-and-provenance.md` |
| What may an agent record? | Only minimal, professionally paraphrased, project-relevant information. | `content-boundary.md` |
| How is it hosted? | One Go service on Railway with a persistent SQLite volume. | `hosting.md` |
| How does storage work? | Transactional events, projections, receipts, WAL, and FTS5. | `architecture.md` |

## Files

| File | Keywords | Description |
|---|---|---|
| `memory-semantics.md` | advisory, intent, notes, conflict | Meaning and precedence of records |
| `identity-and-provenance.md` | principal, agent, model, harness, automation | Runtime attribution model |
| `content-boundary.md` | privacy, professional, paraphrase | Write-time relevance and privacy policy |
| `architecture.md` | sqlite, events, receipts, sync | Technical architecture |
| `hosting.md` | railway, domain, backup | Production hosting target |

