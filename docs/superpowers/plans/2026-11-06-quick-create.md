# Quick Create (List / Board) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close P0 gaps so list/board quick-create respects `issue:create`, never bypasses mandatory custom fields when a type is resolved, and surfaces empty-title / API errors to the user — per the frozen SRS.

**Architecture:** Keep existing `QuickCreateInput` + `POST /issues`. Enforce permission on route + hide UI. Ensure Create always resolves a type (client default and/or server default) before `validateMandatoryCustomFields`. Improve UX for empty title and 4xx (toast, keep title, optional link to full form).

**Tech Stack:** Go, Gin, Vue 3, Vitest / `go test`

**Spec:** [docs/specs/2026-11-06-quick-create.md](../../specs/2026-11-06-quick-create.md)

## Global Constraints

- P0 ACs only; no new create API; no tree/calendar quick-create; do not change `RelationSubIssuesCard` behavior.
- Mandatory + conditional visibility rules already in `validateMandatoryCustomFields` — do not weaken them.
- Prefer i18n for new user-facing strings.

---

### Task 1: Backend require `issue:create` on POST /issues

**Files:** `backend/internal/router/router.go` (+ auth test if pattern exists)

- [ ] Add `RequirePermission(db, "issue:create", "project")` (or equivalent) on `POST /v1/issues`
- [ ] Test: no permission → 403
- [ ] Commit: `fix(auth): require issue:create on issue create endpoint`

---

### Task 2: Resolve default type before mandatory CF check

**Files:** `backend/internal/service/issue_service.go`, tests in `issue_mandatory_fields_test.go` or new

- [ ] When `TypeID` omitted, resolve project default issue type (same as UI intent) before `validateMandatoryCustomFields`
- [ ] Test: required CF on default type + create without type_id → 400 with field name, no row
- [ ] Commit: `fix(issue): apply default type before mandatory CF validation`

---

### Task 3: Frontend hide quick-create without `issue:create`

**Files:** `Project.vue`, `IssueList.vue`, `IssueKanban.vue`, `QuickCreateInput.vue` (if needed)

- [ ] Gate all quick-create entry points + align header 新建 with `can('issue:create')`
- [ ] Vitest or component test if feasible
- [ ] Commit: `fix(ui): hide quick create without issue:create permission`

---

### Task 4: Empty title message + API error toast (AC-01.2 / AC-06.1 UI)

**Files:** `QuickCreateInput.vue`, i18n locales

- [ ] Empty Enter → visible validation (inline or toast); no API call
- [ ] On create 4xx: toast with server message; keep title; optional “完整表单” link to IssueCreate with query prefill
- [ ] Ensure client always sends resolved `default-type-id` when available
- [ ] Vitest for empty title / error toast
- [ ] Commit: `fix(ui): surface quick-create validation and API errors`

---

### Task 5: Acceptance notes

**Files:** `docs/dev/acceptance/2026-11-06-quick-create.md`

- [ ] Run focused tests; fill AC table; leave SRS 已冻结 if any AC unmet
- [ ] Commit: `docs: record quick-create acceptance run table`

---

## Spec coverage

| AC | Task |
|----|------|
| AC-01.1 | existing + Task 3/4 polish |
| AC-01.2 | Task 4 |
| AC-02.1 | existing (verify only) |
| AC-04.1 | Task 1 + Task 3 |
| AC-06.1 | Task 2 + Task 4 |
