# scripts/

Utility scripts for local development and demos.

## `seed_demo.sql`

Loads realistic **demo data** so the admin panel and dashboards aren't empty:
operators (mixed plans / providers / statuses), subscriptions, WhatsApp
devices, chat sessions + messages, a product catalog, leads, customer profiles,
tags, and a knowledge base.

Intended for a **fresh database**. It is not idempotent — re-running on an
already-seeded DB fails on duplicate emails. To re-seed, reset the database
first (`docker compose down -v`), start the stack once so migrations run, then
apply the script.

### Apply it (Docker)

```bash
docker cp scripts/seed_demo.sql aichathubs-db:/tmp/seed_demo.sql
docker exec aichathubs-db \
  psql -U postgres -d smart_live_chats -v ON_ERROR_STOP=1 -f /tmp/seed_demo.sql
```

> If you changed `DB_PORT` or the Postgres user/db name, adjust the container
> name (`aichathubs-db`), `-U`, and `-d` accordingly.

### Demo logins created

| Where | Credentials |
|---|---|
| **User dashboard** (`/login`) | `demo@aichathubs.local` / `demo123456` |
| **Admin panel** (`/admin/login`) | your `ADMIN_USERNAME` / `ADMIN_PASSWORD` from `backend/.env` (default `admin` / `admin123456`) |

The product catalog, leads, customer profiles, tags, and knowledge base are
attached to the demo operator account; chat sessions are shared globally, so the
demo user sees all of them.

> ⚠️ This is **sample data with a known password** — only load it into
> development/demo environments, never a real deployment.
