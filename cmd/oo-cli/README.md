# oo-cli

Cobra CLI wrapper around [`github.com/eslider/go-onlyoffice`](../..). Replaces the Python scripts that used to live under `eSlider/cv/bin/office`.

## Install

```bash
go install github.com/eslider/go-onlyoffice/cmd/oo-cli@latest
```

Ensure `$(go env GOPATH)/bin` is on `PATH`.

## Config

Copy [`../../.env.example`](../../.env.example) to `.env` in the working directory:

| Variable | Alias | Default |
|---|---|---|
| `ONLYOFFICE_URL` | `ONLYOFFICE_HOST` | — |
| `ONLYOFFICE_USER` | `ONLYOFFICE_NAME` | — |
| `ONLYOFFICE_PASS` | `ONLYOFFICE_PASSWORD` | — |
| `ONLYOFFICE_CALENDAR_ID` | — | `1` |
| `ONLYOFFICE_PROJECT_ID` | `ONLYOFFICE_CALENDAR_PROJECT_ID` | `33` |

## Commands

| Python (legacy) | Go CLI |
|---|---|
| `list-calendars.py` | `oo-cli cal-list` |
| `list-events.py` | `oo-cli cal-events [--start] [--end]` |
| `add-event.py` | `oo-cli cal-add TITLE START END [--calendar] [--description] [--all-day]` |
| `delete-event.py` | `oo-cli cal-delete ID [ID...]` |
| `list-tasks.py` | `oo-cli task-list [--project] [--status] [--all] [--verbose]` |
| `add-task.py` | `oo-cli task-add TITLE [--project] [--description] [--deadline] [--priority]` |
| `add-subtask.py` | `oo-cli subtask-add PARENT TITLE [TITLE...]` |
| `update-task.py` | `oo-cli task-update ID open\|closed` or `--delete` |
| `list-contacts.py` | `oo-cli crm-contacts [--companies\|--persons] [--search]` |
| `add-contact.py` | `oo-cli crm-add-contact --company ...` or `--person-first` / `--person-last` |
| `list-deals.py` | `oo-cli crm-deals` / `oo-cli crm-deals --stages` |
| `add-deal.py` | `oo-cli crm-add-deal TITLE [--stage] [--bid] [--contact]` |
| `list-cases.py` | `oo-cli crm-cases` |
| `sync-applications.py` | `oo-cli applications-sync --path .../applications/2026 [--apply]` |

## Notes

- The CLI loads `.env` via `godotenv`; the library itself does not.
- `applications-sync` defaults to dry-run; pass `--apply` to write to CRM.
