"""
Модуль для синхронизации рассадки гостей между листами Google Sheets.

Максимально точно повторяет логику исходного Google Apps Script:
- Лист «Список гостей» (имена и выбранный стол)
- Лист «Рассадка» (столы по колонкам, гости по строкам)
- Перестройка шапки «Рассадки» из правила Data Validation столбца G
- Двусторонняя синхронизация:
  * Список гостей → Рассадка
  * Рассадка → Список гостей

На этом этапе модуль НЕ подключён к боту или API — это отдельный слой,
который можно вызывать вручную или из обёртки-триггера.
"""

from __future__ import annotations

import asyncio
import json
import logging
from typing import Dict, List, Optional, Tuple

from google.auth.transport.requests import AuthorizedSession
from google.oauth2.service_account import Credentials

from config import (
    GOOGLE_SHEETS_ID,
    GOOGLE_SHEETS_CREDENTIALS,
    GOOGLE_SHEETS_SHEET_NAME,
)
from google_sheets import get_google_sheets_client

logger = logging.getLogger(__name__)


# Имена листов и номера колонок — в точности как в Apps Script
GUEST_SHEET = GOOGLE_SHEETS_SHEET_NAME  # "Список гостей"
SEATING_SHEET = "Рассадка"

COL_NAME = 1  # A: имя гостя
COL_TABLE = 7  # G: столик


# ========== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ДЛЯ ДОСТУПА К GOOGLE SHEETS ==========


def _get_authorized_session() -> Optional[AuthorizedSession]:
    """
    Создать авторизованную сессию Google API на основе тех же credentials,
    что и для gspread. Нужна для чтения dataValidation (списка столов) напрямую
    через Google Sheets API.
    """
    if not GOOGLE_SHEETS_CREDENTIALS:
        logger.warning("GOOGLE_SHEETS_CREDENTIALS не установлен, невозможно создать сессию Google API")
        return None

    try:
        creds_dict = json.loads(GOOGLE_SHEETS_CREDENTIALS)
        creds = Credentials.from_service_account_info(
            creds_dict,
            scopes=["https://www.googleapis.com/auth/spreadsheets"],
        )
        return AuthorizedSession(creds)
    except Exception as e:
        logger.error(f"Ошибка создания AuthorizedSession для Google Sheets API: {e}")
        import traceback

        logger.error(traceback.format_exc())
        return None


def _get_tables_from_validation_sync() -> List[str]:
    """
    Получить список названий столов из правила Data Validation ячейки G2
    на листе «Список гостей», максимально повторяя Apps Script:

      const rule = guestSheet.getRange(2, COL_TABLE).getDataValidation();
      ...
      const tables = args[0].filter(x => x && x.toString().trim() !== "");
    """
    session = _get_authorized_session()
    if not session:
        return []

    try:
        from urllib.parse import quote

        # A1-нотация для G2 на листе гостей
        range_a1 = f"'{GUEST_SHEET}'!G2"
        url = (
            f"https://sheets.googleapis.com/v4/spreadsheets/{quote(GOOGLE_SHEETS_ID)}"
            f"?ranges={quote(range_a1)}&includeGridData=true"
        )

        resp = session.get(url)
        resp.raise_for_status()
        data = resp.json()

        sheets = data.get("sheets", [])
        if not sheets:
            logger.warning("Не удалось получить данные листа для Data Validation (sheets пуст)")
            return []

        grid_data = sheets[0].get("data", [])
        if not grid_data:
            logger.warning("Не удалось получить gridData для Data Validation")
            return []

        row_data = grid_data[0].get("rowData", [])
        if not row_data:
            logger.warning("rowData пуст при чтении Data Validation")
            return []

        values = row_data[0].get("values", [])
        if not values:
            logger.warning("values пуст при чтении Data Validation")
            return []

        cell = values[0]
        dv = cell.get("dataValidation")
        if not dv:
            logger.warning("У ячейки G2 нет правила Data Validation")
            return []

        condition = dv.get("condition", {})
        cond_type = condition.get("type")
        cond_values = condition.get("values", []) or []

        # Нас интересует только список фиксированных значений (ONE_OF_LIST)
        if cond_type != "ONE_OF_LIST":
            logger.warning(f"Тип Data Validation для G2 не ONE_OF_LIST, а {cond_type}")
            return []

        tables: List[str] = []
        for v in cond_values:
            raw = v.get("userEnteredValue")
            if not raw:
                continue
            name = str(raw).strip()
            if name:
                tables.append(name)

        logger.info(f"Получено {len(tables)} столов из Data Validation G2: {tables}")
        return tables
    except Exception as e:
        logger.error(f"Ошибка при чтении Data Validation для столов: {e}")
        import traceback

        logger.error(traceback.format_exc())
        return []


