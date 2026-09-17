---
type: reference
status: current
related:
  - README.md
---

# go-onlyoffice — docs

Индекс справочников. Общее — [README.md](../README.md), правила — [AGENTS.md](../AGENTS.md).

## Файлы
- [unified-file-client.md](unified-file-client.md) — единый файловый клиент:
  `Entry`/`FileStore`/`FileClient`, бэкенды REST/DAV/SQL/ES, env, как добавить
  бэкенд.
- [community-server-db.md](community-server-db.md) — read-only SQL-бэкенд
  (MySQL/PostgreSQL): схема, SSH-туннель, DSN, MinIO download.
- [elasticsearch.md](elasticsearch.md) — поиск: индекс OnlyOffice `files_file`
  и свой `oo_docs_text` (PDF/сканы), туннель.
- [index-and-search.md](index-and-search.md) — карта контуров поиска и как
  обновлять индексы (`oo index`, `ooscan`/`pdfamount` для match).
- [rclone-webdav.md](rclone-webdav.md) — rclone-монтирование Documents
  (`deploy/docker-compose.rclone-webdav.yml`), smoke, ограничения.
- [crm-associations.md](crm-associations.md) — правила ассоциаций CRM.
- [rate-limiting.md](rate-limiting.md) — rate limit, exponential backoff,
  `Retry-After`, общий cooldown против 429; env `OO_RATE_LIMIT`/`OO_BURST`/
  `OO_RETRY_*`.

## Тесты
Команды и туннели — раздел Testing в [README.md](../README.md#testing).
