# GrepQuest brand assets

The production identity follows the selected **05 — Match Door** concept. The
room outline establishes the roguelike setting, the signal-green line represents
a successful `grep` match, its `Q`-shaped door is the exit, and the amber diamond
is the earned drop.

## Source files

- `logo/grepquest-mark.svg` is the primary transparent mark.
- `logo/grepquest-lockup.svg` pairs the mark with the custom terminal wordmark.
- `logo/grepquest-mark-monochrome.svg` is the one-color small-format fallback.
- `tokens.css` defines the canonical visual-direction colors.
- `export/` contains deterministic PNG exports generated from the SVG sources.
- `readme/grepquest-ranger-dungeon-banner.png` is the 16:9 repository hero.

Do not add shadows, gradients, checkerboard backgrounds, or extra fantasy
symbols. Keep clear space equal to one match block around the mark. At sizes
below 32 pixels, use the monochrome mark or the terminal signature `[Q]-->*`.

Regenerate raster exports from the pinned Nix development shell:

```console
just brand-build
```
