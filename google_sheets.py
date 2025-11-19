"""
Интеграция с Google Sheets для экспорта данных гостей
"""
import os
import json
import logging
from typing import List, Dict, Optional
import asyncio

logger = logging.getLogger(__name__)

try:
    import gspread
    from google.oauth2.service_account import Credentials
    GSPREAD_AVAILABLE = True
except ImportError:
    GSPREAD_AVAILABLE = False
    logger.warning("gspread не установлен. Интеграция с Google Sheets недоступна.")

from config import GOOGLE_SHEETS_ID, GOOGLE_SHEETS_CREDENTIALS, GOOGLE_SHEETS_SHEET_NAME, GOOGLE_SHEETS_INVITATIONS_SHEET_NAME, GOOGLE_SHEETS_ADMINS_SHEET_NAME

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
        # Запускаем синхронный код в executor
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _add_guest_to_sheets_sync, first_name, last_name, age, category, side)
        return result
        
    except Exception as e:
        logger.error(f"Ошибка добавления гостя в Google Sheets: {e}")
        return False

def _add_guest_to_sheets_sync(first_name: str, last_name: str, age: Optional[int] = None, 
                              category: Optional[str] = None, side: Optional[str] = None):
    """Синхронная функция для добавления/обновления гостя в Google Sheets"""
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        # Открываем таблицу
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        
        # Формируем полное имя для поиска
        full_name = f"{first_name} {last_name}".strip()
        
        # Получаем все данные из столбца A (имена)
        try:
            all_values = worksheet.get_all_values()
            
            # Ищем существующую строку по имени и фамилии (столбец A)
            found_row = None
            for row_idx, row in enumerate(all_values, start=1):
                if len(row) > 0:
                    # Сравниваем имя в столбце A (без учета регистра и пробелов)
                    existing_name = row[0].strip() if row[0] else ""
                    if existing_name.lower() == full_name.lower():
                        found_row = row_idx
                        break
            
            if found_row:
                # Гость уже существует - обновляем столбцы C, D, E
                # Столбец C = индекс 3 (подтверждение)
                # Столбец D = индекс 4 (Родство/Категория)
                # Столбец E = индекс 5 (Сторона)
                worksheet.update_cell(found_row, 3, "ДА")  # Столбец C - Подтверждение
                
                # Обновляем Родство (столбец D) если передано
                if category:
                    worksheet.update_cell(found_row, 4, category)  # Столбец D - Родство
                
                # Обновляем Сторону (столбец E) если передано
                if side:
                    worksheet.update_cell(found_row, 5, side)  # Столбец E - Сторона
                
                logger.info(f"Гость {full_name} найден в строке {found_row}, обновлено подтверждение на 'ДА' и данные (родство: {category}, сторона: {side})")
                return True
            else:
                # Гость не найден - добавляем новую строку
                row_data = [
                    full_name,              # Столбец A - Имя и Фамилия
                    str(age) if age else "", # Столбец B - Возраст
                    "ДА",                   # Столбец C - Подтверждение (чекбокс)
                    category if category else "", # Столбец D - Категория
                    side if side else ""    # Столбец E - Сторона
                ]
                
                worksheet.append_row(row_data)
                logger.info(f"Гость {full_name} добавлен в Google Sheets (новая строка)")
                return True
                
        except Exception as e:
            logger.error(f"Ошибка при поиске/добавлении гостя в Google Sheets: {e}")
            # Если поиск не удался, пробуем просто добавить
            row_data = [
                full_name,              # Столбец A - Имя и Фамилия
                str(age) if age else "", # Столбец B - Возраст
                "ДА",                   # Столбец C - Подтверждение (чекбокс)
                category if category else "", # Столбец D - Категория
                side if side else ""    # Столбец E - Сторона
            ]
            worksheet.append_row(row_data)
            logger.info(f"Гость {full_name} добавлен в Google Sheets (fallback)")
            return True
            
    except Exception as e:
        logger.error(f"Ошибка добавления гостя в Google Sheets: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

async def cancel_invitation(first_name: str, last_name: str):
    """
    Отменить приглашение - обновить столбец C на "НЕТ"
    
    Args:
        first_name: Имя
        last_name: Фамилия
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return False
    
    try:
        # Запускаем синхронный код в executor
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _cancel_invitation_sync, first_name, last_name)
        return result
        
    except Exception as e:
        logger.error(f"Ошибка отмены приглашения в Google Sheets: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

def _cancel_invitation_sync(first_name: str, last_name: str):
    """Синхронная функция для отмены приглашения в Google Sheets"""
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        # Открываем таблицу
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        
        # Формируем полное имя для поиска
        full_name = f"{first_name} {last_name}".strip()
        
        # Получаем все данные
        all_values = worksheet.get_all_values()
        
        # Ищем существующую строку по имени и фамилии
        found_row = None
        for row_idx, row in enumerate(all_values, start=1):
            if len(row) > 0:
                existing_name = row[0].strip() if row[0] else ""
                if existing_name.lower() == full_name.lower():
                    found_row = row_idx
                    break
        
        if found_row:
            # Обновляем столбец C на "НЕТ"
            worksheet.update_cell(found_row, 3, "НЕТ")  # Столбец C - Подтверждение
            logger.info(f"Приглашение для {full_name} отменено (строка {found_row})")
            return True
        else:
            logger.warning(f"Гость {full_name} не найден в Google Sheets для отмены")
            return False
            
    except Exception as e:
        logger.error(f"Ошибка отмены приглашения в Google Sheets: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

def normalize_telegram_id(telegram_id: str) -> str:
    """
    Нормализует телеграм ID к единому формату (без @, без t.me/)
    
    Args:
        telegram_id: Может быть в формате:
            - t.me/username
            - @username
            - username
    
    Returns:
        Нормализованный username (без @ и без t.me/)
    """
    if not telegram_id:
        return ""
    
    telegram_id = telegram_id.strip()
    
    # Убираем t.me/
    if telegram_id.startswith("t.me/"):
        telegram_id = telegram_id[5:]
    elif telegram_id.startswith("https://t.me/"):
        telegram_id = telegram_id[13:]
    elif telegram_id.startswith("http://t.me/"):
        telegram_id = telegram_id[12:]
    
    # Убираем @
    if telegram_id.startswith("@"):
        telegram_id = telegram_id[1:]
    
    return telegram_id

async def get_invitations_list() -> List[Dict[str, str]]:
    """
    Получить список приглашений из Google Sheets (вкладка "Пригласительные")
    
    Returns:
        Список словарей с ключами 'name' и 'telegram_id'
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return []
    
    try:
        # Запускаем синхронный код в executor
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _get_invitations_list_sync)
        return result
        
    except Exception as e:
        logger.error(f"Ошибка получения списка приглашений из Google Sheets: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []

def _get_invitations_list_sync() -> List[Dict[str, str]]:
    """Синхронная функция для получения списка приглашений из Google Sheets"""
    try:
        client = get_google_sheets_client()
        if not client:
            return []
        
        # Открываем таблицу
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        
        # Пытаемся открыть вкладку "Пригласительные"
        try:
            worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_INVITATIONS_SHEET_NAME)
        except Exception as e:
            logger.error(f"Вкладка '{GOOGLE_SHEETS_INVITATIONS_SHEET_NAME}' не найдена: {e}")
            return []
        
        # Получаем все данные
        all_values = worksheet.get_all_values()
        
        # Пропускаем заголовок (первую строку, если есть)
        # Столбец A - имя, столбец B - телеграм ID
        invitations = []
        start_row = 0
        
        # Проверяем, есть ли заголовок
        if all_values and len(all_values) > 0:
            first_row = all_values[0]
            # Если первая строка похожа на заголовок (содержит "имя" или "телеграм"), пропускаем
            if any(keyword in str(first_row[0]).lower() for keyword in ['имя', 'name', 'гость']):
                start_row = 1
        
        # Обрабатываем данные
        for row in all_values[start_row:]:
            if len(row) >= 2:
                name = row[0].strip() if row[0] else ""
                telegram_id = row[1].strip() if row[1] else ""
                
                # Пропускаем пустые строки
                if not name and not telegram_id:
                    continue
                
                # Нормализуем телеграм ID
                normalized_id = normalize_telegram_id(telegram_id)
                
                if name and normalized_id:
                    invitations.append({
                        'name': name,
                        'telegram_id': normalized_id
                    })
        
        logger.info(f"Получено {len(invitations)} приглашений из Google Sheets")
        return invitations
        
    except Exception as e:
        logger.error(f"Ошибка получения списка приглашений из Google Sheets: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []

async def get_admins_list() -> List[Dict[str, any]]:
    """
    Получить список админов из Google Sheets (вкладка "Админ бота")
    
    Returns:
        Список словарей с ключами 'username' и 'user_id' (если есть)
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return []
    
    try:
        # Запускаем синхронный код в executor
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _get_admins_list_sync)
        return result
        
    except Exception as e:
        logger.error(f"Ошибка получения списка админов из Google Sheets: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []

def _get_admins_list_sync() -> List[Dict[str, any]]:
    """Синхронная функция для получения списка админов из Google Sheets"""
    try:
        client = get_google_sheets_client()
        if not client:
            return []
        
        # Открываем таблицу
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        
        # Пытаемся открыть вкладку "Админ бота"
        try:
            worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_ADMINS_SHEET_NAME)
        except Exception as e:
            logger.error(f"Вкладка '{GOOGLE_SHEETS_ADMINS_SHEET_NAME}' не найдена: {e}")
            return []
        
        # Получаем все данные
        all_values = worksheet.get_all_values()
        
        # Пропускаем заголовок (первую строку, если есть)
        # Столбец A - username, столбец B - user_id (опционально)
        admins = []
        start_row = 0
        
        # Проверяем, есть ли заголовок
        if all_values and len(all_values) > 0:
            first_row = all_values[0]
            # Если первая строка похожа на заголовок, пропускаем
            if any(keyword in str(first_row[0]).lower() for keyword in ['username', 'админ', 'admin']):
                start_row = 1
        
        # Обрабатываем данные
        for row in all_values[start_row:]:
            if len(row) >= 1:
                username = row[0].strip() if row[0] else ""
                user_id = row[1].strip() if len(row) >= 2 and row[1] else None
                
                # Пропускаем пустые строки
                if not username:
                    continue
                
                # Убираем @ если есть
                username = username.replace('@', '').lower()
                
                admin_data = {
                    'username': username,
                    'name': username,
                    'telegram': username
                }
                
                # Если есть user_id, добавляем его
                if user_id and user_id.isdigit():
                    admin_data['user_id'] = int(user_id)
                
                admins.append(admin_data)
        
        logger.info(f"Получено {len(admins)} админов из Google Sheets")
        return admins
        
    except Exception as e:
        logger.error(f"Ошибка получения списка админов из Google Sheets: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []

async def save_admin_to_sheets(username: str, user_id: int):
    """
    Сохранить или обновить админа в Google Sheets (вкладка "Админ бота")
    
    Args:
        username: Username админа (без @)
        user_id: Telegram user_id админа
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return False
    
    try:
        # Запускаем синхронный код в executor
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _save_admin_to_sheets_sync, username, user_id)
        return result
        
    except Exception as e:
        logger.error(f"Ошибка сохранения админа в Google Sheets: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

def _save_admin_to_sheets_sync(username: str, user_id: int):
    """Синхронная функция для сохранения админа в Google Sheets"""
    try:
        logger.info(f"Попытка сохранить админа в Google Sheets: username={username}, user_id={user_id}")
        
        # Проверяем credentials
        if not GOOGLE_SHEETS_CREDENTIALS:
            logger.error("GOOGLE_SHEETS_CREDENTIALS не установлен!")
            return False
        
        client = get_google_sheets_client()
        if not client:
            logger.error("Не удалось создать клиент Google Sheets")
            return False
        
        logger.info(f"Клиент Google Sheets создан успешно, открываем таблицу: {GOOGLE_SHEETS_ID}")
        
        # Открываем таблицу
        try:
            spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
            logger.info(f"Таблица открыта успешно: {spreadsheet.title}")
        except Exception as e:
            logger.error(f"Ошибка открытия таблицы по ID {GOOGLE_SHEETS_ID}: {e}")
            import traceback
            logger.error(traceback.format_exc())
            return False
        
        # Пытаемся открыть вкладку "Админ бота"
        try:
            worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_ADMINS_SHEET_NAME)
            logger.info(f"Вкладка '{GOOGLE_SHEETS_ADMINS_SHEET_NAME}' найдена")
        except Exception as e:
            logger.error(f"Вкладка '{GOOGLE_SHEETS_ADMINS_SHEET_NAME}' не найдена: {e}")
            logger.info(f"Доступные вкладки: {[ws.title for ws in spreadsheet.worksheets()]}")
            return False
        
        username_lower = username.lower().replace('@', '')
        
        # Получаем все данные
        try:
            all_values = worksheet.get_all_values()
            logger.info(f"Получено {len(all_values)} строк из вкладки 'Админ бота'")
        except Exception as e:
            logger.error(f"Ошибка чтения данных из вкладки: {e}")
            import traceback
            logger.error(traceback.format_exc())
            return False
        
        # Пропускаем заголовок
        start_row = 0
        if all_values and len(all_values) > 0:
            first_row = all_values[0]
            logger.info(f"Первая строка: {first_row}")
            if any(keyword in str(first_row[0]).lower() for keyword in ['username', 'админ', 'admin']):
                start_row = 1
                logger.info("Обнаружен заголовок, пропускаем первую строку")
        
        # Ищем существующую строку по username
        found_row = None
        for row_idx, row in enumerate(all_values[start_row:], start=start_row + 1):
            if len(row) > 0:
                existing_username = row[0].strip().replace('@', '').lower() if row[0] else ""
                logger.debug(f"Строка {row_idx}: username='{existing_username}', ищем '{username_lower}'")
                if existing_username == username_lower:
                    found_row = row_idx
                    logger.info(f"Найдена существующая строка для @{username} в строке {found_row}")
                    break
        
        if found_row:
            # Обновляем user_id в столбце B
            try:
                worksheet.update_cell(found_row, 2, str(user_id))  # Столбец B - user_id
                logger.info(f"✅ Админ @{username} обновлен в Google Sheets (строка {found_row}, user_id: {user_id})")
                
                # Проверяем, что обновление прошло успешно
                updated_value = worksheet.cell(found_row, 2).value
                logger.info(f"Проверка: значение в ячейке B{found_row} = '{updated_value}'")
            except Exception as e:
                logger.error(f"Ошибка обновления ячейки: {e}")
                import traceback
                logger.error(traceback.format_exc())
                return False
        else:
            # Добавляем новую строку
            row_data = [
                username_lower,  # Столбец A - username
                str(user_id)     # Столбец B - user_id
            ]
            try:
                worksheet.append_row(row_data)
                logger.info(f"✅ Админ @{username} добавлен в Google Sheets (user_id: {user_id})")
                
                # Проверяем, что добавление прошло успешно
                all_values_after = worksheet.get_all_values()
                logger.info(f"После добавления: {len(all_values_after)} строк в таблице")
            except Exception as e:
                logger.error(f"Ошибка добавления строки: {e}")
                import traceback
                logger.error(traceback.format_exc())
                return False
        
        return True
        
    except Exception as e:
        logger.error(f"Критическая ошибка сохранения админа в Google Sheets: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False
