# 通知自动送达 — 验收记录（2026-11-06）

**SRS：** [docs/specs/2026-11-06-notification-auto-trigger.md](../../specs/2026-11-06-notification-auto-trigger.md)

**分支：** `feat/mandatory-fields`（工作树 `mandatory-fields`）

## 自动化运行记录

| 命令 | 结果 |
|------|------|
| `cd backend && go test ./internal/service/ -count=1 -short` | **PASS**（`ok github.com/reqmango/backend/internal/service`） |

说明：本机 PowerShell profile 可能对 `go test` 输出前置一段报错，**不影响** `go test` 退出码与测试结果。

## SRS §6 验收清单（结果 + 验证方式）

| AC-ID | 场景摘要 | 优先级 | 结果 | 验证方式 |
|-------|----------|--------|------|----------|
| AC-01.1 | 指派通知 | P0 | **已通过（自动化）** | `notification_auto_trigger_test.go`：`TestCreate_SendsAssignmentToAssigneesNotActor`、`TestUpdate_AssigneeDiffSendsAssignmentNotifications` |
| AC-02.1 | 评论通知 | P0 | **已通过（自动化）** | `TestComment_NotifiesAssigneesAndWatchers` |
| AC-03.1 | 状态变更通知 | P0 | **已通过（自动化）** | `TestUpdate_StateChangeNotifiesAssigneeAfterAssigneeRefresh`（`Update` 在同一次事务中先刷新经办人再发 `issue_state_changed`，覆盖与指派同请求的时序） |
| AC-04.1 | 提及通知 | P0 | **已通过（自动化）** | `TestComment_MentionNotifiesUser`（DB type=`assignment`，与 SRS 表一致） |
| AC-05.1 | 不自通知 | P0 | **已通过（自动化）** | `TestNotification_TriggerSkipsSelfRecipient`；`TriggerNotification`/`TriggerNotificationsBulk` 与审批请求人过滤 |
| AC-06.1 | 未读可见 | P0 | **已通过（自动化）** | `TestNotification_GetSummaryUnread` |
| AC-07.1 | 到期提醒 | P1 | **未纳入本轮** | 按计划跳过非平凡 E2E |

## 小结

- P0 自动触发与不自通知行为由 `internal/service` 包测试锁定；**P1 `issue_due_soon` 未做 E2E**。
