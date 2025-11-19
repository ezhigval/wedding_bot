import asyncio
import logging
from aiogram import Bot, Dispatcher, F
from aiogram.types import Message, CallbackQuery, FSInputFile, WebAppInfo, InlineKeyboardButton, InlineKeyboardMarkup
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.fsm.state import State, StatesGroup
from aiogram.fsm.storage.memory import MemoryStorage

from config import BOT_TOKEN, GROOM_NAME, BRIDE_NAME, PHOTO_PATH, ADMIN_USER_ID, WEBAPP_URL, WEDDING_ADDRESS, ADMINS_FILE, ADMINS_LIST, GROUP_LINK, GROUP_ID
import json
import os
from utils import format_wedding_date
from keyboards import get_invitation_keyboard, get_admin_keyboard, get_send_invitation_keyboard, get_group_management_keyboard
from google_sheets import (
    get_invitations_list, normalize_telegram_id, get_admins_list, save_admin_to_sheets,
    get_all_guests_from_sheets, get_guests_count_from_sheets, cancel_guest_registration_by_user_id
)

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

# Состояния для управления группой
class GroupManagementStates(StatesGroup):
    waiting_message = State()
    waiting_add_member = State()
    waiting_remove_member = State()

async def get_user_display_name(user):
    """Получает имя пользователя из Telegram"""
    # Используем имя из Telegram (name_mapping больше не используется, все в Google Sheets)
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
    
    guests_count = await get_guests_count_from_sheets()
    guests = await get_all_guests_from_sheets()
    
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
    
    # Добавляем кнопку "Вернуться"
    from keyboards import get_admin_keyboard
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
    ])
    
    await callback.message.answer(stats_text, reply_markup=keyboard, parse_mode="HTML")
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
    for i, guest in enumerate(guests, 1):
        first_name = guest.get('first_name', '')
        last_name = guest.get('last_name', '')
        username = guest.get('username', '')
        username_text = f" (@{username})" if username else ""
        guests_text += f"{i}. {first_name} {last_name}{username_text}\n"
    
    guests_text += f"\n<b>Всего: {len(guests)} гостей</b>"
    
    # Добавляем кнопку "Вернуться"
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
    ])
    
    await callback.message.answer(guests_text, reply_markup=keyboard, parse_mode="HTML")
    await callback.answer()

@dp.callback_query(F.data == "admin_reload")
async def admin_reload(callback: CallbackQuery):
    """Перезагрузка Mini App (информационное сообщение)"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
    ])
    
    await callback.message.answer(
        f"🔄 <b>Информация о Mini App</b>\n\n"
        f"Mini App работает автоматически.\n"
        f"Для обновления контента:\n"
        f"1. Измените файлы в папке webapp/\n"
        f"2. Перезапустите сервер командой /restart (если доступно)\n\n"
        f"🌐 URL: {WEBAPP_URL}",
        reply_markup=keyboard,
        parse_mode="HTML"
    )
    await callback.answer("✅ Информация отправлена")

# Команды name_mapping удалены - все данные теперь в Google Sheets

@dp.callback_query(F.data == "admin_back")
async def admin_back(callback: CallbackQuery):
    """Возврат в главное меню админа"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.message.answer(
        "👋 <b>Главное меню</b>\n\n"
        "Выберите действие:",
        reply_markup=get_admin_keyboard(),
        parse_mode="HTML"
    )
    await callback.answer()

