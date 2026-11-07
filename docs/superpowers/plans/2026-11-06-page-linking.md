# Page ↔ Issue Linking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or executing-plans. Steps use `- [ ]`.

**Goal:** Expose issue↔page link management in issue detail with empty state, add (dedupe), unlink, navigate-to-page, read-only without edit permission.

**Architecture:** API already in `issue.ts` / page_service. Add detail UI tab or section; pin dedupe as idempotent success (single row). Gate mutate on `issue:edit`.

**Tech Stack:** Vue 3, Go, Vitest

**Spec:** [docs/specs/2026-11-06-page-linking.md](../../specs/2026-11-06-page-linking.md)

### Task 1: Confirm/fix backend list/add/remove + tests
- [ ] Idempotent add; delete unlink; permission checks
- [ ] Commit: `test(page): lock issue-page link API behaviors`

### Task 2: Issue detail UI for linked pages
- [ ] New component or tab; empty state; add picker; remove; click opens Pages route with page id
- [ ] Hide mutate controls without `issue:edit`
- [ ] Vitest
- [ ] Commit: `feat(ui): manage linked pages on issue detail`

### Task 3: Acceptance notes
- [ ] `docs/dev/acceptance/2026-11-06-page-linking.md`
- [ ] Commit: `docs: record page-linking acceptance run table`
