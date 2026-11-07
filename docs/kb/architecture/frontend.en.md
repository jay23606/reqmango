# Frontend Architecture

**Last updated**: 2026-11-07

> The tree documents **structure and conventions**. File counts grow fast; when in doubt read
> `frontend/src/`.

---

## Tech stack

| Component | Choice |
|-----------|--------|
| Framework | Vue 3 (Composition API + `<script setup>`) |
| Build | Vite 5 |
| Language | TypeScript |
| State | Pinia |
| Routing | Vue Router 4 |
| CSS | Tailwind CSS |
| HTTP | Axios |
| Rich text | TipTap |
| SSE | fetch + ReadableStream (AI streaming) |

The runtime (Node 24) is documented in [Tech Stack](tech-stack.md).

## Project structure

```
frontend/src/
├── main.ts
├── App.vue
├── style.css
│
├── api/                  — Axios wrappers, one module per backend resource
│   ├── index.ts          — Axios instance + JWT interceptor
│   ├── auth.ts / workspace.ts / project.ts
│   ├── issue.ts / cycle.ts / module.ts / page.ts
│   ├── time-track.ts / estimate-point.ts
│   ├── ai.ts             — AI chat SSE / search / create
│   ├── workflow.ts / automation.ts / issue-type.ts
│   ├── custom-field.ts / conditional-field.ts / saved-view.ts
│   ├── dashboard.ts / report.ts / metric.ts
│   ├── agent.ts / plugin.ts / mcp.ts / role.ts
│   └── …                 — see api/ for the rest
│
├── types/                — TypeScript types, split per module
│   ├── index.ts
│   ├── filters.ts        — filter types + FILTER_FIELDS + buildRQL/parseRQL
│   ├── issue.ts / project.ts / cycle.ts / module.ts
│   └── …                 — see types/ for the rest
│
├── stores/               — Pinia
│   ├── auth.ts           — auth state
│   ├── cycle.ts          — cycle CRUD + progress
│   ├── module.ts         — module CRUD + tree
│   ├── issue.ts / project.ts
│   ├── notification.ts
│   └── agentLoop.ts / agentPipeline.ts
│
├── router/index.ts       — Vue Router config (with the minRoleLevel guard)
│
├── views/                — page-level components
│   ├── Login.vue, Register.vue, Home.vue
│   ├── Workspace.vue, WorkspaceSettings.vue, WorkspaceOverview.vue
│   ├── WorkspaceAnalytics.vue, Analytics.vue
│   ├── Project.vue, ProjectPages.vue, ProjectSettings.vue, ProjectIntake.vue
│   ├── IssueCreate.vue, IssueDetail.vue
│   ├── CycleCreate.vue, CycleDetail.vue
│   ├── CustomFields.vue, IssueTypeList.vue
│   ├── IntakeForm.vue, Roadmap.vue, Initiatives.vue
│   ├── Dashboard.vue, PluginManager.vue
│   ├── ApprovalList.vue, MetricsView.vue
│   ├── agent/            — Agent console (hidden by default in Stage B)
│   └── agents/           — Agent templates/sessions/runtimes/memory (same)
│
├── components/           — reusable components, grouped by domain
│   ├── Agent / AI        — AgentAuditLog, AgentSelector, AIChatSidebar, AICopilot,
│   │                       AIChartRenderer, AIResultActions …
│   ├── Dashboard         — DashboardGrid, WidgetCard (incl. ai_summary), WidgetConfigPanel
│   ├── Issue             — IssueCard, IssueList, IssueKanban, IssueDetailPanel,
│   │                       IssuePropertySidebar (estimate picker / spent),
│   │                       SubIssuePanel, IssueTreeView, IssueGantt, IssueCalendar,
│   │                       TimeTrackPanel (timer + manual minutes), MergeIntoIssueDialog
│   ├── Cycle / Module    — CycleCard, CycleBurndownChart, ModuleTree, ModuleDetailPanel …
│   ├── Config            — CustomField*, ConditionalField*, EstimatePointManager (project settings)
│   ├── Workflow          — WorkflowManager, WorkflowRule*, StateTransition,
│   │                       WorkflowVisualization
│   ├── Automation        — AutomationManager, AutomationRuleBuilder (incl. dispatch_agent,
│   │                       NL draft prefill), AutomationTemplateList
│   ├── Template          — ProjectTemplateManager, WorkItemTemplateManager,
│   │                       WorkspaceIssueTypeManager
│   ├── Page              — PageTree, PageVersionPanel, PageVersionDiff, PageTabConfig
│   ├── Filter / View     — FilterBar (unified filter bar), SavedViewSelector, QuickFilterChips
│   ├── Import / Triage   — ImportIssuesModal, QuickCreateInput, TriagePanel
│   ├── Common            — CommentList, AttachmentManager, NotificationCenter,
│   │                       RichTextEditor, UserSelect, LabelSelector, ConfirmDialog …
│   └── RQL               — RQLInput, RQLHistory
│
└── composables/          — composables
    ├── useAPI / useSSE / useChatSSE  — requests and streaming
    ├── useAI               — AI SSE consumption
    ├── useFilters          — filter state (Provide/Inject, two-way sync with RQL)
    ├── useIssueFilters     — issue filtering
    ├── useRQL              — RQL query history
    ├── usePermission       — RBAC permission checks
    ├── useAutoTriageWatch  — triage state watcher
    ├── useProductFlags     — feature flags (incl. Agent console unlock)
    ├── useWorkspaceId      — current workspace context
    ├── useRelationHelpers  — relation type helpers
    ├── useDashboard / useReportChart / useMarkdown
    ├── useI18n / useDarkMode / useToast / useConfirm
    └── …
```

