# GrepQuest website

This directory contains the static GrepQuest landing page. It follows the selected
**Living Terminal** direction documented in `docs/design-directions/README.md`.

The website source and production assets are versioned with the repository.
Generated `dist`, local Astro state, dependencies, and visual-review captures
remain ignored. Deployment is a separate, explicitly authorized release action.

## Local development

Run commands from `website/` inside the reviewed repository environment:

```console
direnv exec . bun install
direnv exec . bun run dev
direnv exec . bun run build
```

The static production output is written to `website/dist/`. The Astro configuration
serves the website from the domain root `/`.

## Visual verification

`tests/render-review.mjs` captures desktop, tablet, mobile, reduced-motion, and
200-percent text-size states. It also checks page overflow, internal anchors,
interactive feedback, external requests, and axe-core accessibility rules. Set
`PLAYWRIGHT_CHROMIUM_EXECUTABLE` to a compatible local Chromium executable before
running it with `bun tests/render-review.mjs`.

## Asset and dependency provenance

- The Match Door logo is copied from `brand/logo/`.
- The transparent Ranger PNG is copied from
  `assets/ascii-motion/reference/grepquest-ranger-reference-transparent.png`.
- The Ranger ASCII frame is generated from the reviewed 60 × 60 ASCII Motion source
  used by `internal/tui/rangerart`.
- IBM Plex Sans and IBM Plex Mono are bundled locally through Fontsource and licensed
  under OFL-1.1.
- The GitHub icon is supplied by Simple Icons under CC0-1.0.
- Astro is MIT-licensed, Playwright is Apache-2.0-licensed, and axe-core Playwright is
  MPL-2.0-licensed.

The page makes no font, analytics, tracker, or UI-CDN requests.
