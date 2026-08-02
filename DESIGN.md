---
name: ClankSpace
description: A quiet human window into an agent-maintained project context log.
colors:
  accent: "oklch(47% 0.09 155)"
  accent-soft: "oklch(95% 0.025 150)"
  canvas: "oklch(97.5% 0.006 95)"
  surface: "oklch(99.2% 0.003 95)"
  ink: "oklch(24% 0.012 120)"
  muted: "oklch(52% 0.012 110)"
  faint: "oklch(72% 0.009 105)"
  line: "oklch(88% 0.009 95)"
  warning: "oklch(54% 0.105 65)"
typography:
  family: "Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif"
  pageTitle: "600 1.7rem/1.2"
  entryTitle: "620 0.98rem/1.4"
  body: "400 0.875rem/1.55"
  metadata: "500 0.72rem/1.4"
rounded:
  control: "7px"
  menu: "9px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "16px"
  lg: "24px"
  xl: "40px"
---

# Design System: ClankSpace

## Overview

**Creative north star: the quiet project ledger.**

Scene: a maintainer opens ClankSpace for thirty seconds between coding sessions, in ordinary daytime light, to understand what several agents have carried forward. The interface should disappear behind a dense but readable log.

The default project screen is one reverse-chronological stream. There is no proposed-work form, coordination stage, kanban surface, agent brief panel, or dashboard metric layer. The intelligence lives in agent integrations. The web surface provides inspection and light governance.

## Information hierarchy

1. Project identity and attached repository context.
2. Search and small filters.
3. The append log, newest first.
4. Progressive entry details: rationale, provenance, paths, PRs, and lifecycle.
5. Secondary project actions: append manually, issue agent access, attach a repository, and export.

## Layout

- A narrow project rail remains visible on desktop and becomes a horizontal project strip on small screens.
- The main reading column is approximately 820px wide and left aligned. It is not centred like a marketing page.
- Log entries share one continuous ruled surface. They are rows, not cards.
- Each row uses a compact time column and a flexible content column.
- Long rationale is capped near 72 characters per line.
- Repository evidence appears as inline links on relevant records or a quiet project context line, not a competing section.

## Typography

Use one system sans family. Project names are clear but not heroic. Titles, summaries, rationale, and provenance rely on weight and spacing rather than a display typeface. Paths, SHAs, branches, and run identifiers use the platform monospace stack.

## Color

Use restrained tinted neutrals. Green marks the active project, current lifecycle, links, and primary focus. Amber is reserved for stale or contested context. Superseded and withdrawn records become quieter but remain legible. Color never substitutes for a written state.

## Components

### Log entry

- Small kind and lifecycle text above the title.
- Title and concise project implication remain visible.
- Rationale follows as a plainly labelled sentence when present.
- Actor, human principal, harness, model, role, branch, paths, and evidence form one subdued provenance block.
- Supersede is a secondary text action shown only for current notes.

### Search and filters

Search is the widest control. Kind and lifecycle filters sit beside it. Filtering is immediate and preserves a clear result count. Empty results explain which filters are active and offer a reset.

### Project actions

Manual append is visible but secondary. Infrequent actions live in a familiar overflow menu. Dialogs are acceptable for these short, bounded administrative actions because they are not the primary workflow.

### Empty state

State that no material context has been recorded. Point agents toward the CLI/MCP workflow. Do not imply that the human needs to populate a project board.

## Interaction

- No decorative animation.
- Hover and focus only clarify interactivity.
- Native details disclosure is suitable for long provenance or project settings.
- Refreshing, filtering, and opening entry details must not move the user unexpectedly.

## Do

- Make the latest material understanding scannable in seconds.
- Merge intent, decisions, understandings, checkpoints, and trajectories into one chronological stream.
- Keep rationale and attribution near every entry.
- Make superseded context visibly historical without hiding it.
- Let a quiet, lightly ruled page feel intentionally boring.

## Do not

- Ask a human to describe proposed work to the dashboard.
- Present conflict detection as a web workflow.
- Split the default screen into Now, Agent View, Intent, and Evidence panels.
- Use hero statements, activity metrics, coloured panels, timeline theatre, or nested cards.
- Present records as instructions or canonical decisions.
- Use colored side stripes, gradient text, glass effects, or display fonts.
