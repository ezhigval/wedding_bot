# ⚡ Быстрая настройка Google Sheets на Render

## Проблема
В логах видно: `WARNING:google_sheets:GOOGLE_SHEETS_CREDENTIALS не установлен`

Это значит, что переменная окружения `GOOGLE_SHEETS_CREDENTIALS` не настроена на Render.

## Решение

### Шаг 1: Получите JSON credentials

1. Если у вас уже есть Service Account JSON файл - откройте его
2. Если нет - следуйте инструкции в `GOOGLE_SHEETS_ADMINS_SETUP.md`

### Шаг 2: Подготовьте JSON для Render

1. Откройте JSON файл в текстовом редакторе
2. **Удалите все переносы строк** - JSON должен быть в одну строку
3. Скопируйте весь JSON (от `{` до `}`)

Пример правильного формата (в одну строку):
```
{"type":"service_account","project_id":"your-project","private_key_id":"abc123","private_key":"-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC...\n-----END PRIVATE KEY-----\n","client_email":"wedding-bot@your-project.iam.gserviceaccount.com","client_id":"123456789","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_x509_cert_url":"https://www.googleapis.com/robot/v1/metadata/x509/wedding-bot%40your-project.iam.gserviceaccount.com"}
```

### Шаг 3: Добавьте на Render

1. Откройте ваш сервис на Render: https://dashboard.render.com
2. Перейдите в **Environment** (Переменные окружения)
3. Найдите или создайте переменную: `GOOGLE_SHEETS_CREDENTIALS`
4. Вставьте JSON (весь в одну строку) в поле значения
5. Нажмите **Save Changes**
6. Render автоматически перезапустит сервис

### Шаг 4: Проверьте доступ Service Account к таблице

1. Откройте вашу Google таблицу
2. Нажмите **"Настройки доступа"** (Share)
3. Найдите email Service Account (из JSON файла, поле `client_email`)
4. Убедитесь, что у него права **"Редактор"** (Editor)
5. Если нет - добавьте email и установите права "Редактор"

### Шаг 5: Проверьте вкладку "Админ бота"

1. В Google таблице создайте вкладку **"Админ бота"** (если её нет)
2. В столбце A добавьте username админов (без @):
   - `ezhigval`
   - `mrfilmpro`
3. Столбец B заполнится автоматически при вызове `/set_me_admins`

## Проверка

После настройки:

1. Подождите, пока Render перезапустит сервис (1-2 минуты)
2. Напишите боту `/set_me_admins`
3. Проверьте логи на Render - должно быть:
   ```
   INFO:google_sheets:Клиент Google Sheets создан успешно
   INFO:google_sheets:✅ Админ @ezhigval добавлен в Google Sheets
   ```
4. Проверьте Google таблицу - в столбце B должен появиться ваш `user_id`
5. Попробуйте `/admin` - должен работать!

## Если не работает

1. **Проверьте логи на Render** - там будет точная ошибка
2. **Убедитесь, что JSON в одну строку** - без переносов
3. **Проверьте права доступа** - Service Account должен быть редактором таблицы
4. **Проверьте название вкладки** - должно быть точно "Админ бота"

## Важно

- JSON должен быть **в одну строку** без переносов
- Service Account email должен быть добавлен в таблицу как **"Редактор"**
- Вкладка должна называться точно **"Админ бота"**

