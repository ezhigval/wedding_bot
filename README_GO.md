# Wedding Bot - Go версия

Это рефакторинг свадебного бота с Python на Go.

## Структура проекта

```
wedding-bot/
├── cmd/
│   └── server/
│       └── main.go          # Главный файл сервера
├── internal/
│   ├── api/                 # API endpoints
│   ├── bot/                 # Telegram бот
│   ├── cache/               # Кэш статистики игр
│   ├── config/               # Конфигурация
│   ├── daily_reset/         # Ежедневный сброс игр
│   ├── google_sheets/       # Работа с Google Sheets
│   └── keyboards/           # Клавиатуры для бота
├── webapp/                  # Статические файлы Mini App
├── webapp-react/            # React приложение
├── go.mod                   # Go модули
└── Dockerfile               # Docker образ для деплоя
```

## Переменные окружения

Те же самые переменные окружения, что и в Python версии:

- `BOT_TOKEN` - токен Telegram бота
- `WEDDING_DATE` - дата свадьбы (YYYY-MM-DD)
- `GROOM_NAME` - имя жениха
- `BRIDE_NAME` - имя невесты
- `WEDDING_ADDRESS` - адрес свадьбы
- `WEBAPP_URL` - URL Mini App
- `ADMIN_USER_ID` - ID администратора
- `GOOGLE_SHEETS_ID` - ID Google таблицы
- `GOOGLE_SHEETS_CREDENTIALS` - JSON credentials для Google Sheets API
- И другие (см. `internal/config/config.go`)

## Деплой на Render.com

1. Создайте новый Web Service на Render.com
2. Подключите репозиторий GitHub
3. Выберите ветку `go-dev`
4. Настройки:
   - **Build Command**: `go build -o server ./cmd/server`
   - **Start Command**: `./server`
   - **Environment**: Go
   - **Port**: 10000 (или переменная `PORT`)

5. Добавьте все переменные окружения из `.env` файла

6. Деплой должен пройти успешно

## Локальный запуск

```bash
# Установите зависимости
go mod download

# Запустите сервер
go run ./cmd/server
```

Или соберите бинарный файл:

```bash
go build -o server ./cmd/server
./server
```

## Отличия от Python версии

- Все модули переписаны на Go
- Используется стандартная библиотека Go и goroutines вместо asyncio
- SQLite кэш использует `modernc.org/sqlite` (pure Go)
- HTTP сервер на `gorilla/mux` вместо `aiohttp`
- Telegram бот на `gopkg.in/telebot.v3` вместо `aiogram`

## Статус миграции

✅ Все основные модули переписаны:
- ✅ config.go
- ✅ google_sheets.go (все функции)
- ✅ api.go (все endpoints)
- ✅ bot.go (базовая структура)
- ✅ cache.go
- ✅ daily_reset.go
- ✅ keyboards.go
- ✅ main.go

⚠️ Требуется тестирование и доработка:
- Некоторые функции бота требуют дополнительной реализации
- Некоторые admin функции требуют доработки
- Нужно протестировать все endpoints

## Примечания

- Фронтенд (React) остается без изменений
- Все переменные окружения остаются теми же
- Google Sheets API работает так же
- Структура данных в Google Sheets не изменилась

