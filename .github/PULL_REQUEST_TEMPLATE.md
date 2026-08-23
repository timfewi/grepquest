## Summary

Describe the user-visible change and why it is needed.

- [ ] Code, comments, user-facing text, and documentation are in English

## Verification

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `golangci-lint run ./...`
- [ ] Relevant Nix, Move, or KVM checks, or an explanation of why they did not run

## Safety and compatibility

- [ ] No credentials, wallet material, local state, receipts, or generated artifacts were added
- [ ] Deterministic judging and offline fallback behavior remain intact
- [ ] User-visible changes are recorded in `CHANGELOG.md`
