# Architecture Overview

reqmango uses a decoupled frontend-backend architecture: Go + Vue 3, full stack.

**Last updated**: 2026-11-07

> This page describes **layers and responsibilities**, and deliberately does not maintain
> inventory numbers (file counts, route counts) that change with every commit. For the
> concrete list, read the directory — or `backend/internal/router/router.go`, the single
> place where all routes are registered.

---

## Tech Stack Overview

| Layer | Technology | Status |
|-------|------------|--------|
| Backend framework | Go 1.25 + Gin | ✅ Primary |
| ORM | GORM | ✅ |
| Database | PostgreSQL 16+ | ✅ |
| Authentication | JWT (golang-jwt/v5) | ✅ |
| LLM integration | DeepSeek / Anthropic / OpenAI-compatible | ✅ |
| Frontend framework | Vue 3 + Composition API | ✅ |
| Frontend build | Vite 5 | ✅ |
| State management | Pinia | ✅ |
| CSS | Tailwind CSS | ✅ |
| Type system | TypeScript | ✅ |
| E2E tests | Playwright | ✅ |

Versions come from `backend/go.mod` and `frontend/package.json`; see [Tech Stack](tech-stack.md).

---

## System Layers

```
┌──────────────────────────────────────────┐
│              Frontend (Vue 3)             │
│  Views → Components → Stores → API →     │
│         TypeScript Types                 │
└──────────────────┬───────────────────────┘
                   │ HTTP (JSON + JWT) + SSE (AI streaming)
┌──────────────────▼───────────────────────┐
│            Go Backend (Gin)              │
│                                           │
│  Router → Middleware → Handler           │
│                          │                │
│                        Service            │
│                          │                │
│                    Model (GORM)           │
│                          │                │
│                      PostgreSQL           │
└──────────────────────────────────────────┘
```

## Backend Layer Responsibilities

| Layer | Directory | Responsibility |
|-------|-----------|----------------|
| Handler | `internal/handler/` | Binds HTTP requests, calls Service, writes responses. No business logic |
| Service | `internal/service/` | Pure business logic, cross-model operations, returns `AppError` |
| Model | `internal/model/` | GORM model definitions, table mapping |
| DTO | `internal/dto/` | Request/response structs, decoupled from Model |
| Middleware | `internal/middleware/` | JWT auth, RBAC, tenant scoping, CORS, logging, rate limiting |
| Router | `internal/router/` | Route registration; mounts middleware and handlers |
| Common | `internal/common/` | Error types, constants, pagination helpers |
| RQL | `internal/rql/` | ReqMan Query Language (lexer/parser/executor) |
| AI | `internal/ai/` | LLM client, harness, triage and chat orchestration |
| Workflow | `internal/workflow/` | Workflow execution engine |

## Frontend Layer Responsibilities

| Layer | Directory | Responsibility |
|-------|-----------|----------------|
| Views | `src/views/` | Page-level components mapped to routes (includes the hidden Agent consoles under `views/agent/`, `views/agents/`) |
| Components | `src/components/` | Reusable components |
| Stores | `src/stores/` | Pinia state (auth / cycle / module / issue / project / notification / agentLoop / agentPipeline) |
| API | `src/api/` | Axios wrappers, one module per backend resource |
| Types | `src/types/` | TypeScript definitions (including `filters.ts`) |
| Composables | `src/composables/` | Composables (useAI / useRQL / useFilters / usePermission / useSSE / useI18n …) |
| Router | `src/router/` | Vue Router config (with the `minRoleLevel` guard) |

---

## Implemented Modules

