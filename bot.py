import asyncio
import logging
from aiogram import Bot, Dispatcher, F
from aiogram.types import Message, CallbackQuery, FSInputFile, WebAppInfo, InlineKeyboardButton, InlineKeyboardMarkup
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.fsm.state import State, StatesGroup
from aiogram.fsm.storage.memory import MemoryStorage

from config import BOT_TOKEN, GROOM_NAME, BRIDE_NAME, PHOTO_PATH, ADMIN_USER_ID, WEBAPP_URL, WEDDING_ADDRESS, ADMINS_FILE, ADMINS_LIST
import json
import os
from utils import format_wedding_date
from database import (
    init_db, get_all_guests, get_guests_count,
    get_name_by_username, add_name_mapping, get_all_name_mappings, delete_name_mapping,
    init_default_mappings, delete_guest
)
from keyboards import get_invitation_keyboard, get_admin_keyboard, get_send_invitation_keyboard
from google_sheets import get_invitations_list, normalize_telegram_id, get_admins_list, save_admin_to_sheets

# Настройка логирования
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Инициализация диспетчера
dp = Dispatcher(storage=MemoryStorage())

# Бот будет создан в init_bot() после проверки токена
bot = None

# Состояния для регистрации (больше не используются, оставлены для совместимости)
class RegistrationStates(StatesGroup):
    waiting_first_name = State()
    waiting_last_name = State()
    confirming = State()

# Состояния для рассылки приглашений
class InvitationStates(StatesGroup):
    waiting_guest_selection = State()

async def get_user_display_name(user):
    """Получает имя пользователя из таблицы соответствия или из Telegram"""
    # Сначала проверяем таблицу соответствия
    if user.username:
        mapped_name = await get_name_by_username(user.username)
        if mapped_name:
            first_name, last_name = mapped_name
            return f"{first_name} {last_name}"
    
    # Если нет в таблице, используем имя из Telegram
    if user.first_name:
        if user.last_name:
            return f"{user.first_name} {user.last_name}"
        return user.first_name
    
    return "друг"  # Fallback

@dp.message(Command("start"))
async def cmd_start(message: Message, state: FSMContext):
    """Обработчик команды /start"""
    await state.clear()
    
    # Получаем имя из таблицы соответствия или Telegram
    display_name = await get_user_display_name(message.from_user)
    
    # Отправляем приветственное сообщение с фото
    try:
        photo = FSInputFile(PHOTO_PATH)
        await message.answer_photo(
            photo=photo,
            caption=f"👋 Привет, {display_name}!",
            parse_mode="HTML"
        )
    except (FileNotFoundError, Exception) as e:
        # Если фото нет или произошла ошибка, отправляем только текст
        logger.warning(f"Не удалось отправить фото в приветствии: {e}")
        await message.answer(f"👋 Привет, {display_name}!")
    
    # Отправляем приглашение
    await send_invitation_card(message)

async def send_invitation_card(message: Message):
    """Отправляет красивую открытку-приглашение"""
    wedding_text = f"""
💒 <b>Свадьба</b>

👫 <b>{GROOM_NAME} и {BRIDE_NAME}</b>

📅 <b>{format_wedding_date()}</b>

📍 <b>Адрес:</b> {WEDDING_ADDRESS}

━━━━━━━━━━━━━━━━━━━━
Мы будем рады видеть вас на нашем торжестве! 
Этот день будет особенным, и ваше присутствие сделает его ещё более незабываемым! 💕

Просим предварительно подтвердить ваше присутствие в приложении.
━━━━━━━━━━━━━━━━━━━━
"""
    
    await message.answer(
        wedding_text,
        reply_markup=get_invitation_keyboard(),
        parse_mode="HTML"
    )

# Все функции подтверждения присутствия перенесены в Mini App

