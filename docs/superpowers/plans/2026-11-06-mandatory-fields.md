# Mandatory Custom Fields Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `is_required` custom fields enforceable on create, update (value set/clear), type convert, and non-dry-run import, with matching frontend cues — per the frozen SRS.

**Architecture:** Keep `IssueService.validateMandatoryCustomFields` as the shared checker (type binding via `issue_type_fields.is_required`). Call it from create (already), convert-type (merge existing CF values), and from `CustomFieldService` set/update/delete paths when the field is required. Import validates each row before insert. Frontend already marks create; ensure detail editors surface the same errors.

**Tech Stack:** Go 1.25, GORM, Gin, Vue 3, Vitest / `go test`

**Spec:** [docs/specs/2026-11-06-mandatory-fields.md](../../specs/2026-11-06-mandatory-fields.md)

## Global Constraints

- Acceptance is Given/When/Then; P0 ACs must pass.
- Empty string / missing / nil = unset; `false` and `0` = set.
- Type-binding `issue_type_fields.is_required` is authoritative.
- Update of unrelated fields (title only) does **not** force filling historical holes (SRS §8).
- Chinese user-facing messages may remain; prefer i18n keys on new frontend strings.
- Do not implement conditional visibility here (separate SRS).

---

### Task 1: Unit-test create already rejects missing required CF

**Files:**
- Create: `backend/internal/service/issue_mandatory_fields_test.go`
- Test helpers: reuse `setupTestDB` / `seedTestData` from `page_service_test.go` (same package `service`)

**Interfaces:**
- Consumes: `IssueService.Create`, `validateMandatoryCustomFields`
- Produces: failing regression suite that documents AC-02.1 / AC-02.2

- [ ] **Step 1: Write the failing/passing regression tests**

```go
package service

import (
	"testing"

	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/require"
)

func seedRequiredCF(t *testing.T, db *gorm.DB, wsID, typeID uint64, name string) uint64 {
	t.Helper()
	f := &model.CustomField{Name: name, FieldType: "text", WorkspaceID: wsID}
	require.NoError(t, db.Create(f).Error)
	require.NoError(t, db.Create(&model.IssueTypeField{
		TypeID: typeID, FieldID: f.ID, IsRequired: true,
	}).Error)
	return f.ID
}

func TestCreate_RejectsMissingRequiredCustomField(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.Issue{}, &model.IssueType{}, &model.State{},
		&model.CustomField{}, &model.IssueTypeField{}, &model.IssueCustomFieldValue{},
	))
	wsID, projID, userID := seedTestData(t, db)
	// ensure project has a state + issue type — extend seed as needed like other issue tests
	// ...
}
```

Fill seed so `Create` works; assert missing CF → `*common.AppError` / BadRequest containing the field name; with value → success.

- [ ] **Step 2: Run tests**

Run: `cd backend && go test ./internal/service/ -run TestCreate_RejectsMissingRequiredCustomField -count=1`
Expected: PASS if create path already correct; if seed incomplete, fix test harness first (not production code).

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/issue_mandatory_fields_test.go
git commit -m "test(issue): lock create-time mandatory custom field checks"
```

---

### Task 2: Reject clearing a required field via CustomFieldService

**Files:**
- Modify: `backend/internal/service/custom_field_service.go` (`SetIssueValue`, `UpdateIssueValue`, delete-value, `BulkSetIssueValues`)
- Modify: `backend/internal/service/issue_mandatory_fields_test.go` (or `custom_field_mandatory_test.go`)

**Interfaces:**
- Consumes: issue’s `issue_type_id`, `issue_type_fields.is_required`
- Produces: set/clear paths return BadRequest when required value would become empty

- [ ] **Step 1: Write failing tests**

```go
func TestSetIssueValue_RejectsEmptyRequired(t *testing.T) {
	// seed issue of type with required field F already valued "x"
	// SetIssueValue(issueID, {FieldID: F, Value: ""})
	// require.Error; value in DB still "x"
}

