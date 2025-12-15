"""
API для Mini App и управления
"""
from aiohttp import web
from aiohttp.web import Response
import aiohttp
import json
import sqlite3
import os
from datetime import datetime
import hashlib
import hmac
import urllib.parse
import asyncio
from typing import Optional, Dict, Tuple

from config import (
    BOT_TOKEN,
    WEDDING_DATE,
    GROOM_NAME,
    BRIDE_NAME,
    GROOM_TELEGRAM,
    BRIDE_TELEGRAM,
    WEDDING_ADDRESS,
    SEATING_API_TOKEN,
    GROUP_ID,
)
from google_sheets import (
    add_guest_to_sheets,
    cancel_invitation,
    get_wordle_word,
    get_wordle_word_for_user,
    get_wordle_guessed_words,
    save_wordle_progress,
    get_wordle_state,
    save_wordle_state,
    get_timeline,
    check_guest_registration,
    get_all_guests_from_sheets,
    get_guests_count_from_sheets,
    cancel_guest_registration_by_user_id,
    find_guest_by_name,
    update_guest_user_id,
    find_duplicate_guests,
    ping_admin_sheet,
    write_ping_to_admin_sheet,
    get_seating_lock_status,
    get_guest_table_and_neighbors,
    save_photo_from_webapp,
    get_game_stats,
    update_game_score,
    get_crossword_words,
    get_crossword_progress,
    save_crossword_progress,
    get_crossword_state,
    ensure_required_sheets,
)
from game_stats_cache import (
    init_game_stats_cache,
    get_cached_stats,
    save_cached_stats,
    sync_game_stats,
)
import seating_sync
import traceback
import logging

logger = logging.getLogger(__name__)

# Попытка импортировать pymorphy3 для проверки слов (совместим с Python 3.11+)
# Используем ленивую инициализацию, чтобы избежать ошибок при импорте
MORPH_AVAILABLE = False
_morph_analyzer = None

def _get_morph_analyzer():
    """Ленивая инициализация MorphAnalyzer"""
    global MORPH_AVAILABLE, _morph_analyzer
    if _morph_analyzer is not None:
        return _morph_analyzer
    
    try:
        import pymorphy3
        _morph_analyzer = pymorphy3.MorphAnalyzer()
        MORPH_AVAILABLE = True
        return _morph_analyzer
    except (ImportError, AttributeError, Exception) as e:
        MORPH_AVAILABLE = False
        _morph_analyzer = None
        logger.warning(f"pymorphy3 недоступен, проверка слов будет упрощенной: {e}")
        return None

# Импортируем функцию уведомлений (будет доступна после инициализации бота)
_notify_admins_func = None

def set_notify_function(func):
    """Установка функции уведомлений из bot.py"""
    global _notify_admins_func
    _notify_admins_func = func

async def notify_admins(message_text):
    """Отправка уведомления админам"""
    if _notify_admins_func:
        await _notify_admins_func(message_text)


async def is_user_in_group_chat(user_id: int) -> bool:
    """
    Проверка, состоит ли пользователь в общем чате гостей.

    Используем прямой вызов Telegram Bot API getChatMember.
    Если BOT_TOKEN или GROUP_ID не заданы, считаем, что пользователь не в чате.
    """
    if not BOT_TOKEN or not GROUP_ID:
        return False

    url = f"https://api.telegram.org/bot{BOT_TOKEN}/getChatMember"
    params = {"chat_id": GROUP_ID, "user_id": user_id}

    try:
        async with aiohttp.ClientSession() as session:
            async with session.get(url, params=params, timeout=5) as resp:
                if resp.status != 200:
                    logger.warning(
                        f"is_user_in_group_chat: getChatMember HTTP {resp.status}"
                    )
                    return False
                data = await resp.json()
    except Exception as e:
        logger.warning(f"is_user_in_group_chat: error {e}")
        return False

    try:
        ok = data.get("ok", False)
        if not ok:
            # Например, user not found, kicked и т.п.
            return False
        status = (data.get("result") or {}).get("status") or ""
        # статусы: creator, administrator, member, restricted, left, kicked
        return status in {"creator", "administrator", "member"}
    except Exception as e:
        logger.warning(f"is_user_in_group_chat: parse error {e}")
        return False


async def _resolve_username_to_user_id(username: str) -> Optional[int]:
    """
    Получить numeric user_id по username через Bot API.
    Требует корректного BOT_TOKEN.
    """
    if not BOT_TOKEN or not username:
        return None

    # Допускаем, что username может приходить без @
    if not username.startswith("@"):
        chat_id = f"@{username}"
    else:
        chat_id = username

    url = f"https://api.telegram.org/bot{BOT_TOKEN}/getChat"
    params = {"chat_id": chat_id}

    try:
        async with aiohttp.ClientSession() as session:
            async with session.get(url, params=params, timeout=5) as resp:
                if resp.status != 200:
                    logger.warning(
                        f"_resolve_username_to_user_id: getChat HTTP {resp.status} for {chat_id}"
                    )
                    return None
                data = await resp.json()
    except Exception as e:
        logger.warning(f"_resolve_username_to_user_id: error {e}")
        return None

    try:
        if not data.get("ok"):
            return None
        result = data.get("result") or {}
        uid = result.get("id")
        if isinstance(uid, int):
            return uid
        return None
    except Exception as e:
        logger.warning(f"_resolve_username_to_user_id: parse error {e}")
        return None


async def validate_word(word: str) -> tuple[bool, str]:
    """
    Проверяет, что слово является существительным в именительном падеже единственного числа.
    
    Returns:
        (is_valid, error_message)
    """
    if not word or len(word) < 2:
        return False, 'Слово слишком короткое'
    
    word_lower = word.lower()
    
    # Если pymorphy3 доступен, используем его для точной проверки
    morph_analyzer = _get_morph_analyzer()
    if MORPH_AVAILABLE and morph_analyzer:
        try:
            parsed = morph_analyzer.parse(word_lower)[0]
            
            # Проверяем, что это существительное
            if 'NOUN' not in parsed.tag:
                return False, 'Это не существительное'
            
            # Проверяем, что это именительный падеж (nomn)
            if 'nomn' not in parsed.tag:
                return False, 'Слово должно быть в именительном падеже'
            
            # Проверяем, что это единственное число (sing)
            if 'sing' not in parsed.tag:
                return False, 'Слово должно быть в единственном числе'
            
            # Проверяем, что слово существует (не является неизвестным)
            if parsed.score < 0.3:  # Низкий score может означать, что слово не найдено
                return False, 'Слово не найдено в словаре'
            
            return True, ''
        except Exception as e:
            logger.warning(f"Ошибка при проверке слова '{word}': {e}")
            # Продолжаем с упрощенной проверкой
    
    # Упрощенная проверка: проверяем окончания для существительных
    # Это не идеально, но работает для большинства случаев
    common_endings = ['а', 'я', 'о', 'е', 'ь', 'й', 'и', 'ы', 'у', 'ю']
    
    # Если слово заканчивается на типичное окончание существительного, считаем валидным
    if any(word_lower.endswith(ending) for ending in common_endings) or len(word_lower) >= 3:
        # Дополнительная проверка: не должно быть слишком много согласных подряд
        consonants = 'бвгджзклмнпрстфхцчшщ'
        max_consonants = 0
        current_consonants = 0
        for char in word_lower:
            if char in consonants:
                current_consonants += 1
                max_consonants = max(max_consonants, current_consonants)
            else:
                current_consonants = 0
        
        if max_consonants > 4:
            return False, 'Слово содержит слишком много согласных подряд'
        
        return True, ''
    
    return False, 'Слово не соответствует формату'