@dp.message(Command("guests"))
async def cmd_guests(message: Message):
    """Команда для просмотра списка гостей (только для администраторов)"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        return
    
    guests = await get_all_guests()
    
    if not guests:
        await message.answer("Пока никто не подтвердил присутствие.")
        return
    
    guests_text = "📋 <b>Список гостей:</b>\n\n"
    for i, (first_name, last_name, username, confirmed_at) in enumerate(guests, 1):
        username_text = f" (@{username})" if username else ""
        guests_text += f"{i}. {first_name} {last_name}{username_text}\n"
    
    guests_text += f"\n<b>Всего: {len(guests)} гостей</b>"
    
    await message.answer(guests_text, parse_mode="HTML")

@dp.message(Command("invite"))
async def cmd_invite(message: Message):
    """Команда для отправки приглашения"""
    await send_invitation_card(message)

@dp.message(Command("help"))
async def cmd_help(message: Message):
    """Команда помощи"""
    help_text = """
📖 <b>Доступные команды:</b>

/start - Получить приглашение
/invite - Отправить приглашение еще раз
/guests - Посмотреть список гостей (для организаторов)
/help - Показать эту справку

💡 Просто нажми /start, чтобы получить приглашение!
💒 Все функции подтверждения присутствия доступны в Mini App.
"""
    await message.answer(help_text, parse_mode="HTML")

def load_admins():
    """Загрузка списка админов из Google Sheets (синхронная версия для использования в синхронном коде)"""
    # Используем синхронную версию из google_sheets
    try:
        from google_sheets import _get_admins_list_sync
        admins = _get_admins_list_sync()
        
        # Если Google Sheets недоступен или пуст, используем fallback
        if not admins:
            logger.warning("Не удалось загрузить админов из Google Sheets, используем fallback")
            # Сначала пробуем загрузить из файла
            admins = []
            try:
                if os.path.exists(ADMINS_FILE):
                    with open(ADMINS_FILE, 'r', encoding='utf-8') as f:
                        data = json.load(f)
                        file_admins = data.get('admins', [])
                        admins.extend(file_admins)
                        logger.info(f"Загружено {len(admins)} админов из файла")
            except Exception as file_error:
                logger.warning(f"Не удалось загрузить админов из файла: {file_error}")
            
            # Если в файле нет админов, используем env переменную
            if not admins:
                for username in ADMINS_LIST:
                    admins.append({
                        'username': username.lower(),
                        'name': username,
                        'telegram': username
                    })
        
        return admins
    except Exception as e:
        logger.error(f"Ошибка загрузки админов: {e}")
        # Fallback: сначала файл, потом env
        admins = []
        try:
            if os.path.exists(ADMINS_FILE):
                with open(ADMINS_FILE, 'r', encoding='utf-8') as f:
                    data = json.load(f)
                    file_admins = data.get('admins', [])
                    admins.extend(file_admins)
                    logger.info(f"Загружено {len(admins)} админов из файла (fallback)")
        except Exception as file_error:
            logger.warning(f"Не удалось загрузить админов из файла: {file_error}")
        
        # Если в файле нет админов, используем env переменную
        if not admins:
            for username in ADMINS_LIST:
                admins.append({
                    'username': username.lower(),
                    'name': username,
                    'telegram': username
                })
        return admins

async def save_admin_user_id(username, user_id):
    """Сохранение user_id админа в Google Sheets"""
    try:
        # Сохраняем в Google Sheets
        result = await save_admin_to_sheets(username, user_id)
        if result:
            logger.info(f"Сохранен user_id для админа {username} в Google Sheets: {user_id}")
            return True
        else:
            logger.warning(f"Не удалось сохранить админа {username} в Google Sheets, сохраняем в файл")
            # Fallback: сохраняем в файл
            try:
                if os.path.exists(ADMINS_FILE):
                    with open(ADMINS_FILE, 'r', encoding='utf-8') as f:
                        data = json.load(f)
                        admins = data.get('admins', [])
                else:
                    admins = []
                
                # Обновляем или добавляем админа
                found = False
                for admin in admins:
                    if admin.get('username') == username.lower():
                        admin['user_id'] = user_id
                        found = True
                        break
                
                if not found:
                    admins.append({
                        'username': username.lower(),
                        'user_id': user_id
                    })
                
                with open(ADMINS_FILE, 'w', encoding='utf-8') as f:
                    json.dump({'admins': admins}, f, ensure_ascii=False, indent=2)
                
                logger.info(f"Сохранен user_id для админа {username} в файл: {user_id}")
                return True
            except Exception as file_error:
                logger.error(f"Ошибка сохранения в файл: {file_error}")
                return False
    except Exception as e:
        logger.error(f"Ошибка сохранения user_id админа: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False

def get_admin_user_ids():
    """Получить список user_id всех админов"""
    admins = load_admins()
    user_ids = []
    for admin in admins:
        if 'user_id' in admin:
            user_ids.append(admin['user_id'])
    return user_ids

def is_admin(user_id):
    """Проверка, является ли пользователь администратором"""
    # Проверка по старому способу (ADMIN_USER_ID)
    if ADMIN_USER_ID and str(user_id) == str(ADMIN_USER_ID):
        return True
    
    # Проверка по новому способу (из файла)
    admin_ids = get_admin_user_ids()
    return user_id in admin_ids

async def notify_admins(message_text):
    """Отправка уведомления всем админам"""
    if bot is None:
        logger.warning("Бот не инициализирован, уведомление не отправлено")
        return
    
    admin_ids = get_admin_user_ids()
    
    if not admin_ids:
        logger.warning("Нет админов для отправки уведомлений")
        logger.info(f"Список админов из env: {ADMINS_LIST}")
        logger.info(f"Проверьте, что админы написали /start боту для сохранения их user_id")
        return
    
    logger.info(f"Отправка уведомления {len(admin_ids)} админам: {admin_ids}")
    
    for admin_id in admin_ids:
        try:
            await bot.send_message(admin_id, message_text, parse_mode="HTML")
            logger.info(f"Уведомление отправлено админу {admin_id}")
        except Exception as e:
            logger.error(f"Ошибка отправки уведомления админу {admin_id}: {e}")
            import traceback
            logger.error(traceback.format_exc())

@dp.message(Command("set_me_admins"))
async def cmd_set_me_admins(message: Message):
    """Команда для регистрации админа"""
    username = message.from_user.username
    
    if not username:
        await message.answer(
            "❌ <b>Ошибка</b>\n\n"
            "У вас не установлен username в Telegram.\n"
            "Пожалуйста, установите username в настройках Telegram и попробуйте снова.",
            parse_mode="HTML"
        )
        return
    
    username_lower = username.lower()
    
    # Загружаем список админов из Google Sheets
    try:
        admins_list = await get_admins_list()
        admin_usernames = [admin.get('username', '').lower() for admin in admins_list]
        
        # Проверяем, есть ли этот username в списке админов
        if username_lower not in admin_usernames:
            # Fallback: проверяем env переменную
            if username_lower not in [admin.lower() for admin in ADMINS_LIST]:
                await message.answer(
                    "❌ <b>Доступ запрещен</b>\n\n"
                    f"Ваш username (@{username}) не найден в списке администраторов.\n\n"
                    "Проверьте, что ваш username добавлен во вкладку 'Админ бота' в Google Sheets.",
                    parse_mode="HTML"
                )
                return
    except Exception as e:
        logger.error(f"Ошибка загрузки списка админов: {e}")
        # Fallback на env переменную
        if username_lower not in [admin.lower() for admin in ADMINS_LIST]:
            await message.answer(
                "❌ <b>Доступ запрещен</b>\n\n"
                f"Ваш username (@{username}) не найден в списке администраторов.",
                parse_mode="HTML"
            )
            return
    
    # Сохраняем user_id админа
    save_result = await save_admin_user_id(username_lower, message.from_user.id)
    
    if save_result:
        # Проверяем, что админ действительно сохранен
        # Обновляем список админов в памяти
        try:
            admins_list_after = await get_admins_list()
            admin_found = False
            for admin in admins_list_after:
                if admin.get('username') == username_lower and admin.get('user_id') == message.from_user.id:
                    admin_found = True
                    break
            
            if admin_found:
                await message.answer(
                    "✅ <b>Вы успешно зарегистрированы как администратор!</b>\n\n"
                    f"Username: @{username}\n"
                    f"User ID: {message.from_user.id}\n\n"
                    "Теперь вы будете получать уведомления о регистрациях гостей.\n\n"
                    "Используйте /admin для доступа к панели управления.",
                    parse_mode="HTML"
                )
                logger.info(f"Админ @{username} (user_id: {message.from_user.id}) зарегистрирован через /set_me_admins")
            else:
                await message.answer(
                    "⚠️ <b>Регистрация завершена, но требуется проверка</b>\n\n"
                    f"Username: @{username}\n"
                    f"User ID: {message.from_user.id}\n\n"
                    "Данные сохранены, но проверка доступа может занять несколько секунд.\n"
                    "Попробуйте использовать /admin через несколько секунд.\n\n"
                    "Если доступ не появится, проверьте логи на сервере.",
                    parse_mode="HTML"
                )
                logger.warning(f"Админ @{username} сохранен, но не найден при проверке")
        except Exception as e:
            logger.error(f"Ошибка проверки сохранения админа: {e}")
            await message.answer(
                "⚠️ <b>Регистрация завершена</b>\n\n"
                f"Username: @{username}\n"
                f"User ID: {message.from_user.id}\n\n"
                "Попробуйте использовать /admin. Если доступ не появится, проверьте логи.",
                parse_mode="HTML"
            )
    else:
        await message.answer(
            "❌ <b>Ошибка сохранения</b>\n\n"
            f"Не удалось сохранить данные в Google Sheets.\n\n"
            "Возможные причины:\n"
            "• Не настроены credentials для Google Sheets\n"
            "• Service account не имеет доступа к таблице\n"
            "• Вкладка 'Админ бота' не существует\n\n"
            "Проверьте логи на сервере для деталей.",
            parse_mode="HTML"
        )
        logger.error(f"Не удалось сохранить админа @{username} в Google Sheets")

@dp.message(Command("start"))
async def cmd_start(message: Message, state: FSMContext):
    """Обработчик команды /start"""
    await state.clear()
    
    # Получаем имя из таблицы соответствия или Telegram
    display_name = await get_user_display_name(message.from_user)
    
    # Отправляем приветственное сообщение с фото
    try:
        photo = FSInputFile(PHOTO_PATH)
        await message.answer_photo(
            photo=photo,
            caption=f"👋 Привет, {display_name}!",
            parse_mode="HTML"
        )
    except (FileNotFoundError, Exception) as e:
        # Если фото нет или произошла ошибка, отправляем только текст
        logger.warning(f"Не удалось отправить фото в приветствии: {e}")
        await message.answer(f"👋 Привет, {display_name}!")
    
    # Отправляем приглашение
    await send_invitation_card(message)

@dp.message(Command("admin"))
async def cmd_admin(message: Message):
    """Панель администратора"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        return
    
    admin_text = f"""
🔧 <b>Панель администратора</b>

💒 Свадьба: {GROOM_NAME} и {BRIDE_NAME}
📅 Дата: {format_wedding_date()}
🌐 Mini App: {WEBAPP_URL}

Используйте кнопки ниже для управления:
"""
    await message.answer(admin_text, reply_markup=get_admin_keyboard(), parse_mode="HTML")

