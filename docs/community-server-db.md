---
type: reference
status: current
related:
  - README.md
  - filestore_pg.go
  - docs/elasticsearch.md
---

# Community Server DB — прямой SQL-доступ (read-only)

## Что это

Бэкенд `pgStore` (`filestore_pg.go`) читает файлы и папки **напрямую из БД
Community Server**, без HTTP-слоя. Реализует `FileStore` (`List`/`Stat`/
`Download`) и `Searcher` по имени. Запись запрещена: все write-методы
возвращают `ErrReadOnly`.

## Что за БД (research, live)

Проверено на VM `onlyoffice-v2` (SSH `127.0.0.1:32`):

- Community Server работает на **MySQL 8.0**, не на PostgreSQL.
  - Хост: `127.0.0.1:3306` внутри VM, база `onlyoffice`.
  - Конфиг: `/etc/onlyoffice/communityserver/appsettings.production.json`,
    `providerName: MySql.Data.MySqlClient`.
  - Таблицы: `files_file`, `files_folder`, `files_folder_tree`,
    `files_security`, тенанты — `tenants_tenants` (не `tenants`).
- PostgreSQL 16 в той же VM — **наш** контур (`edw_docs`, роли `edw`/`edw_ro`,
  office-assistant), к OnlyOffice отношения не имеет. `files_file` в PG нет.
- Портал хранит файлы в **S3/MinIO** (DiscStorage только для мелочи).
  Бакет `office`, объект — по ключу (см. ниже).

Вывод: бэкенд назван по issue «PostgreSQL», но живой источник — MySQL.
`database/sql` + драйвер по DSN: `mysql` для MySQL, `pgx` для PostgreSQL.
`Name()` возвращает фактический движок (`mysql` или `postgres`).

## Схема

`files_file` — одна строка **на версию** (PK `tenant_id, id, version`):

| поле | смысл |
|------|-------|
| `id` | id файла (тот же, что в REST/ES) |
| `version` | номер версии этой строки |
| `version_group` | номер версии |
| `current_version` | `1` = текущая версия, `0` = старая |
| `folder_id` | id родительской папки |
| `title` | имя файла с расширением |
| `content_length` | размер в байтах |
| `create_on`, `modified_on` | даты (UTC, без зоны) |
| `tenant_id` | тенант (портал) |

`files_folder`: `id`, `parent_id`, `title`, `create_on`, `modified_on`,
`tenant_id`. `files_folder_tree`: `folder_id`, `parent_id`, `level` — готовое
дерево, пока не используется.

Текущую строку файла берём по `current_version = 1`.

## Доступ (SSH-туннель)

MySQL слушает только `127.0.0.1:3306` внутри VM. Снаружи — SSH-туннель
(SSH в VM открыт как `127.0.0.1:32`):

```bash
ssh -f -N -o ControlMaster=no -o ControlPath=none \
    -p 32 -i ~/.ssh/id_ed25519 \
    -L 3306:127.0.0.1:3306 root@127.0.0.1

# MySQL DSN затем:
# root:<pw>@tcp(127.0.0.1:3306)/onlyoffice?parseTime=true
```

Любой свободный локальный порт подойдёт (напр. `13306`); тогда тот же порт —
в DSN. `-o ControlMaster=no -o ControlPath=none` обязательны: иначе forward
уходит в persistent master из `~/.ssh/config`.

Креды MySQL — в конфиге Community Server внутри VM:
`/etc/onlyoffice/communityserver/appsettings.production.json` →
`ConnectionStrings.connectionString` (поля `User ID`, `Password`), база
`onlyoffice`. В самом MySQL-контейнере (`onlyoffice-mysql-server`) база пустая;
рабочий сервер — host-mysqld на `127.0.0.1:3306` (207 таблиц). Не печатать
пароль.

## Переменные

| env | default | смысл |
|-----|---------|-------|
| `ONLYOFFICE_DSN` | — | DSN драйвера (MySQL `...@tcp(...)/...` или `postgres://...`) |
| `ONLYOFFICE_PG_DRIVER` | авто | `postgres` или `mysql`; иначе по форме DSN |
| `ONLYOFFICE_PG_TENANT` | `ONLYOFFICE_TENANT` | фильтр `tenant_id` (пусто = все) |
| `ONLYOFFICE_PG_HOST/PORT/USER/PASSWORD/DBNAME/SSLMODE` | — | собрать PG DSN, если `ONLYOFFICE_DSN` пуст |

