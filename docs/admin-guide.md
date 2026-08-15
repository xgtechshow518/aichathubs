# AIChatsHub — Admin Guide

This guide covers the **admin panel**: the separate control surface for whoever
operates an AIChatsHub deployment. It's for platform operators, not end users —
if you're looking for how to set up and configure the server, see the
[README](../README.md); if you want the day-to-day agent/dashboard workflow,
that's the regular app (dashboard), not the admin panel.

> Screenshots are not included yet. If you'd like to contribute them, drop
> images in `docs/images/` and reference them here — see
> [CONTRIBUTING](../CONTRIBUTING.md).

---

## Contents

- [Signing in](#signing-in)
- [The admin panel at a glance](#the-admin-panel-at-a-glance)
- [Dashboard](#dashboard)
- [Users](#users)
- [WhatsApp Devices](#whatsapp-devices)
- [Payments (subscriptions)](#payments-subscriptions)
- [Chat Analytics](#chat-analytics)
- [AI Bot Prompt](#ai-bot-prompt)
- [AI Reply Delay](#ai-reply-delay)
- [System](#system)
- [Security notes](#security-notes)
- [Troubleshooting](#troubleshooting)

---

## Signing in

The admin panel lives at a separate URL and uses its own credentials — it is
**not** the same as a normal user account.

1. Open **`/admin/login`** on your frontend (e.g. `http://localhost:5173/admin/login`).
2. Enter the admin **username** and **password**.

Credentials come from `backend/.env`:

| Variable | Default | Notes |
|---|---|---|
| `ADMIN_USERNAME` | `admin` | Change it |
| `ADMIN_PASSWORD` | `admin123456` | **Change it before exposing the server** |

The backend prints a startup **WARNING** while `ADMIN_PASSWORD` is still the
built-in default. After editing `backend/.env`, recreate the backend so it picks
up the new values:

```bash
docker compose up -d --force-recreate backend
```

> There is only **one** admin login (a single shared username/password), not a
> list of admin accounts. Anyone with these credentials has full admin access.

---

## The admin panel at a glance

After login you land in the admin layout with a left sidebar. The sections are:

| Menu item | What it's for |
|---|---|
| **Dashboard** | Platform-wide stats and trends |
| **Users** | Manage every registered account |
| **WhatsApp Devices** | See/inspect all connected numbers |
| **Payments** | Browse Stripe subscriptions |
| **Chat Analytics** | Message/session volume and breakdowns |
| **AI Bot Prompt** | Set the global AI system prompt |
| **AI Reply Delay** | Tune the bot's "typing" delay |
| **System** | Health, uptime, and row counts |

Use **Logout** (top-right) when you're done — admin sessions are token-based.

---

## Dashboard

A read-only overview of the whole deployment:

- **Users** — total, verified, and new users today / this week.
- **Subscriptions** — active and trialing counts.
- **Devices** — total vs. currently connected WhatsApp devices.
- **Chats** — total and active chat sessions, total messages, messages today.
- **User registration trend** — sign-ups per day over the last 30 days.
- **Plan distribution** — how users split across subscription plans.
- **Provider distribution** — how accounts were created (email vs. Google /
  Facebook social login).

Use this as your daily health check. No actions here — it's informational.

---

## Users

The most-used section. Lists every registered account with search, filters, and
per-user actions.

### Finding users

- **Search** by email or name.
- **Filter** by subscription plan, subscription status, or auth provider.
- Results are paginated (20 per page by default) and show each user's **device
  count**.

### Viewing a user

Open a user to see their devices, subscription history, chat/message counts,
and whether they've configured a knowledge base (plus Q&A item count).

### Editing a user

You can override fields that would normally require a Stripe round-trip:

| Field | Effect |
|---|---|
| **Name** | Display name |
| **Subscription plan** | Move the user to a different plan |
| **Subscription status** | e.g. `active`, `trialing`, `canceled` |
| **Max devices** | How many WhatsApp numbers they may connect (0–1000) |
| **Trial ends at** | Set/clear a trial end date |
| **Email verified** | Manually mark verified/unverified |

> Email, provider, password, and IDs are **not** editable here by design — use
> the dedicated password-reset action for passwords.

### Per-user actions

- **Reset password** — either type a new password (min 8 chars) or leave it
  blank to have the server **generate a random 12-char password**, which is
  returned once so you can share it with the user out-of-band. (If the account
  was OAuth-only, this also enables email/password login for it.)
- **Suspend** — blocks the user from logging in and invalidates their existing
  sessions immediately. You can record a reason.
- **Unsuspend** — restores access.
- **Edit knowledge base** — toggle the user's AI **auto-reply** on/off and edit
  their **per-user system prompt** on their behalf. Changes take effect on the
  next message (the AI context cache is refreshed automatically).
- **Delete** — permanently removes the user. **This cannot be undone** — prefer
  *Suspend* unless you're sure.

---

## WhatsApp Devices

A global view of every WhatsApp number connected across all users.

- **Search** by phone number, JID, or the owner's email.
- **Filter** by status (e.g. `connected`).
- Each row shows the owning user (email/name).
- **Drill down**: from a device you can list its **chat sessions**, and from a
  session view its **messages** (read-only) — useful for support and abuse
  investigation.

> This view is for oversight. Connecting/disconnecting a number is done by the
> user in their own dashboard, not here.

---

## Payments (subscriptions)

Lists Stripe subscription records across all users.

- **Filter** by status and plan.
- Each row shows the associated user (email/name).

This section is only meaningful when **billing is enabled**
(`BILLING_ENABLED=true` + Stripe keys). In free mode it will be empty — see
[Billing modes](../README.md#billing-modes).

---

## Chat Analytics

Aggregate conversation metrics:

- **Messages per day** and **new sessions per day** (last 30 days).
- **Platform breakdown** (e.g. WhatsApp).
- **Status breakdown** of sessions (active, etc.).
- **Message-type breakdown** by sender (customer / agent / bot).
- Total sessions and total messages.

Good for spotting volume trends and how much of the load the AI is handling.

---

## AI Bot Prompt

Sets the **global system prompt** that steers the AI assistant's behavior and
tone for every reply, platform-wide.

- The panel shows the **effective** prompt, whether it's **custom** (set here) or
  the **default** (from `GEMINI_SYSTEM_PROMPT` in `backend/.env`), plus who last
  changed it and when.
- **Save** a custom prompt to override the default for all users.
- **Clear** the field (save empty) to fall back to the env default again.

Changes apply on the **next message** — saving automatically refreshes the AI's
cached context so the new prompt takes effect without a restart.

> Individual users can also set their own per-user prompt (see
> [Users → Edit knowledge base](#users)). The relationship between the global
> prompt and per-user prompts depends on your setup; the global prompt is the
> platform-wide baseline.

---

## AI Reply Delay

Controls how long the bot "waits" before sending an auto-reply, to make replies
feel more human instead of instant.

- Set a **minimum** and **maximum** delay in **seconds**; the bot picks a random
  value in that range per reply.
- Rules: both values must be **non-negative**, **min ≤ max**, and **max ≤ 600**
  (10 minutes) — the upper bound guards against a typo that would hang replies.
- **Reset to defaults** restores the built-in range with one click.

Changes apply immediately (the delay cache is refreshed on save/reset).

---

## System

A health and diagnostics view:

- **Database status** — `connected` / `disconnected` (live ping).
- **Uptime** — how long the backend process has been running.
- **Row counts** for the main tables: users, subscriptions, chat sessions,
  chat messages, WhatsApp devices, knowledge bases, Q&A items, and tags.

Handy for a quick "is everything alive and roughly the right size?" check.

---

## Security notes

- **Change the default admin credentials** before exposing the server. The
  backend warns you at startup while they're default.
- The admin login is a **single shared credential** — treat it like a root
  password. Anyone who has it can read all conversations, reset passwords, and
  delete users.
- Admin actions are powerful and some are **irreversible** (delete user). Prefer
  **Suspend** over **Delete** when in doubt.
- Generated passwords from **Reset password** are shown **once** — copy them
  immediately and share securely.
- Serve the panel over **HTTPS** in production so the admin token and
  credentials aren't sent in the clear.

---

## Troubleshooting

| Symptom | Likely cause / fix |
|---|---|
| Can't log in with `admin` / `admin123456` | You changed `ADMIN_USERNAME` / `ADMIN_PASSWORD` in `backend/.env` — use those. If you just edited `.env`, recreate the backend: `docker compose up -d --force-recreate backend`. |
| Payments page is empty | Billing is off. Set `BILLING_ENABLED=true` and add Stripe keys — see the README. |
| Bot prompt change didn't take effect | It applies on the **next** message. If it still doesn't, confirm Gemini is configured (`GEMINI_API_KEY` set) — the AI is disabled without a key. |
| Devices show but none are "connected" | Users link/connect numbers from their own dashboard via QR code; a device can exist but be disconnected. |
| Startup log warns about `ADMIN_PASSWORD` / `JWT_SECRET` | You're still on the built-in defaults — change them before going live. |

---

Questions or gaps in this guide? Open an issue, or email
[hello@awkiss.com](mailto:hello@awkiss.com).
