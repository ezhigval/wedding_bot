# Wedding Bot

`Wedding Bot` — единый Go-сервис для Telegram-бота и Telegram Mini App. Приложение обслуживает приглашение на свадьбу, RSVP, игровые активности, тайминг, рассадку, фото/видео и административные сценарии, используя Google Sheets как основной операционный storage.

## Возможности

- Telegram-бот с пользовательскими и админскими сценариями.
- Telegram Mini App с вкладками приглашения, RSVP, фото, играми, таймингом и рассадкой.
- Регистрация гостей с поддержкой дополнительных гостей.
- Регистрация сохраняет в `Список гостей` canonical `user_id`, а `username` использует только как compatibility fallback, пока numeric ID недоступен.
- Проверка регистрации по `user_id`, `initData`, `username` и подписанной cookie-сессии.
- Загрузка фото и видео в Google Drive с записью метаданных в Google Sheets.
- Игры `Wordle`, `Crossword`, `Flappy Bird`, `Dragon` с рейтингом и прогрессом.
- Публикация общей и персональной рассадки.
- Двусторонняя админская синхронизация `Список гостей` <-> `Рассадка` с пакетным чтением Google Sheets и защитой от лишних read-запросов.
- Ежедневный scheduler для reset игровых сценариев.

## Стек

### Backend

- Go `1.24`
- `gorilla/mux`
- `go-telegram-bot-api`
- Google Sheets API / Google Drive API
- `zerolog`
- `tollbooth`
- `unrolled/secure`
- SQLite (`modernc.org/sqlite`) для локального кэша

### Frontend

- React `18`
- TypeScript
- Vite
- Tailwind CSS
- Framer Motion
- `@tanstack/react-query`

## Структура репозитория

```text
.
├── cmd/
│   ├── server               # основной entrypoint сервиса
│   └── google_drive_oauth   # helper для получения Google Drive refresh token
├── internal/
│   ├── api                  # HTTP API, auth/initData, middleware
│   ├── bot                  # Telegram bot и admin flows
│   ├── cache                # memory/SQLite cache
│   ├── config               # загрузка env и runtime config
│   ├── daily_reset          # scheduler ежедневного reset
│   ├── google_sheets        # доменная логика и доступ к Google Sheets/Drive
│   └── keyboards            # Telegram keyboards
├── docs                     # актуальная эксплуатационная документация
├── res                      # runtime ресурсы
├── webapp                   # собранный frontend bundle, который раздаёт Go-сервер
├── webapp-react             # исходники frontend
├── Dockerfile
└── Makefile
```

## Требования

- Go `1.24+`
- Node.js `20.19+` или `22+`
- npm `10+`
- Доступ сервисного аккаунта Google к рабочей таблице
- Включённые Google Sheets API и Google Drive API
- Telegram bot token

## Быстрый старт

### 1. Подготовить env

```bash
cp .env.example .env.local
```

Заполните `.env.local` рабочими значениями. Файл `.env.local` специально не должен попадать в Git.

### 2. Установить зависимости

```bash
make deps
```

Альтернатива без `make`:

```bash
go mod download
cd webapp-react && npm ci
```

### 3. Собрать и проверить проект

```bash
make verify
```

Команда выполнит:

- `go vet ./...`
- `go test ./...`
- `go build ./cmd/server`
- `cd webapp-react && npm run lint`
- `cd webapp-react && npm run build`

### 4. Запустить сервис

```bash
make run
```

По умолчанию сервис стартует на `http://localhost:10000`.

## Переменные окружения

Полный шаблон находится в [.env.example](.env.example).

Ключевые переменные:

- `BOT_TOKEN` — Telegram bot token.
- `WEBAPP_URL` — публичный URL Mini App.
- `GOOGLE_SHEETS_ID` — основная Google Sheets таблица.
- `GOOGLE_SHEETS_CREDENTIALS` или `GOOGLE_SHEETS_CREDENTIALS_BASE64` — credentials сервисного аккаунта.
- `GOOGLE_DRIVE_FOLDER_ID` — папка Google Drive для фото и видео.
- `GOOGLE_DRIVE_OAUTH_CLIENT_ID`, `GOOGLE_DRIVE_OAUTH_CLIENT_SECRET`, `GOOGLE_DRIVE_OAUTH_REFRESH_TOKEN` — опциональный OAuth контур для загрузки в личный Google Drive.
- `GROUP_ID`, `GROUP_LINK` — настройки общего чата гостей.
- `SEATING_API_TOKEN` — токен для защищённых операций с рассадкой.
- `DEBUG=false` — production-default, включает verbose logging только в debug.

## Получение Google Drive refresh token

Если нужен OAuth-контур для загрузки в личный Google Drive:

```bash
go run ./cmd/google_drive_oauth
```

Helper поднимет локальный callback server, выведет auth URL и напечатает готовый `GOOGLE_DRIVE_OAUTH_REFRESH_TOKEN`.

## Docker

```bash
docker build -t wedding-bot .
docker run --env-file .env.local -p 10000:10000 wedding-bot
```

Контейнер запускает только runtime-необходимые артефакты: бинарник, `webapp/`, `res/` и каталог `data/`.

## Проверка качества

Локально подтверждены:

- `go test ./cmd/... ./internal/...`
- `go test -race ./cmd/... ./internal/...`
- `go vet ./cmd/... ./internal/...`
- `go build ./cmd/server`
- `cd webapp-react && npm run lint`
- `cd webapp-react && npm run build`
- `cd webapp-react && npm audit --omit=dev`

## Документация

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [docs/ROADMAP.md](docs/ROADMAP.md)
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)
- [docs/DEPLOY_CHECKLIST.md](docs/DEPLOY_CHECKLIST.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [docs/RULES.md](docs/RULES.md)
