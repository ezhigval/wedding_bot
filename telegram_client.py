"""
Модуль для работы с Telegram Client API (Telethon)
Используется для поиска username по номеру телефона
"""
import logging
import asyncio
from typing import Optional

logger = logging.getLogger(__name__)

# Пытаемся импортировать Telethon
try:
    from telethon import TelegramClient
    from telethon.errors import SessionPasswordNeededError, PhoneNumberInvalidError
    TELETHON_AVAILABLE = True
except ImportError:
    TELETHON_AVAILABLE = False
    logger.warning("Telethon не установлен. Поиск username по номеру телефона недоступен.")

# Глобальный клиент (инициализируется один раз)
_client = None

async def init_telegram_client(api_id: str, api_hash: str, phone: str, session_file: str = "telegram_session"):
    """
    Инициализация Telegram клиента
    
    Args:
        api_id: API ID из my.telegram.org
        api_hash: API Hash из my.telegram.org
        phone: Номер телефона админа (формат: +79001234567)
        session_file: Имя файла для сохранения сессии
    
    Returns:
        TelegramClient или None если не удалось инициализировать
    """
    if not TELETHON_AVAILABLE:
        logger.warning("Telethon недоступен")
        return None
    
    global _client
    
    try:
        api_id_int = int(api_id) if api_id else None
        if not api_id_int or not api_hash or not phone:
            logger.warning("Не указаны API_ID, API_HASH или PHONE для Telegram Client")
            return None
        
        _client = TelegramClient(session_file, api_id_int, api_hash)
        await _client.connect()
        
        # Проверяем, авторизован ли клиент
        if not await _client.is_user_authorized():
            logger.warning("Telegram клиент не авторизован. Нужно отправить код подтверждения.")
            # Отправляем код
            await _client.send_code_request(phone)
            # В реальном использовании нужно будет запросить код у админа
            # Для автоматизации можно сохранить код в переменной окружения
            return None
        
        logger.info("Telegram клиент успешно инициализирован")
        return _client
        
    except Exception as e:
        logger.error(f"Ошибка инициализации Telegram клиента: {e}")
        return None

async def get_username_by_phone(phone_number: str) -> Optional[str]:
    """
    Получить username по номеру телефона
    
    Args:
        phone_number: Номер телефона (формат: +79001234567, 89001234567, 79001234567)
    
    Returns:
        Username (без @) или None если не найден
    """
    if not TELETHON_AVAILABLE or not _client:
        logger.warning("Telegram клиент недоступен")
        return None
    
    try:
        # Нормализуем номер телефона
        phone_normalized = phone_number.strip()
        if phone_normalized.startswith("8"):
            phone_normalized = "+7" + phone_normalized[1:]
        elif phone_normalized.startswith("7"):
            phone_normalized = "+" + phone_normalized
        elif not phone_normalized.startswith("+"):
            phone_normalized = "+" + phone_normalized
        
        # Импортируем контакт
        result = await _client.import_contacts([{
            'phone': phone_normalized,
            'first_name': 'Temp',
            'last_name': 'User'
        }])
        
        if result.imported and len(result.imported) > 0:
            user_id = result.imported[0].user_id
            # Получаем информацию о пользователе
            user = await _client.get_entity(user_id)
            if user and hasattr(user, 'username') and user.username:
                logger.info(f"Найден username для {phone_number}: @{user.username}")
                return user.username
        
        logger.warning(f"Username не найден для номера {phone_number}")
        return None
        
    except Exception as e:
        logger.error(f"Ошибка поиска username по номеру телефона {phone_number}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return None

async def close_client():
    """Закрыть соединение с Telegram"""
    global _client
    if _client:
        await _client.disconnect()
        _client = None

