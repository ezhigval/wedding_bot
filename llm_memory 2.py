"""
Модуль для хранения памяти и контекста диалогов для LLM
"""
import sqlite3
import json
import logging
import os
from datetime import datetime
from typing import List, Dict, Optional
import aiosqlite

logger = logging.getLogger(__name__)

DB_PATH = "data/llm_memory.db"

async def init_memory_db():
    """Инициализация базы данных для памяти LLM"""
    os.makedirs("data", exist_ok=True)
    
    async with aiosqlite.connect(DB_PATH) as db:
        # Таблица для хранения фактов/информации
        await db.execute("""
            CREATE TABLE IF NOT EXISTS facts (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                key TEXT UNIQUE NOT NULL,
                value TEXT NOT NULL,
                context TEXT,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                usage_count INTEGER DEFAULT 0
            )
        """)
        
        # Таблица для хранения истории диалогов
        await db.execute("""
            CREATE TABLE IF NOT EXISTS conversations (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                chat_id TEXT NOT NULL,
                user_id TEXT,
                user_name TEXT,
                message TEXT NOT NULL,
                response TEXT,
                timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                importance REAL DEFAULT 0.5
            )
        """)
        
        # Индексы для быстрого поиска
        await db.execute("CREATE INDEX IF NOT EXISTS idx_facts_key ON facts(key)")
        await db.execute("CREATE INDEX IF NOT EXISTS idx_conversations_chat_id ON conversations(chat_id)")
        await db.execute("CREATE INDEX IF NOT EXISTS idx_conversations_timestamp ON conversations(timestamp DESC)")
        
        await db.commit()

async def save_fact(key: str, value: str, context: Optional[str] = None):
    """Сохраняет факт/информацию в память"""
    try:
        async with aiosqlite.connect(DB_PATH) as db:
            await db.execute("""
                INSERT OR REPLACE INTO facts (key, value, context, updated_at, usage_count)
                VALUES (?, ?, ?, CURRENT_TIMESTAMP, 
                    COALESCE((SELECT usage_count FROM facts WHERE key = ?), 0) + 1)
            """, (key, value, context, key))
            await db.commit()
            logger.info(f"Сохранен факт: {key} = {value}")
    except Exception as e:
        logger.error(f"Ошибка сохранения факта: {e}")

async def get_fact(key: str) -> Optional[str]:
    """Получает факт по ключу"""
    try:
        async with aiosqlite.connect(DB_PATH) as db:
            async with db.execute("SELECT value FROM facts WHERE key = ?", (key,)) as cursor:
                row = await cursor.fetchone()
                if row:
                    # Увеличиваем счетчик использования
                    await db.execute("UPDATE facts SET usage_count = usage_count + 1 WHERE key = ?", (key,))
                    await db.commit()
                    return row[0]
        return None
    except Exception as e:
        logger.error(f"Ошибка получения факта: {e}")
        return None

async def get_all_facts() -> List[Dict]:
    """Получает все сохраненные факты"""
    try:
        async with aiosqlite.connect(DB_PATH) as db:
            async with db.execute("SELECT key, value, context FROM facts ORDER BY usage_count DESC, updated_at DESC") as cursor:
                rows = await cursor.fetchall()
                return [{"key": row[0], "value": row[1], "context": row[2]} for row in rows]
    except Exception as e:
        logger.error(f"Ошибка получения фактов: {e}")
        return []

async def save_conversation(chat_id: str, user_id: Optional[str], user_name: Optional[str], 
                           message: str, response: Optional[str] = None, importance: float = 0.5):
    """Сохраняет диалог в историю"""
    try:
        async with aiosqlite.connect(DB_PATH) as db:
            await db.execute("""
                INSERT INTO conversations (chat_id, user_id, user_name, message, response, importance)
                VALUES (?, ?, ?, ?, ?, ?)
            """, (chat_id, user_id, user_name, message, response, importance))
            await db.commit()
    except Exception as e:
        logger.error(f"Ошибка сохранения диалога: {e}")

async def get_recent_conversations(chat_id: str, limit: int = 10) -> List[Dict]:
    """Получает последние диалоги из чата"""
    try:
        async with aiosqlite.connect(DB_PATH) as db:
            async with db.execute("""
                SELECT user_name, message, response, timestamp
                FROM conversations
                WHERE chat_id = ?
                ORDER BY timestamp DESC
                LIMIT ?
            """, (chat_id, limit)) as cursor:
                rows = await cursor.fetchall()
                return [
                    {
                        "user_name": row[0],
                        "message": row[1],
                        "response": row[2],
                        "timestamp": row[3]
                    }
                    for row in rows
                ]
    except Exception as e:
        logger.error(f"Ошибка получения диалогов: {e}")
        return []

async def extract_and_save_info(message: str, response: str) -> List[str]:
    """
    Извлекает важную информацию из диалога и сохраняет в память
    Возвращает список сохраненных фактов
    """
    saved_facts = []
    
    # Ключевые слова для извлечения информации
    info_patterns = {
        "время": ["время", "когда", "во сколько", "начало", "конец", "старт"],
        "место": ["место", "где", "адрес", "локация", "зал", "холл"],
        "дресс-код": ["дресс-код", "одежда", "наряд", "цвет", "палитра", "что надеть"],
        "программа": ["программа", "расписание", "план", "события", "мероприятия", "таймлайн"],
        "контакты": ["контакт", "телефон", "связаться", "написать", "связь"],
        "подарки": ["подарок", "пожелания", "вино", "бюджет", "что подарить"],
        "транспорт": ["транспорт", "как добраться", "дорога", "машина", "парковка"],
        "питание": ["еда", "меню", "питание", "ресторан", "банкет"],
        "развлечения": ["развлечения", "музыка", "танцы", "игры"],
    }
    
    # Простое извлечение информации
    text_lower = (message + " " + response).lower()
    
    for key, patterns in info_patterns.items():
        if any(pattern in text_lower for pattern in patterns):
            # Извлекаем релевантную часть
            for pattern in patterns:
                if pattern in text_lower:
                    # Сохраняем как факт (берем более релевантную часть ответа)
                    fact_key = f"chat_info_{key}"
                    # Пытаемся найти релевантную часть ответа
                    fact_value = response
                    if len(fact_value) > 300:
                        # Берем первые 300 символов, но пытаемся найти упоминание ключевого слова
                        idx = text_lower.find(pattern)
                        if idx > 0:
                            start = max(0, idx - 50)
                            end = min(len(response), idx + 250)
                            fact_value = response[start:end]
                        else:
                            fact_value = response[:300]
                    
                    await save_fact(fact_key, fact_value, context=message)
                    saved_facts.append(key)
                    break
    
    return saved_facts

async def get_context_for_llm(chat_id: str, max_facts: int = 5) -> str:
    """Формирует контекст для LLM из сохраненной памяти"""
    context_parts = []
    
    # Получаем последние диалоги
    recent_convs = await get_recent_conversations(chat_id, limit=3)
    if recent_convs:
        context_parts.append("Последние сообщения в чате:")
        for conv in reversed(recent_convs):  # В хронологическом порядке
            if conv["user_name"]:
                context_parts.append(f"- {conv['user_name']}: {conv['message']}")
            if conv["response"]:
                context_parts.append(f"  Ответ: {conv['response'][:100]}...")
    
    # Получаем важные факты
    facts = await get_all_facts()
    if facts:
        context_parts.append("\nВажная информация из предыдущих диалогов:")
        for fact in facts[:max_facts]:
            context_parts.append(f"- {fact['key']}: {fact['value'][:150]}...")
    
    return "\n".join(context_parts) if context_parts else ""

