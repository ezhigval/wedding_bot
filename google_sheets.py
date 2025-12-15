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

from config import (
    GOOGLE_SHEETS_ID,
    GOOGLE_SHEETS_CREDENTIALS,
    GOOGLE_SHEETS_SHEET_NAME,
    GOOGLE_SHEETS_INVITATIONS_SHEET_NAME,
    GOOGLE_SHEETS_ADMINS_SHEET_NAME,
    GOOGLE_SHEETS_TIMELINE_SHEET_NAME,
)

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
        # Столбец A - username, столбец B - user_id,
        # столбец D - api_id, столбец E - api_hash, столбец F - phone,
        # столбец G - login_code (используем для авторизации Telegram Client)
        for row_idx, row in enumerate(all_values[start_row:], start=start_row + 1):
            if len(row) >= 1:
                username = row[0].strip() if row[0] else ""
                user_id = row[1].strip() if len(row) >= 2 and row[1] else None
                api_id = row[3].strip() if len(row) >= 4 and row[3] else None  # Столбец D
                api_hash = row[4].strip() if len(row) >= 5 and row[4] else None  # Столбец E
                phone = row[5].strip() if len(row) >= 6 and row[5] else None  # Столбец F
                login_code = row[6].strip() if len(row) >= 7 and row[6] else None  # Столбец G

                # Пропускаем пустые строки
                if not username:
                    continue

                # Убираем @ если есть
                username_clean = username.replace('@', '').lower()

                admin_data = {
                    'username': username_clean,
                    'name': username_clean,
                    'telegram': username_clean,
                    'row_index': row_idx,
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
                if login_code:
                    admin_data['login_code'] = login_code

                admins.append(admin_data)
        
        logger.info(f"Получено {len(admins)} админов из Google Sheets")
        return admins
        
    except Exception as e:
        logger.error(f"Ошибка получения списка админов из Google Sheets: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []


def _get_admin_login_code_and_clear_sync(admin_user_id: int) -> str:
    """
    Считает одноразовый код авторизации из вкладки "Админ бота" и сразу очищает ячейку.

    Логика:
    - ищем строку, где столбец B (user_id) совпадает с admin_user_id;
    - берём значение из столбца G (login_code);
    - если код есть — очищаем ячейку G и возвращаем код.
    """
    try:
        client = get_google_sheets_client()
        if not client:
            return ""

        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        try:
            worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_ADMINS_SHEET_NAME)
        except Exception as e:
            logger.error(f"Вкладка '{GOOGLE_SHEETS_ADMINS_SHEET_NAME}' не найдена: {e}")
            return ""

        all_values = worksheet.get_all_values()
        if not all_values:
            return ""

        start_row = 0
        if all_values:
            first_row = all_values[0]
            if any(keyword in str(first_row[0]).lower() for keyword in ['username', 'админ', 'admin']):
                start_row = 1

        for row_idx, row in enumerate(all_values[start_row:], start=start_row + 1):
            if len(row) < 2:
                continue

            user_id_cell = row[1].strip() if row[1] else ""
            try:
                user_id_value = int(user_id_cell) if user_id_cell else None
            except ValueError:
                user_id_value = None

            if user_id_value != admin_user_id:
                continue

            # Нашли строку текущего админа
            code = row[6].strip() if len(row) >= 7 and row[6] else ""
            if not code:
                logger.info(f"В столбце G для админа {admin_user_id} код не найден (строка {row_idx})")
                return ""

            # Очищаем ячейку с кодом (столбец G = 7)
            worksheet.update_cell(row_idx, 7, "")
            logger.info(f"Считан и очищен код авторизации для админа {admin_user_id} из строки {row_idx}")
            return code

        logger.info(f"Строка для админа {admin_user_id} в листе 'Админ бота' не найдена")
        return ""

    except Exception as e:
        logger.error(f"Ошибка при чтении кода авторизации из 'Админ бота': {e}")
        import traceback
        logger.error(traceback.format_exc())
        return ""


async def get_admin_login_code_and_clear(admin_user_id: int) -> str:
    """
    Асинхронная обёртка для _get_admin_login_code_and_clear_sync.
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен, не можем считать код авторизации")
        return ""

    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _get_admin_login_code_and_clear_sync, admin_user_id)

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
        
        # Получаем данные ТОЛЬКО из первых двух столбцов (A: время, B: событие),
        # чтобы любые дополнительные столбцы (C, D, ...) не ломали план-сетку.
        values = worksheet.get("A:B") or []
        
        timeline: List[Dict[str, str]] = []
        for raw_row in values:
            # Гарантируем, что у нас всегда есть как минимум 2 ячейки
            row = (raw_row + ["", ""])[:2]
            time_cell = (row[0] or "").strip()
            event_cell = (row[1] or "").strip()

            # Пропускаем полностью пустые строки и строки без события
            if not event_cell:
                continue

            timeline.append(
                {
                    "time": time_cell,
                    "event": event_cell,
                }
            )
        
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
        missing_ids: List[Dict] = []      # подтверждён, но без user_id
        username_ids: List[Dict] = []     # в столбце F лежит @username / t.me/...
        
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
            
            raw_user_id = row[5].strip() if len(row) > 5 and row[5] else ""
            numeric_user_id = raw_user_id if raw_user_id.isdigit() else ""
            
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
                "user_id": raw_user_id,
            }

            # Кейс 1: подтверждённый гость без user_id вообще
            if not raw_user_id:
                missing_ids.append(info)
            # Кейс 2: в ячейке что-то есть, но это не цифры — пытаемся распознать username
            elif not numeric_user_id:
                # Пробуем вытащить чистый username из разных форматов
                candidate = raw_user_id.strip()
                candidate = candidate.replace("https://t.me/", "").replace("http://t.me/", "")
                candidate = candidate.replace("t.me/", "")
                if candidate.startswith("@"):
                    candidate = candidate[1:]
                candidate = candidate.split()[0].strip()
                if candidate:
                    username_ids.append(
                        {
                            "row": row_idx,
                            "full_name": full_name,
                            "raw_value": raw_user_id,
                            "username": candidate,
                        }
                    )

            # Для поиска дубликатов по user_id учитываем только числовые ID
            if numeric_user_id:
                by_user_id.setdefault(numeric_user_id, []).append(info)
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
        dup_by_name = [rows for key, rows in by_name_key.items() if len(rows) > 1]
        
        if dup_by_user_id or dup_by_name or missing_ids or username_ids:
            logger.warning(
                "Найдены возможные проблемы в списке гостей: "
                f"{len(dup_by_user_id)} по user_id, "
                f"{len(dup_by_name)} по имени, "
                f"{len(missing_ids)} без user_id, "
                f"{len(username_ids)} с username вместо user_id"
            )
        else:
            logger.info("Дубликаты гостей не обнаружены")
        
        return {
            "by_user_id": dup_by_user_id,
            "by_name": dup_by_name,
            "missing_ids": missing_ids,
            "username_ids": username_ids,
        }
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


# ========== КОНФИГ / ФЛАГИ ==========


def _get_or_create_config_sheet_sync():
    """
    Получить (или создать) лист 'Config'.

    Используется для хранения флагов:
      - SEATING_LOCKED (true/false)
      - SEATING_LOCKED_AT (timestamp)
    """
    client = get_google_sheets_client()
    if not client:
        raise RuntimeError("Google Sheets клиент недоступен")

    spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
    try:
        sheet = spreadsheet.worksheet("Config")
    except Exception:
        sheet = spreadsheet.add_worksheet(title="Config", rows=20, cols=4)
    return sheet


def _get_seating_lock_status_sync() -> Dict:
    """
    Прочитать статус закрепления рассадки из листа Config.
    """
    try:
        sheet = _get_or_create_config_sheet_sync()
        values = sheet.get_all_values()

        data: Dict[str, str] = {}
        for row in values:
            if len(row) < 2:
                continue
            key = (row[0] or "").strip()
            value = (row[1] or "").strip()
            if key:
                data[key] = value

        locked = data.get("SEATING_LOCKED", "").lower() in ("1", "true", "yes")
        locked_at = data.get("SEATING_LOCKED_AT") or ""

        return {"locked": locked, "locked_at": locked_at}
    except Exception as e:
        logger.error(f"Ошибка при чтении статуса закрепления рассадки: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return {"locked": False, "locked_at": ""}


async def get_seating_lock_status() -> Dict:
    """
    Асинхронная обёртка для чтения статуса закрепления рассадки.
    """
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _get_seating_lock_status_sync)


def _lock_seating_sync() -> Dict:
    """
    Закрепить текущую рассадку:
      - Создать/обновить лист 'Рассадка_фикс' (копия текущей 'Рассадка')
      - Записать флаги SEATING_LOCKED и SEATING_LOCKED_AT в лист Config

    Возвращает словарь с информацией о статусе:
      {'locked': True/False, 'locked_at': '...', 'reason': str}
    """
    try:
        client = get_google_sheets_client()
        if not client:
            return {"locked": False, "locked_at": "", "reason": "no_client"}

        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)

        # Проверяем текущий статус
        status = _get_seating_lock_status_sync()
        if status.get("locked"):
            return {
                "locked": True,
                "locked_at": status.get("locked_at", ""),
                "reason": "already_locked",
            }

        # Читаем текущую рассадку
        try:
            seating_sheet = spreadsheet.worksheet("Рассадка")
        except Exception as e:
            logger.error(f"Лист 'Рассадка' не найден при закреплении: {e}")
            return {"locked": False, "locked_at": "", "reason": "no_seating_sheet"}

        values = seating_sheet.get_all_values()

        # Создаём/очищаем лист 'Рассадка_фикс'
        try:
            fixed_sheet = spreadsheet.worksheet("Рассадка_фикс")
            fixed_sheet.clear()
        except Exception:
            fixed_sheet = spreadsheet.add_worksheet(title="Рассадка_фикс", rows=100, cols=26)

        if values:
            rows = len(values)
            cols = max(len(row) for row in values)
            # Нормализуем размеры
            norm_values = []
            for row in values:
                r = list(row)
                if len(r) < cols:
                    r.extend([""] * (cols - len(r)))
                norm_values.append(r)

            end_col_letter = chr(ord("A") + cols - 1)
            fixed_sheet.update(f"A1:{end_col_letter}{rows}", norm_values)

        # Обновляем Config
        config_sheet = _get_or_create_config_sheet_sync()
        now_str = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        config_sheet.update("A1:B2", [
            ["SEATING_LOCKED", "1"],
            ["SEATING_LOCKED_AT", now_str],
        ])

        logger.info(f"Рассадка закреплена в листе 'Рассадка_фикс' в {now_str}")
        return {"locked": True, "locked_at": now_str, "reason": "ok"}
    except Exception as e:
        logger.error(f"Ошибка при закреплении рассадки: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return {"locked": False, "locked_at": "", "reason": "exception"}


async def lock_seating() -> Dict:
    """
    Асинхронная обёртка для закрепления рассадки.
    """
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _lock_seating_sync)


# ========== РАССАДКА: ЧТЕНИЕ С ЛИСТА «РАССАДКА» / «РАССАДКА_ФИКС» ==========


def _get_seating_from_sheets_sync() -> List[Dict]:
    """
    Получить текущую рассадку из листа "Рассадка".

    Формат:
        [
          {
            "table": "Стол №1",
            "guests": ["Фамилия Имя", ...]
          },
          ...
        ]
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен, чтение рассадки невозможно")
        return []

    try:
        client = get_google_sheets_client()
        if not client:
            return []

        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        try:
            seating = spreadsheet.worksheet("Рассадка")
        except Exception as e:
            logger.error(f"Лист 'Рассадка' не найден: {e}")
            return []

        values = seating.get_all_values()
        if not values:
            return []

        header_row = values[0]
        cols = len(header_row)
        if cols < 2:
            # Нет ни одного столика
            return []

        # Заголовки столов начинаются с колонки B (индекс 1)
        tables: List[Dict] = []
        for idx in range(1, cols):
            table_name = (header_row[idx] or "").strip()
            if not table_name:
                continue

            guests: List[str] = []
            # Строки 2..N (индексы 1..len(values)-1)
            for r in range(1, len(values)):
                row = values[r]
                if idx >= len(row):
                    continue
                guest_name = (row[idx] or "").strip()
                if not guest_name:
                    continue
                guests.append(guest_name)

            tables.append({"table": table_name, "guests": guests})

        logger.info(
            f"Прочитана рассадка: {len(tables)} столов "
            f"({sum(len(t['guests']) for t in tables)} гостей)"
        )
        return tables
    except Exception as e:
        logger.error(f"Ошибка при чтении рассадки: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []


async def get_seating_from_sheets() -> List[Dict]:
    """
    Асинхронная обёртка для получения рассадки с листа "Рассадка".
    """
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _get_seating_from_sheets_sync)


def _get_guest_table_and_neighbors_sync(user_id: int) -> Optional[Dict]:
    """
    Найти для гостя по user_id:
      - зафиксированный стол (из 'Рассадка_фикс')
      - список соседей (другие имена в том же столбце)
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен, поиск стола невозможен")
        return None

    try:
        client = get_google_sheets_client()
        if not client:
            return None

        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)

        # 1. Находим полное имя гостя по user_id на листе "Список гостей"
        try:
            guest_sheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        except Exception as e:
            logger.error(f"Лист гостей '{GOOGLE_SHEETS_SHEET_NAME}' не найден: {e}")
            return None

        guest_values = guest_sheet.get_all_values()
        if not guest_values:
            return None

        full_name = ""
        for row in guest_values[1:]:
            if len(row) <= 5:
                continue
            uid_cell = (row[5] or "").strip()
            if uid_cell == str(user_id):
                full_name = (row[0] or "").strip()
                break

        if not full_name:
            logger.info(f"Гость с user_id={user_id} не найден в 'Список гостей'")
            return None

        # 2. Ищем это имя в зафиксированной рассадке ('Рассадка_фикс')
        try:
            seating_sheet = spreadsheet.worksheet("Рассадка_фикс")
        except Exception as e:
            logger.error(f"Лист 'Рассадка_фикс' не найден: {e}")
            return None

        values = seating_sheet.get_all_values()
        if not values or len(values) < 2:
            return None

        header_row = values[0]
        cols = len(header_row)
        target_table = None
        neighbors: List[str] = []

        for col_idx in range(1, cols):
            table_name = (header_row[col_idx] or "").strip()
            if not table_name:
                continue

            column_names: List[str] = []
            for r in range(1, len(values)):
                row = values[r]
                if col_idx >= len(row):
                    continue
                name = (row[col_idx] or "").strip()
                if not name:
                    continue
                column_names.append(name)

            # ищем полное совпадение имени в этом столе
            if any(n == full_name for n in column_names):
                target_table = table_name
                neighbors = [n for n in column_names if n != full_name]
                break

        if not target_table:
            logger.info(
                f"Гость '{full_name}' (user_id={user_id}) не найден "
                f"в зафиксированной рассадке"
            )
            return None

        return {
            "full_name": full_name,
            "table": target_table,
            "neighbors": neighbors,
        }
    except Exception as e:
        logger.error(f"Ошибка при поиске стола и соседей для user_id={user_id}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return None


async def get_guest_table_and_neighbors(user_id: int) -> Optional[Dict]:
    """
    Асинхронная обёртка для get_guest_table_and_neighbors.
    """
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _get_guest_table_and_neighbors_sync, user_id)


# ========== ФОТОГОСТИ: СОХРАНЕНИЕ МЕТАДАННЫХ ФОТО В ОТДЕЛЬНУЮ ВКЛАДКУ ==========


def _save_photo_from_user_sync(
    user_id: int,
    username: Optional[str],
    full_name: str,
    file_id: str,
) -> bool:
    """
    Сохранить информацию о фото, присланном гостем, в лист 'Фото'.

    Формат строки:
      A: timestamp
      B: user_id
      C: username
      D: full_name
      E: file_id (Telegram)
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен, сохранение фото невозможно")
        return False

    try:
        client = get_google_sheets_client()
        if not client:
            return False

        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        try:
            sheet = spreadsheet.worksheet("Фото")
        except Exception:
            sheet = spreadsheet.add_worksheet(title="Фото", rows=100, cols=5)

        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        row = [
            timestamp,
            str(user_id),
            (username or "").lstrip("@"),
            full_name,
            file_id,
        ]
        sheet.append_row(row)
        logger.info(
            f"Сохранено фото от user_id={user_id}, username={username}, file_id={file_id}"
        )
        return True
    except Exception as e:
        logger.error(f"Ошибка при сохранении фото пользователя {user_id}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False


async def save_photo_from_user(
    user_id: int,
    username: Optional[str],
    full_name: str,
    file_id: str,
) -> bool:
    """
    Асинхронная обёртка для сохранения фото гостя.
    """
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(
        None, _save_photo_from_user_sync, user_id, username, full_name, file_id
    )

def _save_photo_from_webapp_sync(
    user_id: int,
    username: Optional[str],
    full_name: str,
    photo_data: str,  # base64 или URL
) -> bool:
    """
    Сохранить информацию о фото из веб-приложения в лист 'Фото'.
    
    Формат строки:
      A: timestamp
      B: user_id
      C: username
      D: full_name
      E: photo_data (base64 или URL)
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен, сохранение фото невозможно")
        return False

    try:
        client = get_google_sheets_client()
        if not client:
            return False

        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        try:
            sheet = spreadsheet.worksheet("Фото")
        except Exception:
            sheet = spreadsheet.add_worksheet(title="Фото", rows=100, cols=5)

        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        row = [
            timestamp,
            str(user_id),
            (username or "").lstrip("@"),
            full_name,
            photo_data[:500] if len(photo_data) > 500 else photo_data,  # Ограничиваем длину
        ]
        sheet.append_row(row)
        logger.info(
            f"Сохранено фото из веб-приложения от user_id={user_id}, username={username}"
        )
        return True
    except Exception as e:
        logger.error(f"Ошибка при сохранении фото из веб-приложения {user_id}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False


async def save_photo_from_webapp(
    user_id: int,
    username: Optional[str],
    full_name: str,
    photo_data: str,
) -> bool:
    """
    Асинхронная обёртка для сохранения фото из веб-приложения.
    """
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(
        None, _save_photo_from_webapp_sync, user_id, username, full_name, photo_data
    )

# ========== ИГРЫ: СТАТИСТИКА И ОЧКИ ==========

def _get_rank_by_score(total_score: int) -> str:
    """
    Определяет звание игрока по общему счету.
    
    Звания:
    - 0-50: Незнакомец
    - 50-100: Ты хто?
    - 100-150: Люся
    - 150-200: Бедный родственник
    - 200-300: Братуха
    - 300-400: Батя в здании
    - 400-500: Монстр
    """
    if total_score < 50:
        return 'Незнакомец'
    elif total_score < 100:
        return 'Ты хто?'
    elif total_score < 150:
        return 'Люся'
    elif total_score < 200:
        return 'Бедный родственник'
    elif total_score < 300:
        return 'Братуха'
    elif total_score < 400:
        return 'Батя в здании'
    else:
        return 'Монстр'

def _get_guest_name_by_user_id(user_id: int) -> Tuple[str, str]:
    """
    Получить имя и фамилию гостя по user_id из основной таблицы гостей.
    Возвращает (first_name, last_name) или ("", "") если не найдено.
    """
    if not GSPREAD_AVAILABLE:
        return ("", "")
    
    try:
        client = get_google_sheets_client()
        if not client:
            return ("", "")
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        try:
            worksheet = spreadsheet.worksheet(GOOGLE_SHEETS_SHEET_NAME)
        except Exception:
            return ("", "")
        
        all_values = worksheet.get_all_values()
        for row in all_values:
            if len(row) > 5 and str(row[5]).strip() == str(user_id):
                # Нашли user_id в столбце F (индекс 5)
                full_name = row[0].strip() if row[0] else ""
                if full_name:
                    name_parts = full_name.split(maxsplit=1)
                    first_name = name_parts[0] if name_parts else ""
                    last_name = name_parts[1] if len(name_parts) > 1 else ""
                    return (first_name, last_name)
        
        return ("", "")
    except Exception as e:
        logger.error(f"Ошибка получения имени гостя {user_id}: {e}")
        return ("", "")


def _get_game_stats_sync(user_id: int) -> Optional[Dict]:
    """
    Получить статистику игрока по user_id из листа 'Игры'.
    
    Формат строки:
      A: user_id
      B: first_name
      C: last_name
      D: total_score (общий счет)
      E: dragon_score (счет в Дракончике)
      F: flappy_score (счет в ФлэппиБёрд)
      G: crossword_score (счет в Кроссводе)
      H: rank (звание)
      I: last_updated (дата последнего обновления в формате ISO)
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return None
    
    try:
        client = get_google_sheets_client()
        if not client:
            return None
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        # Проверяем и создаем необходимые вкладки
        _ensure_required_sheets_sync()
        
        try:
            sheet = spreadsheet.worksheet("Игры")
        except Exception:
            # Создаем лист если его нет
            sheet = spreadsheet.add_worksheet(title="Игры", rows=100, cols=9)
            # Добавляем заголовки
            sheet.append_row(["user_id", "first_name", "last_name", "total_score", "dragon_score", "flappy_score", "crossword_score", "wordle_score", "rank", "last_updated"])
            return None
        
        # Ищем строку с user_id
        all_values = sheet.get_all_values()
        for i, row in enumerate(all_values[1:], start=2):  # Пропускаем заголовок
            if row and len(row) > 0 and str(row[0]) == str(user_id):
                return {
                    'user_id': int(row[0]) if row[0] else user_id,
                    'first_name': row[1] if len(row) > 1 and row[1] else "",
                    'last_name': row[2] if len(row) > 2 and row[2] else "",
                    'total_score': int(row[3]) if len(row) > 3 and row[3] else 0,
                    'dragon_score': int(row[4]) if len(row) > 4 and row[4] else 0,
                    'flappy_score': int(row[5]) if len(row) > 5 and row[5] else 0,
                    'crossword_score': int(row[6]) if len(row) > 6 and row[6] else 0,
                    'wordle_score': int(row[7]) if len(row) > 7 and row[7] else 0,
                    'wordle_score': int(row[7]) if len(row) > 7 and row[7] else 0,
                    'rank': row[8] if len(row) > 8 and row[8] else 'Незнакомец',
                    'last_updated': row[9] if len(row) > 9 and row[9] else None,
                }
        
        return None
    except Exception as e:
        logger.error(f"Ошибка получения статистики игрока {user_id}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return None


def _update_game_score_sync(
    user_id: int,
    game_type: str,  # 'dragon', 'flappy', 'crossword'
    score: int,
) -> bool:
    """
    Обновить счет игрока в конкретной игре.
    Также обновляет общий счет и звание.
    Сохраняет имя и фамилию пользователя из таблицы гостей.
    
    Формат строки в листе "Игры":
      A: user_id
      B: first_name
      C: last_name
      D: total_score (общий счет)
      E: dragon_score (счет в Дракончике)
      F: flappy_score (счет в ФлэппиБёрд)
      G: crossword_score (счет в Кроссводе)
      H: rank (звание)
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return False
    
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        # Получаем имя и фамилию пользователя
        first_name, last_name = _get_guest_name_by_user_id(user_id)
        if not first_name and not last_name:
            logger.warning(f"Не удалось найти имя для user_id={user_id}, используем пустые значения")
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        
        # Проверяем и создаем необходимые вкладки
        _ensure_required_sheets_sync()
        
        try:
            sheet = spreadsheet.worksheet("Игры")
        except Exception:
            # Создаем лист если его нет
            sheet = spreadsheet.add_worksheet(title="Игры", rows=100, cols=9)
            # Добавляем заголовки
            sheet.append_row(["user_id", "first_name", "last_name", "total_score", "dragon_score", "flappy_score", "crossword_score", "wordle_score", "rank", "last_updated"])
        
        # Определяем индекс столбца для конкретной игры (0-based для списков, 1-based для gspread)
        # Теперь индексы сдвинуты из-за добавления first_name и last_name
        game_col_index = {
            'dragon': 4,    # E (индекс 4 в списке, столбец 5 в gspread)
            'flappy': 5,    # F (индекс 5 в списке, столбец 6 в gspread)
            'crossword': 6, # G (индекс 6 в списке, столбец 7 в gspread)
            'wordle': 7,    # H (индекс 7 в списке, столбец 8 в gspread)
        }
        
        if game_type not in game_col_index:
            logger.error(f"Неизвестный тип игры: {game_type}")
            return False
        
        game_col_list = game_col_index[game_type]  # Индекс в списке (0-based)
        game_col_gspread = game_col_list + 1  # Столбец в gspread (1-based)
        
        # Ищем строку с user_id
        all_values = sheet.get_all_values()
        user_row_gspread = None  # Номер строки для gspread (1-based)
        
        for i, row in enumerate(all_values[1:], start=2):  # Пропускаем заголовок, начинаем с 2
            if row and len(row) > 0 and str(row[0]) == str(user_id):
                user_row_gspread = i
                break
        
        if user_row_gspread:
            # Обновляем существующую строку
            row_data = all_values[user_row_gspread - 1]  # -1 потому что all_values 0-based
            current_score = int(row_data[game_col_list]) if len(row_data) > game_col_list and row_data[game_col_list] else 0
            
            # Получаем текущие счета всех игр
            dragon_score = int(row_data[4]) if len(row_data) > 4 and row_data[4] else 0
            flappy_score = int(row_data[5]) if len(row_data) > 5 and row_data[5] else 0
            crossword_score = int(row_data[6]) if len(row_data) > 6 and row_data[6] else 0
            
            # Прибавляем очки к счету игры (накопительно)
            # Конвертируем игровые очки в рейтинговые по формулам:
            # Dragon: 200 игровых очков = 1 рейтинговое очко
            # Flappy: 2 игровых очка = 1 рейтинговое очко
            # Crossword: 1 игровое очко = 25 рейтинговых очков
            
            if game_type == 'dragon':
                rating_points = score // 200
                dragon_score += rating_points
            elif game_type == 'flappy':
                rating_points = score // 2
                flappy_score += rating_points
            elif game_type == 'crossword':
                # Конвертируем игровые очки в рейтинговые: 1 игровое очко = 25 рейтинговых очков
                rating_points = score * 25
                crossword_score += rating_points
            elif game_type == 'wordle':
                # Wordle: каждое отгаданное слово = 5 рейтинговых очков
                # score здесь - количество отгаданных слов
                rating_points = score * 5
                wordle_score += rating_points
            
            # Обновляем счет игры
            if game_type == 'dragon':
                sheet.update_cell(user_row_gspread, game_col_gspread, dragon_score)
            elif game_type == 'flappy':
                sheet.update_cell(user_row_gspread, game_col_gspread, flappy_score)
            elif game_type == 'crossword':
                sheet.update_cell(user_row_gspread, game_col_gspread, crossword_score)
            elif game_type == 'wordle':
                sheet.update_cell(user_row_gspread, game_col_gspread, wordle_score)
            
            # Обновляем имя и фамилию (на случай если они изменились)
            sheet.update_cell(user_row_gspread, 2, first_name)  # B - first_name
            sheet.update_cell(user_row_gspread, 3, last_name)   # C - last_name
            
            # Пересчитываем общий счет
            total_score = dragon_score + flappy_score + crossword_score + wordle_score
            sheet.update_cell(user_row_gspread, 4, total_score)  # D - total_score (столбец 4 в gspread)
            
            # Обновляем все счета игр (на случай если нужно обновить другие столбцы)
            if game_type != 'dragon':
                sheet.update_cell(user_row_gspread, 5, dragon_score)  # E - dragon_score
            if game_type != 'flappy':
                sheet.update_cell(user_row_gspread, 6, flappy_score)  # F - flappy_score
            if game_type != 'crossword':
                sheet.update_cell(user_row_gspread, 7, crossword_score)  # G - crossword_score
            if game_type != 'wordle':
                sheet.update_cell(user_row_gspread, 8, wordle_score)  # H - wordle_score
            
            # Определяем звание
            rank = _get_rank_by_score(total_score)
            
            sheet.update_cell(user_row_gspread, 9, rank)  # I - rank (столбец 9 в gspread)
            
            # Обновляем дату последнего изменения
            last_updated = datetime.now().isoformat()
            sheet.update_cell(user_row_gspread, 10, last_updated)  # J - last_updated (столбец 10 в gspread)
        else:
            # Создаем новую строку
            # Конвертируем игровые очки в рейтинговые
            if game_type == 'dragon':
                rating_points = score // 200
                dragon_score = rating_points
            else:
                dragon_score = 0
            
            if game_type == 'flappy':
                rating_points = score // 2
                flappy_score = rating_points
            else:
                flappy_score = 0
            
            if game_type == 'crossword':
                # Конвертируем игровые очки в рейтинговые: 1 игровое очко = 25 рейтинговых очков
                crossword_score = score * 25
            else:
                crossword_score = 0
            
            if game_type == 'wordle':
                # Wordle: каждое отгаданное слово = 5 рейтинговых очков
                # score здесь - количество отгаданных слов
                wordle_score = score * 5
            else:
                wordle_score = 0
            
            total_score = dragon_score + flappy_score + crossword_score + wordle_score
            
            rank = _get_rank_by_score(total_score)
            
            last_updated = datetime.now().isoformat()
            row = [
                str(user_id),
                first_name,
                last_name,
                total_score,
                dragon_score,
                flappy_score,
                crossword_score,
                wordle_score,
                rank,
                last_updated,
            ]
            sheet.append_row(row)
        
        # Определяем звание для логирования (если еще не определено)
        if 'rank' not in locals():
            rank = _get_rank_by_score(total_score)
        
        logger.info(f"Обновлен счет для user_id={user_id} ({first_name} {last_name}), игра={game_type}, счет={score}, звание={rank}")
        return True
    except Exception as e:
        logger.error(f"Ошибка обновления счета игрока {user_id}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False


async def get_game_stats(user_id: int) -> Optional[Dict]:
    """Асинхронная обёртка для получения статистики игрока"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _get_game_stats_sync, user_id)


async def update_game_score(
    user_id: int,
    game_type: str,
    score: int,
) -> bool:
    """Асинхронная обёртка для обновления счета"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _update_game_score_sync, user_id, game_type, score)

# ========== WORDLE ==========

def _get_wordle_word_sync() -> Optional[str]:
    """
    Получает последнее (актуальное) слово из вкладки "Wordle".
    Последнее слово в таблице = актуальное слово для отгадывания.
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return None
    
    try:
        client = get_google_sheets_client()
        if not client:
            return None
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        _ensure_required_sheets_sync()
        
        try:
            sheet = spreadsheet.worksheet("Wordle")
        except Exception:
            logger.error("Вкладка 'Wordle' не найдена")
            return None
        
        # Получаем все значения из первого столбца (слова)
        all_values = sheet.get_all_values()
        
        if not all_values:
            return None
        
        # Пропускаем заголовок (первую строку)
        words = []
        for row in all_values[1:]:  # Пропускаем заголовок
            if row and len(row) > 0 and row[0].strip():
                words.append(row[0].strip().upper())
        
        # Возвращаем последнее слово (актуальное)
        if words:
            return words[-1]
        
        return None
    except Exception as e:
        logger.error(f"Ошибка получения слова Wordle: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return None


def _get_wordle_guessed_words_sync(user_id: int) -> List[str]:
    """
    Получает список отгаданных слов пользователя из вкладки "Wordle_Прогресс".
    """
    if not GSPREAD_AVAILABLE:
        return []
    
    try:
        client = get_google_sheets_client()
        if not client:
            return []
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        _ensure_required_sheets_sync()
        
        try:
            sheet = spreadsheet.worksheet("Wordle_Прогресс")
        except Exception:
            return []
        
        all_values = sheet.get_all_values()
        for row in all_values[1:]:  # Пропускаем заголовок
            if row and len(row) > 0 and str(row[0]).strip() == str(user_id):
                # Нашли пользователя, возвращаем отгаданные слова
                if len(row) > 1 and row[1]:
                    # Слова разделены запятыми
                    words_str = row[1].strip()
                    if words_str:
                        return [w.strip().upper() for w in words_str.split(',') if w.strip()]
                return []
        
        return []
    except Exception as e:
        logger.error(f"Ошибка получения прогресса Wordle для user_id={user_id}: {e}")
        return []


def _save_wordle_progress_sync(user_id: int, guessed_words: List[str]) -> bool:
    """
    Сохраняет прогресс пользователя в Wordle (отгаданные слова).
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return False
    
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        _ensure_required_sheets_sync()
        
        try:
            sheet = spreadsheet.worksheet("Wordle_Прогресс")
        except Exception:
            logger.error("Вкладка 'Wordle_Прогресс' не найдена")
            return False
        
        all_values = sheet.get_all_values()
        user_row = None
        user_row_index = None
        
        # Ищем строку пользователя
        for i, row in enumerate(all_values[1:], start=2):  # Пропускаем заголовок, начинаем с 2
            if row and len(row) > 0 and str(row[0]).strip() == str(user_id):
                user_row = row
                user_row_index = i
                break
        
        # Сохраняем слова через запятую
        words_str = ','.join(guessed_words)
        
        if user_row_index:
            # Обновляем существующую строку
            sheet.update_cell(user_row_index, 2, words_str)  # Столбец B
        else:
            # Создаем новую строку
            sheet.append_row([str(user_id), words_str])
        
        return True
    except Exception as e:
        logger.error(f"Ошибка сохранения прогресса Wordle для user_id={user_id}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False


async def get_wordle_word() -> Optional[str]:
    """Асинхронная обёртка для получения актуального слова Wordle"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _get_wordle_word_sync)


async def get_wordle_guessed_words(user_id: int) -> List[str]:
    """Асинхронная обёртка для получения отгаданных слов пользователя"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _get_wordle_guessed_words_sync, user_id)


async def save_wordle_progress(user_id: int, guessed_words: List[str]) -> bool:
    """Асинхронная обёртка для сохранения прогресса Wordle"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _save_wordle_progress_sync, user_id, guessed_words)


# ========== WORDLE: СОХРАНЕНИЕ СОСТОЯНИЯ ИГРЫ ==========

def _get_wordle_state_sync(user_id: int) -> Optional[Dict]:
    """
    Получить состояние игры Wordle для пользователя.
    Возвращает: {current_word, attempts: [[...]], last_word_date}
    """
    if not GSPREAD_AVAILABLE:
        return None
    
    try:
        client = get_google_sheets_client()
        if not client:
            return None
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        _ensure_required_sheets_sync()
        
        try:
            sheet = spreadsheet.worksheet("Wordle_Состояние")
        except Exception:
            return None
        
        all_values = sheet.get_all_values()
        for row in all_values[1:]:  # Пропускаем заголовок
            if row and len(row) > 0 and str(row[0]).strip() == str(user_id):
                current_word = row[1].strip() if len(row) > 1 and row[1] else None
                attempts_json = row[2].strip() if len(row) > 2 and row[2] else "[]"
                last_word_date = row[3].strip() if len(row) > 3 and row[3] else None
                
                try:
                    attempts = json.loads(attempts_json) if attempts_json else []
                except:
                    attempts = []
                
                return {
                    'current_word': current_word,
                    'attempts': attempts,
                    'last_word_date': last_word_date
                }
        
        return None
    except Exception as e:
        logger.error(f"Ошибка получения состояния Wordle для user_id={user_id}: {e}")
        return None


def _save_wordle_state_sync(user_id: int, current_word: str, attempts: List[List[Dict]], last_word_date: str) -> bool:
    """
    Сохранить состояние игры Wordle для пользователя.
    """
    if not GSPREAD_AVAILABLE:
        return False
    
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        _ensure_required_sheets_sync()
        
        try:
            sheet = spreadsheet.worksheet("Wordle_Состояние")
        except Exception:
            # Создаем лист если его нет
            sheet = spreadsheet.add_worksheet(title="Wordle_Состояние", rows=100, cols=4)
            sheet.append_row(["user_id", "current_word", "attempts", "last_word_date"])
        
        all_values = sheet.get_all_values()
        user_row_index = None
        
        for i, row in enumerate(all_values[1:], start=2):  # Пропускаем заголовок
            if row and len(row) > 0 and str(row[0]).strip() == str(user_id):
                user_row_index = i
                break
        
        attempts_json = json.dumps(attempts)
        
        if user_row_index:
            # Обновляем существующую строку
            sheet.update_cell(user_row_index, 2, current_word)  # current_word
            sheet.update_cell(user_row_index, 3, attempts_json)  # attempts
            sheet.update_cell(user_row_index, 4, last_word_date)  # last_word_date
        else:
            # Создаем новую строку
            sheet.append_row([str(user_id), current_word, attempts_json, last_word_date])
        
        return True
    except Exception as e:
        logger.error(f"Ошибка сохранения состояния Wordle для user_id={user_id}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False


def _get_wordle_word_for_user_sync(user_id: int) -> Optional[str]:
    """
    Получить актуальное слово для пользователя с учетом автоматической смены раз в сутки.
    Если прошло больше суток с последнего слова, берем следующее из таблицы.
    """
    if not GSPREAD_AVAILABLE:
        return None
    
    try:
        client = get_google_sheets_client()
        if not client:
            return None
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        _ensure_required_sheets_sync()
        
        # Получаем все слова из таблицы Wordle
        try:
            word_sheet = spreadsheet.worksheet("Wordle")
        except Exception:
            return None
        
        all_values = word_sheet.get_all_values()
        words = []
        for row in all_values[1:]:  # Пропускаем заголовок
            if row and len(row) > 0 and row[0].strip():
                words.append(row[0].strip().upper())
        
        if not words:
            return None
        
        # Получаем состояние пользователя
        state = _get_wordle_state_sync(user_id)
        
        if not state or not state.get('current_word') or not state.get('last_word_date'):
            # Первый раз или нет сохраненного состояния - берем первое слово
            current_word = words[0]
            today = datetime.now().strftime('%Y-%m-%d')
            _save_wordle_state_sync(user_id, current_word, [], today)
            return current_word
        
        # Проверяем, прошло ли больше суток
        last_date = state['last_word_date']
        today = datetime.now().strftime('%Y-%m-%d')
        
        if last_date != today:
            # Нужно сменить слово на следующее
            current_word = state['current_word']
            try:
                current_index = words.index(current_word)
                next_index = (current_index + 1) % len(words)  # Циклически переходим к следующему
                next_word = words[next_index]
            except ValueError:
                # Текущее слово не найдено в списке, берем первое
                next_word = words[0]
            
            # Сохраняем новое слово и сбрасываем попытки
            _save_wordle_state_sync(user_id, next_word, [], today)
            return next_word
        
        # Слово не менялось, возвращаем текущее
        return state['current_word']
    
    except Exception as e:
        logger.error(f"Ошибка получения слова Wordle для user_id={user_id}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return None


async def get_wordle_word_for_user(user_id: int) -> Optional[str]:
    """Асинхронная обёртка для получения слова Wordle для пользователя"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _get_wordle_word_for_user_sync, user_id)


async def get_wordle_state(user_id: int) -> Optional[Dict]:
    """Асинхронная обёртка для получения состояния Wordle"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _get_wordle_state_sync, user_id)


async def save_wordle_state(user_id: int, current_word: str, attempts: List[List[Dict]], last_word_date: str) -> bool:
    """Асинхронная обёртка для сохранения состояния Wordle"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _save_wordle_state_sync, user_id, current_word, attempts, last_word_date)


# ========== КРОССВОД ==========

def _ensure_required_sheets_sync():
    """
    Проверяет и создает все необходимые вкладки в Google Sheets, если их нет.
    """
    if not GSPREAD_AVAILABLE:
        return False
    
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        existing_sheets = [ws.title for ws in spreadsheet.worksheets()]
        
        required_sheets = {
            "Кроссвод": {
                "headers": ["слово", "описание"],
                "default_data": [
                    ["СВАДЬБА", "Главное событие дня"],
                    ["ТАМАДА", "Ведущий праздника"],
                    ["ФАТА", "Головной убор невесты"],
                    ["БУКЕТ", "Цветы в руках невесты"],
                    ["КОЛЬЦО", "Символ брака"],
                    ["ТОРТ", "Сладкое угощение"],
                    ["ТОСТ", "Поздравление гостей"],
                    ["ТАНЕЦ", "Развлечение на празднике"],
                ]
            },
            "Кроссвод_Прогресс": {
                "headers": ["user_id", "current_crossword_index", "guessed_words_json"],
                "default_data": []
            },
            "Wordle": {
                "headers": ["слово"],
                "default_data": [
                    ["ГОСТИ"],
                    ["ТАНЕЦ"],
                    ["БУКЕТ"],
                ]
            },
            "Wordle_Прогресс": {
                "headers": ["user_id", "отгаданные_слова"],
                "default_data": []
            },
            "Wordle_Состояние": {
                "headers": ["user_id", "current_word", "attempts", "last_word_date"],
                "default_data": []
            },
            "Игры": {
                "headers": ["user_id", "first_name", "last_name", "total_score", "dragon_score", "flappy_score", "crossword_score", "wordle_score", "rank", "last_updated"],
                "default_data": []
            },
            "Фото": {
                "headers": ["timestamp", "user_id", "username", "full_name", "photo_data"],
                "default_data": []
            },
        }
        
        for sheet_name, sheet_config in required_sheets.items():
            if sheet_name not in existing_sheets:
                try:
                    headers = sheet_config["headers"]
                    default_data = sheet_config.get("default_data", [])
                    sheet = spreadsheet.add_worksheet(title=sheet_name, rows=100, cols=len(headers))
                    sheet.append_row(headers)
                    
                    # Добавляем дефолтные данные, если они есть
                    if default_data:
                        for row in default_data:
                            sheet.append_row(row)
                        logger.info(f"Создана вкладка '{sheet_name}' с заголовками и {len(default_data)} дефолтными строками")
                    else:
                        logger.info(f"Создана вкладка '{sheet_name}' с заголовками: {headers}")
                except Exception as e:
                    logger.error(f"Ошибка создания вкладки '{sheet_name}': {e}")
        
        return True
    except Exception as e:
        logger.error(f"Ошибка проверки/создания вкладок: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False


def _get_crossword_words_sync(crossword_index: int = 0) -> List[Dict[str, str]]:
    """
    Получить слова для кроссвода из листа 'Кроссвод'.
    
    Каждый столбец = один кроссворд:
      Столбец A (index 0): слова для кроссворда 1
      Столбец B (index 1): описания для кроссворда 1
      Столбец C (index 2): слова для кроссворда 2
      Столбец D (index 3): описания для кроссворда 2
      и т.д.
    
    Args:
        crossword_index: Индекс кроссворда (0 = первый, 1 = второй, и т.д.)
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return []
    
    try:
        client = get_google_sheets_client()
        if not client:
            return []
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        
        # Проверяем и создаем необходимые вкладки
        _ensure_required_sheets_sync()
        
        try:
            sheet = spreadsheet.worksheet("Кроссвод")
        except Exception:
            # Создаем лист если его нет
            sheet = spreadsheet.add_worksheet(title="Кроссвод", rows=100, cols=10)
            return []
        
        all_values = sheet.get_all_values()
        words = []
        
        # Вычисляем индексы столбцов для данного кроссворда
        # crossword_index 0: столбцы 0 (слова) и 1 (описания)
        # crossword_index 1: столбцы 2 (слова) и 3 (описания)
        word_col = crossword_index * 2
        desc_col = crossword_index * 2 + 1
        
        # Пропускаем заголовок (если есть)
        start_row = 0
        if all_values and len(all_values) > 0:
            # Проверяем, есть ли заголовок (первая строка может быть заголовком)
            if all_values[0] and (all_values[0][0] == 'слово' or all_values[0][0] == 'Слово'):
                start_row = 1
        
        for row in all_values[start_row:]:
            if len(row) > max(word_col, desc_col):
                word = row[word_col].strip().upper() if row[word_col] else ''
                description = row[desc_col].strip() if row[desc_col] else ''
                if word and description:
                    words.append({
                        'word': word,
                        'description': description
                    })
        
        # Если слов нет для данного кроссворда и это первый кроссворд, добавляем дефолтные
        if not words and crossword_index == 0:
            default_words = [
                ["СВАДЬБА", "Главное событие дня"],
                ["ТАМАДА", "Ведущий праздника"],
                ["ФАТА", "Головной убор невесты"],
                ["БУКЕТ", "Цветы в руках невесты"],
                ["КОЛЬЦО", "Символ брака"],
                ["ТОРТ", "Сладкое угощение"],
                ["ТОСТ", "Поздравление гостей"],
                ["ТАНЕЦ", "Развлечение на празднике"],
            ]
            try:
                # Заполняем столбцы A и B (первый кроссворд)
                for i, word_desc in enumerate(default_words):
                    row_num = i + 1  # Начинаем с первой строки (после заголовка, если есть)
                    if start_row == 0:
                        row_num = i  # Если заголовка нет, начинаем с 0
                    
                    # Убеждаемся, что строка существует
                    while len(all_values) <= row_num:
                        all_values.append([''] * (desc_col + 1))
                    
                    # Обновляем значения
                    if len(all_values[row_num]) <= word_col:
                        all_values[row_num].extend([''] * (word_col - len(all_values[row_num]) + 1))
                    if len(all_values[row_num]) <= desc_col:
                        all_values[row_num].extend([''] * (desc_col - len(all_values[row_num]) + 1))
                    
                    all_values[row_num][word_col] = word_desc[0]
                    all_values[row_num][desc_col] = word_desc[1]
                
                # Обновляем лист
                sheet.update('A1', all_values)
                logger.info(f"Добавлены дефолтные слова в кроссвод: {len(default_words)} слов")
                # Возвращаем дефолтные слова
                words = [
                    {'word': row[0], 'description': row[1]}
                    for row in default_words
                ]
            except Exception as e:
                logger.error(f"Ошибка добавления дефолтных слов в кроссвод: {e}")
        
        return words
    except Exception as e:
        logger.error(f"Ошибка получения слов кроссворда: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []


def _get_crossword_progress_sync(user_id: int, crossword_index: int = 0) -> List[str]:
    """
    Получить прогресс пользователя в кроссводе (список отгаданных слов для конкретного кроссворда).
    
    Формат строки в листе 'Кроссвод_Прогресс':
      A: user_id
      B: current_crossword_index
      C: guessed_words_json (JSON объект: {"0": ["слово1", "слово2"], "1": ["слово3"], ...})
    
    Args:
        user_id: ID пользователя
        crossword_index: Индекс кроссворда (0 = первый, 1 = второй, и т.д.)
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return []
    
    try:
        client = get_google_sheets_client()
        if not client:
            return []
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        
        # Проверяем и создаем необходимые вкладки
        _ensure_required_sheets_sync()
        
        try:
            sheet = spreadsheet.worksheet("Кроссвод_Прогресс")
        except Exception:
            # Создаем лист если его нет
            sheet = spreadsheet.add_worksheet(title="Кроссвод_Прогресс", rows=100, cols=2)
            # Добавляем заголовки
            sheet.append_row(["user_id", "отгаданные_слова"])
            return []
        
        all_values = sheet.get_all_values()
        for row in all_values[1:]:  # Пропускаем заголовок
            if len(row) > 0 and str(row[0]) == str(user_id):
                if len(row) > 1 and row[1]:
                    # Разбиваем строку с отгаданными словами
                    guessed_words = [w.strip().upper() for w in row[1].split(',') if w.strip()]
                    return guessed_words
        
        return []
    except Exception as e:
        logger.error(f"Ошибка получения прогресса кроссворда для {user_id}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return []


def _save_crossword_progress_sync(user_id: int, guessed_words: List[str], crossword_index: int = 0) -> bool:
    """
    Сохранить прогресс пользователя в кроссводе для конкретного кроссворда.
    
    Args:
        user_id: ID пользователя
        guessed_words: Список отгаданных слов
        crossword_index: Индекс кроссворда (0 = первый, 1 = второй, и т.д.)
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен")
        return False
    
    try:
        client = get_google_sheets_client()
        if not client:
            return False
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        
        # Проверяем и создаем необходимые вкладки
        _ensure_required_sheets_sync()
        
        try:
            sheet = spreadsheet.worksheet("Кроссвод_Прогресс")
        except Exception:
            # Создаем лист если его нет
            sheet = spreadsheet.add_worksheet(title="Кроссвод_Прогресс", rows=100, cols=3)
            # Добавляем заголовки
            sheet.append_row(["user_id", "current_crossword_index", "guessed_words_json"])
        
        all_values = sheet.get_all_values()
        user_row_gspread = None
        existing_guessed_words_json = {}
        existing_crossword_index = crossword_index
        
        for i, row in enumerate(all_values[1:], start=2):  # Пропускаем заголовок
            if row and len(row) > 0 and str(row[0]) == str(user_id):
                user_row_gspread = i
                # Получаем существующий прогресс
                if len(row) > 1 and row[1]:
                    try:
                        existing_crossword_index = int(row[1])
                    except:
                        pass
                if len(row) > 2 and row[2]:
                    try:
                        existing_guessed_words_json = json.loads(row[2])
                    except:
                        existing_guessed_words_json = {}
                break
        
        # Обновляем прогресс для данного кроссворда
        crossword_key = str(crossword_index)
        existing_guessed_words_json[crossword_key] = [w.upper() for w in guessed_words]
        guessed_words_json_str = json.dumps(existing_guessed_words_json)
        
        if user_row_gspread:
            # Обновляем существующую строку
            # Обновляем индекс только если он больше текущего (нельзя вернуться назад)
            existing_index = existing_crossword_index
            if crossword_index > existing_index:
                sheet.update_cell(user_row_gspread, 2, crossword_index)  # current_crossword_index
            sheet.update_cell(user_row_gspread, 3, guessed_words_json_str)  # guessed_words_json
        else:
            # Создаем новую строку
            sheet.append_row([str(user_id), crossword_index, guessed_words_json_str])
        
        logger.info(f"Сохранен прогресс кроссвода для user_id={user_id}, кроссворд {crossword_index}: {len(guessed_words)} слов")
        return True
    except Exception as e:
        logger.error(f"Ошибка сохранения прогресса кроссвода для {user_id}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False


async def ensure_required_sheets():
    """Асинхронная обёртка для проверки и создания необходимых вкладок"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _ensure_required_sheets_sync)


async def get_crossword_words(crossword_index: int = 0) -> List[Dict[str, str]]:
    """Асинхронная обёртка для получения слов кроссвода"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _get_crossword_words_sync, crossword_index)


async def get_crossword_progress(user_id: int, crossword_index: int = 0) -> List[str]:
    """Асинхронная обёртка для получения прогресса кроссвода"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _get_crossword_progress_sync, user_id, crossword_index)


async def save_crossword_progress(user_id: int, guessed_words: List[str], crossword_index: int = 0) -> bool:
    """Асинхронная обёртка для сохранения прогресса кроссвода"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _save_crossword_progress_sync, user_id, guessed_words, crossword_index)


def _get_crossword_state_sync(user_id: int) -> Dict:
    """
    Получить состояние кроссворда для пользователя (текущий индекс кроссворда).
    """
    if not GSPREAD_AVAILABLE:
        return {'current_crossword_index': 0}
    
    try:
        client = get_google_sheets_client()
        if not client:
            return {'current_crossword_index': 0}
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        _ensure_required_sheets_sync()
        
        try:
            sheet = spreadsheet.worksheet("Кроссвод_Прогресс")
        except Exception:
            return {'current_crossword_index': 0}
        
        all_values = sheet.get_all_values()
        for row in all_values[1:]:  # Пропускаем заголовок
            if row and len(row) > 0 and str(row[0]).strip() == str(user_id):
                if len(row) > 1 and row[1]:
                    try:
                        return {'current_crossword_index': int(row[1])}
                    except:
                        pass
                return {'current_crossword_index': 0}
        
        return {'current_crossword_index': 0}
    except Exception as e:
        logger.error(f"Ошибка получения состояния кроссворда для user_id={user_id}: {e}")
        return {'current_crossword_index': 0}


async def get_crossword_state(user_id: int) -> Dict:
    """Асинхронная обёртка для получения состояния кроссворда"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _get_crossword_state_sync, user_id)

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
