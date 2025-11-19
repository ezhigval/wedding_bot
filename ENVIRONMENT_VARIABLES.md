# Переменные окружения для Render

Скопируйте и вставьте эти переменные в раздел **Environment** на Render:

## Обязательные переменные:

```
BOT_TOKEN=8295772557:AAH_udSU1FyjYH-mekFDZht2Yuy0EKI7s5g
WEBAPP_URL=https://wedding-bot-r0y0.onrender.com
WEDDING_DATE=2026-06-06
GROOM_NAME=Валентин
BRIDE_NAME=Мария
WEDDING_ADDRESS=Санкт-Петербург
GROOM_TELEGRAM=ezhigval
BRIDE_TELEGRAM=mrfilmpro
ADMINS=@ezhigval, @mrfilmpro
GOOGLE_SHEETS_ID=15-S90u4kI97Kp1NRNhyyA_cuFriUwWAgmGEa80zZ5EI
```

## Опциональные переменные (если нужно изменить):

```
ADMIN_USER_ID=ваш_telegram_user_id
GOOGLE_SHEETS_CREDENTIALS={"type":"service_account",...}
```

## Как добавить на Render:

1. Откройте ваш сервис на Render
2. Перейдите в раздел **Environment**
3. Нажмите **Add Environment Variable**
4. Добавьте каждую переменную по отдельности:
   - **Key**: `BOT_TOKEN`
   - **Value**: `8295772557:AAH_udSU1FyjYH-mekFDZht2Yuy0EKI7s5g`
5. Повторите для всех переменных

## Важно:

- **BOT_TOKEN** - обязательно, без пробелов и кавычек
- **WEBAPP_URL** - должен совпадать с URL вашего сервиса на Render
- **ADMINS** - список админов через запятую, с @ или без
- **GOOGLE_SHEETS_CREDENTIALS** - если используете Google Sheets, вставьте весь JSON одной строкой

