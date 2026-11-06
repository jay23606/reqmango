# Sub-Items Panel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or executing-plans. Steps use `- [ ]`.

**Goal:** Make issue detail sub-items panel meet P0 ACs: empty state, quick create child, add existing, hierarchy errors visible, count consistent.

**Architecture:** Existing `RelationSubIssuesCard.vue` + `GET/POST children` / create with `parent_id`. Close gaps: error surfacing, count badge sync, Vitest/GWT-style service+UI tests. Reorder is P1 — skip UI unless already present.

**Tech Stack:** Vue 3, Go, Vitest / `go test`

**Spec:** [docs/specs/2026-11-06-sub-items-panel.md](../../specs/2026-11-06-sub-items-panel.md)

### Task 1: Backend/service tests for children create + hierarchy fail
- [ ] Assert illegal depth/type returns 4xx and no new child; legal create appears in children list
- [ ] Commit: `test(issue): cover sub-item create and hierarchy failures`

### Task 2: UI — empty state, errors, count sync
- [ ] `RelationSubIssuesCard.vue` (+ detail header/tab count): empty CTA; toast/inline on hierarchy fail; count matches list length
- [ ] Vitest for empty + error display if practical
- [ ] Commit: `fix(ui): harden sub-items panel empty state errors and counts`

### Task 3: Acceptance notes
- [ ] `docs/dev/acceptance/2026-11-06-sub-items-panel.md`; SRS stays 已冻结 if manual ACs remain
- [ ] Commit: `docs: record sub-items-panel acceptance run table`
