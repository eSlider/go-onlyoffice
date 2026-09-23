---
type: reference
status: current
related:
  - docs/elasticsearch.md
  - docs/unified-file-client.md
  - docs/community-server-db.md
---

# Поиск и индексация

Четыре разных контура поиска. Не путать: у каждого свой индекс, свои входы и
свой способ обновления.

| Контур | Что ищет | Индекс | Обновление | Вход |
|--------|----------|--------|------------|------|
| REST `@search` | только имена в БД | нет | — (живой запрос) | `oo search` (по умолчанию `--backend oo`) |
| ES `files_file` | имя + текст Office | Elasticsearch портала | сервер, асинхронно | `oo search --content` |
| ES `oo_docs_text` | PDF/сканы (свой) | Elasticsearch портала | `oo index` | `oo search --backend own` |

## Карта кода

- `filestore_core.go` — интерфейсы `Searcher`, модели `SearchQuery`/`SearchHit`.
- `filestore_es.go` — `ESSearcher` (индекс OnlyOffice `files_file`).
- `filestore_es_text.go` — `ESTextIndex` (`oo_docs_text`): `Ensure`, `Put`, `Delete`,
  `Search`.
- `filestore_text_index.go` — `TextIndexer`: обход папок (`FileStore.List`), download
  (`FileStore.Download`), извлечение текста (`internal/docpipe`), запись в
  `TextIndex`; пул воркеров.
- `filestore_facade.go` — связка бэкендов (`Files().Search()`, порядок и fallback).
- CLI: `cmd/oo/search.go`, `cmd/oo/index.go`.
- Разовые бинари для match (`ooscan`, `pdfamount`) живут в приватном
  `oo-workspace`.
- `internal/docpipe` — текст из PDF (pdftotext), для сканов OCR
  (ocrmypdf/tesseract), вложения PDF (pdfdetach).

## Поиск

```bash
# имя, индекс портала
oo search "Rechnung" --limit 50 --json
# имя + текст Office (docx/xlsx/pptx)
oo search "Mahngebühr" --content
# свой индекс: PDF и сканы
oo search "S1019" --backend own --folder 649
```

Флаги `oo search`: `--content`, `--folder ID`, `--limit N`, `--backend oo|own`,
`--substring`, `--json`. Требует `ONLYOFFICE_ES_URL` (см.
[elasticsearch.md](elasticsearch.md)); `--backend own` дополнительно ничего не
требует от сервера — читает `oo_docs_text`.

Почему не REST: `GET /api/2.0/files/@search/{query}` ищет только имя в БД
(`fileDao.Search`), ES не трогает. Почему PDF не в `files_file`: сервер индексит
контент только для форматов из `files.index.formats` (по умолчанию
`.pptx|.xlsx|.docx`) — отсюда свой `oo_docs_text`.

## Обновление своего индекса (`oo index`)

```bash
# одна папка
oo index folder 649 --recursive --exts pdf
# точечно по id
oo index files 3576 3578
# без записи: что было бы проиндексировано
oo index folder 649 --recursive --dry-run
```

Флаги: `--recursive`, `--exts pdf` (по умолчанию), `--limit N`,
`--workers 3`, `--lang deu+eng`, `--min-chars N` (порог текстового слоя, ниже
которого включается OCR), `--work-dir`, `--backend rest|dav`, `--dry-run`,
`--json`.

Свойства:

- Идемпотентно: upsert по `id` файла; повтор не двоит.
- Сервер OnlyOffice не меняется: индекс живёт рядом (`ONLYOFFICE_ES_TEXT_INDEX`,
  по умолчанию `oo_docs_text`).
- Медленно на сканах (OCR на каждый файл). Ограничивай `--folder`/`--limit`,
  не индексируй корень целиком.
- Индексация PDF в `files_file` не делается — только `oo_docs_text`.

## Bulk-инструменты

Плоские TSV-инструменты (`ooscan`, `pdfamount`) и сверка Excel живут в
приватном `oo-workspace`, не в публичной библиотеке.

## Грабли

- ES слушает только `127.0.0.1:9200` внутри VM — SSH-туннель обязателен
  (`-o ControlMaster=no -o ControlPath=none`, см. [elasticsearch.md](elasticsearch.md)).
- Портальные листинги/скачивание упираются в 429; все bulk-пути идут через
  `DoRetry` (линейный бэкофф), `ooscan` дополнительно спит 350 мс на папку.
- `files_file` обновляется сервером асинхронно — свежий файл виден не сразу.
- `title` analyzer `whitespacecustom`: имя — один токен, подстрока только через
  `--substring` (или wildcard).
