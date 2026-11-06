# Type Hierarchy Rules Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Lock P0 hierarchy enforcement with tests and complete admin config for `level` / `parent_type_id` / `allowed_child_type_ids` — per frozen SRS. Core create/update checks already exist in `IssueService`.

**Architecture:** Keep existing validation messages in `IssueService`. Add focused Go tests for AC-01–04. Extend type create DTO + workspace/project type UIs so AC-05.1 can save and read back `allowed_child_type_ids`.

**Tech Stack:** Go, Vue 3, Vitest / `go test`

**Spec:** [docs/specs/2026-11-06-type-hierarchy-rules.md](../../specs/2026-11-06-type-hierarchy-rules.md)

## Global Constraints

- Do not introduce “child level must equal parent+1”.
- Do not migrate historical dirty data.
- Leave create-with-parent-and-omitted-type_id as type-check skip (aligned with quick-create fix); document in acceptance. Depth/cycle still enforced.
- Prefer stable English error substrings already used by service.

---

### Task 1: Backend tests AC-01.1 … AC-04.2

**Files:** Create `backend/internal/service/issue_hierarchy_rules_test.go`

- [ ] Tests for depth>5, root type as child, shallower level, allowed_child whitelist, parent_type_id mismatch — assert 4xx messages and no orphan child
- [ ] Commit: `test(issue): lock type hierarchy P0 acceptance rules`

---

### Task 2: allowed_child on IssueType create + API round-trip test

**Files:** `dto/request/issue_type.go`, `issue_type_service.go`, tests

- [ ] Optional `AllowedChildTypeIDs` on create; update already supports
- [ ] Test create/update + get returns same list (AC-05.1 API)
- [ ] Commit: `feat(issue-type): persist allowed_child_type_ids on create`

---

### Task 3: Admin UI — workspace + project type editors

**Files:** `WorkspaceIssueTypeManager.vue`, `IssueTypeList.vue` (shared fields if practical)

- [ ] Expose level, parent_type_id, allowed_child_type_ids multi-select; save via existing update/create APIs
- [ ] Light Vitest or smoke if feasible
- [ ] Commit: `feat(ui): configure type hierarchy fields in type managers`

---

### Task 4: Acceptance notes

**Files:** `docs/dev/acceptance/2026-11-06-type-hierarchy-rules.md`

- [ ] Fill AC table; SRS stays 已冻结 until UI manually verified if needed
- [ ] Commit: `docs: record type-hierarchy acceptance run table`

---

## Spec coverage

| AC | Task |
|----|------|
| AC-01.1 … AC-04.2 | Task 1 |
| AC-05.1 | Task 2 + Task 3 |
