# GrepQuest

![GrepQuest Ranger in the Signal Cellar.][readme-hero]

[readme-hero]: brand/readme/grepquest-ranger-dungeon-banner.png

GrepQuest is a local-first terminal roguelite for learning GNU `grep`. Every
answer is executed by a deterministic judge; generated text never decides
whether a solution is correct.

> **Status: `0.1.0-dev`.** The core tutorial is usable, but APIs, challenge
> formats, receipts, and the optional Sui package can still change. The native
> sandbox is for trusted local development and is not suitable for hosting
> untrusted users.

![GrepQuest Living Terminal showing the mission, Ranger, and command buffer.][readme-tui]

[readme-tui]: brand/readme/grepquest-living-terminal.png

## Install and run

Linux with Nix is the supported environment:

```console
nix build path:.#
./result/bin/grepquest version
./result/bin/grepquest challenges
./result/bin/grepquest play
```

For a source checkout without building the Nix package:

```console
nix develop path:.#
go run ./cmd/grepquest version
go run ./cmd/grepquest play
```

The tutorial works offline and does not need an account, wallet, or API key.

## Development

The development shell pins Go, GNU grep, SQLite, formatters, linters, the Sui
CLI, and Firecracker on x86-64 Linux:

```console
direnv exec . grepquest-dev-doctor
direnv exec . go test ./...
direnv exec . sui move test --path move/grepquest
direnv exec . go run ./cmd/grepquest play
```

The development shell also provides `just` shortcuts for the common workflows:

```console
direnv exec . just --list
direnv exec . just move-check
direnv exec . just brand-build
direnv exec . just ascii-motion --help
direnv exec . just check
```

The pinned ASCII Motion authoring server is development-only and sandboxed to
`assets/ascii-motion`; the game has no Node.js or browser runtime dependency.
See [docs/ASCII_MOTION.md](docs/ASCII_MOTION.md) for the offline-first workflow
and the explicit `--live` browser-sync mode. Set
`GREPQUEST_REDUCED_MOTION=1` to keep the terminal player character static.

The expensive Firecracker workflows remain explicit as `just vm-build` and
`just vm-test`; the normal `just check` source gate does not boot a VM.

If Direnv has not been approved, use
`nix develop path:.# --command go test ./...`; review `.envrc` before approving
it. The first environment build downloads the pinned official Sui release;
Nix verifies its SHA-256 digest and caches it in the Nix store. The shell never
creates a wallet, switches networks, or submits a transaction. It points
`SUI_CONFIG_DIR` at an isolated, empty, read-only development configuration so
Move builds cannot accidentally use a maintainer wallet. See
[CONTRIBUTING.md](CONTRIBUTING.md) for the full contribution checks.

The initial native sandbox is for development only. It accepts a single
`grep` invocation, rejects shell syntax, bounds execution time and output, and
runs against a disposable fixture directory. It is not a security boundary;
the Firecracker backend is the production target.

Set `OPENROUTER_API_KEY` to enable optional personalized cosmetic loot. The key
is read only at runtime and is not stored. `GREPQUEST_AI_MODEL` selects a model;
the default is `openai/gpt-4o-mini`. Requests require structured outputs, ZDR,
and `data_collection: deny`. SQLite caches validated responses and enforces ten
requests per session and thirty per UTC day. Any missing key, provider failure,
invalid output, or exhausted budget falls back to built-in offline loot without
affecting judging, XP, rarity, or item IDs.

Badge exports use canonical version-1 receipts signed by a local Ed25519 key.
See [docs/BADGE.md](docs/BADGE.md) for the explicit profile README flow and its
self-attestation limitations.

The optional Sui social layer uses address-owned profiles, immutable seasons,
public events, exact wallet/transaction confirmation, and client-side ranking
with self-attested and verified tiers kept separate. It never runs during local
play. See [docs/SUI.md](docs/SUI.md).

## Commands

```console
grepquest version
grepquest play                 # start the tutorial
grepquest play --sandbox firecracker \
  --firecracker-binary "$GREPQUEST_FIRECRACKER" \
  --kernel "$GREPQUEST_KERNEL" --kernel-sha256 "$GREPQUEST_KERNEL_SHA256" \
  --initrd "$GREPQUEST_INITRD" --initrd-sha256 "$GREPQUEST_INITRD_SHA256" \
  --rootfs "$GREPQUEST_ROOTFS" --rootfs-sha256 "$GREPQUEST_ROOTFS_SHA256"
grepquest challenges           # list bundled challenges
grepquest badge render --out . # render local SVG and proof JSON
grepquest receipt verify grepquest-proof.json
grepquest sui leaderboard --package 0xPACKAGE --season core-v1 --tier self
grepquest sui submit --receipt grepquest-proof.json --package 0xPACKAGE \
  --profile 0xPROFILE --season-object 0xSEASON --wallet 0xWALLET # preview only
```

Progress is stored beneath `$XDG_STATE_HOME/grepquest` (or
`~/.local/state/grepquest`).

## MicroVM artifacts

On x86-64 Linux, Nix builds all three digest-addressed guest artifacts:

```console
nix build .#guestKernel .#guestInitrd .#guestRootfs --cores 2 --max-jobs 1
```

The root filesystem is mounted read-only. Stage one switches to that root;
stage two mounts `/run` as `tmpfs` and starts the static guest agent. The host
does not configure a NIC. The agent installs fixtures as root but launches GNU
grep as UID/GID 1000. Hidden tests remain in the host judge and are not sent to
the guest with the visible challenge.

The development shell includes Firecracker automatically on x86-64 Linux, but
KVM remains a host capability: the kernel must expose `/dev/kvm`, and the
current user must be able to read and write it. Check both tool and device
access with `grepquest-dev-doctor`.

The real KVM acceptance test remains opt-in. Run
`grepquest-test-firecracker` from the repository root; it builds the three
guest artifacts, passes their Nix store paths to the test, and starts the real
microVM test. It does not change host permissions or use `sudo`.

Production hosting requires an external jailer/cgroup supervisor. The
read-only `just vm-host-check` command and the remaining trust assumptions are
documented in [docs/FIRECRACKER.md](docs/FIRECRACKER.md).

## Sui Move agent skills

Portable, source-backed Sui Move skills live under `.agents/skills`. They cover
task routing, contract authoring, review, and interactive learning. Set
`SUI_MOVE_KB_ROOT` to the companion local knowledge repository or put
`move-kb` on `PATH` for commit-bound documentation searches; the skills fall
back to official Sui sources when the local database is unavailable.

## Project and license

- Changes: [CHANGELOG.md](CHANGELOG.md)
- Contributions: [CONTRIBUTING.md](CONTRIBUTING.md)
- Community standards: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- Governance: [GOVERNANCE.md](GOVERNANCE.md)
- Support: [SUPPORT.md](SUPPORT.md)
- Security reporting and current boundaries: [SECURITY.md](SECURITY.md)
- Release checklist: [docs/RELEASING.md](docs/RELEASING.md)
- Third-party obligations: [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)

GrepQuest source is licensed under the [MIT License](LICENSE). Bundled challenge
packs and third-party components retain the licenses documented beside them and
in the third-party notices.