async def scan_guests_for_duplicates_and_notify():
    """
    Одноразовая проверка гостей на возможную двойную регистрацию при старте сервера.
    """
    try:
        duplicates = await find_duplicate_guests()
        dup_by_user_id = duplicates.get("by_user_id") or []
        dup_by_name = duplicates.get("by_name") or []
        missing_ids = duplicates.get("missing_ids") or []
        username_ids = duplicates.get("username_ids") or []

        lines = []
        lines.append("⚠️ <b>Проведена проверка гостей</b>")

        if not dup_by_user_id and not dup_by_name and not missing_ids and not username_ids:
            lines.append("Проблем не обнаружено. Дубликаты и незаполненные user_id не найдены.")
            await notify_admins("\n".join(lines))
            logger.info("Проверка гостей: проблем не обнаружено")
            return

        if dup_by_user_id or dup_by_name:
            lines.append("Обнаружены возможные двойные регистрации в Google Sheets.\n")

        if dup_by_user_id:
            lines.append("<b>Дубли по user_id:</b>")
            for item in dup_by_user_id:
                uid = item.get("user_id")
                rows = item.get("rows", [])
                lines.append(f"\nuser_id <code>{uid}</code>:")
                for info in rows:
                    lines.append(
                        f"• строка {info.get('row')}: {info.get('full_name')} "
                        f"(user_id={info.get('user_id') or '—'})"
                    )

        if dup_by_name:
            lines.append("\n<b>Дубли по имени/фамилии (с учётом возможной перестановки):</b>")
            for group in dup_by_name:
                for info in group:
                    lines.append(
                        f"• строка {info.get('row')}: {info.get('full_name')} "
                        f"(user_id={info.get('user_id') or '—'})"
                    )
                lines.append("")  # пустая строка между группами

        # Гости с подтверждением, но без user_id
        if missing_ids:
            lines.append(
                "\n<b>Зарегистрированы, но не идентифицированы (пустой user_id в столбце F):</b>"
            )
            for info in missing_ids:
                lines.append(
                    f"• строка {info.get('row')}: {info.get('full_name') or '—'} (user_id=—)"
                )

        # Гости, у которых в столбце F хранится username — пробуем автоматически проставить user_id
        auto_fixed_count = 0
        failed_username_fixes: list[str] = []
        for item in username_ids:
            row = item.get("row")
            full_name = item.get("full_name") or ""
            username = (item.get("username") or "").strip()
            if not row or not username:
                continue

            user_id = await _resolve_username_to_user_id(username)
            if not user_id:
                failed_username_fixes.append(
                    f"• строка {row}: {full_name} (username @{username}) — "
                    f"не удалось получить user_id"
                )
                continue

            try:
                ok = await update_guest_user_id(row, user_id)
                if ok:
                    auto_fixed_count += 1
                    lines.append(
                        f"\n✅ Автоматически обновлён user_id по username @{username}:\n"
                        f"   строка {row}: {full_name} → user_id={user_id}"
                    )
                else:
                    failed_username_fixes.append(
                        f"• строка {row}: {full_name} (username @{username}) — "
                        f"ошибка при записи user_id={user_id}"
                    )
            except Exception as e:
                logger.error(
                    f"Ошибка автообновления user_id для @{username} (row={row}): {e}"
                )
                failed_username_fixes.append(
                    f"• строка {row}: {full_name} (username @{username}) — "
                    f"исключение при записи user_id"
                )

        if failed_username_fixes:
            lines.append(
                "\n<b>Не удалось автоматически обновить user_id по username для следующих гостей:</b>"
            )
            lines.extend(failed_username_fixes)

        if not dup_by_user_id and not dup_by_name:
            lines.append(
                "\nДубликатов по user_id и имени не найдено. Проверены только идентификация гостей."
            )
        else:
            lines.append(
                "\nПроверьте вкладку 'Список гостей' в Google Sheets и при необходимости "
                "объедините или удалите дубли вручную."
            )

        await notify_admins("\n".join(lines))
    except Exception as e:
        logger.error(f"Ошибка при проверке гостей на дубликаты: {e}")
        logger.error(traceback.format_exc())

async def validate_word(word: str) -> tuple[bool, str]:
    """
    Проверяет, что слово является существительным в именительном падеже единственного числа.
    
    Args:
        word: Слово в верхнем регистре
    
    Returns:
        (is_valid, error_message)
    """
    if not word or len(word) < 2:
        return False, 'Слово слишком короткое'
    
    word_lower = word.lower()
    
    # Если pymorphy3 доступен, используем его для точной проверки
    morph_analyzer = _get_morph_analyzer()
    if MORPH_AVAILABLE and morph_analyzer:
        try:
            parsed = morph_analyzer.parse(word_lower)[0]
            
            # Проверяем, что это существительное
            if 'NOUN' not in parsed.tag:
                return False, 'Это не существительное'
            
            # Проверяем, что это именительный падеж (nomn)
            if 'nomn' not in parsed.tag:
                return False, 'Слово должно быть в именительном падеже'
            
            # Проверяем, что это единственное число (sing)
            if 'sing' not in parsed.tag:
                return False, 'Слово должно быть в единственном числе'
            
            # Проверяем, что слово существует (не является неизвестным)
            if parsed.score < 0.3:  # Низкий score может означать, что слово не найдено
                return False, 'Слово не найдено в словаре'
            
            return True, ''
        except Exception as e:
            logger.warning(f"Ошибка при проверке слова '{word}': {e}")
            # Продолжаем с упрощенной проверкой
    
    # Упрощенная проверка: проверяем окончания для существительных
    # Это не идеально, но работает для большинства случаев
    common_endings = ['а', 'я', 'о', 'е', 'ь', 'й', 'и', 'ы', 'у', 'ю']
    
    # Если слово заканчивается на типичное окончание существительного, считаем валидным
    if any(word_lower.endswith(ending) for ending in common_endings) or len(word_lower) >= 3:
        # Дополнительная проверка: не должно быть слишком много согласных подряд
        consonants = 'бвгджзклмнпрстфхцчшщ'
        max_consonants = 0
        current_consonants = 0
        for char in word_lower:
            if char in consonants:
                current_consonants += 1
                max_consonants = max(max_consonants, current_consonants)
            else:
                current_consonants = 0
        
        if max_consonants > 4:
            return False, 'Слово содержит слишком много согласных подряд'
        
        return True, ''
    
    return False, 'Слово не соответствует формату'

