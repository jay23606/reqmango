# Frontend Architecture（前端架构）

**最后更新**: 2026-11-07

> 目录树列的是**结构与约定**。文件数量增长很快，具体清单以 `frontend/src/` 为准。

---

## 技术栈

| 组件 | 选型 |
|------|------|
| 框架 | Vue 3 (Composition API + `<script setup>`) |
| 构建 | Vite 5 |
| 语言 | TypeScript |
| 状态管理 | Pinia |
| 路由 | Vue Router 4 |
| CSS | Tailwind CSS |
| HTTP | Axios |
| 富文本 | TipTap |
| SSE | fetch + ReadableStream (AI streaming) |

运行时版本（Node 24）见[技术栈](tech-stack.md)。

## 项目结构

```
frontend/src/
├── main.ts
├── App.vue
├── style.css
│
├── api/                  — Axios 封装，一个后端资源一个模块
│   ├── index.ts          — Axios 实例 + JWT 拦截器
│   ├── auth.ts / workspace.ts / project.ts
│   ├── issue.ts / cycle.ts / module.ts / page.ts
│   ├── time-track.ts / estimate-point.ts
│   ├── ai.ts             — AI Chat SSE / Search / Create
│   ├── workflow.ts / automation.ts / issue-type.ts
│   ├── custom-field.ts / conditional-field.ts / saved-view.ts
│   ├── dashboard.ts / report.ts / metric.ts
│   ├── agent.ts / plugin.ts / mcp.ts / role.ts
│   └── …                 — 其余见 api/
│
├── types/                — TypeScript 类型，按模块拆分
│   ├── index.ts
│   ├── filters.ts        — 筛选条件类型 + FILTER_FIELDS + buildRQL/parseRQL
│   ├── issue.ts / project.ts / cycle.ts / module.ts
│   └── …                 — 其余见 types/
│
├── stores/               — Pinia
│   ├── auth.ts           — 认证状态
│   ├── cycle.ts          — 周期 CRUD + 进度
│   ├── module.ts         — 模块 CRUD + 树形
│   ├── issue.ts / project.ts
│   ├── notification.ts
│   └── agentLoop.ts / agentPipeline.ts
│
├── router/index.ts       — Vue Router 配置（含 minRoleLevel 守卫）
│
├── views/                — 页面级组件
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
│   ├── agent/            — Agent 控制台（Stage B 默认隐藏）
│   └── agents/           — Agent 模板/会话/运行时/记忆等（同上）
│
├── components/           — 可复用组件，按领域分组
│   ├── Agent / AI        — AgentAuditLog, AgentSelector, AIChatSidebar, AICopilot,
│   │                       AIChartRenderer, AIResultActions …
│   ├── Dashboard         — DashboardGrid, WidgetCard（含 ai_summary）, WidgetConfigPanel
│   ├── Issue             — IssueCard, IssueList, IssueKanban, IssueDetailPanel,
│   │                       IssuePropertySidebar（含估点选择 / 已记工时）,
│   │                       SubIssuePanel, IssueTreeView, IssueGantt, IssueCalendar,
│   │                       TimeTrackPanel（计时 + 手动分钟录入）, MergeIntoIssueDialog
│   ├── Cycle / Module    — CycleCard, CycleBurndownChart, ModuleTree, ModuleDetailPanel …
│   ├── Config            — CustomField*, ConditionalField*, EstimatePointManager（项目设置）
│   ├── Workflow          — WorkflowManager, WorkflowRule*, StateTransition,
│   │                       WorkflowVisualization
│   ├── Automation        — AutomationManager, AutomationRuleBuilder（含 dispatch_agent、
│   │                       NL 草稿预填）, AutomationTemplateList
│   ├── Template          — ProjectTemplateManager, WorkItemTemplateManager,
│   │                       WorkspaceIssueTypeManager
│   ├── Page              — PageTree, PageVersionPanel, PageVersionDiff, PageTabConfig
│   ├── Filter / View     — FilterBar（统一筛选栏）, SavedViewSelector, QuickFilterChips
│   ├── Import / Triage   — ImportIssuesModal, QuickCreateInput, TriagePanel
│   ├── Common            — CommentList, AttachmentManager, NotificationCenter,
│   │                       RichTextEditor, UserSelect, LabelSelector, ConfirmDialog …
│   └── RQL               — RQLInput, RQLHistory
│
└── composables/          — 组合式函数
    ├── useAPI / useSSE / useChatSSE  — 请求与流式消费
    ├── useAI               — AI SSE 消费
    ├── useFilters          — 筛选状态（Provide/Inject，RQL 双向同步）
    ├── useIssueFilters     — 工作项筛选
    ├── useRQL              — RQL 查询历史
    ├── usePermission       — RBAC 权限检查
    ├── useAutoTriageWatch  — 分诊状态监听
    ├── useProductFlags     — 功能开关（含 Agent 控制台解锁）
    ├── useWorkspaceId      — 当前工作空间上下文
    ├── useRelationHelpers  — 关联类型辅助
    ├── useDashboard / useReportChart / useMarkdown
    ├── useI18n / useDarkMode / useToast / useConfirm
    └── …
```

