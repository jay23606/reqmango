# 列表 / 看板快速新建 — 验收记录（2026-11-06）

**SRS：** [docs/specs/2026-11-06-quick-create.md](../../specs/2026-11-06-quick-create.md)（状态仍为 **已冻结**；完整列表 / 看板人工或 E2E 闭环前不改为「已实现」。）

**分支：** `feat/mandatory-fields`

## 自动化运行记录

| 命令 | 结果 |
|------|------|
| `cd backend; go test ./internal/service/ -run TestCreate_OmittedTypeID_UsesDefaultForMandatoryCFCheck -count=1` | **PASS**（约 0.33s，`ok github.com/reqmango/backend/internal/service`） |
| `cd frontend; npm test -- --run src/__tests__/QuickCreateInput.spec.ts` | **PASS**（Vitest 3.2.7，`QuickCreateInput.spec.ts`：**3 tests passed**） |

说明：本机 PowerShell 配置可能对命令输出前置一段 profile 报错，**不影响** `go test` / Vitest 退出码与测试结果。

## §6 验收清单（结果 + 验证方式）

| AC-ID | 场景摘要 | 优先级 | 结果 | 验证方式 |
|-------|----------|--------|------|----------|
| AC-01.1 | 列表创建成功并留在本页 | P0 | **待人工 / E2E** | 浏览器在列表顶栏快速新建：断言新项出现、未跳转、输入框可继续录入；本轮无专属 Vitest。 |
| AC-01.2 | 空标题不创建 | P0 | **已通过（自动化）** | `QuickCreateInput.spec.ts`：`shows validation and does not call create when title is empty`（toast `quickCreate.titleRequired`、`data-test="quick-create-title-error"`，且未调用 `createIssue`）。 |
| AC-02.1 | 看板列内创建带状态 | P0 | **待人工 / E2E** | 看板列「+」路径与 `state_id` 落库以浏览器或集成测试为准；本轮未跑。 |
| AC-04.1 | 无权限隐藏入口 | P0 | **待人工** | 无 `issue:create` 时不展示快速新建入口，需权限矩阵下页面核对。 |
| AC-06.1 | 必填字段不可绕过 | P0 | **已通过（自动化，部分）** | 后端：`issue_mandatory_fields_test.go`：`TestCreate_OmittedTypeID_UsesDefaultForMandatoryCFCheck`（省略 `type_id` 时用默认类型做必填校验 → 4xx 含字段名、无 Issue/CF 落库）。前端：`QuickCreateInput.spec.ts`：`toasts server message and shows full-form link on 4xx`（toast 服务端文案、内联错误、`RouterLink` 指向完整表单且保留标题与 `type_id` / `state_id` query）。 |

## 小结

- 本轮聚焦 **`TestCreate_OmittedTypeID_UsesDefaultForMandatoryCFCheck`** 与 **`QuickCreateInput`** Vitest **均通过**。
- **SRS §6 仍不能标为全部已验收**（列表成功流、看板列内、无权限入口等依赖人工或 E2E），故 **SRS 文档状态保持「已冻结」**。
