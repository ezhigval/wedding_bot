import os
from datetime import datetime
from dotenv import load_dotenv

load_dotenv()

# Токен бота
BOT_TOKEN = os.getenv("BOT_TOKEN", "").strip() if os.getenv("BOT_TOKEN") else ""
# Дополнительная очистка от пробелов и кавычек
if BOT_TOKEN:
    BOT_TOKEN = BOT_TOKEN.strip().strip('"').strip("'").strip()

# Данные о свадьбе
WEDDING_DATE = datetime.strptime(os.getenv("WEDDING_DATE", "2026-06-06"), "%Y-%m-%d")
GROOM_NAME = os.getenv("GROOM_NAME", "Валентин")
BRIDE_NAME = os.getenv("BRIDE_NAME", "Мария")
WEDDING_ADDRESS = os.getenv("WEDDING_ADDRESS", "Санкт-Петербург")

# URL Mini App
WEBAPP_URL = os.getenv("WEBAPP_URL", "https://your-webapp-url.com")

# ID администратора (для команд управления)
ADMIN_USER_ID = os.getenv("ADMIN_USER_ID")

# Путь к базе данных
DB_PATH = "data/wedding.db"

# Путь к фотографии (для бота)
PHOTO_PATH = "res/welcome_photo.jpeg"
# Путь к фотографии для Mini App
WEBAPP_PHOTO_PATH = "res/welcome_photo.jpeg"

# Путь к веб-приложению
WEBAPP_PATH = "webapp"

# Телеграм-аккаунты для связи
GROOM_TELEGRAM = os.getenv("GROOM_TELEGRAM", "ezhigval")
BRIDE_TELEGRAM = os.getenv("BRIDE_TELEGRAM", "mrfilmpro")

# Ссылка на группу для гостей
GROUP_LINK = os.getenv("GROUP_LINK", "https://t.me/+ow7ttcFCmoUzYzRi")
GROUP_ID = os.getenv("GROUP_ID", "")  # ID группы (можно получить через @userinfobot или API)

# Файл с админами
ADMINS_FILE = "admins.json"

# Список админов из переменной окружения (формат: @username1, @username2)
ADMINS_ENV = os.getenv("ADMINS", "@ezhigval, @mrfilmpro")
ADMINS_LIST = [admin.strip().replace('@', '') for admin in ADMINS_ENV.split(',') if admin.strip()]

# Google Sheets настройки
GOOGLE_SHEETS_ID = os.getenv("GOOGLE_SHEETS_ID", "15-S90u4kI97Kp1NRNhyyA_cuFriUwWAgmGEa80zZ5EI")
GOOGLE_SHEETS_CREDENTIALS = os.getenv("GOOGLE_SHEETS_CREDENTIALS", "")  # JSON credentials
GOOGLE_SHEETS_SHEET_NAME = "Список гостей"
GOOGLE_SHEETS_INVITATIONS_SHEET_NAME = "Пригласительные"  # Вкладка для списка приглашений
GOOGLE_SHEETS_ADMINS_SHEET_NAME = "Админ бота"  # Вкладка для списка админов
GOOGLE_SHEETS_TIMELINE_SHEET_NAME = "Публичная План-сетка"  # Вкладка для плана-сетки мероприятия

