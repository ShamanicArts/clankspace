---
type: devlog
status: complete
summary: Correct the product presentation by making the dashboard a quiet append-log viewer.
note_created: 2026-08-02
updated: 2026-08-02
---

# Append-log dashboard

## Why this changed

The previous dashboard promoted a rare coordination warning into the primary product workflow. It asked a human to describe proposed work, compared it with active trajectories, and presented Continue, Inspect, and Realign controls. That made ClankSpace look like a project-management or collision-detection web application.

ClankSpace is agent-native infrastructure. The skill, CLI, and MCP tools retrieve and append project context inside existing coding sessions. The dashboard is only a quick human window into the resulting log.

## Changes

- Rewrote product and design context around the agent loop and secondary human surface.
- Removed the proposed-work check, collision comparison, agent brief, work panel, and evidence dashboard.
- Opened each project directly into one reverse-chronological stream of notes and trajectories.
- Added immediate search plus kind and lifecycle filters.
- Kept rationale, human/agent attribution, harness, provider, model, role, run type, branch, paths, and evidence adjacent to entries.
- Moved project access, repository attachment, and export into an overflow menu.
- Preserved manual append and supersession as quiet governance paths.

## Verification

- JavaScript syntax check passed.
- Focused HTTP API and service tests passed.
- Controlled browser verification passed at 1280 by 800 and an iPhone-sized 390 by 844 viewport.
- Login, populated log, search, manual-entry dialog, light and dark appearances, and responsive layout were exercised.
- Search reduced the seeded five-entry Shuv/Shamanic context log to the two relevant voice records.
- No console errors or horizontal overflow were observed.
