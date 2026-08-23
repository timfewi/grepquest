# Third-party notices

GrepQuest source code is MIT-licensed. Its Go dependencies retain their own
licenses, including Bubble Tea, Bubbles, and Lip Gloss (MIT), and modernc.org
SQLite. Distribution artifacts must include the dependency license texts.
`just release-bundle` creates a host CLI archive containing the discovered
top-level Go module license texts, this notice, and the core-pack license.
Review the generated manifest before publishing it.

The bundled GrepQuest Core Challenge Pack is original project content licensed
separately under CC-BY-4.0. Redistributions must retain its attribution and the
license notice in `packs/core/v1/LICENSE`.

GNU grep is executed as an independent program and is licensed under GPL-3.0-
or-later. The Nix guest artifacts also contain BusyBox (GPL-2.0-only) and a
Linux kernel (GPL-2.0-only with the Linux syscall note). A release that
redistributes those artifacts must include the corresponding licenses and
satisfy each component's source-distribution obligations; the guest image is
not covered wholesale by GrepQuest's MIT license.

Firecracker and the Sui Move framework are Apache-2.0 licensed. Firecracker is
not currently redistributed by this repository. The Move source links to the
network's published Sui framework when built or published; the framework's
license remains independent of GrepQuest's MIT license.

The development shell pins `ascii-motion-mcp` 2.2.0 and its npm dependency
closure for authoring ASCII assets. `ascii-motion-mcp` is MIT-licensed and is
not part of the GrepQuest runtime package. Its transitive packages retain their
own licenses; redistribution of the development package must preserve those
notices.
