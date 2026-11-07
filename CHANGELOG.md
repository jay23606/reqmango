# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[semantic versioning](https://semver.org/spec/v2.0.0.html). While the major
version is `0`, a minor bump may include breaking changes; they are called out
under **Changed** / **Upgrade notes**.

## [Unreleased]

### Added

- Time tracking & estimate points P0: persist `estimate_point_id` on issues, project Estimate settings UI, manual time log (seconds), `time_track:*` / `settings:manage` enforcement, CSV estimate import mapping (`docs/specs/2026-11-07-time-tracking-estimates.md`).
- Soft-merge duplicate issues into a target without deleting sources ([PR #33](https://github.com/vinthuy/reqmango/pull/33)): cancelled state when available, duplicate relation, label/assignee union, detail UI 「合并到…」.
- Work-item gap tranche from ROADMAP Next ([PR #32](https://github.com/vinthuy/reqmango/pull/32)):
  mandatory and conditional custom fields, list/board quick create with
  `issue:create` gating, type hierarchy rules and admin fields, sub-items panel
  hardening, issue↔page linking UI, P0 notification auto-triggers, and
  attachment size/permission/count locks. Specs under `docs/specs/2026-11-06-*`;
  acceptance notes under `docs/dev/acceptance/`.

### Changed

- CI moved every GitHub Action to the major that runs on Node 24 natively
  (`checkout@v7`, `setup-go@v7`, `setup-node@v7`, `setup-python@v7`,
  `upload-artifact@v7`, `configure-pages@v6`, `upload-pages-artifact@v5`,
  `deploy-pages@v5`, `docker/*-action@v4`/`v6`/`v7`). GitHub had deprecated the
  Node 20 action runtime and printed a warning on every job; nothing changes for
  contributors. The Go jobs also pin `cache-dependency-path` to `backend/go.sum`,
  so the module cache is restored instead of re-downloaded on every run.
- CI now pins `ubuntu-24.04` instead of `ubuntu-latest`. GitHub is migrating the
  `ubuntu-latest` label to Ubuntu 26 and annotates every job in the meantime; a
  silent base-image swap is also the kind of change that breaks Playwright's
  browser dependencies. Move the label once CI is green on 26.
- The backend runtime image moved from `alpine:3.20` to `alpine:3.24`; 3.20 had
  stopped receiving packages in 2026, and the builder stage was already on 3.24.
- The release workflow has now been exercised end to end against a throwaway
  `v0.0.0-rc.1` tag: both images published for `linux/amd64` and `linux/arm64`
  with provenance and SBOM attestations, `latest` and `0.1` stayed on 0.1.0, and
  the GitHub Release came out as a pre-release. `docs/dev/release.md` records
  the rehearsal and one correction it turned up: a pre-release publishes only
  its exact version tag, never `0.2` or `latest`.
- The frontend now builds and tests on **Node.js 24** (LTS) instead of Node 20,
  which reached end-of-life in April 2026. CI runs 24, the image builder is
  `node:24-alpine`, `frontend/package.json` declares `engines.node >= 24`, and
  `.nvmrc` pins the same major. `@types/node` moved to 24 to match.

### Fixed

- Documentation drift in the published knowledge base, corrected against the
  code instead of re-translated. `backend-go.md` claimed `PORT` defaults to
  8080 (it is 8000) and listed `config.yaml` as a config source (the backend
  reads `.env` plus environment variables); `API.md` documented the login
  response as `{token, user}` (it is `access_token` / `token_type` /
  `expires_at`, with no user object) and the error body as `{message, code}`
  (it is `error_code` / `message` / `detail`); `project-layout.md` described
  the runtime image as distroless (it is Alpine) and listed three paths that do
  not exist. A truncated line in `kb/architecture/README.md` rendered as a bare
  `m_advanced_agents=1`.
- Inventory numbers were **removed** from the architecture pages rather than
  refreshed. File, module and route counts disagreed with each other across
  four documents and with the code; each page now points at the directory or
  `internal/router/router.go` as the source of truth, and `tech-stack.md` no
  longer transcribes patch versions.
- `PRD.en.md` was an abridged copy (266 lines against 601); it is now a
  complete translation of the v5.0 Chinese PRD, including the sections the
  abridgement had dropped.
- `data-model.md` listed the July 2026 core tables and left later modules out.
  It now names every table `AutoMigrate` creates (131 on 2026-10-25) and points
  at the model files for columns. `api-conventions.md` described `page` /
  `page_size`, a `{data, message}` envelope and a 10-second Axios timeout;
  the code uses `limit` / `offset`, several response shapes, and a 30-second
  timeout. The same pagination claim in `backend-go.md` is corrected.
- The design notes for saved views, pages, notifications, type hierarchy and
  relations still described the July drafts: nine notification types, English
  relation names, `copy-from-workspace`, and a child type that must be exactly
  one level deeper. They now follow the models, routes and the checks in
  `IssueService`.
- `PRD.md` still described a UUID schema, Redis, object storage, and SSO/SAML
  connectors that the process does not have. The core diagram now uses
  `uint64` keys, and the integration and auth sections list only GitHub, Git,
  Slack, outbound webhooks, JWT, and personal access tokens.

## [0.1.0] - 2026-10-24

First tagged release. Reqmango is a self-hosted project manager whose intake
queue triages new requests before they reach the backlog, with AI assisting
triage, issue analysis, label suggestions and cycle summaries.

### Added

- **Work items** — list and kanban views, state transitions, hierarchy up to six
  levels, assignees, labels, relations (blocks / relates / duplicates), comments
  and attachments.
- **Intake** — public intake forms and channels, a triage queue with
  accept/reject, and AI-suggested type, priority and duplicate candidates.
- **Custom fields and templates** — text, number, dropdown, boolean, date,
  member and URL fields; workspace type blueprints; project and issue templates.
- **Workflows and automation** — issue state machines with approvals and role
  gates; trigger → condition → action rules with a dry-run preview.
- **Planning** — cycles/sprints, modules, releases, pages with version history,
  and project metrics/dashboards.
- **Search** — RQL plus multi-field filters, saved views and search templates.
- **AI layer** — provider-agnostic (DeepSeek by default), optional; analyze,
  label suggestions, cycle summary, and a copilot that previews writes before
  applying them.
- **Bulk import** — CSV/JSON with pre-validation, preview (`dry_run`), explicit
  column mapping and per-row warnings.
- **Platform** — JWT auth, role-based access control, API tokens, notification
  centre with SSE, webhooks, audit-friendly tenancy scoping, MCP/CLI client in
  `sdk/`.
- **Operations** — single-command Docker Compose, `GET /health`, `GET /readyz`,
  `GET /metrics` (superuser), slow-request logging, graceful shutdown.
- **Docs** — English/Chinese README and contributing guide, architecture notes,
  API reference, and a production deployment checklist.

### Security

- **Demo and E2E seed data are now opt-in** (`SEED_DEMO`, `SEED_E2E`, both
  off by default). Every release before this one seeded a superuser whose
  password was published in the repository, so a reachable instance could be
  taken over with a documented credential.
- **Bootstrap admin** — a fresh deployment with demo data off creates one
  superuser from `ADMIN_EMAIL`/`ADMIN_PASSWORD`, or prints a random password
  once. An existing account is never promoted automatically.
- Added `SECURITY.md` (private reporting, scope, response targets) and
  `docs/deploy/production.md` (secrets, network exposure, TLS/SSE, backups).
- CI scans the working tree with gitleaks and replays every API route as an
  outsider, a member and a cross-tenant caller (`cmd/route_audit`).

### Fixed

- Workflow transitions are now scoped to their owning domain, so an agent
  workflow id can no longer address an issue state transition (BUG-58).
- The CI Docker job built from the repository root, where no Dockerfile exists;
  it now builds both component images and runs on every change instead of only
  on tags that never existed.

### Upgrade notes

- `SEED_DEMO` defaults to `false`. `docker compose up` with `.env.example`
  keeps the demo experience; any existing deployment should set it explicitly
  and **rotate or delete the old demo accounts** — turning the flag off does not
  remove rows that were already seeded.
- `SECRET_KEY` must be set explicitly, or every restart invalidates all
  sessions.
- Images: `ghcr.io/vinthuy/reqmango-backend` and
  `ghcr.io/vinthuy/reqmango-frontend`, multi-arch (`linux/amd64`,
  `linux/arm64`).

[Unreleased]: https://github.com/vinthuy/reqmango/compare/v0.1.0...master
[0.1.0]: https://github.com/vinthuy/reqmango/releases/tag/v0.1.0