Имена — в [`.env.example`](../.env.example). Секретов нет.

## Использование

Напрямую: `NewPGStore(PGConfigFromEnv())`.

Через фасад (эпик #34): SQL-стор регистрируется на `FileClient`. После этого
`Read()` и все чтения (`Stat`/`List`) идут в БД, `Write()` остаётся REST/DAV.

```go
c := onlyoffice.NewClient(onlyoffice.GetEnvironmentCredentials())

sql, err := c.SQLFileStore()            // открыть из env; caller закрывает
if err != nil { /* нет DSN / нет связи */ }
if closer, ok := sql.(interface{ Close() error }); ok { defer closer.Close() }

f := c.Files()
f.RegisterStore(onlyoffice.ProviderPG, sql)
e, _ := f.Stat(ctx, "19423")            // e.Provider == "mysql" — ответил SQL
```

`Client.FileStore("pg"|"sql"|"postgres"|"mysql")` тоже отдаёт SQL-стор
(открывает из env). Если DSN нет/битый — возвращается не `nil`, а заглушка,
чей метод отдаёт ошибку открытия; ошибку как таковую даёт `SQLFileStore()`.
Различить бэкенд в ответе можно по `Entry.Provider` (`mysql` у SQL, `rest` у
REST).

## Download (MinIO)

`Download` не ходит в REST. Ключ объекта собирается из строки `files_file`:

```
00/00/<tenant>/files/folder_<shard>/file_<id>/v<version>/content.<ext>
shard = (id/1000 + 1) * 1000
```

`shard` — не `folder_id`, а следующая тысяча над `id` (файл 3727 →
`folder_4000`). Проверено live по бакету `office`.

Стриминг переиспользует `downloadMinioObject` из `storage_fallback.go`
(та же подпись SigV4 и `MINIO_*`), без дублирования.

Ограничение: схема валидна только для файлов, лежащих в **MinIO/S3** (старые
папки). Файлы в **Disc**-хранилище портала (`Data/Products/Files/...`, новые
папки) по этому ключу недоступны — `Download` вернёт `404`. Если
`MINIO_ACCESS_KEY`/`MINIO_SECRET_KEY` не заданы, `Download` вернёт явную
ошибку; `Stat`/`List`/`Search` работают и без них.

## Тесты

```bash
go test ./...                              # unit: rebind, csObjectKey, маппинг
go test -race ./...

# integration (нужен DSN; skip без него)
ONLYOFFICE_DSN='root:<pw>@tcp(127.0.0.1:3306)/onlyoffice?parseTime=true' \
ONLYOFFICE_PG_TENANT=1 \
ONLYOFFICE_PG_TEST_FILE_ID=19423 \
ONLYOFFICE_PG_TEST_FOLDER_ID=676 \
  go test -tags=integration -run 'TestIntegrationPGStore|TestIntegrationSQLFacade' -v ./...

# плюс MINIO_* для сверки Download с REST (иначе этот шаг skip)
MINIO_ENDPOINT=http://127.0.0.1:9000 MINIO_BUCKET=office \
MINIO_ACCESS_KEY=... MINIO_SECRET_KEY=... \
  go test -tags=integration -run TestIntegrationPGStore -v ./...
```

- `TestIntegrationPGStore` — `Stat`/`List`/`Download` SQL против REST и
  `ErrReadOnly` у write-методов.
- `TestIntegrationSQLFacade` — SQL-стор, зарегистрированный на фасаде, реально
  обслуживает чтения: `Read().Name()` = SQL-бэкенд, `Entry.Provider == "mysql"`
  (у REST — `"rest"`), сверка `Stat`/`List` с REST, и прямой
  `Client.FileStore("pg")`.

Без `ONLYOFFICE_DSN` оба теста делают чистый `skip`.

## Грабли

- MySQL хранит `datetime` без зоны; `parseTime=true` (ставится автоматически)
  читает их как UTC. REST отдаёт `+02:00` — сравнивать моменты, не строки.
- `GetFile` (REST) не отдаёт `contentLength` — размер сверять с `Stat` SQL.
- Один файл = много строк `files_file` (по версиям). Без `current_version = 1`
  получите дубликаты.
- `folder_id` не входит в ключ MinIO; ключ считает `shard` от `id`.
- Searcher SQL ищет только по имени (`LIKE`). Контент — Elasticsearch
  ([elasticsearch.md](elasticsearch.md)).
