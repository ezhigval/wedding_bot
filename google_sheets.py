"""
Интеграция с Google Sheets для экспорта данных гостей
"""
import os
import json
import logging
from typing import List, Dict, Optional

logger = logging.getLogger(__name__)

try:
    import gspread
    from google.oauth2.service_account import Credentials
    GSPREAD_AVAILABLE = True
except ImportError:
    GSPREAD_AVAILABLE = False
    logger.warning("gspread не установлен. Интеграция с Google Sheets недоступна.")

from config import GOOGLE_SHEETS_ID, GOOGLE_SHEETS_CREDENTIALS, GOOGLE_SHEETS_SHEET_NAME

def get_google_sheets_client():
    """Получить клиент Google Sheets"""
    if not GSPREAD_AVAILABLE:
        return None
    
    if not GOOGLE_SHEETS_CREDENTIALS:
        logger.warning("GOOGLE_SHEETS_CREDENTIALS не установлен")
        return None
    
    try:
        # Парсим JSON credentials из переменной окружения
        creds_dict = json.loads(GOOGLE_SHEETS_CREDENTIALS)
        creds = Credentials.from_service_account_info(
            creds_dict,
            scopes=['https://www.googleapis.com/auth/spreadsheets']
        )
        client = gspread.authorize(creds)
        return client
    except Exception as e:
        logger.error(f"Ошибка создания клиента Google Sheets: {e}")
        return None

async def add_guest_to_sheets(first_name: str, last_name: str, age: Optional[int] = None, 
                             category: Optional[str] = None, side: Optional[str] = None):
    """
    Добавить гостя в Google Sheets
    
    Args:
        first_name: Имя
        last_name: Фамилия
        age: Возраст (опционально)
        category: Категория - "Семья", "Родственники", "Друзья" (опционально)
        side: Сторона - "Жених", "Невеста", "Общие" (опционально)
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return False
    
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        # Открываем таблицу
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        
        # Находим первую свободную строку
        all_values = worksheet.get_all_values()
        next_row = len(all_values) + 1
        
        # Подготавливаем данные
        row_data = [
            f"{first_name} {last_name}",  # Столбец A - Имя и Фамилия
            str(age) if age else "",       # Столбец B - Возраст
            "ДА",                          # Столбец C - Подтверждение (чекбокс)
            category if category else "", # Столбец D - Категория
            side if side else ""           # Столбец E - Сторона
        ]
        
        # Добавляем строку
        worksheet.append_row(row_data)
        
        logger.info(f"Гость {first_name} {last_name} добавлен в Google Sheets")
        return True
        
    except Exception as e:
        logger.error(f"Ошибка добавления гостя в Google Sheets: {e}")
        return False

