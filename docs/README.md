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
- [rclone-webdav.md](rclone-webdav.md) — rclone-монтирование Documents
  (`deploy/docker-compose.rclone-webdav.yml`), smoke, ограничения.
- [crm-associations.md](crm-associations.md) — правила ассоциаций CRM.

## Тесты
Команды и туннели — раздел Testing в [README.md](../README.md#testing).