@dp.callback_query(F.data == "admin_stats")
async def admin_stats(callback: CallbackQuery):
    """Статистика для администратора"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    guests_count = await get_guests_count()
    guests = await get_all_guests()
    
    stats_text = f"""
📊 <b>Статистика</b>

👥 Всего гостей: {guests_count}

📋 Последние регистрации:
"""
    for i, (first_name, last_name, username, confirmed_at) in enumerate(guests[:5], 1):
        username_text = f" (@{username})" if username else ""
        stats_text += f"{i}. {first_name} {last_name}{username_text}\n"
    
    if len(guests) > 5:
        stats_text += f"\n... и еще {len(guests) - 5} гостей"
    
    await callback.message.answer(stats_text, parse_mode="HTML")
    await callback.answer()

@dp.callback_query(F.data == "admin_guests")
async def admin_guests_list(callback: CallbackQuery):
    """Список гостей для администратора"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    guests = await get_all_guests()
    
    if not guests:
        await callback.message.answer("Пока никто не подтвердил присутствие.")
        await callback.answer()
        return
    
    guests_text = "📋 <b>Список всех гостей:</b>\n\n"
    for i, (first_name, last_name, username, confirmed_at) in enumerate(guests, 1):
        username_text = f" (@{username})" if username else ""
        guests_text += f"{i}. {first_name} {last_name}{username_text}\n"
    
    guests_text += f"\n<b>Всего: {len(guests)} гостей</b>"
    
    await callback.message.answer(guests_text, parse_mode="HTML")
    await callback.answer()

