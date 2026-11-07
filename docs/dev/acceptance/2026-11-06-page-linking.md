# 工作项 ↔ 页面互链 — 验收记录（2026-11-06）

**SRS：** [docs/specs/2026-11-06-page-linking.md](../../specs/2026-11-06-page-linking.md)（状态仍为 **已冻结**；完成人工全量验收前不改为「已实现」。）

## 自动化运行记录

| 命令 | 结果 |
|------|------|
| `cd backend; go test ./internal/service/ -run IssuePages -count=1` | **PASS**（幂等插入、跨项目拒绝、重复删除返回 404） |
| `cd frontend; npx vitest run src/__tests__/IssueLinkedPagesCard.spec.ts` | **PASS**（只读隐藏变更、编辑可见、跳转带 `?page=`） |

说明：本机 PowerShell 用户 profile 可能对终端输出前置一段与 `System.Text.ASCIIEncoding` 相关的报错，**不影响** `go test` / Vitest 的退出码与测试结果。

## 第 6 节 验收清单（结果 + 验证方式）

| AC-ID | 场景摘要 | 优先级 | 结果 | 验证方式 |
|-------|----------|--------|------|----------|
| AC-01.1 | 空状态 | P0 | **已通过（自动化 UI）** | `IssueLinkedPagesCard.spec.ts`：无 `issue:edit` 时无添加；`data-test="linked-pages-empty"` |
| AC-02.1 | 添加关联 | P0 | **部分自动化 + 待人工** | 路由 `POST` 仍需浏览器选页并确认列表与 `GET /issues/:id/pages` |
| AC-02.2 | 重复不双行 | P0 | **已通过（自动化服务）** | `issue_pages_test.go`：`TestIssuePages_AddPage_IdempotentAndSameProject` |
| AC-03.1 | 解除关联 | P0 | **部分自动化 + 待人工** | `RemovePage` 未链接返回 404；浏览器点「解除关联」后列表与 API |
| AC-04.1 | 跳转打开页面 | P0 | **已通过（自动化 UI）** | `IssueLinkedPagesCard.spec.ts`：`router.push` → `/workspace/:slug/project/:id/pages?page=` |
| AC-05.1 | 只读无编辑入口 | P0 | **已通过（自动化 UI）** | 无 `issue:edit`：`linked-pages-add-toggle`、`linked-pages-unlink` 不出现 |

## 小结

- **后端**：`AddPage` 使用 `ON CONFLICT DO NOTHING` 做幂等；同项目校验；`ListPages` 经 `issue_pages` 联结并限定 `pages.project_id`。
- **前端**：关联页签下「关联文档」卡片；变更需 `issue:edit`；深链沿用 `ProjectPages` 的 `?page=`。
- **待人工**：AC-02.1、AC-03.1 的完整浏览器路径与网络面板确认。
