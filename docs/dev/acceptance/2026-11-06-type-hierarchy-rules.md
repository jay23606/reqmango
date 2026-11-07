# 类型层级规则 — 验收记录（2026-11-06）

**SRS：** [docs/specs/2026-11-06-type-hierarchy-rules.md](../../specs/2026-11-06-type-hierarchy-rules.md)（状态仍为 **已冻结**；完成人工 / UI 全量验收前不改为「已实现」。）

## 自动化运行记录

| 命令 | 结果 |
|------|------|
| `cd backend && go test ./internal/service/ -run IssueHierarchy -count=1` | **PASS**（约 0.57s，`ok github.com/reqmango/backend/internal/service`） |
| `cd frontend && npx vue-tsc --noEmit` | **PASS**（无类型错误） |
| `cd frontend && npm run build`（含 `vite build`） | **失败（环境）**：esbuild 临时目录 `Access is denied`；与本次 Vue 改动无关时可用管理员权限或清理 Temp 后重试 |

说明：本机 PowerShell 配置可能对 `go test` 输出前置一段 profile 报错，**不影响** `go test` 退出码与测试结果。

## §6 验收清单（结果 + 验证方式）

| AC-ID | 场景摘要 | 优先级 | 结果 | 验证方式 |
|-------|----------|--------|------|----------|
| AC-01.1 | depth 上限 | P0 | **已通过（自动化）** | `issue_hierarchy_rules_test.go`：`TestIssueHierarchy_AC01_1_MaxDepthExceeded`（文案含 `Maximum hierarchy depth`，且无新子项） |
| AC-02.1 | 根类型不作子 | P0 | **已通过（自动化）** | `TestIssueHierarchy_AC02_1_RootTypeCannotHaveParent`（文案含 `root-level type cannot have a parent`） |
| AC-03.1 | 子类型 level 不得更浅 | P0 | **已通过（自动化）** | `TestIssueHierarchy_AC03_1_ChildTypeLevelNotShallowerThanParent`（文案含 `child type level must not be shallower`） |
| AC-04.1 | allowed_child 名单 | P0 | **已通过（自动化）** | `TestIssueHierarchy_AC04_1_AllowedChildWhitelist`（文案含 `not allowed under the parent type`） |
| AC-04.2 | parent_type_id 匹配 | P0 | **已通过（自动化）** | `TestIssueHierarchy_AC04_2_ParentTypeIDMustMatchParentIssueType`（文案含 `parent type does not allow this child type`） |
| AC-05.1 | 配置可保存读回 | P0 | **待人工（管理 UI）** | 工作空间类型：`WorkspaceIssueTypeManager.vue`；项目类型：`IssueTypeList.vue`。通过 `level` / `parent_type_id` / `allowed_child_type_ids` 保存后，列表刷新或重新打开编辑应显示相同配置；并以浏览器 / API `GET` 核对。 |

## 小结

- 层级与类型约束的 **服务层自动化** 与 SRS §6 AC-01.1–AC-04.2 对齐；**AC-05.1** 需人工在类型管理界面或 API 层确认读写一致，**SRS 状态保持「已冻结」** 直至该条完成。