@dp.callback_query(F.data == "admin_reload")
async def admin_reload(callback: CallbackQuery):
    """Перезагрузка Mini App (информационное сообщение)"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.message.answer(
        f"🔄 <b>Информация о Mini App</b>\n\n"
        f"Mini App работает автоматически.\n"
        f"Для обновления контента:\n"
        f"1. Измените файлы в папке webapp/\n"
        f"2. Перезапустите сервер командой /restart (если доступно)\n\n"
        f"🌐 URL: {WEBAPP_URL}",
        parse_mode="HTML"
    )
    await callback.answer("✅ Информация отправлена")

@dp.message(Command("names"))
async def cmd_names(message: Message):
    """Управление таблицей соответствия имен (только для администратора)"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        return
    
    mappings = await get_all_name_mappings()
    
    if not mappings:
        await message.answer("📋 Таблица соответствия пуста.\n\nИспользуйте /addname для добавления.")
        return
    
    text = "📋 <b>Таблица соответствия username → имя:</b>\n\n"
    for username, first_name, last_name in mappings:
        text += f"@{username} → {first_name} {last_name}\n"
    
    text += "\n💡 Команды:\n"
    text += "/addname username имя фамилия - добавить\n"
    text += "/delname username - удалить\n"
    text += "/importnames - импорт из Google таблицы (скоро)"
    
    await message.answer(text, parse_mode="HTML")

