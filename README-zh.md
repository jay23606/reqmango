# Reqmango

[![CI](https://github.com/vinthuy/reqmango/actions/workflows/ci.yml/badge.svg)](https://github.com/vinthuy/reqmango/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/vinthuy/reqmango?sort=semver)](https://github.com/vinthuy/reqmango/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![容器镜像](https://img.shields.io/badge/ghcr.io-reqmango-2496ED?logo=docker&logoColor=white)](https://github.com/vinthuy/reqmango/pkgs/container/reqmango-backend)

**自建项目管理：新需求先分诊，再进待办。**  
提交 → 类型 / 优先级 / 疑似重复建议 → 拿不准的才由人决定。AI 嵌在 Issue / Intake / Cycle，数据在自己的机器上。

[English](README.md) · [文档站](https://vinthuy.github.io/reqmango/) · [贡献指南](CONTRIBUTING-zh.md) · [Issues](https://github.com/vinthuy/reqmango/issues) · [GitCode 镜像](https://gitcode.com/yongfeng9m-/reqmanpy)

![分诊队列](docs/assets/screenshot-02-triage-queue.png)

<sub>Intake 表单 → 分诊队列：接受 / 拒绝之后才进待办。</sub>

---

## 快速试用（约 2 分钟）

```bash
git clone https://github.com/vinthuy/reqmango.git && cd reqmango
# 国内加速克隆：git clone https://gitcode.com/yongfeng9m-/reqmanpy.git reqmango && cd reqmango
cp .env.example .env && docker compose pull && docker compose up -d
```

等两个容器 healthy 后打开 **http://localhost** —

| | |
|---|---|
| 邮箱 | `demo@example.com` |
| 密码 | `demo1234` |

可选 AI（分诊 / 分析 / 标签）：在 `.env` 设置 `AI_API_KEY` 后再次 `docker compose up -d`。

也可用 `docker compose up --build` 本地构建（冷缓存几分钟）。`curl localhost/version` 可看当前版本。

<table>
  <tr>
    <td width="50%"><img src="docs/assets/screenshot-01-intake-form.png" alt="需求提交表单"><br><sub><b>需求入口</b> — 一个公开链接，收没有账号的人提的需求</sub></td>
    <td width="50%"><img src="docs/assets/screenshot-03-issues.png" alt="工作项列表"><br><sub><b>工作项</b> — 列表 / 看板 / 树形 / 日历 / 甘特，共用同一套筛选</sub></td>
  </tr>
  <tr>
    <td><img src="docs/assets/screenshot-04-issue-detail.png" alt="工作项详情"><br><sub><b>工作项详情</b> — 属性、关联、动态与 AI 面板在一屏内</sub></td>
    <td><img src="docs/assets/screenshot-05-cycles.png" alt="周期"><br><sub><b>周期</b> — Sprint 范围、进度和延期项一目了然</sub></td>
  </tr>
  <tr>
    <td><img src="docs/assets/screenshot-06-analytics.png" alt="分析"><br><sub><b>分析</b> — 优先级分布、工作量与完成趋势</sub></td>
    <td><img src="docs/assets/screenshot-07-workspace.png" alt="工作空间"><br><sub><b>工作空间</b> — 多个项目、一个团队、一次登录</sub></td>
  </tr>
</table>

界面首次访问会跟随浏览器语言（中文 / English），任何页面右上角都能切换。

### 一分钟走查

1. 使用演示账号登录。
2. 打开 **Demo** 项目 → **项目设置** → **请求分诊（Intake）**。
3. 打开 Intake 表单链接，提交一条简短需求（例如「手机端登录偶尔失败」）。
4. 回到分诊队列：接受 / 拒绝；配置了 `AI_API_KEY` 时可对工作项做 AI 分析与标签建议。

---

## 能力一览

| 方向 | 能力 |
|------|------|
| **工作流里的 AI** | Intake 分诊、Issue 分析、标签建议、Cycle / Sprint 总结、Ask / Build Copilot（写入前先预览） |
| 工作项 | 列表 + 看板、状态流转、最多 6 层层级 |
| 自定义字段与模板 | 文本 / 数字 / 下拉 / 布尔 / 日期 / 成员 / URL；工作区类型蓝图；项目模板 |
| 工作流与自动化 | 状态转换、审批、角色限制；触发器 → 条件 → 动作 |
| 关联与搜索 | Blocks / Relates / Duplicates；RQL 与多字段筛选 |
| 通知与安全 | 未读计数；XSS 清洗（bluemonday）；JWT + RBAC |
| API | 100+ REST 接口 |

AI 是项目管理上的一层能力，默认叙事不是独立的 Agent 控制台。代码里可能仍有高级 Agent 路由；日常路径是 Issue、Cycle、Intake 与 Pages。

---

## 环境要求

| 方式 | 需要 |
|------|------|
| **Docker（推荐）** | Docker + Docker Compose |
| 本地开发 | Go **1.25+**、PostgreSQL **16+**、Node.js **24+** |

版本与 CI、`docker-compose.yml` 一致（`postgres:16`，Go 1.25 / Node 24 构建镜像）。

---

## 本地开发（不用 Docker）

```bash
# 数据库
psql -U postgres -c "CREATE DATABASE reqmango;"

# 后端
cd backend
cat > .env << EOF
DATABASE_URL=postgres://postgres:postgres@localhost:5432/reqmango?sslmode=disable
SECRET_KEY=$(openssl rand -hex 32)
ACCESS_TOKEN_EXPIRE_MINUTES=10080
PORT=8000
DEBUG=true
SEED_DEMO=true
EOF
go run ./cmd/server/
```

服务启动时会自动迁移并写入种子数据，因此新建环境只需这一步。`SEED_DEMO=true` 会创建演示账号；不需要演示数据时删掉这一行，首次启动会创建一个管理员并把随机密码打印在日志里（见[生产部署](docs/deploy/production.md)）。

### 数据库迁移

若希望把建表作为一个独立、可审阅的步骤，用 `cmd/migrate`：

```bash
cd backend
go run ./cmd/migrate up      # 应用基线
go run ./cmd/migrate version # 1, dirty=false
```

`migrations/000001_baseline.up.sql` 是单一幂等基线，空库和「已由服务 AutoMigrate
建好的库」都能直接跑。压缩基线之前建好的旧库需要一次性
`go run ./cmd/migrate force 1`，详见[债务说明](docs/dev/debt/2026-10-18-sql-migration-chain.md)。

种子数据含 `demo@example.com` / `demo1234`、工作区 `reqmango-dev`（项目 `CORE` / `MOBILE` / `AI` / `OPENAPI`），以及示例 Sprint / 模块 / 工作项。演示账号的密码是公开的，**只适合本地试用**。

```bash
# 前端（API 在 :8000）
cd frontend
npm install
npm run dev
```

打开 **http://localhost:5173**。

### 本地生产构建

```bash
cd backend && go build -o server ./cmd/server/
cd frontend && npm run build
```

---

## 目录结构

```
reqmango/
├── backend/     # Go + Gin + GORM API
├── frontend/    # Vue 3 + TypeScript + Vite
├── sdk/         # MCP / CLI（可选，非默认产品叙事）
├── docs/        # API、架构、产品规格
└── docker-compose.yml
```

---

## 文档

同样的内容有在线站点：[vinthuy.github.io/reqmango](https://vinthuy.github.io/reqmango/)（[English](https://vinthuy.github.io/reqmango/en/)），由 MkDocs 从 `docs/` 构建。

- [API 参考](docs/API.md)
- [架构文档](docs/kb/architecture/) — [技术栈](docs/kb/architecture/tech-stack.md)、[Go 后端](docs/kb/architecture/backend-go.md)、[前端](docs/kb/architecture/frontend.md)、[数据模型](docs/kb/architecture/data-model.md)
- [生产部署](docs/deploy/production.md) — 密钥、关闭演示数据、TLS、备份
- [配置项](docs/configuration.md) — 全部环境变量与默认值

---

## 贡献

欢迎提交 Issue 与 Pull Request，尤其是小而清晰的改动。

- [贡献指南](CONTRIBUTING-zh.md) · [English](CONTRIBUTING.md)
- [路线图](ROADMAP.md) — 接下来做什么，以及明确不做什么
- [支持与提问](SUPPORT.md) — 哪类问题去哪里问
- [行为准则](CODE_OF_CONDUCT.md) — Contributor Covenant 2.1
- [适合新手的任务清单](docs/dev/good-first-issues.md)
- [安全策略](SECURITY.md) — 漏洞请走私密渠道，不要开公开 Issue
- **GitHub** 是协作主战场 · [GitCode](https://gitcode.com/yongfeng9m-/reqmanpy) 仅作克隆镜像（[双远程说明](docs/dev/dual-remote.md)）

## License

[MIT](LICENSE)
