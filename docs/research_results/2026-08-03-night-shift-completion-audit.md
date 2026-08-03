---
summary: Requirement-by-requirement completion audit for the observable ClankSpace product-validation night shift.
keywords: [completion-audit, operations, deployment, validation, terra, luna, rc-009]
audited_at: 2026-08-03T07:18:46Z
---

# Night-shift completion audit

This audit treats completion as unproven and checks the current worktree, GitHub, runner, production, evaluation, tailnet, immutable corpora, judges, and backups. It does not infer completion from plans or prior summaries.

## 1. Durable operations dashboard and shift log — proven

- Runner tmux windows `ops-dashboard` and `omega-viewer` are alive on the persistent `luna-runner` VM.
- The append-only journal contains 33 valid JSONL entries at `/home/exedev/clankspace-evals/data/ops/shift.jsonl`.
- Operations, raw workflows, and the visual report return HTTP 200 through tailnet-only Tailscale Serve routes on ports 3477, 3478, and 3479.
- The two SSH forwards and static-report server are supervised by project tmux panes rather than this agent process.
- The operations parser distinguishes fake workflow/schema checks as `preflight only`; focused tests pass.
- The operations dashboard and visual report were browser-verified at phone and desktop sizes with no page overflow, broken assets, or console errors in clean tabs.

Evidence: `evals/cmd/clank-ops`, `data/ops/shift.jsonl`, commit `ca1fbda`, and the three tailnet URLs in `docs/state.md`.

## 2. Safe isolated hosting, backup, rollback, and deployment — partially proven

Proven:

- Production and evaluation are separate exe.dev VMs with separate service state.
- Both external `/healthz` and `/readyz` checks currently pass.
- Production runs public-main commit `35503c12`, binary SHA-256 `589c3d81dcfcfe573e1eca8c619a7b6012536d4fcd27cefa08300e1530d3e014`.
- Evaluation runs candidate `62c5682`, binary SHA-256 `68934108bab4893b06caa009ae9ab09fc305210e2cf39273be07791aa384bc46`.
- Evaluation retains multiple explicit binary rollbacks; production retains its prior binary rollback.
- The production service uses a restricted systemd unit with `NoNewPrivileges`, `ProtectSystem=strict`, `ProtectHome`, a dedicated user, and only `/var/lib/clankspace` writable.
- The production backup exists on-host and off-host, both mode 0600 and both SHA-256 `82f91f393f419bdf31fbcb125cf2b985c0b9a44662bb7d5f3446e7a8a43d6bd6`. The recorded disposable restore passed health, readiness, and authenticated project read.

Not proven complete:

- RC-009 candidate `62c5682` is intentionally not promoted to production because its corrected semantic gate has not run.

Evidence: `/var/backups/clankspace/clankspace-prod-20260803T045900Z.db`, `/home/shamanic/Backups/clankspace/clankspace-prod-20260803T045900Z.db`, deployment journal entries, and current systemd/binary inspection.

## 3. Seeded production-like validation — proven for the frozen cohort

- Three independent persisted Luna High sessions exercised frozen go-chi and rs/cors MIT snapshots through two passive user turns followed by a real task.
- One event-gated collaboration episode used two independent Luna High identities, worktrees, credentials, and persisted sessions against one private evaluation project.
- The controller released Lane B only after Lane A's checkpoint was durably observed.
- All pre-task turns issued zero commands and zero Clank calls.
- Routine work proceeded with no checkpoint; aligned overlap proceeded; incompatible architecture paused before edits; later entrant paused with zero changed paths while the incumbent implemented and verified.
- Independent go-chi and rs/cors tests and `git diff --check` pass.
- The complete collaboration `SHA256SUMS` manifest verifies.

Evidence: `product-rc-009` corpora and rollouts, `product-rc-009-collab-001`, and `evals/gates/product-rc-009.result.json`.

## 4. Approved Terra/Luna workforce — proven across the shift; corrected judges pending

Proven:

- Terra workflows `wf_06231bdef862`, `wf_a93573fd1df4`, and `wf_fde5a283c9ca` completed architecture, implementation, review/repair, and acceptance roles with explicit `gpt-5.6-terra` High/Max labels.
- Luna High/Max workflows generated and verified worlds, ran contributors, assessed behavior, and adjudicated immutable packets with explicit provider/model/effort labels and no fallback.

Pending:

- Rollout judge v5 and collaboration judge v3 are new workforce/rubric identities. They are validated and fake-run only; live execution requires their own explicit approval.

## 5. Failure analysis drove product improvements — proven

- RC-006 exposed early work during discussion, wrong-lane yielding, and dishonest verification reporting.
- RC-007 fixed incumbent/later-entrant ownership but exposed aligned over-pause and contradictory checkpoint provenance.
- RC-008 fixed passivity and aligned proceed behavior but exposed an ambiguous live-collision distinction.
- RC-009 adds task-independent `executionRisk`, coherent leadership/basis validation, exact CLI values, pre-auth mutation help, and an AGENTS-level passive bootstrap.
- Before/after evidence shows conflict pre-task commands `7 → 0`, checkpoint mutation retries `2 → 0`, and later-entrant changed files `3 → 0`.

Evidence: commits `c49bd66`, `65c2c0b`, `412a00c`, `15009d7`, `62c5682`; shift journal; and the RC-009 visual report.

## 6. Product and harness verification — proven for changed scope

- Focused tests pass for `internal/service`, `internal/httpapi`, `cmd/clank`, `evals/harness`, and `evals/cmd/clank-ops`.
- `internal/mcpserver` builds and vets; it currently has no test files.
- Focused `go vet` passes for the affected packages.
- `git diff --check` passes.
- The candidate binary on evaluation exactly matches the pinned SHA-256.
- A mistyped audit package name, `internal/mcp`, failed because that directory does not exist; the audit reran against the actual `internal/mcpserver` package and passed. This was audit-command noise, not a product failure.

## 7. Review-ready evidence trail — proven

- Public branch head `20a460b` is pushed and worktree-clean.
- Draft PR #9 is open against `lab/pilot-v1-base`, has a clean merge state, and its current check passes.
- RC-009 gate, blocked result, Markdown report, self-contained HTML report, immutable judge outputs, corpus traces, controller events, checksums, deployment observations, and append-only shift journal are retained.
- The report explicitly separates supported claims from unproven claims and does not include credentials, hidden reasoning, prompts, or private human material.

Evidence: PR #9 and `docs/research_results/2026-08-03-rc009-full-package-validation.{md,html}`.

## 8. Corrected semantic acceptance and final promotion — not achieved

The first-pass judges marked the expected behaviors correct but rejected on evaluator-contract defects:

- rollout judge v4 applied project-global trajectory lifecycle and disallowed a correct resumable pause;
- collaboration judge v2 treated one immaterial optional search miss as a broken product tool contract.

Corrected judge workflows and immutable inputs are ready. They cannot be run live without explicit approval of these exact new manifests:

- `clankspace-judges-v5:codex/gpt-5.6-luna:high-max:task-scoped-resumable-pause`
- `clankspace-collaboration-judges-v3:codex/gpt-5.6-luna:high-max:material-tool-failures`

Therefore:

- RC-009 remains `blocked-adjudication` / `hold-evaluation-only`;
- PR #9 remains draft;
- production correctly remains on `35503c12`;
- the overall night-shift goal is not complete.

## Audit verdict

Everything that can be completed without changing the frozen gate or launching an unapproved workforce is complete and independently inspectable. The sole immediate dependency is explicit approval for the two corrected read-only Luna High/Max adjudicators. Promotion and production deployment must remain downstream of their results.
