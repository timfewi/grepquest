# ASCII Motion workspace

This directory is the only writable project path exposed to the development-only
Ascii Motion MCP server. The GrepQuest runtime never starts Node.js, the MCP server,
or a browser connection.

Run the pinned, offline server from the repository development shell:

```console
grepquest-ascii-motion
```

An MCP client communicates with that process over standard input/output. To connect
the optional Ascii Motion browser explicitly, use:

```console
grepquest-ascii-motion --live
```

Live mode shares the host network namespace so the local browser can connect. The
wrapper still clears the environment and uses Bubblewrap to expose only this
directory as writable `/project`. It rejects arbitrary arguments, alternate project
directories, symlinked asset roots, and invalid ports.

The reviewed Ranger source project lives in
`projects/grepquest_ranger_ascii.asciimtn`. Its original Bubble Tea export is
kept as `exports/grepquest_ranger_char_bubble_tee_component.go.txt` for
provenance; the `.txt` suffix prevents its upstream v1 imports from entering the
Go build. Run `just ranger-build` to regenerate the equivalent source cell table
in `internal/tui/rangerart/asset_generated.go`. The runtime renders only those
exported glyphs and colors, fitting the complete 60x60 canvas to the available
terminal space without a separate line-art overlay. Review the project, export,
and generated source as ordinary code before use.

The checked-in `*.storyboard.txt` files remain human-reviewable design notes.
Commit only small, reviewed source sessions required to reproduce runtime art.

`reference/grepquest-ranger-reference-transparent.png` is the approved visual
source for the player character. Its alpha cutout and front view are intentional
inputs for silhouette tracing and value reduction. The flat-background original
is retained beside it for provenance and edge comparison.
