# Knowledge Base

The KB is the **single source of truth** for the reqmango system, always describing the current actual state.

**Last Updated**: 2026-11-07

## Maintenance Principles

- **Timeliness**: when a feature ships, the corresponding KB document is updated in the same change
- **Accuracy**: the KB describes what the code actually does, not what was intended at design time
- **Conciseness**: the KB is a reference, not a development log. Keep the core, drop the procedure

## KB Update Triggers

| Event | Update scope |
|-------|--------------|
| New feature completed | `architecture/README.md` module table + `data-model.md` tables + `frontend.md` |
| API changes | `api-conventions.md` |
| Data model changes | `data-model.md` |
| Tech stack upgrade | `tech-stack.md` + `architecture/README.md` |

## KB Document List

### Product layer
- [PRD.md](PRD.md) — Product requirements (Chinese, canonical) · [English](PRD.en.md)

### Architecture layer
- [architecture/README.md](architecture/README.md) — Architecture overview + module status
- [architecture/project-layout.md](architecture/project-layout.md) — Project directory structure (Chinese only)
- [architecture/backend-go.md](architecture/backend-go.md) — Go backend architecture
- [architecture/frontend.md](architecture/frontend.md) — Frontend architecture (views / components / API modules)
- [architecture/data-model.md](architecture/data-model.md) — Data model (the 131 tables created by AutoMigrate)
- [architecture/api-conventions.md](architecture/api-conventions.md) — API design conventions
- [architecture/tech-stack.md](architecture/tech-stack.md) — Tech stack details
- E2E tests — Playwright suites under `frontend/e2e/`, covering the main path and core customization

### Feature design documents

These are Chinese only; there are no English siblings yet.
- [architecture/saved-views-design.md](architecture/saved-views-design.md) — Saved views
- [architecture/pages-design.md](architecture/pages-design.md) — Pages/Wiki
- [architecture/notification-design.md](architecture/notification-design.md) — Notifications
- [architecture/type-hierarchy-template-design.md](architecture/type-hierarchy-template-design.md) — Type hierarchy and templates
- [architecture/relation-system-design.md](architecture/relation-system-design.md) — Relation types

### Changelog
- [changelog/README.md](changelog/README.md) — KB update history

---

## 🌐 Language

- **English** (this document)
- [中文文档](README.md)
