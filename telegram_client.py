"""
Модуль для работы с Telegram Client API (Telethon)
Используется для поиска username по номеру телефона
"""
import logging
import asyncio
from typing import Optional, Tuple

logger = logging.getLogger(__name__)

# Пытаемся импортировать Telethon
try:
    from telethon import TelegramClient
    from telethon.errors import (
        SessionPasswordNeededError, 
        PhoneNumberInvalidError,
        PhoneCodeInvalidError,
        PhoneCodeExpiredError,
        PhoneCodeEmptyError,
        FloodWaitError,
        PhoneNumberFloodError
    )
    TELETHON_AVAILABLE = True
except ImportError:
    TELETHON_AVAILABLE = False
    logger.warning("Telethon не установлен. Поиск username по номеру телефона недоступен.")

# Словарь клиентов для каждого админа (ключ - user_id админа)
_clients = {}

# Словарь клиентов, ожидающих код подтверждения (ключ - user_id админа)
_pending_clients = {}

def normalize_admin_phone(phone: str) -> str:
    """
    Нормализует номер телефона админа для использования в Telethon
    Google Sheets не поддерживает символ +, поэтому номера хранятся как 79...
    Telethon требует формат +79...
    
    Args:
        phone: Номер телефона (может быть 79..., +79..., 89...)
    
    Returns:
        Номер телефона в формате +79...
    """
    if not phone:
        return phone
    
    phone = phone.strip()
    
    # Если начинается с 8, заменяем на +7
    if phone.startswith("8"):
        return "+7" + phone[1:]
    # Если начинается с 7 (без +), добавляем +
    elif phone.startswith("7"):
        return "+" + phone
    # Если уже начинается с +, оставляем как есть
    elif phone.startswith("+"):
        return phone
    # Иначе добавляем +
    else:
        return "+" + phone

