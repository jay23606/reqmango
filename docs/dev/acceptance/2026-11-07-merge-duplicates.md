# 合并重复工作项 — 验收记录（2026-11-07）

**SRS：** [docs/specs/2026-11-07-merge-duplicates.md](../../specs/2026-11-07-merge-duplicates.md)（状态仍为 **已冻结**。）

**分支：** `feat/merge-duplicates`

## 自动化运行记录

| 命令 | 结果 |
|------|------|
| `cd backend; go test ./internal/service/ -run TestMergeDuplicates -count=1` | **PASS** |
| `cd frontend; npm test -- --run src/api/issue.api.test.ts src/__tests__/MergeIntoIssueDialog.spec.ts src/__tests__/IssueDetailHeader.spec.ts` | **PASS** |

说明：本机 PowerShell 配置可能对命令输出前置一段 profile 报错，**不影响** `go test` / Vitest 退出码与测试结果。

## §6 验收清单（结果 + 验证方式）

| AC-ID | 场景摘要 | 优先级 | 结果 | 验证方式 |
|-------|----------|--------|------|----------|
| AC-01.1 | 成功合并 | P0 | **已通过（自动化，部分）** | 后端 `issue_merge_test.go`：`TestMergeDuplicates_SoftMergesSameProject`；前端 Vitest：`MergeIntoIssueDialog` 确认步、`mergeIssues` POST `/issues/merge`。UI 全链路待浏览器/E2E。 |
| AC-02.1 | 跨项目拒绝 | P0 | **已通过（自动化）** | `TestMergeDuplicates_RejectsCrossProject` |
| AC-03.1 | 无权限 | P0 | **待人工** | 需权限矩阵下对源/目标无 `issue:edit` 时 API 403；本轮未加 handler 级单测。 |
| AC-04.1 | 自合并拒绝 | P0 | **已通过（自动化）** | `TestMergeDuplicates_RejectsSelf`；前端 `MergeIntoIssueDialog` 预填源 id 时不进入可确认态。 |

## 小结

- 后端软合并语义与合并 API 单测 **已通过**。
- 前端：工作项详情「合并到…」、创建页重复建议「创建并合并到该项」、i18n 与 Vitest **已通过**。
- **SRS §6 仍不能标为全部已验收**（无权限、浏览器端到端合并闭环待人工/E2E），故 **SRS 文档状态保持「已冻结」**。
