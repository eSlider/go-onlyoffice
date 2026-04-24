# AGENTS — go-onlyoffice

Canonical Go client for OnlyOffice Workspace (Projects + Calendar + CRM) and the `oo` command.

## Topology

- **Library — flat package `onlyoffice` at repo root.** Split by *domain file*, not by subpackage, so every call site reads `c.XxxYyy()` against a single `*Client`. Files:
  - `client.go` — `Client`, `Credentials`, `Defaults`, env helpers, `NewClient`.
  - `request.go` — `Request`, `Query`, `Time`, `Token`, `MetaResponse`, `Permissions`.
  - `auth.go` — `Authenticate`, `AuthenticateContext`, `InvalidateToken`, `Auth`, token lifecycle.
  - `http.go` — transport + DRY response decoders (`ResponseArray`/`ResponseObject`/`postFormObject`/`putFormObject`/`deleteObject`).
  - `projects.go`, `tasks.go`, `users.go`, `calendar.go`, `crm.go`, `files.go` — typed / untyped domain methods.
  - Pure stdlib + `google/go-querystring`; no UI, no dotenv.
- **CLI — `cmd/oo/` as `package main`.** Cobra wrapper that loads `.env` via `godotenv` at startup. **Subject-based command tree** mirroring [`tea`](https://gitea.com/gitea/tea):
  - `main.go` — entry point (docstring lists the command tree).
  - `common.go` — `rootCmd`, `newOO`, `printTable`/`printObject`, `--output table|json` flag.
  - `calendar.go`, `projects.go`, `tasks.go`, `users.go`, `contacts.go`, `opportunities.go`, `cases.go`, `crm_tasks.go`, `apps.go` — one file per subject, each registers its subject `cobra.Command` in `init()` and attaches verb subcommands (`list`/`get`/`create`/`update`/`delete`/…).
  - CLI-only deps (`spf13/cobra`, `joho/godotenv`) stay out of the library.
- **Applications sync — `cmd/oo/applications/`.** README→CRM bridge, CV-specific; kept under `cmd/oo/` so it's clear it's internal to the binary, not a library feature.

## Rules

- Library must never call `godotenv.Load()` — the CLI does that.
- New endpoints go into the library first; CLI commands are thin wrappers.
- Prefer `ResponseObject` / `postFormObject` / `putFormObject` / `deleteObject` over hand-rolled `json.Unmarshal(responseField(...))` blocks — they exist for DRY, use them.
- Domain split is by file, **not** by subpackage. Don't introduce `internal/` or `pkg/*` subpackages inside the library — it flattens the `*Client` call surface for a reason.
- CLI commands follow **subject → verb** structure (`oo <subject> <verb>`), never `oo <verb>-<subject>`. Add new commands to the existing subject file if one fits; create a new `cmd/oo/<subject>.go` for a genuinely new domain.
- Every table output goes through `printTable(headers, rows)`; every single-object through `printObject(v)`. Do not `fmt.Println` rows ad-hoc or the `--output json` flag breaks for that command.
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
