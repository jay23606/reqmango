# 页面 ↔ 工作项互链 — 需求规格说明书（SRS）

| 项 | 值 |
|----|-----|
| 状态 | 已实现 |
| 日期 | 2026-11-06 |
| 关联 | ROADMAP Next · [sdd.md](../dev/sdd.md) |

---

## 1. 背景与目标

项目 Pages 与工作项都已存在，但若详情无法关联文档，知识与执行脱节。目标：在工作项上关联/解除同项目页面，并从工作项跳到页面。

## 2. 范围 / 非目标

**范围**

- 工作项详情可：列出已关联页面、添加同项目页面、解除关联。
- 点击关联页面进入该项目的 Pages 视图并定位到该页。
- 权限：有 `issue:edit`（或等价）才能改关联；只读可查看列表。
- 复用现有 `issue_pages` 与 `GET/POST/DELETE /issues/:issueId/pages`。

**非目标**

- 不在本 SRS 做页面内反向「关联工作项」完整编辑器（若已有 convert-to-issue 等，保持现状）。
- 不做跨项目页面关联。
- 不做页面全文搜索替代选择器。

## 3. 现状对照

| 已有 | 缺口 |
|------|------|
| `issue_pages`；`ListPages` / `AddPage` / 删除；前端 `issue.ts` API | Issue 详情 UI 缺少关联页面管理入口（需本 SRS 验收） |

## 4. 功能需求

| ID | 需求 |
|----|------|
| FR-01 | 详情展示已关联页面列表（空状态明确） |
| FR-02 | 添加同项目页面（防重复） |
| FR-03 | 解除关联 |
| FR-04 | 从关联项跳转到 Pages 并打开该页 |
| FR-05 | 无编辑权限时只读 |

## 5. 验收场景（Given / When / Then）

### AC-01.1 空列表
Given 工作项尚未关联任何页面，用户打开详情  
When 进入关联页面区域  
Then 看到空状态说明与「添加」入口（若有编辑权限）

### AC-02.1 添加页面
Given 同项目存在页面 P，用户有 issue:edit  
When 选择 P 添加关联  
Then 列表出现 P 的标题  
And `GET /issues/:id/pages` 含 P

### AC-02.2 重复添加
Given P 已关联  
When 再次添加 P  
Then 不产生重复行（成功幂等或 4xx，二选一在 Design 写死；列表仍仅一行）

### AC-03.1 解除关联
Given P 已关联  
When 用户解除 P  
Then 列表不再含 P  
And 页面 P 本身仍存在

### AC-04.1 跳转
Given P 已关联  
When 用户点击 P  
Then 进入该项目 Pages，并打开 P

### AC-05.1 只读
Given 用户无 issue:edit  
When 打开详情关联页面区域  
Then 可见已关联列表  
And 无添加/解除控件

## 6. 验收清单

| AC-ID | 场景摘要 | 优先级 | 结果 |
|-------|----------|--------|------|
| AC-01.1 | 空状态 | P0 | |
| AC-02.1 | 添加关联 | P0 | |
| AC-02.2 | 重复不双行 | P0 | |
| AC-03.1 | 解除关联 | P0 | |
| AC-04.1 | 跳转打开页面 | P0 | |
| AC-05.1 | 只读无编辑入口 | P0 | |

## 7. 接口与数据约束要点

- `GET /api/v1/issues/:issueId/pages`
- `POST /api/v1/issues/:issueId/pages?page_id=`
- `DELETE /api/v1/issues/:issueId/pages?page_id=`
- 页面必须同 `project_id`；表 `issue_pages`。

## 8. 风险与开放问题

- 详情页签放置位置（关联 / 独立「页面」）在 Design 定，GWT 只要求「详情内可完成」。
