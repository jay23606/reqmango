# Go Backend Architecture（Go 后端架构）

**最后更新**: 2026-11-07

> 目录树只列**主链路**和代表性文件。文件会持续增删，具体清单以目录为准——
> 这份文档承诺的是分层职责和扩展点，不是文件数量。

---

## 项目路径

```
backend/
├── cmd/server/main.go              # 入口
├── cmd/route_audit/                # 路由审计工具（列出/校验注册的路由）
├── internal/
│   ├── config/config.go            # 配置加载（Viper + 环境变量 + .env）
│   ├── common/
│   │   ├── constants.go            # 角色、优先级、状态组常量
│   │   ├── error_codes.go          # 错误码定义（含 i18n 消息）
│   │   ├── errors.go               # AppError 类型
│   │   └── pagination.go           # 分页辅助
│   ├── model/                      # GORM 模型（领域见「数据模型」）
│   │   ├── base.go                # BaseModel 嵌入结构（ID/CreatedAt/UpdatedAt/DeletedAt）
│   │   ├── user.go                # User（含 IsSuperuser、GetID、IsSuper 方法）
│   │   ├── workspace.go           # Workspace, WorkspaceMember
│   │   ├── project.go             # Project, ProjectMember
│   │   ├── issue.go               # Issue, IssueAssignee, IssueLabel, IssueActivity
│   │   ├── issue_type.go          # IssueType + 字段绑定
│   │   ├── state.go               # State, StateTransition
│   │   ├── cycle.go / module.go   # 周期、模块
│   │   ├── page.go                # Page（层级文档）+ 模板 + 版本
│   │   ├── workflow.go            # Workflow + AutomationRule + AutomationExecution
│   │   ├── custom_field.go        # CustomField + 7 种类型
│   │   ├── role.go / permission.go # RBAC
│   │   └── …                      # 其余模型见 internal/model/
│   ├── dto/
│   │   ├── request/               # 请求 DTO，带 binding 校验标签
│   │   └── response/              # 响应 DTO，与 Model 解耦
│   ├── service/                    # 业务逻辑，按资源一文件（*_service.go）
│   │   ├── auth_service.go        # JWT 签发与验证
│   │   ├── workspace_service.go   # Workspace CRUD + 成员
│   │   ├── project_service.go     # Project CRUD + 成员 + 统计 + 归档
│   │   ├── issue_service.go       # Issue CRUD + 搜索 + 批量 + 导入/导出 + 关联
│   │   ├── cycle_service.go       # Cycle CRUD + 状态流转 + 进度 + 燃尽图
│   │   ├── module_service.go      # Module CRUD + 树形 + 统计
│   │   ├── page_service.go        # Page CRUD + 树形 + 归档
│   │   ├── workflow_service.go    # Workflow CRUD + 转换验证
│   │   ├── workflow_executor.go   # Workflow 执行引擎
│   │   ├── notification_service.go # 通知 + 标记已读 + 提醒
│   │   ├── webhook_service.go     # Webhook 发送 + HMAC-SHA256 签名
│   │   ├── automation_service.go  # 自动化规则 CRUD + 触发执行
│   │   ├── role_service.go        # RBAC 角色 + 权限查询
│   │   ├── sse_hub.go             # SSE 实时事件中心
│   │   └── …                      # 其余服务见 internal/service/
│   ├── handler/                    # HTTP Handler，一资源一文件
│   │   ├── auth_handler.go / workspace_handler.go / project_handler.go
│   │   ├── issue_handler.go / cycle_handler.go / module_handler.go
│   │   ├── workflow_handler.go / notification_handler.go / webhook_handler.go
│   │   ├── sse_handler.go         # SSE 端点
│   │   └── …                      # 其余见 internal/handler/
│   ├── router/router.go            # 全部路由注册的唯一入口
│   ├── middleware/                 # 中间件（10 个）
│   │   ├── auth.go                # JWT：解析 Bearer token → 注入 currentUser
│   │   ├── authorization.go       # RBAC：RequirePermission / RequireRoleLevel
│   │   ├── access.go              # 工作空间成员准入：RequireWorkspaceMember
│   │   ├── scope.go               # 租户收敛：所有认证路由统一校验资源归属
│   │   ├── metrics.go             # 延迟直方图 + 慢请求日志
│   │   ├── recovery.go            # panic 恢复，统一 JSON 错误
│   │   ├── cors.go                # CORS 策略
│   │   ├── language.go            # i18n 语言检测（Header/Query）
│   │   ├── logger.go              # 请求日志
│   │   └── rate_limiter.go        # Token Bucket 限流（默认 500 req/min，0=禁用）
│   ├── ai/                         # AI 子系统
│   │   ├── llm/client.go          # LLM 客户端（DeepSeek / Anthropic）
│   │   ├── service/               # AI 对话、分诊、分析、Agent 编排
│   │   ├── harness/               # 工具/技能执行、pipeline、reviewer
│   │   ├── loop/                  # Loop 运行时（已废止产品化）
│   │   └── handler/               # AI / Agent / Loop / Pipeline 端点
│   ├── rql/                        # RQL 查询语言引擎
│   │   ├── lexer.go / parser.go / ast.go   # 词法 → 语法 → AST
│   │   ├── builder.go / executor.go        # AST → SQL/GORM、执行
│   │   ├── handler.go / service.go         # HTTP 端点与查询服务
│   │   └── *_test.go
│   ├── workflow/rules.go           # 状态机规则
│   ├── plugin/                     # 插件注册表 + 钩子
│   ├── security/sanitize.go        # HTML 清洗（bluemonday）
│   ├── scheduler/                  # 后台定时任务（记忆清理等）
│   ├── version/version.go          # 构建信息，供 GET /version
│   ├── i18n/
│   │   ├── i18n.go                 # 国际化逻辑
│   │   ├── messages_en.json        # 英文错误消息
│   │   └── messages_zh.json        # 中文错误消息
│   ├── client/agent_client.go      # Agent 调用客户端
│   ├── issuetype/visible.go        # 类型可见性判定
│   ├── randutil/randutil.go        # 随机串（密码/密钥）
│   ├── testutil/                   # 测试夹具（DB / HTTP / fixtures）
│   └── seed/
│       ├── seed.go                 # 种子数据（演示用户 + Issues + Workspace/Project）
│       └── seed_rbac.go            # RBAC 种子（55 权限 + 3 默认角色）
```

