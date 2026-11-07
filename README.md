# Reqmango

[![CI](https://github.com/vinthuy/reqmango/actions/workflows/ci.yml/badge.svg)](https://github.com/vinthuy/reqmango/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/vinthuy/reqmango?sort=semver)](https://github.com/vinthuy/reqmango/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Container images](https://img.shields.io/badge/ghcr.io-reqmango-2496ED?logo=docker&logoColor=white)](https://github.com/vinthuy/reqmango/pkgs/container/reqmango-backend)

**Self-hosted project management where new requests are triaged first.**  
Submit → type / priority / duplicate hints → you only decide the uncertain ones. AI stays inside Issue / Intake / Cycle; your data stays on your machine.

[中文文档](README-zh.md) · [Docs](https://vinthuy.github.io/reqmango/) · [Contributing](CONTRIBUTING.md) · [Issues](https://github.com/vinthuy/reqmango/issues) · [GitCode mirror](https://gitcode.com/yongfeng9m-/reqmanpy)

![Triage queue](docs/assets/screenshot-02-triage-queue.png)

<sub>Intake form → triage queue: accept / reject before anything hits the backlog.</sub>

---

## Try it (≈2 minutes)

```bash
git clone https://github.com/vinthuy/reqmango.git && cd reqmango
# China clone mirror: git clone https://gitcode.com/yongfeng9m-/reqmanpy.git reqmango && cd reqmango
cp .env.example .env && docker compose pull && docker compose up -d
```

Wait until both containers are healthy, then open **http://localhost** —

| | |
|---|---|
| Email | `demo@example.com` |
| Password | `demo1234` |

Optional AI (Intake triage / analyze / labels): set `AI_API_KEY` in `.env`, then `docker compose up -d` again.

Cold `docker compose up --build` also works (several minutes). `curl localhost/version` shows which build is running.

<table>
  <tr>
    <td width="50%"><img src="docs/assets/screenshot-01-intake-form.png" alt="Intake form"><br><sub><b>Intake form</b> — one public link collects requests from people who have no account</sub></td>
    <td width="50%"><img src="docs/assets/screenshot-03-issues.png" alt="Issue list"><br><sub><b>Issues</b> — list, kanban, tree, calendar and gantt over the same filters</sub></td>
  </tr>
  <tr>
    <td><img src="docs/assets/screenshot-04-issue-detail.png" alt="Issue detail"><br><sub><b>Issue detail</b> — properties, relations, activity and the AI panel in one place</sub></td>
    <td><img src="docs/assets/screenshot-05-cycles.png" alt="Cycles"><br><sub><b>Cycles</b> — sprints with scope, progress and what slipped</sub></td>
  </tr>
  <tr>
    <td><img src="docs/assets/screenshot-06-analytics.png" alt="Analytics"><br><sub><b>Analytics</b> — priority mix, workload and completion over time</sub></td>
    <td><img src="docs/assets/screenshot-07-workspace.png" alt="Workspace"><br><sub><b>Workspace</b> — several projects, one team, one login</sub></td>
  </tr>
</table>

The UI follows the browser language on first visit (Chinese or English) and can be switched from any page.

### 60-second walkthrough

1. Sign in with the demo account.
2. Open the **Demo** project → **Project settings** → **Intake triage**.
3. Open the Intake form link, submit a short request (e.g. “login sometimes fails on mobile”).
4. Return to the triage queue: accept / reject, and (with `AI_API_KEY`) run AI analyze / label suggestions on issues.

---

## What you get

| Area | Capability |
|------|------------|
| **AI in the work loop** | Intake triage, issue analyze, label suggestions, Cycle / sprint summary, Ask / Build Copilot with preview before write |
| Work items | List + Kanban, state transitions, hierarchy (up to 6 levels) |
| Custom fields & templates | Text / number / dropdown / boolean / date / member / URL; workspace type blueprints; project templates |
| Workflow & automation | Transitions, approvals, role gates; trigger → condition → action |
| Relations & search | Blocks / Relates / Duplicates; RQL + multi-field filters |
| Notifications & security | Unread counts; XSS sanitization (bluemonday); JWT + RBAC |
| API | 100+ REST endpoints |

AI is a layer on top of project management — not a separate “agent IDE” as the default story. Advanced Agent console routes may still exist in the codebase; day-to-day PM paths are Issue, Cycle, Intake, and Pages.

---

## Requirements

| Mode | Need |
|------|------|
| **Docker (recommended)** | Docker + Docker Compose |
| Local dev | Go **1.25+**, PostgreSQL **16+**, Node.js **24+** |

Stack matches CI and `docker-compose.yml` (`postgres:16`, Go 1.25 and Node 24
build images).

---

## Local development (without Docker)

```bash
# Database
psql -U postgres -c "CREATE DATABASE reqmango;"

# Backend
cd backend
cat > .env << EOF
DATABASE_URL=postgres://postgres:postgres@localhost:5432/reqmango?sslmode=disable
SECRET_KEY=$(openssl rand -hex 32)
ACCESS_TOKEN_EXPIRE_MINUTES=10080
PORT=8000
DEBUG=true
SEED_DEMO=true
EOF
go run ./cmd/server/
```

The server migrates and seeds on start, so this is all a new environment needs. `SEED_DEMO=true` creates the demo accounts; drop that line and the first boot instead creates one admin and prints a random password to the log (see [production deployment](docs/deploy/production.md)).

### Database migrations

`cmd/migrate` is the explicit alternative for environments that want a reviewed
schema step separate from the application boot:

```bash
cd backend
go run ./cmd/migrate up      # apply the baseline
go run ./cmd/migrate version # 1, dirty=false
```

`migrations/000001_baseline.up.sql` is a single idempotent baseline, so it runs
on an empty database *and* on one already created by the server's AutoMigrate.
An existing database that predates the squash needs a one-time
`go run ./cmd/migrate force 1`; see
[the debt note](docs/dev/debt/2026-10-18-sql-migration-chain.md).

Seed data includes `demo@example.com` / `demo1234`, workspace `reqmango-dev` (projects `CORE` / `MOBILE` / `AI` / `OPENAPI`), sample sprints / modules / issues. Those demo passwords are public, so the demo is for local evaluation only.

```bash
# Frontend (API on :8000)
cd frontend
npm install
npm run dev
```

Open **http://localhost:5173**.

### Production build (local)

```bash
cd backend && go build -o server ./cmd/server/
cd frontend && npm run build
```

---

## Project layout

```
reqmango/
├── backend/     # Go + Gin + GORM API
├── frontend/    # Vue 3 + TypeScript + Vite
├── sdk/         # MCP / CLI (optional; not the default product story)
├── docs/        # API, architecture, product specs
└── docker-compose.yml
```

---

## Docs

The same content is published as a site — [vinthuy.github.io/reqmango](https://vinthuy.github.io/reqmango/) ([English](https://vinthuy.github.io/reqmango/en/)) — built from `docs/` by MkDocs.

- [API Reference](docs/API.md)
- [Architecture](docs/kb/architecture/) — [tech stack](docs/kb/architecture/tech-stack.md), [Go backend](docs/kb/architecture/backend-go.md), [frontend](docs/kb/architecture/frontend.md), [data model](docs/kb/architecture/data-model.md)
- [Production deployment](docs/deploy/production.md) — secrets, demo data off, TLS, backups
- [Configuration](docs/configuration.md) — every environment variable and its default

---

## Contributing

We welcome Issues and Pull Requests — especially small, focused changes.

- [Contributing guide](CONTRIBUTING.md) · [中文](CONTRIBUTING-zh.md)
- [Roadmap](ROADMAP.md) — what we're building next, and what we're not
- [Support](SUPPORT.md) — questions, bug reports, and where to ask what
- [Code of Conduct](CODE_OF_CONDUCT.md) — Contributor Covenant 2.1
- [Good first issues](docs/dev/good-first-issues.md) (curated starter tasks)
- [Security policy](SECURITY.md) — report vulnerabilities privately, never in a public issue
- **GitHub** is the collaboration home · [GitCode](https://gitcode.com/yongfeng9m-/reqmanpy) is a clone mirror ([dual-remote](docs/dev/dual-remote.md))

## License

[MIT](LICENSE)
