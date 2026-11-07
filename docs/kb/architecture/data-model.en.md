# Data Model

**Last updated**: 2026-11-07

The production schema is whatever `AutoMigrate` in `backend/cmd/server/main.go` creates. The 131 tables below are that list. Names come from GORM's naming strategy, including each model's `TableName()`. Structs in `internal/model` that exist only as JSON are not tables; they are listed at the end.

Column-level detail lives in the model file. This page does not copy column lists.

---

## Common fields

Models that embed `BaseModel` (`internal/model/base.go`) have:

| Field | Type | Notes |
|-------|------|-------|
| `id` | `uint64` | Autoincrement primary key |
| `created_at` / `updated_at` | `time.Time` | Maintained by GORM |
| `deleted_at` | `gorm.DeletedAt` | Soft delete, indexed |
| `created_by_id` / `updated_by_id` | `*uint64` | Nullable |

Primary keys are `uint64`, not UUIDs. The `depth` comments on Issue and Page cap the hierarchy at 5.

---

## Users and tenancy

| Table | Model | Stores |
|-------|-------|--------|
| `users` | User | Accounts |
| `workspaces` | Workspace | Workspaces |
| `workspace_members` | WorkspaceMember | Membership and role level |
| `projects` | Project | Projects |
| `project_members` | ProjectMember | Project membership |
| `project_subscribers` | ProjectSubscriber | Project subscribers |
| `personal_access_tokens` | PersonalAccessToken | Long-lived user tokens. Only the SHA-256 hash is stored; the plaintext is returned once at creation |

---

## Work items

| Table | Model | Stores |
|-------|-------|--------|
| `issues` | Issue | Work items (nullable `estimate_point_id` → `estimate_points`) |
| `issue_assignees` | IssueAssignee | Assignees |
| `issue_labels` | IssueLabel | Label links |
| `issue_cycles` | IssueCycle | Cycle links |
| `issue_pages` | IssuePage | Page links |
| `issue_activities` | IssueActivity | Activity log |
| `issue_watchers` | IssueWatcher | Watchers |
| `states` | State | States (project or workspace) |
| `labels` | Label | Labels |
| `comments` | Comment | Comments. Agent suggestions are JSON on the comment, not their own table |
| `attachments` | Attachment | Attachment metadata |
| `time_tracks` | TimeTrack | Time entries; `duration` in **seconds**; running (`ended_at` null) or completed |
| `recurrence_rules` | RecurrenceRule | Recurrence rules |
| `relation_types` | RelationType | Relation types (inward/outward names) |
| `issue_relations` | IssueRelation | Links between work items |
| `project_intake_settings` | ProjectIntakeSetting | Per-project intake form / webhook / email switches and token |

---

## Types, fields, templates

| Table | Model | Stores |
|-------|-------|--------|
| `issue_types` | IssueType | Work item types |
| `issue_type_fields` | IssueTypeField | Type-to-field bindings |
| `issue_type_imports` | IssueTypeImport | Type import records |
| `issue_type_templates` | IssueTypeTemplate | Workspace type blueprints |
| `issue_type_template_fields` | IssueTypeTemplateField | Blueprint fields |
| `custom_fields` | CustomField | Custom field definitions |
| `custom_field_options` | CustomFieldOption | Dropdown options |
| `issue_custom_field_values` | IssueCustomFieldValue | Values on a work item |
| `project_custom_field_enrollments` | ProjectCustomFieldEnrollment | Whether a project enables a field |
| `conditional_fields` | ConditionalField | Field visibility rules |
| `project_templates` | ProjectTemplate | Project templates |
| `project_template_types` | ProjectTemplateType | Types included in a template |
| `work_item_templates` | WorkItemTemplate | Templates for creating work items |

---

## Planning

| Table | Model | Stores |
|-------|-------|--------|
| `cycles` | Cycle | Cycles |
| `modules` | Module | Module tree |
| `module_issues` | ModuleIssue | Module-to-issue links |
| `module_inheritance_overrides` | ModuleInheritanceOverride | Module inheritance overrides |
| `releases` | Release | Releases |
| `release_issues` | ReleaseIssue | Release-to-issue links |
| `initiatives` | Initiative | Workspace-level goals |
| `initiative_projects` | InitiativeProject | Goal-to-project links |
| `estimate_points` | EstimatePoint | Point estimates |
| `estimate_categories` | EstimateCategory | Category estimates |
| `estimate_times` | EstimateTime | Time estimates |
| `project_estimate_settings` | ProjectEstimateSettings | Which estimate mode a project uses |

---

## State machines and automation

`workflows` is the work-item state machine. `state_transitions` belong only to it. Orchestration DAGs live on `agent_workflows` and the node / edge / run tables. The two no longer share a `workflow_kind` column.

| Table | Model | Stores |
|-------|-------|--------|
| `workflows` | Workflow | State machines |
| `state_transitions` | StateTransition | State transitions |
| `automation_rules` | AutomationRule | Automation rules |
| `automation_executions` | AutomationExecution | Rule executions |
| `automation_rule_overrides` | AutomationRuleOverride | Rule overrides |
| `approvals` | Approval | Approval requests |
| `approval_records` | ApprovalRecord | Approval actions |
| `agent_workflows` | AgentWorkflow | Multi-agent orchestration definitions |
| `workflow_nodes` | WorkflowNode | Orchestration nodes |
| `workflow_edges` | WorkflowEdge | Orchestration edges |
| `workflow_runs` | WorkflowRun | One orchestration run |
| `workflow_node_runs` | WorkflowNodeRun | A node execution inside a run |

---

## Pages, views, reports

