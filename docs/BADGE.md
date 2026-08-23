# GitHub profile badge

GrepQuest never writes to Git or GitHub automatically. Render the badge into a
temporary directory, inspect both files, then copy them into your profile
repository yourself:

```console
mkdir -p /tmp/grepquest-badge
grepquest badge render --out /tmp/grepquest-badge
grepquest receipt verify /tmp/grepquest-badge/grepquest-proof.json
cp /tmp/grepquest-badge/grepquest.svg assets/grepquest.svg
cp /tmp/grepquest-badge/grepquest-proof.json assets/grepquest-proof.json
```

Review `git diff` and commit explicitly. Add this to the profile README:

```markdown
[![GrepQuest proof](assets/grepquest.svg)](assets/grepquest-proof.json)
```

The local signing key lives at `$XDG_STATE_HOME/grepquest/signing-key`, or
`~/.local/state/grepquest/signing-key`, with mode `0600`. Do not commit or share
it. The public key is embedded in the receipt so `grepquest receipt verify` can
check the hash and signature without a server. This is a self-attested proof:
it proves who signed unchanged receipt data, not that an unmodified client
performed the run.