---

## 分层架构

```
Router ──→ Middleware Chain ──→ Handler ──→ Service ──→ Model (GORM)
    │           │                    │            │
    │     Auth/RBAC/Scope            │            └── DB 操作
    │     CORS/Lang/Log/RateLimit    └── DTO 绑定 + HTTP 响应
    └── Gin 路由树（数百个端点，全部注册在 internal/router/router.go）
```

### Handler 层
- 使用 Gin Context 绑定请求参数（`c.ShouldBindJSON`, `c.Param`, `c.Query`）
- 调用 Service 层方法
- 返回统一格式的 JSON 响应
- **不包含任何业务逻辑**

### Service 层
- 纯 Go 业务逻辑
- 使用 GORM 进行数据库操作
- 返回自定义 `AppError` 类型（含 HTTP 状态码和 i18n 错误码）
- 跨 Model 操作（如 Issue 关联 Assignee、Label、Attachment）
- 注入依赖（Notification、Webhook、Automation、Slack 等服务）

### Model 层
- GORM 标签定义表结构 + 索引
- 所有模型嵌入 `BaseModel`（ID uint64, CreatedAt, UpdatedAt, DeletedAt）
- `DeletedAt gorm.DeletedAt` 实现软删除
- 关联关系通过 GORM Tag + Preload 管理

### DTO 层
- `request/` — 客户端请求结构体，带 `binding` 验证标签
- `response/` — 服务端响应结构体
- 与 Model 分离，避免暴露数据库内部字段

### RQL 层
- 自研 RQL（reqmango Query Language）查询语言
- 完整的词法分析 → 语法分析 → AST → SQL/GORM 生成流水线
- 端点：`POST /api/v1/rql/search`
- 功能对标主流查询语言，社区版免费可用

---

## 核心组件

### AppError（统一错误处理）

```go
type AppError struct {
    StatusCode int
    ErrorCode  string  // RQL 错误码（含 i18n 支持）
    Message    string
    Err        error
}
```

预定义错误工厂：`NotFound()`, `Conflict()`, `Unauthorized()`, `Forbidden()`, `Validation()`, `Internal()`, `BadRequest()`

### i18n 错误消息
- `messages_en.json` / `messages_zh.json` 双语言
- `ErrorCode` 映射到具体错误消息
- 语言通过 `Accept-Language` Header 或 `lang` Query 参数检测

### Pagination

```go
func ParsePagination(limitStr, offsetStr string, defaultLimit, maxLimit int) Pagination
```

调用方传入查询参数 `limit` 和 `offset`。工作项列表默认 50、上限 100；其它接口的默认值不同。总数在响应头 `X-Total-Count`。

### 认证流程

1. `POST /api/v1/auth/login` — bcrypt 验证密码，返回 `access_token` + `token_type` + `expires_at`
2. 后续请求在 `Authorization: Bearer <token>` 头中携带 JWT
3. Auth 中间件解析 JWT，查询用户，注入 `c.Set("currentUser", user)`
4. Handler 通过 `c.MustGet("currentUser").(*model.User)` 获取

### RBAC 与租户收敛

- `authorization.go` 提供 `RequirePermission(db, permCode)` 和 `RequireRoleLevel(db, minLevel)` 两个中间件
- `access.go` 提供 `RequireWorkspaceMember`，挡住非成员
- `scope.go` 在每条认证路由上统一校验「请求里的 workspace/project 与资源真实归属一致」，
  handler 忘了检查也不会跨租户读写
