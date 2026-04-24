# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-04-24

### Changed — project structure

- **Library files reorganised by domain** (mechanical split; zero API surface
  change). The former monolithic `onlyoffice.go` (687 LOC) is now split into:
  - `client.go` — `Client`, `Credentials`, `Defaults`, env helpers, `NewClient`.
  - `request.go` — `Request`, `Query`, `Time`, `Token`, `MetaResponse`,
    `Permissions`, `requestBodyReader`.
  - `auth.go` — `Authenticate`, `AuthenticateContext`, `InvalidateToken`,
    `Auth`, `ensureToken`, `authHeader`, `tokenValid`.
  - `http.go` — transport helpers and DRY response decoders
    (`ResponseArray`, `ResponseObject`, `postFormObject`, `putFormObject`,
    `deleteObject`, `unmarshalResponseObject`).
  - `projects.go` — `Project`, `Projects`, `Milestone`, `ProjectOwner` +
    `GetProjects` / `CreateProject` / `UpdateProject` / `DeleteProject` /
    `GetProjectByID` / `GetProjectMilestones`.
  - `tasks.go` — `Task`, `ProjectTaskStatus`, `TaskPriority`,
    `ProjectGetTasksRequest` et al. **plus** the form-encoded helpers
    formerly in `tasks_extra.go` (`ListTasks`, `AddTask`, `AddSubtask`,
    `UpdateTaskStatus`, `DeleteTask`, `GetTaskByID`).
  - `users.go` — `User`, `Contact`, `Group`, `GetUsers`, `SelfUserID`.
  - `calendar.go`, `crm.go`, `files.go` — unchanged in scope, refactored
    through the new DRY helpers.
  - `httpx.go` → **renamed** `http.go`.
  - `onlyoffice.go` and `tasks_extra.go` — **deleted** (content redistributed).

### Changed — CLI **BREAKING**

- **Binary renamed `oo-cli` → `oo`.** Install with
  `go install github.com/eslider/go-onlyoffice/cmd/oo@latest`.
- **Package path `cmd/oo-cli` → `cmd/oo`.** The old path is removed.
- **`internal/cli` is gone.** Cobra commands now live directly under
  `cmd/oo/` as `package main`, split by domain: `calendar.go`, `crm.go`,
  `tasks.go`, `apps.go`, `common.go`. Rationale: cobra wiring is a CLI-only
  concern and does not belong inside a `pkg-level internal/`.
- **`internal/applications` → `cmd/oo/applications/`.** This is a
  CV-specific CRM workflow — not a general OnlyOffice feature — and is only
  consumed by the `oo` CLI. Keeping it under `cmd/oo/` prevents accidental
  external adoption and makes the coupling explicit.
- **`examples/applications/` removed.** It imported an internal package,
  which was a policy smell. Remaining examples (`basic`, `calendar`, `crm`,
  `subtasks`) use only the exported library surface.

### Added

- `(*Client).ResponseObject` — GET-and-decode-object counterpart to the
  existing `ResponseArray`.
- `(*Client).postFormObject` / `putFormObject` / `deleteObject` — eliminate
  the ~15 identical "form request → `responseField` → `json.Unmarshal`"
  blocks previously duplicated across `crm.go` / `tasks_extra.go` /
  `calendar.go` / `files.go`.

### Migration

External library consumers: **no changes required**. The module path
(`github.com/eslider/go-onlyoffice`), the `onlyoffice` package name, and
every exported symbol are unchanged.

CLI users: replace `oo-cli` with `oo` in scripts and CI. The command set
and flags are identical.

## [0.3.2] - 2026-04-24

### Fixed

- `Project.String()` no longer interprets the title as a format string
  (`fmt.Sprintf(*p.Title)`) and is now nil-safe on a zero-value `Project`.
- `internal/applications.buildSummary` no longer panics at regex compile time
  on Go 1.23+ — the previous `(?= ...)` lookahead is replaced with an RE2-safe
  non-capturing trailing delimiter.

