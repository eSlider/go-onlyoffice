# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
