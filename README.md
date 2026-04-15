# Wedding Bot

Telegram-бот и Telegram Mini App для свадебного приглашения, RSVP, игр, фото, тайминга, рассадки и административного управления гостями.

Проект состоит из одного Go-сервиса, который:

- поднимает HTTP API;
- обслуживает статический фронтенд Mini App;
- запускает long polling Telegram-бота;
- работает с Google Sheets как с основной бизнес-базой данных;
- использует локальный кэш для снижения нагрузки на внешние источники.

## Что уже умеет проект

- Регистрация гостей через Mini App и через Telegram-бота.
- Добавление дополнительных гостей в рамках одной регистрации.
- Проверка регистрации по `user_id`, `initData`, `username` и сессионной cookie.
- Отмена регистрации.
- Публичная часть приглашения: главная, тайминг, дресс-код, меню, пожелания, фото, рассадка.
- Игры для гостей: `Dragon`, `Flappy Bird`, `Crossword`, `Wordle`.
- Рейтинг, очки и звания игроков.
- Сохранение игрового прогресса в Google Sheets.
- Загрузка фото и видео из Mini App и из бота.
- Админ-меню в Telegram: гости, рассадка, игры, группа, рассылки.
- Отправка рассылок в личные сообщения и в группу.
- Проверка участия пользователя в общем чате гостей.
- Ежедневный планировщик сброса игровых активностей.
- Базовый GitHub Actions CI для backend и frontend.

## Технологический стек

### Backend

- Go `1.24`
- `gorilla/mux`
- `go-telegram-bot-api`
- Google Sheets API
- SQLite (`modernc.org/sqlite`) для части кэша
- `zerolog`
- `tollbooth`
- `unrolled/secure`

### Frontend

- React `18`
- TypeScript
- Vite
- Tailwind CSS
- Framer Motion
- Telegram WebApp SDK

## Структура репозитория

```text
.
├── cmd/server                # Точка входа Go-сервиса
├── internal/api              # HTTP API, middleware, Telegram initData/auth
├── internal/bot              # Telegram-бот и админские сценарии
├── internal/cache            # In-memory и SQLite-кэш
├── internal/config           # Конфигурация из env
├── internal/daily_reset      # Планировщик и ежедневный сброс игровых активностей
├── internal/google_sheets    # Основная бизнес-логика и доступ к данным
├── internal/keyboards        # Reply/inline keyboards для бота
├── webapp-react              # Исходники React Mini App
├── webapp                    # Собранный фронтенд, который раздаёт Go-сервер
├── res                       # Медиа-ресурсы
└── docs                      # Актуальная документация по проекту
```

## Документация

- [Архитектура](docs/ARCHITECTURE.md)
- [Чек-лист деплоя](docs/DEPLOY_CHECKLIST.md)
- [Правила проекта](docs/RULES.md)
- [Роадмап](docs/ROADMAP.md)

## Архитектура в одном абзаце

Telegram-бот и Mini App используют общий Go-backend. Backend читает и пишет бизнес-данные в Google Sheets, обслуживает фронтенд из папки `webapp`, хранит часть промежуточных данных в памяти и SQLite, а также синхронизирует игровые механики и административные действия. Google Sheets в этом проекте выполняет роль операционной БД и панели контента одновременно.

## Основные доменные сущности

- Гость: имя, фамилия, сторона, категория, подтверждение участия, идентификатор Telegram.
- Дополнительный гость: отдельная строка в `Список гостей`, привязанная к владельцу основной регистрации.
- Приглашение: имя, Telegram username, `user_id`, статус отправки.
- Игровой профиль: очки по играм, общий счёт, звание, дата обновления.
- Wordle/Crossword прогресс: состояние по каждому игроку.
- Тайминг: список публичных событий свадьбы.
- Рассадка: опубликованный список столов из `Рассадка_фикс` и персональный поиск стола гостя.
- Фото и видео: метаданные загрузки и ссылка на файл в Google Drive.

## Переменные окружения

Смотри полный шаблон в [.env.example](.env.example).

Ключевые переменные:

- `BOT_TOKEN` — токен Telegram-бота.
- `GOOGLE_SHEETS_ID` — ID основной таблицы.
- `GOOGLE_SHEETS_CREDENTIALS` или `GOOGLE_SHEETS_CREDENTIALS_BASE64` — credentials сервисного аккаунта для Google Sheets.
- `GOOGLE_DRIVE_FOLDER_ID` — ID папки Google Drive или полная ссылка на папку, куда складываются все загруженные фото.
- `GOOGLE_DRIVE_OAUTH_CLIENT_ID`, `GOOGLE_DRIVE_OAUTH_CLIENT_SECRET`, `GOOGLE_DRIVE_OAUTH_REFRESH_TOKEN` — OAuth-параметры обычного Google-аккаунта для загрузки файлов в личный Drive.
- `WEBAPP_URL` — публичный URL Mini App.
- `WEBAPP_PATH` — путь до собранного фронтенда, по умолчанию `webapp`.
- `GROUP_ID` и `GROUP_LINK` — общий чат гостей.
- `WEDDING_DATE`, `GROOM_NAME`, `BRIDE_NAME`, `WEDDING_ADDRESS` — публичные данные мероприятия.
- `DEBUG` — дев-режим.

## Минимальные требования

- Go `1.24+`
- Node.js `18+`
- Доступ сервисного аккаунта Google к таблице
- Включённый Google Drive API
- Для загрузки в личный Google Drive: OAuth client id/secret и refresh token пользователя
- Telegram bot token

## Локальный запуск

### 1. Подготовить окружение

```bash
cp .env.example .env.local
```

Заполнить переменные в `.env.local`.

### 2. Установить зависимости

```bash
go mod download
cd webapp-react
npm install
cd ..
```

### 3. Собрать фронтенд

```bash
cd webapp-react
npm run build
cd ..
```

Сборка попадёт в папку `webapp/`, которую раздаёт Go-сервер.

### 4. Запустить сервис

```bash
go run ./cmd/server
```

По умолчанию используется порт `10000`, если `PORT` не задан.

## Docker

```bash
docker build -t wedding-bot .
docker run --env-file .env.local -p 10000:10000 wedding-bot
```

## Как устроено хранение данных

Основной источник правды — Google Sheets. В проекте используются вкладки:

- `Список гостей`
- `Пригласительные`
- `Админ бота`
- `Публичная План-сетка`
- `Рассадка`
- `Рассадка_фикс`
- `Игры`
- `Wordle`
- `Wordle_Прогресс`
- `Wordle_Состояние`
- `Кроссворд`
- `Кроссворд_Прогресс`
- `Фото`
- `Config`

Часть этих листов создаётся автоматически через `EnsureRequiredSheets`.

## API высокого уровня

Публичные и клиентские endpoint'ы:

- `GET /health`
- `GET /api/config`
- `POST /api/parse-init-data`
- `POST /api/check-registration`
- `POST /api/register`
- `POST /api/cancel-registration`
- `GET /api/guests`
- `GET /api/stats`
- `GET /api/timeline`
- `POST /api/upload-photo` — загрузка фото и видео из Mini App
- `GET /api/game-stats`
- `POST /api/update-game-score`
- `GET /api/wordle/*`
- `POST /api/wordle/*`
- `GET /api/crossword/*`
- `POST /api/crossword/*`
- `GET /api/seating/info`
- `GET /api/seating/personal`

Контракт рассадки для Mini App:

- `/api/seating/info` возвращает `{ visible, published_at, tables }`, где `tables` — массив объектов `{ table, guests }`.
- `/api/seating/personal` возвращает `{ visible, published_at, table, neighbors, full_name }` для персонального места гостя.

Подробности и потоки есть в [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Текущее состояние проекта

Проект уже покрывает основной пользовательский сценарий свадебного приглашения и игровой активности, но остаётся зоной активной разработки. Главные технические фокусы сейчас:

- стабилизация identity/auth между Telegram и браузерным режимом;
- завершение ежедневного сброса игровых сценариев;
- повышение предсказуемости Google Sheets как хранилища;
- рост покрытия тестами;
- формализация правил эксплуатации и разработки.

Подробный план зафиксирован в [docs/ROADMAP.md](docs/ROADMAP.md).
