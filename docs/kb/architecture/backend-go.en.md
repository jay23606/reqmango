# Go Backend Architecture

**Last updated**: 2026-11-07

> The tree lists the **main path** and representative files only. Files come and go; when in
> doubt read the directory. What this page promises is layer responsibilities and extension
> points, not file counts.

---

## Project Structure

```
backend/
├── cmd/server/main.go              # Entry point
├── cmd/route_audit/                # Route audit tool (list / verify registered routes)
├── internal/
│   ├── config/config.go            # Config loading (Viper + env vars + .env)
│   ├── common/
│   │   ├── constants.go            # Role, priority, state group constants
│   │   ├── error_codes.go          # Error codes (with i18n messages)
│   │   ├── errors.go               # AppError type
│   │   └── pagination.go           # Pagination helpers
│   ├── model/                      # GORM models (see Data Model for the domains)
│   │   ├── base.go                 # BaseModel (ID/CreatedAt/UpdatedAt/DeletedAt)
│   │   ├── user.go                 # User (IsSuperuser, GetID, IsSuper)
│   │   ├── workspace.go            # Workspace, WorkspaceMember
│   │   ├── project.go              # Project, ProjectMember
│   │   ├── issue.go                # Issue, IssueAssignee, IssueLabel, IssueActivity
│   │   ├── issue_type.go           # IssueType + field binding
│   │   ├── state.go                # State, StateTransition
│   │   ├── cycle.go / module.go    # Cycles, modules
│   │   ├── page.go                 # Page (hierarchical docs) + templates + versions
│   │   ├── workflow.go             # Workflow + AutomationRule + AutomationExecution
│   │   ├── custom_field.go         # CustomField + 7 types
│   │   ├── role.go / permission.go # RBAC
│   │   └── …                       # Remaining models in internal/model/
│   ├── dto/
│   │   ├── request/                # Request DTOs with `binding` validation tags
│   │   └── response/               # Response DTOs, decoupled from Model
│   ├── service/                    # Business logic, one file per resource (*_service.go)
│   │   ├── auth_service.go         # JWT issuing and verification
│   │   ├── workspace_service.go    # Workspace CRUD + members
│   │   ├── project_service.go      # Project CRUD + members + stats + archive
│   │   ├── issue_service.go        # Issue CRUD + search + batch + import/export + relations
│   │   ├── cycle_service.go        # Cycle CRUD + transitions + progress + burndown
│   │   ├── module_service.go       # Module CRUD + tree + stats
│   │   ├── page_service.go         # Page CRUD + tree + archive
│   │   ├── workflow_service.go     # Workflow CRUD + transition validation
│   │   ├── workflow_executor.go    # Workflow execution engine
│   │   ├── notification_service.go # Notifications + read state + reminders
│   │   ├── webhook_service.go      # Webhook delivery + HMAC-SHA256 signing
│   │   ├── automation_service.go   # Automation rule CRUD + trigger execution
│   │   ├── role_service.go         # RBAC roles + permission queries
│   │   ├── sse_hub.go              # SSE real-time event hub
│   │   └── …                       # Remaining services in internal/service/
│   ├── handler/                    # HTTP handlers, one file per resource
│   │   ├── auth_handler.go / workspace_handler.go / project_handler.go
│   │   ├── issue_handler.go / cycle_handler.go / module_handler.go
│   │   ├── workflow_handler.go / notification_handler.go / webhook_handler.go
│   │   ├── sse_handler.go         # SSE endpoint
│   │   └── …                       # Remaining handlers in internal/handler/
│   ├── router/router.go            # The single place all routes are registered
│   ├── middleware/                 # Middleware (10 files)
│   │   ├── auth.go                # JWT: parse Bearer token → inject currentUser
│   │   ├── authorization.go       # RBAC: RequirePermission / RequireRoleLevel
│   │   ├── access.go              # Workspace member admission: RequireWorkspaceMember
│   │   ├── scope.go               # Tenancy: every authenticated route verifies ownership
│   │   ├── metrics.go             # Latency histogram + slow-request logging
│   │   ├── recovery.go            # Panic recovery, unified JSON error
│   │   ├── cors.go                # CORS policy
│   │   ├── language.go            # i18n language detection (Header/Query)
│   │   ├── logger.go              # Request logging
│   │   └── rate_limiter.go        # Token bucket (default 500 req/min, 0=disabled)
│   ├── ai/                         # AI subsystem
│   │   ├── llm/client.go          # LLM client (DeepSeek / Anthropic)
│   │   ├── service/               # AI chat, triage, analysis, agent orchestration
│   │   ├── harness/               # Tool/skill execution, pipeline, reviewer
│   │   ├── loop/                  # Loop runtime (not productized)
│   │   └── handler/               # AI / Agent / Loop / Pipeline endpoints
│   ├── rql/                        # RQL query language engine
│   │   ├── lexer.go / parser.go / ast.go   # Lexer → parser → AST
│   │   ├── builder.go / executor.go        # AST → SQL/GORM, execution
│   │   ├── handler.go / service.go         # HTTP endpoint and query service
│   │   └── *_test.go
│   ├── workflow/rules.go           # State machine rules
│   ├── plugin/                     # Plugin registry + hooks
│   ├── security/sanitize.go        # HTML sanitizing (bluemonday)
│   ├── scheduler/                  # Background jobs (memory cleanup, etc.)
│   ├── version/version.go          # Build info behind GET /version
│   ├── i18n/
│   │   ├── i18n.go                 # Internationalization logic
│   │   ├── messages_en.json        # English error messages
│   │   └── messages_zh.json        # Chinese error messages
│   ├── client/agent_client.go      # Agent invocation client
│   ├── issuetype/visible.go        # Issue type visibility rules
│   ├── randutil/randutil.go        # Random strings (passwords, keys)
│   ├── testutil/                   # Test fixtures (DB / HTTP / fixtures)
│   └── seed/
│       ├── seed.go                 # Demo seed (users + issues + workspace/project)
│       └── seed_rbac.go            # RBAC seed (55 permissions + 3 default roles)
```

