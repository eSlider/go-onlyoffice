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
- Поиск по содержимому PDF в индексе OnlyOffice не работает (для PDF нет
  `attachment.content`) — только Office-форматы. Решение для PDF — свой индекс
  `oo_docs_text` (F6 #42), см. ниже.

# PDF и сканы — свой индекс (F6 #42)

## Проблема

`oo search --content "S1019"` не находил номер внутри PDF-счёта: в индексе
OnlyOffice PDF лежит только по имени.

## Почему PDF исключён (исходники CommunityServer)

Разобрано в `ONLYOFFICE/CommunityServer`:

- `web/core/ASC.Web.Core/Files/FileUtility.cs` — `CanIndex(fileName)` читает
  серверную настройку `files.index.formats` (в `web/studio/ASC.Web.Studio/web.appsettings.config`
  значение по умолчанию `".pptx|.xlsx|.docx"`).
- `web/studio/ASC.Web.Studio/Products/Files/Core/Search/FilesWrapper.cs` —
  `GetDocumentStream*` возвращает `null`, если `!FileUtility.CanIndex(Title)`,
  файл зашифрован или больше `MaxFileSize`.
- `module/ASC.ElasticSearch/Core/WrapperWithDoc.cs` + mapping в `Wrapper.cs` —
  маппинг `document.attachment.content` и ingest-pipeline `attachments`
  формат-агностичны: они распарсят любой поток.

Вывод: PDF исключён **только настройкой** `files.index.formats`; жёсткого
ограничения на формат в коде нет.

## Варианты и решение

| # | Вариант | Оценка |
|---|---------|--------|
| a | Включить `.pdf` в `files.index.formats` + reindex | Правка сервера OO; настройка может потеряться при обновлении; полный reindex 39k док-в; Tika **не OCR** — сканы без текстового слоя дадут пустой контент. Отклонён без решения PO. |
| b | Server-side ingest/attachment для PDF | По факту то же, что (a): сервер кормит поток только для `CanIndex`. |
| c | **Свой индекс** `oo_docs_text`, наполняемый `internal/docpipe` | **Выбран.** Сервер OO не трогаем; детерминированно; работает OCR для сканов; независимо от обновлений OO; любые форматы; фильтры папка/тип. |
| d | Локальный поиск без индекса | Отклонён как основной: качаем и извлекаем на каждый запрос, нет выдачи/ранжирования/highlight. |

Итог: **вариант c**. Индекс OnlyOffice (`files_file`) не изменяется; наш
индекс живёт рядом.

## Устройство

- `file_es_text.go` — `ESTextIndex` (`Name() = "es-text"`):
  `Ensure` (создаёт индекс с явным маппингом), `Put` (bulk, `refresh`),
  `Delete` (по `id`), `Search` (`multi_match` по `title^2` + `content`,
  фильтры `folder`/`ext`, highlight).
- `file_text_index.go` — `TextIndexer`: листает папки (`FileStore.List`),
  качает файлы (`FileStore.Download`), извлекает текст через
  `internal/docpipe` (`pdftotext`, для сканов — `ocrmypdf`/`tesseract`),
  пишет в `TextIndex`. Пул воркеров (по умолчанию 3).
- CLI: `oo index folder|files` наполняет индекс; `oo search --backend own`
  ищет по нему.

### Встроенные вложения PDF

Оцифрованные PDF несут вложения (`<doc>.md` — текст/таблицы скана,
`<doc>.yaml`/`.json` — метаданные, `.xml` — EN 16931 CII eRechnung,
`factur-x.xml` у ZUGFeRD; см. `office-assistant/docs/reference/document-metadata.md`).
`TextIndexer` обходит их: `pdfdetach -list` перечисляет, `-save` сохраняет,
каждое вложение проходит штатный `docpipe.ToMarkdown` (PDF/картинки → OCR,
`.md`/`.txt` — как есть). Форматы, которые docpipe не конвертирует
(`.xml`/`.html` — снимаются теги; `.json`/`.csv` — как текст), извлекаются
текстом; нечитаемые — пропускаются.

Текст склеивается: тело, затем по секции на вложение с маркером
`[attachment: <имя>]` (функция `docpipe.JoinWithAttachments`). Индекс — тот же
`file_id`, upsert идемпотентен. Нет вложений или pdfdetach/формат нечитаем —
индексируется тело (без падения).

Поля `oo_docs_text`:

| поле | тип | смысл |
|------|-----|-------|
| `id` | keyword | id файла Documents |
| `title` | text (+`.keyword`) | имя файла |
| `folder` | keyword | id папки |
| `ext` | keyword | расширение |
| `content` | text | извлечённый текст (pdftotext/OCR) |

## CLI

```bash
set -a; . .env; set +a          # ONLYOFFICE_URL/USER/PASS + ONLYOFFICE_ES_URL
oo index folder 634             # PDF в папке 634
oo index folder 634 --recursive --exts pdf,png --limit 100
oo index files 3576 3578        # точечно
oo index folder 634 --dry-run   # показать план, ничего не менять

oo search "S1021" --content --backend own
oo search "S1021" --backend own --folder 634 --json
```

`--backend` у `oo search`: `oo` (по умолчанию, индекс OnlyOffice) или `own`
(наш `ONLYOFFICE_ES_TEXT_INDEX`).

## Переменные (дополнение)

| env | default | смысл |
|-----|---------|-------|
| `ONLYOFFICE_ES_TEXT_INDEX` | `oo_docs_text` | индекс своего конвейера |

`ONLYOFFICE_ES_URL` — общий для обоих индексов.

## Тесты

```bash
go test -run 'ESText|TextIndexer|Index' ./ ./cmd/oo/        # unit, без сети
ONLYOFFICE_ES_URL=http://127.0.0.1:9200 \
  go test -tags=integration -run TestIntegrationESTextIndex -v .
```

Интеграционный тест создаёт временный индекс, наполняет, ищет по контенту,
проверяет фильтры и удаление, затем удаляет индекс;
`TestIntegrationESTextIndexPDFAttachment` индексирует
`testdata/pdf-with-attachment.pdf` реальным конвейером (pdfdetach + pdftotext)
и ищет токен, лежащий только во вложении. Unit-тесты используют
fake-store/fake-extractor и не требуют pdftotext/OCR (парсер списка, склейка
`JoinWithAttachments`, снятие тегов `xmlToText` — чистые).

## Грабли

- Наполнение — ручное (`oo index`); после изменения/добавления PDF повтори.
  Повтор идемпотентен (upsert по id файла).
- В индексе ищется только то, что проиндексировано; `oo index` качает каждый
  файл и (для сканов) гоняет OCR — это медленно, отсюда `--limit`/`--exts`.
- `folder` фильтруется как id папки, а не как путь.
- Дубликаты (напр. `S1055.pdf` и `2026-08-20-S1055-…`) дадут несколько строк —
  это ожидаемо, дедуп — на стороне потребителя.
- Вложения: нужен `pdfdetach` (poppler); если его нет — индексируется только
  тело. Вложенный PDF/картинка с плохим текстовым слоем проходит OCR, это
  медленно. `.json`-метаданные (CuraSoft) индексируются как текст и могут
  добавить шумовых токенов.