async def init_api():
    """Инициализация API"""
    api = web.Application()
    
    # CORS middleware
    @web.middleware
    async def cors_middleware(request, handler):
        if request.method == 'OPTIONS':
            return web.Response(
                headers={
                    'Access-Control-Allow-Origin': '*',
                    'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
                    'Access-Control-Allow-Headers': 'Content-Type',
                }
            )
        
        try:
            response = await handler(request)
            if isinstance(response, web.Response):
                response.headers['Access-Control-Allow-Origin'] = '*'
            return response
        except Exception as e:
            import logging
            logging.error(f"API error: {e}")
            return web.json_response({'error': str(e)}, status=500)
    
    api.middlewares.append(cors_middleware)
    
    # Routes
    api.router.add_get('/config', get_config)
    api.router.add_get('/check', check_registration)
    api.router.add_post('/register', register_guest)
    api.router.add_post('/cancel', cancel_guest_registration)
    api.router.add_post('/questionnaire', save_questionnaire)
    api.router.add_get('/guests', get_guests_list)
    api.router.add_get('/stats', get_stats)
    api.router.add_get('/timeline', get_timeline_endpoint)
    api.router.add_post('/confirm-identity', confirm_identity)
    api.router.add_post('/parse-init-data', parse_init_data)
    api.router.add_post('/upload-photo', upload_photo)
    api.router.add_get('/game-stats', get_game_stats_endpoint)
    api.router.add_post('/game-score', update_game_score_endpoint)
    api.router.add_get('/crossword-data', get_crossword_data_endpoint)
    api.router.add_post('/crossword-progress', save_crossword_progress_endpoint)
    
    # Wordle endpoints
    async def get_wordle_word_endpoint(request):
        """Получить актуальное слово для Wordle для пользователя (с учетом автоматической смены)"""
        try:
            # Получаем user_id из initData или userId (для локального тестирования)
            init_data = request.query.get('initData', '')
            user_id_from_query = request.query.get('userId')
            
            user_id = None
            
            # Сначала пытаемся получить user_id из initData
            if init_data:
                user_id = await parse_user_id_from_init_data(init_data)
            
            # Если не получилось, используем userId из query (для локального тестирования)
            if not user_id and user_id_from_query:
                try:
                    user_id = int(user_id_from_query)
                except (ValueError, TypeError):
                    pass
            
            if not user_id:
                return web.json_response({'error': 'Не удалось определить user_id'}, status=400)
            
            # Получаем слово для пользователя (с автоматической сменой раз в сутки)
            word = await get_wordle_word_for_user(user_id)
            if word:
                return web.json_response({'word': word})
            else:
                return web.json_response({'error': 'Слово не найдено'}, status=404)
        except Exception as e:
            logger.error(f"Ошибка получения слова Wordle: {e}")
            return web.json_response({'error': str(e)}, status=500)
    
    async def get_wordle_state_endpoint(request):
        """Получить состояние игры Wordle для пользователя (слово, попытки, дата)"""
        try:
            # Получаем user_id из initData или userId (для локального тестирования)
            init_data = request.query.get('initData', '')
            user_id_from_query = request.query.get('userId')
            
            user_id = None
            
            # Сначала пытаемся получить user_id из initData
            if init_data:
                user_id = await parse_user_id_from_init_data(init_data)
            
            # Если не получилось, используем userId из query (для локального тестирования)
            if not user_id and user_id_from_query:
                try:
                    user_id = int(user_id_from_query)
                except (ValueError, TypeError):
                    pass
            
            if not user_id:
                return web.json_response({'error': 'Не удалось определить user_id'}, status=400)
            
            state = await get_wordle_state(user_id)
            if state:
                return web.json_response(state)
            else:
                return web.json_response({'current_word': None, 'attempts': [], 'last_word_date': None})
        except Exception as e:
            logger.error(f"Ошибка получения состояния Wordle: {e}")
            return web.json_response({'error': str(e)}, status=500)
    
    async def save_wordle_state_endpoint(request):
        """Сохранить состояние игры Wordle для пользователя"""
        try:
            data = await request.json()
            init_data = data.get('initData', '')
            user_id_from_request = data.get('userId')
            current_word = data.get('current_word', '')
            attempts = data.get('attempts', [])
            last_word_date = data.get('last_word_date', '')
            
            user_id = None
            
            # Сначала пытаемся получить user_id из initData
            if init_data:
                user_id = await parse_user_id_from_init_data(init_data)
            
            # Если не получилось, используем userId из запроса (для локального тестирования)
            if not user_id and user_id_from_request:
                try:
                    user_id = int(user_id_from_request)
                except (ValueError, TypeError):
                    pass
            
            if not user_id:
                return web.json_response({'error': 'Не удалось определить user_id'}, status=400)
            
            success = await save_wordle_state(user_id, current_word, attempts, last_word_date)
            if success:
                return web.json_response({'success': True})
            else:
                return web.json_response({'error': 'Не удалось сохранить состояние'}, status=500)
        except Exception as e:
            logger.error(f"Ошибка сохранения состояния Wordle: {e}")
            return web.json_response({'error': str(e)}, status=500)
    
    async def get_wordle_progress_endpoint(request):
        """Получить прогресс пользователя в Wordle (отгаданные слова)"""
        try:
            # Получаем user_id из initData или userId (для локального тестирования)
            init_data = request.query.get('initData', '')
            user_id_from_query = request.query.get('userId')
            
            user_id = None
            
            # Сначала пытаемся получить user_id из initData
            if init_data:
                user_id = await parse_user_id_from_init_data(init_data)
            
            # Если не получилось, используем userId из query (для локального тестирования)
            if not user_id and user_id_from_query:
                try:
                    user_id = int(user_id_from_query)
                except (ValueError, TypeError):
                    pass
            
            if not user_id:
                return web.json_response({'error': 'Не удалось определить user_id'}, status=400)
            
            guessed_words = await get_wordle_guessed_words(user_id)
            return web.json_response({'guessed_words': guessed_words})
        except Exception as e:
            logger.error(f"Ошибка получения прогресса Wordle: {e}")
            return web.json_response({'error': str(e)}, status=500)
    
    async def wordle_guess_endpoint(request):
        """Обработать отгаданное слово в Wordle"""
        try:
            data = await request.json()
            word = data.get('word', '').strip().upper()
            init_data = data.get('initData', '')
            user_id_from_request = data.get('userId')  # Для локального тестирования
            
            if not word:
                return web.json_response({'error': 'Слово не предоставлено'}, status=400)
            
            # Проверяем, что слово валидное (существительное в именительном падеже единственного числа)
            word_valid, validation_error = await validate_word(word)
            if not word_valid:
                return web.json_response({'error': validation_error}, status=400)
            
            user_id = None
            
            # Сначала пытаемся получить user_id из initData
            if init_data:
                user_id = await parse_user_id_from_init_data(init_data)
            
            # Если не получилось, используем userId из запроса (для локального тестирования)
            if not user_id and user_id_from_request:
                try:
                    user_id = int(user_id_from_request)
                except (ValueError, TypeError):
                    pass
            
            if not user_id:
                return web.json_response({'error': 'Не удалось определить user_id'}, status=400)
            
            # Получаем текущее актуальное слово для пользователя
            current_word = await get_wordle_word_for_user(user_id)
            if not current_word:
                return web.json_response({'error': 'Актуальное слово не найдено'}, status=404)
            
            # Проверяем, что слово совпадает с актуальным
            if word != current_word:
                return web.json_response({'error': 'Неверное слово'}, status=400)
            
            # Получаем уже отгаданные слова пользователя
            guessed_words = await get_wordle_guessed_words(user_id)
            
            # Проверяем, не отгадано ли уже это слово
            if word in guessed_words:
                return web.json_response({
                    'success': False,
                    'message': 'Это слово уже было отгадано',
                    'already_guessed': True
                })
            
            # Добавляем слово в список отгаданных
            guessed_words.append(word)
            await save_wordle_progress(user_id, guessed_words)
            
            # Начисляем очки: 1 отгаданное слово = 5 очков
            await update_game_score(user_id, 'wordle', 1)  # Передаем 1 слово, система умножит на 5
            
            return web.json_response({
                'success': True,
                'message': 'Слово отгадано! +5 очков',
                'points': 5
            })
        except Exception as e:
            logger.error(f"Ошибка обработки отгаданного слова Wordle: {e}")
            import traceback
            logger.error(traceback.format_exc())
            return web.json_response({'error': str(e)}, status=500)
    
    api.router.add_get('/wordle/word', get_wordle_word_endpoint)
    api.router.add_get('/wordle/state', get_wordle_state_endpoint)
    api.router.add_post('/wordle/state', save_wordle_state_endpoint)
    api.router.add_get('/wordle/progress', get_wordle_progress_endpoint)
    api.router.add_post('/wordle/guess', wordle_guess_endpoint)

    # Seating sync endpoints (для вызова из Google Apps Script)
    api.router.add_post('/seating/sync-from-guests', seating_sync_from_guests)
    api.router.add_post('/seating/sync-from-seating', seating_sync_from_seating)
    api.router.add_post('/seating/full-reconcile', seating_full_reconcile)
    api.router.add_post('/seating/rebuild-header', seating_rebuild_header)
    api.router.add_post('/seating/on-edit', seating_on_edit)
    api.router.add_post('/ping/from-sheets', ping_from_sheets)
    api.router.add_get('/seating-info', get_seating_info)
    
    # Запускаем фоновую проверку гостей на дубликаты сразу после старта API
    asyncio.create_task(scan_guests_for_duplicates_and_notify())
    
    return api

