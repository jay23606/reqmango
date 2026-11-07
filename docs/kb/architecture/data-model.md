# Data Model（数据模型）

**最后更新**: 2026-11-07

生产库的表由 `backend/cmd/server/main.go` 的 `AutoMigrate` 创建。下面 131 张表就是这份清单，表名用 GORM 的命名策略（含各模型的 `TableName()`）解析得出。`internal/model` 里还有只当 JSON 用的结构体，不会建表，列在文末。

列级细节以对应的 `model` 文件为准。这里不抄字段清单，避免再写出一份过期的列目录。

---

## 通用字段

嵌入 `BaseModel`（`internal/model/base.go`）的模型都有：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `uint64` | 自增主键 |
| `created_at` / `updated_at` | `time.Time` | GORM 自动维护 |
| `deleted_at` | `gorm.DeletedAt` | 软删除；带索引 |
| `created_by_id` / `updated_by_id` | `*uint64` | 可空 |

主键是 `uint64`，不是 UUID。Issue 和 Page 的 `depth` 注释上限是 5。

---

## 用户与租户

| 表 | 模型 | 存什么 |
|----|------|--------|
| `users` | User | 账号 |
| `workspaces` | Workspace | 工作空间 |
| `workspace_members` | WorkspaceMember | 空间成员与角色级别 |
| `projects` | Project | 项目 |
| `project_members` | ProjectMember | 项目成员 |
| `project_subscribers` | ProjectSubscriber | 项目订阅者 |
| `personal_access_tokens` | PersonalAccessToken | 用户的长期令牌。只存 SHA-256，明文只在创建时返回 |

---

## 工作项

| 表 | 模型 | 存什么 |
|----|------|--------|
| `issues` | Issue | 工作项（含可空 `estimate_point_id` → `estimate_points`） |
| `issue_assignees` | IssueAssignee | 负责人 |
| `issue_labels` | IssueLabel | 标签关联 |
| `issue_cycles` | IssueCycle | 周期关联 |
| `issue_pages` | IssuePage | 页面关联 |
| `issue_activities` | IssueActivity | 操作历史 |
| `issue_watchers` | IssueWatcher | 关注者 |
| `states` | State | 状态（项目或工作空间） |
| `labels` | Label | 标签 |
| `comments` | Comment | 评论。Agent 建议以 JSON 存在评论上，不是独立表 |
| `attachments` | Attachment | 附件元数据 |
| `time_tracks` | TimeTrack | 工时条目；`duration` 为**秒**；可运行中（`ended_at` 空）或已结束 |
| `recurrence_rules` | RecurrenceRule | 重复创建规则 |
| `relation_types` | RelationType | 关联类型（in/out 名称） |
| `issue_relations` | IssueRelation | 工作项之间的关联 |
| `project_intake_settings` | ProjectIntakeSetting | 项目的 Intake 表单 / Webhook / 邮件开关与 token |

---

## 类型、字段、模板

| 表 | 模型 | 存什么 |
|----|------|--------|
| `issue_types` | IssueType | 工作项类型 |
| `issue_type_fields` | IssueTypeField | 类型与字段的绑定 |
| `issue_type_imports` | IssueTypeImport | 类型导入记录 |
| `issue_type_templates` | IssueTypeTemplate | 工作空间级类型蓝图 |
| `issue_type_template_fields` | IssueTypeTemplateField | 蓝图字段 |
| `custom_fields` | CustomField | 自定义字段定义 |
| `custom_field_options` | CustomFieldOption | 下拉选项 |
| `issue_custom_field_values` | IssueCustomFieldValue | 工作项上的字段值 |
| `project_custom_field_enrollments` | ProjectCustomFieldEnrollment | 项目是否启用某字段 |
| `conditional_fields` | ConditionalField | 字段显隐规则 |
| `project_templates` | ProjectTemplate | 项目模板 |
| `project_template_types` | ProjectTemplateType | 模板包含的类型 |
| `work_item_templates` | WorkItemTemplate | 创建工作项用的模板 |

---

## 规划

