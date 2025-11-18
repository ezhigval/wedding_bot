import aiosqlite
from config import DB_PATH
import os

async def init_db():
    """Инициализация базы данных"""
    # Создаем папку data если её нет
    os.makedirs("data", exist_ok=True)
    
    async with aiosqlite.connect(DB_PATH) as db:
        await db.execute("""
            CREATE TABLE IF NOT EXISTS guests (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                user_id INTEGER,
                first_name TEXT NOT NULL,
                last_name TEXT NOT NULL,
                username TEXT,
                confirmed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                UNIQUE(user_id)
            )
        """)
        await db.commit()

async def add_guest(user_id: int, first_name: str, last_name: str, username: str = None):
    """Добавляет гостя в базу данных"""
    async with aiosqlite.connect(DB_PATH) as db:
        await db.execute("""
            INSERT OR REPLACE INTO guests (user_id, first_name, last_name, username)
            VALUES (?, ?, ?, ?)
        """, (user_id, first_name, last_name, username))
        await db.commit()

async def get_guest(user_id: int):
    """Получает информацию о госте"""
    async with aiosqlite.connect(DB_PATH) as db:
        async with db.execute("""
            SELECT first_name, last_name, confirmed_at 
            FROM guests 
            WHERE user_id = ?
        """, (user_id,)) as cursor:
            row = await cursor.fetchone()
            return row

async def get_all_guests():
    """Получает список всех гостей"""
    async with aiosqlite.connect(DB_PATH) as db:
        async with db.execute("""
            SELECT first_name, last_name, username, confirmed_at 
            FROM guests 
            ORDER BY confirmed_at DESC
        """) as cursor:
            rows = await cursor.fetchall()
            return rows

async def get_guests_count():
    """Возвращает количество подтвердивших гостей"""
    async with aiosqlite.connect(DB_PATH) as db:
        async with db.execute("SELECT COUNT(*) FROM guests") as cursor:
            row = await cursor.fetchone()
            return row[0] if row else 0