---

## Layered Architecture

```
Router ──→ Middleware Chain ──→ Handler ──→ Service ──→ Model (GORM)
    │           │                    │            │
    │     Auth/RBAC/Scope            │            └── DB operations
    │     CORS/Lang/Log/RateLimit    └── DTO binding + HTTP response
    └── Gin route tree (hundreds of endpoints, all registered in internal/router/router.go)
```

### Handler Layer
- Binds request parameters through the Gin context (`c.ShouldBindJSON`, `c.Param`, `c.Query`)
- Calls Service methods
- Returns the unified JSON response shape
- **Contains no business logic**

### Service Layer
- Pure Go business logic
- Uses GORM for database access
- Returns the custom `AppError` type (HTTP status + i18n error code)
- Performs cross-model operations (e.g. Issue with assignees, labels, attachments)
- Receives dependencies (Notification, Webhook, Automation, Slack services)

### Model Layer
- GORM tags define tables and indexes
- All models embed `BaseModel` (ID uint64, CreatedAt, UpdatedAt, DeletedAt)
- `DeletedAt gorm.DeletedAt` implements soft delete
- Associations use GORM tags + Preload

### DTO Layer
- `request/` — client payloads with `binding` validation tags
- `response/` — server payloads
- Kept separate from Model so internal columns never leak

### RQL Layer
- Purpose-built RQL (reqmango Query Language)
- Full pipeline: lexer → parser → AST → SQL/GORM
- Endpoint: `POST /api/v1/rql/search`
- Feature parity with mainstream query languages, free in the community edition

---

## Core Components

### AppError (unified error handling)

```go
type AppError struct {
    StatusCode int
    ErrorCode  string  // RQL error code (i18n aware)
    Message    string
    Err        error
}
```

Error factories: `NotFound()`, `Conflict()`, `Unauthorized()`, `Forbidden()`, `Validation()`, `Internal()`, `BadRequest()`

### i18n error messages
- `messages_en.json` / `messages_zh.json`
- `ErrorCode` maps to a concrete message
- Language comes from the `Accept-Language` header or the `lang` query parameter

### Pagination

```go
func ParsePagination(limitStr, offsetStr string, defaultLimit, maxLimit int) Pagination
```

Callers pass the `limit` and `offset` query parameters. The work-item list defaults to 50 and caps at 100; other handlers pick their own defaults. The total is the `X-Total-Count` response header.

### Authentication flow

1. `POST /api/v1/auth/login` — bcrypt verification, returns `access_token` + `token_type` + `expires_at`
2. Subsequent requests carry the JWT in `Authorization: Bearer <token>`
3. The auth middleware parses the JWT, loads the user, and injects `c.Set("currentUser", user)`
4. Handlers read it via `c.MustGet("currentUser").(*model.User)`

### RBAC and tenancy

