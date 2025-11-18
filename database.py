import aiosqlite
from config import DB_PATH
import os

async def init_db():
    """Инициализация базы данных"""
    # Создаем папку data если её нет
    os.makedirs("data", exist_ok=True)
    
    async with aiosqlite.connect(DB_PATH) as db:
        # Таблица гостей
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
        
        # Таблица соответствия username -> имя
        await db.execute("""
            CREATE TABLE IF NOT EXISTS name_mapping (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                username TEXT NOT NULL UNIQUE,
                first_name TEXT NOT NULL,
                last_name TEXT NOT NULL,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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

async def delete_guest(user_id: int):
    """Удаляет гостя из базы данных по user_id"""
    async with aiosqlite.connect(DB_PATH) as db:
        await db.execute("""
            DELETE FROM guests 
            WHERE user_id = ?
        """, (user_id,))
        await db.commit()
        return True

async def add_name_mapping(username: str, first_name: str, last_name: str):
    """Добавляет или обновляет соответствие username -> имя"""
    async with aiosqlite.connect(DB_PATH) as db:
        await db.execute("""
            INSERT OR REPLACE INTO name_mapping (username, first_name, last_name)
            VALUES (?, ?, ?)
        """, (username.lower(), first_name, last_name))
        await db.commit()

async def get_name_by_username(username: str):
    """Получает имя по username из таблицы соответствия"""
    if not username:
        return None
    
    async with aiosqlite.connect(DB_PATH) as db:
        async with db.execute("""
            SELECT first_name, last_name 
            FROM name_mapping 
            WHERE username = ?
        """, (username.lower(),)) as cursor:
            row = await cursor.fetchone()
            if row:
                return (row[0], row[1])  # (first_name, last_name)
            return None

async def get_all_name_mappings():
    """Получает все соответствия username -> имя"""
    async with aiosqlite.connect(DB_PATH) as db:
        async with db.execute("""
            SELECT username, first_name, last_name 
            FROM name_mapping 
            ORDER BY username
        """) as cursor:
            rows = await cursor.fetchall()
            return rows

async def delete_name_mapping(username: str):
    """Удаляет соответствие username -> имя"""
    async with aiosqlite.connect(DB_PATH) as db:
        await db.execute("""
            DELETE FROM name_mapping 
            WHERE username = ?
        """, (username.lower(),))
        await db.commit()

async def init_default_mappings():
    """Инициализация начальных данных соответствия"""
    try:
        existing = await get_name_by_username("ezhigval")
        if not existing:
            await add_name_mapping("ezhigval", "Валентин", "Ежов")
    except:
        pass  # Игнорируем ошибки при инициализации

