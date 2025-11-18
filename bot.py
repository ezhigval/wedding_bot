import asyncio
import logging
from aiogram import Bot, Dispatcher, F
from aiogram.types import Message, CallbackQuery, FSInputFile
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.fsm.state import State, StatesGroup
from aiogram.fsm.storage.memory import MemoryStorage

from config import BOT_TOKEN, GROOM_NAME, BRIDE_NAME, PHOTO_PATH, ADMIN_USER_ID, WEBAPP_URL
from utils import get_time_until_wedding, format_wedding_date
from database import (
    init_db, add_guest, get_guest, get_all_guests, get_guests_count,
    get_name_by_username, add_name_mapping, get_all_name_mappings, delete_name_mapping,
    init_default_mappings
)
from keyboards import get_invitation_keyboard, get_registration_keyboard, get_admin_keyboard

# Настройка логирования
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Инициализация бота и диспетчера
# Проверяем токен перед созданием бота
if BOT_TOKEN:
    # Дополнительная очистка токена
    BOT_TOKEN = BOT_TOKEN.strip().strip('"').strip("'")
    bot = Bot(token=BOT_TOKEN)
else:
    bot = None
dp = Dispatcher(storage=MemoryStorage())

# Состояния для регистрации
class RegistrationStates(StatesGroup):
    waiting_first_name = State()
    waiting_last_name = State()
    confirming = State()

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
    
    # Проверяем, зарегистрирован ли уже пользователь
    guest = await get_guest(message.from_user.id)
    
    if guest:
        first_name, last_name, confirmed_at = guest
        await message.answer(
            f"👋 Привет, {display_name}!\n\n"
            f"Ты уже подтвердил(а) своё присутствие на нашей свадьбе! 🎉\n"
            f"Дата подтверждения: {confirmed_at}"
        )
        await send_invitation_card(message)
    else:
        await message.answer(f"👋 Привет, {display_name}!")
        await send_invitation_card(message)

async def send_invitation_card(message: Message):
    """Отправляет красивую открытку-приглашение"""
    wedding_text = f"""
💒 <b>Свадьба</b>

👫 <b>{GROOM_NAME} и {BRIDE_NAME}</b>

📅 <b>{format_wedding_date()}</b>

{get_time_until_wedding()}

━━━━━━━━━━━━━━━━━━━━
Мы будем рады видеть вас на нашем торжестве! 
Этот день будет особенным, и ваше присутствие сделает его ещё более незабываемым! 💕
━━━━━━━━━━━━━━━━━━━━
"""
    
    # Пытаемся отправить фото, если оно есть
    try:
        photo = FSInputFile(PHOTO_PATH)
        await message.answer_photo(
            photo=photo,
            caption=wedding_text,
            reply_markup=get_invitation_keyboard(),
            parse_mode="HTML"
        )
    except (FileNotFoundError, Exception) as e:
        # Если фото нет или произошла ошибка, отправляем только текст
        logger.warning(f"Не удалось отправить фото: {e}")
        await message.answer(
            wedding_text,
            reply_markup=get_invitation_keyboard(),
            parse_mode="HTML"
        )

@dp.callback_query(F.data == "confirm_attendance")
async def process_confirm_attendance(callback: CallbackQuery, state: FSMContext):
    """Обработчик нажатия на кнопку 'Приду'"""
    await callback.answer()
    
    # Проверяем, не зарегистрирован ли уже
    guest = await get_guest(callback.from_user.id)
    if guest:
        await callback.message.answer(
            "✅ Ты уже подтвердил(а) своё присутствие! 🎉"
        )
        return
    
    await callback.message.answer(
        "🎉 Отлично! Мы очень рады, что ты сможешь быть с нами!\n\n"
        "Пожалуйста, заполни небольшую форму:"
    )
    await callback.message.answer(
        "👤 Введите ваше <b>имя</b>:",
        reply_markup=get_registration_keyboard(),
        parse_mode="HTML"
    )
    await state.set_state(RegistrationStates.waiting_first_name)