def _get_worksheets_sync():
    """
    Получить объекты листов «Список гостей» и «Рассадка».
    """
    client = get_google_sheets_client()
    if not client:
        raise RuntimeError("Google Sheets клиент недоступен")

    spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
    guest = spreadsheet.worksheet(GUEST_SHEET)
    seating = spreadsheet.worksheet(SEATING_SHEET)
    return guest, seating


# ========== ПЕРЕСТРОЙКА ШАПКИ РАССАДКИ ==========


def _rebuild_seating_header_sync() -> bool:
    """
    Перестроить шапку листа «Рассадка» на основе списка столов из Data Validation G2.

    Полностью повторяет идею:
      - читаем список столов из правила валидации G2
      - очищаем первую строку B1:LastColumn
      - записываем названия столов по колонкам B, C, D...
    """
    try:
        guest, seating = _get_worksheets_sync()
    except Exception as e:
        logger.error(f"Не удалось получить листы для перестройки шапки рассадки: {e}")
        return False

    tables = _get_tables_from_validation_sync()
    if not tables:
        logger.warning("Список столов из Data Validation пуст, шапка рассадки не будет изменена")
        return False

    try:
        from gspread.utils import rowcol_to_a1

        # Очищаем первую строку с B1 по последнюю реально используемую колонку
        last_col = seating.col_count or 2
        if last_col < 2:
            last_col = 2

        start_a1 = "B1"
        end_a1 = rowcol_to_a1(1, last_col)
        seating.batch_clear([f"{start_a1}:{end_a1}"])

        # Записываем столы в B1, C1, D1, ...
        # Если столов меньше, чем колонок — остальные останутся пустыми.
        header_row = ["" for _ in range(last_col - 1)]
        for i, name in enumerate(tables):
            if i >= len(header_row):
                break
            header_row[i] = name

        seating.update(f"{start_a1}:{end_a1}", [header_row])
        logger.info(f"Шапка рассадки обновлена: {tables}")
        return True
    except Exception as e:
        logger.error(f"Ошибка при перестройке шапки рассадки: {e}")
        import traceback

        logger.error(traceback.format_exc())
        return False


async def rebuild_seating_header() -> bool:
    logger.info("[seating_sync] Запуск rebuild_seating_header()")
    loop = asyncio.get_event_loop()
    result = await loop.run_in_executor(None, _rebuild_seating_header_sync)
    logger.info(f"[seating_sync] Завершён rebuild_seating_header(), result={result}")
    return result


# ========== СИНХРОНИЗАЦИЯ: «СПИСОК ГОСТЕЙ» → «РАССАДКА» ==========