| Table | Model | Stores |
|-------|-------|--------|
| `pages` | Page | Project documents |
| `page_templates` | PageTemplate | Page templates |
| `page_versions` | PageVersion | Page versions |
| `project_page_tabs` | ProjectPageTab | Project page tabs |
| `project_updates` | ProjectUpdate | Project status updates |
| `saved_views` | SavedView | Saved filter views |
| `saved_dashboards` | SavedDashboard | Dashboards |
| `dashboard_widgets` | DashboardWidget | Dashboard widgets |
| `saved_reports` | SavedReport | Saved reports |
| `search_templates` | SearchTemplate | Search templates |
| `metric_charts` | MetricChart | Metric charts |

---

## Permissions

| Table | Model | Stores |
|-------|-------|--------|
| `roles` | Role | Roles. Levels: Admin=20, Member=15, Guest=5 |
| `permissions` | Permission | Permission codes, `resource:action` |
| `role_permissions` | RolePermission | Role-to-permission links |
| `field_permissions` | FieldPermission | Field-level read/write |

Seed data is `internal/seed/seed_rbac.go`: 55 permissions and 3 default roles.

---

## Integrations and plugins

| Table | Model | Stores |
|-------|-------|--------|
| `webhooks` | Webhook | Outbound webhooks |
| `github_connections` | GitHubConnection | GitHub connections |
| `slack_connections` | SlackConnection | Slack connections |
| `git_integrations` | GitIntegration | Git integration config |
| `git_issue_links` | GitIssueLink | Work-item to pull-request links |
| `mcp_configs` | MCPConfig | MCP server config |
| `plugins` | Plugin | Plugins |
| `plugin_event_logs` | PluginEventLog | Plugin events |

---

## Notifications and chat

| Table | Model | Stores |
|-------|-------|--------|
| `notifications` | Notification | In-app notifications |
| `chats` | Chat | Conversations |
| `messages` | Message | Messages |
| `message_reactions` | MessageReaction | Message reactions |

---

## AI and agents

These tables are created by process startup. The `/agents*` console is hidden by default and Harness / Loop is not a product direction, but the tables are still migrated.

| Table | Model | Stores |
|-------|-------|--------|
| `ai_configs` | AIConfig | Workspace AI config |
| `ai_threads` | AIThread | AI conversation threads |
| `ai_messages` | AIMessage | Conversation messages |
| `agents` | Agent | Agents that can be assigned and @-mentioned |
| `agent_activities` | AgentActivity | Agent action audit |
| `agent_registry` | AgentEntry | Built-in agent registry (`internal/ai/registry`) |
| `project_agent_members` | ProjectAgentMember | Agent members of a project |
| `agent_templates` | AgentTemplate | Agent templates |
| `agent_configs` | AgentConfig | Agent configs |
| `agent_sessions` | AgentSession | Sessions |
| `agent_tasks` | AgentTask | Tasks |
| `agent_task_snapshots` | AgentTaskSnapshot | Task snapshots |
| `task_logs` | TaskLog | Task logs |
| `agent_decisions` | AgentDecision | Decision records |
| `agent_cost_budgets` | AgentCostBudget | Cost budgets |
| `agent_slas` | AgentSLA | SLAs |
| `runtimes` | Runtime | Runtimes |
| `skills` | Skill | Skills |
| `skill_execution_logs` | SkillExecutionLog | Skill executions |
| `tools` | Tool | Tools |
| `tool_permissions` | ToolPermission | Tool permissions |
| `tool_call_logs` | ToolCallLog | Tool calls |
| `memory_entries` | MemoryEntry | Memory entries |
| `memory_sessions` | MemorySession | Memory sessions |
| `memory_vector_indexes` | MemoryVectorIndex | Memory vector indexes |
| `squads` | Squad | Squads |
| `squad_members` | SquadMember | Squad members |
| `squad_tasks` | SquadTask | Squad tasks |
| `squad_executions` | SquadExecution | Squad executions |
| `autopilot_tasks` | AutopilotTask | Autopilot tasks |
| `autopilot_executions` | AutopilotExecution | Autopilot executions |
| `autopilot_webhooks` | AutopilotWebhook | Autopilot webhooks |
| `agent_loops` | Loop | Loop definitions |
| `agent_loop_runs` | LoopRun | Loop runs |
| `agent_loop_iterations` | LoopIteration | Loop iterations |
| `agent_pipelines` | Pipeline | Pipelines |
| `agent_pipeline_runs` | PipelineRun | Pipeline runs |
| `developer_jobs` | DeveloperJob | Developer-agent jobs |
| `tester_jobs` | TesterJob | Tester-agent jobs |
| `cicd_configs` | CICDConfig | CI/CD config |
| `build_records` | BuildRecord | Build records |
| `sdlc_workflows` | SDLCWorkflow | SDLC workflows |
| `sdlc_stages` | SDLCStage | SDLC stages |

---

## Other

| Table | Model | Stores |
|-------|-------|--------|
| `data_migrations` | DataMigration | Markers so a data backfill runs once |

---

## Structs that are not tables

These types are JSON shapes. `AutoMigrate` does not include them:

| Type | File | Use |
|------|------|-----|
| `IssueSuggestion` / `SuggestionOutcome` | `comment.go` | Agent suggestions on a comment, and what the user did with them |
| `FieldCondition` | `conditional_field.go` | JSON shape of a condition value |
| `AgentInvocationTarget` | `agent.go` | Who may invoke an agent |
| `AgentToolCall` | `agent.go` | One tool call inside a run, stored on `agent_activities.tool_calls` |

---

## 🌐 Language

- **English** (this document)
- [中文文档](https://vinthuy.github.io/reqmango/kb/architecture/data-model/)
