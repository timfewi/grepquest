# Release process

`0.1.0-dev` is a development snapshot, not a production security claim. The
native sandbox must remain labeled as trusted-local-only until the Firecracker
host boundary has completed a separate production review.

## Source readiness

1. Confirm `internal/buildinfo/VERSION` and `CHANGELOG.md` describe the intended
   version. Nix derives package versions from the same file.
2. Run formatting, unit tests, race tests, vet, lint, module verification, and
   `nix flake check path:.#` from a clean checkout.
3. Enter the pinned development shell, run `grepquest-dev-doctor`, then run
   `sui move build --path move/grepquest` and
   `sui move test --path move/grepquest`. Publishing is not a release test.
4. Run `grepquest-test-firecracker` only on a dedicated x86-64 Linux host with
   accessible `/dev/kvm`. The command builds and selects the three fresh,
   digest-addressed guest artifacts without changing host permissions.
5. Review the repository for credentials, wallet material, local state,
   generated receipts, private fixtures, and machine-specific paths.
6. Review dependency licenses and vulnerability results. Binary distributions
   must ship the required license texts described in `THIRD_PARTY_NOTICES.md`.
7. Run `just release-bundle` from a clean tree. Inspect the generated dependency
   manifest, archive contents, and SHA-256 checksum before uploading anything.

## GitHub repository settings

- Keep Actions workflow permissions read-only by default.
- Protect `main` with pull requests, required CI checks, and resolved review
  conversations. Require an independent approval once another maintainer is
  available.
- Enable private vulnerability reporting and Dependabot security updates.
- Disable force pushes and branch deletion for `main`.
- Add repository topics, description, homepage, and the MIT license indicator.

## Development tag

After the source checks pass, review the complete staged diff, create a signed
SemVer prerelease tag such as `v0.1.0-dev`, and publish release notes from the
matching changelog entry. Attach only reproducible artifacts with SHA-256
checksums and their third-party license bundle. Tagging, pushing, and creating a
GitHub release are manual maintainer actions.

Before a stable or publicly hosted release, complete Firecracker jailer/cgroup
hardening, define supported platform/toolchain versions, and publish the Sui
package IDs and upgrade-cap custody model if the optional social layer is used.
Use `just vm-host-check` as a prerequisite audit, not as a substitute for the
external production supervisor described in `docs/FIRECRACKER.md`.
