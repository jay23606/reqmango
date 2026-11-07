# 参与 Reqmango 贡献

感谢你愿意共建。Reqmango 是**自建项目管理**：新需求先分诊再进待办；AI 嵌在 Issue / Intake / Cycle 里，而不是默认做成独立 Agent IDE。

[English](CONTRIBUTING.md)

## 协作入口

| | 地址 | 用途 |
|---|---|---|
| **GitHub（主战场）** | https://github.com/vinthuy/reqmango | Issue / PR / Discussions / Release |
| **GitCode（镜像）** | https://gitcode.com/yongfeng9m-/reqmanpy | 国内加速克隆；**日常 Issue/PR 请到 GitHub** |

两套仓库**不是**两个产品。讨论与贡献统一到 GitHub。同步说明：[dual-remote.md](docs/dev/dual-remote.md)。

## 开始之前

1. 先读 [README-zh.md](README-zh.md) 的一句话定位。
2. **小步 PR**，避免一次性大重构。
3. 先看 [GitHub Issues](https://github.com/vinthuy/reqmango/issues)，并参考 [good first issues 清单](docs/dev/good-first-issues.md)，避免重复开工。

## 快速环境

### Docker（推荐）

```bash
git clone https://github.com/vinthuy/reqmango.git
# 国内仅克隆可走镜像：git clone https://gitcode.com/yongfeng9m-/reqmanpy.git reqmango
cd reqmango
cp .env.example .env
docker compose up --build
```

打开 http://localhost — `demo@example.com` / `demo1234`。这两个账号来自
`.env.example` 里的 `SEED_DEMO=true`，密码是公开的，只适合本地使用
（见[生产部署](docs/deploy/production.md)）。

可选 AI：在 `.env` 设置 `AI_API_KEY` 后重启。

### 本地开发

- Go **1.25+**、PostgreSQL **16+**、Node.js **24+**
- 后端：`cd backend && go run ./cmd/server/`（`:8000`）
- 前端：`cd frontend && npm install && npm run dev`（http://localhost:5173）

E2E 套件用演示账号登录，所以本地后端需要在 `backend/.env` 里设
`SEED_DEMO=true`（见 README）。不设时服务只创建一个引导管理员，并打印一次随机密码。

账号与环境变量见 README。

## 改哪里

| 方向 | 路径 |
|------|------|
| UI / 文案 | `frontend/src/`，语言包 `frontend/src/locales/{zh-CN,en-US}.json` |
| API / 业务 | `backend/internal/` |
| 部署 | `docker-compose.yml`、前后端 Dockerfile |
| 文档 | `docs/`、根目录 README |

新增界面文案时 **中英 locale 一起改**，用 `t('namespace.key')`，不要让页面露出 `common.refresh` 这种 key。

## 流程

1. Fork **GitHub**，从 `master` 拉分支。
2. 只做与 Issue 相关的改动。
3. 本地验证：
   - 前端：`cd frontend && npx vitest run`（动到类型时再跑 `npx vue-tsc --noEmit`）
- 翻译：`cd frontend && npm run i18n:check`（检查静态 key；动态拼接 key 需另行验证）

编号并发回归：设置 PostgreSQL 连接变量 `TEST_DATABASE_URL`，运行 `cd backend && go test ./internal/service -run TestIssueSequence`。测试会创建并清理独立 schema，数据库角色需有创建 schema 的权限；未设置变量时跳过，CI 已配置。

E2E 必须使用临时 PostgreSQL 数据库，不要连接生产环境：测试会创建账户并修改种子数据。Intake fixture 会删除工作区，但注册账户保留到测试库被丢弃；CI 会在每个任务结束后丢弃数据库。
   - 后端：`cd backend && go test ./internal/...`
   - 手动走一遍复现路径或主流程
4. 在 **GitHub** 按 PR 模板提 PR，并关联 Issue。

CI 见 `.github/workflows/ci.yml`，请保持绿色。

## 欢迎的贡献

- 有复现步骤的缺陷修复
- i18n / 文档 / Docker 与 DX
- Issue、Intake、Cycle、Pages 上的小体验改进
- 补测试

## 请先讨论再做的

- 把 Agent 控制台 / 市场做成默认主叙事的大改
- 无说明的破坏性 API 变更
- 无功能需求的依赖大升级
- 提交密钥、真实 `.env`、大体积二进制

## 行为准则

参与本项目即表示同意遵守 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)（Contributor Covenant 2.1）。举报发到 **vinthuy@qq.com**，标题加 `[conduct]`。

## 提问与求助

安装问题、“这算 bug 吗”、使用咨询请看 [SUPPORT.md](SUPPORT.md) 或 GitHub Discussions，不要开 Issue。安全问题只走 [SECURITY.md](SECURITY.md)。

## 维护者

双远程同步说明：[docs/dev/dual-remote.md](docs/dev/dual-remote.md)

发布流程（打 tag → 多架构镜像 → GitHub Release）：[docs/dev/release.md](docs/dev/release.md)

## License

贡献内容按 [MIT License](LICENSE) 授权。
