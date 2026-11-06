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

Requirements for this tranche are drafted as SRS (Given/When/Then) under
[docs/specs/](docs/specs/README.md); process: [docs/dev/sdd.md](docs/dev/sdd.md).
Freeze those specs before design/plan. Historical task notes:
[Plane gap analysis](docs/dev/plane-gap-implementation-plan.md).

| Candidate | Why it matters |
|-----------|----------------|
| Quick create from the work-item list | Removes a modal from the most frequent action |
| Mandatory field validation + conditional fields | Makes custom fields trustworthy, not advisory |
| Type hierarchy rules | Keeps deep hierarchies (6 levels) coherent |
| Sub-items panel | Parent items are otherwise a dead end in the UI |
| Page linking | Pages exist; connecting them to work items is the missing half |
| Notification auto-trigger | The notification centre only pays off if events reach it |
| Attachment backend | Attachments are currently the weakest part of the item model |

## Later — deeper workflows

- Merge duplicate work items (the intake queue already suggests duplicates)
- Time tracking and estimate categories
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
