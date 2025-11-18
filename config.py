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

# Путь к фотографии
PHOTO_PATH = "data/wedding_photo.jpg"

# Путь к веб-приложению
WEBAPP_PATH = "webapp"

