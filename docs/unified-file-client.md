---
type: reference
status: current
related:
  - README.md
  - file_core.go
  - file_facade.go
  - docs/elasticsearch.md
  - docs/community-server-db.md
---

# Unified file client — контракт файловых бэкендов

## Что это

Один файловый клиент на все бэкенды (эпик #34). Модель и интерфейсы —
`file_core.go`. Фасад `FileClient` — `file_facade.go`. Бэкенды:
REST, WebDAV, SQL (PostgreSQL/MySQL), Elasticsearch. Правило одно:
код зовёт `c.Files()` и не знает про транспорт.

## Модель

- `Kind` — `File` (0) или `Folder` (1).
- `Entry` — бэкенд-независимая строка: `ID`, `ParentID`, `Title`, `Kind`,
  `Size`, `MIME`, `Created`, `Modified`, `Updated` (сырая строка API),
  `Version`, `Provider`, `FilesCount`/`FoldersCount` (папки).
  Чего бэкенд не даёт — остаётся в нуле.
- `SearchQuery` — `Text`, `InContent`, `FolderID`, `Extensions`, `Limit`.
- `SearchHit` — `Entry` + `Score`, `Highlight`, `Path`.

## Интерфейсы

`FileStore` — операции с файлами:

```go
type FileStore interface {
    Name() string
    List(ctx, parentID) ([]Entry, error)
    Stat(ctx, id) (Entry, error)
    CreateFolder(ctx, parentID, title) (Entry, error)
    Upload(ctx, parentID, title, r) (Entry, error)
    Download(ctx, id, w) (int64, error)
    Move(ctx, ids, parentID) error
    Copy(ctx, ids, parentID) error
    Rename(ctx, id, title) error
    Delete(ctx, ids) error
}
```

`Searcher` — поиск (необязательный):

```go
type Searcher interface {
    Search(ctx, q SearchQuery) ([]SearchHit, error)
    Name() string
}
```

`TextIndex` (`file_es_text.go`) — свой индекс: `Put`, `Delete`, `Search`,
`Name`. `ESTextIndex` реализует и `Searcher`, и `TextIndex`.

## Бэкенды

| бэкенд | провайдер | файл | что умеет |
|--------|-----------|------|-----------|
| REST | `rest` | `file_rest.go` | read + write, Documents API |
| WebDAV | `dav` | `file_dav.go` | read + write, Documents fileops |
| SQL | `postgres` / `mysql` | `file_pg.go` | **read-only** |
| OnlyOffice ES | `elasticsearch` | `file_es.go` | поиск (имя + контент Office) |
| свой ES-индекс | `es-text` | `file_es_text.go` | поиск + запись (PDF/сканы) |

- REST: `Stat` знает только файлы; папки — через `List`.
- WebDAV: `Move`/`Copy`/`Delete` сперва `Stat`-ят id (папка/файл), потом зовут
  fileops.
- SQL: `List`/`Stat`/`Download`/`Search` (по имени). Все write-методы →
  `ErrReadOnly`. `Download` идёт в S3/MinIO по layout портала.
- OnlyOffice ES: индекс `files_file`, контент только для docx/xlsx/pptx.
- Свой ES: индекс `oo_docs_text`, контент из `internal/docpipe`, в т.ч.
  встроенные PDF-вложения.

## Фасад `FileClient`

`c.Files()` → `*FileClient`. Он же реализует `FileStore`, старый код
компилируется.

- `Read()` — первый зарегистрированный из `readOrder`:
  `postgres` → `rest` → `dav`.
- `Write()` — первый из `writeOrder`: `rest` → `dav`. SQL не пишет.
- `Search()` — первый из `searchOrder`: `elasticsearch`. Нет бэкенда →
  ошибка (`ONLYOFFICE_ES_URL`).
- `RegisterStore(name, s)` / `RegisterSearcher(name, s)` — добавить бэкенд.

Fallback:

- `List`/`Stat` идут по `readOrder`; переходят к следующему только на
  transient-ошибке (429/502/503/504). Иначе ошибка финальная.
- `Download` **без** fallback: часть байтов уже в `w`, второй бэкенд допишет.
- Запись (`CreateFolder`/`Upload`/`Move`/`Copy`/`Rename`/`Delete`) — только
  `Write()`, без fallback.

`newFileClient` сам кладёт `rest` и `dav`; ES-поиск — если задан
`ONLYOFFICE_ES_URL`. SQL-стор регистрирует вызывающий: фасад создаётся на
каждый `c.Files()`, регистрируй на том же экземпляре.

```go
f := c.Files()
if pg, err := onlyoffice.NewPGStore(onlyoffice.PGConfigFromEnv()); err == nil {
    defer pg.Close()
    f.RegisterStore(onlyoffice.ProviderPG, pg)
}
entries, _ := f.List(ctx, "649")   // пойдёт в SQL
```

## CLI

```bash
# поиск: --backend oo (индекс OnlyOffice) | own (свой oo_docs_text)
oo search "Rechnung"
oo search "Mahngebühr" --content
oo search "S1021" --content --backend own --folder 634 --limit 50 --json

# наполнение своего индекса (PDF/сканы, idempotent upsert по file id)
oo index folder 634
oo index folder 634 --recursive --exts pdf,png --limit 100
oo index files 3576 3578
oo index folder 634 --dry-run
```

`oo index` флаги: `--recursive`, `--exts` (default `pdf`), `--limit`,
`--workers` (3), `--lang` (`deu+eng`), `--min-chars`, `--work-dir`,
`--backend rest|dav`, `--dry-run`, `--json`.

Библиотека:

```go
idx, _ := onlyoffice.NewESTextIndex(onlyoffice.ESTextConfigFromEnv())
ti := onlyoffice.NewTextIndexer(store, idx)   // store = FileStore
res, _ := ti.IndexFolder(ctx, "634", onlyoffice.IndexOptions{Recursive: true})
```

## Env (только имена)

| env | default | кто читает |
|-----|---------|------------|
| `ONLYOFFICE_ES_URL` | — | ES (оба индекса), обязателен |
| `ONLYOFFICE_ES_INDEX` | `files_file` | индекс OnlyOffice |
| `ONLYOFFICE_ES_TEXT_INDEX` | `oo_docs_text` | свой индекс |
| `ONLYOFFICE_TENANT` | пусто | фильтр `tenantId` |
| `ONLYOFFICE_DSN` | — | SQL DSN (MySQL/PostgreSQL) |
| `ONLYOFFICE_PG_DRIVER` | auto | `postgres` / `mysql` |
| `ONLYOFFICE_PG_TENANT` | `ONLYOFFICE_TENANT` | SQL tenant |
| `ONLYOFFICE_PG_HOST` `_PORT` `_USER` `_PASSWORD` `_DBNAME` `_SSLMODE` | — | DSN по частям |
| `MINIO_ENDPOINT` `MINIO_BUCKET` `MINIO_ACCESS_KEY` `MINIO_SECRET_KEY` | — | download SQL-стора |
| `OO_URL` `OO_USER` `OO_PASS` | — | CLI-алиасы |

Имена — в [`.env.example`](../.env.example). Секретов в репо нет.

## Ограничения

- OnlyOffice ES: контент только Office-форматов. PDF — только по имени.
  Встроенные вложения PDF сервер не индексирует.
- Свой индекс `oo_docs_text`: покрывает PDF/сканы и вложения (pdfdetach), но
  наполняется вручную (`oo index`) и идемпотентен. Фильтр `folder` — id папки,
  не путь. Дубли дают несколько строк — дедуп на потребителе.
- SQL: read-only. `InContent` игнорируется (только имя). Download — через
  MinIO-схему, не HTTP.
- ES: без auth, слушает localhost внутри VM — нужен SSH-туннель
  (см. [elasticsearch.md](elasticsearch.md)).
- `oo index` качает каждый файл и для сканов гоняет OCR — медленно; отсюда
  `--limit` и `--exts`. Нужен `pdfdetach` (poppler); без него — только тело PDF.

## Как добавить бэкенд

1. Файл `file_<name>.go`. Реализуй `FileStore` (`Name` + 9 методов). Нужен
   поиск — добавь `Searcher`; нужна запись своего индекса — `TextIndex`.
2. Добавь const провайдера рядом с `ProviderREST`/`ProviderDAV`.
3. Зарегистрируй: в `newFileClient` или снаружи через
   `RegisterStore`/`RegisterSearcher`.
4. Внеси имя в `readOrder` / `writeOrder` / `searchOrder`.
5. Есть CLI-команда — добавь значение в `--backend`.
6. Тесты: unit (чистые builders/парсеры, без сети) + интеграционный
   (`//go:build integration`, skip без кред).

## Тесты

```bash
go test ./...                                  # unit, без сети
go test -tags=integration ./...                # live (креды в .env)
go test ./ -run 'FileStore|Facade|ESText|PG'
```

## См. также

- [elasticsearch.md](elasticsearch.md) — индекс OnlyOffice и свой `oo_docs_text`.
- [community-server-db.md](community-server-db.md) — SQL-стор и схема БД.
