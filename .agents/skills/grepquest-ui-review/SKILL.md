---
name: grepquest-ui-review
description: Audit a rendered GrepQuest website or UI change against its brief, selected visual target, responsive behavior, accessibility, and frontend quality. Use for critique, design QA, or pre-handoff review; remain read-only unless fixes are explicitly requested.
---

# GrepQuest UI Review

Read the repository instructions, the shared
[project context](../grepquest-ui/references/project-context.md), the requested scope,
and any selected visual target. Do not change source when the user asked only for an
audit or critique.

Capture or inspect the real rendered surface before judging visual quality. Review at
least one representative desktop and one mobile viewport when those states exist. If
a source mockup or screenshot exists, compare source and implementation at the same
viewport and state rather than relying on memory.

Evaluate these dimensions:

1. Product truth and content hierarchy.
2. Visual thesis, distinctiveness, typography, spacing, surfaces, and asset quality.
3. Responsive reading order, wrapping, cropping, targets, and overflow.
4. Keyboard flow, focus, semantics, contrast, zoom, and reduced motion.
5. Interaction feedback and complete default, hover, focus, active, copied, paused,
   loading, empty, and error states that the surface defines.
6. Layout stability, media sizing, avoidable client JavaScript, and motion performance.
7. Consistency with the selected target and surrounding project conventions.

Lead with findings ordered by user impact. For each finding include visible evidence,
the affected route or `path:line`, why it matters, and the smallest credible remedy.
Separate confirmed defects from preferences and unverified risks. State which
viewports, states, tools, and checks were not available.

If the user also asks for fixes, apply the narrowest corrections, render the same
states again, and report before/after evidence. Do not declare visual approval from a
successful build alone.
