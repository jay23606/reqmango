# Pipeline Status（功能管线状态）

最后更新：2026-11-07

---

## 活跃功能

| 功能 | Spec | Design | Plan | Implement | 备注 |
|------|------|--------|------|-----------|------|
| 存量质量 | — | — | — | 🔄 | 主路径已 PASS；不新开大功能。租户隔离与路由鉴权由 [route audit](../security.md#8-tenancy-guard-and-the-route-audit-backendcmdroute_audit) 的 CI 门禁持续验证：四模式均以 `-strict` 跑在全新 `AutoMigrate` 库上，无 XTENANT / ESCALATE / LEAK / WRITE / unresolved |
| （无新开功能） | | | | | |

### 2026-10-18 运维与质量补强（已完成）

| 项 | 内容 | 文档 |
|----|------|------|
| 健康探针 | `/health`（无依赖存活）、`/readyz`（2s 内 DB ping），注册在限流与语言中间件之前；Dockerfile `HEALTHCHECK` 与 CI/E2E 启动等待均改用探针 | [security.md §9](../security.md#9-health-and-readiness-probes) |
| 审计门禁 | 新增 `-strict`，CI job `Tenancy audit` 四模式逐份新库运行，任何越权/未覆盖/500 直接失败 | [security.md §8](../security.md#8-tenancy-guard-and-the-route-audit-backendcmdroute_audit) |
| 请求指标 | 按路由模板聚合请求量/4xx/5xx/P95，超阈值慢请求打 `[SLOW]` 日志；`GET /api/v1/metrics` 仅超管 | [security.md §10](../security.md#10-request-metrics-and-slow-request-logging) |
| 工作流转换 | 一张表一个属主：`state_transitions` 只属于 `workflows`（`StateWorkflowService`），编排归 `agent_workflows`（`WorkflowService`）；退役 `workflow_kind` 并恢复外键 | [security.md §11](../security.md#11-workflow-ownership-one-table-per-domain) · [spec](../superpowers/specs/2026-10-18-workflow-model-separation-design.md) |
| 全新库缺陷 | 补 6 个缺失 AutoMigrate 模型、autopilot 跨租户读取、报表空 `IN ()`、initiative 歧义列 | [security.md §12](../security.md#12-gaps-the-fresh-database-audit-found) |

### 2026-10-24 开源就绪（L1–L4，已完成）

| 层 | 内容 | 文档 |
|----|------|------|
| L1 安全 | 演示 / E2E 种子账号改为显式开关，默认不存在；生产清单列明必须关闭的项 | [security.md](../security.md) · [production](../deploy/production.md) |
| L2 发布 | 版本化构建、多架构 `ghcr.io` 镜像、tag 驱动的 Release 工作流；已用一次性 rc tag 端到端预演过一次（amd64+arm64、provenance/SBOM、prerelease 标记），预演流程见 release 文档 | [release](release.md#rehearsing-the-workflow) · [CHANGELOG](../../CHANGELOG.md) |
| L3 社区 | 行为准则、支持指南、Dependabot（限流）、公开 ROADMAP、受保护的 `master`（9 项必需检查） | [CONTRIBUTING](../../CONTRIBUTING.md) · [ROADMAP](../../ROADMAP.md) |
| L4 文档与语言 | MkDocs 站点（中英双语、`--strict` 构建）发布到 GitHub Pages；前端首访按浏览器语言协商，登录 / 注册页可见切换器 | [docs 站](https://vinthuy.github.io/reqmango/) · [configuration](../configuration.md) |

---

## 已完成（近期）

| 功能 | 状态 | 文档 |
|------|------|------|
| 工时记录与估点 P0 | ✅ | [SRS](../specs/2026-11-07-time-tracking-estimates.md) · [acceptance](acceptance/2026-11-07-time-tracking-estimates.md) · PR #34 |
| 合并重复工作项（软合并） | ✅ | [SRS](../specs/2026-11-07-merge-duplicates.md) · PR #33 |
| 产品化主路径（原生 PM + 核心定制门禁） | ✅ PASS | [acceptance](acceptance/2026-09-25-product-core-qa.md) §H |
| AI 项目管理（原生 AI 项目管理路径 P1–2+B1–C2） | ✅ PASS | [redesign](../superpowers/specs/2026-09-25-ai-project-management-redesign.md) · [preflight](acceptance/2026-09-25-release-preflight.md) |
| 工作项导入预校验与字段映射（Plane 差距 Task 3） | ✅ PASS | [design](../superpowers/specs/2026-10-24-issue-import-preview-design.md) · [plan](../superpowers/plans/2026-10-24-issue-import-preview.md) |

---

## 已走完管线并归档（早期）

| 功能 | 备注 |
|------|------|
| Cycle / Module | 见历史提交与 KB 模块表 |

---

## 废止 / 降级（勿再排期）

| 文档 | 状态 |
|------|------|
| 独立 Agent 平台 / 全 SDLC Agent PRD | **已归档** → [superseded/agent-platform](../superseded/agent-platform/README.md) |
| Harness / Loop 设计与 Phase 计划 | **已归档** → 同上 |
| Agent-Project Integration PRD/ARCH | **已归档** → 同上 |
| MCP/CLI 作为增长主路径 | 非当前 AI 产品排期 |

原路径保留 stub 重定向，避免外链断裂。

---

## 图例

| 标记 | 含义 |
|------|------|
| ✅ | 完成 |
| 🔄 | 进行中 |
| ❌ | 取消 / 废止 |
| - | 不适用 |
