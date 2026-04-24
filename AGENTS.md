# AGENTS — go-onlyoffice

Canonical Go client for OnlyOffice Workspace (Projects + Calendar + CRM) and the `oo-cli` command.

## Topology

- **Library (root package `onlyoffice`)** — `onlyoffice.go`, `httpx.go`, `calendar.go`, `crm.go`, `tasks_extra.go`, `files.go`. Pure stdlib + `google/go-querystring`; no UI, no dotenv.
- **CLI (`cmd/oo-cli` + `internal/cli`)** — Cobra wrapper that loads `.env` via `godotenv` at startup. CLI-only deps (`spf13/cobra`, `joho/godotenv`) must stay out of the library surface.
- **Applications sync (`internal/applications`)** — README/CRM bridge, consumes the library.

## Rules

- Library must never call `godotenv.Load()` — the CLI does that.
- New endpoints go into the library first; CLI commands are thin wrappers.
- No secrets in the repo; use `.env` (gitignored). Commit `.env.example` only.
- Prefer httptest-based unit tests (see `httpx_test.go`); integration tests (`client_test.go`) auto-skip without `ONLYOFFICE_URL`.
- Follow SemVer on tags; this repo is tagged at GitHub under `git@github.com:eSlider/go-onlyoffice.git`.

## Related

- [`eSlider/inventar`](https://git.produktor.io/eSlider/inventar) — ASR/ADR (see ASR-0008 Go library module conventions).
- [`eSlider/inventar-sync`](https://git.produktor.io/eSlider/inventar-sync) — OnlyOffice → Gitea issue sync, consumes this library.
- [`produktor.io/vidarr`](https://git.produktor.io/produktor.io/vidarr) — legacy consumer being migrated from `pkg/onlyoffice` to this module.
