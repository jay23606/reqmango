# Notification Auto-Trigger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or executing-plans. Steps use `- [ ]`.

**Goal:** Ensure P0 events (assign, comment, state change, mention) create notifications for correct recipients, never self-notify, and unread list/count work.

**Architecture:** Audit `TriggerNotification` call sites in issue/comment services; fix gaps; add focused service tests. Skip P1 `issue_due_soon` E2E unless already trivial.

**Tech Stack:** Go, Vitest optional for notification center

**Spec:** [docs/specs/2026-11-06-notification-auto-trigger.md](../../specs/2026-11-06-notification-auto-trigger.md)

### Task 1: Audit + fix missing triggers / self-notify
- [ ] Compare SRS event table to `issue_service.go` / `comment_service.go` / `notification_service.go`
- [ ] Fix gaps; ensure actor excluded from recipients
- [ ] Commit: `fix(notification): complete P0 auto-triggers and no self-notify`

### Task 2: Service tests for assign/comment/state/mention + unread
- [ ] Commit: `test(notification): lock P0 auto-trigger acceptance`

### Task 3: Acceptance notes
- [ ] `docs/dev/acceptance/2026-11-06-notification-auto-trigger.md`
- [ ] Commit: `docs: record notification-auto-trigger acceptance run table`
