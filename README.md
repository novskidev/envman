# envman

[![CI](https://github.com/novskidev/envman/actions/workflows/ci.yml/badge.svg)](https://github.com/novskidev/envman/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/novskidev/envman)](https://github.com/novskidev/envman/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/novskidev/envman)](https://go.dev/dl/)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

CLI environment variable sync checker — catch `.env` drift **before** deploy, not after the app crashes.

`envman` validates and compares environment variables across files, docker-compose, and remote VPS
over SSH. Targeted at self-hosters without an enterprise secret manager.

> **Status:** v0.2.3 — Phase 1 & 2 done. See [PRD.md](PRD.md) for the full product spec.

## Features

| Feature | Command |
|---|---|
| Diff checker `.env.example` vs `.env` (missing / empty / extra) | `envman check` |
| Multi-environment compare as a table | `envman compare` |
| Remote sync check over SSH (read-only) | `envman check --remote user@host:/path/.env` |
| CI/CD exit code gate | `envman check --ci` |
| Secret leak scanner (real secrets in example files) | `envman scan` |
| Type/format validation + `# required` flagging | `envman validate` |
| Inspect env vars declared in docker-compose.yml | `envman compose` |
| JSON / Markdown reports | `envman check --json / --markdown` |

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/novskidev/envman/main/install.sh | sh
```

or build from source (Go >= 1.21):

```sh
go build -o envman .
```

## Usage

```sh
$ envman check
ref:     .env.example (4 vars)
compare: .env (2 vars)

MISSING: JWT_SECRET
MISSING: LOG_LEVEL
EMPTY:   PORT
EXTRA:   EXTRA_LOCAL
OK: 1  MISSING: 2  EMPTY: 1  EXTRA: 1
```

```sh
$ envman check --remote deploy@vps.example.com:/srv/app/.env --ci
MISSING: MY_NEW_VAR
OK: 12  MISSING: 1  EMPTY: 0  EXTRA: 2
$ echo $?
1
```

```sh
$ envman compare .env.local .env.staging .env.production
Variable        .env.local  .env.staging  .env.production
DATABASE_URL    OK          OK            MISSING
JWT_SECRET      OK          EMPTY         OK
PORT            OK          OK            OK
```

### Secret leak scanner

```sh
$ envman scan --file .env.example
Found 2 possible real secret(s) in .env.example (should be placeholders):
  VITE_SUPABASE_ANON_KEY = eyJhb…Uts   [JWT (HS256)]
  DB_PASSWORD = superse…t0r            [key-name heuristic]
```

Exit code 1 if anything is flagged. Catches secrets that slipped into example files.

### Validation rules (`.envman.yaml`)

```yaml
# .envman.yaml
rules:
  DATABASE_URL:
    required: true
    type: url
  PORT:
    required: true
    type: port
  DEBUG:
    type: boolean
  APP_ID:
    pattern: "^app_[a-z0-9]{16}$"
```

```sh
$ envman validate
REQUIRED: DATABASE_URL: missing required variable
TYPE:     PORT: expected port 1-65535, got "abc"
```

Variables marked with `# required` in `.env.example` are enforced too:

```sh
# .env.example
# required
DATABASE_URL=postgres://localhost/db
API_KEY=your-key      # required
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | all OK |
| 1 | problems found (MISSING / EMPTY / EXTRA, unless `--allow-extra`) |
| 2 | usage or local file error |
| 3 | remote (SSH) error |

Use `--allow-extra` when your local `.env` legitimately has dev-only variables.

## CI/CD gate

Drop this before your deploy step — the deploy only runs if env files are in sync:

```yaml
- name: Env sync check (local vs server)
  run: |
    curl -fsSL https://raw.githubusercontent.com/novskidev/envman/main/install.sh | sh
    envman check --remote ${{ secrets.SERVER }}:/srv/app/.env --ci --allow-extra
  env:
    SERVER: deploy@vps.example.com
```

The runner needs an SSH deploy key (via ssh-agent) with read access to the server.

## SSH security notes

- **Read-only by design** — `envman` only runs `cat` on the remote host. No push/sync in this phase.
- **Host key verification on by default** — first connection to an unknown host fails with
  a hint: `ssh-keyscan -H <host> >> ~/.ssh/known_hosts`. Use `--insecure-ssh` explicitly to opt out.
- Auth uses keys in `~/.ssh/id_ed25519`, `id_rsa`, `id_ecdsa`, `id_dsa`, then ssh-agent.

## Config reference (`.envman.yaml`)

All rule types: `required`, `type`, `pattern`.

| `type` | accepts |
|---|---|
| `url` | `http(s)://…` |
| `port` | 1–65535 |
| `boolean` | `true`/`false` |
| `integer` | whole number |
| `float` | decimal number |
| `string` | anything (default) |

```yaml
rules:
  DATABASE_URL:
    required: true
    type: url
  PORT:
    required: true
    type: port
  DEBUG:
    type: boolean
  APP_ID:
    pattern: "^app_[a-z0-9]{16}$"
```

## Roadmap

- **Phase 3 (partial):** docker-compose inspect + JSON/Markdown reports done. `envman push`/`pull` sync (explicit confirm) + Telegram notifier pending.
- **Out of scope:** full secret management (not a Vault/Doppler replacement).

## Development

```sh
go test ./...
go build ./...
```

## Contributing

1. Fork + branch.
2. Keep it stdlib-first and boring — no new deps unless they earn their place.
3. Every behavior change ships with a test.
4. PR against `main`. CI (build + test + vet) must pass.

## License

MIT — see [LICENSE](LICENSE).