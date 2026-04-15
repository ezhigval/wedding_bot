# Чек-лист деплоя

Подробная инструкция находится в [DEPLOYMENT.md](DEPLOYMENT.md). Этот файл — короткий операционный список перед релизом.

## Перед выкладкой

- `main` синхронизирован с удалённым репозиторием.
- Рабочее дерево чистое.
- Production `.env.local` подготовлен вне Git.

## Обязательные проверки

```bash
make verify
cd webapp-react && npm audit --omit=dev
```

## Инфраструктура

- `BOT_TOKEN` актуален.
- `GOOGLE_SHEETS_ID` указывает на рабочую таблицу.
- `GOOGLE_SHEETS_CREDENTIALS` или `GOOGLE_SHEETS_CREDENTIALS_BASE64` заполнены корректно.
- `GOOGLE_DRIVE_FOLDER_ID` указывает на рабочую папку.
- `WEBAPP_URL` совпадает с реальным публичным адресом.
- `GROUP_ID` и `GROUP_LINK` заполнены, если используется общий чат.
- `DEBUG=false`.

## После запуска

- `/health` отвечает `200`.
- `/api/config` отвечает корректным JSON.
- Mini App открывается.
- Бот отвечает на `/start`.
- RSVP и фото-поток работают.
- В логах нет ошибок инициализации Google Sheets/Drive и panic.