@dp.callback_query(F.data == "admin_names")
async def admin_names(callback: CallbackQuery):
    """Управление таблицей соответствия имен"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
    ])
    
    text = (
        "📋 <b>Управление именами</b>\n\n"
        "Все данные теперь хранятся в Google Sheets.\n\n"
        "Таблица соответствия имен больше не используется.\n"
        "Имена гостей берутся из Google Sheets или из Telegram профиля."
    )
    
    await callback.message.answer(text, reply_markup=keyboard, parse_mode="HTML")
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
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])
        
        await callback.message.answer(
            "❌ <b>Список приглашений пуст</b>\n\n"
            "Проверьте вкладку 'Пригласительные' в Google Sheets.\n"
            "Убедитесь, что:\n"
            "• Столбец A содержит имена гостей\n"
            "• Столбец B содержит телеграм ID (формат: @username, t.me/username или просто username)",
            reply_markup=keyboard,
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
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
    ])
    
    await callback.message.answer(guests_list, reply_markup=keyboard, parse_mode="HTML")
    
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
    
    # Создаем клавиатуру с кнопкой для отправки приглашения
    bot_invite_keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(
            text="💒 Открыть приглашение",
            web_app=WebAppInfo(url=WEBAPP_URL)
        )]
    ])
    
    # Отправляем сообщение с текстом приглашения и кнопками
    await message.answer(
        f"💌 <b>Приглашение для {guest_name}</b>\n\n"
        f"📋 <b>Готовый текст:</b>\n"
        f"<code>{invitation_text}</code>\n\n"
        f"📱 Телеграм: @{telegram_id}\n\n"
        f"<b>Инструкция:</b>\n"
        f"1. Нажмите '💬 Открыть диалог' - откроется диалог с @{telegram_id}\n"
        f"2. Скопируйте текст выше и отправьте гостю\n"
        f"3. Добавьте кнопку '💒 Открыть приглашение' (используйте кнопку ниже)",
        reply_markup=keyboard,
        parse_mode="HTML"
    )
    
    # Отправляем отдельное сообщение с кнопкой для примера
    await message.answer(
        f"📋 <b>Пример кнопки для гостя:</b>\n\n"
        f"Используйте эту кнопку в сообщении гостю:",
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
    
    # Отменяем регистрацию администратора в Google Sheets
    await cancel_guest_registration_by_user_id(callback.from_user.id)
    
    await callback.message.answer(
        "✅ <b>Данные сброшены!</b>\n\n"
        "Ваша регистрация удалена из базы данных.\n"
        "Теперь вы можете пройти весь путь заново, нажав /start",
        parse_mode="HTML"
    )
    await callback.answer("✅ Данные сброшены!")

# ========== ОБРАБОТЧИКИ УПРАВЛЕНИЯ ГРУППОЙ ==========

@dp.callback_query(F.data == "admin_group")
async def admin_group_menu(callback: CallbackQuery):
    """Меню управления группой"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    if not GROUP_ID:
        await callback.answer(
            "❌ GROUP_ID не настроен в конфигурации!\n\n"
            "Добавьте GROUP_ID в переменные окружения.",
            show_alert=True
        )
        return
    
    keyboard = get_group_management_keyboard()
    
    await callback.message.answer(
        f"💬 <b>Управление группой</b>\n\n"
        f"🔗 Ссылка: {GROUP_LINK}\n"
        f"🆔 ID группы: <code>{GROUP_ID}</code>\n\n"
        f"Выберите действие:",
        reply_markup=keyboard,
        parse_mode="HTML"
    )
    await callback.answer()

@dp.callback_query(F.data == "group_send_message")
async def group_send_message_start(callback: CallbackQuery, state: FSMContext):
    """Начало процесса отправки сообщения в группу"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    if not GROUP_ID:
        await callback.answer("❌ GROUP_ID не настроен", show_alert=True)
        return
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_group")]
    ])
    
    await callback.message.answer(
        "📢 <b>Отправка сообщения в группу</b>\n\n"
        "Введите текст сообщения, которое будет отправлено в группу от имени группы:",
        reply_markup=keyboard,
        parse_mode="HTML"
    )
    
    await state.set_state(GroupManagementStates.waiting_message)
    await callback.answer()

@dp.message(GroupManagementStates.waiting_message)
async def process_group_message(message: Message, state: FSMContext):
    """Обработка сообщения для отправки в группу"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        await state.clear()
        return
    
    if not GROUP_ID:
        await message.answer("❌ GROUP_ID не настроен в конфигурации.")
        await state.clear()
        return
    
    try:
        # Отправляем сообщение в группу
        # Бот должен быть администратором группы с правами на отправку сообщений от имени группы
        await bot.send_message(
            chat_id=GROUP_ID,
            text=message.text,
            parse_mode="HTML"
        )
        
        await message.answer(
            f"✅ <b>Сообщение отправлено в группу!</b>\n\n"
            f"📝 Текст:\n<code>{message.text}</code>",
            parse_mode="HTML"
        )
        
        logger.info(f"Админ {message.from_user.id} отправил сообщение в группу {GROUP_ID}")
    except Exception as e:
        error_msg = str(e)
        logger.error(f"Ошибка отправки сообщения в группу: {e}")
        
        if "chat not found" in error_msg.lower():
            await message.answer(
                "❌ <b>Группа не найдена!</b>\n\n"
                "Проверьте, что:\n"
                "1. Бот добавлен в группу\n"
                "2. GROUP_ID указан правильно\n"
                "3. Бот является администратором группы",
                parse_mode="HTML"
            )
        elif "not enough rights" in error_msg.lower() or "rights" in error_msg.lower():
            await message.answer(
                "❌ <b>Недостаточно прав!</b>\n\n"
                "Бот должен быть администратором группы с правами на отправку сообщений.",
                parse_mode="HTML"
            )
        else:
            await message.answer(
                f"❌ <b>Ошибка отправки сообщения:</b>\n\n"
                f"<code>{error_msg}</code>",
                parse_mode="HTML"
            )
    
    await state.clear()