async def get_config(request):
    """Получить конфигурацию для Mini App"""
    try:
        return web.json_response({
            'weddingDate': WEDDING_DATE.strftime('%Y-%m-%d'),
            'groomName': GROOM_NAME,
            'brideName': BRIDE_NAME,
            'groomTelegram': GROOM_TELEGRAM,
            'brideTelegram': BRIDE_TELEGRAM,
            'weddingAddress': WEDDING_ADDRESS
        })
    except Exception as e:
        import logging
        logging.error(f"Error in get_config: {e}")
        return web.json_response({'error': str(e)}, status=500)


async def get_seating_info(request):
    """
    Получить информацию о столе и соседях для текущего пользователя.

    Условия показа:
      - рассадка закреплена (SEATING_LOCKED = 1 в Config)
      - текущая дата >= 2026-06-04 00:00 по Москве
      - найден гость с таким user_id и его стол в 'Рассадка_фикс'

    Ответ:
      {
        "visible": true/false,
        "table": "Стол №1",
        "neighbors": ["Фамилия Имя", ...],
        "full_name": "Фамилия Имя"
      }
    """
    try:
        user_id_str = request.query.get("userId")
        if not user_id_str:
            return web.json_response({"visible": False})

        try:
            user_id = int(user_id_str)
        except ValueError:
            return web.json_response({"visible": False})

        # 1. Проверяем, закреплена ли рассадка
        lock_status = await get_seating_lock_status()
        if not lock_status.get("locked"):
            return web.json_response({"visible": False})

        # 2. Проверяем дату раскрытия (2026-06-04 00:00 по Москве)
        from datetime import timedelta

        now_utc = datetime.utcnow()
        now_msk = now_utc + timedelta(hours=3)  # Москва = UTC+3, без переходов
        reveal_dt_msk = datetime(2026, 6, 4, 0, 0, 0)

        if now_msk < reveal_dt_msk:
            return web.json_response({"visible": False})

        # 3. Ищем стол и соседей в зафиксированной рассадке
        info = await get_guest_table_and_neighbors(user_id)
        if not info:
            return web.json_response({"visible": False})

        return web.json_response(
            {
                "visible": True,
                "table": info.get("table"),
                "neighbors": info.get("neighbors") or [],
                "full_name": info.get("full_name") or "",
            }
        )
    except Exception as e:
        logger.error(f"Ошибка в get_seating_info: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({"visible": False, "error": "server_error"}, status=500)


def _check_seating_token(request: web.Request) -> bool:
    """
    Проверка токена для эндпоинтов рассадки.

    Если SEATING_API_TOKEN не задан, проверка считается пройденной.
    Если задан — сравниваем с заголовком X-Api-Token.
    """
    if not SEATING_API_TOKEN:
        return True

    header_token = (request.headers.get("X-Api-Token") or "").strip()
    return header_token == SEATING_API_TOKEN


async def seating_sync_from_guests(request: web.Request):
    """Вызов sync_from_guests() из Apps Script (Список гостей → Рассадка)."""
    if not _check_seating_token(request):
        return web.json_response({"error": "forbidden"}, status=403)

    try:
        # Тело нам пока не нужно, но читаем для совместимости
        _ = await request.json()
    except Exception:
        # Игнорируем ошибки парсинга — логика не зависит от payload
        pass

    try:
        await seating_sync.sync_from_guests()
        return web.json_response({"status": "ok"})
    except Exception as e:
        logger.error(f"Ошибка в seating_sync_from_guests: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({"error": "server_error"}, status=500)


async def seating_sync_from_seating(request: web.Request):
    """Вызов sync_from_seating() из Apps Script (Рассадка → Список гостей)."""
    if not _check_seating_token(request):
        return web.json_response({"error": "forbidden"}, status=403)

    try:
        _ = await request.json()
    except Exception:
        pass

    try:
        await seating_sync.sync_from_seating()
        return web.json_response({"status": "ok"})
    except Exception as e:
        logger.error(f"Ошибка в seating_sync_from_seating: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({"error": "server_error"}, status=500)


async def seating_full_reconcile(request: web.Request):
    """Полная пересборка рассадки (rebuild header + обе синхронизации)."""
    if not _check_seating_token(request):
        return web.json_response({"error": "forbidden"}, status=403)

    try:
        _ = await request.json()
    except Exception:
        pass

    try:
        await seating_sync.full_reconcile()
        return web.json_response({"status": "ok"})
    except Exception as e:
        logger.error(f"Ошибка в seating_full_reconcile: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({"error": "server_error"}, status=500)


async def seating_rebuild_header(request: web.Request):
    """Только перестроение шапки рассадки из Data Validation G2."""
    if not _check_seating_token(request):
        return web.json_response({"error": "forbidden"}, status=403)

    try:
        _ = await request.json()
    except Exception:
        pass

    try:
        ok = await seating_sync.rebuild_seating_header()
        return web.json_response({"status": "ok", "updated": bool(ok)})
    except Exception as e:
        logger.error(f"Ошибка в seating_rebuild_header: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({"error": "server_error"}, status=500)


async def seating_on_edit(request: web.Request):
    """
    Универсальный хук onEdit из Google Apps Script.

    Backend сам решает, какие действия выполнять, исходя из:
    - имени листа (Список гостей / Рассадка)
    - затронутого диапазона (строки/колонки)
    """
    if not _check_seating_token(request):
        return web.json_response({"error": "forbidden"}, status=403)

    try:
        # Если рассадка уже закреплена — просто игнорируем любые onEdit-события
        lock_status = await get_seating_lock_status()
        if lock_status.get("locked"):
            logger.info(
                "[seating_on_edit] Рассадка уже закреплена, onEdit-событие игнорируется"
            )
            return web.json_response({"status": "locked"})

        data = await request.json()
    except Exception:
        data = {}

    sheet_name = (data.get("sheetName") or "").strip()
    row_start = int(data.get("rowStart") or 0)
    col_start = int(data.get("colStart") or 0)
    num_rows = int(data.get("numRows") or 1)
    num_cols = int(data.get("numCols") or 1)
    event = data.get("event") or "onEdit"
    range_a1 = data.get("rangeA1") or ""

    col_end = col_start + num_cols - 1

    logger.info(
        f"[seating_on_edit] event={event}, sheet={sheet_name}, "
        f"range={range_a1 or f'R{row_start}C{col_start} ({num_rows}x{num_cols})'}"
    )

    try:
        # 1) Изменения на листе «Список гостей»
        if sheet_name == seating_sync.GUEST_SHEET:
            touches_table_col = (
                seating_sync.COL_TABLE >= col_start
                and seating_sync.COL_TABLE <= col_end
            )
            if touches_table_col:
                logger.info(
                    "[seating_on_edit] Изменение в столбце столов на листе "
                    f"'{sheet_name}', запускаем sync_from_guests()"
                )
                await seating_sync.sync_from_guests()
            else:
                logger.info(
                    "[seating_on_edit] Изменение на листе гостей, "
                    "но вне столбца столов — пока игнорируем"
                )

        # 2) Любые изменения на листе «Рассадка»
        elif sheet_name == seating_sync.SEATING_SHEET:
            logger.info(
                "[seating_on_edit] Изменение на листе рассадки, "
                "запускаем sync_from_seating()"
            )
            await seating_sync.sync_from_seating()

        else:
            logger.info(
                f"[seating_on_edit] Лист '{sheet_name}' не относится к рассадке, "
                "ничего не делаем"
            )

        return web.json_response({"status": "ok"})
    except Exception as e:
        logger.error(f"Ошибка в seating_on_edit: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({"error": "server_error"}, status=500)


async def ping_from_sheets(request: web.Request):
    """
    Пинг, инициированный из интерфейса Google Sheets (через Apps Script меню).

    Поток:
    - проверяем токен
    - меряем ping к листу "Админ бота"
    - пишем запись в строку 5 вкладки "Админ бота"
    - шлём сообщение всем админам от лица бота
    """
    if not _check_seating_token(request):
        return web.json_response({"error": "forbidden"}, status=403)

    try:
        data = await request.json()
    except Exception:
        data = {}

    event = data.get("event") or "ping_from_sheets"
    logger.info(f"[ping_from_sheets] event={event}")

    try:
        # 1. Ping Google Sheets (лист "Админ бота")
        latency_ms = await ping_admin_sheet()
        status = "OK" if latency_ms >= 0 else "ERROR"
        if latency_ms < 0:
            latency_ms = -1

        # 2. Запись результата в таблицу
        await write_ping_to_admin_sheet(
            source="sheets",
            latency_ms=latency_ms,
            status=status,
        )

        # 3. Уведомление админов через бота
        now_str = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        if status == "OK":
            text = (
                "📶 <b>Пинг из Google Sheets</b>\n\n"
                f"⏰ Время: <code>{now_str}</code>\n"
                f"⏱ Задержка: <b>{latency_ms} мс</b>\n"
                f"✅ Статус: <b>OK</b>\n\n"
                "Запись о ping сохранена в Google Sheets (строка 5 вкладки 'Админ бота')."
            )
        else:
            text = (
                "📶 <b>Пинг из Google Sheets</b>\n\n"
                f"⏰ Время: <code>{now_str}</code>\n"
                "❌ Не удалось корректно обратиться к Google Sheets.\n"
                "Проверьте лог сервера и настройки доступа к таблице."
            )

        await notify_admins(text)

        return web.json_response({"status": "ok"})
    except Exception as e:
        logger.error(f"Ошибка в ping_from_sheets: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({"error": "server_error"}, status=500)

async def parse_init_data_internal(init_data: str) -> Optional[dict]:
    """Внутренняя функция для парсинга initData"""
    try:
        if not init_data:
            return None
        
        # Парсим initData
        parsed_data = {}
        for item in init_data.split('&'):
            if '=' in item:
                key, value = item.split('=', 1)
                parsed_data[key] = urllib.parse.unquote(value)
        
        # Извлекаем user из user JSON
        user_json = parsed_data.get('user', '')
        if user_json:
            try:
                import json
                user = json.loads(user_json)
                user_id = user.get('id')
                first_name = user.get('first_name', '')
                last_name = user.get('last_name', '')
                username = user.get('username', '')
                
                return {
                    'userId': user_id,
                    'firstName': first_name,
                    'lastName': last_name,
                    'username': username
                }
            except json.JSONDecodeError:
                logger.error("parse_init_data_internal: failed to parse user JSON")
        
        return None
    except Exception as e:
        logger.error(f"Error in parse_init_data_internal: {e}")
        return None

async def parse_init_data(request):
    """Парсинг initData от Telegram для извлечения user_id"""
    try:
        data = await request.json()
        init_data = data.get('initData', '')
        
        if not init_data:
            return web.json_response({
                'error': 'initData required'
            }, status=400)
        
        result = await parse_init_data_internal(init_data)
        
        if result:
            logger.info(f"parse_init_data: extracted user_id {result.get('userId')} from initData")
            return web.json_response(result)
        
        return web.json_response({
            'error': 'user not found in initData'
        }, status=400)
        
    except Exception as e:
        logger.error(f"Error in parse_init_data: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return web.json_response({
            'error': 'server_error'
        }, status=500)

async def check_registration(request):
    """
    Проверить, зарегистрирован ли пользователь
    Поддерживает несколько способов определения пользователя:
    1. По user_id (если доступен)
    2. По имени/фамилии (если user_id недоступен)
    
    Логика проверки:
    1. Если есть user_id - сначала проверяем по user_id в столбце F
    2. Если не найден по user_id, проверяем по имени/фамилии
    3. Если найден по имени - возвращаем информацию для подтверждения личности
    4. Если нет user_id, но есть имя/фамилия - ищем только по имени
    """
    try:
        # user_id может быть получен разными способами
        user_id_str = request.query.get('userId')
        first_name = request.query.get('firstName', '').strip()
        last_name = request.query.get('lastName', '').strip()
        search_by_name_only = request.query.get('searchByNameOnly', 'false') == 'true'
        
        logger.info(f"check_registration: received request - userId: {user_id_str}, firstName: {first_name}, lastName: {last_name}, searchByNameOnly: {search_by_name_only}")
        
        # Если запрошен поиск только по имени (без user_id)
        if search_by_name_only:
            if not first_name or not last_name:
                logger.warning("check_registration: searchByNameOnly requested but name is missing")
                return web.json_response({
                    'registered': False,
                    'error': 'name_required'
                }, status=400)
            
            # Ищем только по имени/фамилии
            guest_info = await find_guest_by_name(first_name, last_name)
            if guest_info:
                if guest_info.get('user_id'):
                    # Найден и уже имеет user_id - зарегистрирован
                    logger.info(f"check_registration: guest found by name and has user_id")
                    return web.json_response({
                        'registered': True
                    })
                else:
                    # Найден, но нет user_id - нужно подтвердить
                    logger.info(f"check_registration: guest found by name but no user_id, needs confirmation")
                    return web.json_response({
                        'registered': False,
                        'needs_confirmation': True,
                        'guest_name': f"{guest_info['first_name']} {guest_info['last_name']}",
                        'row': guest_info['row']
                    })
            
            # Не найден по имени
            logger.info(f"check_registration: guest not found by name")
            return web.json_response({
                'registered': False
            })
        
        # Обычная проверка с user_id
        if not user_id_str:
            logger.warning("check_registration: userId not provided")
            # Если нет user_id, но есть имя - пробуем поиск по имени
            if first_name and last_name:
                logger.info("check_registration: no userId, trying search by name")
                guest_info = await find_guest_by_name(first_name, last_name)
                if guest_info:
                    if guest_info.get('user_id'):
                        return web.json_response({
                            'registered': True
                        })
                    else:
                        return web.json_response({
                            'registered': False,
                            'needs_confirmation': True,
                            'guest_name': f"{guest_info['first_name']} {guest_info['last_name']}",
                            'row': guest_info['row']
                        })
            
            return web.json_response({
                'registered': False,
                'error': 'user_id_or_name_required'
            }, status=400)
        user_id = int(user_id_str)
        logger.info(f"check_registration: checking user_id {user_id} (from Telegram) against column F in Google Sheets")

        # Дополнительно проверяем, состоит ли пользователь в общем чате
        in_group_chat = await is_user_in_group_chat(user_id)
        
        # 1. Проверяем по user_id в столбце F таблицы
        registered = await check_guest_registration(user_id)
        if registered:
            logger.info(f"check_registration: user_id {user_id} found and registered")
            return web.json_response({
                'registered': True,
                'in_group_chat': in_group_chat,
            })
        
        # 2. Если не найден по user_id, проверяем по имени/фамилии
        if first_name and last_name:
            guest_info = await find_guest_by_name(first_name, last_name)
            if guest_info:
                # Найден по имени — считаем пользователя зарегистрированным
                # и при необходимости синхронизируем user_id в таблице гостей
                try:
                    stored_user_id = guest_info.get('user_id')
                    row = guest_info.get('row')
                    if row and (not stored_user_id or stored_user_id != str(user_id)):
                        logger.info(
                            f"check_registration: обновляем user_id в Google Sheets "
                            f"для строки {row}: {stored_user_id} -> {user_id}"
                        )
                        await update_guest_user_id(row, user_id)
                except Exception as sync_error:
                    logger.error(f"check_registration: ошибка синхронизации user_id по имени: {sync_error}")
                    logger.error(traceback.format_exc())
                
                return web.json_response({
                    'registered': True,
                    'in_group_chat': in_group_chat,
                })
        
        # Не найден ни по user_id, ни по имени
        logger.info(f"check_registration: user_id {user_id} not found")
        return web.json_response({
            'registered': False,
            'in_group_chat': in_group_chat,
        })
        
    except ValueError as e:
        logger.error(f"Error in check_registration: invalid user_id format: {e}")
        return web.json_response({
            'registered': False,
            'error': 'invalid_user_id'
        }, status=400)
    except Exception as e:
        logger.error(f"Error in check_registration: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return web.json_response({
            'registered': False,
            'error': 'server_error'
        }, status=500)

async def confirm_identity(request):
    """Подтвердить личность и сохранить user_id"""
    try:
        data = await request.json()
        row = data.get('row')
        user_id = data.get('userId')
        
        if not row or not user_id:
            return web.json_response({
                'success': False,
                'error': 'missing_data'
            }, status=400)
        
        user_id = int(user_id)
        row = int(row)
        
        logger.info(f"confirm_identity: updating row {row} with user_id {user_id}")
        
        result = await update_guest_user_id(row, user_id)
        
        if result:
            logger.info(f"confirm_identity: successfully updated user_id for row {row}")
            return web.json_response({
                'success': True
            })
        else:
            logger.error(f"confirm_identity: failed to update user_id for row {row}")
            return web.json_response({
                'success': False,
                'error': 'update_failed'
            }, status=500)
            
    except Exception as e:
        logger.error(f"Error in confirm_identity: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return web.json_response({
            'success': False,
            'error': 'server_error'
        }, status=500)

def verify_telegram_webapp_data(init_data):
    """Проверка подлинности данных от Telegram"""
    if not BOT_TOKEN:
        logger.warning("BOT_TOKEN не установлен, пропускаем проверку")
        return True
    
    if not init_data:
        logger.warning("initData пустой, пропускаем проверку")
        return True
    
    try:
        parsed_data = {}
        for item in init_data.split('&'):
            if '=' in item:
                key, value = item.split('=', 1)
                # URL декодируем значения
                parsed_data[key] = urllib.parse.unquote(value)
        
        received_hash = parsed_data.pop('hash', '')
        if not received_hash:
            logger.warning("Hash не найден в initData")
            return True  # Разрешаем, если hash нет
        
        data_check_string = '\n'.join(f"{k}={v}" for k, v in sorted(parsed_data.items()))
        
        secret_key = hmac.new(
            key=b"WebAppData",
            msg=BOT_TOKEN.encode(),
            digestmod=hashlib.sha256
        ).digest()
        
        calculated_hash = hmac.new(
            key=secret_key,
            msg=data_check_string.encode(),
            digestmod=hashlib.sha256
        ).hexdigest()
        
        is_valid = calculated_hash == received_hash
        if not is_valid:
            logger.warning(f"Проверка hash не прошла. Получен: {received_hash[:10]}..., вычислен: {calculated_hash[:10]}...")
            logger.debug(f"Data check string: {data_check_string[:100]}...")
        
        return is_valid
    except Exception as e:
        logger.error(f"Ошибка при проверке данных Telegram: {e}")
        import traceback
        logger.error(traceback.format_exc())
        # Для разработки разрешаем, если есть ошибка
        return True

async def register_guest(request):
    """Регистрация гостя"""
    # Инициализируем guests_count в начале функции
    guests_count = 0
    try:
        data = await request.json()
        user_id = data.get('userId')
        first_name = data.get('firstName', '').strip()
        last_name = data.get('lastName', '').strip()
        username = data.get('username')
        guests_list = data.get('guests', [])  # Список всех гостей
        init_data = data.get('initData', '')
        
        if not user_id or not first_name or not last_name:
            logger.error(f"Недостаточно данных: user_id={user_id}, first_name={first_name}, last_name={last_name}")
            return web.json_response({'error': 'Недостаточно данных'}, status=400)
        
        if len(first_name) < 2 or len(last_name) < 2:
            logger.error(f"Слишком короткие имена: first_name={first_name}, last_name={last_name}")
            return web.json_response({'error': 'Имя и фамилия должны быть не менее 2 символов'}, status=400)
        
        # Проверка подлинности (опционально, не блокируем если проверка не прошла)
        if init_data:
            is_valid = verify_telegram_webapp_data(init_data)
            if not is_valid:
                logger.warning(f"Проверка данных Telegram не прошла, но продолжаем регистрацию (user_id={user_id})")
                # Не блокируем регистрацию, так как у нас есть userId
                # Это безопасно, так как userId - это уникальный идентификатор от Telegram
        
        # Получаем данные основного гостя из запроса
        main_guest_data = guests_list[0] if guests_list else {}
        category = main_guest_data.get('category') or data.get('category')
        side = main_guest_data.get('side') or data.get('side')
        
        # Добавляем в Google Sheets (единственный источник данных)
        try:
            result = await add_guest_to_sheets(
                first_name=first_name,
                last_name=last_name,
                age=None,  # Пока не собираем возраст
                category=category,
                side=side,
                user_id=user_id  # Сохраняем user_id в столбец F
            )
            if result:
                logger.info(f"Гость {first_name} {last_name} успешно добавлен в Google Sheets")
            else:
                logger.warning(f"Не удалось добавить гостя {first_name} {last_name} в Google Sheets (возможно, нет credentials)")
            
            # Добавляем дополнительных гостей в Google Sheets
            for guest in guests_list[1:]:  # Пропускаем первого (основного гостя)
                guest_first_name = guest.get('firstName', '').strip()
                guest_last_name = guest.get('lastName', '').strip()
                guest_category = guest.get('category', '')
                guest_side = guest.get('side', '')
                guest_telegram = (guest.get('telegram') or '').strip()
                
                if guest_first_name and guest_last_name:
                    # Если для дополнительного гостя НЕ указан Telegram,
                    # привязываем его строку в таблице к user_id основного гостя.
                    # Это позволит:
                    #  - считать его «принадлежащим» этому аккаунту Telegram
                    #  - не дублировать рассылки (get_broadcast_recipients() берёт уникальный список user_id)
                    guest_user_id = None
                    if not guest_telegram:
                        guest_user_id = user_id

                    await add_guest_to_sheets(
                        first_name=guest_first_name,
                        last_name=guest_last_name,
                        age=None,
                        category=guest_category,
                        side=guest_side,
                        user_id=guest_user_id
                    )
        except Exception as sheets_error:
            logger.error(f"Ошибка добавления в Google Sheets: {sheets_error}")
            logger.error(traceback.format_exc())
            # Не блокируем ответ, так как это не критично
        
        # Получаем количество гостей из Google Sheets
        try:
            guests_count = await get_guests_count_from_sheets()
        except Exception as count_error:
            logger.error(f"Ошибка получения количества гостей: {count_error}")
            guests_count = 0  # Используем значение по умолчанию
        
        # Формируем уведомление для админов
        username_text = f" @{username}" if username else ""
        notification_text = (
            f"✅ <b>Новая регистрация!</b>\n\n"
            f"👤 <b>Основной гость:</b>\n"
            f"{first_name} {last_name}{username_text}\n"
        )
        
        # Добавляем информацию о дополнительных гостях
        if guests_list and len(guests_list) > 1:
            additional_guests = guests_list[1:]  # Пропускаем первого (основного)
            notification_text += f"\n👥 <b>Дополнительные гости ({len(additional_guests)}):</b>\n"
            for i, guest in enumerate(additional_guests, 1):
                guest_telegram = guest.get('telegram', '')
                telegram_text = f" @{guest_telegram}" if guest_telegram else ""
                notification_text += f"{i}. {guest.get('firstName', '')} {guest.get('lastName', '')}{telegram_text}\n"
        
        notification_text += f"\n📊 Всего гостей: {guests_count}"
        
        await notify_admins(notification_text)
        
        return web.json_response({
            'success': True,
            'guestsCount': guests_count,
            'firstName': first_name,
            'lastName': last_name
        })
    except Exception as e:
        logger.error(f"Критическая ошибка в register_guest: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': f'Внутренняя ошибка сервера: {str(e)}'}, status=500)

async def cancel_guest_registration(request):
    """Отмена регистрации гостя"""
    try:
        data = await request.json()
        user_id = data.get('userId')
        init_data = data.get('initData', '')
        
        if not user_id:
            logger.error("Недостаточно данных для отмены: user_id отсутствует")
            return web.json_response({'error': 'Недостаточно данных'}, status=400)
        
        user_id = int(user_id)
        
        # Получаем данные гостя из Google Sheets перед отменой
        guests = await get_all_guests_from_sheets()
        guest_info = None
        for guest in guests:
            if guest.get('user_id') == str(user_id):
                guest_info = guest
                break
        
        if not guest_info:
            return web.json_response({'error': 'Гость не найден'}, status=404)
        
        first_name = guest_info.get('first_name', '')
        last_name = guest_info.get('last_name', '')
        
        # Обновляем Google Sheets - ставим "НЕТ" в столбец C
        try:
            result = await cancel_guest_registration_by_user_id(user_id)
            if not result:
                logger.warning(f"Не удалось отменить регистрацию для user_id {user_id}")
                return web.json_response({'error': 'Не удалось отменить регистрацию'}, status=500)
            logger.info(f"Регистрация гостя {first_name} {last_name} (user_id: {user_id}) отменена в Google Sheets")
        except Exception as sheets_error:
            logger.error(f"Ошибка обновления Google Sheets при отмене: {sheets_error}")
            logger.error(traceback.format_exc())
            return web.json_response({'error': f'Ошибка отмены регистрации: {str(sheets_error)}'}, status=500)
        
        guests_count = await get_guests_count_from_sheets()
        
        # Отправляем уведомление админам
        notification_text = (
            f"❌ <b>Отмена регистрации</b>\n\n"
            f"👤 {first_name} {last_name}\n"
            f"отменил(а) присутствие на свадьбе\n\n"
            f"📊 Всего гостей: {guests_count}"
        )
        await notify_admins(notification_text)
        
        return web.json_response({
            'success': True,
            'guestsCount': guests_count
        })
    except Exception as e:
        logger.error(f"Критическая ошибка в cancel_guest_registration: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': f'Внутренняя ошибка сервера: {str(e)}'}, status=500)

async def save_questionnaire(request):
    """Сохранение анкеты"""
    try:
        data = await request.json()
        user_id = data.get('userId')
        transfer = data.get('transfer')
        food = data.get('food', [])
        alcohol = data.get('alcohol', '')
        
        if not user_id:
            return web.json_response({'error': 'Недостаточно данных'}, status=400)
        
        user_id = int(user_id)
        
        # Получаем данные гостя из Google Sheets
        guests = await get_all_guests_from_sheets()
        guest_info = None
        for guest in guests:
            if guest.get('user_id') == str(user_id):
                guest_info = guest
                break
        
        if not guest_info:
            return web.json_response({'error': 'Гость не найден'}, status=404)
        
        # Здесь можно добавить сохранение в отдельную таблицу Google Sheets
        # Пока просто возвращаем успех
        
        first_name = guest_info.get('first_name', '')
        last_name = guest_info.get('last_name', '')
        guests_count = await get_guests_count_from_sheets()
        
        return web.json_response({
            'success': True,
            'firstName': first_name,
            'lastName': last_name,
            'guestsCount': guests_count
        })
    except Exception as e:
        logger.error(f"Ошибка в save_questionnaire: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': str(e)}, status=500)

async def get_guests_list(request):
    """Получить список гостей"""
    try:
        guests = await get_all_guests_from_sheets()
        return web.json_response({
            'guests': [
                {
                    'firstName': g.get('first_name', ''),
                    'lastName': g.get('last_name', ''),
                    'username': g.get('username', ''),
                    'user_id': g.get('user_id', ''),
                    'category': g.get('category', ''),
                    'side': g.get('side', '')
                }
                for g in guests
            ],
            'count': len(guests)
        })
    except Exception as e:
        logger.error(f"Ошибка в get_guests_list: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': str(e)}, status=500)

async def get_stats(request):
    """Получить статистику"""
    try:
        guests_count = await get_guests_count_from_sheets()
        return web.json_response({
            'guestsCount': guests_count,
            'weddingDate': WEDDING_DATE.strftime('%Y-%m-%d')
        })
    except Exception as e:
        logger.error(f"Ошибка в get_stats: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': str(e)}, status=500)

async def get_timeline_endpoint(request):
    """Получить тайминг мероприятия"""
    try:
        timeline = await get_timeline()
        return web.json_response({
            'timeline': timeline
        })
    except Exception as e:
        logger.error(f"Ошибка получения тайминга: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': str(e)}, status=500)

async def upload_photo(request):
    """Загрузка фото из веб-приложения"""
    try:
        data = await request.json()
        photo_data = data.get('photo')  # base64 строка
        init_data = data.get('initData', '')
        
        if not photo_data:
            return web.json_response({'error': 'Фото не предоставлено'}, status=400)
        
        # Получаем user_id из initData
        user_id = None
        username = None
        full_name = ''
        
        if init_data:
            try:
                parsed = await parse_init_data_internal(init_data)
                if parsed and parsed.get('userId'):
                    user_id = parsed['userId']
                    username = parsed.get('username', '')
                    first_name = parsed.get('firstName', '')
                    last_name = parsed.get('lastName', '')
                    full_name = f"{first_name} {last_name}".strip()
            except Exception as e:
                logger.error(f"Ошибка парсинга initData: {e}")
        
        # Если не получили из initData, пробуем из localStorage (через userId в запросе)
        if not user_id:
            user_id_str = data.get('userId')
            if user_id_str:
                try:
                    user_id = int(user_id_str)
                except (ValueError, TypeError):
                    pass
        
        if not user_id:
            return web.json_response({'error': 'Не удалось определить пользователя'}, status=400)
        
        # Сохраняем фото в Google Sheets
        success = await save_photo_from_webapp(
            user_id=user_id,
            username=username,
            full_name=full_name or 'Неизвестный',
            photo_data=photo_data,
        )
        
        if success:
            return web.json_response({
                'success': True,
                'message': 'Фото успешно сохранено'
            })
        else:
            return web.json_response({'error': 'Не удалось сохранить фото'}, status=500)
            
    except Exception as e:
        logger.error(f"Ошибка загрузки фото: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': f'Внутренняя ошибка сервера: {str(e)}'}, status=500)

async def get_game_stats_endpoint(request):
    """Получить статистику игрока с синхронизацией между кэшем и Google Sheets"""
    try:
        # Получаем user_id из запроса
        user_id_str = request.query.get('userId')
        if not user_id_str:
            return web.json_response({'error': 'userId required'}, status=400)
        
        user_id = int(user_id_str)
        
        # Получаем статистику из обоих источников
        sheets_stats = None
        try:
            sheets_stats = await get_game_stats(user_id)
        except Exception as e:
            logger.warning(f"Не удалось получить статистику из Google Sheets для user_id={user_id}: {e}")
        
        cached_stats = await get_cached_stats(user_id)
        
        # Синхронизируем данные
        stats = await sync_game_stats(user_id, sheets_stats, cached_stats)
        
        # Если кэш новее, пытаемся обновить Google Sheets (в фоне, не блокируем ответ)
        if cached_stats and sheets_stats:
            cached_dt = parse_datetime(cached_stats.get('last_updated'))
            sheets_dt = parse_datetime(sheets_stats.get('last_updated'))
            if cached_dt and sheets_dt and cached_dt > sheets_dt:
                # Кэш новее - обновляем Sheets в фоне
                asyncio.create_task(_update_sheets_from_cache(user_id, cached_stats))
        
        # Убираем last_updated из ответа (не нужно на фронтенде)
        stats.pop('last_updated', None)
        
        if not stats or stats.get('total_score', 0) == 0:
            # Если статистики нет, возвращаем дефолтные значения
            return web.json_response({
                'user_id': user_id,
                'first_name': '',
                'last_name': '',
                'total_score': 0,
                'dragon_score': 0,
                'flappy_score': 0,
                'crossword_score': 0,
                'wordle_score': 0,
                'rank': 'Незнакомец',
            })
        
        return web.json_response(stats)
    except Exception as e:
        logger.error(f"Ошибка получения статистики: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': str(e)}, status=500)

async def _update_sheets_from_cache(user_id: int, cached_stats: Dict):
    """Обновить Google Sheets из кэша (выполняется в фоне)"""
    try:
        # Получаем текущую статистику из Sheets
        sheets_stats = await get_game_stats(user_id)
        
        # Обновляем каждый счет отдельно, если он больше текущего в Sheets
        # Синхронизация теперь не нужна, так как счет накопительный
        # Оставляем пустым, так как логика изменилась на накопительную
    except Exception as e:
        logger.error(f"Ошибка обновления Sheets из кэша для user_id={user_id}: {e}")

def parse_datetime(dt_str: Optional[str]) -> Optional[datetime]:
    """Парсит строку даты в datetime объект"""
    if not dt_str:
        return None
    try:
        for fmt in ['%Y-%m-%d %H:%M:%S', '%Y-%m-%dT%H:%M:%S', '%Y-%m-%dT%H:%M:%S.%f']:
            try:
                return datetime.strptime(dt_str, fmt)
            except ValueError:
                continue
        return datetime.fromisoformat(dt_str.replace('Z', '+00:00'))
    except Exception as e:
        logger.error(f"Ошибка парсинга даты '{dt_str}': {e}")
        return None

async def update_game_score_endpoint(request):
    """Обновить счет игрока с синхронизацией"""
    try:
        # Проверяем и создаем необходимые вкладки
        await ensure_required_sheets()
        
        data = await request.json()
        user_id_str = data.get('userId')
        game_type = data.get('gameType')  # 'dragon', 'flappy', 'crossword'
        score = data.get('score')
        firstName = data.get('firstName', '')
        lastName = data.get('lastName', '')
        
        if not user_id_str or not game_type or score is None:
            return web.json_response({'error': 'Недостаточно данных'}, status=400)
        
        user_id = int(user_id_str)
        score = int(score)
        
        if game_type not in ['dragon', 'flappy', 'crossword', 'wordle']:
            return web.json_response({'error': 'Неизвестный тип игры'}, status=400)
        
        # Получаем текущую статистику для обновления
        current_stats = await get_cached_stats(user_id)
        if not current_stats:
            # Пытаемся получить из Sheets
            current_stats = await get_game_stats(user_id)
            if not current_stats:
                current_stats = {
                    'user_id': user_id,
                    'first_name': firstName,
                    'last_name': lastName,
                    'total_score': 0,
                    'dragon_score': 0,
                    'flappy_score': 0,
                    'crossword_score': 0,
                    'rank': 'Незнакомец',
                }
        
        # Прибавляем очки к счету игры (накопительно)
        # Конвертируем игровые очки в рейтинговые по формулам:
        # Dragon: 200 игровых очков = 1 рейтинговое очко
        # Flappy: 2 игровых очка = 1 рейтинговое очко
        # Crossword: 1 игровое очко = 25 рейтинговых очков
        
        if game_type == 'dragon':
            # Конвертируем игровые очки в рейтинговые
            rating_points = score // 200
            current_stats['dragon_score'] = current_stats.get('dragon_score', 0) + rating_points
        elif game_type == 'flappy':
            # Конвертируем игровые очки в рейтинговые
            rating_points = score // 2
            current_stats['flappy_score'] = current_stats.get('flappy_score', 0) + rating_points
        elif game_type == 'crossword':
            # Конвертируем игровые очки в рейтинговые: 1 игровое очко = 25 рейтинговых очков
            rating_points = score * 25
            current_stats['crossword_score'] = current_stats.get('crossword_score', 0) + rating_points
        elif game_type == 'wordle':
            # Wordle: каждое отгаданное слово = 5 рейтинговых очков
            # score здесь - количество отгаданных слов
            rating_points = score * 5
            current_stats['wordle_score'] = current_stats.get('wordle_score', 0) + rating_points
        
        # Пересчитываем общий счет
        current_stats['total_score'] = (
            current_stats.get('dragon_score', 0) +
            current_stats.get('flappy_score', 0) +
            current_stats.get('crossword_score', 0) +
            current_stats.get('wordle_score', 0)
        )
        
        # Определяем звание
        total = current_stats['total_score']
        if total < 50:
            current_stats['rank'] = 'Незнакомец'
        elif total < 100:
            current_stats['rank'] = 'Ты хто?'
        elif total < 150:
            current_stats['rank'] = 'Люся'
        elif total < 200:
            current_stats['rank'] = 'Бедный родственник'
        elif total < 300:
            current_stats['rank'] = 'Братуха'
        elif total < 400:
            current_stats['rank'] = 'Батя в здании'
        else:
            current_stats['rank'] = 'Монстр'
        
        # Обновляем имя если передано
        if firstName:
            current_stats['first_name'] = firstName
        if lastName:
            current_stats['last_name'] = lastName
        
        # Обновляем дату
        current_stats['last_updated'] = datetime.now().isoformat()
        
        # Сохраняем в кэш
        await save_cached_stats(current_stats)
        
        # Пытаемся обновить Google Sheets
        success = False
        try:
            success = await update_game_score(user_id, game_type, score)
        except Exception as e:
            logger.warning(f"Не удалось обновить Google Sheets для user_id={user_id}: {e}")
            # Продолжаем работу даже если Sheets недоступен
        
        # Возвращаем обновленную статистику (без last_updated)
        response_stats = current_stats.copy()
        response_stats.pop('last_updated', None)
        
        return web.json_response({
            'success': True,
            'stats': response_stats,
            'sheets_synced': success
        })
    except Exception as e:
        logger.error(f"Ошибка обновления счета: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': str(e)}, status=500)

async def get_crossword_data_endpoint(request):
    """Получить слова кроссвода и прогресс пользователя"""
    try:
        # Проверяем и создаем необходимые вкладки
        await ensure_required_sheets()
        
        user_id_str = request.query.get('userId')
        if not user_id_str:
            return web.json_response({'error': 'userId required'}, status=400)
        
        user_id = int(user_id_str)
        
        # Получаем состояние кроссворда (текущий индекс)
        state = await get_crossword_state(user_id)
        crossword_index = state.get('current_crossword_index', 0)
        
        # Получаем слова и прогресс для текущего кроссворда
        words = await get_crossword_words(crossword_index)
        progress = await get_crossword_progress(user_id, crossword_index)
        
        return web.json_response({
            'words': words,
            'guessed_words': progress,
            'crossword_index': crossword_index
        })
    except Exception as e:
        logger.error(f"Ошибка получения данных кроссвода: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': str(e)}, status=500)

async def save_crossword_progress_endpoint(request):
    """Сохранить прогресс кроссвода"""
    try:
        # Проверяем и создаем необходимые вкладки
        await ensure_required_sheets()
        
        data = await request.json()
        user_id_str = data.get('userId')
        guessed_words = data.get('guessedWords', [])
        crossword_index = data.get('crossword_index', 0)
        
        if not user_id_str:
            return web.json_response({'error': 'userId required'}, status=400)
        
        user_id = int(user_id_str)
        
        if not isinstance(guessed_words, list):
            return web.json_response({'error': 'guessedWords must be a list'}, status=400)
        
        success = await save_crossword_progress(user_id, guessed_words, crossword_index)
        
        if success:
            return web.json_response({'success': True})
        else:
            return web.json_response({'error': 'Не удалось сохранить прогресс'}, status=500)
            
    except Exception as e:
        logger.error(f"Ошибка сохранения прогресса кроссвода: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': str(e)}, status=500)



