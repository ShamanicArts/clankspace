---
name: ClankSpace
description: A shared field notebook for concurrent human and agent intent.
colors:
  coordination-green: "oklch(48% 0.11 155)"
  coordination-green-strong: "oklch(40% 0.105 155)"
  coordination-green-soft: "oklch(93% 0.035 150)"
  warm-field: "oklch(96.5% 0.012 95)"
  paper: "oklch(99% 0.006 95)"
  graphite: "oklch(24% 0.015 120)"
  field-muted: "oklch(49% 0.016 110)"
  field-line: "oklch(86% 0.014 95)"
  divergence-amber: "oklch(50% 0.115 65)"
  divergence-soft: "oklch(94% 0.04 75)"
typography:
  display:
    fontFamily: "Georgia, serif"
    fontSize: "3.2rem"
    fontWeight: 500
    lineHeight: 1
    letterSpacing: "-0.035em"
  headline:
    fontFamily: "Georgia, serif"
    fontSize: "1.75rem"
    fontWeight: 500
    lineHeight: 1.15
    letterSpacing: "-0.02em"
  body:
    fontFamily: "Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.88rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.68rem"
    fontWeight: 850
    lineHeight: 1.2
    letterSpacing: "0.13em"
rounded:
  control: "8px"
  surface: "13px"
  stage: "15px"
spacing:
  xs: "5px"
  sm: "9px"
  md: "18px"
  lg: "28px"
  xl: "42px"
components:
  button-primary:
    backgroundColor: "{colors.coordination-green}"
    textColor: "{colors.paper}"
    rounded: "{rounded.control}"
    padding: "0.72rem 1rem"
    typography: "{typography.body}"
  button-secondary:
    backgroundColor: "transparent"
    textColor: "{colors.graphite}"
    rounded: "{rounded.control}"
    padding: "0.72rem 1rem"
    typography: "{typography.body}"
  coordination-stage:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.graphite}"
    rounded: "{rounded.stage}"
    padding: "42px"
  divergence-panel:
    backgroundColor: "{colors.divergence-soft}"
    textColor: "{colors.graphite}"
    rounded: "{rounded.surface}"
    padding: "18px"
---

# Design System: ClankSpace

## Overview

**Creative North Star: "The Shared Field Notebook"**

ClankSpace feels like a durable notebook left open between trusted technical collaborators: warm enough for human judgment, structured enough for precise agent provenance, and quiet until two directions intersect. The interface preserves the existing editorial warmth while using familiar product controls and predictable task flow.

The primary screen is not an admin inventory. It stages the coordination moment, then reveals work in motion, the bounded agent brief, and the intent trail. Density is earned by attribution and rationale rather than decoration.

**Key Characteristics:**

- Warm, restrained, and technical without looking terminal-native.
- Editorial project titles paired with practical system UI text.
- One green coordination accent; amber appears only for genuine possible divergence.
- Borders and tonal layers carry structure; cards are reserved for consequential interactions.
- Responsive structure, visible focus, and motion limited to state feedback.

## Colors

The palette is a warm field of tinted neutrals with green reserved for coordination and amber reserved for intersecting direction.

### Primary

- **Coordination Green:** Primary actions, current direction, selected state, and agent-context emphasis.
- **Coordination Green Soft:** The bounded agent brief and calm positive state.

### Secondary

- **Divergence Amber:** Possible collision language and the path intersection, never routine decoration.
- **Divergence Soft:** The complete comparison surface behind a coordination warning.

### Neutral

- **Warm Field:** Application background and quiet spatial separation.
- **Paper:** Forms and the primary coordination stage.
- **Graphite:** Primary text and high-trust identity marks.
- **Field Muted:** Supporting rationale, timestamps, and secondary provenance.
- **Field Line:** Structural boundaries and timelines.

**The Consequence Rule.** Green means current coordination. Amber means a human choice is needed. Neither color is decorative.

## Typography

**Display Font:** Georgia with the platform serif fallback.

**Body Font:** Inter with the platform system sans stack.

**Label/Mono Font:** System sans for labels; platform monospace for paths, branches, and PR numbers.

