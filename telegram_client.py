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

# Словарь клиентов для каждого админа (ключ - user_id админа)
_clients = {}

async def get_or_init_client(admin_user_id: int, api_id: str, api_hash: str, phone: str):
    """
    Получить или инициализировать Telegram клиент для конкретного админа
    
    Args:
        admin_user_id: User ID админа
        api_id: API ID из my.telegram.org
        api_hash: API Hash из my.telegram.org
        phone: Номер телефона админа (формат: +79001234567)
    
    Returns:
        TelegramClient или None если не удалось инициализировать
    """
    if not TELETHON_AVAILABLE:
        logger.warning("Telethon недоступен")
        return None
    
    # Если клиент уже существует и подключен - возвращаем его
    if admin_user_id in _clients:
        client = _clients[admin_user_id]
        if client.is_connected():
            return client
        else:
            # Переподключаемся
            try:
                await client.connect()
                if await client.is_user_authorized():
                    return client
            except:
                pass
    
    # Создаем новый клиент
    try:
        api_id_int = int(api_id) if api_id else None
        if not api_id_int or not api_hash or not phone:
            logger.warning(f"Не указаны API_ID, API_HASH или PHONE для админа {admin_user_id}")
            return None
        
        # Уникальное имя файла сессии для каждого админа
        session_file = f"telegram_session_{admin_user_id}"
        
        client = TelegramClient(session_file, api_id_int, api_hash)
        await client.connect()
        
        # Проверяем, авторизован ли клиент
        if not await client.is_user_authorized():
            logger.warning(f"Telegram клиент для админа {admin_user_id} не авторизован. Нужно отправить код подтверждения.")
            # Отправляем код
            await client.send_code_request(phone)
            # В реальном использовании нужно будет запросить код у админа
            # Пока возвращаем None
            await client.disconnect()
            return None
        
        # Сохраняем клиент
        _clients[admin_user_id] = client
        logger.info(f"Telegram клиент для админа {admin_user_id} успешно инициализирован")
        return client
        
    except Exception as e:
        logger.error(f"Ошибка инициализации Telegram клиента для админа {admin_user_id}: {e}")
        return None

async def init_telegram_client(api_id: str, api_hash: str, phone: str, session_file: str = "telegram_session"):
    """
    Инициализация Telegram клиента (устаревшая функция, используйте get_or_init_client)
    
    Args:
        api_id: API ID из my.telegram.org
        api_hash: API Hash из my.telegram.org
        phone: Номер телефона админа (формат: +79001234567)
        session_file: Имя файла для сохранения сессии
    
    Returns:
        TelegramClient или None если не удалось инициализировать
    """
    logger.warning("init_telegram_client устарела, используйте get_or_init_client")
    # Для обратной совместимости создаем временный клиент
    if not TELETHON_AVAILABLE:
        logger.warning("Telethon недоступен")
        return None
    
    try:
        api_id_int = int(api_id) if api_id else None
        if not api_id_int or not api_hash or not phone:
            logger.warning("Не указаны API_ID, API_HASH или PHONE для Telegram Client")
            return None
        
        client = TelegramClient(session_file, api_id_int, api_hash)
        await client.connect()
        
        # Проверяем, авторизован ли клиент
        if not await client.is_user_authorized():
            logger.warning("Telegram клиент не авторизован. Нужно отправить код подтверждения.")
            await client.send_code_request(phone)
            await client.disconnect()
            return None
        
        logger.info("Telegram клиент успешно инициализирован")
        return client
        
    except Exception as e:
        logger.error(f"Ошибка инициализации Telegram клиента: {e}")
        return None

async def get_username_by_phone(phone_number: str, admin_user_id: int = None, client: TelegramClient = None) -> Optional[str]:
    """
    Получить username по номеру телефона
    
    Args:
        phone_number: Номер телефона (формат: +79001234567, 89001234567, 79001234567)
        admin_user_id: User ID админа (для использования его клиента)
        client: TelegramClient для использования (если передан, используется он)
    
    Returns:
        Username (без @) или None если не найден
    """
    # Определяем какой клиент использовать
    if client:
        telegram_client = client
    elif admin_user_id and admin_user_id in _clients:
        telegram_client = _clients[admin_user_id]
    else:
        logger.warning("Telegram клиент недоступен")
        return None
    
    if not TELETHON_AVAILABLE or not telegram_client:
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
        result = await telegram_client.import_contacts([{
            'phone': phone_normalized,
            'first_name': 'Temp',
            'last_name': 'User'
        }])
        
        if result.imported and len(result.imported) > 0:
            user_id = result.imported[0].user_id
            # Получаем информацию о пользователе
            user = await telegram_client.get_entity(user_id)
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

async def close_client(admin_user_id: int = None):
    """Закрыть соединение с Telegram для конкретного админа или всех"""
    global _clients
    if admin_user_id:
        if admin_user_id in _clients:
            await _clients[admin_user_id].disconnect()
            del _clients[admin_user_id]
    else:
        # Закрываем все клиенты
        for client in _clients.values():
            try:
                await client.disconnect()
            except:
                pass
        _clients.clear()

