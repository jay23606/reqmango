# 自定义字段条件显隐 — 验收记录（2026-11-06）

**SRS：** [docs/specs/2026-11-06-conditional-fields.md](../../specs/2026-11-06-conditional-fields.md)（状态仍为 **已冻结**；完整人工 / 端到端核对前不改为「已实现」。）

**分支：** `feat/mandatory-fields`

## 自动化运行记录

| 命令 | 结果 |
|------|------|
| `cd backend; go test ./internal/service/ -count=1` | **PASS**（约 2.2s，`ok github.com/reqmango/backend/internal/service`） |
| `cd frontend; npm test -- --run src/__tests__/IssuePropertySidebar.spec.ts` | **PASS**（Vitest 3.2.7，`IssuePropertySidebar.spec.ts`：**11 tests passed**） |

说明：本机 PowerShell 配置可能对命令输出前置一段 profile 报错，**不影响** `go test` / Vitest 退出码与测试结果。

### 接口说明（与 SRS §7）

- SRS 仍写「`/projects/:projectId/conditional-fields`」；**实现与客户端**以工作空间为界：`/v1/conditional-fields`（及 `POST /v1/conditional-fields/evaluate`）。正式 API/手工验收时以 handler 与 OpenAPI 为准。

## §6 验收清单（结果 + 验证方式）

| AC-ID | 场景摘要 | 优先级 | 结果 | 验证方式 |
|-------|----------|--------|------|----------|
| AC-01.1 | 可创建并读回规则 | P0 | **已通过（自动化）** | `internal/service/conditional_field_service_test.go`：`TestConditionalField_CreateAndList`（创建后 `List` 读回且 `is_enabled`） |
| AC-02.1 | 评估结果隐藏目标字段 | P0 | **已通过（自动化）** | `TestEvaluateFieldVisibility_HidesMatchedField`（Done → id 不在可见集；Backlog → 在可见集） |
| AC-03.1 | UI 不展示隐藏字段 | P0 | **已通过（自动化，详情侧栏）；创建页待人工** | 详情：`IssuePropertySidebar.spec.ts`（`visibleFieldIds` 排除的字段不渲染、可见才标 `*`）。创建页 `IssueCreate` 仍以浏览器核对显隐与评估调用为准。 |
| AC-04.1 | 隐藏必填不拦提交 | P0 | **已通过（自动化）** | `issue_mandatory_fields_test.go`：`TestSetIssueValue_AllowsEmptyWhenRequiredButHiddenByState`、`TestDeleteIssueValue_AllowsWhenRequiredButHiddenByState`、`TestBulkSetIssueValues_AllowsEmptyWhenRequiredButHiddenByState`；`TestCreate_SkipsRequiredWhenHiddenByState`（创建不落库） |
| AC-04.2 | 可见必填仍强制 | P0 | **已通过（自动化）** | `TestCreate_StillRequiresWhenVisible`、`TestSetIssueValue_RejectsEmptyRequired`（等业务拒绝文案含字段名） |
| AC-05.1 | 关规则后恢复可见 | P0 | **已通过（自动化）** | `TestEvaluateFieldVisibility_DisabledRuleShowsField`（`disabled` / `deleted` 子用例，断言含 AC-05.1） |

## 小结

- 后端 **`internal/service`** 与聚焦 Vitest **`IssuePropertySidebar`** 在本次运行中 **均通过**。
- **SRS §6 仍未全部视为人工闭环**（尤其创建页 UI 与线上 API 路径核对），故 **SRS 文档状态保持「已冻结」**。
