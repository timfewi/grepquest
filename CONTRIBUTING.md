# Contributing to GrepQuest

Thanks for helping improve GrepQuest. Small, focused pull requests are easiest
to review. Participation is governed by the
[Code of Conduct](CODE_OF_CONDUCT.md).

All source code, comments, identifiers, user-facing text, documentation,
changelog entries, issue text, and pull request text must be written in English.
Non-English fixture content is acceptable only when a test or lesson explicitly
depends on that language.

## Development environment

Linux and Nix are the supported environment:

```console
nix develop path:.#
grepquest-dev-doctor
just check
```

Run `just --list` to see focused recipes such as `just check-go`,
`just move-check`, and `just nix-check`. The complete `just check` recipe runs
the normal Go, Move, and Nix source gates without booting a VM.

The shell provides the Sui CLI version pinned in `flake.nix`. Changes to
`move/grepquest` must build and test with that version. `SUI_CONFIG_DIR` points
to an isolated configuration with no private keys, so these commands cannot
use a maintainer wallet:

```console
sui move build --path move/grepquest
sui move test --path move/grepquest
```

If a required check cannot run, explain why in the pull request. Do not publish
a Move package or submit a transaction as part of validation.

On x86-64 Linux, the shell also provides Firecracker. `/dev/kvm` access is a
host responsibility and is never changed by Direnv or Nix. Run the opt-in real
microVM acceptance test only on a suitable dedicated development host:

```console
grepquest-dev-doctor
grepquest-test-firecracker
```

## Pull requests

- Add or update tests for observable behavior.
- Preserve deterministic judging and the offline first-run path.
- Keep challenge fixtures bounded and do not include hidden-test data in guest
  images.
- Update `CHANGELOG.md` for user-visible changes.
- Do not commit credentials, wallet keys, generated receipts, local databases,
  build output, or machine-specific paths.

Report vulnerabilities through GitHub's private security advisory flow rather
than a public issue. See [SECURITY.md](SECURITY.md).

By contributing, you agree that your contributions are licensed under the MIT
license in [LICENSE](LICENSE). Challenge-pack contributions under `packs/` are
also provided under the license declared by that pack.
