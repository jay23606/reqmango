# Conditional Custom Fields Visibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply conditional hide rules so create/detail UI and write-path mandatory checks only enforce **visible** required custom fields — per the frozen SRS.

**Architecture:** Keep `ConditionalFieldService.EvaluateFieldVisibility` as the source of truth for the visible field ID set. Thread a small visibility context (type, state, priority, assignees, labels) into `validateMandatoryCustomFields` and CustomFieldService required checks; skip required enforcement for field IDs not in the visible set. Fix the frontend API client to `/v1/conditional-fields`, then wire create + detail editors to evaluate and hide controls.

**Tech Stack:** Go 1.25, GORM, Gin, Vue 3, Vitest / `go test`

**Spec:** [docs/specs/2026-11-06-conditional-fields.md](../../specs/2026-11-06-conditional-fields.md)

## Global Constraints

- P0 ACs only; condition types in scope for evaluate: `type` / `state` / `priority` / `assignee` / `label` (existing). Do not build CF-value→CF visibility.
- Hide-on-match semantics; no AND/OR expression trees.
- Mandatory SRS still applies for **visible** required fields.
- Empty / missing / nil = unset; `false` / `0` = set.
- Prefer constructing `NewConditionalFieldService(s.db)` inside IssueService/CustomFieldService over changing `NewIssueService` signature (keeps existing tests compiling).
- Actual API is `/v1/conditional-fields?workspace_id=` (not project-scoped); align frontend + SRS notes in acceptance, do not invent a second CRUD surface.
- Chinese messages OK; new UI strings prefer i18n keys.

---

### Task 1: Lock Evaluate + CRUD (AC-01.1 / AC-02.1 / AC-05.1 backend)

**Files:**
- Create: `backend/internal/service/conditional_field_service_test.go`
- Reuse: `setupTestDB` / workspace seed helpers from other service tests

**Interfaces:**
- Consumes: `ConditionalFieldService.Create`, `List`, `EvaluateFieldVisibility`, `Update`/`Delete`
- Produces: regression suite for create-and-list, evaluate hide, disable/delete restores visibility

- [ ] **Step 1: Write tests**

```go
func TestConditionalField_CreateAndList(t *testing.T) { /* AC-01.1 */ }
func TestEvaluateFieldVisibility_HidesMatchedField(t *testing.T) { /* AC-02.1 state=Done hides F */ }
func TestEvaluateFieldVisibility_DisabledRuleShowsField(t *testing.T) { /* AC-05.1 */ }
```

- [ ] **Step 2: Run** — expect PASS if evaluate already correct; fix evaluate bugs only if tests fail for real logic errors (not harness).

- [ ] **Step 3: Commit**

```bash
git commit -m "test(conditional-field): lock evaluate and CRUD visibility rules"
```

---

### Task 2: Mandatory validation skips hidden fields (AC-04.1 / AC-04.2)

**Files:**
- Modify: `backend/internal/service/issue_service.go` (`validateMandatoryCustomFields` + Create / ConvertType / import call sites)
- Modify: `backend/internal/service/issue_mandatory_fields_test.go` (or new `issue_conditional_mandatory_test.go`)

**Interfaces:**
- Add context args (or a small struct) to `validateMandatoryCustomFields`:

```go
type fieldVisibilityContext struct {
	IssueTypeID uint64
	StateID     uint64
	Priority    string
	AssigneeIDs []uint64
	LabelIDs    []uint64
}
```

After loading required bindings, call `EvaluateFieldVisibility(workspaceID, &typeID, stateID, …)`; **skip** any required field whose ID is not in the visible set.

Create/Convert/Import must pass state/priority/assignees/labels from the request (or defaults). Import without those may use zero/empty context (fields with state-based hide rules stay visible unless context matches — document in test).

- [ ] **Step 1: Failing tests**

```go
func TestCreate_SkipsRequiredWhenHiddenByState(t *testing.T) { /* AC-04.1 */ }
func TestCreate_StillRequiresWhenVisible(t *testing.T) { /* AC-04.2 */ }
```

- [ ] **Step 2: Implement + PASS**

- [ ] **Step 3: Commit**

```bash
git commit -m "fix(issue): skip mandatory CF checks for conditionally hidden fields"
```

---

### Task 3: CustomFieldService required clears respect visibility

**Files:**
- Modify: `backend/internal/service/custom_field_service.go`
- Test: extend mandatory / conditional tests

**Interfaces:**
- Before rejecting empty/delete on required fields, load issue context and evaluate; if field is hidden for current issue context, allow clear/delete (or no-op success). If visible and required, keep BadRequest.

- [ ] **Step 1: Failing test** — required field hidden by state → SetIssueValue("") succeeds (or does not return 缺少必填字段)

- [ ] **Step 2: Implement + PASS + commit**

```bash
git commit -m "fix(custom-field): allow clearing required CF when conditionally hidden"
```

---

### Task 4: Fix frontend conditional-field API client

**Files:**
- Rewrite: `frontend/src/api/conditional-field.ts` to match handler DTOs + `/conditional-fields?workspace_id=` and `POST /conditional-fields/evaluate`
- Optional small unit test of URL/payload shape if project already tests API modules

- [ ] **Step 1: Align types with `dto/request/conditional_field.go` + handler evaluate request/response**

- [ ] **Step 2: Commit**

```bash
git commit -m "fix(ui): align conditional-field API client with /v1/conditional-fields"
```

---

### Task 5: Create + detail UI apply visibility (AC-03.1 / AC-05.1 UI)

**Files:**
- Modify: `frontend/src/views/IssueCreate.vue`
- Modify: `frontend/src/components/IssuePropertySidebar.vue` (and callers if workspace_id / state needed)
- Test: Vitest — hidden field not rendered; required marker only when visible

**Behavior:**
- Debounce evaluate on type/state/priority/assignee/label changes (~200–300ms ok).
- Filter linked CF list by `visible_field_ids`.
- `isFieldRequired` / `canSubmit` / `attemptCustomFieldUpdate` only for visible fields.

- [ ] **Step 1: Failing Vitest**

- [ ] **Step 2: Implement + PASS**

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(ui): hide conditionally invisible custom fields on create and detail"
```

---

### Task 6: Acceptance notes

**Files:**
- Create: `docs/dev/acceptance/2026-11-06-conditional-fields.md`
- Do **not** flip SRS status to 已实现 until P0 checklist filled; leave 已冻结 if any AC unmet

- [ ] **Step 1: Run** `go test ./internal/service/ -count=1` and focused Vitest

- [ ] **Step 2: Fill AC table + commit**

```bash
git commit -m "docs: record conditional-fields acceptance run table"
```

---

## Spec coverage

| AC | Task |
|----|------|
| AC-01.1 | Task 1 |
| AC-02.1 | Task 1 |
| AC-03.1 | Task 5 |
| AC-04.1 / AC-04.2 | Task 2 (+ Task 3 CF path) |
| AC-05.1 | Task 1 + Task 5 |

## Sequencing note

After this plan: **quick-create** → **type-hierarchy** → **sub-items** → page-linking / notifications / attachments.
