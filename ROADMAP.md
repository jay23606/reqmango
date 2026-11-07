# Roadmap

This is a direction, not a schedule. Work happens in the open as issues and
pull requests; anything here can move if a bug, a security report, or a
well-argued use case gets in front of it.

The source of truth for what is finished is [CHANGELOG.md](CHANGELOG.md) and the
[releases](https://github.com/vinthuy/reqmango/releases). The internal pipeline
status (Chinese) lives in [docs/dev/pipeline-status.md](docs/dev/pipeline-status.md).

## Now — make the project easy to adopt and safe to self-host

- [x] One-time security baseline: demo seed data opt-in, bootstrap admin,
      `SECURITY.md`, production checklist (v0.1.0)
- [x] Tagged releases and multi-arch `ghcr.io` images (v0.1.0)
- [x] Documentation site (MkDocs, published from `docs/`) with a browsable
      quickstart, deployment, configuration and API reference
- [x] Language handling that follows the browser on first visit instead of
      hardcoding Chinese, with a visible switcher
- [x] Community files and automation: code of conduct, support guide,
      Dependabot, protected `master`, public roadmap (this file)
- [x] Screenshots for the README, covering intake, triage, issues, cycles and
      analytics (`docs/assets/screenshot-*.png`)

## Next — close the remaining work-item gaps

**Shipped in code** on branch `feat/mandatory-fields` /
[PR #32](https://github.com/vinthuy/reqmango/pull/32) (CI green). SRS + acceptance
tables: [docs/specs/](docs/specs/README.md), [docs/dev/acceptance/](docs/dev/acceptance/).
Process: [docs/dev/sdd.md](docs/dev/sdd.md). Historical notes:
[Plane gap analysis](docs/dev/plane-gap-implementation-plan.md).

| Candidate | Status |
|-----------|--------|
| Quick create from the work-item list | Done (P0; spot-check list/board in UI) |
| Mandatory field validation + conditional fields | Done (P0; import CF column mapping still a follow-up) |
| Type hierarchy rules | Done (P0; admin UI spot-check) |
| Sub-items panel | Done (P0) |
| Page linking | Done (P0) |
| Notification auto-trigger | Done (P0; due-soon remains P1) |
| Attachment backend | Done (P0) |

## Later — deeper workflows

- [x] [Merge duplicate work items](docs/specs/2026-11-07-merge-duplicates.md) ([PR #33](https://github.com/vinthuy/reqmango/pull/33))
- [x] Time tracking and estimate categories (P0: issue estimate points + manual time log + settings mount — `docs/specs/2026-11-07-time-tracking-estimates.md`; categories/time-mode on issues still Later)
- Recurring work items
- Release management polish and a release picker in the item view

## Not planned

- The archived **standalone agent platform / full-SDLC agent** direction. It is
  kept in [`docs/superseded/`](docs/superseded/) for history and is not on the
  roadmap. AI stays a layer inside Issue / Intake / Cycle.
- MCP/CLI as a growth story. The SDK exists and is maintained, but it is not the
  default product path.
- A paid/cloud edition. Reqmango is self-hosted and MIT-licensed.

## How priorities are chosen

1. Security and tenancy defects.
2. Data loss or corruption.
3. Install/upgrade failures.
4. Issues with the most 👍 and the clearest reproduction.
5. The candidates above.

If you want something sooner, the most effective move is a focused pull request
or a well-scoped issue — see [CONTRIBUTING.md](CONTRIBUTING.md).
