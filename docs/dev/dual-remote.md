# Dual remotes (maintainers)

**One product, one discussion home, two remotes.**

| Role | Host | Audience | Use for |
|------|------|----------|---------|
| **Primary** | [GitHub](https://github.com/vinthuy/reqmango) (`github`) | All contributors (EN / ZH) | Issues, PRs, Discussions, Releases, CI |
| **Mirror (CN clone)** | [GitCode](https://gitcode.com/yongfeng9m-/reqmanpy) (`origin`) | Faster clone in China | Mirror of `master` + tags; **do not** open day-to-day Issues/PRs here |
| **Mirror (optional)** | [Gitee](https://gitee.com/vinthuy/reqmango) (`gitee`) | Users who cannot reach GitHub | Read-only convenience; no CI |

Do **not** run two roadmaps, two issue trackers as equal homes, or two product stories.

## Product sentence (same on both)

- **EN:** Self-hosted project management: triage first; custom issue types, workflows, and automation; AI in Issue / Intake / Cycle — data stays on your machine.
- **ZH:** 自建项目管理：新需求先分诊再进待办；可自定义工作项类型 / 工作流 / 自动化；AI 嵌在 Issue / Intake / Cycle，数据在自己的机器上。

## Sync workflow

After merging to `master` (prefer merge via GitHub PR so CI gates apply):

```bash
git checkout master
git pull github master
git push github master    # source of truth
git push origin master    # GitCode mirror
git push gitee master     # Gitee mirror
```

Tag releases on **GitHub first** (triggers Actions), then mirror the same tag:

```bash
git push github v0.2.0
git push origin v0.2.0
git push gitee v0.2.0
```

## Where contributors should open Issues / PRs

| Contributor | Prefer |
|-------------|--------|
| Everyone | **GitHub** Issues / PRs / Discussions |
| Clone only (CN network) | `git clone` from GitCode is fine; still open PR on GitHub |

If a PR somehow lands on GitCode only, cherry-pick onto GitHub `master` and close the GitCode side with a link — never diverge product scope.

## Branch protection (GitHub primary)

`master` on GitHub is protected. GitCode/Gitee have no equivalent gate.

| Rule | Value | Why |
|------|-------|-----|
| Required checks | `Lint`, `Security`, `Test`, `E2E`, `Tenancy audit`, `Migration path`, `Build`, `Docker Build (backend)`, `Docker Build (frontend)` | Must match job names in `ci.yml` **verbatim** |
| Force push / branch deletion | blocked | History is public and mirrored |
| Admin bypass | allowed | Single maintainer hotfixes |
| Unresolved conversations block merge | yes | |
| Require branch up to date | no | Avoid rebase churn |

```bash
# required-checks.json: {"strict": false, "contexts": ["Lint", ...]}
gh api -X PATCH /repos/vinthuy/reqmango/branches/master/protection/required_status_checks \
  --input required-checks.json
```

## GitHub About / Topics

See [github-public-profile.md](./github-public-profile.md). GitCode About should point visitors to GitHub for Issues/PRs.

## Naming note

GitCode path may still show a historical slug (`reqmanpy`). Prefer **Reqmango** in titles and README; rename on GitCode when convenient.