async def get_or_init_client(admin_user_id: int, api_id: str, api_hash: str, phone: str):
    """
    Получить или инициализировать Telegram клиент для конкретного админа
    
    Args:
        admin_user_id: User ID админа
        api_id: API ID из my.telegram.org
        api_hash: API Hash из my.telegram.org
        phone: Номер телефона админа (формат: 79001234567 или +79001234567)
               Google Sheets хранит номера как 79... (без +), функция автоматически нормализует
    
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
        
        # Нормализуем номер телефона (Google Sheets хранит номера как 79..., а Telethon требует +79...)
        phone_normalized = normalize_admin_phone(phone)
        
        # Проверяем, авторизован ли клиент
        if not await client.is_user_authorized():
            logger.info(f"Telegram клиент для админа {admin_user_id} не авторизован. Отправляем код подтверждения...")
            try:
                # Отправляем код подтверждения
                code_request = await client.send_code_request(phone_normalized)
                # Сохраняем клиент в словарь ожидающих авторизации
                _pending_clients[admin_user_id] = {
                    'client': client,
                    'phone': phone_normalized,
                    'phone_code_hash': code_request.phone_code_hash
                }
                logger.info(f"Код подтверждения отправлен на номер {phone_normalized} для админа {admin_user_id}")
                # Возвращаем None, чтобы бот мог запросить код у админа
                return None
            except Exception as e:
                logger.error(f"Ошибка отправки кода подтверждения для админа {admin_user_id}: {e}")
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
        phone: Номер телефона админа (формат: 79001234567 или +79001234567)
               Google Sheets хранит номера как 79... (без +), функция автоматически нормализует
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
        
        # Нормализуем номер телефона (Google Sheets хранит номера как 79..., а Telethon требует +79...)
        phone_normalized = normalize_admin_phone(phone)
        
        # Проверяем, авторизован ли клиент
        if not await client.is_user_authorized():
            logger.warning("Telegram клиент не авторизован. Нужно отправить код подтверждения.")
            await client.send_code_request(phone_normalized)
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

async def authorize_with_code(admin_user_id: int, code: str) -> Tuple[bool, str]:
    """
    Авторизовать Telegram клиент с кодом подтверждения
    
    Args:
        admin_user_id: User ID админа
        code: Код подтверждения из Telegram
    
    Returns:
        (success: bool, message: str) - успех авторизации и сообщение
    """
    if not TELETHON_AVAILABLE:
        return False, "Telethon недоступен"
    
    if admin_user_id not in _pending_clients:
        return False, "Нет ожидающего авторизации клиента. Попробуйте использовать функцию поиска username по номеру телефона."
    
    try:
        pending = _pending_clients[admin_user_id]
        client = pending['client']
        phone_code_hash = pending['phone_code_hash']
        
        # Пытаемся авторизоваться с кодом
        try:
            await client.sign_in(phone=pending['phone'], code=code, phone_code_hash=phone_code_hash)
            
            # Проверяем, авторизован ли клиент
            if await client.is_user_authorized():
                # Перемещаем клиент в активные
                _clients[admin_user_id] = client
                del _pending_clients[admin_user_id]
                logger.info(f"Telegram клиент для админа {admin_user_id} успешно авторизован")
                return True, "✅ Авторизация успешна! Теперь можно использовать поиск username по номеру телефона."
            else:
                return False, "❌ Авторизация не удалась. Проверьте код и попробуйте снова."
        except PhoneCodeInvalidError:
            # Неверный код
            logger.warning(f"Неверный код подтверждения для админа {admin_user_id}")
            return False, "INVALID_CODE"  # Специальный код для неверного кода
        except PhoneCodeExpiredError:
            # Код устарел
            logger.warning(f"Код подтверждения устарел для админа {admin_user_id}")
            return False, "EXPIRED_CODE"  # Специальный код для устаревшего кода
        except PhoneCodeEmptyError:
            # Пустой код
            logger.warning(f"Пустой код подтверждения для админа {admin_user_id}")
            return False, "EMPTY_CODE"  # Специальный код для пустого кода
        except SessionPasswordNeededError:
            # Требуется пароль 2FA
            logger.info(f"Для админа {admin_user_id} требуется пароль 2FA")
            return False, "2FA_PASSWORD_REQUIRED"  # Специальный код для запроса пароля
        except Exception as e:
            error_msg = str(e).lower()
            if "code" in error_msg and ("invalid" in error_msg or "wrong" in error_msg):
                logger.warning(f"Неверный код подтверждения для админа {admin_user_id}: {e}")
                return False, "INVALID_CODE"
            elif "code" in error_msg and ("expired" in error_msg or "timeout" in error_msg):
                logger.warning(f"Код подтверждения устарел для админа {admin_user_id}: {e}")
                return False, "EXPIRED_CODE"
            else:
                logger.error(f"Ошибка авторизации с кодом для админа {admin_user_id}: {e}")
                return False, f"❌ Ошибка авторизации: {str(e)}"
    except Exception as e:
        logger.error(f"Ошибка при авторизации клиента для админа {admin_user_id}: {e}")
        return False, f"❌ Ошибка: {str(e)}"

async def resend_code(admin_user_id: int) -> Tuple[bool, str]:
    """
    Повторно отправить код подтверждения
    
    Args:
        admin_user_id: User ID админа
    
    Returns:
        (success: bool, message: str) - успех отправки и сообщение
    """
    if not TELETHON_AVAILABLE:
        return False, "Telethon недоступен"
    
    if admin_user_id not in _pending_clients:
        return False, "Нет ожидающего авторизации клиента. Начните процесс авторизации заново."
    
    try:
        pending = _pending_clients[admin_user_id]
        client = pending['client']
        phone = pending['phone']
        
        # Отправляем новый код
        code_request = await client.send_code_request(phone)
        
        # Обновляем phone_code_hash
        pending['phone_code_hash'] = code_request.phone_code_hash
        
        logger.info(f"Новый код подтверждения отправлен для админа {admin_user_id}")
        return True, "✅ Новый код подтверждения отправлен в ваш Telegram"
    except FloodWaitError as e:
        # Ограничение частоты запросов
        wait_seconds = e.seconds
        wait_minutes = wait_seconds // 60
        wait_seconds_remainder = wait_seconds % 60
        
        if wait_minutes > 0:
            wait_time = f"{wait_minutes} мин. {wait_seconds_remainder} сек."
        else:
            wait_time = f"{wait_seconds} сек."
        
        logger.warning(f"Ограничение частоты запросов кода для админа {admin_user_id}: нужно подождать {wait_time}")
        return False, f"RATE_LIMIT:{wait_seconds}"  # Специальный код с временем ожидания
    except PhoneNumberFloodError:
        # Все варианты отправки кода уже использованы
        logger.warning(f"Все варианты отправки кода использованы для админа {admin_user_id}")
        return False, "ALL_OPTIONS_USED"  # Специальный код
    except Exception as e:
        error_msg = str(e).lower()
        # Проверяем, не является ли это ошибкой о том, что все варианты использованы
        if "all available options" in error_msg or "already used" in error_msg:
            logger.warning(f"Все варианты отправки кода использованы для админа {admin_user_id}: {e}")
            return False, "ALL_OPTIONS_USED"
        else:
            logger.error(f"Ошибка повторной отправки кода для админа {admin_user_id}: {e}")
            return False, f"❌ Ошибка отправки кода: {str(e)}"

async def authorize_with_password(admin_user_id: int, password: str) -> Tuple[bool, str]:
    """
    Авторизовать Telegram клиент с паролем 2FA
    
    Args:
        admin_user_id: User ID админа
        password: Пароль двухфакторной аутентификации
    
    Returns:
        (success: bool, message: str) - успех авторизации и сообщение
    """
    if not TELETHON_AVAILABLE:
        return False, "Telethon недоступен"
    
    if admin_user_id not in _pending_clients:
        return False, "Нет ожидающего авторизации клиента"
    
    try:
        pending = _pending_clients[admin_user_id]
        client = pending['client']
        
        # Авторизуемся с паролем
        await client.sign_in(password=password)
        
        # Проверяем, авторизован ли клиент
        if await client.is_user_authorized():
            # Перемещаем клиент в активные
            _clients[admin_user_id] = client
            del _pending_clients[admin_user_id]
            logger.info(f"Telegram клиент для админа {admin_user_id} успешно авторизован с паролем 2FA")
            return True, "✅ Авторизация успешна! Теперь можно использовать поиск username по номеру телефона."
        else:
            return False, "❌ Авторизация не удалась. Проверьте пароль и попробуйте снова."
    except Exception as e:
        logger.error(f"Ошибка авторизации с паролем для админа {admin_user_id}: {e}")
        return False, f"❌ Ошибка авторизации: {str(e)}"

async def close_client(admin_user_id: int = None):
    """Закрыть соединение с Telegram для конкретного админа или всех"""
    global _clients, _pending_clients
    if admin_user_id:
        if admin_user_id in _clients:
            await _clients[admin_user_id].disconnect()
            del _clients[admin_user_id]
        if admin_user_id in _pending_clients:
            await _pending_clients[admin_user_id]['client'].disconnect()
            del _pending_clients[admin_user_id]
    else:
        # Закрываем все клиенты
        for client in _clients.values():
            try:
                await client.disconnect()
            except:
                pass
        for pending in _pending_clients.values():
            try:
                await pending['client'].disconnect()
            except:
                pass
        _clients.clear()
        _pending_clients.clear()

