# AIChatsHub

Self-hostable AI customer-chat platform. Connect a WhatsApp number, upload a
knowledge base, and let a Gemini-powered assistant auto-reply to your
customers — with a live agent dashboard, product catalog, and lead capture.

Built with **Go (Echo)** + **React (Vite)** + **PostgreSQL**. Bring your own
credentials: nothing is hardcoded, and the app runs happily with only a
database configured — every integration (AI, email, social login, billing)
turns on the moment you add its key.

> **Heads up:** this repo ships **no secrets**. Every deployment supplies its
> own keys via `.env` files (git-ignored). See [Configuration](#configuration).

> 💼 **Don't want to self-host?** The software is free forever — but if you'd
> rather not manage servers, I offer paid **setup, managed hosting, and
> maintenance**. See [Services & Support](#services--support) or email
> [hello@awkiss.com](mailto:hello@awkiss.com).

---

## Features

- **AI auto-reply** — Gemini answers customer messages from your uploaded Q&A
  knowledge base and product catalog.
- **WhatsApp integration** — link one or more numbers via QR code. Powered by
  the third-party open-source library
  [whatsmeow](https://github.com/tulir/whatsmeow) (a Go WhatsApp Web multidevice
  client) — no official WhatsApp Business API or paid gateway required.
- **Live agent dashboard** — real-time chat via WebSocket, take over from the
  bot, tag conversations, blacklist contacts.
- **Product catalog & leads** — import products, the bot recommends them and
  captures leads.
- **Auth** — email/password (with optional email verification) plus optional
  Google / Facebook social login.
- **Optional billing** — Stripe per-connected-device subscriptions, off by
  default (fully free when self-hosted).
- **Admin panel** — manage users, devices, subscriptions, and the global bot
  prompt.

## Tech stack

| | |
|---|---|
| **Frontend** | React 19 · TypeScript · Vite 7 · Ant Design · Zustand · Axios |
| **Backend** | Go 1.26 · Echo v4 · GORM · JWT · WebSocket |
| **Database** | PostgreSQL 16 (`pg_trgm` extension) |
| **AI** | Google Gemini (`gemini-3.7-flash` by default) |
| **WhatsApp** | whatsmeow (session store in SQLite) |
| **Billing** | Stripe (optional) |

---

## Quick start (Docker)

The fastest way to run the whole stack — Postgres, backend, and frontend — is
Docker Compose. You need [Docker Desktop](https://www.docker.com/products/docker-desktop/)
(or Docker Engine + Compose v2).

```bash
git clone https://github.com/xgtechshow518/aichathubs.git
cd aichathubs

# 1. Create your backend config from the template
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env

# 2. (Optional) edit backend/.env to add your Gemini/SMTP/OAuth/Stripe keys.
#    It runs without them — see "Graceful degradation" below.

# 3. Build and start everything
docker compose up --build
```

Then open:

| Service | URL |
|---|---|
| Frontend | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| Postgres | localhost:5432 (user/pass `postgres`) |

Docker Desktop shows a single **`aichathubs`** group with three child
containers: `aichathubs-db`, `aichathubs-backend`, `aichathubs-frontend`.

Stop with `Ctrl+C`, or `docker compose down` (add `-v` to also wipe the
database and WhatsApp session volumes).

### Ports already in use?

If `5432`, `8080`, or `5173` are taken on your machine, copy the root port
template and change them — Compose picks it up automatically:

```bash
cp .env.example .env   # edit DB_PORT / BACKEND_PORT / FRONTEND_PORT
```

---

## Configuration

All configuration is via environment variables. Two `.env` files, each with a
committed `.env.example` template:

- **`backend/.env`** — the important one: database, secrets, and all integration
  keys. See [`backend/.env.example`](backend/.env.example) for the full,
  documented list.
- **`frontend/.env`** — public browser config (API URL, Google client ID). See
  [`frontend/.env.example`](frontend/.env.example).
- **`.env`** (root, optional) — only Docker Compose published ports.

### Required vs optional

| Variable(s) | Required? | Effect if unset |
|---|---|---|
| `DATABASE_URL` | ✅ Required | App won't start |
| `JWT_SECRET` | ✅ Set a strong value | Insecure default (warned at startup) |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | ✅ Change them | Insecure default (warned at startup) |
| `GEMINI_API_KEY` | Optional | AI auto-reply disabled |
| `SMTP_*` | Optional | New users auto-verified (no email sent) |
| `GOOGLE_CLIENT_ID` / `_SECRET` | Optional | "Sign in with Google" hidden |
| `FACEBOOK_CLIENT_ID` / `_SECRET` | Optional | "Sign in with Facebook" hidden |
| `BILLING_ENABLED` + `STRIPE_*` | Optional | Free mode: unlimited devices, no paywall |

### Graceful degradation

The app is designed to run with **minimal** config and light up features as you
add keys. On startup the backend logs exactly what's on:

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

The frontend queries `GET /api/config` (booleans only, no secrets) to know
which login buttons to show.

### Where to get each credential

| Key | Where |
|---|---|
| **Gemini API key** | https://aistudio.google.com/apikey |
| **Google OAuth** | https://console.cloud.google.com/apis/credentials — add redirect URIs `http://localhost:5173/login` and `http://localhost:8080/api/auth/google/callback` |
| **SMTP (Gmail)** | App Password at https://myaccount.google.com/apppasswords |
| **Stripe** | https://dashboard.stripe.com/apikeys (secret key + a subscription Price ID + webhook secret) |

---

## Billing modes

- **Free (default)** — `BILLING_ENABLED=false`. No Stripe needed. Users get
  unlimited WhatsApp devices and full access; the subscription gate is off.
- **Paid** — `BILLING_ENABLED=true` + Stripe keys + a `STRIPE_PRICE_STARTER`
  price ID. Users are charged per connected WhatsApp device (Stripe
  subscription quantity), with a device limit enforced at connect time.

---

## Manual setup (without Docker)

<details>
<summary>Prerequisites: Go 1.26+, Node.js 20+, PostgreSQL 16+</summary>

**Database**

```sql
CREATE DATABASE smart_live_chats;
```

**Backend**

```bash
cd backend
cp .env.example .env      # set DATABASE_URL to your local Postgres
go mod download
go run ./cmd/server       # serves on http://localhost:8080
```

**Frontend**

```bash
cd frontend
cp .env.example .env      # set VITE_API_URL=http://localhost:8080
npm install
npm run dev               # serves on http://localhost:5173
```

</details>

---

## Admin panel

A separate admin login lives at `/admin/login` (frontend) backed by
`ADMIN_USERNAME` / `ADMIN_PASSWORD`. From there you can manage users, devices,
subscriptions, the global bot system-prompt, and AI reply-delay settings.

---

## Project structure

```
aichathubs/
├── backend/                  # Go (Echo) API
│   ├── cmd/server/           # entry point
│   ├── internal/
│   │   ├── handlers/         # HTTP + WebSocket handlers
│   │   ├── models/           # GORM models
│   │   ├── gemini/           # AI service
│   │   ├── whatsapp/         # whatsmeow manager
│   │   ├── middleware/       # JWT, admin, subscription
│   │   ├── database/         # DB connection + migrations
│   │   ├── email/            # SMTP verification codes
│   │   └── config/           # env config + startup summary
│   ├── Dockerfile
│   └── .env.example
├── frontend/                 # React (Vite) app
│   ├── src/{pages,components,services,store,types}
│   ├── Dockerfile.dev
│   └── .env.example
├── docker-compose.yml
├── .env.example              # Docker port overrides
└── LICENSE
```

---

## Security

- **Never commit `.env` files.** They are git-ignored; only `.env.example`
  templates are tracked.
- Change `JWT_SECRET` and `ADMIN_PASSWORD` from their defaults before exposing
  the server (the backend warns you at startup while they're default).
- Treat any key that has ever been committed as compromised — rotate it.

## Services & Support

**AIChatsHub is free and open source — and always will be.** You can self-host
it yourself using the guide above at no cost.

That said, deploying and running a production chat platform (server, domain,
SSL, WhatsApp linking, API keys, updates, backups) takes time. If you'd rather
skip the setup or need it maintained, I offer paid services:

| Service | What it covers |
|---|---|
| **Setup & installation** | Deploy on your server or VPS — domain, SSL, WhatsApp linking, Gemini/SMTP/OAuth keys wired up and tested |
| **Managed hosting** | I host and run it for you — updates, backups, monitoring, uptime |
| **Maintenance & updates** | Keep it patched and current: dependency upgrades, security fixes, breakage repair |
| **Customization** | New features, custom branding, and integrations tailored to your business |
| **Priority support** | Direct assistance with faster response times |

Pricing is quoted per request based on your needs — get in touch:

📧 **[hello@awkiss.com](mailto:hello@awkiss.com)**

> Self-hosting stays 100% free. Paid services are an optional convenience for
> teams who'd rather not manage servers themselves.

## Sponsor

If this project saves you time and you'd like to support its continued
development, sponsorships are appreciated (but never required):

- ❤️ [GitHub Sponsors](https://github.com/sponsors/xgtechshow518)
- ☕ [Buy Me a Coffee](https://buymeacoffee.com/xgtechshowl)

---

## Acknowledgments & third-party notices

WhatsApp connectivity is provided entirely by the third-party open-source
library **[whatsmeow](https://github.com/tulir/whatsmeow)** (MPL-2.0), an
unofficial Go client for WhatsApp Web multidevice. This project is **not
affiliated with, endorsed by, or sponsored by WhatsApp or Meta**. "WhatsApp"
is a trademark of Meta Platforms, Inc.

Using an unofficial client may violate WhatsApp's Terms of Service and can lead
to your number being rate-limited or banned. You are responsible for how you use
it — prefer a dedicated/test number and comply with all applicable laws and
WhatsApp's terms.

Other notable open-source dependencies: [Echo](https://echo.labstack.com/),
[GORM](https://gorm.io/), [React](https://react.dev/),
[Ant Design](https://ant.design/), and the
[Google Gemini API](https://ai.google.dev/).

## License

[MIT](LICENSE) © 2026 AIChatsHub