def _sync_from_guests_sync() -> None:
    """
    Синхронизация из «Список гостей» в «Рассадка».

    Логика:
    - читаем имена и избранные столы (A и G) из листа гостей
    - очищаем всю зону рассадки (кроме заголовка)
    - читаем заголовок столов из первой строки «Рассадки»
    - распределяем гостей по «карманам» столов
    - внутри каждого стола сортируем гостей по алфавиту
    - записываем имена в соответствующие колонки с 2-й строки
    """
    try:
        import gspread
        from gspread.utils import rowcol_to_a1
    except ImportError:
        logger.error("gspread не установлен — sync_from_guests невозможен")
        return

    try:
        guest, seating = _get_worksheets_sync()
    except Exception as e:
        logger.error(f"Не удалось получить листы для sync_from_guests: {e}")
        return

    try:
        g_values = guest.get_all_values()
        g_last = len(g_values)
        if g_last < 2:
            logger.info("В листе 'Список гостей' нет данных для синхронизации (меньше 2 строк)")
            return

        # Строки 2..g_last (индекс 1..)
        names: List[str] = []
        tables: List[str] = []
        for row in g_values[1:]:
            name = row[COL_NAME - 1].strip() if len(row) >= COL_NAME and row[COL_NAME - 1] else ""
            table = row[COL_TABLE - 1].strip() if len(row) >= COL_TABLE and row[COL_TABLE - 1] else ""
            names.append(name)
            tables.append(table)

        s_values = seating.get_all_values()
        s_last = len(s_values)
        if s_last == 0:
            logger.warning("Лист 'Рассадка' пуст, шапка не найдена")
            return

        s_cols = len(s_values[0]) if s_values[0] else 0
        if s_cols < 2:
            logger.warning("В листе 'Рассадка' недостаточно колонок для рассадки")
            return

        # Очищаем содержимое рассадки (строки 2..s_last, колонки 2..s_cols)
        if s_last >= 2:
            start_row = 2
            end_row = s_last
            start_col = 2
            end_col = s_cols
            start_a1 = rowcol_to_a1(start_row, start_col)
            end_a1 = rowcol_to_a1(end_row, end_col)
            empty_block = [["" for _ in range(end_col - start_col + 1)] for _ in range(end_row - start_row + 1)]
            seating.update(f"{start_a1}:{end_a1}", empty_block)

        # Заголовок столов — первая строка, начиная с колонки B
        header = s_values[0][1:s_cols]

        table_map: Dict[str, int] = {}
        table_buckets: Dict[str, List[str]] = {}

        for idx, t in enumerate(header):
            if not t:
                continue
            col_index = 2 + idx  # B=2
            table_map[t] = col_index
            table_buckets[t] = []

        # Раскидываем гостей по столам
        for name, table in zip(names, tables):
            if not name or not table:
                continue
            col = table_map.get(table)
            if not col:
                # Стол из списка гостей отсутствует в шапке рассадки — как в Apps Script, пропускаем
                continue
            table_buckets[table].append(name)

        # Записываем гостей по столам, сортируя по алфавиту
        for table_name, bucket in table_buckets.items():
            if not bucket:
                continue
            bucket_sorted = sorted(bucket, key=lambda x: x.lower())
            col = table_map[table_name]
            start_row = 2
            end_row = start_row + len(bucket_sorted) - 1
            start_a1 = rowcol_to_a1(start_row, col)
            end_a1 = rowcol_to_a1(end_row, col)
            values = [[name] for name in bucket_sorted]
            seating.update(f"{start_a1}:{end_a1}", values)

        logger.info("Синхронизация 'Список гостей' → 'Рассадка' завершена")
    except Exception as e:
        logger.error(f"Ошибка в sync_from_guests: {e}")
        import traceback

        logger.error(traceback.format_exc())


async def sync_from_guests() -> None:
    logger.info("[seating_sync] Запуск sync_from_guests()")
    loop = asyncio.get_event_loop()
    await loop.run_in_executor(None, _sync_from_guests_sync)
    logger.info("[seating_sync] Завершён sync_from_guests()")


# ========== СИНХРОНИЗАЦИЯ: «РАССАДКА» → «СПИСОК ГОСТЕЙ» ==========


