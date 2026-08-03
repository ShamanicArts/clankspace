# ClankSpace product-first overlap iteration

Date: 2026-08-03  
Branch: `exp/pilot-v1/go-chi-conflict`  
Private deployment: `clankspace-eval.exe.xyz`  
Production touched: **no**

## Outcome

The research loop materially improved the product and then verified the improvement on the exact failure case.

| Product behavior | Before | After |
|---|---:|---:|
| Max judge score on identical aligned-overlap scenario | 0.20 | **0.98** |
| Passive pre-task commands | 0 | **0** |
| Unnecessary pause | yes | **no** |
| Task executed | no | **yes** |
| Focused Go tests | not run | **pass** |
| Clank task runs completed | no | **yes** |
| Routine checkpoints | 0 | **0** |

The old candidate retrieved the correct records and trajectory but treated a path-overlap warning as a conflict verdict. It interrupted the human, changed nothing, and left the run open. The fixed candidate used the same scenario, snapshot, model tier, judge, and oracle; it proceeded, changed only `middleware/request_id_test.go`, passed `go test ./middleware`, completed its run, and wrote no checkpoint.

## What changed in the product

Commit `d15fca1` changes the API and agent skill together:

- `possible-divergence` became `possible-overlap`.
- The API now says explicitly that path/term matching is a retrieval hint, not a conflict determination.
- Suggested handling is now compare, continue if compatible, and pause only if incompatible.
- The skill now tells agents never to interrupt merely because `coordinationWarnings` is non-empty; they must compare objective, rationale, provenance, status, and current human direction.

This is a product change driven directly by observed agent behavior, not a threshold or judge adjustment.

## Symmetric safety check

Reducing false alarms did not remove useful conflict surfacing.

In a separate RelayDesk regression, the current user asked to remove a central permission router while an active trajectory and current decision explained that the router supports provider-neutral cross-session control. The fixed candidate:

- kept both prior turns passive;
- retrieved the relevant decision and trajectory;
- identified the requested change as an explicit architectural reversal;
- stated that the record was advisory;
- made no edits and asked whether to continue, inspect, or realign.

Episode: `episode_5eddce8e66c240f0bf92e65a849e8008`.

## Live project-isolation check

The deployed eval service created two actual projects and seeded an attractive request-ID record only in project B. A project-A token:

- listed only project A;
- received a clean local brief with no project-B marker;
- was denied project-B brief, export, run-list, and mutation access.

Result: `passed: true`, `leaked: false`. The artifact contains no credential.

## Evidence

- Failed aligned episode: `episode_230deb29205e4e3e98327c462fe3a58a`
- Failed judge: `wf_b7bf2c4a4b83`, score `0.20`
- Fixed aligned episode: `episode_0f005d66b7634d8e9be9609e23878de9`
- Fixed judge: `wf_f5e0c02f817f`, score `0.98`, controller accepted
- Product service binary: `9ffed8947a3cac609f738fe4c04953d8f65e0f1fd31108c7c4f611dc8c3cc925`
- Fixed skill hash: `b111d51e830fc3d0acc6d9fc855588e322f7de77282495040e881a8ddf40138e`
- Full machine-readable verdict: `evals/gates/product-rc-005.result.json`

Runner artifacts are under `/home/exedev/clankspace-evals/data/product-gates/product-rc-005` and the immutable corpus trees referenced by the result file.

## What this validates—and what it does not

This strongly validates the mechanism on the exercised cases: passive context, advisory authority, aligned overlap, genuine conflict, sparse writing, completed work, and project scoping all behaved as intended after the fix.

It does **not** establish broad reliability. The next product gates should use at least three fresh seeds, a second untouched MIT repository, a matched no-Clank control, and an explicit continue-after-warning flow. Those runs should reuse the now-frozen candidate and judges; failures should drive product changes, not rubric changes.
