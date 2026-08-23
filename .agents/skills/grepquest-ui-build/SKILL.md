---
name: grepquest-ui-build
description: Implement a selected GrepQuest web design or a scoped change to an established website with responsive, accessible, production-quality frontend code. Use after a visual target exists; do not use for concept-only exploration or read-only review.
---

# GrepQuest UI Build

Read the repository instructions, the shared
[project context](../grepquest-ui/references/project-context.md), and the selected
visual target. If a net-new surface has no selected target, stop implementation and
route to `grepquest-ui-explore`. A small edit to an established interface may use the
existing rendered UI as its target.

Before editing, inspect the active site, package-manager lockfile, framework config,
global styles, tokens, shared components, representative routes, and repository checks.
Preserve the established stack and dependency versions. Add a dependency only when it
materially improves the requested outcome and fits the declared environment.

Implement the complete user journey shown by the selected target:

- Use real GrepQuest content, commands, states, and captures.
- Keep navigation, anchors, primary calls to action, copy controls, disclosures, and
  visible interactive states functional.
- Create semantic HTML and resilient layouts before adding decoration or motion.
- Support mobile, tablet, desktop, keyboard, visible focus, 200-percent zoom, and
  reduced motion from the first pass.
- Reserve animation for meaningful state changes; specify the properties that change
  and avoid broad `transition: all` rules.
- Use real assets or an established icon library. Preserve aspect ratios and explicit
  media dimensions.
- Keep the static page useful when optional client JavaScript fails.

Render the implementation at representative desktop and mobile sizes whenever tooling
allows. Compare it with the selected target, inspect hierarchy, cropping, spacing,
type, controls, focus, and overflow, then correct visible mismatches. Run the focused
format, type, build, accessibility, and browser checks declared by the repository.

Do not deploy or publish. Report the rendered states inspected, checks run, and any
remaining visual or functional uncertainty.
