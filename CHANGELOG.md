# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.8.2](https://github.com/eSlider/go-onlyoffice/compare/v0.8.1...v0.8.2) (2026-07-23)


### Bug Fixes

* **ci:** retry tag fetch in Release workflow ([#11](https://github.com/eSlider/go-onlyoffice/issues/11)) ([31b4aa0](https://github.com/eSlider/go-onlyoffice/commit/31b4aa0d90393e6d93513284e453b7da7e92c62d))

## [0.8.1](https://github.com/eSlider/go-onlyoffice/compare/v0.8.0...v0.8.1) (2026-07-23)


### Bug Fixes

* **oo:** assign owner and deadline on task create ([#9](https://github.com/eSlider/go-onlyoffice/issues/9)) ([f608207](https://github.com/eSlider/go-onlyoffice/commit/f60820734924284d6c40b46c79380a24573432f3))

## [0.8.0](https://github.com/eSlider/go-onlyoffice/compare/v0.7.0...v0.8.0) (2026-07-23)


### Features

* **office:** users admin UI, table rendering, and user save fixes ([40b74eb](https://github.com/eSlider/go-onlyoffice/commit/40b74eb8abf4d63793a2cb9e59bd899734368b7e))
* **search:** add produktor SearXNG JSON client ([79b1a6b](https://github.com/eSlider/go-onlyoffice/commit/79b1a6b2c397c2a8f8aeef7e2d8e6d81e0a93db2))
* **search:** add produktor SearXNG JSON client ([bf144f2](https://github.com/eSlider/go-onlyoffice/commit/bf144f2bb37c44aa37ff33c6b03f59d7b5023dc7))


### Bug Fixes

* **oo:** skip junk dirs in applications Discover ([470ac93](https://github.com/eSlider/go-onlyoffice/commit/470ac938b1e388a17aabc5146cd62070dbb00587))
* **oo:** skip junk dirs in applications Discover ([830f0c7](https://github.com/eSlider/go-onlyoffice/commit/830f0c7208f94521a59dfe6f96d496509e6b29e9))


### Code Refactoring

* **office:** generalize DataTable layout and document TUI table skill ([38932b0](https://github.com/eSlider/go-onlyoffice/commit/38932b0e0eb9696c41a48ad74c943034ebed59e5))

## [0.7.0](https://github.com/eSlider/go-onlyoffice/compare/v0.6.0...v0.7.0) (2026-07-23)


### Features

* **office:** mail preview, infinite scroll, pane resize, and scrollbars ([72844df](https://github.com/eSlider/go-onlyoffice/commit/72844dfe243e04c61ea9bc4db66c55abc08d3ae8))
* **office:** users admin UI, table rendering, and user save fixes ([40b74eb](https://github.com/eSlider/go-onlyoffice/commit/40b74eb8abf4d63793a2cb9e59bd899734368b7e))


### Bug Fixes

* **oo:** skip junk dirs in applications Discover ([470ac93](https://github.com/eSlider/go-onlyoffice/commit/470ac938b1e388a17aabc5146cd62070dbb00587))
* **oo:** skip junk dirs in applications Discover ([830f0c7](https://github.com/eSlider/go-onlyoffice/commit/830f0c7208f94521a59dfe6f96d496509e6b29e9))


### Code Refactoring

* **office:** generalize DataTable layout and document TUI table skill ([38932b0](https://github.com/eSlider/go-onlyoffice/commit/38932b0e0eb9696c41a48ad74c943034ebed59e5))

## [0.6.0](https://github.com/eSlider/go-onlyoffice/compare/v0.5.0...v0.6.0) (2026-06-24)


### Features

* **office:** table detail panes and Alt+1/2/3 layout toggles ([934af21](https://github.com/eSlider/go-onlyoffice/commit/934af21bc93a54df9f40aacb4d89468b00d7562e))


### Documentation

* CHANGELOG for office v0.5.1 release ([2985070](https://github.com/eSlider/go-onlyoffice/commit/29850701f8c9ab0c6d6427f63eaa189a1c558140))

## [0.5.0](https://github.com/eSlider/go-onlyoffice/compare/v0.4.0...v0.5.0) (2026-06-24)


### Features

* **office:** scrollable panes, nav tree drill-down, and item actions ([fe2ee48](https://github.com/eSlider/go-onlyoffice/commit/fe2ee481b9c0f959b798fd40679084a8c4bc4ba2))

## [0.4.0](https://github.com/eSlider/go-onlyoffice/compare/v0.3.2...v0.4.0) (2026-06-24)


### ⚠ BREAKING CHANGES

* **cli:** subject-based command tree (tea-style) + global --output flag
* rename oo-cli → oo, split library by domain, relocate applications

### Features

* **cli:** subject-based command tree (tea-style) + global --output flag ([1e6d22f](https://github.com/eSlider/go-onlyoffice/commit/1e6d22f38dd7a70d4cf1b69ef0b8749b01fded1a))
* **crm:** dedupe duplicates, fix deal titles, and add cleanup CLI ([c77519f](https://github.com/eSlider/go-onlyoffice/commit/c77519fac5ae9dffaad3f1f0ae722d969e62dde9))
* **crm:** merge company slogan variants in dedupe grouping ([3eb649c](https://github.com/eSlider/go-onlyoffice/commit/3eb649c58bf489b221f90734e04f33063f736914))
* **files:** project/task Documents API + oo projects|tasks files ([e03fd62](https://github.com/eSlider/go-onlyoffice/commit/e03fd6220012c73b110f1e0b02a12971174eb290))
* **mails:** add Workspace mail CLI with pagination and parsed from fields ([1cb228e](https://github.com/eSlider/go-onlyoffice/commit/1cb228e5d83ab453660bbace5f936a77f9381fc2))
* **office:** add Workspace TUI with shared bootstrap and test suite ([7358ff5](https://github.com/eSlider/go-onlyoffice/commit/7358ff5c326cdc873cb68b3f96d31155ecba1ee3))


### Code Refactoring

* rename oo-cli → oo, split library by domain, relocate applications ([cff8145](https://github.com/eSlider/go-onlyoffice/commit/cff8145f8c35656a77b7a36338428bf9be0eda0f))


### Documentation

* CHANGELOG 0.6.0, README, AGENTS.md, cmd/oo/main.go tree. ([e03fd62](https://github.com/eSlider/go-onlyoffice/commit/e03fd6220012c73b110f1e0b02a12971174eb290))

## [Unreleased]

## [0.7.0] — 2026-06-24

### Added — `office` TUI

- **Pane layout** — default **10% / 60% / 30%** split (nav / list / detail); drag vertical borders to resize; **Alt+1/2/3** still toggles panes.
- **Filter mode** (`f`) — live filter on nav + list; right pane is the search query; **Esc** clears.
- **Mail preview** — HTML bodies rendered colorized in the terminal (glamour); read-only scrollable document pane.
- **Mail infinite scroll** — loads the next page automatically when the cursor nears the end of the list.
- **Scrollbars** — vertical scrollbar on nav, list body, and detail preview when content exceeds the viewport.
- **Calendar** — unified calendars + events leaf with **Type** column and date range (−7d … +30d).
- **Tasks** — humanized status labels and relative deadlines in the list.
- **Projects** — open/closed row colors; status toggle in the detail form; **Save** sends `responsibleId` and status via `UpdateProjectStatus`.

### Changed — `office` TUI

- Flattened navigation: **Projects**, **Tasks**, **By project**, **Calendar**, **CRM**, **Mail**, **Users** (removed Browse; Calendar no longer splits Calendars/Events).
- Middle pane truncates overflowing cells with `…`; full text on the cursor or Space-selected row.
- Detail form tab order: Title → Description → Status → Save → Delete.
- **j/k** scrolls mail/file preview and read-only detail forms; mouse wheel scrolls detail when hovering the content area.

### Added — library

- `UpdateProjectStatus` — `PUT /api/2.0/project/{id}/status` for open/closed lifecycle.

## [0.5.1] — 2026-06-24

### Added — `office` TUI

- **Alt+1 / Alt+2 / Alt+3** — show/hide left (nav), middle (table), and right (detail) panes.
- Visible panes share **100% terminal width** evenly; pane content fills its column.
- Tab focus skips hidden panes.

### Changed — `office` TUI

- Split detail pane: form/document top (~72%), CRUD action bar bottom.
- Middle pane: multi-column table with sort, selection, and full-width columns.
- Project list columns: ID, Title, Tasks (open/closed), Documents, Users.
- Row selection auto-loads detail; files show document preview, entities show forms.

## [0.6.0] - 2026-04-24

### Added — library

- **Project & task Documents API** in [`files.go`](files.go):
  - `FileEntry`, `FolderEntry`, `ProjectFilesResponse` types.
  - `GetProjectFiles`, `GetTaskFiles`, `GetFile`.
  - `UploadProjectFile` — `POST /api/2.0/files/{folderId}/upload` into the
    project's `projectFolder` (resolved via `GetProjectByID` or first folder
    from `GetProjectFiles`).
  - `AttachFilesToTask` — `POST .../project/task/{id}/files` with form
    `files=<id>` (OnlyOffice expects **existing** file ids, not multipart).
  - `UploadTaskFile` — uploads via `UploadProjectFile` using the task's
    `projectOwner.id`, then attaches.
  - `DetachTaskFile` — `DELETE .../files?fileid=`.
  - `RenameFile` — `PUT /api/2.0/files/file/{id}.json` with JSON body.
  - `DeleteFiles` — `PUT /api/2.0/files/fileops/delete.json` with `fileIds`.
  - `DownloadFile` — `GetFile` then `GET` on `viewUrl` with `Authorization`.
  - Helpers: `FileEntryNumericID`, `FileEntryTitle`, `SafeLocalFileName`.
- **`putJSON`** on `*Client` in [`http.go`](http.go) for JSON PUT bodies.

### Added — CLI

- `oo projects files list|upload|download|rename|delete` — see
  [`cmd/oo/projects_files.go`](cmd/oo/projects_files.go); `list` supports
  `--folders`.
- `oo tasks files list|upload|detach` — see [`cmd/oo/tasks_files.go`](cmd/oo/tasks_files.go).

### Added — tests

- [`files_integration_test.go`](files_integration_test.go) — live roundtrip
  against OnlyOffice (same credential rules as `client_test.go`).
- [`files_test.go`](files_test.go) + `testdata/*.json` — envelope decode unit
  tests (no network).

## [0.5.0] - 2026-04-24

### Changed — CLI **BREAKING**

- **Subject-based command tree** (`tea`-style). The flat `oo verb-noun`
  layout is replaced with `oo <subject> <verb>`:

  | Old | New |
  |---|---|
  | `oo cal-list` | `oo calendar list` |
  | `oo cal-events` | `oo calendar events` |
  | `oo cal-add` / `cal-delete` | `oo calendar add` / `calendar delete` |
  | `oo task-list` | `oo tasks list` |
  | `oo task-add` | `oo tasks create` |
  | `oo task-update` | `oo tasks update` (deletion moved to `oo tasks delete`) |
  | `oo subtask-add` | `oo tasks subtask add` |
  | `oo crm-contacts` | `oo contacts list` (plus filtered `oo persons list` / `oo companies list`) |
  | `oo crm-add-contact --company …` | `oo companies create --name …` |
  | `oo crm-add-contact --person-first …` | `oo persons create --first …` |
  | `oo crm-deals` | `oo opportunities list` |
  | `oo crm-deals --stages` | `oo opportunities stages` |
  | `oo crm-add-deal` | `oo opportunities create` |
  | `oo crm-cases` | `oo cases list` |
  | `oo applications-sync` | `oo applications sync` |

- **New subjects**: `oo projects {list,get,milestones,create,update,delete}`,
  `oo users {list,self}` (plus top-level `oo whoami`), `oo crm-tasks
  {list,create,delete,categories}`, `oo cases {create,delete,member-add}`,
  `oo contacts {get,info-add}`.

- **Global `--output/-o` flag**: all list-like commands now support
  `--output table` (default; tabwriter-aligned, truncated to 80 chars
  per cell) and `--output json`. Nested `bidCurrency` flattened to its
  `abbreviation` in the table view.

- **Module aliases**: `oo calendar|cal`, `oo projects|prj`, `oo tasks|task`,
  `oo persons|person`, `oo companies|company`, `oo opportunities|deals|deal`,
  `oo cases|case`, `oo applications|apps`. `delete|rm` on every leaf that
  removes things.

### Added

- `cmd/oo/common.go` — shared `printTable(headers, rows)` and
  `printObject(v)` helpers that dispatch on the `--output` flag.
- `cmd/oo/users.go` — exposes `oo users list`, `oo users self`, `oo whoami`
  via the library's `GetUsers` + `SelfUserID`.
- `cmd/oo/projects.go` — full CRUD for projects backed by `CreateProject`,
  `UpdateProject`, `DeleteProject`, `GetProjectByID`, `GetProjectMilestones`.
- `cmd/oo/crm_tasks.go` — dedicated `oo crm-tasks` subject (distinct from
  project `oo tasks`).
- `cmd/oo/contacts.go` — unified contacts/persons/companies with shared
  list/filter implementation.

### Changed — library (minor)

- `newOO` in `cmd/oo/common.go` now calls `AuthenticateContext(cmd.Context())`
  so CLI aborts propagate to the auth request.

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
