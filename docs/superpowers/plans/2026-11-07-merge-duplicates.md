# Merge Duplicate Issues Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Let users merge source issue A into target B in the same project: close A as duplicate, link A→B, append a merge summary on B, union labels, write activity — per frozen SRS.

**Architecture:** There is already `POST /issues/merge` → `IssueService.MergeDuplicates` that **hard-deletes** sources after copying labels/assignees. **Replace that behavior** with SRS semantics: keep source row, mark cancelled/duplicate, link to target, append summary, union labels — one transaction. Prefer extending the existing endpoint/DTO (`target_issue_id` + `source_issue_ids`) rather than adding a second merge API. UI: issue detail “合并到…” + wire Intake/create duplicate suggestion.

**Design pins (open questions closed for P0):**
- Comments / attachments: **not migrated** (P1).
- Custom fields: **not merged** (keep target values).
- Labels: union onto target (keep existing KeepSourceLabels path).
- Assignees: optional union (existing KeepSourceAssignees); default **true** for P0.
- Description: append HTML block “Merged from #SEQ: title” (+ truncated source body).
- Source after merge: **do not delete**; set cancelled-like state if present, else leave state and set a clear activity + relation/duplicate link so default open lists can filter later.
- Idempotent: source already merged into same target → 200 no-op; into different target → 409.
- Self / cross-project / missing edit on either → 4xx as SRS.
- **Breaking:** callers that relied on hard-delete lose that; document in CHANGELOG.

**Tech Stack:** Go, Gin, Vue 3, Vitest / `go test`

**Spec:** [docs/specs/2026-11-07-merge-duplicates.md](../../specs/2026-11-07-merge-duplicates.md)

---

### Task 1: Backend Merge API + tests (AC-01–04)

**Files:** `issue_service.go`, `issue_handler.go`, `router.go`, `dto/request`, new `issue_merge_test.go`

- [ ] `POST /issues/:id/merge` body `{ "target_issue_id": N }`, permission `issue:edit` on project
- [ ] Implement Merge with prechecks + transaction
- [ ] Tests: success, cross-project, no permission (handler), self-merge, idempotent
- [ ] Commit: `feat(issue): add merge-into duplicate API`

### Task 2: Frontend merge UI

**Files:** Issue detail actions; Intake / create duplicate warning if easy; `issue.ts` API; i18n

- [ ] “合并到…” picker (search same project) + confirm dialog
- [ ] On success: toast, navigate to target or refresh source showing merged badge
- [ ] Vitest for confirm/disabled states if practical
- [ ] Commit: `feat(ui): merge duplicate issue into target`

### Task 3: Intake “Merge into suggested” entry (optional if Task 2 covers)

- [ ] From duplicate suggestion, prefill target
- [ ] Commit: `feat(ui): merge from intake duplicate suggestion` (skip if already in Task 2)

### Task 4: Acceptance notes

- [ ] `docs/dev/acceptance/2026-11-07-merge-duplicates.md`
- [ ] Commit: `docs: record merge-duplicates acceptance run table`

---

## Spec coverage

| AC | Task |
|----|------|
| AC-01.1 | Task 1–2 |
| AC-02.1 / 03.1 / 04.1 | Task 1 |