| 表 | 模型 | 存什么 |
|----|------|--------|
| `cycles` | Cycle | 周期 |
| `modules` | Module | 模块树 |
| `module_issues` | ModuleIssue | 模块与工作项 |
| `module_inheritance_overrides` | ModuleInheritanceOverride | 模块继承覆盖 |
| `releases` | Release | 发布 |
| `release_issues` | ReleaseIssue | 发布与工作项 |
| `initiatives` | Initiative | 工作空间级目标 |
| `initiative_projects` | InitiativeProject | 目标与项目 |
| `estimate_points` | EstimatePoint | 点数估算 |
| `estimate_categories` | EstimateCategory | 分类估算 |
| `estimate_times` | EstimateTime | 时间估算 |
| `project_estimate_settings` | ProjectEstimateSettings | 项目启用哪种估算 |

---

## 状态机与自动化

`workflows` 是工作项状态机。`state_transitions` 只属于它。编排用的 DAG 在 `agent_workflows` 及其节点 / 边 / 运行表，两边不再共用 `workflow_kind`。

| 表 | 模型 | 存什么 |
|----|------|--------|
| `workflows` | Workflow | 状态机 |
| `state_transitions` | StateTransition | 状态转换 |
| `automation_rules` | AutomationRule | 自动化规则 |
| `automation_executions` | AutomationExecution | 规则执行记录 |
| `automation_rule_overrides` | AutomationRuleOverride | 规则覆盖 |
| `approvals` | Approval | 审批单 |
| `approval_records` | ApprovalRecord | 审批动作 |
| `agent_workflows` | AgentWorkflow | 多 Agent 编排定义 |
| `workflow_nodes` | WorkflowNode | 编排节点 |
| `workflow_edges` | WorkflowEdge | 编排边 |
| `workflow_runs` | WorkflowRun | 一次编排运行 |
| `workflow_node_runs` | WorkflowNodeRun | 节点运行 |

---

## 页面、视图、报表

| 表 | 模型 | 存什么 |
|----|------|--------|
| `pages` | Page | 项目文档 |
| `page_templates` | PageTemplate | 页面模板 |
| `page_versions` | PageVersion | 页面版本 |
| `project_page_tabs` | ProjectPageTab | 项目页 Tab |
| `project_updates` | ProjectUpdate | 项目状态更新 |
| `saved_views` | SavedView | 保存的筛选视图 |
| `saved_dashboards` | SavedDashboard | 仪表盘 |
| `dashboard_widgets` | DashboardWidget | 仪表盘卡片 |
| `saved_reports` | SavedReport | 保存的报表 |
| `search_templates` | SearchTemplate | 搜索模板 |
| `metric_charts` | MetricChart | 指标图 |

---

## 权限

| 表 | 模型 | 存什么 |
|----|------|--------|
| `roles` | Role | 角色。级别 Admin=20、Member=15、Guest=5 |
| `permissions` | Permission | 权限码，`resource:action` |
| `role_permissions` | RolePermission | 角色与权限 |
| `field_permissions` | FieldPermission | 字段级读写 |

种子数据在 `internal/seed/seed_rbac.go`，当前写入 55 条权限和 3 个默认角色。

---

## 集成与插件

| 表 | 模型 | 存什么 |
|----|------|--------|
| `webhooks` | Webhook | 出站 Webhook |
| `github_connections` | GitHubConnection | GitHub 连接 |
| `slack_connections` | SlackConnection | Slack 连接 |
| `git_integrations` | GitIntegration | Git 集成配置 |
| `git_issue_links` | GitIssueLink | 工作项与 PR 的链接 |
| `mcp_configs` | MCPConfig | MCP server 配置 |
| `plugins` | Plugin | 插件 |
| `plugin_event_logs` | PluginEventLog | 插件事件 |

---

## 通知与聊天

| 表 | 模型 | 存什么 |
|----|------|--------|
| `notifications` | Notification | 站内通知 |
| `chats` | Chat | 会话 |
| `messages` | Message | 消息 |
| `message_reactions` | MessageReaction | 消息反应 |