- 超级用户（IsSuperuser=true）自动旁路所有权限检查
- 权限格式：`resource:action`（如 `issue:create`, `workspace:manage`）
- 角色级别：Admin=20（全部权限）、Member=15（创建编辑）、Guest=5（只读）
- 支持 workspace-level 和 project-level 自定义角色
- 5 个 RBAC API：`GET/POST /api/v1/workspaces/:wsParam/roles`、`PUT/DELETE /api/v1/workspaces/:wsParam/roles/:id`、`GET /api/v1/permissions`

### 限流

- Token Bucket 算法，按客户端 IP 分组
- 默认 500 req/min（可通过 `RATE_LIMIT_REQUESTS` 环境变量配置）
- `limit=0` 时完全禁用（开发/CI 环境）
- 响应头：`X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

### SSE 实时通知

- `sse_hub.go` — 广播 Hub 实现
- `sse_handler.go` — SSE 端点 `/api/v1/sse`
- 通知类型通过 SSE 实时推送到前端

---

## API 路由总览

所有路由前缀 `/api/v1`。下表按路由组给资源范围；完整路径以 `internal/router/router.go` 为准
（需要机器可读的清单时用 `go run ./cmd/route_audit`）：

| 路由组 | 资源 |
|--------|------|
| `/auth/` | 注册 / 登录 / 当前用户 |
| `/workspaces/` | CRUD + 成员 + AI Agent + MCP + GitHub + Slack 配置 |
| `/projects/` | CRUD + 成员 + 统计 + 报表 + Webhook + 自动化 |
| `/projects/:id/cycles` | Cycle CRUD + 状态流转 + 进度 + 燃尽图 |
| `/projects/:id/modules` | Module CRUD + 树形 + Issue 关联 + 统计 |
| `/projects/:id/pages` | Page CRUD + 归档 |
| `/projects/:id/issue-types` | 问题类型 CRUD + 字段绑定 |
| `/projects/:id/workflows` | Workflow CRUD + 规则 |
| `/projects/:id/releases` | Release + Roadmap |
| `/projects/:id/ai/` | 聊天/搜索/图表/分诊；`analyze`（Issue 或项目）；`sprint-plan`；`automation-preview`（NL→规则草稿，不落库） |
| `/projects/:id/automations` | 自动化规则 CRUD + 执行（含 `dispatch_agent`） |
| `/projects/:id/dashboards` | Dashboard CRUD + widgets；`GET .../full` 返回 widget_data（含 `ai_summary`） |
| `/projects/:id/estimate-points` | 估点目录 / settings；写操作 `settings:manage` |
| `/issues/` | CRUD（含 `estimate_point_id`）+ 搜索 + 批量 + 导入/导出（CSV `estimate`→点）+ 树 + 关联 + 工时 + 软合并 + 周期性 |
| `/issues/:id/time-tracks` | 手动录入 / start / stop / list / summary / delete；`time_track:*`；`duration` 秒 |
| `/comments/` | CRUD + 回复 + 解决 |
| `/custom-fields/` | CRUD + 选项 + 条件规则 |
| `/templates/` | 项目模板 + 类型模板 + 工作项模板 |
| `/rql/search` | RQL 查询执行 |
| `/notifications/` | 列表 + 已读 + 提醒 |
| `/webhook/github/:id` | GitHub Webhook 接收（公开端点） |
| `/intake/:projectId` | 公开问题提交（无需认证） |
| `/sse` | 实时事件推送 |
| `/workspaces/:wsParam/roles` | RBAC 角色 CRUD |
| `/permissions` | 全局权限枚举查询 |

---

## 配置

通过环境变量 + `.env` 文件配置（Viper 依次在 `.`、`..`、`../..` 找 `.env`）。
完整清单见[配置项](../../configuration.md)，`backend-go` 只列影响架构的几个：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DATABASE_URL` | PostgreSQL DSN | `postgres://postgres:postgres@localhost:5432/reqmango?sslmode=disable` |
| `SECRET_KEY` | JWT 签名密钥 | 未设置时随机生成并告警 |
| `PORT` | 监听端口 | `8000` |
| `RATE_LIMIT_REQUESTS` | 限流阈值（req/min） | 500 |
| `RATE_LIMIT_WINDOW_SEC` | 限流窗口 | 60 |
| `AI_API_KEY` | AI 密钥（也读 `DEEPSEEK_API_KEY`） | 空 |
| `AI_PROVIDER` | AI 提供商 | deepseek |
| `AI_MODEL` | AI 模型 | deepseek-chat |
| `AI_BASE_URL` | AI API 地址 | `https://api.deepseek.com/v1` |
| `DEBUG` | Gin 调试模式 | true |
| `SLOW_REQUEST_MS` | 慢请求阈值（ms，0 关闭） | 1000 |

---

## 🌐 语言

- **中文** (本文档)
- [English](https://vinthuy.github.io/reqmango/en/kb/architecture/backend-go/)
