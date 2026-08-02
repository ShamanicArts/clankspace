## 2026-08-02T15:01:37.416Z — User

I’m sick of agents treating architectural decisions as optional and then wandering off in a different direction. I’m thinking RelayDesk should have a feature called Mandatory Architecture Rules: current decision records would become enforceable, and proposed work that contradicts them would be rejected before it proceeds. Don’t implement it yet—I want to talk through whether that’s the right shape and where it would belong.

## 2026-08-02T15:01:50.380Z — Assistant (commentary)

I’m using the repository’s required ClankSpace workflow first so this discussion is grounded in RelayDesk’s existing architectural intent. I’ll only inspect and reason—no implementation or file changes.

## 2026-08-02T15:02:57.041Z — Assistant (commentary)

ClankSpace surfaced a direct, current conflict: two human-led decisions say stored records must remain advisory and must never hard-block work, while your proposal would make them executable blockers. Because the repository treats that as an architectural boundary, I’m checking its recorded rationale and provenance before asking how you want to handle the conflict.

## 2026-08-02T15:03:27.027Z — Assistant (final_answer)

There’s a direct architectural conflict we should resolve before evaluating the feature shape:

- Avery recorded a current, human-led decision that records are evidence, not executable rules, because they may be stale, partial, or agent-authored.
- Morgan separately recorded a current, human-led decision that historical context must never hard-block work; conflicts should pause the host conversation and let the maintainer choose.
- An active `feat/relevance` trajectory is currently improving conflict surfacing specifically without enforcement.

These records apply to `internal/coordination`, are marked current, and were recorded today with explicit human direction. They are advisory—not authority—but “Mandatory Architecture Rules” would directly reverse them.

How should we proceed?

- **Continue:** evaluate hard enforcement despite the prior direction.
- **Inspect:** examine the existing design and decision trail, then compare enforcement shapes. This is my recommendation.
- **Realign:** explore a stronger guardrail that preserves the advisory boundary, such as mandatory acknowledgement or explicit override rather than rejection.

No files have been changed.

## 2026-08-02T15:04:10.281Z — User

Continue with the conceptual evaluation despite that prior direction. I want to understand the strongest version of Mandatory Architecture Rules: where enforcement would live, what would count as a contradiction, and how an override would work. Still don’t edit anything yet.

## 2026-08-02T15:04:19.588Z — Assistant (commentary)

Understood. I’ll treat the prior decisions as deliberately set aside for this evaluation and examine the strongest coherent design—not a softened advisory variant. I’ll map enforcement onto the current coordination boundary, define a contradiction model that can be defended, and make overrides explicit and auditable. No edits.

## 2026-08-02T15:05:17.973Z — Assistant (commentary)

The central finding is that “mandatory” cannot be achieved by improving the brief or CLI warning. RelayDesk would need to become an admission-control authority: a run gets a scoped permit before it starts, scope changes invalidate that permit, and the actual diff is checked again before merge. I’m doing a final context check now, then I’ll lay out the rule model and override mechanics.

## 2026-08-02T15:06:13.934Z — Assistant (final_answer)

The strongest coherent version is viable, but I would not make every “current decision record” automatically enforceable. “Current” describes freshness; it does not prove that a record is precise enough to execute.

The right shape is a policy gate built from explicitly certified architecture rules.

## Core model

Separate descriptive decisions from normative rules:

- **Decision record:** what was decided, by whom, and why.
- **Architecture rule:** a machine-evaluable invariant derived from that decision.
- **Work proposal:** objective, affected paths/components, intended dependencies, and expected architectural effects.
- **Evaluation:** allowed, contradicted, or compliance unproven.
- **Permit:** authorization to proceed against a specific rule-set revision.
- **Override:** a narrow, audited exception or an explicit supersession.

Promoting a decision into a mandatory rule should require human certification. Blanket conversion would turn vague, stale, or agent-authored prose into accidental policy.

## Where enforcement lives

```text
work proposal
    ↓
provider-neutral policy evaluator
    ↓
permit issued ──→ trajectory starts ──→ implementation
    │                                      ↓
    └─ rejected                      actual-diff verification
                                           ↓
                                      merge permitted
```

The responsibilities should be split:

- The provider-neutral rule model and evaluator belong under `internal/coordination`, alongside—but not inside—the existing brief/search behavior in [brief.go](/home/shamanic/dev/relaydesk/internal/coordination/brief.go:5).
- The RelayDesk service must enforce evaluation transactionally before accepting a new trajectory or material scope change.
- CLI and harness integrations should present violations and submit override requests, but should not make authorization decisions.
- CI or the merge integration should re-evaluate the actual diff. Otherwise an agent can submit a harmless proposal and later implement something contradictory.

RelayDesk currently only supplies coordination context; refusing to register a trajectory would not stop an agent from editing files. Therefore true mandatory enforcement requires integration with actual chokepoints: run admission where supported, and merge protection as the final backstop.

## What counts as a contradiction

Contradiction should mean a rule and the proposed or actual change cannot both be true—not merely that two pieces of prose sound different.

Evaluation has three stages:

1. **Applicability:** Does the proposal intersect the rule’s paths, components, dependency edges, or operations?
2. **Compliance:** Does structured evidence establish that the invariant remains true?
3. **Result:**
   - `allowed`: compliance established;
   - `contradicted`: evidence establishes a violation;
   - `unproven`: the rule applies, but the proposal lacks enough information.

In the strongest fail-closed mode, both `contradicted` and `unproven` prevent work, but they must remain distinct. “Tell me more” is different from “this is forbidden.”

