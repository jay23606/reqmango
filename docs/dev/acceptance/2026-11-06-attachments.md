# 工作项附件 — 验收记录（2026-11-06）

**SRS：** [docs/specs/2026-11-06-attachments.md](../../specs/2026-11-06-attachments.md)（状态 **已冻结**）

**分支：** `feat/mandatory-fields`（本 worktree）

## 自动化运行记录

| 命令 | 结果 |
|------|------|
| `cd backend && go test ./internal/handler/ -count=1 -run TestAttachmentHandler -timeout 2m` | **PASS**（约 &lt;1s，`ok github.com/reqmango/backend/internal/handler`） |
| `cd backend && go build ./...` | **PASS** |
| `cd frontend && npm test -- --run src/__tests__/IssueTabAttachments.spec.ts` | **PASS**（Vitest，`2 passed`） |

说明：本机 PowerShell profile 可能在对 `go` / `npm` 输出前打印一段无关报错，**不影响**命令退出码与测试结果。

### 已知测试取舍

- **流式超大请求（MaxBytesReader → 413）**：未在自动化中构造完整 &gt;10MB multipart 体（执行时间过长、内存压力大）。生产路由上 `Create` 已对 `Request.Body` 使用 `http.MaxBytesReader`；**FR-04** 在测试中通过 **multipart 内声明超过 10MB 的实体**（`validateFile` → 400，文案含 `10MB`）覆盖；UI 侧对超过 10MB 的文件在提交前提示并拦截。

## §6 验收清单（结果 + 验证方式）

| AC-ID | 场景摘要 | 优先级 | 结果 | 验证方式 |
|-------|----------|--------|------|----------|
| AC-01.1 | 上传成功 | P0 | **待 E2E** | 需浏览器/API：有 `issue:edit` 时 POST `/issues/:id/attachments` + GET 列表 |
| AC-02.1 | 下载一致 | P0 | **待 E2E** | 下载字节与上传文件比对 |
| AC-03.1 | 删除后不可用 | P0 | **已通过（自动化）** | `TestAttachmentHandler_DeleteThenDownload404`：删除后 GET download → **404** |
| AC-04.1 | 超 10MB 拒绝 | P0 | **已通过（自动化，校验路径）** | `TestAttachmentHandler_UploadOver10MBRejectedByValidator`：正文 &gt;10MB → **400**，`error` 含 `10MB`；UI `AttachmentManager` 客户端预检 + toast |
| AC-05.1 | 只读权限边界 | P0 | **已通过（混合）** | 后端：`TestAttachmentHandler_PostAndDeleteRequireIssueEdit`（无 `issue:edit` → POST/DELETE **403**）；`TestAttachmentHandler_ListByIssue_AllowsGuest`（GET 列表 **200**）。前端：`IssueTabAttachments` 用 `useProjectPermissions` 设 `readOnly`，隐藏上传区/删除 |
| AC-06.1 | 计数一致 | P0 | **已通过（后端 + 前端）** | `BuildIssueResponse` 对 `attachments` 表 `COUNT` 填充 `attachment_count`；详情页签监听 `@attachment-count-change` 与列表同步 |

## 小结

- Handler 层已用 **Gin 集成测试**钉住删除后下载 404、10MB 拒绝文案、以及 `issue:edit` 对 POST/DELETE 的门禁。
- Issue API **`attachment_count`** 与数据库附件数量一致；附件页签计数随本页列表 `countChange` 更新。
- **AC-01.1 / AC-02.1** 仍以端到端或手工确认为准。
