set shell := ["bash", "-euo", "pipefail", "-c"]

# List available recipes.
default:
    @just --list

# Check the pinned development tools and host capabilities.
doctor:
    grepquest-dev-doctor

# Build the reproducible Nix package.
build:
    nix build path:.#

# Run the CLI with optional arguments, for example: just cli version
cli *args:
    go run ./cmd/grepquest {{ args }}

# Start the local tutorial.
play:
    go run ./cmd/grepquest play

# Start the sandboxed Ascii Motion MCP server. Pass --live explicitly for browser sync.
ascii-motion *args:
    grepquest-ascii-motion {{ args }}

# Regenerate the responsive Ranger runtime asset from the reviewed Ascii Motion project.
ranger-build:
    go generate ./internal/tui/rangerart

# Regenerate transparent PNG brand exports from the production SVG sources.
brand-build:
    bash brand/build.sh

# Format all Go packages.
fmt:
    go fmt ./...

# Fail when a tracked or untracked Go source file needs formatting.
fmt-check:
    #!/usr/bin/env bash
    unformatted="$(git ls-files -co --exclude-standard -z -- '*.go' | xargs -0 --no-run-if-empty gofmt -l)"
    if [[ -n "$unformatted" ]]; then
      printf 'Go files need formatting:\n%s\n' "$unformatted" >&2
      exit 1
    fi

# Run Go unit tests.
test:
    go test ./...

# Run Go tests with the race detector.
test-race:
    go test -race ./...

# Run the Go static analyzer.
vet:
    go vet ./...

# Run the configured Go linter.
lint:
    golangci-lint run ./...

# Run all required Go checks.
check-go: fmt-check vet test-race lint

# Build the Sui Move package with the pinned CLI.
move-build:
    sui move build --path move/grepquest

# Run the Sui Move package tests with the pinned CLI.
move-test:
    sui move test --path move/grepquest

# Build and test the Sui Move package.
move-check: move-build move-test

# Check Nix formatting and all flake outputs for the current system.
nix-check:
    nixfmt --check flake.nix
    nix flake check path:.#

# Run the normal Go, Move, and Nix source gates.
check: check-go move-check nix-check

# Build the Firecracker guest artifacts without booting a VM.
vm-build:
    nix build path:.#guestKernel path:.#guestInitrd path:.#guestRootfs --cores 2 --max-jobs 1

# Build and boot the real Firecracker acceptance VM; requires writable /dev/kvm.
vm-test:
    grepquest-test-firecracker

# Audit a pre-provisioned production Firecracker jail and cgroup boundary.
vm-host-check:
    grepquest-firecracker-host-check

# Build a host CLI archive with checksums and dependency license texts.
release-bundle output="dist":
    bash tools/release-bundle.sh "{{ output }}"