## Routes

Routes are defined in `frontend/src/router/index.ts`. The table lists main paths only
(detail sub-routes omitted):

| Path | View | Notes |
|------|------|-------|
| `/` | Home | Workspace list |
| `/login` | Login | Sign in |
| `/register` | Register | Sign up |
| `/workspace/:slug` | Workspace | Workspace detail + project list |
| `/workspace/:slug/settings` | WorkspaceSettings | Workspace settings |
| `/workspace/:slug/overview` | WorkspaceOverview | Workspace overview |
| `/workspace/:slug/roadmap` | Roadmap | Roadmap |
| `/workspace/:slug/initiatives` | Initiatives | Strategic goals |
| `/workspace/:slug/analytics` | WorkspaceAnalytics | Workspace analytics |
| `/workspace/:slug/project/:id` | Project | Project home |
| `/workspace/:slug/project/:id/pages` | ProjectPages | Page docs |
| `/workspace/:slug/project/:id/settings` | ProjectSettings | Project settings |
| `.../settings/workflows/:workflowId` | WorkflowDetail | Workflow detail |
| `/workspace/:slug/project/:id/analytics` | Analytics | Project analytics |
| `/workspace/:slug/project/:id/dashboards` | Dashboard | Dashboards (supports an `ai_summary` widget) |
| `/workspaces/:wid/projects/:pid/issues/:iid` | IssueDetail | Issue detail |
| `/workspaces/:wid/projects/:pid/issues/new` | IssueCreate | Create issue |
| `/workspaces/:wid/projects/:pid/custom-fields` | CustomFields | Custom fields |
| `/workspaces/:wid/projects/:pid/issue-types` | IssueTypeList | Issue types |
| `/workspaces/:wid/projects/:pid/cycles/new` | CycleCreate | Create cycle |
| `/workspaces/:wid/projects/:pid/cycles/:cid` | CycleDetail | Cycle detail |
| `/workspace/:slug/project/:id/issues/:iid` | IssueDetail | Issue detail (slug style) |
| `/workspace/:slug/project/:id/issues/new` | IssueCreate | Create issue (slug style) |
| `/workspace/:slug/project/:id/cycles/new` | CycleCreate | Create cycle (slug style) |
| `/workspace/:slug/project/:id/cycles/:cid` | CycleDetail | Cycle detail (slug style) |
| `/intake/:projectId` | IntakeForm | Public submission |

Two path styles (`/workspace/:slug/...` and `/workspaces/:wid/...`) coexist for historical
reasons. New pages should follow the slug style so the split does not spread.

## Architecture patterns

### API layer

```typescript
// api/index.ts — Axios instance
const api = axios.create({ baseURL: '/api/v1' })
// Request interceptor: attaches the JWT bearer token
// Response interceptor: 401 → redirect to login
```

### AI SSE consumption

```typescript
// composables/useAI.ts / useChatSSE.ts
// fetch + ReadableStream consuming SSE
// Event types: text | tool_call | tool_result | done | error
```

### Component conventions

- Every component uses `<script setup lang="ts">`
- Props: `defineProps<T>()`
- Emits: `defineEmits<T>()`
- State: local `ref` at page level, Pinia store across components

---

## Internationalization (i18n)

The frontend ships two locales:

- `locales/zh-CN.json` — Chinese
- `locales/en-US.json` — English
- `composables/useI18n.ts` — i18n composable
- `components/LanguageSwitcher.vue` — language switcher

New copy must be added to both files; `npm run i18n:check` compares the key sets.

---

## 🌐 Language

- **English** (this document)
- [中文文档](https://vinthuy.github.io/reqmango/kb/architecture/frontend/)
