# Time Tracking & Estimate Points Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Persist issue `estimate_point_id`, mount project estimate settings, harden time-track RBAC + manual entry + i18n, and wire CSV import — per frozen SRS.

**Architecture:** DTOs already expose `estimate_point_id`; add nullable column on `issues` (GORM AutoMigrate + optional SQL migration), wire Create/Update/BuildIssueResponse/import. Mount existing `EstimatePointManager` in ProjectSettings behind `settings:manage`. Time tracks already work for start/stop; add `Create` manual entry (duration **seconds**), `RequirePermission` on routes, fix seed minutes→seconds, i18n TimeTrackPanel, show summary spent next to estimate in sidebar/tab.

**Design pins:**
- Clear estimate: PATCH with `estimate_point_id: 0` (pointer present, value 0) → NULL; omit field → no change (same as `release_id`).
- Validate point belongs to issue’s project.
- Manual log: `POST /issues/:id/time-tracks` body `{ duration: seconds, description?, ended_at? }`; if `ended_at` omitted use now; `started_at = ended_at - duration`.
- Estimate mutate routes: `settings:manage`; GET stays auth+project access.
- P0 display: estimate name/value in sidebar + total spent hours from summary; no points↔hours conversion.
- Categories / time-mode on issues: out of scope.

**Tech Stack:** Go, Gin, GORM, Vue 3, Vitest / `go test`

**Spec:** [docs/specs/2026-11-07-time-tracking-estimates.md](../../specs/2026-11-07-time-tracking-estimates.md)

## File map

| Area | Touch |
|------|--------|
| Model | `backend/internal/model/issue.go` — `EstimatePointID *uint64` |
| Migration (optional) | `backend/migrations/000003_issue_estimate_point.up.sql` |
| Issue service | Create / Update / BuildIssueResponse / importItems |
| Time track | DTO Create; service Create; handler Create; router + RBAC |
| Estimate routes | `settings:manage` on mutating estimate endpoints |
| Seed | `seed.go` duration ×60 + EndedAt; python seed if easy |
| FE | Sidebar estimate picker; ProjectSettings mount; TimeTrackPanel i18n + manual form; API create |
| Tests | `issue_estimate_test.go`, `time_track_service_test.go`, import assert |

---

### Task 1: Persist `estimate_point_id` on issues (AC-01.1, AC-02.1)

**Files:** `model/issue.go`, `issue_service.go`, `issue_estimate_test.go` (new), optional migration

- [ ] Add `EstimatePointID *uint64` (+ optional Preload relation) on `model.Issue`
- [ ] Create: copy `req.EstimatePointID` after validating point.project_id
- [ ] Update: if `req.EstimatePointID != nil` { if `*id == 0` clear else validate+set }; activity field `estimate_point`
- [ ] `BuildIssueResponse`: set `resp.EstimatePointID = issue.EstimatePointID`
- [ ] Tests: create with point; update set; update clear with 0; wrong-project point → 400
- [ ] Commit: `feat(issue): persist estimate_point_id on create/update`

### Task 2: CSV import → estimate_point_id (AC-06.1)

**Files:** `issue_service.go` import path; `issue_import_test.go`

- [ ] After parse `EstimatePoints`, resolve project point by `value` (int match) or `name` string; set `EstimatePointID` on create request
- [ ] Test: import CSV estimate `3` maps to point with value 3
- [ ] Commit: `feat(import): map CSV estimate to estimate_point_id`

### Task 3: Manual time entry + RBAC + seed seconds (AC-04.1, AC-05.1)

**Files:** `dto/request/time_track.go`, `time_track_service.go`, `time_track_handler.go`, `router.go`, `seed/seed.go`, `scripts/seed_massive_data.py`, tests

- [ ] `TimeTrackCreateRequest{ Duration int64 binding required min=1; Description *string; EndedAt *time.Time }`
- [ ] `Create(issueID, userID, req)` → row with StartedAt/EndedAt/Duration
- [ ] `POST /:issueId/time-tracks` → Create
- [ ] Permissions: start/create → `time_track:create`; list/summary → `time_track:view`; delete → `time_track:delete` (project scope, same pattern as issue routes)
- [ ] Seed: `duration = minutes * 60`, set `EndedAt = StartedAt + duration`
- [ ] Tests: Create adds to summary; (optional) permission via middleware already covered elsewhere — service test enough
- [ ] Commit: `feat(time-track): manual entry, RBAC, seed seconds`

### Task 4: Estimate settings RBAC + ProjectSettings UI (AC-03.1)

**Files:** `router.go` estimate groups; `ProjectSettings.vue`; i18n `settings.estimates` if missing

- [ ] Mutating estimate-points/categories/time/settings → `settings:manage`
- [ ] Menu item `estimates` + section rendering `<EstimatePointManager :project-id="..." />`
- [ ] Commit: `feat(settings): mount estimate point manager with settings:manage`

### Task 5: Issue sidebar estimate + TimeTrackPanel polish (AC-01–04 UX)

**Files:** `IssuePropertySidebar.vue` (+ parent emit/update), `TimeTrackPanel.vue`, `api/time-track.ts`, locales keys for manual entry

- [ ] Load project estimate points; picker bound to `issue.estimate_point_id`; emit update (0/null to clear)
- [ ] Show spent summary near estimate or in time tab (reuse getTimeSummary)
- [ ] TimeTrackPanel: `t('timeTrack.*')`; form minutes→seconds for manual create
- [ ] Vitest smoke if cheap
- [ ] Commit: `feat(ui): estimate sidebar picker and time-track i18n/manual log`

### Task 6: Acceptance + ROADMAP/CHANGELOG

- [ ] `docs/dev/acceptance/2026-11-07-time-tracking-estimates.md` checklist
- [ ] Mark ROADMAP Later item when done; CHANGELOG entry when opening PR
- [ ] Commit: `docs: time-tracking-estimates acceptance notes`

---

## Spec coverage

| AC | Task |
|----|------|
| AC-01.1 / 02.1 | Task 1, 5 |
| AC-03.1 | Task 4 |
| AC-04.1 / 05.1 | Task 3, 5 |
| AC-06.1 | Task 2 |
