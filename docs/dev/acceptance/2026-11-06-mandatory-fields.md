# 自定义字段必填 — 验收记录（2026-11-06）

**SRS：** [docs/specs/2026-11-06-mandatory-fields.md](../../specs/2026-11-06-mandatory-fields.md)（状态仍为 **已冻结**；完成人工 / API 全量验收前不改为「已实现」。）

**分支：** `feat/mandatory-fields`

## 自动化运行记录

| 命令 | 结果 |
|------|------|
| `cd backend && go test ./internal/service/ -count=1` | **PASS**（约 1.9s，`ok github.com/reqmango/backend/internal/service`） |

说明：本机 PowerShell 配置可能对 `go test` 输出前置一段 profile 报错，**不影响** `go test` 退出码与测试结果。

### 导入相关 Postgres 测试

- `issue_import_test.go` 中带 `importTestDB` 的用例在 **未设置 `TEST_DATABASE_URL`** 时会 **Skip**（提示：`set TEST_DATABASE_URL to run issue import tests`）。
- 在本轮默认环境下，**AC-04.2 对应的导入用例未实际执行**（与 Skip 行为一致）。

### 已知限制（导入）

- **导入尚无「自定义字段列 → 字段 ID」的 CSV/JSON 映射**；正式导入路径上的必填校验已在服务层实现，但 **无法通过导入文件单列映射填充指定 CF** 来做端到端 CSV 验收。需在具备 `TEST_DATABASE_URL` 的环境结合 API / 手工数据通路验证。

## §6 验收清单（结果 + 验证方式）

| AC-ID | 场景摘要 | 优先级 | 结果 | 验证方式 |
|-------|----------|--------|------|----------|
| AC-02.1 | 创建缺必填拒绝 | P0 | **已通过（自动化）** | `internal/service/issue_mandatory_fields_test.go`：`TestCreate_RejectsMissingRequiredCustomField`（缺失 / 空字符串 → 4xx 文案含字段名，且无落库） |
| AC-02.2 | 创建含必填成功 | P0 | **已通过（自动化）** | `TestCreate_WithRequiredCustomFieldValue_Succeeds` |
| AC-03.1 | 更新清空必填拒绝 | P0 | **已通过（自动化，服务层）** | `TestSetIssue_RejectsEmptyRequired`、`TestDeleteIssueValue_RejectsRequired`、`TestBulkSetIssueValues_RejectsEmptyRequired`（清空 / 删除 / 批量设空 → 拒绝且原值保留） |
| AC-04.1 | 转换类型缺必填拒绝 | P0 | **已通过（自动化）** | `TestConvertType_RejectsWhenTargetRequiresMissingCF`（类型未变更） |
| AC-04.2 | 正式导入缺必填不落脏数据 | P0 | **待验证（需 Postgres）** | 用例 `TestImportFormalWriteRejectsMissingMandatoryCustomField` 等 **需 `TEST_DATABASE_URL`**；默认 CI/本地未配则 **Skip**。另见上文「无 CF 列映射」限制。 |
| AC-05.1 | 创建页必填 UI | P0 | **待人工** | 可选：曾启动全量 `vitest run`，因 PowerShell 将子进程 stderr 记为错误且运行未完整收敛，**未采纳为 AC-05.1 证据**；需浏览器或针对性组件/ E2E 验证创建页 `*` 与提交前提示（SRS AC-05.1）。 |

## 小结

- 后端 **`internal/service`** 包测试在本次运行中 **全部通过**。
- **完整 SRS §6「结果」列仍不能标为全部已验收**：导入 Postgres 用例未跑、AC-05.1 需人工或前端测试；故 **SRS 文档状态保持「已冻结」**。