---

## AI 与 Agent

这些表会随进程迁移创建。产品上 `/agents*` 默认隐藏，Harness / Loop 不作为产品方向，但表还在。

| 表 | 模型 | 存什么 |
|----|------|--------|
| `ai_configs` | AIConfig | 工作空间 AI 配置 |
| `ai_threads` | AIThread | AI 对话线程 |
| `ai_messages` | AIMessage | 对话消息 |
| `agents` | Agent | 可指派、可 @ 的 Agent |
| `agent_activities` | AgentActivity | Agent 动作审计 |
| `agent_registry` | AgentEntry | 内置 Agent 注册表（`internal/ai/registry`） |
| `project_agent_members` | ProjectAgentMember | 项目里的 Agent 成员 |
| `agent_templates` | AgentTemplate | Agent 模板 |
| `agent_configs` | AgentConfig | Agent 配置 |
| `agent_sessions` | AgentSession | 会话 |
| `agent_tasks` | AgentTask | 任务 |
| `agent_task_snapshots` | AgentTaskSnapshot | 任务快照 |
| `task_logs` | TaskLog | 任务日志 |
| `agent_decisions` | AgentDecision | 决策记录 |
| `agent_cost_budgets` | AgentCostBudget | 成本预算 |
| `agent_slas` | AgentSLA | SLA |
| `runtimes` | Runtime | 运行时 |
| `skills` | Skill | 技能 |
| `skill_execution_logs` | SkillExecutionLog | 技能执行 |
| `tools` | Tool | 工具 |
| `tool_permissions` | ToolPermission | 工具权限 |
| `tool_call_logs` | ToolCallLog | 工具调用 |
| `memory_entries` | MemoryEntry | 记忆条目 |
| `memory_sessions` | MemorySession | 记忆会话 |
| `memory_vector_indexes` | MemoryVectorIndex | 记忆向量索引 |
| `squads` | Squad | 协作组 |
| `squad_members` | SquadMember | 组成员 |
| `squad_tasks` | SquadTask | 组任务 |
| `squad_executions` | SquadExecution | 组执行 |
| `autopilot_tasks` | AutopilotTask | Autopilot 任务 |
| `autopilot_executions` | AutopilotExecution | Autopilot 执行 |
| `autopilot_webhooks` | AutopilotWebhook | Autopilot Webhook |
| `agent_loops` | Loop | Loop 定义 |
| `agent_loop_runs` | LoopRun | Loop 运行 |
| `agent_loop_iterations` | LoopIteration | Loop 迭代 |
| `agent_pipelines` | Pipeline | Pipeline |
| `agent_pipeline_runs` | PipelineRun | Pipeline 运行 |
| `developer_jobs` | DeveloperJob | Developer Agent 作业 |
| `tester_jobs` | TesterJob | Tester Agent 作业 |
| `cicd_configs` | CICDConfig | CI/CD 配置 |
| `build_records` | BuildRecord | 构建记录 |
| `sdlc_workflows` | SDLCWorkflow | SDLC 流程 |
| `sdlc_stages` | SDLCStage | SDLC 阶段 |

---

## 其它

| 表 | 模型 | 存什么 |
|----|------|--------|
| `data_migrations` | DataMigration | 数据迁移标记，避免同一回填跑两次 |

---

## 不建表的结构体

这些类型嵌在 JSON 列里，`AutoMigrate` 不包含它们：

| 类型 | 位置 | 用途 |
|------|------|------|
| `IssueSuggestion` / `SuggestionOutcome` | `comment.go` | 评论上的 Agent 建议与用户处置 |
| `FieldCondition` | `conditional_field.go` | 条件值的 JSON 形状 |
| `AgentInvocationTarget` | `agent.go` | Agent 的调用授权 |
| `AgentToolCall` | `agent.go` | 一次运行里的工具调用，存在 `agent_activities.tool_calls` |

---

## 🌐 语言

- **中文** (本文档)
- [English](https://vinthuy.github.io/reqmango/en/kb/architecture/data-model/)