- `authorization.go` provides `RequirePermission(db, permCode)` and `RequireRoleLevel(db, minLevel)`
- `access.go` provides `RequireWorkspaceMember`, rejecting non-members
- `scope.go` runs on every authenticated route and checks that the workspace/project in the
  request actually owns the resource, so a handler that forgets to check cannot read or write
  across tenants
- Superusers (IsSuperuser=true) bypass every permission check
- Permission format: `resource:action` (e.g. `issue:create`, `workspace:manage`)
- Role levels: Admin=20 (all), Member=15 (create/edit), Guest=5 (read-only)
- Workspace-level and project-level custom roles are supported
- 5 RBAC APIs: `GET/POST /api/v1/workspaces/:wsParam/roles`, `PUT/DELETE /api/v1/workspaces/:wsParam/roles/:id`, `GET /api/v1/permissions`

### Rate limiting

- Token bucket, keyed by client IP
- Default 500 req/min (`RATE_LIMIT_REQUESTS`)
- `limit=0` disables it entirely (dev/CI)
- Response headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

### SSE real-time notifications

- `sse_hub.go` — broadcast hub
- `sse_handler.go` — `/api/v1/sse`
- Notifications are pushed to the frontend over SSE

---

## API Route Overview

Every route is under `/api/v1`. The table gives the resource scope per group; for exact paths
read `internal/router/router.go`, or run `go run ./cmd/route_audit` for a machine-readable list:

| Route group | Resources |
|-------------|-----------|
| `/auth/` | Register / login / current user |
| `/workspaces/` | CRUD + members + AI agent + MCP + GitHub + Slack config |
| `/projects/` | CRUD + members + statistics + reports + webhook + automation |
| `/projects/:id/cycles` | Cycle CRUD + transitions + progress + burndown |
| `/projects/:id/modules` | Module CRUD + tree + Issue relations + statistics |
| `/projects/:id/pages` | Page CRUD + archive |
| `/projects/:id/issue-types` | Issue type CRUD + field binding |
| `/projects/:id/workflows` | Workflow CRUD + rules |
| `/projects/:id/releases` | Release + roadmap |
| `/projects/:id/ai/` | Chat/search/charts/triage; `analyze` (issue or project); `sprint-plan`; `automation-preview` (NL draft, not persisted) |
| `/projects/:id/automations` | Automation rule CRUD + execution (incl. `dispatch_agent`) |
| `/projects/:id/dashboards` | Dashboard CRUD + widgets; `GET .../full` returns widget_data (incl. `ai_summary`) |
| `/projects/:id/estimate-points` | Estimate catalogs / settings; writes need `settings:manage` |
| `/issues/` | CRUD (incl. `estimate_point_id`) + search + batch + import/export (CSV `estimate`→point) + tree + relations + soft-merge + recurrence |
| `/issues/:id/time-tracks` | Manual create / start / stop / list / summary / delete; `time_track:*`; `duration` seconds |
| `/comments/` | CRUD + replies + resolve |
| `/custom-fields/` | CRUD + options + conditional rules |
| `/templates/` | Project templates + type templates + work item templates |
| `/rql/search` | RQL execution |
| `/notifications/` | List + read + reminders |
| `/webhook/github/:id` | GitHub webhook receiver (public) |
| `/intake/:projectId` | Public issue submission (no auth) |
| `/sse` | Real-time event stream |
| `/workspaces/:wsParam/roles` | RBAC role CRUD |
| `/permissions` | Global permission enumeration |

---

## Configuration

Configuration comes from environment variables plus a `.env` file (Viper searches `.`, `..`,
`../..`). The full list lives in [Configuration](../../configuration.md); the architecturally
relevant ones are:

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL DSN | `postgres://postgres:postgres@localhost:5432/reqmango?sslmode=disable` |
| `SECRET_KEY` | JWT signing secret | random key + warning when unset |
| `PORT` | Listen port | `8000` |
| `RATE_LIMIT_REQUESTS` | Rate limit (req/min) | 500 |
| `RATE_LIMIT_WINDOW_SEC` | Rate limit window | 60 |
| `AI_API_KEY` | AI key (also reads `DEEPSEEK_API_KEY`) | empty |
| `AI_PROVIDER` | AI provider | deepseek |
| `AI_MODEL` | AI model | deepseek-chat |
| `AI_BASE_URL` | AI API endpoint | `https://api.deepseek.com/v1` |
| `DEBUG` | Gin debug mode | true |
| `SLOW_REQUEST_MS` | Slow request threshold (ms, 0 disables) | 1000 |

---

## 🌐 Language

- **English** (this document)
- [中文文档](https://vinthuy.github.io/reqmango/kb/architecture/backend-go/)