### Changed

- `Client.Query` now routes token acquisition through the shared
  `ensureToken` path instead of duplicating the auth-expiry check inline.
- Request body marshalling is consolidated into an unexported
  `requestBodyReader` helper (DRY; no change to the public surface).
- The `Request.Debug` field is preserved for backwards compatibility but no
  longer changes behaviour — both branches used to unmarshal into the same
  target value. We'll remove the field in a future major release.

### Tests

- Deleted `httptest.NewServer` fixtures that emulated OnlyOffice protocol
  endpoints. Replaced them with:
  - pure-Go unit tests in `unit_test.go` (no network);
  - real integration tests in `client_test.go` guarded by
    `//go:build integration`. Run with
    `go test -tags=integration ./...`. Tests skip cleanly when
    `ONLYOFFICE_URL/USER/PASS` (or aliases) are absent.
- New policy documented in `AGENTS.md` and
  `.cursor/rules/no-synthetic-mocks.mdc`.

## [0.3.1] - 2026-04-24

### Added

- `AuthenticateContext(ctx)` — context-aware auth that honours cancellation and
  deadlines. Preferred entry point for long-running syncs (cron, watchers).
- `InvalidateToken()` — clears the cached token to force re-auth on the next
  request. Use this to recover from a mid-sync 401 when the server has revoked
  the session while the local `Expires` timestamp still looks fresh.

### Notes

- Plain `Authenticate()` is unchanged and remains a convenience wrapper around
  `AuthenticateContext(context.Background())`.
- No breaking changes; a patch release.

## [0.3.0] - 2026-04-24

### Added

- Calendar helpers: `ListCalendars`, `ListEvents`, `AddEvent`, `DeleteEvent`.
- CRM helpers: contacts (`ListContacts`, `GetContact`, `FindCompany`, `FindPerson`,
  `CreateCompany`, `CreatePerson`, `AddContactInfo`, `DeleteContact`), deals
  (`ListOpportunities`, `GetOpportunity`, `CreateOpportunity`,
  `AddOpportunityMember`, `ListDealStages`, `DeleteOpportunity`), cases
  (`ListCases`, `CreateCase`, `AddCaseMember`, `DeleteCase`), CRM tasks
  (`ListCRMTasks`, `CreateCRMTask`, `DeleteCRMTask`, `ListTaskCategories`), and
  history notes (`AddHistoryNote`).
- Project task extras: `GetProjectByID`, `ListTasks`, `ListAllTasks`,
  `GetTaskByID`, `AddTask`, `AddSubtask`, `UpdateTaskStatus`, `DeleteTask`.
- File upload: `UploadOpportunityFile` (multipart).
- `SelfUserID` cached lookup of `people/@self`.
- `Defaults` struct + `SetDefaults` + `GetEnvironmentDefaults` for optional
  calendar/project fallbacks.
- Alias env vars accepted by `GetEnvironmentCredentials`:
  `ONLYOFFICE_HOST` / `ONLYOFFICE_NAME` / `ONLYOFFICE_PASSWORD`.
- Public `Authenticate()` that primes the token eagerly.
- Bundled CLI: `cmd/oo-cli` (Cobra) with commands `cal-list`, `cal-events`,
  `cal-add`, `cal-delete`, `task-list`, `task-add`, `subtask-add`,
  `task-update`, `crm-contacts`, `crm-add-contact`, `crm-deals`,
  `crm-add-deal`, `crm-cases`, `applications-sync`.
- httptest-based unit tests for form / multipart / CRM helpers.

### Changed

- The `onlyoffice.Client` struct gained unexported fields (`defaults`, `selfID`,
  `noteCatID`); the public API is unchanged and remains backwards compatible.

## [0.2.0] - earlier

- Badges, docs expansion (Gitea sync use case, Gantt/PM workflows).

## [0.1.0] - earlier

- Initial release: OnlyOffice Project Management API client.