func TestDeleteIssueValue_RejectsRequired(t *testing.T) {
	// DeleteIssueValue on required field → error; row remains
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `cd backend && go test ./internal/service/ -run 'TestSetIssueValue_RejectsEmptyRequired|TestDeleteIssueValue_RejectsRequired' -count=1`
Expected: FAIL (currently allows empty upsert / delete)

- [ ] **Step 3: Implement**

In `SetIssueValue` / `UpdateIssueValue` after trim: if bound `IsRequired` and value == `""` → `return nil, common.BadRequest("缺少必填字段: " + field.Name)`.

In delete: if required → BadRequest, do not delete.

In `BulkSetIssueValues`: do not skip silently; fail the bulk or per-item — prefer fail entire bulk with BadRequest listing names (simpler, AC-clear).

Helper (same package):

```go
func (s *CustomFieldService) isTypeFieldRequired(typeID, fieldID uint64) (bool, string, error) {
	// join issue_type_fields + custom_fields; return isRequired, name
}
```

- [ ] **Step 4: Run — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/custom_field_service.go backend/internal/service/*mandatory*.go
git commit -m "fix(custom-field): refuse clearing required issue field values"
```

---

### Task 3: ConvertType validates required fields of the target type

**Files:**
- Modify: `backend/internal/service/issue_service.go` (`ConvertType`)
- Modify: `backend/internal/dto/request/issue.go` if convert should accept optional `custom_field_values` to satisfy new required fields in one call
- Test: `issue_mandatory_fields_test.go`

**Interfaces:**
- Consumes: existing `IssueCustomFieldValue` rows + optional map on convert request
- Produces: convert aborted with 4xx when target type requires missing fields (AC-04.1)

- [ ] **Step 1: Write failing test**

```go
func TestConvertType_RejectsWhenTargetRequiresMissingCF(t *testing.T) {
	// issue type A (no required); type B requires field F; issue has no F
	// ConvertType → error; issue_type_id unchanged
}
```

- [ ] **Step 2: Run — expect FAIL**

- [ ] **Step 3: Implement**

Before updating `issue_type_id` in `ConvertType`:

1. Load all `IssueCustomFieldValue` for issue into `map[uint64]interface{}`.
2. If request gains `CustomFieldValues`, merge (request wins).
3. `validateMandatoryCustomFields(projectID, workspaceID, req.TargetTypeID, merged)`.
4. On success, persist any new CF values from the request, then change type.

Add to `ConvertTypeRequest`:

```go
CustomFieldValues map[uint64]interface{} `json:"custom_field_values"`
```

- [ ] **Step 4: Run — expect PASS**

- [ ] **Step 5: Commit**

```bash
git commit -m "fix(issue): validate mandatory fields on type convert"
```

---

### Task 4: Import (non dry_run) rejects rows missing required CF

**Files:**
- Modify: `backend/internal/service/issue_service.go` import helpers (`importItems` / CSV-JSON path)
- Test: extend `issue_import_test.go` or mandatory test file

**Interfaces:**
- Same validator; dry_run may report errors without insert (keep dry_run behavior useful)

- [ ] **Step 1: Write failing test** — formal import row with type that requires F and F absent → row/batch error, no orphan issue without F

- [ ] **Step 2: Run — expect FAIL** if not already enforced

- [ ] **Step 3: Call `validateMandatoryCustomFields` after type resolved, before create; surface field names in import error list

- [ ] **Step 4: PASS + commit**

```bash
git commit -m "fix(import): enforce mandatory custom fields on write import"
```

---

### Task 5: Frontend — detail / value editor shows required + error

**Files:**
- Modify: `frontend/src/views/IssueCreate.vue` (verify only; already has `*`)
- Modify: detail custom field editor component(s) used from IssueDetail / panel (locate via `is_required` / `CustomField` grep)
- Test: Vitest on the composable or component if one exists; else minimal component test

- [ ] **Step 1: Find the detail CF editor** (`rg is_required frontend/src`)

- [ ] **Step 2: Failing test** — required field shows marker; empty save surfaces message

- [ ] **Step 3: Implement** — mirror create: `*` + block or toast using `issue.requiredFieldsMsg`

- [ ] **Step 4: `cd frontend && npm test -- --run <spec>`**

- [ ] **Step 5: Commit**

```bash
git commit -m "fix(ui): show required custom fields on issue detail edit"
```

---

### Task 6: Acceptance notes + SRS status

**Files:**
- Create: `docs/dev/acceptance/2026-11-06-mandatory-fields.md` (checklist copy from SRS §6 with Result column)
- Modify: SRS status → `已实现` only after manual/API pass; leave `已冻结` until verified

- [ ] **Step 1: Run backend suite**

`cd backend && go test ./internal/service/ -count=1`

- [ ] **Step 2: Hand-check AC-02.1, AC-03.1, AC-04.1 against running server or API tests**

- [ ] **Step 3: Fill acceptance checklist; commit**

```bash
git commit -m "docs: record mandatory-fields acceptance run table"
```

---

## Spec coverage

| AC | Task |
|----|------|
| AC-02.1 / AC-02.2 | Task 1 |
| AC-03.1 | Task 2 |
| AC-04.1 | Task 3 |
| AC-04.2 | Task 4 |
| AC-05.1 | Task 5 (create already; detail in Task 5) |

## Plan sequencing for other SRS

After this plan ships: **conditional-fields** (depends on mandatory) → **quick-create** → remaining (type hierarchy, sub-items, page linking, notifications, attachments) as separate plans under `docs/superpowers/plans/`.
