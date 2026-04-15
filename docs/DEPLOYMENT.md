# Deployment

## 1. Что нужно до деплоя

- доступ к production `.env.local` или набору env-переменных;
- рабочий `BOT_TOKEN`;
- доступ сервисного аккаунта к Google Sheets;
- включённые Google Sheets API и Google Drive API;
- заполненный `WEBAPP_URL`;
- доступ к production-хосту или контейнерной среде.

## 2. Локальная верификация перед выкладкой

```bash
make verify
cd webapp-react && npm audit --omit=dev
```

Если Docker используется как основной способ деплоя:

```bash
docker build -t wedding-bot .
```

## 3. Деплой без Docker

### На машине с исходниками

```bash
cp .env.example .env.local
# заполнить .env.local production-значениями
make verify
go run ./cmd/server
```

### Через собранный бинарник

```bash
make frontend-build
go build -o server ./cmd/server
./server
```

Нужно запускать из корня репозитория или передать корректные `WEBAPP_PATH`, `PHOTO_PATH`, `WEBAPP_PHOTO_PATH`, `DB_PATH`.

## 4. Деплой через Docker

```bash
docker build -t wedding-bot .
docker run --env-file .env.local -p 10000:10000 wedding-bot
```

Что попадает в runtime image:

- `server`
- `webapp/`
- `res/`
- `/app/data`

## 5. Smoke checks после старта

Проверить:

```bash
curl -fsS http://localhost:10000/health
curl -fsS http://localhost:10000/api/config
```

И вручную:

- Mini App открывается без белого экрана;
- бот отвечает на `/start`;
- регистрация гостя проходит;
- фото/видео реально попадают в Google Drive;
- тайминг и рассадка загружаются;
- игры не падают на первом запросе.

## 6. Что смотреть в логах

- ошибки загрузки конфигурации;
- ошибки инициализации Google Sheets;
- ошибки инициализации Google Drive;
- статус запуска Telegram-бота;
- запуск `daily_reset`;
- HTTP ошибки `5xx`.

## 7. Rollback

1. Остановить текущий процесс/контейнер.
2. Вернуться к предыдущей стабильной ревизии.
3. Пересобрать backend и frontend из той же ревизии.
4. Повторить smoke checks.

## 8. Важное замечание по секретам

`.env.local` должен храниться только локально или в secret manager. Если секреты раньше уже были закоммичены, их нужно ротировать независимо от того, что файл больше не отслеживается в Git.