# Состояния для добавления имени
class NameMappingStates(StatesGroup):
    waiting_username = State()
    waiting_name = State()

@dp.message(Command("addname"))
async def cmd_addname(message: Message, state: FSMContext):
    """Добавление соответствия username → имя"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        return
    
    # Парсим команду: /addname username имя фамилия
    parts = message.text.split(maxsplit=3)
    if len(parts) >= 4:
        username = parts[1].replace('@', '').strip()
        first_name = parts[2].strip()
        last_name = parts[3].strip()
        
        await add_name_mapping(username, first_name, last_name)
        await message.answer(
            f"✅ Добавлено: @{username} → {first_name} {last_name}"
        )
        await state.clear()
    else:
        await message.answer(
            "📝 Формат команды:\n"
            "/addname username имя фамилия\n\n"
            "Пример:\n"
            "/addname ezhigval Валентин Ежов"
        )

@dp.message(Command("delname"))
async def cmd_delname(message: Message):
    """Удаление соответствия username → имя"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        return
    
    parts = message.text.split(maxsplit=1)
    if len(parts) >= 2:
        username = parts[1].replace('@', '').strip()
        await delete_name_mapping(username)
        await message.answer(f"✅ Удалено соответствие для @{username}")
    else:
        await message.answer(
            "📝 Формат команды:\n"
            "/delname username\n\n"
            "Пример:\n"
            "/delname ezhigval"
        )