def _sync_from_seating_sync() -> None:
    """
    Синхронизация из «Рассадка» в «Список гостей».

    Логика:
    - читаем заголовки столов из первой строки «Рассадки»
    - для каждой колонки-стола читаем имена гостей внизу
    - собираем словарь new_tables: {имя -> стол}
    - на листе «Список гостей» проходим по всем именам (A) и,
      если имя есть в new_tables, записываем соответствующий стол в G.
    """
    try:
        import gspread
    except ImportError:
        logger.error("gspread не установлен — sync_from_seating невозможен")
        return

    try:
        guest, seating = _get_worksheets_sync()
    except Exception as e:
        logger.error(f"Не удалось получить листы для sync_from_seating: {e}")
        return

    try:
        s_values = seating.get_all_values()
        rows = len(s_values)
        if rows == 0:
            logger.warning("Лист 'Рассадка' пуст, нечего синхронизировать")
            return

        cols = len(s_values[0]) if s_values[0] else 0
        if cols < 2:
            logger.warning("В листе 'Рассадка' нет колонок для столов")
            return

        # Заголовки столов: первая строка, колонки B..последняя
        header = s_values[0][1:cols]
        table_map: Dict[str, int] = {}

        for idx, t in enumerate(header):
            if not t:
                continue
            col_index = 2 + idx  # B=2
            table_map[t] = col_index

        # Список имён на листе гостей
        g_values = guest.get_all_values()
        g_last = len(g_values)
        if g_last < 2:
            logger.info("В листе 'Список гостей' нет данных для sync_from_seating")
            return

        g_names: List[str] = []
        for row in g_values[1:]:
            name = row[COL_NAME - 1].strip() if len(row) >= COL_NAME and row[COL_NAME - 1] else ""
            g_names.append(name)

        # Собираем новые назначения столов
        new_tables: Dict[str, str] = {}
        for table_name, col in table_map.items():
            # чтение значений для этого стола из уже загруженного s_values
            # строки 2..rows-1: индексы 1..rows-1
            for r in range(1, rows):
                row = s_values[r]
                if len(row) <= col - 1:
                    continue
                guest_name = (row[col - 1] or "").strip()
                if not guest_name:
                    continue
                # если имя встречается несколько раз, последнее побеждает — как и в исходном скрипте
                new_tables[guest_name] = table_name

        # Применяем к листу гостей
        updates = 0
        for i, name in enumerate(g_names):
            table_name = new_tables.get(name)
            if not table_name:
                continue
            row_index = i + 2  # сдвиг, т.к. данные начинаются со 2-й строки
            guest.update_cell(row_index, COL_TABLE, table_name)
            updates += 1

        logger.info(f"Синхронизация 'Рассадка' → 'Список гостей' завершена, обновлено строк: {updates}")
    except Exception as e:
        logger.error(f"Ошибка в sync_from_seating: {e}")
        import traceback

        logger.error(traceback.format_exc())


async def sync_from_seating() -> None:
    logger.info("[seating_sync] Запуск sync_from_seating()")
    loop = asyncio.get_event_loop()
    await loop.run_in_executor(None, _sync_from_seating_sync)
    logger.info("[seating_sync] Завершён sync_from_seating()")


# ========== ПОЛНАЯ ПЕРЕСБОРКА / ВЫРАВНИВАНИЕ ==========


def _full_reconcile_sync() -> None:
    """
    Полная пересборка, аналог Apps Script fullReconcile():

        rebuildSeatingHeader();
        syncFromGuests();
        syncFromSeating();
    """
    # Шапка может не обновиться (например, если нет Data Validation) —
    # но даже в этом случае продолжаем остальную синхронизацию.
    _rebuild_seating_header_sync()
    _sync_from_guests_sync()
    _sync_from_seating_sync()


async def full_reconcile() -> None:
    logger.info("[seating_sync] Запуск full_reconcile()")
    loop = asyncio.get_event_loop()
    await loop.run_in_executor(None, _full_reconcile_sync)
    logger.info("[seating_sync] Завершён full_reconcile()")


