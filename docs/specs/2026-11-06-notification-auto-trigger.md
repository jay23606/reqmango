# 通知自动送达 — 需求规格说明书（SRS）

| 项 | 值 |
|----|-----|
| 状态 | 已实现 |
| 日期 | 2026-11-06 |
| 关联 | ROADMAP Next · [sdd.md](../dev/sdd.md) · [notification-design.md](../kb/architecture/notification-design.md) |

---

## 1. 背景与目标

通知中心若只靠手动或零星写入，价值有限。目标：关键业务事件自动写入收件人通知，类型正确，发送者本人不刷自己的收件箱（除非产品另有约定——本 SRS 约定**不通知触发者本人**）。

## 2. 范围 / 非目标

**范围**

以下事件必须自动生成通知（有明确收件人时）：

| 事件 key | 典型收件人 | 通知 type |
|----------|------------|-----------|
| `issue_assigned` | 被指派人 | assignment |
| `issue_mentioned` | 被 @ 用户 | assignment |
| `issue_state_changed` | 关注者/经办等相关人（以实现已有集合为准，SRS 验收「至少经办人在集合内则收到」） | status |
| `issue_commented` | 除作者外的经办/关注等相关人 | comment |
| `issue_created` | 可选：项目关注者；若现网仅部分场景，P1 | info |
| `issue_due_soon` | 经办人（由到期扫描触发，需超级用户扫描接口时保持现状权限） | reminder |
| `cycle_created` | 相关成员（以实现为准） | info |

- 收件箱未读计数增加；打开通知中心可见标题/正文/关联 issue。
- 触发者本人不收到因自己动作产生的同类通知。

**非目标**

- 不做邮件/Slack 渠道（Slack 集成另册）；本 SRS 仅站内 `notifications` 表。
- 不做细粒度「每事件订阅偏好」UI（可后续 SRS）。
- 不改审批类事件的产品范围（若代码已有 approval_* 落入 info，保持并在 Design 列表化）。

## 3. 现状对照

| 已有 | 缺口 |
|------|------|
| `TriggerNotification` / `Bulk`；指派、评论、状态、提及、创建等已有调用点 | 需用 GWT 钉死「事件 → 收件人 → 未读」全链路；补齐遗漏事件 |
| 通知列表 API、前端通知中心 | 与事件矩阵不一致处按本 SRS 修 |

## 4. 功能需求

| ID | 需求 |
|----|------|
| FR-01 | 指派产生 assignment 通知 |
| FR-02 | 评论产生 comment 通知（非作者） |
| FR-03 | 状态变更产生 status 通知 |
| FR-04 | @提及产生 assignment 类通知 |
| FR-05 | 触发者不自通知 |
| FR-06 | 未读计数与列表可见 |
| FR-07 | （P1）到期提醒 `issue_due_soon` |

## 5. 验收场景（Given / When / Then）

### AC-01.1 指派
Given 用户 A 将工作项指派给用户 B（A≠B）  
When 指派成功  
Then B 的通知收件箱新增一条 type 为 assignment（或产品映射的 assignment）的未读通知  
And 通知关联该 issue  
And A 不因该动作新增「指派给自己」类通知

### AC-02.1 评论
Given 工作项经办人为 B，A 添加评论（A≠B）  
When 评论保存成功  
Then B 收到 comment 类未读通知

### AC-03.1 状态变更
Given 工作项经办人为 B，A 将状态从「待办」改为「进行中」  
When 更新成功  
Then B 收到 status 类未读通知（若 B 在现网收件人集合中；若不在，以实现集合为准并在跑表注明角色）

### AC-04.1 提及
Given A 在评论中 @B  
When 评论保存  
Then B 收到提及类未读通知

### AC-05.1 不自通知
Given A 指派工作项给 A 自己  
When 指派成功  
Then A 不会因此新增一条指派通知（或仅一条且可接受——**本 SRS 要求零条自指派通知**）

### AC-06.1 未读计数
Given B 有一条未读通知  
When B 打开应用通知入口  
Then 未读计数 ≥ 1  
And 列表可见该通知标题

### AC-07.1 到期提醒（P1）
Given 工作项经办人为 B，目标日在扫描窗口内（如 24h）  
When 授权用户触发到期扫描  
Then B 收到 reminder 类通知

## 6. 验收清单

| AC-ID | 场景摘要 | 优先级 | 结果 |
|-------|----------|--------|------|
| AC-01.1 | 指派通知 | P0 | |
| AC-02.1 | 评论通知 | P0 | |
| AC-03.1 | 状态变更通知 | P0 | |
| AC-04.1 | 提及通知 | P0 | |
| AC-05.1 | 不自通知 | P0 | |
| AC-06.1 | 未读可见 | P0 | |
| AC-07.1 | 到期提醒 | P1 | |

## 7. 接口与数据约束要点

- 写入：`NotificationService.TriggerNotification` / `TriggerNotificationsBulk`。
- 事件 key 与 type 映射见上表；与 [notification-design.md](../kb/architecture/notification-design.md) 冲突时以本 SRS + 代码调用点为准并回写设计笔记。
- 列表/未读：现有 notifications API。

## 8. 风险与开放问题

- 「状态变更收件人集合」若含关注者，跑表须写明如何制造关注者数据。
- `issue_updated` 是否一律通知：默认**不**作为 P0，避免噪音。
