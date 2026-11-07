# Architecture Overview

reqmango 采用前后端分离架构，Go + Vue 3 全栈。

**最后更新**: 2026-11-07

> 本文描述**分层与职责**，不维护文件数/路由数这类会随手改的清单数字。
> 需要具体清单时，直接看对应目录，或 `backend/internal/router/router.go`（全部路由注册的唯一入口）。

---

## 技术栈总览

| 层级 | 技术 | 状态 |
|------|------|------|
| 后端框架 | Go 1.25 + Gin | ✅ 主力 |
| ORM | GORM | ✅ |
| 数据库 | PostgreSQL 16+ | ✅ |
| 认证 | JWT (golang-jwt/v5) | ✅ |
| LLM 集成 | DeepSeek / Anthropic / OpenAI-compatible | ✅ |
| 前端框架 | Vue 3 + Composition API | ✅ |
| 前端构建 | Vite 5 | ✅ |
| 状态管理 | Pinia | ✅ |
| CSS | Tailwind CSS | ✅ |
| 类型系统 | TypeScript | ✅ |
| E2E 测试 | Playwright | ✅ |

版本以 `backend/go.mod` 与 `frontend/package.json` 为准，详见[技术栈](tech-stack.md)。

---

## 系统分层

```
┌──────────────────────────────────────────┐
│              前端 (Vue 3)                 │
│  Views → Components → Stores → API →     │
│         TypeScript Types                 │
└──────────────────┬───────────────────────┘
                   │ HTTP (JSON + JWT) + SSE (AI streaming)
┌──────────────────▼───────────────────────┐
│            Go 后端 (Gin)                  │
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

## 后端分层职责

| 层 | 目录 | 职责 |
|----|------|------|
| Handler | `internal/handler/` | HTTP 请求绑定，调用 Service，写响应。不含业务逻辑 |
| Service | `internal/service/` | 纯业务逻辑，跨 Model 操作，返回 AppError |
| Model | `internal/model/` | GORM 模型定义，表结构映射 |
| DTO | `internal/dto/` | 请求/响应结构体，与 Model 解耦 |
| Middleware | `internal/middleware/` | JWT 认证、RBAC 鉴权、租户收敛、CORS、日志、限流 |
| Router | `internal/router/` | 路由注册，挂载中间件和 Handler |
| Common | `internal/common/` | 错误类型、常量、分页工具 |
| RQL | `internal/rql/` | ReqMan Query Language (lexer/parser/executor) |
| AI | `internal/ai/` | LLM 客户端、harness、分诊与对话编排 |
| Workflow | `internal/workflow/` | 工作流执行引擎 |

## 前端分层职责

| 层 | 目录 | 职责 |
|----|------|------|
| Views | `src/views/` | 页面级组件，对应路由（含 `views/agent/`、`views/agents/` 的 Agent 控制台，默认隐藏） |
| Components | `src/components/` | 可复用组件 |
| Stores | `src/stores/` | Pinia 状态管理（auth / cycle / module / issue / project / notification / agentLoop / agentPipeline） |
| API | `src/api/` | Axios 封装，按后端资源一模块一文件 |
| Types | `src/types/` | TypeScript 类型定义（含 filters.ts） |
| Composables | `src/composables/` | 组合式函数（useAI / useRQL / useFilters / usePermission / useSSE / useI18n …） |
| Router | `src/router/` | Vue Router 配置（含 minRoleLevel 守卫） |

---

## 已实现模块一览

| 模块 | Go 后端 | 前端 | 说明 |
|------|---------|------|------|
| Auth（认证） | ✅ | ✅ | JWT register/login/me |
| Workspace（工作空间） | ✅ | ✅ | CRUD + 成员管理 |
| Project（项目） | ✅ | ✅ | CRUD + 归档 + 统计 + Lead + Subscriber |
| Issue（工作项） | ✅ | ✅ | CRUD + 层级/归档/批量/导入导出/Issue-Page关联 |
| State（状态） | ✅ | ✅ | 5 固定分组，项目级 CRUD |
| Label（标签） | ✅ | ✅ | 项目级 CRUD |
| Cycle（周期） | ✅ | ✅ | CRUD + 开始/结束/取消 + 燃尽图 + 进度 |
| Module（模块） | ✅ | ✅ | 树形 CRUD + Issue 关联 + 进度统计 |
| CustomField（自定义字段） | ✅ | ✅ | 7 种类型 + 选项管理 + Issue 值绑定 |
| IssueType（工作项类型） | ✅ | ✅ | 工作空间级 + 项目级，层级绑定 |
| TypeTemplate | ✅ | ✅ | 工作空间级类型蓝图 |
| ProjectTemplate | ✅ | ✅ | 项目模板 + Apply |
| WorkItemTemplate | ✅ | ✅ | 工作项创建模板 |
| Workflow（工作流） | ✅ | ✅ | 状态流转 + 审批规则 |
| Automation（自动化） | ✅ | ✅ | Trigger-Condition-Action（点分事件）；AI 模板 + `dispatch_agent`；NL `automation-preview` 草稿后确认保存 |
| Comment（评论） | ✅ | ✅ | 嵌套回复 + resolve |
| Notification（通知） | ✅ | ✅ | CRUD + 已读/未读 + 摘要统计 |
| Saved Views（保存视图） | ✅ | ✅ | JSONB 筛选/排序/列配置 + 默认视图 |
| Pages（页面文档） | ✅ | ✅ | 树形层级 + 归档/恢复 + Markdown |
| Release（发布管理） | ✅ | ✅ | CRUD + Issue 关联 + 进度 |
| Estimate（估算） | ✅ | ✅ | 项目目录 Points/Categories/Time；**Issue 挂 `estimate_point_id`（points）**；设置页 `EstimatePointManager`；mutate 需 `settings:manage`；categories/time 挂到 Issue 仍 Later |
| Attachment（附件） | ✅ | ✅ | 文件上传 + 元数据 |
| Relation（关联类型） | ✅ | ✅ | 自定义 in/out 命名 + Issue 关联 |
| RQL（查询语言） | ✅ | ✅ | 自定义 DSL 搜索 |
| AI（智能助手） | ✅ | ✅ | Chat SSE + NL Search + Smart Create + Analyze（Issue/项目）+ Page AI + Triage；项目总结可存 Page；NL→自动化预览；Dashboard `ai_summary` |
| Dashboard（仪表盘） | ✅ | ✅ | Widget 网格：图表/数字卡/燃尽/报表/列表 + `ai_summary`（打开加载、可刷新；无定时出报） |
| Time Tracking（工时） | ✅ | ✅ | Start/Stop/手动录入/`duration` **秒**；List/Summary；`time_track:view|create|delete`；侧栏展示累计已记工时 |
| Merge Duplicates（合并重复） | ✅ | ✅ | `POST /issues/merge` 软合并（不删源）；详情「合并到…」 |
| Recurring（重复工作项） | ✅ | ✅ | daily/weekly/monthly/cron + UI 配置 |
| Intake & Triage（接收分诊） | ✅ | ✅ | 公开提交 + Accept/Reject + AI 分析 |
| Conditional Fields（条件字段） | ✅ | ✅ | 字段显隐规则 |
| Cover Image（封面图） | ✅ | ✅ | 工作项封面 |
| Command Palette（⌘K） | ✅ | ✅ | 键盘快速导航 |
| AI Settings（AI 配置） | ✅ | ✅ | Provider/Model/APIKey UI 配置 |
| RBAC（权限角色） | ✅ | ✅ | 55 权限 + 3 默认角色 + 自定义角色管理 |
| Quick Create（快速创建） | ✅ | ✅ | 行内快速创建工作项 |
| Git Integration（Git 集成） | ✅ | ✅ | GitHub/GitLab 原生集成 + Webhook |
| Project CustomField Enrollment | ✅ | ✅ | 项目级自定义字段启用/禁用 |
| Workspace Workflow（工作空间工作流） | ✅ | ✅ | 状态机 CRUD + transitions（CORE-04 已修）；项目可覆盖 |
| Calendar/Gantt View | ✅ | ✅ | 日历视图 (IssueCalendar.vue) + 甘特图视图 (IssueGantt.vue) |
| AI Agent 系统 | ✅ | 🔒 | 开箱 PM Agent + 指派/@Agent；控制台默认隐藏（Stage B）；Harness/Loop **废止产品化** |
| 聊天系统 | ✅ | ✅ | Chat, Messages, SSE streaming |
| 页面版本管理 | ✅ | ✅ | PageVersion, PageTemplate |
| 审批系统 | ✅ | ✅ | Approval |
| 工作流可视化 | ✅ | ✅ | Workflow Nodes/Edges/Execute/Runs |

---

## 当前推进中

| 模块 | 说明 |
|------|------|
| 存量质量 | 产品主路径与核心定制门禁已 PASS；继续稳定性与数据质量，不新开 AI 大功能 |

## 产品方向备忘

- **现行**：原生 PM + 原生 AI 项目管理路径 AI（Intake / Analyze / 自动化预览 / `ai_summary`）
- **核心门禁**：工作项类型 / 自定义字段 / 工作流（含 workspace 状态机）/ 自动化 — 验收 §H
- **废止**：Harness / Loop / 独立 Agent 平台 Agent 控制台产品化 → [superseded/agent-platform](https://github.com/vinthuy/reqmango/blob/master/docs/superseded/agent-platform/README.md)
- **Stage B**：`/agents*` 默认重定向，解锁键 `localStorage.rm_advanced_agents=1`

## 未来可扩展

（暂无；不以 Agent 平台产品化为排期）

---

## 🌐 语言

- **中文** (本文档)
- [English](https://vinthuy.github.io/reqmango/en/kb/architecture/)
