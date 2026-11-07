# Community funnel metrics

Track monthly on the **GitHub** primary repo. Goal: know whether people can find,
try, and contribute — not vanity alone.

| Month | Stars | Forks | Unique clones* | Issue opens | First external PR → merge (days) | Notes |
|-------|------:|------:|---------------:|------------:|---------------------------------:|-------|
| 2026-11 | 10 | 3 | — | — | — | Baseline before v0.2.0 outreach |

\* Insights → Traffic (admin). Record mid-month snapshot.

## How to refresh

```bash
gh repo view vinthuy/reqmango --json stargazerCount,forkCount
gh api repos/vinthuy/reqmango/traffic/clones --jq '.count'
gh issue list --repo vinthuy/reqmango --state all --limit 1 --json number
# First external PR merge latency: review merged PRs where author != maintainers
```

## Targets (directional)

- Stars are lagging; prioritize **clone → Docker up → first Issue**.
- Aim for first external PR merge under **7 days** once someone claims a GFI.
