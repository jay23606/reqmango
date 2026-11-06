# 子项面板 — 验收记录（2026-11-06）

**SRS：** [docs/specs/2026-11-06-sub-items-panel.md](../../specs/2026-11-06-sub-items-panel.md)（状态仍为 **已冻结**；完成人工全量验收前不改为「已实现」。）

## 自动化运行记录

| 命令 | 结果 |
|------|------|
| `cd backend; go test ./internal/service/ -run "TestIssue(SubItems|Hierarchy)" -count=1` | **PASS**（覆盖子项 `Children` 列表 + 既有层级失败用例） |
| `cd frontend; npx vitest run src/__tests__/RelationSubIssuesCard.spec.ts src/__tests__/IssueTabRelations.spec.ts` | **PASS**（空状态 / 计数 / 层级错误条与权限 mock） |

说明：本机 PowerShell 用户 profile 可能对终端输出前置一段与 `System.Text.ASCIIEncoding` 相关的报错，**不影响** `go test` / Vitest 的退出码与测试结果。

## 第 6 节 验收清单（结果 + 验证方式）

| AC-ID | 场景摘要 | 优先级 | 结果 | 验证方式 |
|-------|----------|--------|------|----------|
| AC-01.1 | 空状态与 CTA | P0 | **已通过（自动化 UI）** | `RelationSubIssuesCard.spec.ts`：`subIssue.emptyHint`、空态 CTA `data-test="empty-quick-create-subissue"` / `empty-add-existing-subissue` |
| AC-02.1 | 快速新建子项 | P0 | **部分自动化 + 待人工** | 组件层已有 quick-create 交互测试；端到端需在详情「关联」页签确认创建后列表与 `parent_id` |
| AC-03.1 | 添加已有子项 | P0 | **待人工** | `IssueTabRelations`：`updateIssue` 失败时 toast + `hierarchyError` 行内提示；浏览器验证选择器成功后列表与 `parent_id` |
| AC-04.1 | 深度非法失败 | P0 | **已通过（自动化服务）** | `issue_hierarchy_rules_test.go`：`TestIssueHierarchy_AC01_1_MaxDepthExceeded`；子项列表面板：`TestIssueSubItems_MaxDepthFailureLeavesChildrenListUnchanged` |
| AC-05.1 | 计数一致 | P0 | **已通过（自动化 UI）+ 详情页签** | `RelationSubIssuesCard`：`subissue-count-badge` 与列表长度；`IssueDetail.vue`：关联页签 badge = 关系数 + 直接子项数 |
| AC-06.1 | 重排序持久化 | P1 | **未测（SRS P1）** | 若有 `reorder-sub-issues` UI 再补 |

## 小结

- 服务层已用 `Create` + `Children` 与深度失败用例钉住 **数据不变 / 合法子项出现**；UI 侧已加强 **空状态、层级错误可见性、计数与列表对齐**。
- **AC-02.1 / AC-03.1** 的浏览器级流程仍为 **待人工**，**SRS 状态保持「已冻结」** 直至上述条目完成。
