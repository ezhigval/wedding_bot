"""
Модуль для кэширования игровой статистики на сервере
Синхронизация с Google Sheets с проверкой дат последнего обновления
"""
import json
import logging
import os
from datetime import datetime
from typing import Optional, Dict
import aiosqlite

logger = logging.getLogger(__name__)

DB_PATH = "data/game_stats_cache.db"

async def init_game_stats_cache():
    """Инициализация базы данных для кэша игровой статистики"""
    os.makedirs("data", exist_ok=True)
    
    async with aiosqlite.connect(DB_PATH) as db:
        await db.execute("""
            CREATE TABLE IF NOT EXISTS game_stats_cache (
                user_id INTEGER PRIMARY KEY,
                first_name TEXT,
                last_name TEXT,
                total_score INTEGER DEFAULT 0,
                dragon_score INTEGER DEFAULT 0,
                flappy_score INTEGER DEFAULT 0,
                crossword_score INTEGER DEFAULT 0,
                rank TEXT DEFAULT 'Незнакомец',
                last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """)
        
        await db.execute("CREATE INDEX IF NOT EXISTS idx_game_stats_user_id ON game_stats_cache(user_id)")
        await db.execute("CREATE INDEX IF NOT EXISTS idx_game_stats_updated ON game_stats_cache(last_updated DESC)")
        
        await db.commit()

async def get_cached_stats(user_id: int) -> Optional[Dict]:
    """Получить статистику из кэша"""
    try:
        async with aiosqlite.connect(DB_PATH) as db:
            db.row_factory = aiosqlite.Row
            async with db.execute(
                "SELECT * FROM game_stats_cache WHERE user_id = ?",
                (user_id,)
            ) as cursor:
                row = await cursor.fetchone()
                if row:
                    return {
                        'user_id': row['user_id'],
                        'first_name': row['first_name'] or '',
                        'last_name': row['last_name'] or '',
                        'total_score': row['total_score'],
                        'dragon_score': row['dragon_score'],
                        'flappy_score': row['flappy_score'],
                        'crossword_score': row['crossword_score'],
                        'rank': row['rank'] or 'Незнакомец',
                        'last_updated': row['last_updated'],
                    }
        return None
    except Exception as e:
        logger.error(f"Ошибка получения статистики из кэша для user_id={user_id}: {e}")
        return None

async def save_cached_stats(stats: Dict) -> bool:
    """Сохранить статистику в кэш"""
    try:
        async with aiosqlite.connect(DB_PATH) as db:
            await db.execute("""
                INSERT OR REPLACE INTO game_stats_cache 
                (user_id, first_name, last_name, total_score, dragon_score, 
                 flappy_score, crossword_score, rank, last_updated)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            """, (
                stats['user_id'],
                stats.get('first_name', ''),
                stats.get('last_name', ''),
                stats.get('total_score', 0),
                stats.get('dragon_score', 0),
                stats.get('flappy_score', 0),
                stats.get('crossword_score', 0),
                stats.get('rank', 'Незнакомец'),
                stats.get('last_updated', datetime.now().isoformat()),
            ))
            await db.commit()
            return True
    except Exception as e:
        logger.error(f"Ошибка сохранения статистики в кэш: {e}")
        return False

def parse_datetime(dt_str: Optional[str]) -> Optional[datetime]:
    """Парсит строку даты в datetime объект"""
    if not dt_str:
        return None
    try:
        # Пробуем разные форматы
        for fmt in ['%Y-%m-%d %H:%M:%S', '%Y-%m-%dT%H:%M:%S', '%Y-%m-%dT%H:%M:%S.%f']:
            try:
                return datetime.strptime(dt_str, fmt)
            except ValueError:
                continue
        # Если ничего не подошло, пробуем ISO формат
        return datetime.fromisoformat(dt_str.replace('Z', '+00:00'))
    except Exception as e:
        logger.error(f"Ошибка парсинга даты '{dt_str}': {e}")
        return None

async def sync_game_stats(user_id: int, sheets_stats: Optional[Dict], cached_stats: Optional[Dict]) -> Dict:
    """
    Синхронизирует статистику между Google Sheets и кэшем.
    Возвращает более актуальные данные и обновляет оба источника при необходимости.
    
    Args:
        user_id: ID пользователя
        sheets_stats: Статистика из Google Sheets (может содержать last_updated)
        cached_stats: Статистика из кэша (содержит last_updated)
    
    Returns:
        Dict с актуальной статистикой
    """
    # Если нет данных ни в одном источнике
    if not sheets_stats and not cached_stats:
        return {
            'user_id': user_id,
            'first_name': '',
            'last_name': '',
            'total_score': 0,
            'dragon_score': 0,
            'flappy_score': 0,
            'crossword_score': 0,
            'rank': 'Незнакомец',
            'last_updated': datetime.now().isoformat(),
        }
    
    # Если есть только один источник - используем его
    if not sheets_stats:
        return cached_stats
    if not cached_stats:
        # Добавляем текущую дату если её нет
        if 'last_updated' not in sheets_stats:
            sheets_stats['last_updated'] = datetime.now().isoformat()
        await save_cached_stats(sheets_stats)
        return sheets_stats
    
    # Сравниваем даты
    sheets_dt = parse_datetime(sheets_stats.get('last_updated'))
    cached_dt = parse_datetime(cached_stats.get('last_updated'))
    
    # Если даты нет в одном из источников, считаем его устаревшим
    if not sheets_dt and not cached_dt:
        # Оба без дат - используем тот, у которого больше счет
        if sheets_stats.get('total_score', 0) >= cached_stats.get('total_score', 0):
            sheets_stats['last_updated'] = datetime.now().isoformat()
            await save_cached_stats(sheets_stats)
            return sheets_stats
        else:
            return cached_stats
    
    if not sheets_dt:
        # В Sheets нет даты - используем кэш
        return cached_stats
    
    if not cached_dt:
        # В кэше нет даты - используем Sheets и обновляем кэш
        await save_cached_stats(sheets_stats)
        return sheets_stats
    
    # Сравниваем даты
    if sheets_dt > cached_dt:
        # Sheets новее - обновляем кэш
        await save_cached_stats(sheets_stats)
        return sheets_stats
    elif cached_dt > sheets_dt:
        # Кэш новее - возвращаем кэш (но не обновляем Sheets здесь, это делается отдельно)
        return cached_stats
    else:
        # Даты равны - используем Sheets (как основной источник)
        await save_cached_stats(sheets_stats)
        return sheets_stats

