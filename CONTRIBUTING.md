# Contributing to AIChatsHub

First off — thanks for taking the time to contribute! 🎉 AIChatsHub is an
open-source project and community contributions are welcome.

This document explains how to propose changes and what to expect. Please read
it before opening an issue or pull request.

---

## Ground rules

- **The maintainer has the final say.** Contributions are *proposals*. Whether
  a change fits the project's direction, quality bar, and long-term maintenance
  plan is decided by the maintainer. A declined PR isn't a judgment on you —
  it just may not fit the roadmap.
- **Be respectful.** All interactions are governed by our
  [Code of Conduct](CODE_OF_CONDUCT.md).
- **Open an issue before large changes.** For anything beyond a small fix,
  please open an issue first to discuss it. This avoids wasted effort on a PR
  that might not be merged.

---

## How to report a bug

1. Search [existing issues](https://github.com/xgtechshow518/aichathubs/issues)
   to avoid duplicates.
2. Open a new issue and include:
   - What you expected to happen vs. what actually happened
   - Steps to reproduce
   - Your environment (OS, Docker version, browser)
   - Relevant logs (`docker compose logs backend` / `frontend`) — **redact any
     secrets or API keys first**

## How to suggest a feature

Open an issue describing the feature, the problem it solves, and who benefits.
Keep in mind the project aims to stay lean and self-hostable — not every idea
will fit, and that's okay.

---

## Development setup

See the [README](README.md) for full instructions. In short:

```bash
git clone https://github.com/xgtechshow518/aichathubs.git
cd aichathubs
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env
docker compose up --build
```

- **Backend**: Go 1.26 · Echo · GORM · PostgreSQL
- **Frontend**: React 19 · TypeScript · Vite

**Never commit `.env` files or real secrets.** Only `.env.example` templates
are tracked. If you accidentally commit a secret, treat it as compromised and
rotate it immediately.

---

## Submitting a pull request

1. **Fork** the repo and create a branch from `main`
   (e.g. `fix/whatsapp-reconnect` or `feat/export-leads`).
2. Make your change. Keep PRs **focused** — one logical change per PR is much
   easier to review than a large mixed one.
3. **Match the surrounding code style.** For the frontend, run the linter
   (`npm run lint` in `frontend/`). For the backend, run `go build ./...` and
   `go vet ./...` and make sure it compiles cleanly.
4. Update documentation (README, `.env.example`, comments) if your change
   affects configuration or behavior. If you edit `README.md`, please also
   update the Simplified Chinese translation `README.zh-CN.md` (or note in your
   PR that it needs updating). The **English `README.md` is the source of
   truth** — translations may lag.
5. **Sign off your commits** (see below).
6. Open the PR against `main` with a clear description of *what* changed and
   *why*.

The maintainer will review it. You may be asked to make changes — that's a
normal part of the process.

---

## Sign your work — Developer Certificate of Origin (DCO)

To keep the project's licensing clean, every commit must be **signed off**.
This is a simple, lightweight statement that you wrote the code (or otherwise
have the right to submit it) and agree to license it under the project's
[MIT License](LICENSE). It is **not** a copyright assignment — you keep
ownership of your contribution.

Add a `Signed-off-by` line to each commit by using the `-s` flag:

```bash
git commit -s -m "fix: reconnect WhatsApp session after network drop"
```

This appends a line like:

```
Signed-off-by: Your Name <your.email@example.com>
```

By signing off, you certify the [Developer Certificate of Origin](https://developercertificate.org/)
(reproduced below). Use your real name and a valid email.

<details>
<summary>Developer Certificate of Origin v1.1 (full text)</summary>

```
By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I have the right
    to submit it under the open source license indicated in the file; or

(b) The contribution is based upon previous work that, to the best of my
    knowledge, is covered under an appropriate open source license and I have
    the right under that license to submit that work with modifications,
    whether created in whole or in part by me, under the same open source
    license (unless I am permitted to submit under a different license), as
    indicated in the file; or

(c) The contribution was provided directly to me by some other person who
    certified (a), (b) or (c) and I have not modified it.

(d) I understand and agree that this project and the contribution are public and
    that a record of the contribution (including all personal information I
    submit with it, including my sign-off) is maintained indefinitely and may be
    redistributed consistent with this project or the open source license(s)
    involved.
```

</details>

---

## License

By contributing, you agree that your contributions will be licensed under the
project's [MIT License](LICENSE).

## Questions?

Open an issue, or email [hello@awkiss.com](mailto:hello@awkiss.com).
