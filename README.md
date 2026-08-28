# envman

CLI environment variable sync checker — catch `.env` drift **before** deploy, not after the app crashes.

`envman` validates and compares environment variables across files, docker-compose, and remote VPS
over SSH. Targeted at self-hosters without an enterprise secret manager.

> **Status:** MVP (Phase 1). See [PRD.md](PRD.md) for the full product spec.

## Features (Phase 1)

| Feature | Command |
|---|---|
| Diff checker `.env.example` vs `.env` (missing / empty / extra) | `envman check` |
| Multi-environment compare as a table | `envman compare` |
| Remote sync check over SSH (read-only) | `envman check --remote user@host:/path/.env` |
| CI/CD exit code gate | `envman check --ci` |

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/novian/envman/main/install.sh | sh
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
    curl -fsSL https://raw.githubusercontent.com/novian/envman/main/install.sh | sh
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

## Roadmap

- **Phase 2:** type/format validation (`.envman.yaml`), secret leak scanner, required/optional flagging.
- **Phase 3:** docker-compose aware, `envman push`/`pull` sync (explicit confirm), Markdown/JSON reports + Telegram.
- **Out of scope:** full secret management (not a Vault/Doppler replacement).

## Development

```sh
go test ./...
go build ./...
```

## License

MIT — see [LICENSE](LICENSE).