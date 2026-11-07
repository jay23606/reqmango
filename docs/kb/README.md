# Knowledge Base（全量知识库）

KB 是 reqmango 系统的**唯一真相来源**，始终描述系统当前的实际状态。

**最后更新**: 2026-11-07

**产品方向**: 原生项目管理 + 原生 AI 项目管理路径 AI（Intake / Analyze / 自动化 / 仪表盘摘要）。  
**核心门禁**: 自定义工作项类型 · 自定义字段 · 自定义工作流 · 自动化规则。  
**废止**: Harness / Loop / 独立 Agent 平台 Agent 控制台产品化 → [superseded/agent-platform](../superseded/agent-platform/README.md)

---

## 🌐 语言

- **中文** (本文档)
- [English](README.en.md)

## 维护原则

- **时效性**：当功能实现完成，对应的 KB 文档必须同步更新
- **准确性**：KB 描述的是代码实际做了什么，不是设计时打算做什么
- **简洁性**：KB 是参考文档，不是开发日志。保留核心信息，去除过程性内容

## KB 更新时机

| 触发事件 | 更新范围 |
|----------|----------|
| 新功能完成 | 更新 `architecture/README.md` 模块表 + `data-model.md` 表清单 + `frontend.md` |
| API 变更 | 更新 `api-conventions.md` |
| 数据模型变更 | 更新 `data-model.md` |
| 技术栈升级 | 更新 `architecture/README.md` |

## KB 文档清单

### 产品层
- [PRD.md](PRD.md) — 产品需求文档

### 架构层
- [architecture/README.md](architecture/README.md) — 架构总览 + 模块状态
- [architecture/project-layout.md](architecture/project-layout.md) — 项目目录结构
- [architecture/backend-go.md](architecture/backend-go.md) — Go 后端架构
- [architecture/frontend.md](architecture/frontend.md) — 前端架构（视图 / 组件 / API 模块的组织方式）
- [architecture/data-model.md](architecture/data-model.md) — 数据模型（AutoMigrate 的 131 张表）
- [architecture/api-conventions.md](architecture/api-conventions.md) — API 设计约定
- [architecture/tech-stack.md](architecture/tech-stack.md) — 技术栈详情
- E2E 测试 — Playwright 场景套件，位于 `frontend/e2e/`，覆盖主路径与核心定制

### 功能设计文档
- [architecture/saved-views-design.md](architecture/saved-views-design.md) — 保存视图
- [architecture/pages-design.md](architecture/pages-design.md) — 页面文档/Wiki
- [architecture/notification-design.md](architecture/notification-design.md) — 通知系统
- [architecture/type-hierarchy-template-design.md](architecture/type-hierarchy-template-design.md) — 类型层级与模板
- [architecture/relation-system-design.md](architecture/relation-system-design.md) — 关联类型系统

### 变更日志
- [changelog/README.md](changelog/README.md) — KB 更新历史
