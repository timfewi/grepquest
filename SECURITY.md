# Security

Please report vulnerabilities privately through GitHub's security advisory
flow. Do not include credentials, wallet keys, or sensitive fixture data.

## Supported versions

GrepQuest is currently a pre-release project. Security fixes are made only on
the latest `main` branch and the newest `0.1.0-dev` prerelease. Older snapshots
are unsupported. Do not use the development native sandbox as a public or
multi-user execution service.

GitHub private vulnerability reporting must be enabled before the repository
is announced publicly. If it is unavailable, contact the maintainer privately
instead of opening a public issue; do not send live secrets or wallet keys.

## Current boundary

The `native` sandbox is a developer convenience, not a security boundary. It
parses only a `grep` argv, rejects shell operators, runs with a minimal
environment and bounded output/time, and uses disposable fixtures, but it is
still a host process. Do not use untrusted challenge packs with it.

The Firecracker backend is the intended security boundary for untrusted player
input. It verifies full SHA-256 digests for the kernel, initrd, and rootfs;
configures no NIC or host filesystem share; mounts the rootfs read-only; and
exposes only a bounded JSON protocol over vsock. The guest agent drops to
UID/GID 1000 for GNU grep execution. Firecracker's optional jailer and host
cgroup hardening remain required before a public multi-user deployment.

OpenRouter is optional and keys remain host-only runtime values. The adapter
accepts only OpenRouter's official HTTPS endpoint, refuses redirects, requests
ZDR endpoints with provider data collection denied, validates structured
outputs, strips terminal control sequences, and never passes its environment to
the guest. Keys must never appear in logs, receipts, child environments, Nix
derivations, or guest images.

Sui integration is opt-in and delegates signing to an external Sui CLI. The
application does not store wallet keys, switch networks, request faucet funds,
or publish packages. Score submission defaults to preview-only, validates the
signed local receipt, verifies the active network and wallet against the
requested values, and requires an exact network/wallet/receipt confirmation
before execution. Wallet subprocesses receive an allowlisted environment that
excludes OpenRouter and GitHub credentials. Public
GraphQL reads reject cleartext remote endpoints and redirects, bound response
size and pagination, and treat every event field as untrusted input.
