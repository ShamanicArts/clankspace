---
type: knowledge-index
summary: Master lookup for ClankSpace domain and operational knowledge.
note_created: 2026-08-02
updated: 2026-08-03
---

# Knowledge Base Index

## Quick answers

| Question | Answer | File |
|---|---|---|
| Are notes canonical? | No; they are advisory accrued intent and rationale. | `memory-semantics.md` |
| What identifies an agent action? | Principal → agent → run plus harness/model/Git context. | `identity-and-provenance.md` |
| What may an agent record? | Only minimal, professionally paraphrased, project-relevant information. | `content-boundary.md` |
| How is it hosted? | Permanent production is one Railway service and SQLite volume; ClankSpace evals occupy an isolated slice of the reusable exe.dev agent-compute plane. | `hosting.md` |
| How does storage work? | Transactional events, projections, receipts, WAL, and FTS5. | `architecture.md` |
| How does a local agent find its project and token? | Nearest repository pointer plus a user-local project credential. | `local-client.md` |
| How does a trusted collaborator join? | Operator-issued project identity, non-secret repository pointer, local token storage, and the portable skill. | `../pilot-onboarding.md` |

## Files

| File | Keywords | Description |
|---|---|---|
| `memory-semantics.md` | advisory, intent, notes, conflict | Meaning and precedence of records |
| `identity-and-provenance.md` | principal, agent, model, harness, automation | Runtime attribution model |
| `content-boundary.md` | privacy, professional, paraphrase | Write-time relevance and privacy policy |
| `architecture.md` | sqlite, events, receipts, sync | Technical architecture |
| `hosting.md` | railway, exe.dev, agents, domain, backup, rollback | Permanent production, shared compute boundary, migration, and recovery |
| `local-client.md` | cli, config, credentials, local, auth | Repository routing and local authentication |
