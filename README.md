# Wedding Bot - Telegram Mini App

Свадебный Telegram бот с веб-приложением для управления гостями, играми и рассадкой.

## 🚀 Технологии

### Backend
- **Go (Golang)** 1.24.0
- **Telegram Bot API** (`github.com/go-telegram-bot-api/telegram-bot-api/v5`)
- **Google Sheets API** - хранение данных
- **SQLite** - кэширование игровой статистики
- **Gorilla Mux** - HTTP роутинг
- **Zerolog** - структурированное логирование
- **Rate Limiting** - защита от перегрузки
- **Security Headers** - улучшенная безопасность

### Frontend
- **React.js** 18.2.0
- **TypeScript**
- **Vite** - сборка
- **Tailwind CSS** - стилизация
- **Framer Motion** - анимации
- **React Query** - кэширование запросов
- **Lottie** - анимации

## 📁 Архитектура проекта

```
wedding-bot/
├── cmd/
│   └── server/          # Точка входа сервера
├── internal/
│   ├── api/             # HTTP API handlers
│   ├── bot/             # Telegram bot handlers
│   ├── cache/           # Кэширование (SQLite + in-memory)
│   ├── config/          # Конфигурация
│   ├── daily_reset/     # Ежедневный сброс игр
│   ├── google_sheets/   # Работа с Google Sheets
│   └── keyboards/       # Клавиатуры бота
├── webapp-react/        # React приложение
├── res/                 # Ресурсы (фото, анимации)
├── Dockerfile           # Docker конфигурация
└── go.mod              # Go зависимости
```

## 🛠 Установка и запуск

### Требования
- Go 1.24.0+
- Node.js 18+
- Google Sheets API credentials
- Telegram Bot Token

### 1. Клонирование репозитория
```bash
git clone <repository-url>
cd wedding-bot
```

### 2. Настройка переменных окружения

Создайте файл `.env` в корне проекта:

```env
# Telegram
BOT_TOKEN=your_telegram_bot_token
GROUP_ID=your_group_id
GROUP_LINK=https://t.me/your_group

# Google Sheets
GOOGLE_SHEETS_ID=your_sheet_id
GOOGLE_CREDENTIALS_PATH=path/to/credentials.json

# Server
PORT=8080
DEBUG=false

# Web App
WEBAPP_URL=https://your-domain.com
WEBAPP_PATH=webapp-react/dist
WEBAPP_PHOTO_PATH=res/welcome_photo.jpeg

# Wedding Info
GROOM_NAME=Имя жениха
BRIDE_NAME=Имя невесты
WEDDING_DATE=2026-06-05
WEDDING_ADDRESS=Адрес свадьбы
GROOM_TELEGRAM=@username
BRIDE_TELEGRAM=@username
```

### 3. Установка зависимостей

**Backend:**
```bash
go mod download
```

**Frontend:**
```bash
cd webapp-react
npm install
```

### 4. Сборка

**Backend:**
```bash
go build ./cmd/server
```

**Frontend:**
```bash
cd webapp-react
npm run build
```

### 5. Запуск

**Локально:**
```bash
./server
```

**Docker:**
```bash
docker build -t wedding-bot .
docker run -p 8080:8080 --env-file .env wedding-bot
```

## 📦 Деплой на Render.com

1. Подключите GitHub репозиторий к Render
2. Выберите "Web Service"
3. Настройки:
   - **Build Command:** `go build ./cmd/server`
   - **Start Command:** `./server`
   - **Environment:** Docker
4. Добавьте все переменные окружения из `.env`
5. Деплой запустится автоматически

## 🎮 Функционал

### Для гостей
- Регистрация через Telegram Mini App
- Просмотр информации о свадьбе
- Игры: Дракончик, Flappy Bird, Кроссворд, Wordle
- Система рейтинга и званий
- Загрузка фотографий
- Просмотр рассадки и timeline

### Для администраторов
- Управление гостями
- Рассылка сообщений
- Управление играми (Wordle, Кроссворд)
- Просмотр статистики
- Управление группой

## 🔒 Безопасность

- Rate limiting (100 запросов/минуту)
- Security headers (XSS, clickjacking защита)
- CORS настройки
- Валидация Telegram WebApp данных
- Структурированное логирование

## 📊 Производительность

- In-memory кэширование для частых запросов
- SQLite кэш для игровой статистики
- React Query для кэширования на фронтенде
- Оптимизированные запросы к Google Sheets

## 📝 API Endpoints

- `GET /api/config` - конфигурация
- `POST /api/register` - регистрация гостя
- `GET /api/guests` - список гостей
- `GET /api/game-stats` - статистика игр
- `POST /api/update-game-score` - обновление счета
- `GET /api/wordle/word` - текущее слово Wordle
- `POST /api/wordle/guess` - угадывание слова
- `GET /api/crossword/data` - данные кроссворда
- `GET /health` - health check

## 👤 Контакты

- **Telegram:** [@ezhigval](https://t.me/ezhigval)
- **Email:** smailikin70@yandex.ru

## 📄 Лицензия

Проект создан для личного использования.