**Character:** Project names and consequential prompts receive a measured editorial voice. Controls, provenance, paths, and runtime details remain compact and operational.

### Hierarchy

- **Display** (500, 3.2rem, 1): project identity only; 2.2rem on compact screens.
- **Headline** (500, 1.75rem, 1.15): the proposed-work coordination prompt.
- **Title** (500, 1.45rem, 1.2): work, brief, intent, and evidence section titles.
- **Body** (400, 0.88rem, 1.5): rationale and project implications, capped near 72 characters where prose runs long.
- **Label** (850, 0.68rem, 0.13em, uppercase): short state and sequence cues only.

**The Two Voices Rule.** Serif identifies the project and consequential reasoning. Sans-serif carries every action, state, and piece of evidence.

## Elevation

The system is flat by default. One diffuse ambient shadow lifts the login surface and primary coordination stage; timelines, trajectories, evidence, and navigation use tonal layering and one-pixel boundaries instead of stacked cards.

### Shadow Vocabulary

- **Ambient Stage** (`0 8px 30px oklch(24% 0.015 120 / 0.045)`): the proposed-work stage only.
- **Modal Lift** (`0 18px 50px oklch(24% 0.015 120 / 0.08)`): temporary dialogs and login.

**The Flat-by-Default Rule.** If a surface is not asking for a decision or temporarily interrupting the workflow, it does not receive a shadow.

## Components

### Buttons

- **Shape:** Compact and gently curved (8px).
- **Primary:** Coordination Green with Paper text and 0.72rem by 1rem padding.
- **Hover / Focus:** Darker coordination green on hover; a visible translucent green outline on keyboard focus; one-pixel active movement.
- **Secondary:** Transparent with a Field Line border. It never competes with the current decision.

### Chips

- **Style:** Tiny neutral pills for runtime provenance. Path chips use monospace and a soft field background.
- **State:** Chips are descriptive, never interactive filters.

### Cards / Containers

- **Corner Style:** 13px for consequential comparisons and the agent brief; 15px for the primary stage.
- **Background:** Paper for the proposed-work stage, Coordination Green Soft for the agent brief, Divergence Soft for collision comparison.
- **Shadow Strategy:** Only the proposed-work stage receives ambient lift.
- **Border:** One-pixel Field Line or a restrained amber mix.
- **Internal Padding:** 18px to 42px according to consequence and viewport.

### Inputs / Fields

- **Style:** Paper background, one-pixel Field Line stroke, 8px corners.
- **Focus:** A three-pixel translucent coordination-green outline outside the control.
- **Error / Disabled:** Errors use explicit text plus red; disabled controls remain legible at reduced opacity.

### Navigation

The top bar holds identity and lock state. The project rail stays quiet, with Paper and a one-pixel border indicating the active project. On mobile it becomes a horizontal strip above the board.

### Direction Comparison

The signature component places the proposed move and active collaborator trajectory on equal sides of an explicit intersection. Actor, objective, rationale, and overlapping path remain visible together. The footer always states that context does not decide for the human and offers Continue, Inspect, or Realign.

## Do's and Don'ts

### Do:

- **Do** lead every project with the proposed-work check and active direction.
- **Do** keep rationale, actor, runtime, and path scope adjacent to material intent.
- **Do** reserve Coordination Green and Divergence Amber for state with consequence.
- **Do** render project agent and run names, not opaque identifiers.
- **Do** keep empty states instructional: declare work, accrue intent, catch divergence.
- **Do** preserve keyboard focus, reduced-motion behavior, and contrast in both appearances.

### Don't:

- **Don't** build a generic admin dashboard organized around object types, counts, and CRUD controls.
- **Don't** make a decision wiki that presents historical notes as settled organizational law.
- **Don't** resemble surveillance or audit products that foreground monitoring people rather than understanding work.
- **Don't** ingest chat transcripts, activity exhaust, or social feeds that reward volume over materiality.
- **Don't** use neon agent-control interfaces, anthropomorphic bot theatres, or decorative AI imagery.
- **Don't** turn the product into a project-management board that requires humans to maintain tickets before agents can coordinate.
- **Don't** use colored side stripes, gradient text, decorative glass, nested cards, or hero-metric layouts.
