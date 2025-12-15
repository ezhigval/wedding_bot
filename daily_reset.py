"""
Модуль для автоматической смены слов/кроссвордов в 00:00 и подсчета баллов для не успевших
"""
import asyncio
import logging
from datetime import datetime, timedelta
from typing import List, Dict, Optional
from google_sheets import (
    get_google_sheets_client,
    GOOGLE_SHEETS_ID,
    GSPREAD_AVAILABLE,
    _ensure_required_sheets_sync,
    _update_game_score_sync,
    _get_wordle_state_sync,
    _save_wordle_state_sync,
    _get_crossword_progress_sync,
    _save_crossword_progress_sync,
    _get_total_crosswords_count_sync,
    _get_crossword_state_sync,
    _set_crossword_index_sync,
    _get_wordle_word_for_user_sync,
    _get_game_stats_sync,
    _get_rank_by_score,
    _get_crossword_words_sync,
)
import json

logger = logging.getLogger(__name__)


def add_half_points_sync(user_id: int, game_type: str, points: int):
    """
    Добавляет половину баллов напрямую в Google Sheets.
    
    Args:
        user_id: ID пользователя
        game_type: 'wordle' или 'crossword'
        points: Количество баллов для добавления (3 для Wordle, 13 для Crossword)
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
            sheet = spreadsheet.worksheet("Игры")
        except Exception:
            logger.error("Вкладка 'Игры' не найдена")
            return False
        
        all_values = sheet.get_all_values()
        user_row_gspread = None
        
        for i, row in enumerate(all_values[1:], start=2):
            if row and len(row) > 0 and str(row[0]) == str(user_id):
                user_row_gspread = i
                break
        
        if user_row_gspread:
            # Получаем текущие значения
            row_data = sheet.row_values(user_row_gspread)
            
            total_score = int(row_data[3]) if len(row_data) > 3 and row_data[3] else 0
            
            if game_type == 'wordle':
                wordle_score = int(row_data[7]) if len(row_data) > 7 and row_data[7] else 0
                wordle_score += points
                sheet.update_cell(user_row_gspread, 8, wordle_score)  # H - wordle_score
            elif game_type == 'crossword':
                crossword_score = int(row_data[6]) if len(row_data) > 6 and row_data[6] else 0
                crossword_score += points
                sheet.update_cell(user_row_gspread, 7, crossword_score)  # G - crossword_score
            
            # Обновляем общий счет
            total_score += points
            sheet.update_cell(user_row_gspread, 4, total_score)  # D - total_score
            
            # Обновляем звание
            rank = _get_rank_by_score(total_score)
            sheet.update_cell(user_row_gspread, 9, rank)  # I - rank
            
            # Обновляем дату последнего изменения
            from datetime import datetime
            sheet.update_cell(user_row_gspread, 10, datetime.now().isoformat())  # J - last_updated
            
            logger.info(f"Добавлено {points} баллов для user_id={user_id}, игра={game_type}, новое звание={rank}")
            return True
        else:
            logger.warning(f"Пользователь {user_id} не найден в таблице 'Игры'")
            return False
            
    except Exception as e:
        logger.error(f"Ошибка добавления половины баллов для user_id={user_id}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False


async def process_daily_reset():
    """
    Обрабатывает ежедневную смену слов/кроссвордов в 00:00.
    
    Логика:
    1. Для каждого пользователя проверяем состояние Wordle и Crossword
    2. Если пользователь не успел отгадать до смены:
       - Wordle: добавляем 3 балла (половина от 5, округлено)
       - Crossword: добавляем 13 баллов (половина от 25, округлено)
    3. Меняем слово/кроссворд на следующие
    """
    if not GSPREAD_AVAILABLE:
        logger.warning("Google Sheets недоступен, пропускаем ежедневный сброс")
        return
    
    try:
        client = get_google_sheets_client()
        if not client:
            logger.error("Не удалось получить клиент Google Sheets")
            return
        
        spreadsheet = client.open_by_key(GOOGLE_SHEETS_ID)
        _ensure_required_sheets_sync()
        
        today = datetime.now().strftime('%Y-%m-%d')
        yesterday = (datetime.now() - timedelta(days=1)).strftime('%Y-%m-%d')
        
        logger.info(f"Начинаем ежедневный сброс: {today}")
        
        # Обрабатываем Wordle
        await process_wordle_reset(spreadsheet, today, yesterday)
        
        # Обрабатываем Crossword
        await process_crossword_reset(spreadsheet, today, yesterday)
        
        logger.info("Ежедневный сброс завершен успешно")
        
    except Exception as e:
        logger.error(f"Ошибка при ежедневном сбросе: {e}")
        import traceback
        logger.error(traceback.format_exc())


async def process_wordle_reset(spreadsheet, today: str, yesterday: str):
    """Обрабатывает сброс Wordle для всех пользователей"""
    try:
        sheet = spreadsheet.worksheet("Wordle_Состояние")
        all_values = sheet.get_all_values()
        
        for i, row in enumerate(all_values[1:], start=2):  # Пропускаем заголовок
            if not row or len(row) == 0 or not row[0]:
                continue
            
            try:
                user_id = int(row[0])
            except (ValueError, TypeError):
                continue
            
            # Получаем состояние пользователя
            state = _get_wordle_state_sync(user_id)
            if not state or not state.get('last_word_date'):
                continue
            
            last_date = state['last_word_date']
            
            # Если последняя дата была вчера или раньше, нужно обработать
            if last_date < today:
                # Проверяем, отгадано ли слово
                attempts = state.get('attempts', [])
                guessed = any(
                    row for row in attempts 
                    if all(cell.get('state') == 'correct' for cell in row)
                )
                
                if not guessed:
                    # Пользователь не успел отгадать - добавляем половину баллов (3 балла)
                    logger.info(f"Пользователь {user_id} не успел отгадать Wordle, добавляем 3 балла")
                    await asyncio.get_event_loop().run_in_executor(
                        None,
                        add_half_points_sync,
                        user_id,
                        'wordle',
                        3
                    )
                
                # Меняем слово на следующее
                new_word = await asyncio.get_event_loop().run_in_executor(
                    None,
                    _get_wordle_word_for_user_sync,
                    user_id
                )
                
                if new_word:
                    # Сохраняем новое состояние с сегодняшней датой
                    await asyncio.get_event_loop().run_in_executor(
                        None,
                        _save_wordle_state_sync,
                        user_id,
                        new_word,
                        [],  # Сбрасываем попытки
                        today,
                        ""   # current_guess
                    )
                    logger.info(f"Обновлено слово Wordle для пользователя {user_id}: {new_word}")
                    
            elif last_date == today:
                # Дата уже сегодня, ничего не делаем
                pass
                
    except Exception as e:
        logger.error(f"Ошибка при обработке сброса Wordle: {e}")
        import traceback
        logger.error(traceback.format_exc())


async def process_crossword_reset(spreadsheet, today: str, yesterday: str):
    """Обрабатывает сброс Crossword для всех пользователей"""
    try:
        sheet = spreadsheet.worksheet("Кроссвод_Прогресс")
        all_values = sheet.get_all_values()
        
        for i, row in enumerate(all_values[1:], start=2):  # Пропускаем заголовок
            if not row or len(row) == 0 or not row[0]:
                continue
            
            try:
                user_id = int(row[0])
            except (ValueError, TypeError):
                continue
            
            # Получаем текущий индекс кроссворда
            crossword_state = await asyncio.get_event_loop().run_in_executor(
                None,
                _get_crossword_state_sync,
                user_id
            )
            
            if not crossword_state:
                continue
            
            current_index = crossword_state.get('current_crossword_index', 0)
            
            # Получаем прогресс для текущего кроссворда
            guessed_words, _, start_date, _ = await asyncio.get_event_loop().run_in_executor(
                None,
                _get_crossword_progress_sync,
                user_id,
                current_index
            )
            
            if not start_date:
                continue
            
            # Если дата начала была вчера или раньше, нужно обработать
            if start_date < today:
                # Получаем слова для текущего кроссворда
                words = await asyncio.get_event_loop().run_in_executor(
                    None,
                    _get_crossword_words_sync,
                    current_index
                )
                
                total_words = len(words)
                
                # Проверяем, решен ли кроссворд
                is_solved = len(guessed_words) >= total_words
                
                if not is_solved:
                    # Пользователь не успел решить - добавляем половину баллов (13 баллов)
                    logger.info(f"Пользователь {user_id} не успел решить кроссворд, добавляем 13 баллов")
                    await asyncio.get_event_loop().run_in_executor(
                        None,
                        add_half_points_sync,
                        user_id,
                        'crossword',
                        13
                    )
                
                # Переходим к следующему кроссворду
                total_crosswords = await asyncio.get_event_loop().run_in_executor(
                    None,
                    _get_total_crosswords_count_sync
                )
                
                next_index = (current_index + 1) % total_crosswords if total_crosswords > 0 else 0
                
                # Устанавливаем новый индекс и сбрасываем прогресс
                await asyncio.get_event_loop().run_in_executor(
                    None,
                    _set_crossword_index_sync,
                    user_id,
                    next_index
                )
                
                # Сохраняем новый прогресс с сегодняшней датой
                await asyncio.get_event_loop().run_in_executor(
                    None,
                    _save_crossword_progress_sync,
                    user_id,
                    [],  # Сбрасываем отгаданные слова
                    next_index,
                    None,  # cell_letters
                    [],  # wrong_attempts
                    today  # start_date
                )
                
                logger.info(f"Обновлен кроссворд для пользователя {user_id}: индекс {next_index}")
                
            elif start_date == today:
                # Дата уже сегодня, ничего не делаем
                pass
                
    except Exception as e:
        logger.error(f"Ошибка при обработке сброса Crossword: {e}")
        import traceback
        logger.error(traceback.format_exc())


async def schedule_daily_reset():
    """
    Планирует ежедневный сброс в 00:00
    """
    while True:
        now = datetime.now()
        # Вычисляем время до следующей полуночи
        next_midnight = (now + timedelta(days=1)).replace(hour=0, minute=0, second=0, microsecond=0)
        if now.hour == 0 and now.minute == 0:
            # Если уже 00:00, запускаем сразу
            next_midnight = now.replace(second=0, microsecond=0)
        
        wait_seconds = (next_midnight - now).total_seconds()
        
        logger.info(f"Следующий ежедневный сброс запланирован на {next_midnight} (через {wait_seconds} секунд)")
        
        await asyncio.sleep(wait_seconds)
        
        # Запускаем сброс
        logger.info("Запуск ежедневного сброса...")
        await process_daily_reset()

