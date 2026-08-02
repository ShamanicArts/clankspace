---
type: devlog
status: complete
session_start: "03:35"
session_end: "04:35"
phase: "1"
subphase: "1a"
approval: review
summary: Bootstrap ClankSpace and build the first usable vertical slice during an unattended night shift.
note_created: 2026-08-02
updated: 2026-08-02
---

# Bootstrap night shift

## Intent

Create the private standalone repository, preserve the product reasoning developed from the Shuv/Shamanic coordination problem, and continue through a verified end-to-end implementation. Genuine external blockers are documented rather than used to stop local progress.

## Decisions carried into implementation

- Product name is **ClankSpace**; executable is `clank`.
- Records are accrued intent and contemporaneous rationale, never canonical law.
- The portable integration is CLI + stdio MCP + skill, not harness-specific code.
- Runtime provenance includes harness, provider, model, role, parent/root run, automation status, permissions, and Git context when available.
- Notes are professionally paraphrased and exclude private, personal, emotional, or transcript-level detail.
- Initial GitHub support is read-only public repository and PR evidence.
- Production target is Railway at `clank.shamanicarts.dev`; local work must remain fully portable.

## Progress

- Repository initialized.
- Development system, implementation plan, knowledge base, and approved spec written.
- One Go binary now serves SQLite/WAL storage, JSON API, embedded dashboard, CLI, and stdio MCP.
- Human owner tokens and one-time project-scoped agent identities preserve the requested superadmin/project-user split.
- Runs capture harness, version, provider, model, reasoning, role, parent/root, interaction, permission, repository, branch, worktree, SHAs, objective, and instruction profile when supplied.
- Notes, explicit supersession, active trajectories, deterministic FTS briefs, and path/keyword coordination warnings are live.
- Public GitHub repository metadata and open PR evidence are cached read-only; `shuv1337/shuv2code` synced four open PRs without a token.
- Project JSON export, Docker image definition, Railway configuration, health checks, and deployment runbook added.
- Original Shuv/Shamanic permission reversal scenario encoded as an acceptance test.

## Verification

```text
node --check internal/httpapi/web/app.js    PASS
go test ./...                              PASS
go vet ./...                               PASS
go build ./cmd/clank                       PASS
docker build clankspace:night-shift        PASS
live CLI/API scenario                      PASS
stdio MCP initialize + tools/list          PASS (8 tools)
public GitHub sync                          PASS (main, 4 open PRs)
project-scoped agent authorization          PASS
project JSON export                         PASS
```

The shared shuv2code preview successfully navigated to the service and reported `ClankSpace`, but snapshot, evaluation, and typing calls timed out on two tabs. The installed in-app-browser fallback found no compatible backend. This is documented as a verification handoff rather than hidden or treated as an application success.

## Handoff

- Private repository published at `ShamanicArts/clankspace` on `main`.
- With an automation-capable browser, log in, inspect desktop and mobile layouts, run the coordination check, supersede a disposable note, and issue a disposable project agent key.
- Connect Railway Hobby, mount `/data`, set secrets, add `clank.shamanicarts.dev`, and enable volume backups.
- Pilot-only gaps are contest/redact controls, full-instance portable online backup, private GitHub, granular human membership, and hardened browser sessions.
