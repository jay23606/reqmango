# Outreach drafts — v0.2.0

Paste-ready copy. Prefer linking **GitHub** Issues/PRs. Demo: local Docker (no public demo yet).

## Show HN

**Title:** Show HN: Reqmango – self-hosted PM where requests are triaged before the backlog

**Body:**

```
I built Reqmango because every self-hosted tracker I tried dumps raw requests
straight into the backlog. We needed a triage step first.

Flow: public Intake form → type / priority / likely-duplicate hints → accept or
reject → only then it becomes a work item. AI (optional API key) helps analyze
and label; it stays inside Issue / Intake / Cycle, not a separate agent product.

Stack: Go + Vue 3 + PostgreSQL. MIT. Multi-arch Docker images on ghcr.io.

Try locally:
  git clone https://github.com/vinthuy/reqmango.git && cd reqmango
  cp .env.example .env && docker compose pull && docker compose up -d
  open http://localhost  (demo@example.com / demo1234)

v0.2.0 adds mandatory/conditional fields, soft-merge duplicates, time tracking
+ estimate points, and more work-item polish.

Repo: https://github.com/vinthuy/reqmango
Docs: https://vinthuy.github.io/reqmango/
```

## Reddit (r/selfhosted)

**Title:** Reqmango – self-hosted project management with Intake triage before backlog

**Body:** short version of the HN body; lead with Docker one-liner; mention MIT + no telemetry; invite issues on GitHub.

## Reddit (r/golang)

**Title:** Reqmango – Go/Gin + Vue self-hosted PM (triage-first Intake)

**Body:** focus on Go backend (GORM, RBAC, tenancy audit CI), link CONTRIBUTING and good-first issues.

## V2EX

**标题：** 开源自建项目管理 Reqmango：需求先分诊再进待办（Go + Vue）

**正文：**

```
做这个的起因是：很多自建看板/Issue 工具把外部提交直接丢进待办，噪音很大。

Reqmango 的主路径是：公开 Intake 表单 → 类型/优先级/疑似重复建议 → 接受或拒绝 → 才进 backlog。
可选配 AI Key 做分析/打标；AI 嵌在 Issue / Intake / Cycle 里，不是默认 Agent 控制台产品。

技术栈：Go 1.25 + Gin、Vue 3、PostgreSQL。MIT。
Docker 两分钟可起：

git clone https://github.com/vinthuy/reqmango.git && cd reqmango
cp .env.example .env && docker compose pull && docker compose up -d
# demo@example.com / demo1234

v0.2.0：必填/条件字段、软合并重复项、工时与估点、子项/附件等。

仓库（Issue/PR 请到 GitHub）：https://github.com/vinthuy/reqmango
文档：https://vinthuy.github.io/reqmango/
国内克隆镜像：https://gitcode.com/yongfeng9m-/reqmanpy
```

## Already posted (2026-11-07)

| Channel | URL |
|---------|-----|
| GitHub Show and tell | https://github.com/vinthuy/reqmango/discussions/39 |
| GitHub Announcements | https://github.com/vinthuy/reqmango/discussions/40 |

## Still needs your login

HN / Reddit / V2EX require interactive login (Cloudflare / 2FA). Paste the sections above after signing in.

## Maintainer notes

- Post HN on a weekday US morning if possible; reply to every top-level comment for 24h.
- Do not cross-post identical text to HN + Reddit same hour; stagger 1–2 days.
- After first external PR merges: thank in CHANGELOG + tweet/Discussions.
