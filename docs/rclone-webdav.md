---
type: reference
status: current
related:
  - deploy/docker-compose.rclone-webdav.yml
  - docs/unified-file-client.md
---

# rclone WebDAV mount — OnlyOffice Documents как файловая ФС

`deploy/docker-compose.rclone-webdav.yml` монтирует WebDAV-дерево OnlyOffice
через `rclone`. Не systemd, только compose. Контейнер — `rclone-webdav`.

## Что монтируется

- Источник — сайдкар `oo-webdav` (`ghcr.io/eslider/oo-webdav`).
- Адрес — `http://172.17.0.1:8098`, префикс `/webdav`.
- Basic-auth — креды портала OnlyOffice (те же, что у `oo`).
- Корень — `Dokumente der Projekte/…` (папка `Fibu EDL` внутри).

## Конфиг

- Образ — `rclone/rclone`.
- Remote — on-the-fly `:webdav:` плюс флаги `--webdav-url`, `--webdav-user`,
  `--webdav-pass`.
- На старте контейнер пишет `/config/rclone/rclone.conf` (`[webdav]`), чтобы
  `rclone ls webdav:` внутри контейнера работал без флагов.
- `ONLYOFFICE_PASSWORD` — plain; rclone сам зовёт `rclone obscure`.
- Флаги mount: `--vfs-cache-mode writes`, `--cache-dir /cache`,
  `--dir-cache-time 1m`, `--allow-other`, `--allow-non-empty`.
- FUSE: `cap_add: [SYS_ADMIN]`, `devices: [/dev/fuse]`,
  `security_opt: apparmor:unconfined`.
- Тома: `onlyoffice-mnt` → `/mnt/onlyoffice`, `rclone-cache` → `/cache`.

### env (только имена)

| Переменная | Значение |
|------------|----------|
| `ONLYOFFICE_USER` | пользователь портала |
| `ONLYOFFICE_PASSWORD` | пароль портала (алиас `ONLYOFFICE_PASS`) |
| `ONLYOFFICE_WEBDAV_URL` | по умолчанию `http://172.17.0.1:8098/webdav` |

## Команды

```bash
# креды (или .env рядом с compose)
set -a; . .secrets/oo.env; set +a

docker compose -f deploy/docker-compose.rclone-webdav.yml up -d
docker compose -f deploy/docker-compose.rclone-webdav.yml ps
docker compose -f deploy/docker-compose.rclone-webdav.yml down

# дерево
docker exec rclone-webdav rclone ls webdav:
docker exec rclone-webdav rclone lsf webdav:

# содержимое точки монтирования
docker exec rclone-webdav ls /mnt/onlyoffice

# чтение/запись как обычная ФС
docker exec rclone-webdav cat "/mnt/onlyoffice/Meine Dokumente/x.txt"
```

## Smoke (проверено 2026-09-16)

```text
$ docker exec rclone-webdav rclone ls webdav:
  5357109 Meine Dokumente/ONLYOFFICE-Audiobeispiel.mp3
    44805 Meine Dokumente/ONLYOFFICE-Beispiel-Tabellenblatt.xlsx
    58988 Meine Dokumente/ONLYOFFICE-Beispieldokument.docx
  ...

$ TS=20260916-212459
$ docker exec rclone-webdav sh -c "printf 'rclone-webdav round-trip $TS\n' \
    > '/mnt/onlyoffice/Meine Dokumente/rclone-smoke-$TS/hello.txt'"
# ждём появления на сервере (host HTTP, мимо монтирования):
GET /webdav/Meine%20Dokumente/rclone-smoke-$TS/hello.txt -> 200 (через 7s)
server bytes: rclone-webdav round-trip 20260916-212459
# читаем обратно через монтирование:
mount read: rclone-webdav round-trip 20260916-212459
MATCH: yes

$ docker exec rclone-webdav rm -f \
    "/mnt/onlyoffice/Meine Dokumente/rclone-smoke-$TS/hello.txt"
GET /webdav/Meine%20Dokumente/rclone-smoke-$TS/hello.txt -> 404 (через 1s)

# пустой каталог: rmdir через монтирование на сервер не доходит,
# удаляем напрямую:
$ docker exec rclone-webdav rclone rmdir "webdav:Meine Dokumente/rclone-smoke-$TS"
PROPFIND rclone-smoke-20260916-212459/ -> 404
no rclone-smoke leftovers
```

Вывод: запись → чтение → удаление файла подтверждены. Файл удалён.

## Ограничения

- Нужен FUSE: `SYS_ADMIN` + `/dev/fuse`. На хосте без FUSE не поедет.
- Сайдкар слушает docker-bridge `172.17.0.1:8098`, публично не выставлен.
  Только localhost/локальные контейнеры.
- Запись идёт через VFS write-back (по умолчанию ~5s). Перед проверкой
  «файл на сервере» опрашивать сервер, не доверять сразу после `write`.
- `rmdir` через монтирование НЕ доходит до `oo-webdav` (пустой каталог
  остаётся на сервере). Удалять каталоги напрямую:
  `rclone rmdir webdav:<path>` или `rclone purge webdav:<path>`.
- VFS-кэш растёт в томе `rclone-cache`; ограничить `--vfs-cache-max-size`.
- Блокировок между редактором OnlyOffice и монтированием нет. Не редактировать
  один и тот же файл одновременно.
- Данные не шифруются на диске хоста в `rclone-cache` (том Docker).
