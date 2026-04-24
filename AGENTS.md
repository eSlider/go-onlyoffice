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
- Follow SemVer on tags; this repo is tagged at GitHub under `git@github.com:eSlider/go-onlyoffice.git`.

### Testing policy (2026-04-24)

**No synthetic OnlyOffice mockups.** Protocol-level behaviour must be
verified against a real OnlyOffice instance. `httptest.NewServer` is only
acceptable for testing the *caller's* logic that the library can't reach
(for example, the user's own HTTP handler). Anywhere we would otherwise
write `mux.HandleFunc("/api/2.0/...")` to emulate OnlyOffice, we write an
**integration test** instead.

- Unit tests (`*_test.go`, no build tag) — pure Go: parsers, encoders,
  struct conversions. No network. No fake servers that emulate the vendor.
- Integration tests (`//go:build integration` tag in `*_integration_test.go`)
  — hit a live OnlyOffice instance. Credentials come from `ONLYOFFICE_URL`,
  `ONLYOFFICE_USER`, `ONLYOFFICE_PASS` (aliases `_HOST`/`_NAME`/`_PASSWORD`
  also accepted). Tests **skip** cleanly when credentials are missing so
  `go test ./...` remains green in CI.
- Run integration with: `go test -tags=integration ./...`.
- New endpoints **must** ship with an integration test before merge.

## Related

- [`eSlider/inventar`](https://git.produktor.io/eSlider/inventar) — ASR/ADR (see ASR-0008 Go library module conventions).
- [`eSlider/inventar-sync`](https://git.produktor.io/eSlider/inventar-sync) — OnlyOffice → Gitea issue sync, consumes this library.
- [`produktor.io/vidarr`](https://git.produktor.io/produktor.io/vidarr) — legacy consumer being migrated from `pkg/onlyoffice` to this module.