@dp.message(Command("importnames"))
async def cmd_importnames(message: Message):
    """Импорт из Google таблицы (будет реализовано позже)"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        return
    
    await message.answer(
        "📊 <b>Импорт из Google таблицы</b>\n\n"
        "Эта функция будет реализована позже.\n\n"
        "Планируемый формат:\n"
        "1. Подключение к Google Sheets API\n"
        "2. Импорт данных из таблицы\n"
        "3. Автоматическое обновление соответствий\n\n"
        "Пока используйте команду /addname для ручного добавления.",
        parse_mode="HTML"
    )

@dp.callback_query(F.data == "admin_names")
async def admin_names(callback: CallbackQuery):
    """Управление таблицей соответствия имен"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    mappings = await get_all_name_mappings()
    
    if not mappings:
        await callback.message.answer(
            "📋 Таблица соответствия пуста.\n\n"
            "Используйте /addname для добавления.\n"
            "Пример: /addname ezhigval Валентин Ежов"
        )
    else:
        text = "📋 <b>Таблица соответствия:</b>\n\n"
        for username, first_name, last_name in mappings:
            text += f"@{username} → {first_name} {last_name}\n"
        
        text += "\n💡 Команды:\n"
        text += "/addname username имя фамилия\n"
        text += "/delname username\n"
        text += "/names - полный список"
        
        await callback.message.answer(text, parse_mode="HTML")
    
    await callback.answer()

@dp.callback_query(F.data == "admin_send_invite")
async def admin_send_invite(callback: CallbackQuery, state: FSMContext):
    """Начать рассылку приглашений"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    # Загружаем список приглашений из Google Sheets
    invitations = await get_invitations_list()
    
    if not invitations:
        await callback.message.answer(
            "❌ <b>Список приглашений пуст</b>\n\n"
            "Проверьте вкладку 'Пригласительные' в Google Sheets.\n"
            "Убедитесь, что:\n"
            "• Столбец A содержит имена гостей\n"
            "• Столбец B содержит телеграм ID (формат: @username, t.me/username или просто username)",
            parse_mode="HTML"
        )
        return
    
    # Формируем список гостей
    guests_list = "📋 <b>Список гостей для приглашения:</b>\n\n"
    for i, inv in enumerate(invitations, 1):
        guests_list += f"{i}. {inv['name']} - @{inv['telegram_id']}\n"
    
    guests_list += "\n" + "=" * 40 + "\n\n"
    guests_list += (
        "💬 <b>Введите данные гостя для отправки приглашения:</b>\n\n"
        "Формат: <b>Имя Фамилия - @telegram_id</b>\n\n"
        "Пример:\n"
        "<code>Иван Иванов - @ivan_ivanov</code>\n\n"
        "Или просто скопируйте строку из списка выше."
    )
    
    await callback.message.answer(guests_list, parse_mode="HTML")
    
    # Устанавливаем состояние ожидания ввода
    await state.set_state(InvitationStates.waiting_guest_selection)

@dp.message(InvitationStates.waiting_guest_selection)
async def process_guest_selection(message: Message, state: FSMContext):
    """Обработка выбора гостя для отправки приглашения"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        await state.clear()
        return
    
    text = message.text.strip()
    
    # Парсим формат "Имя Фамилия - @telegram_id" или "Имя Фамилия - telegram_id"
    # Также поддерживаем просто "Имя Фамилия - @telegram_id" из списка
    parts = text.split(" - ", 1)
    
    if len(parts) != 2:
        await message.answer(
            "❌ <b>Неверный формат!</b>\n\n"
            "Используйте формат: <b>Имя Фамилия - @telegram_id</b>\n\n"
            "Пример:\n"
            "<code>Иван Иванов - @ivan_ivanov</code>",
            parse_mode="HTML"
        )
        return
    
    guest_name = parts[0].strip()
    telegram_id_raw = parts[1].strip()
    
    if not guest_name or not telegram_id_raw:
        await message.answer(
            "❌ <b>Неверный формат!</b>\n\n"
            "Имя и телеграм ID не могут быть пустыми.",
            parse_mode="HTML"
        )
        return
    
    # Нормализуем телеграм ID
    telegram_id = normalize_telegram_id(telegram_id_raw)
    
    if not telegram_id:
        await message.answer(
            "❌ <b>Неверный формат телеграм ID!</b>\n\n"
            "Поддерживаемые форматы:\n"
            "• @username\n"
            "• t.me/username\n"
            "• username",
            parse_mode="HTML"
        )
        return
    
    # Создаем текст приглашения
    invitation_text = (
        f"Дорогой(ая) {guest_name}, с большой радостью сообщаю - мы, {GROOM_NAME} и {BRIDE_NAME}, "
        f"женимся и приглашаем тебя на наш прекрасный праздник."
    )
    
    # Создаем клавиатуру с кнопкой для отправки
    keyboard, _ = get_send_invitation_keyboard(guest_name, telegram_id)
    
    # Отправляем сообщение с текстом и кнопкой
    await message.answer(
        f"💌 <b>Приглашение для {guest_name}</b>\n\n"
        f"{invitation_text}\n\n"
        f"📱 Телеграм: @{telegram_id}\n\n"
        f"Нажмите кнопку ниже, чтобы открыть диалог с этим человеком:",
        reply_markup=keyboard,
        parse_mode="HTML"
    )
    
    # Также отправляем готовый текст для копирования с кнопкой для бота
    # Создаем кнопку для открытия бота
    bot_invite_keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(
            text="💒 Открыть приглашение",
            web_app=WebAppInfo(url=WEBAPP_URL)
        )]
    ])
    
    await message.answer(
        f"📋 <b>Готовый текст для отправки гостю:</b>\n\n"
        f"<code>{invitation_text}</code>\n\n"
        f"После открытия диалога:\n"
        f"1. Скопируйте текст выше\n"
        f"2. Отправьте сообщение гостю\n"
        f"3. Добавьте кнопку 'Открыть приглашение' (используйте кнопку ниже как пример)",
        reply_markup=bot_invite_keyboard,
        parse_mode="HTML"
    )
    
    # Сбрасываем состояние
    await state.clear()

