---
type: reference
status: current
related:
  - README.md
  - ../AGENTS.md
---

# Rate limit, backoff и cooldown

Устойчивость к 429 (openresty). Всё встроено в библиотеку — отдельный пакет не
нужен. Реализация: `ratelimit.go`, `retry.go`.

## Что происходит с каждым запросом

1. **Cooldown-гейт** — общий на процесс. Если недавно пришёл 429, все запросы
   ждут конца окна.
2. **Rate limiter** — token bucket на процесс. Пейсит все HTTP-пути: листинг,
   создание папок, загрузку, `get project`, auth.
3. Запрос уходит.
4. Ответ ≥400 → `*TransientError` (для 429/502/503/504) с `Retry-After`.
5. `DoRetry` — экспоненциальный backoff, без jitter.
6. `Retry-After` длиннее backoff → ждём его; окно уходит в общий cooldown.

Установлено в `NewClient` через `pacedTransport`; отдельный код трогать не надо.

## Env

| Переменная | Default | Смысл |
|---|---|---|
| `OO_RATE_LIMIT` | `4` | запросов/с на процесс; `0` — лимитер выключен |
| `OO_BURST` | `1` | запас токенов token bucket |
| `OO_RETRY_ATTEMPTS` | `7` | всего попыток, включая первую |
| `OO_RETRY_BASE` | `2s` | база экспоненты: ждать перед попыткой N = `Base*2^(N-1)` |
| `OO_RETRY_MAX` | `2m` | потолок ожидания |

Битые значения → default. `OO_RETRY_*` — формат `time.ParseDuration`
(`2s`, `30s`, `2m`).

## Правила

- Детерминированно, без jitter — повторный прогон ждёт столько же.
- `Retry-After` — секунды (`120`) или HTTP-date.
- Cooldown общий: параллельные и последовательные вызовы не бьют в стену.
- Backoff cap не ограничивает `Retry-After` — серверу верим больше.
- Только stdlib.

## Когда руками снять нагрузку

`OO_RATE_LIMIT` ниже (`2`), `OO_BURST=1`; при массовом apply — батчами.

## Тесты

`retry_test.go` — `Retry-After`, экспонента, cap; `ratelimit_test.go` — burst,
`OO_RATE_LIMIT=0`, cooldown. Фейковый сервер отдаёт 429 с заголовком.
