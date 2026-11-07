# 验收记录 — 工时记录与估点（2026-11-07）

| AC-ID | 场景 | 结果 | 证据 |
|-------|------|------|------|
| AC-01.1 | 设置估点 | 自动化 | `TestIssueEstimatePoint_CreateUpdateClear` |
| AC-02.1 | 清空估点 | 自动化 | 同上（`estimate_point_id: 0`） |
| AC-03.1 | 项目配置估点 | 手工/代码 | `ProjectSettings` 挂载 `EstimatePointManager`；mutate 路由 `settings:manage` |
| AC-04.1 | 手动记工时 | 自动化 | `TestTimeTrackCreate_AddsToSummary` + UI 分钟→秒 |
| AC-05.1 | 无权限不可记 | 代码审查 | router `time_track:create\|view\|delete` |
| AC-06.1 | 导入估点 | 自动化 | `TestImportMapsEstimatePointsToID` |

**分支:** `feat/time-tracking-estimates`
