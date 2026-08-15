# AIChatsHub 💬🤖

<p align="right"><a href="README.md">English</a> | <b>简体中文</b></p>

> **开源、可自托管的 AI 客服对话与获客平台**

**AIChatsHub** 是一个一体化、可自托管的 AI 客服平台。连接你的 WhatsApp 商业号码，
上传自定义知识库，让智能 AI 助手 7×24 小时处理咨询、展示你的产品目录并捕获优质销售
线索——还支持无缝转接真人客服。

采用 **Go (Echo)** + **React (Vite)** + **PostgreSQL** 构建。自带凭据：没有任何
硬编码密钥，仅需配置一个数据库即可顺利运行——每一项集成（AI、邮件、社交登录、计费）
都会在你填入对应密钥的那一刻自动启用。

> **提示：** 本仓库**不含任何密钥**。每个部署都通过 `.env` 文件（已被 git 忽略）
> 提供自己的密钥。参见[配置](#配置)。

> 💼 **不想自己搭建？** 软件永久免费——但如果你不想自己管理服务器，我提供付费的
> **安装部署、托管运维和维护**服务。参见[服务与支持](#服务与支持)或发邮件至
> [hello@awkiss.com](mailto:hello@awkiss.com)。

---

## 功能特性

- **AI 自动回复**——Gemini 根据你上传的问答知识库和产品目录回答客户消息。
- **WhatsApp 集成**——通过二维码关联一个或多个号码。由第三方开源库
  [whatsmeow](https://github.com/tulir/whatsmeow)（一个 Go 语言的 WhatsApp Web
  多设备客户端）驱动——无需官方 WhatsApp Business API 或付费网关。
- **真人客服工作台**——通过 WebSocket 实时聊天，可从机器人手中接管对话、为对话打
  标签、拉黑联系人。
- **产品目录与线索**——导入产品，机器人会主动推荐并捕获销售线索。
- **认证**——邮箱/密码登录（可选邮箱验证），以及可选的 Google / Facebook 社交登录。
- **可选计费**——基于 Stripe 的“按连接设备数”订阅，默认关闭（自托管时完全免费）。
- **管理后台**——管理用户、设备、订阅以及全局机器人提示词。

## 技术栈

| | |
|---|---|
| **前端** | React 19 · TypeScript · Vite 7 · Ant Design · Zustand · Axios |
| **后端** | Go 1.26 · Echo v4 · GORM · JWT · WebSocket |
| **数据库** | PostgreSQL 16（`pg_trgm` 扩展） |
| **AI** | Google Gemini（默认 `gemini-3.7-flash`） |
| **WhatsApp** | whatsmeow（会话存储于 SQLite） |
| **计费** | Stripe（可选） |

---

## 快速开始（Docker）

运行整个技术栈——Postgres、后端和前端——最快的方式是 Docker Compose。你需要
[Docker Desktop](https://www.docker.com/products/docker-desktop/)（或 Docker
Engine + Compose v2）。

```bash
git clone https://github.com/xgtechshow518/aichathubs.git
cd aichathubs

# 1. 从模板创建你的后端配置
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env

# 2.（可选）编辑 backend/.env，填入你的 Gemini/SMTP/OAuth/Stripe 密钥。
#    不填也能运行——参见下方“优雅降级”。

# 3. 构建并启动全部服务
docker compose up --build
```

然后打开：

| 服务 | 地址 |
|---|---|
| 前端 | http://localhost:5173 |
| 后端 API | http://localhost:8080 |
| Postgres | localhost:5432（用户名/密码 `postgres`） |

Docker Desktop 会显示一个 **`aichathubs`** 分组，包含三个子容器：
`aichathubs-db`、`aichathubs-backend`、`aichathubs-frontend`。

用 `Ctrl+C` 停止，或运行 `docker compose down`（加 `-v` 可同时清除数据库和
WhatsApp 会话数据卷）。

### 首次登录

- **应用账号**——打开前端并**注册**。在未配置 SMTP（默认情况）时，新注册会被
  **自动验证**，因此你会立即登录进入工作台。系统没有预置用户。
- **管理后台**——访问 `/admin/login`。凭据来自 `backend/.env`；内置默认值为
  **`admin` / `admin123456`**（`ADMIN_USERNAME` / `ADMIN_PASSWORD`）。
  **在对外暴露服务器之前务必修改它们**——只要仍是默认值，后端启动时就会打印警告。

### 端口已被占用？

如果本机上的 `5432`、`8080` 或 `5173` 已被占用，复制根目录的端口模板并修改它们——
Compose 会自动读取：

```bash
cp .env.example .env   # 编辑 DB_PORT / BACKEND_PORT / FRONTEND_PORT
```

---

## 配置

所有配置均通过环境变量完成。共有两个 `.env` 文件，各自都有一份已提交的
`.env.example` 模板：

- **`backend/.env`**——最重要的一个：数据库、密钥以及所有集成密钥。完整的带注释
  清单见 [`backend/.env.example`](backend/.env.example)。
- **`frontend/.env`**——浏览器端的公开配置（API 地址、Google 客户端 ID）。见
  [`frontend/.env.example`](frontend/.env.example)。
- **`.env`**（根目录，可选）——仅用于 Docker Compose 对外发布的端口。

### 必填 vs 可选

| 变量 | 是否必填？ | 未设置时的影响 |
|---|---|---|
| `DATABASE_URL` | ✅ 必填 | 应用无法启动 |
| `JWT_SECRET` | ✅ 请设置强随机值 | 使用不安全的默认值（启动时告警） |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | ✅ 请修改 | 使用不安全的默认值（启动时告警） |
| `GEMINI_API_KEY` | 可选 | AI 自动回复被禁用 |
| `SMTP_*` | 可选 | 新用户自动验证（不发送邮件） |
| `GOOGLE_CLIENT_ID` / `_SECRET` | 可选 | 隐藏“使用 Google 登录” |
| `FACEBOOK_CLIENT_ID` / `_SECRET` | 可选 | 隐藏“使用 Facebook 登录” |
| `BILLING_ENABLED` + `STRIPE_*` | 可选 | 免费模式：设备数不限，无付费墙 |

### 优雅降级

本应用设计为可以在**最小**配置下运行，并随着你添加密钥而逐步点亮各项功能。启动时
后端会明确记录哪些功能已开启：

```
──────────── AIChatsHub configuration ────────────
  AI assistant (Gemini):  disabled
  Email verification:     disabled
  Google login:           disabled
  Facebook login:         disabled
  Billing (Stripe):       disabled
  → No SMTP: new users are auto-verified on signup (no email sent).
  → Billing off: unlimited WhatsApp devices, no subscription gate.
──────────────────────────────────────────────────
```

前端会请求 `GET /api/config`（仅返回布尔值，不含任何密钥）来决定显示哪些登录按钮。

### 各项凭据获取地址

| 密钥 | 获取地址 |
|---|---|
| **Gemini API 密钥** | https://aistudio.google.com/apikey |
| **Google OAuth** | https://console.cloud.google.com/apis/credentials —— 添加重定向 URI `http://localhost:5173/login` 和 `http://localhost:8080/api/auth/google/callback` |
| **SMTP（Gmail）** | 在 https://myaccount.google.com/apppasswords 创建应用专用密码 |
| **Stripe** | https://dashboard.stripe.com/apikeys（密钥 + 一个订阅 Price ID + Webhook 密钥） |

---

## 计费模式

- **免费（默认）**——`BILLING_ENABLED=false`。无需 Stripe。用户可连接的 WhatsApp
  设备数不限，享有完整访问权限；订阅门槛关闭。
- **付费**——`BILLING_ENABLED=true` + Stripe 密钥 + 一个 `STRIPE_PRICE_STARTER`
  price ID。按连接的 WhatsApp 设备数向用户收费（Stripe 订阅数量），并在连接时强制
  执行设备数上限。

---

## 手动部署（不使用 Docker）

<details>
<summary>前置要求：Go 1.26+、Node.js 20.19+（或 22.12+）、PostgreSQL 16+</summary>

**数据库**

```sql
CREATE DATABASE smart_live_chats;
```

**后端**

```bash
cd backend
cp .env.example .env      # 将 DATABASE_URL 设置为你本地的 Postgres
go mod download
go run ./cmd/server       # 服务运行于 http://localhost:8080
```

**前端**

```bash
cd frontend
cp .env.example .env      # 设置 VITE_API_URL=http://localhost:8080
npm install
npm run dev               # 服务运行于 http://localhost:5173
```

</details>

---

## 管理后台

独立的管理后台登录入口位于 `/admin/login`（前端），由 `ADMIN_USERNAME` /
`ADMIN_PASSWORD` 支撑。你可以在这里管理用户、设备、订阅、全局机器人系统提示词，
以及 AI 回复延迟设置。

📖 完整的各功能模块说明请见 **[管理后台指南](docs/admin-guide.md)**（英文）。

---

## 文档

- **[管理后台指南](docs/admin-guide.md)**（英文）——操作管理后台：用户、设备、订阅、
  机器人提示词、回复延迟以及系统健康状况。
- **[配置](#配置)**——所有环境变量及各项凭据的获取地址。
- **[贡献指南](CONTRIBUTING.md)**（英文）· **[行为准则](CODE_OF_CONDUCT.md)**（英文）

---

## 项目结构

```
aichathubs/
├── backend/                  # Go (Echo) API
│   ├── cmd/server/           # 入口
│   ├── internal/
│   │   ├── handlers/         # HTTP + WebSocket 处理器
│   │   ├── models/           # GORM 模型
│   │   ├── gemini/           # AI 服务
│   │   ├── whatsapp/         # whatsmeow 管理器
│   │   ├── middleware/       # JWT、管理员、订阅
│   │   ├── database/         # 数据库连接 + 迁移
│   │   ├── email/            # SMTP 验证码
│   │   └── config/           # 环境配置 + 启动摘要
│   ├── Dockerfile
│   └── .env.example
├── frontend/                 # React (Vite) 应用
│   ├── src/{pages,components,services,store,types}
│   ├── Dockerfile.dev
│   └── .env.example
├── docker-compose.yml
├── .env.example              # Docker 端口覆盖
└── LICENSE
```

---

## 安全

- **切勿提交 `.env` 文件。** 它们已被 git 忽略；只有 `.env.example` 模板会被跟踪。
- 在对外暴露服务器之前，请把 `JWT_SECRET` 和 `ADMIN_PASSWORD` 从默认值改掉
  （只要仍是默认值，后端启动时就会告警）。
- 任何曾经被提交过的密钥都应视为已泄露——请立即轮换。

## 服务与支持

**AIChatsHub 免费且开源——并将永远如此。** 你可以按照上面的指南零成本自行托管。

话虽如此，部署并运营一个生产级的聊天平台（服务器、域名、SSL、WhatsApp 关联、
API 密钥、更新、备份）是要花时间的。如果你想省去搭建过程，或者需要有人帮你维护，
我提供付费服务：

| 服务 | 涵盖内容 |
|---|---|
| **安装部署** | 在你的服务器或 VPS 上部署——域名、SSL、WhatsApp 关联、Gemini/SMTP/OAuth 密钥接入并测试 |
| **托管运维** | 我帮你托管并运营——更新、备份、监控、可用性保障 |
| **维护与更新** | 保持补丁与版本最新：依赖升级、安全修复、故障修复 |
| **定制开发** | 为你的业务量身打造新功能、自定义品牌与集成 |
| **优先支持** | 更快响应的直接协助 |

联系方式：

📧 **[hello@awkiss.com](mailto:hello@awkiss.com)**

> 自托管始终 100% 免费。付费服务只是为不想自己管理服务器的团队提供的可选便利。

## 参与贡献

欢迎贡献！Bug 反馈、功能建议和 Pull Request 都很有帮助。开始之前请阅读
[贡献指南](CONTRIBUTING.md)（英文）——其中涵盖工作流程、代码检查，以及我们对提交
所要求的轻量级签署（DCO）。所有参与均受我们的[行为准则](CODE_OF_CONDUCT.md)
（英文）约束。

## 赞助

如果本项目为你节省了时间，并且你愿意支持它的持续开发，我们非常感谢你的赞助
（但这从不是必须的）：

- ❤️ [GitHub Sponsors](https://github.com/sponsors/xgtechshow518)
- ☕ [Buy Me a Coffee](https://buymeacoffee.com/xgtechshowl)

---

## 致谢与第三方声明

WhatsApp 的连接能力完全由第三方开源库
**[whatsmeow](https://github.com/tulir/whatsmeow)**（MPL-2.0）提供，这是一个
非官方的 Go 语言 WhatsApp Web 多设备客户端。本项目**与 WhatsApp 或 Meta 没有任何
关联，未获其认可或赞助**。“WhatsApp”是 Meta Platforms, Inc. 的商标。

使用非官方客户端可能违反 WhatsApp 的服务条款，并可能导致你的号码被限流或封禁。
如何使用由你自行负责——建议使用专用/测试号码，并遵守所有适用法律及 WhatsApp 条款。

其他值得一提的开源依赖：[Echo](https://echo.labstack.com/)、
[GORM](https://gorm.io/)、[React](https://react.dev/)、
[Ant Design](https://ant.design/) 以及
[Google Gemini API](https://ai.google.dev/)。

## 许可证

[MIT](LICENSE) © 2026 AIChatsHub

---

> ⚠️ 本中文文档为翻译版本，可能滞后于英文原文。如有出入，请以
> [英文 README](README.md) 为准。
