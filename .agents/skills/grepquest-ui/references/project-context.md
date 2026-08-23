# GrepQuest web design context

Use this reference for GrepQuest landing-page and website work. Repository source and
explicit user decisions outrank it when the product changes.

## Product truth

GrepQuest is a local-first terminal roguelite for learning GNU `grep`. Players solve
real command-line challenges, receive deterministic feedback, build mastery, and earn
cosmetic equipment. Generated content is cosmetic; only the deterministic judge can
decide whether an answer is correct.

Verify current claims against these sources before publishing them:

1. `README.md` for status, installation, supported platforms, and commands.
2. `internal/buildinfo/VERSION` and `CHANGELOG.md` for the current version.
3. `packs/core/v1/pack.json` for challenge count and pack identity.
4. `SECURITY.md` for sandbox and integration boundaries.
5. `LICENSE`, `packs/core/v1/LICENSE`, and `THIRD_PARTY_NOTICES.md` for licensing.
6. `CONTRIBUTING.md` for contribution paths and checks.

Never claim that the project is production-ready, supports an unverified platform,
has a secure public multi-user sandbox, or can prove honest local play. AI, OpenRouter,
Sui, wallets, public proofs, and badges are optional supporting layers, not the first
screen's product story.

## Audience and page job

Primary audiences are CLI learners, experienced terminal users who want deeper grep
mastery, and open-source contributors. The landing page has two primary outcomes:

1. Help a visitor understand the game and reach the supported install path.
2. Help a potential contributor inspect the repository and find a concrete entry point.

A new visitor should understand within the first viewport that GrepQuest is a free,
open-source terminal learning game built around real GNU grep commands and
deterministic judging.

## Brand thesis

The durable creative idea is **Signal in the Cellar**: the player filters a useful
signal from dark, noisy data while descending through a compact terminal world. The
experience should feel precise, mysterious, practical, and earned.

Use the product's own materials as the visual vocabulary:

- grep matches, patterns, paths, exit codes, and bounded command output;
- rooms, challenge packs, acts, mastery, levels, and recall;
- actual TUI panels and real in-game inventory items;
- hashes and proof metadata only as quiet technical evidence.

Avoid default aesthetics that are unrelated to the product:

- Matrix rain, constant glitch, fake CRT damage, or unreadable scanlines;
- generic neon cyberpunk, crypto/NFT imagery, or speculative-finance language;
- medieval fantasy pasted over a terminal product;
- interchangeable SaaS cards, gradient blobs, and invented dashboard metrics;
- a fake macOS terminal window when an actual GrepQuest capture can be used;
- monospace body copy across the entire page.

## Content hierarchy

Prefer this narrative order unless the selected direction justifies a better one:

1. What it is and why it is worth trying.
2. How one room works: understand, command, judge, progression.
3. Evidence from the real TUI and real challenge content.
4. Why deterministic, local-first learning is trustworthy.
5. Mastery, loot, and recall as motivation.
6. Open-source contribution paths and current development status.
7. A verified install path and final contribution call to action.

Keep optional AI and Sui details below the core learning story. Mark prerelease status
prominently enough to set expectations without turning the page into a warning screen.

## Visual starting point

Treat these as starting constraints, not a substitute for exploration:

- Dark-first editorial canvas with restrained signal green for matches and success.
- Warm amber for progress and loot; cool cyan for paths and technical metadata.
- Near-white readable body text and quiet green-gray secondary text.
- A proportional sans for prose and a carefully chosen mono face for commands,
  labels, and product captures.
- Lightly rounded or precise angular surfaces; avoid oversized pill-shaped UI.
- One memorable signature element, supported by otherwise disciplined composition.
- Real GrepQuest TUI capture as primary product evidence.

Candidate color roles from the initial brief, subject to contrast testing and the
selected direction:

| Role | Candidate | Purpose |
| --- | --- | --- |
| Void | `#080C0B` | Page background |
| Surface | `#121D18` | Panels and cards |
| Line | `#294037` | Borders and structural rules |
| Text | `#EAF2EC` | Primary text |
| Muted | `#A8B8AD` | Secondary text |
| Signal | `#78F46A` | Match, success, primary action |
| Amber | `#F2C45E` | Progress and loot |
| Cyan | `#74D7E8` | Paths and metadata |

Do not treat the candidate values as approved tokens until a direction is selected and
contrast is verified in real components.

## Interaction principles

- The primary action must be obvious without motion.
- Motion should explain a command, match, clear, or drop state; stop after the moment.
- Respect `prefers-reduced-motion` with a complete static alternative.
- Keep navigation, content, and install commands usable without client JavaScript.
- Copy controls, disclosures, and animation controls are progressive enhancement.
- Interactive targets, visible focus, semantic structure, and readable zoom are part
  of visual quality, not a separate cleanup phase.

## Implementation direction

The target source location is `website/`. For a new site, prefer a static,
content-first architecture such as Astro with TypeScript and ordinary CSS custom
properties, integrated into the pinned Nix development environment. Do not introduce
SSR, a database, a CMS, a large client framework, analytics, or external font requests
without a demonstrated product need and user approval where the scope expands.

Use actual screenshots or captures for visible product imagery. If an asset is missing,
create it through an authorized image-generation or capture workflow and inspect it
before use. Do not replace missing imagery with placeholder boxes, emoji, CSS drawings,
or improvised inline SVG.

## Quality floor

- Responsive from 320 CSS pixels through wide desktop layouts with no page-level
  horizontal overflow.
- WCAG 2.2 AA contrast, keyboard access, semantic headings, visible focus, and a skip
  link.
- Text resize to 200 percent without lost content or functionality.
- Static dimensions or `aspect-ratio` for media to avoid layout shifts.
- Honest empty, loading, error, copied, paused, and reduced-motion states where used.
- Real rendered inspection at mobile and desktop sizes before handoff.
- No deployment or public release without explicit user authorization.
