# ASCII Motion workflow

GrepQuest pins `ascii-motion-mcp` 2.2.0 as a development-only Nix package. It
provides the command-line MCP server used to author, preview, and export ASCII
animation projects. The game does not launch Node.js, a browser, or an MCP
server at runtime.

## Start the authoring server

From the reviewed development shell:

```console
just ascii-motion
```

The default command uses stdio transport, disables network access, and exposes
only `assets/ascii-motion` as writable `/project` storage. Configure an MCP
client to launch `grepquest-ascii-motion`; do not point the tool at the
repository root.

Browser synchronization is an explicit, network-enabled mode:

```console
just ascii-motion --live
```

Only use live mode while actively editing. The wrapper accepts only `--live`,
`--port`, `--help`, and `--version`. It rejects arbitrary paths and executables
outside the Nix store.

## Asset lifecycle

1. Edit `assets/ascii-motion/projects/grepquest_ranger_ascii.asciimtn` through
   the MCP tools. The reviewed Bubble Tea export currently contains one 60x60
   true-color Ranger frame.
2. Review the session as untrusted input: verify its canvas, coordinates, colors,
   layer visibility, and property tracks.
3. Run `just ranger-build`. The repository generator composites the visible
   source layers into `internal/tui/rangerart/asset_generated.go`.
4. Review the generated cell table against
   `exports/grepquest_ranger_char_bubble_tee_component.go.txt`. The runtime fits
   the complete export responsively and never layers substitute line art over it.
5. Keep raw upstream exports in `assets/ascii-motion/exports` with a non-Go file
   suffix. They are provenance, not runtime code.

The deterministic judge, challenge data, XP, and loot selection must never
depend on animation state. `GREPQUEST_REDUCED_MOTION=1` selects complete static
idle/CLEAR states and disables animation ticks.

## Trust boundary

The upstream tool is third-party code. The wrapper uses Bubblewrap with a
read-only Nix store, an isolated temporary home, no inherited environment, and
an asset-directory-only bind mount. Network isolation is relaxed only for the
explicit `--live` mode. Generated text and tool output cannot authorize file
changes elsewhere in the repository.
