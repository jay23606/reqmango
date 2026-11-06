# Attachments Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or executing-plans. Steps use `- [ ]`.

**Goal:** Lock P0 attachment behaviors with tests: upload within limit, reject oversize, list/download, delete then 404 download, permission gates, tab count matches list.

**Architecture:** Existing `AttachmentService` + `AttachmentManager.vue`. Add/extend Go tests for size limit and delete; fix UI/count/permission gaps found.

**Tech Stack:** Go, Vue 3, Vitest

**Spec:** [docs/specs/2026-11-06-attachments.md](../../specs/2026-11-06-attachments.md)

### Task 1: Backend tests — size limit, delete→404, permissions
- [ ] Commit: `test(attachment): lock upload size delete and permission rules`

### Task 2: UI — oversize message + count sync + read-only
- [ ] Commit: `fix(ui): harden attachment manager errors and counts`

### Task 3: Acceptance notes
- [ ] `docs/dev/acceptance/2026-11-06-attachments.md`
- [ ] Commit: `docs: record attachments acceptance run table`
