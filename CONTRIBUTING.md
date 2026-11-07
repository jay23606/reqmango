# Contributing to Reqmango

Thanks for helping. Reqmango is a **self-hosted project management** app: new requests are triaged before they hit the backlog; AI sits inside Issue / Intake / Cycle — not as a separate agent IDE.

[中文版](CONTRIBUTING-zh.md)

## Where to collaborate

| | URL | Role |
|---|---|---|
| **GitHub (primary)** | https://github.com/vinthuy/reqmango | Issues, PRs, Discussions, Releases |
| **GitCode (mirror)** | https://gitcode.com/yongfeng9m-/reqmanpy | Faster clone in China — not for day-to-day Issues/PRs |

One codebase, **one product sentence**. Open contributions on GitHub only. Maintainer sync: [docs/dev/dual-remote.md](docs/dev/dual-remote.md).

## Before you start

1. Read the product one-liner in [README.md](README.md).
2. Prefer a **small, focused PR** over a large rewrite.
3. Check open [GitHub issues](https://github.com/vinthuy/reqmango/issues) plus [good first issues](docs/dev/good-first-issues.md).
4. Optional DX check after i18n work: `cd frontend && npm run i18n:check`

## Quick setup

### Docker (recommended)

```bash
git clone https://github.com/vinthuy/reqmango.git
# China (clone only): git clone https://gitcode.com/yongfeng9m-/reqmanpy.git reqmango
cd reqmango
cp .env.example .env
docker compose up --build
```

Open http://localhost — `demo@example.com` / `demo1234`. Those accounts come from
`SEED_DEMO=true` in `.env.example`; they are public, so use them locally only
(see [production deployment](docs/deploy/production.md)).

Optional AI: set `AI_API_KEY` in `.env`, restart.

### Local (without Docker)

- Go **1.25+**, PostgreSQL **16+**, Node.js **24+**
- Backend: `cd backend && go run ./cmd/server/` (API on `:8000`)
- Frontend: `cd frontend && npm install && npm run dev` (http://localhost:5173)

The E2E suite signs in as the demo accounts, so a local backend needs
`SEED_DEMO=true` in `backend/.env` (see README). Without it the server creates a
single bootstrap admin instead and prints its password once.

See README for seed accounts and env details.

## Where to change what

| Area | Path |
|------|------|
| UI / i18n | `frontend/src/` — locales in `frontend/src/locales/{zh-CN,en-US}.json` |
| API / domain | `backend/internal/` |
| Compose / deploy | `docker-compose.yml`, `backend/Dockerfile`, `frontend/Dockerfile` |
| Docs | `docs/`, root `README.md` / `README-zh.md` |

Keep **zh-CN and en-US in sync** when you add UI strings. Prefer `t('namespace.key')` over hardcoded copy. Do not leave raw keys like `common.refresh` visible in the UI.

## Development workflow

1. Fork **GitHub** and branch from `master`.
2. Make the change; keep scope to the issue.
3. Verify locally:
   - Frontend: `cd frontend && npx vitest run` (and `npx vue-tsc --noEmit` if you touched types)
   - Backend: `cd backend && go test ./internal/...`
   - Manual: reproduce the bug or walk the happy path once
4. Open a **GitHub** PR using the template. Link the issue.

CI (`.github/workflows/ci.yml`) runs lint, Go tests, frontend tests, and e2e — PRs should stay green.

Run E2E against a disposable PostgreSQL database, not production: tests create accounts and mutate seed data. Intake fixtures delete their workspace, but registered accounts remain until the test database is discarded; CI discards its database after each job.

## What we welcome

- Bug fixes with a clear repro
- i18n / docs / DX (Docker, `.env.example`, error messages)
- Small UX improvements on Issue, Intake, Cycle, Pages
- Tests for existing behavior

## What to avoid (unless agreed in an issue)

- Large “Agent console / marketplace” expansions as the default product story
- Silent breaking API changes without migration notes
- Drive-by dependency upgrades with no functional need
- Committing secrets, `.env` with real keys, or large binaries

## Issue labels (maintainer)

| Label | Meaning |
|-------|---------|
| `good first issue` | Scoped, files named, acceptance clear |
| `help wanted` | Maintainer wants outside help |
| `bug` / `enhancement` | Type |
| `area:frontend` / `area:backend` / `area:docs` | Rough ownership |

## Code of conduct

Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
(Contributor Covenant 2.1). Report incidents to **vinthuy@qq.com** with
`[conduct]` in the subject.

## Support

Questions, setup problems and "is this supposed to work like this?" belong in
[SUPPORT.md](SUPPORT.md) / GitHub Discussions, not the issue tracker. Security
reports go through [SECURITY.md](SECURITY.md) only.

## Maintainers

- Dual-remote sync: [docs/dev/dual-remote.md](docs/dev/dual-remote.md)
- Release process: [docs/dev/release.md](docs/dev/release.md)
- GitHub About / Topics: [docs/dev/github-public-profile.md](docs/dev/github-public-profile.md)
- Publishing curated tasks: [good-first-issues.md](docs/dev/good-first-issues.md)

### Translation checks

Run `cd frontend && npm run i18n:check` before submitting UI changes. It fails when a literal translation key is missing in either locale; dynamically concatenated keys need separate validation.

### PostgreSQL numbering regression

Set `TEST_DATABASE_URL` to a PostgreSQL URL and run `cd backend && go test ./internal/service -run TestIssueSequence`. These tests create and clean up an isolated schema, so the database role needs schema creation permission. Without this variable they are skipped; CI supplies it.

## License

By contributing, you agree your contributions are licensed under the [MIT License](LICENSE).
