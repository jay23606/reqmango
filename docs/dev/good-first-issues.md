# Good first issues (curated)

Starter tasks for new contributors. Each item is sized for roughly **0.5–3 hours**.

**How to claim:** Open or claim the task on **GitHub** (primary). Put the ID in the title (e.g. `[good first] GFI-03 TriagePanel i18n`) and comment that you are working on it. One claim per person at a time. See [dual-remote.md](./dual-remote.md).

**Product north star:** self-hosted PM — triage before backlog; AI inside Issue / Intake / Cycle. Prefer these paths over Agent-console sprawl.

**Still open:** GFI-04, GFI-07, GFI-08, GFI-10. The rest of this list is done on `master`.

---

## GFI-01 — Record and add Intake demo GIF

| | |
|---|---|
| **Area** | docs |
| **Status** | **Done** — GitHub issue #1 closed |
| **Files** | `docs/assets/demo.gif` (new), `docs/assets/README.md`, root `README.md` / `README-zh.md` |
| **Task** | Record a 15–30s clip: submit Intake → triage queue (type / priority / duplicate hints if AI key set). Keep under ~5 MB. Link it in both READMEs above the fold. |
| **Done when** | GIF renders on GitHub README; walkthrough text still works without AI key. |

## GFI-02 — GitHub Topics / About checklist in docs

| | |
|---|---|
| **Area** | docs |
| **Status** | **Done** — GitHub issue #11 closed |
| **Files** | `docs/dev/github-public-profile.md` (new) |
| **Task** | Document recommended repository About blurb (EN + ZH) and Topics: `self-hosted`, `project-management`, `issue-tracker`, `vue`, `golang`, `ai`. Maintainers apply in the GitHub UI. |
| **Done when** | Doc exists and is linked from `CONTRIBUTING.md`. |

## GFI-03 — TriagePanel hardcoded English → i18n

| | |
|---|---|
| **Area** | frontend / i18n |
| **Status** | **Done** on `master` (maintainer demo) — GitHub issue #10 closed |
| **Files** | `frontend/src/components/TriagePanel.vue`, locales under `intake.*` |

## GFI-04 — PageTabConfig built-in names via locale

| | |
|---|---|
| **Area** | frontend / i18n |
| **Files** | `frontend/src/components/PageTabConfig.vue` |
| **Task** | Stop storing Chinese literals in `builtInTabs[].name`. Use `pageTab.builtInTabs.*` when toggling/saving (keys already exist). Display already uses `getTabDisplayName`. |
| **Done when** | Enabling a built-in tab persists a stable type; UI label follows locale. |

## GFI-05 — useIssueFilters chip labels i18n

| | |
|---|---|
| **Area** | frontend / i18n |
| **Status** | **Done** on `master` — GitHub #3 and GitCode #3 closed |
| **Files** | `frontend/src/composables/useIssueFilters.ts`, locales |
| **Task** | Replace hardcoded `状态` / `优先级` / `紧急`… with `t()` (e.g. `common.*` / priority keys). |
| **Done when** | Filter chips follow locale; vitest for the composable still pass or are updated. |

## GFI-06 — Priority / state group label helpers i18n

| | |
|---|---|
| **Area** | frontend / i18n |
| **Status** | **Done** on `master` — GitHub #4 and GitCode #4 closed |
| **Files** | `frontend/src/types/issue.ts` (`getPriorityName`, `getStateGroupName`), call sites as needed |
| **Task** | Make labels locale-aware (pass `t` or read `useI18n` from a thin helper). Do not leave Chinese-only maps for UI. |
| **Done when** | Priority/state group names switch with locale in list/kanban/settings samples you touch. |

## GFI-07 — ProjectSettings toast errors i18n

| | |
|---|---|
| **Area** | frontend / i18n |
| **Files** | `frontend/src/views/ProjectSettings.vue`, locales |
| **Task** | Replace English toast fallbacks (`Failed to load settings data`, etc.) with `t('…')` keys. |
| **Done when** | Forced error path shows translated toast in zh and en. |

## GFI-08 — Router meta titles i18n note or fix

| | |
|---|---|
| **Area** | frontend |
| **Files** | `frontend/src/router/index.ts`, document title setter if any |
| **Task** | Hardcoded Chinese `meta.title` (`工作流`, `预算与SLA`, …). Either drive document title through `t()` or document that titles are zh-only and add en keys + wiring. Prefer wiring. |
| **Done when** | Browser tab title follows locale for the routes you change. |

## GFI-09 — Scan script in npm scripts

| | |
|---|---|
| **Area** | DX |
| **Status** | **Done** on `master` — GitHub #7 and GitCode #7 closed |
| **Files** | `frontend/package.json`, `frontend/scripts/scan-missing-i18n-keys.mjs` |
| **Task** | Add npm script e.g. `i18n:check` that runs the missing-key scanner and exits non-zero if real gaps remain (ignore dynamic `pageTab.builtInTabs.` / `plugin.` prefixes). |
| **Done when** | `npm run i18n:check` documented in CONTRIBUTING. |

## GFI-10 — `.env.example` contributor comments

| | |
|---|---|
| **Area** | docs / DX |
| **Files** | `.env.example` |
| **Task** | Ensure comments cover: ports, demo login pointer, optional `AI_*`, and “never commit real keys”. Keep bilingual comments short or EN-only with README-zh pointer. |
| **Done when** | New contributor can configure Compose without reading backend code. |

## GFI-11 — Healthcheck / wait message for Compose

| | |
|---|---|
| **Area** | docker |
| **Status** | **Done** — GitHub #9 closed; GitCode #9 closed |
| **Files** | `docker-compose.yml`, maybe README |
| **Task** | Confirm frontend does not serve before backend is ready (or document retry). Optional: backend health endpoint already used by deps — improve README “first boot takes a few minutes” note. |
| **Done when** | README states expected first-build time; no silent blank page without guidance. |

## GFI-12 — CONTRIBUTING link from README (if missing)

| | |
|---|---|
| **Area** | docs |
| **Status** | **Done** — GitHub issue #12 closed |
| **Files** | `README.md`, `README-zh.md` |
| **Task** | Contributing section should link `CONTRIBUTING.md` / `CONTRIBUTING-zh.md` and `docs/dev/good-first-issues.md`. |
| **Done when** | Links work on GitHub. |

---

## Maintainer checklist when publishing these as GitHub Issues

For each GFI you open on GitHub:

1. Title: `[good first] GFI-XX …`
2. Labels: `good first issue`, `help wanted`, plus `area:frontend` / `area:backend` / `area:docs`
3. Paste the table row + acceptance into the body
4. Assign only after someone comments “I want this”
