# 类型层级规则 — 需求规格说明书（SRS）

| 项 | 值 |
|----|-----|
| 状态 | 已实现 |
| 日期 | 2026-11-06 |
| 关联 | ROADMAP Next · [sdd.md](../dev/sdd.md) · [type-hierarchy-template-design.md](../kb/architecture/type-hierarchy-template-design.md) §2.3 |

---

## 1. 背景与目标

工作项最多 6 层（depth 0–5）。类型之间的父子约束若只存在后端、配置与错误不可见，深层结构会乱。目标：把**现行代码规则**产品化为可配置、可理解、可验收的行为（不是企业参考里的「子 level 必须 = 父 level + 1」）。

## 2. 范围 / 非目标

**范围**

- 创建/更新父子关系时强制执行下列规则（与 `IssueService` 一致）：
  1. 父存在；禁止环与自父子。
  2. 父 `depth >= 5` 拒绝；子 `depth = 父 + 1`。
  3. 根类型（`level == 0` 且无 `parent_type_id`）不能作为子工作项类型。
  4. 父类型 `level > 0` 时，子类型 level 不得比父更浅。
  5. 子类型若设置 `parent_type_id`，父工作项类型必须匹配。
  6. 父类型 `allowed_child_type_ids` 非空时，子类型必须在名单中。
- 管理端可配置 `parent_type_id`、`level`、`allowed_child_type_ids`。
- 违反时 4xx，文案稳定可测。

**非目标**

- 不引入「恰好父 level+1」的企业参考规则。
- 不做自动迁移历史脏数据；历史非法数据可只读，新写入拒绝。
- 不在本 SRS 做工作项模板系统。

## 3. 现状对照

| 已有 | 缺口 |
|------|------|
| 创建/更新路径已有上述校验与错误文案 | 配置 UI / 文档化验收是否完整 |
| `GET /issues/tree`、`GET /issues/:id/children` | 配置 `allowed_child_type_ids` 的管理体验需验收 |

## 4. 功能需求

| ID | 需求 |
|----|------|
| FR-01 | 深度上限 6 层（depth 0–5） |
| FR-02 | 根类型不可作子项类型 |
| FR-03 | level 不得比父类型更浅（当父 level > 0） |
| FR-04 | `parent_type_id` 与 `allowed_child_type_ids` 约束 |
| FR-05 | 管理端可配置上述类型字段 |
| FR-06 | 非法层级操作返回明确 4xx |

## 5. 验收场景（Given / When / Then）

### AC-01.1 超过最大深度
Given 一条工作项链已使父项 `depth = 5`  
When 尝试在其下再创建子工作项  
Then 返回 4xx，信息含 Maximum hierarchy depth（或同等稳定文案）  
And 无新子项

### AC-02.1 根类型不能作子
Given 类型 T 为 level 0 且无 `parent_type_id`  
When 尝试创建 parent 指向其他工作项且 `type_id = T` 的子项  
Then 返回 4xx（root-level type cannot have a parent 或同等文案）

### AC-03.1 子类型 level 过浅
Given 父工作项类型 level = 2，子类型 level = 1  
When 尝试建立该父子关系  
Then 返回 4xx（child type level must not be shallower…）

### AC-04.1 allowed_child_type_ids
Given 父类型 `allowed_child_type_ids` 仅含类型 A  
When 尝试添加类型 B 的子工作项  
Then 返回 4xx（not allowed under the parent type）

### AC-04.2 parent_type_id 不匹配
Given 子类型 `parent_type_id` 指向类型 X，父工作项类型为 Y≠X  
When 尝试建立父子  
Then 返回 4xx

### AC-05.1 配置可读可写
Given 工作空间管理员打开类型管理  
When 设置某类型的 `allowed_child_type_ids` 并保存  
Then 再次读取该类型可见相同名单

## 6. 验收清单

| AC-ID | 场景摘要 | 优先级 | 结果 |
|-------|----------|--------|------|
| AC-01.1 | depth 上限 | P0 | |
| AC-02.1 | 根类型不作子 | P0 | |
| AC-03.1 | level 不得更浅 | P0 | |
| AC-04.1 | allowed_child 名单 | P0 | |
| AC-04.2 | parent_type_id 匹配 | P0 | |
| AC-05.1 | 配置可保存读回 | P0 | |

## 7. 接口与数据约束要点

- 校验发生在 issue 创建/更新（含改 `parent_id` / `type_id`）。
- 类型字段：`level`、`parent_type_id`、`allowed_child_type_ids`（JSON 数组）。
- 树：`GET /issues/tree`；子列表：`GET /issues/:issueId/children`。

## 8. 风险与开放问题

- 无。规则以代码 §2.3 为准；若产品要改为「严格父+1」，另开 SRS 变更。
