# Governance

GrepQuest is currently a maintainer-led project. The repository owner is the
final decision maker for releases, security boundaries, contributor access,
and project scope.

## Decisions

- Small implementation decisions are made in focused pull requests.
- User-visible behavior and architecture changes should include rationale,
  tests, and documentation.
- Security-sensitive changes require explicit threat-boundary review and may
  be held until an appropriate host or integration test is available.
- Generated text remains cosmetic; deterministic judging is a project invariant.

## Maintainers and contributors

Maintainers may triage issues, review and merge pull requests, manage releases,
and enforce the Code of Conduct. Contributor access is granted gradually based
on sustained, constructive work and may be removed to protect the project or
community.

When another active maintainer joins, protected-branch rules should require an
independent approval. Until then, pull requests must pass all required CI and
resolve review conversations.

## Security and continuity

Private vulnerability reports take priority over public feature work. Release
signing keys, wallet capabilities, and infrastructure credentials are never
shared through the repository. If the project becomes inactive, the maintainer
may appoint a successor or archive the repository with its security boundaries
and support status clearly documented.