@dp.callback_query(F.data == "admin_reset_me")
async def admin_reset_me(callback: CallbackQuery):
    """Сброс данных регистрации администратора для тестирования"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    # Удаляем данные о регистрации администратора
    await delete_guest(callback.from_user.id)
    
    await callback.message.answer(
        "✅ <b>Данные сброшены!</b>\n\n"
        "Ваша регистрация удалена из базы данных.\n"
        "Теперь вы можете пройти весь путь заново, нажав /start",
        parse_mode="HTML"
    )
    await callback.answer("✅ Данные сброшены!")

async def init_bot():
    """Инициализация бота"""
    global bot
    
    # Проверка и очистка токена
    token = BOT_TOKEN.strip().strip('"').strip("'") if BOT_TOKEN else ""
    
    if not token or len(token) < 10:
        logger.error("❌ ОШИБКА: BOT_TOKEN не установлен или неверный!")
        logger.error("Пожалуйста, проверьте переменную окружения BOT_TOKEN на Render")
        logger.error("Токен должен быть БЕЗ пробелов и кавычек")
        logger.error("Смотрите инструкцию в файле TOKEN_FIX.md")
        return None
    
    # Проверяем формат токена (должен содержать :)
    if ':' not in token:
        logger.error("❌ ОШИБКА: BOT_TOKEN имеет неверный формат!")
        logger.error("Токен должен быть в формате: 1234567890:ABC...")
        return None
    
    # Создаем бота с очищенным токеном
    try:
        bot = Bot(token=token)
        logger.info("✅ Бот создан успешно")
    except Exception as e:
        logger.error(f"❌ Ошибка при создании бота: {e}")
        return None
    
    # Инициализация базы данных
    await init_db()
    await init_default_mappings()  # Инициализация начальных соответствий
    logger.info("✅ База данных инициализирована")
    
    logger.info("✅ Бот инициализирован")
    logger.info(f"💒 Свадьба: {GROOM_NAME} и {BRIDE_NAME}")
    logger.info(f"📅 Дата: {format_wedding_date()}")
    logger.info(f"🌐 Mini App URL: {WEBAPP_URL}")
    
    return bot

async def main():
    """Главная функция (для запуска только бота)"""
    bot_instance = await init_bot()
    if bot_instance is None:
        logger.error("❌ Не удалось инициализировать бота")
        return
    logger.info("🚀 Бот запущен и готов к работе!")
    await dp.start_polling(bot_instance)

if __name__ == "__main__":
    asyncio.run(main())