@dp.callback_query(F.data == "group_add_member")
async def group_add_member_start(callback: CallbackQuery, state: FSMContext):
    """Начало процесса добавления участника в группу"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    if not GROUP_ID:
        await callback.answer("❌ GROUP_ID не настроен", show_alert=True)
        return
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_group")]
    ])
    
    await callback.message.answer(
        "➕ <b>Добавление участника в группу</b>\n\n"
        "Введите username или user_id участника для добавления:\n\n"
        "Примеры:\n"
        "• <code>@username</code>\n"
        "• <code>123456789</code> (user_id)",
        reply_markup=keyboard,
        parse_mode="HTML"
    )
    
    await state.set_state(GroupManagementStates.waiting_add_member)
    await callback.answer()

@dp.message(GroupManagementStates.waiting_add_member)
async def process_group_add_member(message: Message, state: FSMContext):
    """Обработка добавления участника в группу"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        await state.clear()
        return
    
    if not GROUP_ID:
        await message.answer("❌ GROUP_ID не настроен в конфигурации.")
        await state.clear()
        return
    
    user_input = message.text.strip()
    
    # Парсим user_id или username
    user_id = None
    if user_input.startswith("@"):
        # Это username, нужно получить user_id (в реальности это сложнее, но для простоты попробуем)
        await message.answer(
            "⚠️ <b>Добавление по username требует дополнительных прав.</b>\n\n"
            "Пожалуйста, используйте user_id для добавления участника.\n"
            "User_id можно получить через @userinfobot",
            parse_mode="HTML"
        )
        await state.clear()
        return
    else:
        try:
            user_id = int(user_input)
        except ValueError:
            await message.answer(
                "❌ <b>Неверный формат!</b>\n\n"
                "Введите user_id (число) или @username",
                parse_mode="HTML"
            )
            return
    
    try:
        # Добавляем участника в группу
        await bot.unban_chat_member(
            chat_id=GROUP_ID,
            user_id=user_id,
            only_if_banned=True
        )
        
        # Приглашаем пользователя
        invite_link = GROUP_LINK
        await message.answer(
            f"✅ <b>Участник добавлен в группу!</b>\n\n"
            f"👤 User ID: <code>{user_id}</code>\n"
            f"🔗 Ссылка для приглашения: {invite_link}\n\n"
            f"Отправьте пользователю ссылку для вступления в группу.",
            parse_mode="HTML"
        )
        
        logger.info(f"Админ {message.from_user.id} добавил участника {user_id} в группу {GROUP_ID}")
    except Exception as e:
        error_msg = str(e)
        logger.error(f"Ошибка добавления участника в группу: {e}")
        
        if "chat not found" in error_msg.lower():
            await message.answer(
                "❌ <b>Группа не найдена!</b>\n\n"
                "Проверьте, что бот добавлен в группу и GROUP_ID указан правильно.",
                parse_mode="HTML"
            )
        elif "not enough rights" in error_msg.lower():
            await message.answer(
                "❌ <b>Недостаточно прав!</b>\n\n"
                "Бот должен быть администратором группы с правами на добавление участников.",
                parse_mode="HTML"
            )
        else:
            await message.answer(
                f"❌ <b>Ошибка добавления участника:</b>\n\n"
                f"<code>{error_msg}</code>",
                parse_mode="HTML"
            )
    
    await state.clear()

