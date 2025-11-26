"""
Интеграция с Google Sheets для экспорта данных гостей
"""
import os
import json
import logging
from typing import List, Dict, Optional, Tuple
import asyncio
from datetime import datetime
import time

logger = logging.getLogger(__name__)

try:
    import gspread
    from google.oauth2.service_account import Credentials
    GSPREAD_AVAILABLE = True
except ImportError:
    GSPREAD_AVAILABLE = False
    logger.warning("gspread не установлен. Интеграция с Google Sheets недоступна.")

from config import GOOGLE_SHEETS_ID, GOOGLE_SHEETS_CREDENTIALS, GOOGLE_SHEETS_SHEET_NAME, GOOGLE_SHEETS_INVITATIONS_SHEET_NAME, GOOGLE_SHEETS_ADMINS_SHEET_NAME, GOOGLE_SHEETS_TIMELINE_SHEET_NAME

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
                             category: Optional[str] = None, side: Optional[str] = None, 
                             user_id: Optional[int] = None):
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
        result = await loop.run_in_executor(None, _add_guest_to_sheets_sync, first_name, last_name, age, category, side, user_id)
        return result
        
    except Exception as e:
        logger.error(f"Ошибка добавления гостя в Google Sheets: {e}")
        return False

def _add_guest_to_sheets_sync(first_name: str, last_name: str, age: Optional[int] = None, 
                              category: Optional[str] = None, side: Optional[str] = None, 
                              user_id: Optional[int] = None):
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
                # Гость уже существует - обновляем столбцы C, D, E, F
                # Столбец C = индекс 3 (подтверждение)
                # Столбец D = индекс 4 (Родство/Категория)
                # Столбец E = индекс 5 (Сторона)
                # Столбец F = индекс 6 (user_id)
                worksheet.update_cell(found_row, 3, "ДА")  # Столбец C - Подтверждение
                
                # Обновляем Родство (столбец D) если передано
                if category:
                    worksheet.update_cell(found_row, 4, category)  # Столбец D - Родство
                
                # Обновляем Сторону (столбец E) если передано
                if side:
                    worksheet.update_cell(found_row, 5, side)  # Столбец E - Сторона
                
                # Обновляем user_id (столбец F) если передан
                if user_id:
                    worksheet.update_cell(found_row, 6, str(user_id))  # Столбец F - user_id
                
                logger.info(f"Гость {full_name} найден в строке {found_row}, обновлено подтверждение на 'ДА' и данные (родство: {category}, сторона: {side}, user_id: {user_id})")
                return True
            else:
                # Гость не найден - ищем первую свободную строку (где столбец A пустой)
                # Используем проверку ячеек напрямую, так как get_all_values() не возвращает полностью пустые строки
                empty_row = None
                
                # Сначала проверяем строки из all_values (где есть хотя бы какие-то данные)
                for row_idx, row in enumerate(all_values, start=1):
                    # Проверяем, пуст ли столбец A (имя)
                    if len(row) == 0 or (row[0] if row else "").strip() == "":
                        empty_row = row_idx
                        break
                
                # Если не нашли в существующих строках, проверяем ячейки напрямую
                # Начинаем с первой строки после заголовка (строка 2, если заголовок в строке 1)
                if empty_row is None:
                    # Получаем все значения столбца A для проверки
                    col_a_values = worksheet.col_values(1)  # Получаем все значения столбца A
                    
                    # Ищем первую пустую ячейку в столбце A
                    # Начинаем с индекса 1 (строка 2), так как строка 1 обычно заголовок
                    for idx in range(1, len(col_a_values) + 100):  # Проверяем до 100 строк
                        try:
                            cell_value = worksheet.cell(idx + 1, 1).value  # idx+1 потому что индексация с 1
                            if not cell_value or str(cell_value).strip() == "":
                                empty_row = idx + 1
                                break
                        except Exception:
                            # Если ячейка не существует, значит это пустая строка
                            empty_row = idx + 1
                            break
                
                # Если все еще не нашли, используем следующую после последней заполненной
                if empty_row is None:
                    empty_row = len(all_values) + 1
                
                # Вставляем данные в найденную свободную строку
                row_data = [
                    full_name,              # Столбец A - Имя и Фамилия
                    str(age) if age else "", # Столбец B - Возраст
                    "ДА",                   # Столбец C - Подтверждение (чекбокс)
                    category if category else "", # Столбец D - Категория
                    side if side else "",   # Столбец E - Сторона
                    str(user_id) if user_id else ""  # Столбец F - user_id
                ]
                
                # Обновляем строку используя update вместо append_row
                worksheet.update(f'A{empty_row}:F{empty_row}', [row_data])
                logger.info(f"Гость {full_name} добавлен в Google Sheets в строку {empty_row} (первая свободная строка)")
                return True
                
        except Exception as e:
            logger.error(f"Ошибка при поиске/добавлении гостя в Google Sheets: {e}")
            # Если поиск не удался, пробуем найти свободную строку и добавить туда
            try:
                all_values = worksheet.get_all_values()
                empty_row = None
                
                # Проверяем существующие строки
                for row_idx, row in enumerate(all_values, start=1):
                    if len(row) == 0 or (row[0] if row else "").strip() == "":
                        empty_row = row_idx
                        break
                
                # Если не нашли, проверяем ячейки напрямую
                if empty_row is None:
                    col_a_values = worksheet.col_values(1)
                    for idx in range(1, len(col_a_values) + 100):
                        try:
                            cell_value = worksheet.cell(idx + 1, 1).value
                            if not cell_value or str(cell_value).strip() == "":
                                empty_row = idx + 1
                                break
                        except Exception:
                            empty_row = idx + 1
                            break
                
                if empty_row is None:
                    empty_row = len(all_values) + 1
                
                    row_data = [
                        full_name,              # Столбец A - Имя и Фамилия
                        str(age) if age else "", # Столбец B - Возраст
                        "ДА",                   # Столбец C - Подтверждение (чекбокс)
                        category if category else "", # Столбец D - Категория
                        side if side else "",   # Столбец E - Сторона
                        str(user_id) if user_id else ""  # Столбец F - user_id
                    ]
                    worksheet.update(f'A{empty_row}:F{empty_row}', [row_data])
                logger.info(f"Гость {full_name} добавлен в Google Sheets в строку {empty_row} (fallback)")
                return True
            except Exception as fallback_error:
                logger.error(f"Ошибка при fallback добавлении: {fallback_error}")
                import traceback
                logger.error(traceback.format_exc())
                return False
            
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
            - номер телефона (8, 7, +7) - возвращается как есть
    
    Returns:
        Нормализованный username (без @ и без t.me/) или номер телефона как есть
    """
    if not telegram_id:
        return ""
    
    telegram_id = telegram_id.strip()
    
    # Если это номер телефона (начинается с 8, 7 или +7), возвращаем как есть
    if telegram_id.startswith("+7") or telegram_id.startswith("7") or telegram_id.startswith("8"):
        return telegram_id
    
    # Убираем t.me/
    if telegram_id.startswith("t.me/"):
        telegram_id = telegram_id[5:]
    elif telegram_id.startswith("https://t.me/"):
        telegram_id = telegram_id[13:]
    elif telegram_id.startswith("http://t.me/"):
        telegram_id = telegram_id[12:]
    
    # Убираем @ только если это username
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
        # Столбец A - имя, столбец B - телеграм ID, столбец C - статус отправки ("ДА" или пусто)
        for row in all_values[start_row:]:
            if len(row) >= 1:
                name = row[0].strip() if row[0] else ""
                telegram_id = row[1].strip() if len(row) > 1 and row[1] else ""
                status = row[2].strip() if len(row) > 2 and row[2] else ""  # Столбец C - статус отправки
                user_id = row[3].strip() if len(row) > 3 and row[3] else ""  # Столбец D - user_id (если есть)
                
                # Пропускаем пустые строки (где нет имени)
                if not name:
                    continue
                
                # Нормализуем телеграм ID, если он есть
                normalized_id = normalize_telegram_id(telegram_id) if telegram_id else ""
                
                # Проверяем статус отправки
                is_sent = status.upper() == "ДА"
                
                invitations.append({
                    'name': name,
                    'telegram_id': normalized_id,  # Может быть пустой строкой
                    'user_id': user_id if user_id else None,  # User ID из столбца D
                    'is_sent': is_sent  # Статус отправки из столбца C
                })
        
        logger.info(f"Получено {len(invitations)} приглашений из Google Sheets")
        return invitations
        
    except Exception as e:
        logger.error(f"Ошибка получения списка приглашений из Google Sheets: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []

async def update_invitation_user_id(guest_name: str, user_id: int, username: str = None) -> bool:
    """
    Обновляет user_id и username в таблице приглашений для гостя по имени
    
    Args:
        guest_name: Имя гостя (из столбца A)
        user_id: User ID для сохранения (в столбец C)
        username: Username для обновления столбца B (опционально)
    
    Returns:
        True если обновление успешно, False в противном случае
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return False
    
    try:
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _update_invitation_user_id_sync, guest_name, user_id, username)
        return result
    except Exception as e:
        logger.error(f"Ошибка обновления user_id в таблице приглашений: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

def _update_invitation_user_id_sync(guest_name: str, user_id: int, username: str = None) -> bool:
    """Синхронная функция для обновления user_id и username в таблице приглашений"""
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        
        try:
            worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_INVITATIONS_SHEET_NAME)
        except Exception as e:
            logger.error(f"Вкладка '{GOOGLE_SHEETS_INVITATIONS_SHEET_NAME}' не найдена: {e}")
            return False
        
        # Получаем все данные
        all_values = worksheet.get_all_values()
        
        # Пропускаем заголовок
        start_row = 0
        if all_values and len(all_values) > 0:
            first_row = all_values[0]
            if any(keyword in str(first_row[0]).lower() for keyword in ['имя', 'name', 'гость']):
                start_row = 1
        
        # Ищем строку с именем гостя
        found_row = None
        for row_idx, row in enumerate(all_values[start_row:], start=start_row + 1):
            if len(row) > 0:
                name = row[0].strip() if row[0] else ""
                if name.lower() == guest_name.lower():
                    found_row = row_idx
                    break
        
        if found_row:
            # Если передан username, обновляем столбец B (индекс 2)
            if username:
                # Убираем @ если есть
                username_clean = username.lstrip('@')
                worksheet.update_cell(found_row, 2, username_clean)
            
            # Обновляем столбец D (индекс 4) с user_id, если передан
            if user_id:
                worksheet.update_cell(found_row, 4, str(user_id))
            
            if username and user_id:
                logger.info(f"Обновлены username и user_id для {guest_name} в строке {found_row}: username={username_clean}, user_id={user_id}")
            elif username:
                logger.info(f"Обновлен username для {guest_name} в строке {found_row}: {username_clean}")
            elif user_id:
                logger.info(f"Обновлен user_id для {guest_name} в строке {found_row}: {user_id}")
            return True
        else:
            logger.warning(f"Гость {guest_name} не найден в таблице приглашений")
            return False
        
    except Exception as e:
        logger.error(f"Ошибка обновления user_id в таблице приглашений: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

async def mark_invitation_as_sent(guest_name: str) -> bool:
    """
    Отметить приглашение как отправленное (столбец C = "ДА")
    
    Args:
        guest_name: Имя гостя (из столбца A)
    
    Returns:
        True если обновление успешно, False в противном случае
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return False
    
    try:
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _mark_invitation_as_sent_sync, guest_name)
        return result
    except Exception as e:
        logger.error(f"Ошибка отметки приглашения как отправленного: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

def _mark_invitation_as_sent_sync(guest_name: str) -> bool:
    """Синхронная функция для отметки приглашения как отправленного"""
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        
        try:
            worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_INVITATIONS_SHEET_NAME)
        except Exception as e:
            logger.error(f"Вкладка '{GOOGLE_SHEETS_INVITATIONS_SHEET_NAME}' не найдена: {e}")
            return False
        
        # Получаем все данные
        all_values = worksheet.get_all_values()
        
        # Пропускаем заголовок
        start_row = 0
        if all_values and len(all_values) > 0:
            first_row = all_values[0]
            if any(keyword in str(first_row[0]).lower() for keyword in ['имя', 'name', 'гость']):
                start_row = 1
        
        # Ищем строку с именем гостя
        found_row = None
        for row_idx, row in enumerate(all_values[start_row:], start=start_row + 1):
            if len(row) > 0:
                name = row[0].strip() if row[0] else ""
                if name.lower() == guest_name.lower():
                    found_row = row_idx
                    break
        
        if found_row:
            # Обновляем столбец C (индекс 3) на "ДА"
            worksheet.update_cell(found_row, 3, "ДА")
            logger.info(f"Приглашение для {guest_name} отмечено как отправленное (строка {found_row})")
            return True
        else:
            logger.warning(f"Гость {guest_name} не найден в таблице приглашений")
            return False
        
    except Exception as e:
        logger.error(f"Ошибка отметки приглашения как отправленного: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

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
        # Столбец A - username, столбец B - user_id, столбец D - api_id, столбец E - api_hash, столбец F - phone
        for row in all_values[start_row:]:
            if len(row) >= 1:
                username = row[0].strip() if row[0] else ""
                user_id = row[1].strip() if len(row) >= 2 and row[1] else None
                api_id = row[3].strip() if len(row) >= 4 and row[3] else None  # Столбец D
                api_hash = row[4].strip() if len(row) >= 5 and row[4] else None  # Столбец E
                phone = row[5].strip() if len(row) >= 6 and row[5] else None  # Столбец F
                
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
                
                # Добавляем данные Telegram Client API
                if api_id:
                    admin_data['api_id'] = api_id
                if api_hash:
                    admin_data['api_hash'] = api_hash
                if phone:
                    admin_data['phone'] = phone
                
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

async def get_timeline() -> List[Dict[str, str]]:
    """
    Получить тайминг мероприятия из Google Sheets (вкладка "Публичная План-сетка")
    
    Returns:
        Список словарей с ключами 'time' и 'event'
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return []
    
    try:
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _get_timeline_sync)
        return result
    except Exception as e:
        logger.error(f"Ошибка получения тайминга из Google Sheets: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []

def _get_timeline_sync() -> List[Dict[str, str]]:
    """Синхронная функция для получения тайминга из Google Sheets"""
    try:
        client = get_google_sheets_client()
        if not client:
            return []
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        
        try:
            worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_TIMELINE_SHEET_NAME)
        except Exception:
            logger.warning(f"Вкладка '{GOOGLE_SHEETS_TIMELINE_SHEET_NAME}' не найдена")
            return []
        
        # Получаем все данные (столбец A - время, столбец B - событие)
        values = worksheet.get_all_values()
        
        timeline = []
        for row in values:
            if len(row) >= 2 and row[0].strip() and row[1].strip():
                timeline.append({
                    'time': row[0].strip(),
                    'event': row[1].strip()
                })
        
        return timeline
    except Exception as e:
        logger.error(f"Ошибка получения тайминга: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []

# ========== ФУНКЦИИ ДЛЯ РАБОТЫ С ГОСТЯМИ (ТОЛЬКО GOOGLE SHEETS) ==========

async def check_guest_registration(user_id: int) -> bool:
    """
    Проверить, зарегистрирован ли гость по user_id
    
    Args:
        user_id: Telegram user_id
        
    Returns:
        True если гость зарегистрирован (столбец C = "ДА" и столбец F = user_id)
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return False
    
    try:
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _check_guest_registration_sync, user_id)
        return result
    except Exception as e:
        logger.error(f"Ошибка проверки регистрации гостя: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

def _check_guest_registration_sync(user_id: int) -> bool:
    """Синхронная функция для проверки регистрации гостя"""
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        
        # Получаем все данные
        all_values = worksheet.get_all_values()
        
        # Ищем строку с user_id в столбце F (индекс 5) и подтверждением "ДА" в столбце C (индекс 2)
        for row_idx, row in enumerate(all_values, start=1):
            if len(row) > 5:  # Проверяем, что есть столбец F
                user_id_cell = row[5].strip() if row[5] else ""  # Столбец F (индекс 5)
                confirmation = row[2].strip() if len(row) > 2 and row[2] else ""  # Столбец C (индекс 2)
                
                if user_id_cell == str(user_id) and confirmation.upper() == "ДА":
                    logger.info(f"Гость с user_id {user_id} найден и зарегистрирован (строка {row_idx})")
                    return True
        
        logger.info(f"Гость с user_id {user_id} не найден или не подтвердил участие")
        return False
    except Exception as e:
        logger.error(f"Ошибка проверки регистрации: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

async def get_all_guests_from_sheets() -> List[Dict]:
    """
    Получить список всех зарегистрированных гостей из Google Sheets
    
    Returns:
        Список словарей с информацией о гостях:
        [{'first_name': str, 'last_name': str, 'username': str, 'user_id': str, 'category': str, 'side': str}, ...]
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return []
    
    try:
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _get_all_guests_from_sheets_sync)
        return result
    except Exception as e:
        logger.error(f"Ошибка получения списка гостей: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []

def _get_all_guests_from_sheets_sync() -> List[Dict]:
    """Синхронная функция для получения списка всех гостей"""
    try:
        client = get_google_sheets_client()
        if not client:
            return []
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        
        # Получаем все данные
        all_values = worksheet.get_all_values()
        
        guests = []
        for row in all_values:
            if len(row) > 0 and row[0].strip():  # Проверяем, что есть имя
                full_name = row[0].strip()
                confirmation = row[2].strip() if len(row) > 2 and row[2] else ""  # Столбец C
                
                # Берем только тех, кто подтвердил участие (столбец C = "ДА")
                if confirmation.upper() == "ДА":
                    # Парсим имя и фамилию
                    name_parts = full_name.split(maxsplit=1)
                    first_name = name_parts[0] if name_parts else ""
                    last_name = name_parts[1] if len(name_parts) > 1 else ""
                    
                    # Получаем дополнительные данные
                    category = row[3].strip() if len(row) > 3 and row[3] else ""
                    side = row[4].strip() if len(row) > 4 and row[4] else ""
                    user_id = row[5].strip() if len(row) > 5 and row[5] else ""
                    
                    guests.append({
                        'first_name': first_name,
                        'last_name': last_name,
                        'username': "",  # Username не хранится в таблице, можно добавить столбец G если нужно
                        'user_id': user_id,
                        'category': category,
                        'side': side
                    })
        
        return guests
    except Exception as e:
        logger.error(f"Ошибка получения списка гостей: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []

async def get_guests_count_from_sheets() -> int:
    """
    Получить количество зарегистрированных гостей из Google Sheets
    
    Returns:
        Количество гостей со статусом "ДА" в столбце C
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return 0
    
    try:
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _get_guests_count_from_sheets_sync)
        return result
    except Exception as e:
        logger.error(f"Ошибка получения количества гостей: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return 0

def _get_guests_count_from_sheets_sync() -> int:
    """Синхронная функция для получения количества гостей"""
    try:
        client = get_google_sheets_client()
        if not client:
            return 0
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        
        # Получаем все данные
        all_values = worksheet.get_all_values()
        
        count = 0
        for row in all_values:
            if len(row) > 2 and row[2]:  # Проверяем столбец C (подтверждение)
                confirmation = row[2].strip().upper()
                if confirmation == "ДА":
                    count += 1
        
        return count
    except Exception as e:
        logger.error(f"Ошибка получения количества гостей: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return 0

async def cancel_guest_registration_by_user_id(user_id: int) -> bool:
    """
    Отменить регистрацию гостя по user_id (установить столбец C в "НЕТ")
    
    Args:
        user_id: Telegram user_id
        
    Returns:
        True если операция успешна
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return False
    
    try:
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _cancel_guest_registration_by_user_id_sync, user_id)
        return result
    except Exception as e:
        logger.error(f"Ошибка отмены регистрации гостя: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

def _cancel_guest_registration_by_user_id_sync(user_id: int) -> bool:
    """Синхронная функция для отмены регистрации гостя по user_id"""
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        
        # Получаем все данные
        all_values = worksheet.get_all_values()
        
        # Ищем строку с user_id в столбце F (индекс 5)
        found_row = None
        for row_idx, row in enumerate(all_values, start=1):
            if len(row) > 5:  # Проверяем, что есть столбец F
                user_id_cell = row[5].strip() if row[5] else ""
                if user_id_cell == str(user_id):
                    found_row = row_idx
                    break
        
        if found_row:
            # Обновляем столбец C на "НЕТ"
            worksheet.update_cell(found_row, 3, "НЕТ")  # Столбец C - Подтверждение
            logger.info(f"Регистрация гостя с user_id {user_id} отменена (строка {found_row})")
            return True
        else:
            logger.warning(f"Гость с user_id {user_id} не найден в Google Sheets")
            return False
    except Exception as e:
        logger.error(f"Ошибка отмены регистрации: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

async def delete_guest_from_sheets(user_id: int) -> bool:
    """
    Полностью удалить гостя из Google Sheets по user_id (удалить строку)
    
    Args:
        user_id: Telegram user_id
        
    Returns:
        True если операция успешна
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return False
    
    try:
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _delete_guest_from_sheets_sync, user_id)
        return result
    except Exception as e:
        logger.error(f"Ошибка удаления гостя: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

def _delete_guest_from_sheets_sync(user_id: int) -> bool:
    """Синхронная функция для полного удаления гостя из Google Sheets"""
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        
        # Получаем все данные
        all_values = worksheet.get_all_values()
        
        # Ищем строку с user_id в столбце F (индекс 5)
        found_row = None
        for row_idx, row in enumerate(all_values, start=1):
            if len(row) > 5:  # Проверяем, что есть столбец F
                user_id_cell = row[5].strip() if row[5] else ""
                if user_id_cell == str(user_id):
                    found_row = row_idx
                    break
        
        if found_row:
            # Удаляем строку
            worksheet.delete_rows(found_row)
            logger.info(f"Гость с user_id {user_id} удален из Google Sheets (строка {found_row})")
            return True
        else:
            logger.warning(f"Гость с user_id {user_id} не найден в Google Sheets")
            return False
    except Exception as e:
        logger.error(f"Ошибка удаления гостя: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

async def find_guest_by_name(first_name: str, last_name: str) -> Optional[Dict]:
    """
    Найти гостя по имени и фамилии (без user_id)
    
    Args:
        first_name: Имя
        last_name: Фамилия
        
    Returns:
        Словарь с информацией о госте или None если не найден
        {'first_name': str, 'last_name': str, 'row': int, 'user_id': str}
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return None
    
    try:
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _find_guest_by_name_sync, first_name, last_name)
        return result
    except Exception as e:
        logger.error(f"Ошибка поиска гостя по имени: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return None

def _find_guest_by_name_sync(first_name: str, last_name: str) -> Optional[Dict]:
    """Синхронная функция для поиска гостя по имени.
    
    Учитывает возможную перестановку имени и фамилии местами.
    """
    try:
        client = get_google_sheets_client()
        if not client:
            return None
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        
        all_values = worksheet.get_all_values()
        
        # Ищем по имени и фамилии (столбец A)
        full_name = f"{first_name} {last_name}".strip()
        alt_full_name = f"{last_name} {first_name}".strip()
        full_name_lower = full_name.lower()
        alt_full_name_lower = alt_full_name.lower()
        
        for row_idx, row in enumerate(all_values, start=1):
            if len(row) > 0:
                existing_name = row[0].strip() if row[0] else ""
                confirmation = row[2].strip() if len(row) > 2 and row[2] else ""  # Столбец C
                
                # Проверяем совпадение имени (в обоих порядках, без учета регистра) и подтверждение
                existing_lower = existing_name.lower()
                if existing_lower in (full_name_lower, alt_full_name_lower) and confirmation.upper() == "ДА":
                    user_id = row[5].strip() if len(row) > 5 and row[5] else ""  # Столбец F (индекс 5)
                    
                    # Парсим имя и фамилию из существующей строки
                    name_parts = existing_name.split(maxsplit=1)
                    found_first_name = name_parts[0] if name_parts else first_name
                    found_last_name = name_parts[1] if len(name_parts) > 1 else last_name
                    
                    logger.info(f"Найден гость по имени: {existing_name} (строка {row_idx}, user_id: {user_id})")
                    return {
                        'first_name': found_first_name,
                        'last_name': found_last_name,
                        'row': row_idx,
                        'user_id': user_id
                    }
        
        logger.info(f"Гость {full_name} / {alt_full_name} не найден по имени")
        return None
    except Exception as e:
        logger.error(f"Ошибка поиска гостя по имени: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return None


async def find_duplicate_guests() -> Dict[str, List[Dict]]:
    """
    Поиск возможных дубликатов гостей:
    - по совпадающему user_id (столбец F)
    - по совпадающим имени/фамилии, с учетом возможной перестановки местами.
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен, поиск дубликатов невозможен")
        return {"by_user_id": [], "by_name": []}
    
    try:
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _find_duplicate_guests_sync)
        return result
    except Exception as e:
        logger.error(f"Ошибка поиска дубликатов гостей: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return {"by_user_id": [], "by_name": []}


def _find_duplicate_guests_sync() -> Dict[str, List[Dict]]:
    """Синхронная функция для поиска дубликатов гостей в основной таблице."""
    try:
        client = get_google_sheets_client()
        if not client:
            return {"by_user_id": [], "by_name": []}
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        
        all_values = worksheet.get_all_values()
        
        by_user_id: Dict[str, List[Dict]] = {}
        by_name_key: Dict[str, List[Dict]] = {}
        
        for row_idx, row in enumerate(all_values, start=1):
            if not row or len(row) == 0:
                continue
            
            full_name = row[0].strip() if row[0] else ""
            if not full_name:
                continue
            
            # Столбец C — подтверждение "ДА"/"НЕТ"
            confirmation = row[2].strip().upper() if len(row) > 2 and row[2] else ""
            if confirmation != "ДА":
                # Интересуют только подтверждённые гости
                continue
            
            user_id = row[5].strip() if len(row) > 5 and row[5] else ""
            
            # Разбираем имя и фамилию для нормализации (учитываем возможную перестановку)
            parts = full_name.split()
            first = parts[0] if parts else ""
            last = " ".join(parts[1:]) if len(parts) > 1 else ""
            
            if first and last:
                key_parts = sorted([first.lower(), last.lower()])
                name_key = "|".join(key_parts)
            else:
                name_key = full_name.lower()
            
            info = {
                "row": row_idx,
                "full_name": full_name,
                "user_id": user_id,
            }
            
            if user_id:
                by_user_id.setdefault(user_id, []).append(info)
            by_name_key.setdefault(name_key, []).append(info)
        
        # Дубликаты по user_id.
        # Новая логика: один и тот же user_id может принадлежать основному гостю
        # и его дополнительным гостям БЕЗ собственного Telegram — это нормально.
        # Проблемой считаем только случаи, когда у ОДНОГО user_id
        # несколько строк с ОДНИМ И ТЕМ ЖЕ именем/фамилией (с учётом перестановки).
        dup_by_user_id = []
        for uid, rows in by_user_id.items():
            if len(rows) <= 1:
                continue

            # Группируем строки по нормализованному имени (учёт перестановки Имя/Фамилия)
            name_groups: dict[str, list[dict]] = {}
            for info in rows:
                full_name = (info.get("full_name") or "").strip()
                parts = full_name.split()
                first = parts[0] if parts else ""
                last = " ".join(parts[1:]) if len(parts) > 1 else ""
                if first and last:
                    key_parts = sorted([first.lower(), last.lower()])
                    name_key = "|".join(key_parts)
                else:
                    name_key = full_name.lower()
                name_groups.setdefault(name_key, []).append(info)

            # Собираем только те строки, где для одного user_id
            # одно и то же нормализованное имя встречается более одного раза
            problem_rows: list[dict] = []
            for group_rows in name_groups.values():
                if len(group_rows) > 1:
                    problem_rows.extend(group_rows)

            if problem_rows:
                dup_by_user_id.append({"user_id": uid, "rows": problem_rows})
        
        # Дубликаты по имени (независимо от user_id) оставляем как есть
        dup_by_name = [
            rows for key, rows in by_name_key.items() if len(rows) > 1
        ]
        
        if dup_by_user_id or dup_by_name:
            logger.warning(
                f"Найдены возможные дубликаты гостей: "
                f"{len(dup_by_user_id)} по user_id, {len(dup_by_name)} по имени"
            )
        else:
            logger.info("Дубликаты гостей не обнаружены")
        
        return {"by_user_id": dup_by_user_id, "by_name": dup_by_name}
    except Exception as e:
        logger.error(f"Ошибка при поиске дубликатов гостей: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return {"by_user_id": [], "by_name": []}

async def update_guest_user_id(row: int, user_id: int) -> bool:
    """
    Обновить user_id для существующего гостя
    
    Args:
        row: Номер строки в Google Sheets
        user_id: Telegram user_id
        
    Returns:
        True если успешно обновлено
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return False
    
    try:
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, _update_guest_user_id_sync, row, user_id)
        return result
    except Exception as e:
        logger.error(f"Ошибка обновления user_id гостя: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False


# ========== PING / СВЯЗЬ С GOOGLE SHEETS ДЛЯ ДИАГНОСТИКИ ==========


def _ping_admin_sheet_sync() -> int:
    """
    Небольшой "ping" к листу "Админ бота".

    Делает одно лёгкое чтение ячейки и возвращает примерную задержку в мс.
    Возвращает -1 в случае ошибки.
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен, ping невозможен")
        return -1

    try:
        client = get_google_sheets_client()
        if not client:
            return -1

        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_ADMINS_SHEET_NAME)

        start = time.monotonic()
        # Лёгкое чтение одной ячейки
        _ = worksheet.acell("A1").value
        latency_ms = int((time.monotonic() - start) * 1000)

        logger.info(f"Ping Google Sheets (Админ бота): {latency_ms} ms")
        return latency_ms
    except Exception as e:
        logger.error(f"Ошибка при ping листа 'Админ бота': {e}")
        import traceback
        logger.error(traceback.format_exc())
        return -1


async def ping_admin_sheet() -> int:
    """
    Асинхронный ping к листу "Админ бота".

    Returns:
        Задержка в мс (int) или -1 при ошибке.
    """
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _ping_admin_sheet_sync)


def _write_ping_to_admin_sheet_sync(source: str, latency_ms: int, status: str) -> bool:
    """
    Записать информацию о последнем ping в лист "Админ бота", строка 5.

    Формат:
        A5: timestamp (YYYY-MM-DD HH:MM:SS)
        B5: source ("bot" / "sheets" / др.)
        C5: latency (мс)
        D5: status ("OK" / "ERROR: ...")
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен, запись ping невозможна")
        return False

    try:
        client = get_google_sheets_client()
        if not client:
            return False

        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_ADMINS_SHEET_NAME)

        row = 5
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        values = [[timestamp, source, str(latency_ms), status]]
        worksheet.update(f"A{row}:D{row}", values)

        logger.info(
            f"Записан ping в 'Админ бота': time={timestamp}, source={source}, "
            f"latency_ms={latency_ms}, status={status}"
        )
        return True
    except Exception as e:
        logger.error(f"Ошибка при записи ping в 'Админ бота': {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False


async def write_ping_to_admin_sheet(source: str, latency_ms: int, status: str) -> bool:
    """
    Асинхронная обёртка для записи ping в лист "Админ бота".
    """
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(
        None, _write_ping_to_admin_sheet_sync, source, latency_ms, status
    )

def _update_guest_user_id_sync(row: int, user_id: int) -> bool:
    """Синхронная функция для обновления user_id"""
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        
        # Обновляем столбец F (индекс 6, так как update_cell использует 1-based индексы) - user_id
        worksheet.update_cell(row, 6, str(user_id))
        
        logger.info(f"Обновлен user_id для строки {row}: {user_id}")
        return True
    except Exception as e:
        logger.error(f"Ошибка обновления user_id: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False


# ========== СЕРВИСНЫЕ ФУНКЦИИ ДЛЯ АДМИНА (СПИСОК ГОСТЕЙ И ПЕРЕСТАНОВКА ИМЯ/ФАМИЛИЯ) ==========

def _list_confirmed_guests_sync() -> List[Dict]:
    """
    Синхронная функция: получить список всех подтверждённых гостей.

    Возвращает список словарей:
    [{'row': int, 'full_name': str}, ...]
    """
    try:
        client = get_google_sheets_client()
        if not client:
            return []

        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)

        all_values = worksheet.get_all_values()

        guests: List[Dict] = []
        for row_idx, row in enumerate(all_values, start=1):
            if not row or len(row) == 0:
                continue

            full_name = row[0].strip() if row[0] else ""
            if not full_name:
                continue

            # Столбец C — подтверждение "ДА"/"НЕТ"
            confirmation = row[2].strip().upper() if len(row) > 2 and row[2] else ""
            if confirmation != "ДА":
                continue

            guests.append({
                "row": row_idx,
                "full_name": full_name,
            })

        return guests
    except Exception as e:
        logger.error(f"Ошибка при получении списка подтверждённых гостей: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []


def _swap_guest_name_order_sync(row_index: int) -> Tuple[str, str]:
    """
    Синхронная функция: поменять местами Имя и Фамилию в первой колонке для заданной строки.

    Возвращает кортеж (old_full_name, new_full_name).
    Если изменить не удалось, возвращает ("", "").
    """
    try:
        client = get_google_sheets_client()
        if not client:
            return "", ""

        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)

        cell = worksheet.cell(row_index, 1)
        old_full_name = (cell.value or "").strip()
        if not old_full_name:
            return "", ""

        parts = old_full_name.split()
        if len(parts) < 2:
            # Одно слово — нечего переставлять
            return old_full_name, old_full_name

        first = parts[0]
        last = " ".join(parts[1:])

        new_full_name = f"{last} {first}".strip()
        worksheet.update_cell(row_index, 1, new_full_name)

        logger.info(
            f"Переставлены Имя/Фамилия в строке {row_index}: "
            f"'{old_full_name}' -> '{new_full_name}'"
        )
        return old_full_name, new_full_name
    except Exception as e:
        logger.error(f"Ошибка при перестановке Имя/Фамилия в строке {row_index}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return "", ""


async def list_confirmed_guests() -> List[Dict]:
    """
    Асинхронная обёртка для получения списка подтверждённых гостей.
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен, список подтверждённых гостей недоступен")
        return []

    try:
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(None, _list_confirmed_guests_sync)
    except Exception as e:
        logger.error(f"Ошибка (async) при получении списка подтверждённых гостей: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []


async def swap_guest_name_order(row_index: int) -> Tuple[str, str]:
    """
    Асинхронная обёртка для перестановки Имя/Фамилия для одной строки.

    Возвращает (old_full_name, new_full_name).
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен, перестановка Имя/Фамилия невозможна")
        return "", ""

    try:
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(None, _swap_guest_name_order_sync, row_index)
    except Exception as e:
        logger.error(f"Ошибка (async) при перестановке Имя/Фамилия: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return "", ""
