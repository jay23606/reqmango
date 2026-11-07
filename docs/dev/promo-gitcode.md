# GitCode 镜像说明（历史推广稿）

> **2026-11-07 起：GitHub 为协作主战场**（Issue / PR）。GitCode 仅作国内克隆镜像。  
> 新发帖文案见 [outreach-v0.2.md](./outreach-v0.2.md)。下文为历史粘贴稿，勿再把 GitCode 写成「主场」。
>
> 承接旧策略：**GitCode 曾作国内共建主场**；外站已发的不再追。  

> 仓库：https://gitcode.com/yongfeng9m-/reqmanpy

## 本周在 GitCode 上做这 4 件

| # | 动作 | 材料 |
|---|------|------|
| 1 | **About 简介**（设置页） | 见下方「About」 |
| 2 | **置顶欢迎 Issue** | 正文：`docs/dev/gitcode-welcome-issue.md` |
| 3 | **批量开 GFI Issue** | `scripts/create-gitcode-gfi-issues.mjs`（需 PAT） |
| 4 | **发一条仓库动态** | 见下方「动态」 |

## About（设置 → 仓库简介）

```
自建项目管理：新需求先 Intake 分诊再进待办；可自定义工作项类型 / 工作流 / 自动化；AI 嵌在 Issue·Intake·Cycle，数据在自己机器上。
```

Topics（若支持）：`self-hosted` `project-management` `vue` `golang` `ai` `docker`

展示名尽量用 **Reqmango**（路径仍可为 `reqmanpy`）。

## 动态（粘贴到 GitCode 动态 / 讨论）

已发布到公告讨论 #1（2026-10-31 更新试用命令）：https://gitcode.com/yongfeng9m-/reqmanpy/discussions/1

**标题：** 自建 PM 开源共建：先分诊再进 backlog，类型 / 工作流 / 自动化可配

**正文：**

```
很多团队熟悉「可定制项目管理」（工作项类型、工作流、自动化），也会卡在云端成本和数据驻留，以及入口需求靠人肉分诊。

Reqmango 开源自建：
· Intake 分诊：类型 / 优先级 / 疑似重复建议，拿不准再进 backlog
· 自定义工作项类型 · 工作流 · 自动化
· AI 嵌在 Issue / Intake / Cycle；自备 Key，数据在自己机器上

一键试用（已发布镜像）：
git clone https://gitcode.com/yongfeng9m-/reqmanpy.git reqmango
cd reqmango && cp .env.example .env && docker compose pull && docker compose up -d
→ http://localhost  · demo@example.com / demo1234

欢迎提：
「Jira 里这样配，这里还缺什么」——直接开 Issue。
还开放的上手任务：GFI-04 / 07 / 08 / 10（docs/dev/good-first-issues.md）
贡献指南：CONTRIBUTING-zh.md
开源中国：https://www.oschina.net/news/503197
国际镜像：https://github.com/vinthuy/reqmango
```

## 批量开 Issue（PAT）

GitCode → 个人设置 → Access Tokens（scopes: **projects** + **issues**）：

```powershell
$env:GITCODE_TOKEN = '<你的 PAT>'
node scripts/create-gitcode-gfi-issues.mjs
```

脚本会创建欢迎 Issue + GFI-01～（与清单对齐），已存在同名标题会跳过。

## 渠道边界（避免分心）

| 平台 | 状态 |
|------|------|
| GitCode | **主场，持续** |
| 开源中国 | **资讯已发** https://www.oschina.net/news/503197 （2026-10-25），不要再投 |
| V2EX / 掘金 | 已发，不追加 |
| 思否 / Linux.do | 暂缓 |

## 验收（你这边点开能看到）

- [x] About 已是产品一句话（中文，API 已改）  
- [x] 「欢迎共建」Issue：https://gitcode.com/yongfeng9m-/reqmanpy/issues/10（已置顶）  
- [x] GFI #2–#9 开放；#1 已关（demo.gif 已有）  
- [x] 公告讨论：自建 PM 开源共建（Discussion #1，试用命令已写）  
- [x] 第一个外部贡献者 xiapdo（MR !1）已合并 + Issue #10 致谢（2026-09-27）  
- [x] demo 登录修复上线，Issue #10 已同步公告（2026-09-27）  
