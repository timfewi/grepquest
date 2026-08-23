# Optional Sui social layer

GrepQuest remains fully local without Sui. The Move package in
`move/grepquest` is an optional public sharing layer for signed receipts; it is
not part of judging and cannot establish that a self-attested run was honest.
No wallet key is read or stored by GrepQuest. Writes are delegated to an
installed Sui CLI and the wallet configured there pays gas.

## Model

- `Season` is immutable and binds the season ID to pack/rules SHA-256 digests.
- `PlayerProfile` is address-owned and stores a per-season best in a child
  `Table`; independent players do not write a shared leaderboard object.
- `Achievement` and `LoadoutSnapshot` are optional address-owned exports.
- `ScoreSubmitted` events supply bounded public board data. The client filters
  an exact season and exact proof tier, deduplicates each player, and ranks
  locally. Speed is a separate hardware-dependent category.

The package distinguishes `self` (0), `verified` (1), and `official` (2). A
wallet signature identifies the submitter but does not upgrade the proof tier.
The player entry point accepts only tier 0. Tier 1/2 events use a separate
`VerifierCap` entry point; the admin can issue that least-privilege capability
to a transparent CI or official verifier without giving it season authority.
Both submission entry points require the signed receipt's season, pack digest,
and rules digest to match the immutable `Season` object.

## Build and test

Install a current Sui CLI, then run:

```console
sui move build --path move/grepquest
sui move test --path move/grepquest
```

The Nix development shell supplies the repository-pinned CLI and points
`SUI_CONFIG_DIR` at an isolated configuration with an empty keystore. This
prevents build and test commands from creating or using a maintainer wallet.
Commands that intentionally use an existing wallet must explicitly override
that environment variable with the wallet owner's configuration directory;
never store that directory or its keys in the repository or Nix store.

Publishing is intentionally manual. Use testnet first, inspect the complete
publish transaction in the wallet, and retain the generated publication
metadata. After publication, commit `Published.toml` and `Move.lock` from the
same source tree and verify against the selected network:

```console
sui client switch --env testnet
sui client verify-source move/grepquest
```

Source verification rebuilds and compares the root package bytecode and
linkage. It is required before describing a deployment as source-verified.
Before enabling a verified board, also publish the package ID, source commit,
upgrade policy/cap custody, immutable Season object ID, and the verifier-cap
holders. A later package upgrade changes the trust boundary even when an older
version previously verified.

## Read a board

The public endpoint is rate-limited, so reads are explicitly bounded to at
most 20 pages of 50 events. Self-attested and verified results are never mixed.

```console
grepquest sui leaderboard \
  --package 0xPACKAGE --season core-v1 --tier self --metric score
grepquest sui leaderboard \
  --package 0xPACKAGE --season core-v1 --tier verified --metric recall
```

Supported metrics are `score`, `accuracy`, `recall`, `no-hint`, and `speed`.
For a larger community, use a dedicated provider or indexer rather than
increasing the client bound.

## Preview and submit a score

First render a receipt, create an address-owned `PlayerProfile`, and obtain the
immutable `Season` object ID. The default action is a preview and never opens a
wallet:

```console
grepquest sui submit \
  --receipt grepquest-proof.json \
  --package 0xPACKAGE --profile 0xPROFILE --season-object 0xSEASON \
  --network testnet --wallet 0xEXPECTED_WALLET
```

Add `--execute` only after checking the package, object IDs, score, receipt
hash, proof tier, metrics, and maximum gas budget shown in the preview. The
command reads the Sui CLI's active network and wallet, refuses either mismatch,
and asks for an exact phrase containing the network, wallet address, and receipt
prefix. Only then does it invoke `sui client call`, directly and without a
shell.

Testnet and mainnet writes are never automatic. GrepQuest does not request
faucet funds, switch networks, publish packages, change the active wallet, or
upload local progress on its own.
