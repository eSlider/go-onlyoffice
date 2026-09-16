---
type: reference
status: current
related:
  - README.md
  - file_es.go
---

# Elasticsearch — полнотекстовый поиск OnlyOffice

## Что это

Полнотекстовый поиск OnlyOffice Workspace работает на **Elasticsearch**.
Клиент на сервере — NEST. Индекс — имя таблицы.

Для файлов индекс `files_file`:

| поле | тип | смысл |
|------|-----|-------|
| `id` | integer | id файла (тот же, что в REST/Documents) |
| `title` | text (`whitespacecustom`) | имя файла |
| `tenantId` | integer | тенант (портал) |
| `folders` | nested | список папок: `folderId` (строка), `id`, `tenantId` |
| `document.attachment.content` | text (`document`) | извлеченный текст (ingest-attachment) |
| `document.attachment.content_type` | text | MIME |

Важно:
- Живой сервер — **Elasticsearch 7.16.3**, кластер `elasticsearch`.
- REST `GET /api/2.0/files/@search/{query}` ищет **только по имени в БД**
  (`fileDao.Search`), ES не задействует. Для поиска по содержимому нужен
  прямой ES — это и делает `oo search`.
- `title` analyzer `whitespacecustom` режет по пробелам и lower-case. Полное
  имя файла — один токен (`Rechnung-4711.pdf`), поэтому поиск по имени ищет
  слово целиком, а не подстроку.
- `document.attachment.content` заполняется **только для Office-форматов**
  (docx / xlsx / pptx). У PDF/txt, залитых через API, контент не извлекается.
- Индексация асинхронная (TeamLabSvc) — файл появляется в ES не мгновенно.

## Доступ

ES слушает `127.0.0.1:9200` **внутри** VM OnlyOffice. Снаружи порт закрыт,
SSH в VM открыт на хосте как `127.0.0.1:32` (контейнер `onlyoffice-v2`,
QEMU). Схема — SSH-туннель.

```bash
# из корня go-onlyoffice (ключ и хост — как в infra-доках)
ssh -f -N -o ControlMaster=no -o ControlPath=none \
    -p 32 -i ~/.ssh/id_ed25519 \
    -L 9200:127.0.0.1:9200 root@127.0.0.1

curl -s http://127.0.0.1:9200/ | head            # tagline + version
curl -s 'http://127.0.0.1:9200/_cat/indices?h=index,docs.count'
```

`-o ControlMaster=no -o ControlPath=none` обязательны: иначе forward уходит
в persistent master-соединение из `~/.ssh/config` и порт остаётся занят.

Проверить, что туннель жив:

```bash
curl -s http://127.0.0.1:9200/files_file/_count
```

## Переменные

| env | default | смысл |
|-----|---------|-------|
| `ONLYOFFICE_ES_URL` | — (обязателен) | `scheme://host:port` ES |
| `ONLYOFFICE_ES_INDEX` | `files_file` | индекс |
| `ONLYOFFICE_TENANT` | пусто (все) | фильтр `tenantId` |

Имена — в [`.env.example`](../.env.example). Секретов нет: ES без пароля.

## CLI

```bash
ONLYOFFICE_ES_URL=http://127.0.0.1:9200 oo search "Rechnung"
ONLYOFFICE_ES_URL=http://127.0.0.1:9200 oo search "Mahngebühr" --content
oo search "Rechnung" --folder 649 --limit 50 --json
```

Флаги: `--content` (искать и по тексту), `--folder ID` (папка
`folders.folderId`), `--limit N` (по умолчанию 20, максимум 200),
`--json` = `-o json`.

## Библиотека

`file_es.go` — `ESSearcher` (`Name() = "elasticsearch"`), прямой ES REST на
stdlib `net/http`:

```go
es, _ := onlyoffice.NewESSearcher(onlyoffice.ESConfigFromEnv())
hits, _ := es.Search(ctx, onlyoffice.SearchQuery{
    Text: "Rechnung", InContent: true, Limit: 20,
})
```

Запрос: `multi_match` по `title^2` (+ `document.attachment.content` при
`InContent`), фильтры `tenantId` и `folders.folderId`, `_source`
id/title/folders, `highlight` для фрагмента. Ответ → `[]SearchHit` (модель из
эпика #34; пока объявлена в `file_es.go`, переедет в `file_core.go` с F1 #35).

## Тесты

```bash
# unit — чистые builders/парсеры, без сети
go test ./ -run ES

# integration — нужен ONLYOFFICE_ES_URL (+ креды REST для залива)
set -a; . .env; set +a
ONLYOFFICE_ES_URL=http://127.0.0.1:9200 ONLYOFFICE_TENANT=1 \
  go test -tags=integration -run TestIntegrationESSearch -v .
```

Интеграционный тест заливает временный xlsx (в имени и в ячейке — уникальные
токены), ждёт индексации, проверяет поиск по имени и по содержимому, затем
удаляет проект.

## Грабли

- `locale`/версия ES: 7.16.3, `_search` совместим с REST 7.x.
- ES без auth и слушает только localhost — туннель обязателен.
- Фильтр `tenantId` сузит выдачу; без него видны документы всех тенантов.
- Поиск по содержимому PDF, залитых через API, не работает (нет
  `attachment.content`) — только Office-форматы.