| Module | Go Backend | Frontend | Description |
|--------|------------|----------|-------------|
| Auth | ✅ | ✅ | JWT register/login/me |
| Workspace | ✅ | ✅ | CRUD + member management |
| Project | ✅ | ✅ | CRUD + archive + statistics + Lead + Subscriber |
| Issue | ✅ | ✅ | CRUD + hierarchy/archive/batch/import-export/Issue-Page association |
| State | ✅ | ✅ | 5 fixed groups, project-level CRUD |
| Label | ✅ | ✅ | Project-level CRUD |
| Cycle | ✅ | ✅ | CRUD + start/end/cancel + burndown chart + progress |
| Module | ✅ | ✅ | Tree CRUD + Issue association + progress statistics |
| CustomField | ✅ | ✅ | 7 types + option management + Issue value binding |
| IssueType | ✅ | ✅ | Workspace-level + project-level, hierarchy binding |
| TypeTemplate | ✅ | ✅ | Workspace-level type blueprint |
| ProjectTemplate | ✅ | ✅ | Project template + Apply |
| WorkItemTemplate | ✅ | ✅ | Work item creation template |
| Workflow | ✅ | ✅ | State transitions + approval rules |
| Automation | ✅ | ✅ | Trigger-Condition-Action (dot events); AI templates + `dispatch_agent`; NL `automation-preview` draft then confirm |
| Comment | ✅ | ✅ | Nested replies + resolve |
| Notification | ✅ | ✅ | CRUD + read/unread + summary statistics |
| Saved Views | ✅ | ✅ | JSONB filters/sort/column config + default view |
| Pages | ✅ | ✅ | Tree hierarchy + archive/restore + Markdown |
| Release | ✅ | ✅ | CRUD + Issue association + progress |
| Estimate | ✅ | ✅ | Project catalogs Points/Categories/Time; **issues store `estimate_point_id` (points)**; settings `EstimatePointManager`; mutate needs `settings:manage`; categories/time on issues still Later |
| Attachment | ✅ | ✅ | File upload + metadata |
| Relation | ✅ | ✅ | Custom in/out naming + Issue association |
| RQL | ✅ | ✅ | Custom DSL search |
| AI | ✅ | ✅ | Chat SSE + NL Search + Smart Create + Analyze (issue/project) + Page AI + Triage; project summary→Page; Dashboard `ai_summary` |
| Dashboard | ✅ | ✅ | Widget grid incl. charts/number/burndown/report/list + `ai_summary` (load on open, refresh; no scheduled reports) |
| Time Tracking | ✅ | ✅ | Start/Stop/manual log; `duration` in **seconds**; List/Summary; `time_track:view|create|delete`; sidebar shows spent totals |
| Merge Duplicates | ✅ | ✅ | `POST /issues/merge` soft-merge (keeps source); detail 「Merge into…」 |
| Recurring | ✅ | ✅ | daily/weekly/monthly/cron + UI config |
| Intake & Triage | ✅ | ✅ | Public submission + Accept/Reject + AI analysis |
| Conditional Fields | ✅ | ✅ | Field visibility rules |
| Cover Image | ✅ | ✅ | Issue cover |
| Command Palette (⌘K) | ✅ | ✅ | Keyboard quick navigation |
| AI Settings | ✅ | ✅ | Provider/Model/APIKey UI configuration |
| RBAC | ✅ | ✅ | 55 permissions + 3 default roles + custom role management |
| Quick Create | ✅ | ✅ | Inline quick issue creation |
| Git Integration | ✅ | ✅ | GitHub/GitLab native integration + Webhook |
| Project CustomField Enrollment | ✅ | ✅ | Project-level custom field enable/disable |
| Workspace Workflow | ✅ | ✅ | State machine CRUD + transitions (CORE-04 fixed); projects can override |
| Calendar/Gantt View | ✅ | ✅ | Calendar (IssueCalendar.vue) + Gantt (IssueGantt.vue) |
| AI Agent system | ✅ | 🔒 | Out-of-box PM agents + assign/@agent; console hidden by default (Stage B); Harness/Loop **not productized** |
| Chat system | ✅ | ✅ | Chat, Messages, SSE streaming |
| Page versioning | ✅ | ✅ | PageVersion, PageTemplate |
| Approvals | ✅ | ✅ | Approval |
| Workflow visualisation | ✅ | ✅ | Workflow Nodes/Edges/Execute/Runs |

---

## In Progress

| Module | Description |
|--------|-------------|
| Existing quality | Main product path and the core-customization gate PASS; focus stays on stability and data quality, no new large AI features |

## Product Direction Notes

- **Current**: native PM plus a native AI project-management path (Intake / Analyze / automation preview / `ai_summary`)
- **Core gate**: issue types / custom fields / workflows (including workspace state machines) / automation — acceptance §H
- **Retired**: Harness / Loop / productizing a standalone Agent platform console → [superseded/agent-platform](https://github.com/vinthuy/reqmango/blob/master/docs/superseded/agent-platform/README.md)
- **Stage B**: `/agents*` redirects by default; unlock with `localStorage.rm_advanced_agents=1`

## Future Extensions

(None scheduled; a standalone Agent platform is not on the roadmap.)

---

## 🌐 Language

- **English** (this document)
- [中文文档](https://vinthuy.github.io/reqmango/kb/architecture/)