@dp.message(RegistrationStates.waiting_first_name)
async def process_first_name(message: Message, state: FSMContext):
    """Обработка ввода имени"""
    first_name = message.text.strip()
    
    if len(first_name) < 2:
        await message.answer("❌ Имя слишком короткое. Пожалуйста, введите корректное имя:")
        return
    
    await state.update_data(first_name=first_name)
    await message.answer(
        "👤 Введите вашу <b>фамилию</b>:",
        reply_markup=get_registration_keyboard(),
        parse_mode="HTML"
    )
    await state.set_state(RegistrationStates.waiting_last_name)

@dp.message(RegistrationStates.waiting_last_name)
async def process_last_name(message: Message, state: FSMContext):
    """Обработка ввода фамилии"""
    last_name = message.text.strip()
    
    if len(last_name) < 2:
        await message.answer("❌ Фамилия слишком короткая. Пожалуйста, введите корректную фамилию:")
        return
    
    data = await state.get_data()
    first_name = data.get("first_name")
    
    # Сохраняем гостя в базу данных
    await add_guest(
        user_id=message.from_user.id,
        first_name=first_name,
        last_name=last_name,
        username=message.from_user.username
    )
    
    guests_count = await get_guests_count()
    
    await message.answer(
        f"✅ <b>Спасибо, {first_name} {last_name}!</b>\n\n"
        f"Твоё присутствие подтверждено! 🎉\n\n"
        f"Мы будем рады видеть тебя на нашей свадьбе!\n"
        f"Всего подтвердили: {guests_count} гостей",
        parse_mode="HTML"
    )
    
    await state.clear()

@dp.callback_query(F.data == "cancel_registration")
async def process_cancel_registration(callback: CallbackQuery, state: FSMContext):
    """Обработчик отмены регистрации"""
    await callback.answer("Регистрация отменена")
    await state.clear()
    await callback.message.answer("Регистрация отменена. Если передумаешь, просто нажми /start")

@dp.message(Command("guests"))
async def cmd_guests(message: Message):
    """Команда для просмотра списка гостей (только для администраторов)"""
    # Здесь можно добавить проверку на администратора
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

/start - Получить приглашение и зарегистрироваться
/invite - Отправить приглашение еще раз
/guests - Посмотреть список гостей (для организаторов)
/help - Показать эту справку

💡 Просто нажми /start, чтобы начать!
"""
    await message.answer(help_text, parse_mode="HTML")

def is_admin(user_id):
    """Проверка, является ли пользователь администратором"""
    if not ADMIN_USER_ID:
        return False
    return str(user_id) == str(ADMIN_USER_ID)

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

async def init_bot():
    """Инициализация бота"""
    # Проверка токена
    token = BOT_TOKEN.strip().strip('"').strip("'") if BOT_TOKEN else ""
    
    if not token or len(token) < 10:
        logger.error("❌ ОШИБКА: BOT_TOKEN не установлен или неверный!")
        logger.error("Пожалуйста, проверьте переменную окружения BOT_TOKEN на Render")
        logger.error("Токен должен быть БЕЗ пробелов и кавычек")
        logger.error("Смотрите инструкцию в файле RENDER_FIX.md")
        return False
    
    # Проверяем формат токена (должен содержать :)
    if ':' not in token:
        logger.error("❌ ОШИБКА: BOT_TOKEN имеет неверный формат!")
        logger.error("Токен должен быть в формате: 1234567890:ABC...")
        return False
    
    # Инициализация базы данных
    await init_db()
    await init_default_mappings()  # Инициализация начальных соответствий
    logger.info("✅ База данных инициализирована")
    
    logger.info("✅ Бот инициализирован")
    logger.info(f"💒 Свадьба: {GROOM_NAME} и {BRIDE_NAME}")
    logger.info(f"📅 Дата: {format_wedding_date()}")
    logger.info(f"🌐 Mini App URL: {WEBAPP_URL}")
    
    return True

async def main():
    """Главная функция (для запуска только бота)"""
    await init_bot()
    logger.info("🚀 Бот запущен и готов к работе!")
    await dp.start_polling(bot)

if __name__ == "__main__":
    asyncio.run(main())

