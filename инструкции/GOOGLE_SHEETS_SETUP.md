# Настройка Google Sheets интеграции

## Шаг 1: Создание Service Account в Google Cloud

1. Перейдите в [Google Cloud Console](https://console.cloud.google.com/)
2. Создайте новый проект или выберите существующий
3. Включите Google Sheets API:
   - Перейдите в "APIs & Services" → "Library"
   - Найдите "Google Sheets API" и включите его
4. Создайте Service Account:
   - Перейдите в "APIs & Services" → "Credentials"
   - Нажмите "Create Credentials" → "Service Account"
   - Заполните данные и создайте аккаунт
5. Создайте ключ:
   - Откройте созданный Service Account
   - Перейдите во вкладку "Keys"
   - Нажмите "Add Key" → "Create new key"
   - Выберите формат JSON
   - Скачайте файл

## Шаг 2: Предоставление доступа к таблице

1. Откройте скачанный JSON файл
2. Найдите поле `client_email` (например: `your-service@project.iam.gserviceaccount.com`)
3. Откройте вашу Google таблицу: https://docs.google.com/spreadsheets/d/15-S90u4kI97Kp1NRNhyyA_cuFriUwWAgmGEa80zZ5EI/edit
4. Нажмите "Поделиться" (Share) в правом верхнем углу
5. Вставьте email из `client_email` и дайте права "Редактор" (Editor)
6. Нажмите "Отправить"

## Шаг 3: Настройка переменных окружения на Render

1. Откройте JSON файл с credentials
2. Скопируйте ВСЁ содержимое файла (весь JSON)
3. На Render в разделе Environment Variables добавьте:
   - `GOOGLE_SHEETS_CREDENTIALS` = вставьте весь JSON (как одну строку)
   - `GOOGLE_SHEETS_ID` = `15-S90u4kI97Kp1NRNhyyA_cuFriUwWAgmGEa80zZ5EI` (уже установлено по умолчанию)
   - `ADMINS` = `@ezhigval, @mrfilmpro` (или ваши username)

## Формат данных в таблице

Таблица "Список гостей" должна иметь следующие столбцы:
- **Столбец A**: Имя и Фамилия
- **Столбец B**: Возраст
- **Столбец C**: Подтверждение (чекбокс "ДА")
- **Столбец D**: Категория ("Семья", "Родственники", "Друзья")
- **Столбец E**: Сторона ("Жених", "Невеста", "Общие")

## Примечание

Если Google Sheets интеграция не настроена, приложение будет работать нормально, но данные не будут экспортироваться в таблицу. Ошибки будут логироваться, но не будут блокировать регистрацию гостей.

