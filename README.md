# URL Shortener

Сервис для сокращения ссылок на Go: PostgreSQL, pgx, goose-миграции, HTTP-API и простой веб-интерфейс.

---

## Содержание

- [Возможности](#возможности)
- [Требования](#требования)
- [Переменные окружения](#переменные-окружения)
- [Запуск](#запуск)
  - [Вариант 1. Всё через Docker Compose](#вариант-1-всё-через-docker-compose)
  - [Вариант 2. Go локально, PostgreSQL в Docker](#вариант-2-go-локально-postgresql-в-docker)
  - [Вариант 3. Всё локально, без Docker](#вариант-3-всё-локально-без-docker)
- [API](#api)
- [Быстрая проверка через cURL](#быстрая-проверка-через-curl)
- [Структура проекта](#структура-проекта)
- [Частые проблемы](#частые-проблемы)

---

## Возможности

- Создание короткой ссылки с опциональным TTL и лимитом кликов.
- Переход по короткой ссылке с инкрементом счётчика кликов.
- Поиск и удаление ссылки по оригинальному URL.
- Автоматическая очистка истёкших и исчерпанных ссылок по таймеру.
- HTML-страница ошибок для переходов по «мёртвым» коротким ссылкам.
- Веб-интерфейс для ручного управления ссылками.

---

## Требования

- **Go** 1.27
- **Docker** и **Docker Compose** 
- Свободные порты: `8080` (приложение), `5432` (PostgreSQL)

---

## Переменные окружения

Создайте файл `.env` в корне проекта:

```dotenv
APP_PORT=8080

SHORT_LENGTH=8
CLEANUP_SECONDS=5

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=shortener
DB_SSLMODE=disable
```

| Переменная        | Назначение                                                                 |
|-------------------|----------------------------------------------------------------------------|
| `APP_PORT`        | Порт HTTP-сервера                                                          |
| `SHORT_LENGTH`    | Длина генерируемого короткого кода                                         |
| `CLEANUP_SECONDS` | Интервал фоновой очистки истёкших ссылок (обязательно > 0)                 |
| `DB_HOST`         | Хост PostgreSQL                                                            |
| `DB_PORT`         | Порт PostgreSQL                                                            |
| `DB_USER`         | Пользователь БД                                                            |
| `DB_PASSWORD`     | Пароль пользователя БД                                                     |
| `DB_NAME`         | Имя базы данных                                                            |
| `DB_SSLMODE`      | Режим SSL (`disable` для локальной разработки)                             |

> ⚠️ **`DB_HOST` зависит от способа запуска.**
> - Go локально + Postgres в Docker с проброшенным портом → `DB_HOST=localhost`.
> - Всё через docker-compose → `DB_HOST=postgres` (имя сервиса в compose-сети).

> ⚠️ `CLEANUP_SECONDS` обязателен: при `0` или отрицательном значении `time.NewTicker` паникует.

---

## Запуск

### Вариант 1. Всё через Docker Compose

Самый простой способ — нужен только Docker.

**1. Подготовить `.env`** для этого варианта:

```dotenv
DB_HOST=postgres
DB_PASSWORD=postgres
```

**2. Собрать и запустить:**

```bash
docker compose up --build
```

Compose поднимет три сервиса:

1. `postgres` — БД с проверкой здоровья (`pg_isready`).
2. `migrate` — прогонит миграции из `./migrations` через goose и завершится.
3. `server` — само приложение.

`migrate` и `server` ждут `service_healthy` у Postgres, поэтому гонок «БД ещё не готова» не будет.

**3. Открыть приложение:**

```
http://localhost:8080
```

**4. Остановить:**

```bash
docker compose down       # остановить, данные сохранить
docker compose down -v    # остановить и удалить volume с БД
```

### Вариант 2. Go локально, PostgreSQL в Docker

Удобно для разработки — можно пересобирать бинарник без пересборки образа.

**1. Поднять только Postgres:**

```bash
docker compose up -d postgres
```

Проверить, что контейнер здоров:

```bash
docker compose ps
```

**2. `.env` — с `DB_HOST=localhost`:**

```dotenv
DB_HOST=localhost
DB_PASSWORD=postgres
```

**3. Применить миграции:**

```bash
go run ./cmd/migrate
```

Ожидаемый вывод:

```
... OK   00001_urls.sql
... OK   00002_add_link_limits.sql
migrations completed
```

**4. Запустить сервер:**

```bash
go run ./cmd/server
```

Ожидаемый вывод:

```
server started on 8080
```

**5. Открыть приложение:**

```
http://localhost:8080
```

> **Запускать нужно из корня проекта.** Иначе `template.ParseFiles("web/errors.html")` и `http.FileServer(http.Dir("./web"))` не найдут файлы.

### Вариант 3. Всё локально, без Docker

Понадобится собственный PostgreSQL 14+ на `localhost:5432` с базой `shortener`. Далее те же шаги, что в варианте 2:

```bash
export $(cat .env | xargs)   # или использовать direnv / dotenv
go run ./cmd/migrate
go run ./cmd/server
```

---

## API

| Метод    | Путь                     | Описание                                            |
|----------|--------------------------|-----------------------------------------------------|
| `POST`   | `/shorten`               | Создать короткую ссылку                             |
| `GET`    | `/shorten?url=...`       | Получить информацию о ссылке по оригинальному URL   |
| `DELETE` | `/shorten?url=...`       | Удалить ссылку по оригинальному URL                 |
| `GET`    | `/{shortenedURL}`        | Перейти по короткой ссылке                          |
| `GET`    | `/`                      | Веб-интерфейс                                       |

**Тело запроса `POST /shorten`:**

```json
{
  "url": "https://example.com",
  "expires_in": 60,
  "max_clicks": 3
}
```

- `expires_in` — TTL в секундах (опционально, > 0).
- `max_clicks` — максимум переходов (опционально, > 0).

**Ответ:**

```json
{
  "url": "https://example.com",
  "shortened_url": "aB3xY9Qk",
  "expires_at": "2026-10-02T13:00:00Z",
  "max_clicks": 3,
  "clicks": 0
}
```

Коды ответов `POST /shorten`:

- `201 Created` — ссылка создана (или возвращена уже существующая живая).
- `400 Bad Request` — невалидный URL, TTL или max_clicks.
- `500 Internal Server Error` — внутренняя ошибка.

При переходе по короткой ссылке:

- `302 Found` — редирект на оригинальный URL.
- `410 Gone` — ссылка истекла или исчерпан лимит кликов (HTML-страница).
- `404 Not Found` — короткий код не найден.

---

## Быстрая проверка через cURL

Создать короткую ссылку:

```bash
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","expires_in":60,"max_clicks":3}'
```

Найти по оригинальному URL:

```bash
curl "http://localhost:8080/shorten?url=https://example.com"
```

Перейти по короткой ссылке (в браузере):

```
http://localhost:8080/aB3xY9Qk
```

Удалить:

```bash
curl -X DELETE "http://localhost:8080/shorten?url=https://example.com"
```

---

## Структура проекта

```
.
├── cmd/
│   ├── migrate/        # бинарник миграций (goose)
│   └── server/         # бинарник HTTP-сервера
├── internal/
│   ├── app/            # сборка приложения, middleware
│   ├── config/         # загрузка .env
│   ├── database/       # репозиторий поверх pgxpool
│   ├── domain/         # сущности и интерфейсы
│   ├── service/        # бизнес-логика
│   └── transport/http/ # HTTP-хендлеры, DTO, шаблоны
├── migrations/         # SQL-миграции goose
├── web/                # статика (index.html, errors.html)
├── Dockerfile
└── docker-compose.yml
```

---

## Частые проблемы

### `panic: open web/errors.html: no such file or directory`

Сервер запускается не из корня проекта. Запускайте `go run ./cmd/server` из корня, либо перейдите в директорию, где лежит `web/`.

### `Error loading config: CLEANUP_SECONDS must be a positive integer`

В `.env` не задан `CLEANUP_SECONDS` или он не число. Добавьте, например, `CLEANUP_SECONDS=5`.

### `Error during connection to database: ... connection refused`

- Postgres не поднят → проверьте `docker compose ps`.
- В `.env` неверный `DB_HOST` для выбранного способа запуска (`localhost` vs `postgres`).
- Не совпадает пароль: в `docker-compose.yml` у сервиса `postgres` стоит `POSTGRES_PASSWORD: postgres`, и `DB_PASSWORD` в `.env` должен совпадать.

### `Invalid operation: s.Cleanup * time.Second`

Нужно явное приведение типа:

```go
ticker := time.NewTicker(time.Duration(s.Cleanup) * time.Second)
```

### Порт 8080 занят

Поменяйте `APP_PORT` в `.env`. В `docker-compose.yml` проброс идёт через `${APP_PORT}:${APP_PORT}`, поэтому внутренний и внешний порты совпадают автоматически.

### Ссылка «истекает мгновенно»

Проверьте, что колонка `expires_at` в БД имеет тип `TIMESTAMPTZ` (см. миграцию `00002_add_link_limits.sql`). Если ранее применялась схема с `TIMESTAMP`, откатите и примените миграции заново:

```bash
docker compose down -v
docker compose up -d postgres
go run ./cmd/migrate
go run ./cmd/server
```

### Ссылка «навсегда остаётся в БД»

Убедитесь, что `CLEANUP_SECONDS` > 0 и сервис очистки действительно запущен (в логах при удалении появляется строка `cleanup: removed N inactive links`). Также при попытке создать ссылку с уже мёртвым URL сервис удаляет старую запись и создаёт новую.