@dp.callback_query(F.data == "group_remove_member")
async def group_remove_member_start(callback: CallbackQuery, state: FSMContext):
    """Начало процесса удаления участника из группы"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    if not GROUP_ID:
        await callback.answer("❌ GROUP_ID не настроен", show_alert=True)
        return
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_group")]
    ])
    
    await callback.message.answer(
        "➖ <b>Удаление участника из группы</b>\n\n"
        "Введите user_id участника для удаления:\n\n"
        "Пример: <code>123456789</code>",
        reply_markup=keyboard,
        parse_mode="HTML"
    )
    
    await state.set_state(GroupManagementStates.waiting_remove_member)
    await callback.answer()

@dp.message(GroupManagementStates.waiting_remove_member)
async def process_group_remove_member(message: Message, state: FSMContext):
    """Обработка удаления участника из группы"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        await state.clear()
        return
    
    if not GROUP_ID:
        await message.answer("❌ GROUP_ID не настроен в конфигурации.")
        await state.clear()
        return
    
    user_input = message.text.strip()
    
    try:
        user_id = int(user_input)
    except ValueError:
        await message.answer(
            "❌ <b>Неверный формат!</b>\n\n"
            "Введите user_id (число)",
            parse_mode="HTML"
        )
        return
    
    try:
        # Удаляем участника из группы (баним)
        await bot.ban_chat_member(
            chat_id=GROUP_ID,
            user_id=user_id
        )
        
        await message.answer(
            f"✅ <b>Участник удален из группы!</b>\n\n"
            f"👤 User ID: <code>{user_id}</code>",
            parse_mode="HTML"
        )
        
        logger.info(f"Админ {message.from_user.id} удалил участника {user_id} из группы {GROUP_ID}")
    except Exception as e:
        error_msg = str(e)
        logger.error(f"Ошибка удаления участника из группы: {e}")
        
        if "chat not found" in error_msg.lower():
            await message.answer(
                "❌ <b>Группа не найдена!</b>\n\n"
                "Проверьте, что бот добавлен в группу и GROUP_ID указан правильно.",
                parse_mode="HTML"
            )
        elif "not enough rights" in error_msg.lower():
            await message.answer(
                "❌ <b>Недостаточно прав!</b>\n\n"
                "Бот должен быть администратором группы с правами на удаление участников.",
                parse_mode="HTML"
            )
        else:
            await message.answer(
                f"❌ <b>Ошибка удаления участника:</b>\n\n"
                f"<code>{error_msg}</code>",
                parse_mode="HTML"
            )
    
    await state.clear()

@dp.callback_query(F.data == "group_list_members")
async def group_list_members(callback: CallbackQuery):
    """Список участников группы"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    if not GROUP_ID:
        await callback.answer("❌ GROUP_ID не настроен", show_alert=True)
        return
    
    try:
        # Получаем информацию о группе
        chat = await bot.get_chat(chat_id=GROUP_ID)
        
        # Получаем количество участников
        member_count = chat.members_count if hasattr(chat, 'members_count') else "неизвестно"
        
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_group")]
        ])
        
        await callback.message.answer(
            f"👥 <b>Информация о группе</b>\n\n"
            f"📛 Название: {chat.title}\n"
            f"🆔 ID: <code>{GROUP_ID}</code>\n"
            f"👥 Участников: {member_count}\n"
            f"🔗 Ссылка: {GROUP_LINK}\n\n"
            f"<i>Для получения полного списка участников используйте сторонние боты или API.</i>",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
        
        logger.info(f"Админ {callback.from_user.id} запросил информацию о группе {GROUP_ID}")
    except Exception as e:
        error_msg = str(e)
        logger.error(f"Ошибка получения информации о группе: {e}")
        
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_group")]
        ])
        
        if "chat not found" in error_msg.lower():
            await callback.message.answer(
                "❌ <b>Группа не найдена!</b>\n\n"
                "Проверьте, что:\n"
                "1. Бот добавлен в группу\n"
                "2. GROUP_ID указан правильно",
                reply_markup=keyboard,
                parse_mode="HTML"
            )
        else:
            await callback.message.answer(
                f"❌ <b>Ошибка получения информации:</b>\n\n"
                f"<code>{error_msg}</code>",
                reply_markup=keyboard,
                parse_mode="HTML"
            )
    
    await callback.answer()

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
    
    # База данных больше не используется - все данные в Google Sheets
    logger.info("✅ Бот использует Google Sheets как единственный источник данных")
    
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