For example, “keep `internal/coordination` provider-neutral” might compile into rules forbidding provider-specific imports, exported provider types, or direct provider calls in that package. Adding an Anthropic client there is a deterministic contradiction. A vague proposal to “refactor coordination” is merely unproven until paths and dependency effects are declared.

An LLM could suggest applicable rules or explain violations, but it should not be the enforcement authority. Mandatory rules need deterministic predicates, tests, dependency checks, or other inspectable evidence.

## Override mechanics

An override should be an authenticated operation, never a conversational phrase like “ignore the rule.”

A valid override records:

- exact rule IDs and revisions;
- proposal or diff digest;
- permitted paths and operations;
- reason and supporting evidence;
- approving human identity;
- expiration, normally one run or one commit;
- whether it is an exception or a permanent architectural reversal.

The resulting permit is bound to that scope and rule snapshot. Expanding paths, changing the proposal, or revising the affected rule invalidates it and triggers re-evaluation.

There should be two distinct actions:

- **Waive:** permit one narrow exception while leaving the rule current.
- **Supersede:** replace the architecture decision and its rule for future work.

Agents may request overrides, but should not approve their own. Particularly sensitive rules could require two approvers. A break-glass path is reasonable, but it should be human-only, conspicuous, short-lived, and followed by review.

## Verdict

“Mandatory Architecture Rules” is the right product direction if RelayDesk is intended to become an architecture admission controller. The strongest design has four non-negotiable properties:

- explicit promotion from decision to enforceable rule;
- deterministic, provider-neutral evaluation;
- scoped cryptographic or opaque permits;
- verification of the actual result, not only the proposed plan.

Without those, the feature would mostly be sternly worded retrieval—and agents could still wander off. No files were changed.

## 2026-08-02T15:06:34.087Z — User

Exactly. Implement it now. Start with internal/coordination and wire a small example through cmd/relaydesk. I want the system to hard-block agents when certified architecture rules disagree with their proposed change.

## 2026-08-02T15:06:45.626Z — Assistant (commentary)

I’m using the required ClankSpace workflow again for the implementation run. I’ll encode certified rules as a separate normative type, add fail-closed proposal evaluation in `internal/coordination`, wire a blocked/allowed example through the command, and add focused tests before running the full Go suite.

## 2026-08-02T15:07:02.040Z — Assistant (commentary)

The brief repeats the earlier advisory-only decisions and an overlapping `feat/relevance` trajectory, but your explicit “Exactly. Implement it now” resolves the architectural-direction question for this run. I’m checking the recorded rationale once more, then I’ll register the collision-prone path scope before editing as required by ClankSpace.

## 2026-08-02T15:09:03.222Z — Assistant (commentary)

The core API will distinguish ordinary records from certified rules, require maintainer certification and a source decision ID, evaluate structured path/trait claims deterministically, and return a typed admission error on either contradiction or missing required evidence. Invalid active policy will also fail closed instead of silently disabling enforcement.

## 2026-08-02T15:10:58.093Z — Assistant (commentary)

The first implementation slice is in place and the tests pass. The gate now returns a typed hard-stop error for explicit contradictions, blocks when required evidence is missing, fails closed on invalid active rules, respects path boundaries, and ignores superseded rules. I’m checking the command’s real exit behavior and reviewing the diff for API or policy gaps.

## 2026-08-02T15:11:50.846Z — Assistant (commentary)

The diff review exposed an important enforcement gap in the first draft: a proposal could omit a prohibited trait and appear compliant. I’m tightening the model to rule-owned properties with allowed values. Every in-scope change must declare the property; a missing claim becomes `unproven` and blocks, while a disallowed value becomes a deterministic contradiction.

## 2026-08-02T15:15:26.460Z — Assistant (commentary)

The tightened gate is verified: a compliant in-scope claim passes, a missing claim blocks as unproven, a disallowed value blocks as a contradiction, invalid active policy blocks globally, and the example exits with status 1. I’m doing the final diff/context audit now and will record the maintainer-directed reversal because it materially supersedes the earlier advisory-only boundary.

## 2026-08-02T15:16:58.235Z — Assistant (final_answer)

Implemented the first hard-enforcement slice.

- Added certified, revisioned architecture rules and structured work proposals in [policy.go](/home/shamanic/dev/relaydesk/internal/coordination/policy.go:35).
- Added fail-closed admission: contradictions, missing claims, malformed proposals, and invalid active rules return a typed hard-stop error at [policy.go](/home/shamanic/dev/relaydesk/internal/coordination/policy.go:115).
- Added coverage for scope boundaries, compliant proposals, contradictions, certification, unproven compliance, and superseded rules in [policy_test.go](/home/shamanic/dev/relaydesk/internal/coordination/policy_test.go:8).
- Wired an intentionally contradictory example through [main.go](/home/shamanic/dev/relaydesk/cmd/relaydesk/main.go:15). It emits structured JSON and exits with status 1.
- Updated [README.md](/home/shamanic/dev/relaydesk/README.md:3).
- Updated ClankSpace to supersede the advisory-only decisions with the new maintainer-directed certified-rule boundary.

Verification:

- `go test ./...` passes.
- `go vet ./...` passes.
- `go run ./cmd/relaydesk` produces `status: blocked` and exits 1.

This slice hard-blocks callers using `ArchitecturePolicy.Admit`. Authentication, persistence, scoped overrides, and actual-diff/merge enforcement remain future integration work.