## 路由表

路由定义在 `frontend/src/router/index.ts`，下表只列主路径（不含各详情页的子路由）：

| 路径 | 视图 | 说明 |
|------|------|------|
| `/` | Home | 工作空间列表 |
| `/login` | Login | 登录 |
| `/register` | Register | 注册 |
| `/workspace/:slug` | Workspace | 工作空间详情 + 项目列表 |
| `/workspace/:slug/settings` | WorkspaceSettings | 工作空间配置 |
| `/workspace/:slug/overview` | WorkspaceOverview | 工作空间概览 |
| `/workspace/:slug/roadmap` | Roadmap | 路线图 |
| `/workspace/:slug/initiatives` | Initiatives | 战略目标 |
| `/workspace/:slug/analytics` | WorkspaceAnalytics | 工作空间分析 |
| `/workspace/:slug/project/:id` | Project | 项目主页 |
| `/workspace/:slug/project/:id/pages` | ProjectPages | 页面文档 |
| `/workspace/:slug/project/:id/settings` | ProjectSettings | 项目配置 |
| `.../settings/workflows/:workflowId` | WorkflowDetail | 工作流详情 |
| `/workspace/:slug/project/:id/analytics` | Analytics | 项目分析 |
| `/workspace/:slug/project/:id/dashboards` | Dashboard | 仪表盘（可加 `ai_summary` 卡片，刷新重拉 Analyze） |
| `/workspaces/:wid/projects/:pid/issues/:iid` | IssueDetail | 工作项详情 |
| `/workspaces/:wid/projects/:pid/issues/new` | IssueCreate | 创建工作项 |
| `/workspaces/:wid/projects/:pid/custom-fields` | CustomFields | 自定义字段 |
| `/workspaces/:wid/projects/:pid/issue-types` | IssueTypeList | 工作项类型 |
| `/workspaces/:wid/projects/:pid/cycles/new` | CycleCreate | 创建周期 |
| `/workspaces/:wid/projects/:pid/cycles/:cid` | CycleDetail | 周期详情 |
| `/workspace/:slug/project/:id/issues/:iid` | IssueDetail | 工作项详情（slug 风格） |
| `/workspace/:slug/project/:id/issues/new` | IssueCreate | 创建工作项（slug 风格） |
| `/workspace/:slug/project/:id/cycles/new` | CycleCreate | 创建周期（slug 风格） |
| `/workspace/:slug/project/:id/cycles/:cid` | CycleDetail | 周期详情（slug 风格） |
| `/intake/:projectId` | IntakeForm | 公开提交 |

两种路径风格（`/workspace/:slug/...` 与 `/workspaces/:wid/...`）并存是历史遗留，
新增页面请沿用 slug 风格，避免继续扩散。

## 架构模式

### API 层

```typescript
// api/index.ts — Axios 实例
const api = axios.create({ baseURL: '/api/v1' })
// 请求拦截器：自动附加 JWT Bearer token
// 响应拦截器：401 → 跳转 login
```

### AI SSE 消费

```typescript
// composables/useAI.ts / useChatSSE.ts
// fetch + ReadableStream 消费 SSE
// 事件类型: text | tool_call | tool_result | done | error
```

### 组件模式

- 所有组件使用 `<script setup lang="ts">`
- Props: `defineProps<T>()`
- Emits: `defineEmits<T>()`
- 状态管理: 页面级用 local ref，跨组件用 Pinia store

---

## 国际化（i18n）

前端支持中英文双语言：

- `locales/zh-CN.json` — 中文语言包
- `locales/en-US.json` — 英文语言包
- `composables/useI18n.ts` — 国际化组合式函数
- `components/LanguageSwitcher.vue` — 语言切换组件

新增文案必须同时写两份语言包；`npm run i18n:check` 会比对两边的 key 集合。

---

## 🌐 语言

- **中文** (本文档)
- [English](https://vinthuy.github.io/reqmango/en/kb/architecture/frontend/)
