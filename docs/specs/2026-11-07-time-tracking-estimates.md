# 工时记录与估点 — 需求规格说明书（SRS）

| 项 | 值 |
|----|-----|
| 状态 | 已实现 |
| 日期 | 2026-11-07 |
| 关联 | ROADMAP Later · [sdd.md](../dev/sdd.md) |
| 依赖 | 既有 `time_tracks` / `estimate_points` / `project_estimate_settings` API |

---

## 1. 背景与目标

计时面板与估点目录 API 已存在，但工作项上的估点未持久化，项目估点管理 UI 未挂载，工时缺少手动录入与权限门控。目标：P0 打通 **估点（points）挂到 issue** + **可靠计时/手动记工时** + **项目估点配置**。

## 2. 范围 / 非目标

**范围（P0）**

- Issue 持久化 `estimate_point_id`（项目 points 目录中的一项）；创建/更新/响应可读可写。
- 详情侧栏选择/清空估点；项目设置挂载估点管理（`EstimatePointManager`）；`settings:manage` 管理估点。
- 工时：现有 start/stop/list/delete；新增手动录入（时长秒或分钟约定与 API 一致，文档钉死为 **秒**）；`time_track:create|delete|view` 强制。
- 有估点时展示已记录总时长（及可选与估点换算对照，若点无小时映射则仅显示总秒/格式化时长）。
- CSV 导入 `estimate` 列映射到 `estimate_point_id`（按点值/名称匹配）。
- Seed 中 time track `duration` 单位与运行时一致（秒）。

**非目标**

- 估点类别 / 时间预设挂到 issue（P1）。
- 工时审批、报表大盘、暂停计时、编辑历史条目。
- 列表/看板估点列（P1 可加）。

## 3. 现状对照

| 已有 | 缺口 |
|------|------|
| `TimeTrackPanel` start/stop；time-track API | 手动录入；RBAC；i18n；seed 单位 |
| Estimate points CRUD API；`EstimatePointManager.vue` | 未挂项目设置；issue 无落库字段 |
| DTO `EstimatePointID` | Create/Update/Response 未贯通 |

## 4. 功能需求

| ID | 需求 |
|----|------|
| FR-01 | Issue 存读 `estimate_point_id` |
| FR-02 | 详情侧栏估点选择器 |
| FR-03 | 项目设置配置估点目录 |
| FR-04 | 手动创建 time track；计时面板 i18n |
| FR-05 | time_track 权限强制 |
| FR-06 | 导入写入估点 |
| FR-07 | 详情展示累计已记工时 |

## 5. 验收场景（Given / When / Then）

### AC-01.1 设置估点
Given 项目已有估点「3」  
When 将工作项估点设为「3」并保存  
Then 再次打开该工作项响应含对应 `estimate_point_id` / 显示名

### AC-02.1 清空估点
Given 工作项已有估点  
When 清空估点并保存  
Then 估点为空

### AC-03.1 项目配置估点
Given 有 `settings:manage` 的用户打开项目设置  
When 新增一个估点并保存  
Then 列表可读回；issue 侧栏可选到该点

### AC-04.1 手动记工时
Given 用户有 `time_track:create`  
When 为工作项手动录入 3600 秒  
Then 列表与 summary 含该条；累计增加

### AC-05.1 无权限不可记
Given 用户无 `time_track:create`  
When 调用 start 或手动创建  
Then 403

### AC-06.1 导入估点
Given CSV 含 estimate 列值为已有点值  
When 正式导入  
Then 新建工作项带对应 `estimate_point_id`

## 6. 验收清单

| AC-ID | 场景摘要 | 优先级 | 结果 |
|-------|----------|--------|------|
| AC-01.1 | 设置估点 | P0 | |
| AC-02.1 | 清空估点 | P0 | |
| AC-03.1 | 项目配置 | P0 | |
| AC-04.1 | 手动记工时 | P0 | |
| AC-05.1 | 权限 | P0 | |
| AC-06.1 | 导入 | P0 | |

## 7. 接口与数据约束要点

- Migration：`issues.estimate_point_id` nullable FK → `estimate_points`。
- 手动录入：建议 `POST /issues/:id/time-tracks` body `{ started_at, ended_at }` 或 `{ duration, ended_at }`（实现选一，duration **秒**）。
- 计时路由加 `RequirePermission` 对应 `time_track:*`。

## 8. 风险与开放问题

- 估点与小时换算：P0 可不换算，只显示点数 + 已记时长。
- Categories / time modes：P1。
