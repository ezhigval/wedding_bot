import asyncio
import logging
from aiogram import Bot, Dispatcher, F
from aiogram.types import Message, CallbackQuery, FSInputFile, WebAppInfo, InlineKeyboardButton, InlineKeyboardMarkup
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.fsm.state import State, StatesGroup
from aiogram.fsm.storage.memory import MemoryStorage

from config import BOT_TOKEN, GROOM_NAME, BRIDE_NAME, PHOTO_PATH, ADMIN_USER_ID, WEBAPP_URL, WEDDING_ADDRESS, ADMINS_FILE, ADMINS_LIST, GROUP_LINK, GROUP_ID, GOOGLE_SHEETS_ID
import json
import os
from utils import format_wedding_date
from keyboards import (
    get_invitation_keyboard,
    get_admin_keyboard,
    get_admin_games_keyboard,
    get_admin_wordle_keyboard,
    get_admin_crossword_keyboard,
    get_group_management_keyboard,
    get_guests_selection_keyboard,
    get_invitation_dialog_keyboard,
    build_guest_swap_page,
    get_main_reply_keyboard,
    get_contacts_inline_keyboard,
    get_group_link_keyboard,
    get_admin_root_reply_keyboard,
    get_admin_guests_reply_keyboard,
    get_admin_table_reply_keyboard,
    get_admin_group_reply_keyboard,
    get_admin_bot_reply_keyboard,
)
from google_sheets import (
    get_invitations_list,
    normalize_telegram_id,
    get_admins_list,
    save_admin_to_sheets,
    get_all_guests_from_sheets,
    get_guests_count_from_sheets,
    cancel_guest_registration_by_user_id,
    delete_guest_from_sheets,
    update_invitation_user_id,
    mark_invitation_as_sent,
    list_confirmed_guests,
    swap_guest_name_order,
    ping_admin_sheet,
    write_ping_to_admin_sheet,
    get_seating_from_sheets,
    get_seating_lock_status,
    lock_seating,
    save_photo_from_user,
    get_admin_login_code_and_clear,
    check_guest_registration,
)
from telegram_client import init_telegram_client, get_username_by_phone, get_or_init_client, start_phone_login, authorize_with_code, authorize_with_password, resend_code, get_qr_code, check_qr_authorization
from datetime import datetime
import time

# Настройка логирования
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Инициализация диспетчера
dp = Dispatcher(storage=MemoryStorage())

# Бот будет создан в init_bot() после проверки токена
bot = None

# RegistrationStates удален - больше не используется (регистрация через Mini App)

# Пользовательские настройки
PHOTO_MODE_USERS: set[int] = set()

# Состояния для рассылки приглашений
class InvitationStates(StatesGroup):
    waiting_guest_selection = State()
    waiting_sent_confirmation = State()  # Ожидание подтверждения отправки

# Состояния для управления группой
class GroupManagementStates(StatesGroup):
    waiting_message = State()
    waiting_add_member = State()
    waiting_remove_member = State()

# Состояния для рассылки в личные сообщения
class BroadcastStates(StatesGroup):
    waiting_text = State()
    waiting_photo = State()
    waiting_button_choice = State()
    waiting_custom_button_text = State()
    waiting_custom_button_url = State()
    waiting_confirm = State()

# Состояния для управления играми
class GamesAdminStates(StatesGroup):
    waiting_wordle_word = State()
    waiting_crossword_words = State()


# Состояния для навигации по реплай-админ-меню
class AdminMenuStates(StatesGroup):
    root = State()
    guests = State()
    table = State()
    group = State()
    bot_menu = State()
    add_admin_waiting_username = State()
    find_userid_waiting_username = State()

# Состояния для удаления гостя

# Состояния для авторизации Telegram Client
class TelegramClientAuthStates(StatesGroup):
    waiting_code = State()  # Ожидание кода подтверждения
    waiting_password = State()  # Ожидание пароля 2FA (если включен)

async def get_user_display_name(user):
    """Получает имя пользователя из Telegram"""
    # Используем имя из Telegram (name_mapping больше не используется, все в Google Sheets)
    if user.first_name:
        if user.last_name:
            return f"{user.first_name} {user.last_name}"
        return user.first_name
    
    return "друг"  # Fallback

def is_phone_number(value: str) -> bool:
    """Проверяет, является ли значение номером телефона
    
    Номер телефона начинается с 8, 7 или +7
    Username начинается с @
    """
    if not value:
        return False
    
    value = value.strip()
    
    # Если начинается с @ - это username, не номер
    if value.startswith("@"):
        return False
    
    # Если начинается с 8, 7 или +7 - это номер телефона
    if value.startswith("+7") or value.startswith("7") or value.startswith("8"):
        return True
    
    # Дополнительная проверка: если после очистки от форматирования остаются только цифры
    # и начинается с 7 или 8 - это номер
    cleaned = value.replace(" ", "").replace("-", "").replace("(", "").replace(")", "").replace("+", "")
    if cleaned.isdigit():
        if cleaned.startswith("7") or cleaned.startswith("8"):
            return True
    
    return False

@dp.message(Command("start"))
async def cmd_start(message: Message, state: FSMContext):
    """Обработчик команды /start"""
    await state.clear()
    
    # Получаем имя из таблицы соответствия или Telegram
    display_name = await get_user_display_name(message.from_user)
    user_id = message.from_user.id
    
    # Пытаемся обновить user_id в таблице приглашений, если пользователь там есть
    # Проверяем по полному имени (first_name + last_name)
    full_name = display_name
    if message.from_user.first_name and message.from_user.last_name:
        full_name = f"{message.from_user.first_name} {message.from_user.last_name}"
    elif message.from_user.first_name:
        full_name = message.from_user.first_name
    
    # Пытаемся обновить user_id в таблице приглашений
    try:
        updated = await update_invitation_user_id(full_name, user_id)
        if updated:
            logger.info(f"Обновлен user_id для {full_name} в таблице приглашений: {user_id}")
    except Exception as e:
        logger.warning(f"Не удалось обновить user_id в таблице приглашений: {e}")
        # Не блокируем выполнение, если обновление не удалось
    
    user_id = message.from_user.id
    # Определяем, является ли пользователь админом, для клавиатуры
    is_admin_user = is_admin(user_id)
    
    # Отправляем приветственное сообщение с фото
    try:
        photo = FSInputFile(PHOTO_PATH)
        await message.answer_photo(
            photo=photo,
            caption=f"👋 Привет, {display_name}!",
            parse_mode="HTML",
            reply_markup=get_main_reply_keyboard(
                is_admin=is_admin_user,
                photo_mode_enabled=(user_id in PHOTO_MODE_USERS),
            ),
        )
    except (FileNotFoundError, Exception) as e:
        # Если фото нет или произошла ошибка, отправляем только текст
        logger.warning(f"Не удалось отправить фото в приветствии: {e}")
        await message.answer(f"👋 Привет, {display_name}!")
    
    # Отправляем приглашение
    await send_invitation_card(message)


@dp.message(F.text.in_({"📸 Фоторежим ❌", "📸 Фоторежим ✅"}))
async def toggle_photo_mode(message: Message):
    """Включение/выключение фоторежима для пользователя."""
    user_id = message.from_user.id
    is_admin_user = is_admin(user_id)

    if user_id in PHOTO_MODE_USERS:
        # Выключаем фоторежим
        PHOTO_MODE_USERS.remove(user_id)
        await message.answer(
            "📸 Фоторежим <b>выключен</b>.\n"
            "Фото больше не собираются автоматически.",
            parse_mode="HTML",
            reply_markup=get_main_reply_keyboard(
                is_admin=is_admin_user, photo_mode_enabled=False
            ),
        )
    else:
        # Включаем фоторежим - проверяем регистрацию
        try:
            is_registered = await check_guest_registration(user_id)
            if not is_registered:
                await message.answer(
                    "⚠️ Для использования фоторежима необходимо подтвердить ваше присутствие.\n"
                    "Используйте Mini App для регистрации.",
                    reply_markup=get_main_reply_keyboard(
                        is_admin=is_admin_user, photo_mode_enabled=False
                    )
                )
                return
        except Exception as e:
            logger.error(f"Ошибка при проверке регистрации для фоторежима: {e}")
            # В случае ошибки все равно разрешаем включить фоторежим
        
        PHOTO_MODE_USERS.add(user_id)
        await message.answer(
            "📸 Фоторежим <b>включен</b>.\n"
            "Просто отправляйте фото в этот чат — я всё соберу в свадебный альбом! 🙌",
            parse_mode="HTML",
            reply_markup=get_main_reply_keyboard(
                is_admin=is_admin_user, photo_mode_enabled=True
            ),
        )


@dp.message(F.photo)
async def handle_photo(message: Message):
    """
    Обработка входящих фото.
    Если у пользователя включен фоторежим — сохраняем метаданные в Google Sheets.
    Также проверяем, зарегистрирован ли пользователь.
    """
    user_id = message.from_user.id
    
    # Проверяем, включен ли фоторежим
    if user_id not in PHOTO_MODE_USERS:
        # Проверяем, зарегистрирован ли пользователь
        try:
            is_registered = await check_guest_registration(user_id)
            if is_registered:
                # Пользователь зарегистрирован, но фоторежим не включен
                await message.answer(
                    "📸 Чтобы сохранить фото в свадебный альбом, включите фоторежим.\n"
                    "Нажмите кнопку «📸 Фоторежим ❌» в меню.",
                    reply_markup=get_main_reply_keyboard(
                        is_admin=is_admin(user_id),
                        photo_mode_enabled=False
                    )
                )
            else:
                # Пользователь не зарегистрирован
                await message.answer(
                    "📸 Для сохранения фото в свадебный альбом необходимо подтвердить ваше присутствие.\n"
                    "Используйте Mini App для регистрации."
                )
        except Exception as e:
            logger.error(f"Ошибка при проверке регистрации для фото: {e}")
            # В случае ошибки просто показываем подсказку
            await message.answer(
                "📸 Чтобы сохранить фото, включите фоторежим.\n"
                "Нажмите кнопку «📸 Фоторежим ❌» в меню.",
                reply_markup=get_main_reply_keyboard(
                    is_admin=is_admin(user_id),
                    photo_mode_enabled=False
                )
            )
        return

    try:
        display_name = await get_user_display_name(message.from_user)
        username = message.from_user.username
        photo = message.photo[-1]  # самое большое
        file_id = photo.file_id

        # Дополнительная проверка регистрации перед сохранением
        is_registered = await check_guest_registration(user_id)
        if not is_registered:
            await message.answer(
                "⚠️ Для сохранения фото необходимо подтвердить ваше присутствие.\n"
                "Используйте Mini App для регистрации."
            )
            return

        ok = await save_photo_from_user(
            user_id=user_id,
            username=username,
            full_name=display_name,
            file_id=file_id,
        )

        if ok:
            await message.answer("📸 Фото сохранено в свадебный альбом 🙌")
        else:
            await message.answer(
                "⚠️ Не удалось сохранить фото. Попробуйте ещё раз позже."
            )
    except Exception as e:
        logger.error(f"Ошибка при обработке фото от {user_id}: {e}")
        import traceback

        logger.error(traceback.format_exc())
        await message.answer(
            "⚠️ Произошла ошибка при обработке фото. Попробуйте ещё раз позже."
        )


@dp.message(F.text == "💬 Общий чат")
async def open_group_chat(message: Message):
    """Отправить ссылку на общий свадебный чат."""
    await message.answer(
        "💬 <b>Общий свадебный чат</b>\n\n"
        "Нажмите кнопку ниже, чтобы перейти в беседу.",
        reply_markup=get_group_link_keyboard(),
        parse_mode="HTML",
    )


@dp.message(F.text == "📞 Связаться с нами")
async def contact_organizers(message: Message):
    """Отправить кнопки для связи с организаторами."""
    await message.answer(
        "📞 <b>Связаться с нами</b>\n\n"
        "Выберите, кому написать — откроется личный диалог в Telegram.",
        reply_markup=get_contacts_inline_keyboard(),
        parse_mode="HTML",
    )


@dp.message(F.text == "Гости")
async def admin_menu_guests(message: Message, state: FSMContext):
    """Переход в подменю 'Гости' админ-панели."""
    if not is_admin(message.from_user.id):
        return
    await state.set_state(AdminMenuStates.guests)
    await message.answer(
        "📂 <b>Админ → Гости</b>\n\nВыберите действие:",
        reply_markup=get_admin_guests_reply_keyboard(),
        parse_mode="HTML",
    )


@dp.message(F.text == "Таблица")
async def admin_menu_table(message: Message, state: FSMContext):
    """Переход в подменю 'Таблица' админ-панели."""
    if not is_admin(message.from_user.id):
        return
    await state.set_state(AdminMenuStates.table)
    await message.answer(
        "📊 <b>Админ → Таблица</b>\n\nВыберите действие:",
        reply_markup=get_admin_table_reply_keyboard(),
        parse_mode="HTML",
    )


@dp.message(F.text == "Группа")
async def admin_menu_group(message: Message, state: FSMContext):
    """Переход в подменю 'Группа' админ-панели."""
    if not is_admin(message.from_user.id):
        return
    await state.set_state(AdminMenuStates.group)
    await message.answer(
        "💬 <b>Админ → Группа</b>\n\nВыберите действие:",
        reply_markup=get_admin_group_reply_keyboard(),
        parse_mode="HTML",
    )


@dp.message(F.text == "Написать сообщение")
async def admin_menu_group_send_message(message: Message, state: FSMContext):
    """Старт отправки сообщения в группу (через реплай-меню)."""
    if not is_admin(message.from_user.id):
        return

    if not GROUP_ID:
        await message.answer(
            "❌ GROUP_ID не настроен в конфигурации.",
            parse_mode="HTML",
        )
        return

    keyboard = InlineKeyboardMarkup(
        inline_keyboard=[
            [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_group")]
        ]
    )

    await message.answer(
        "📢 <b>Отправка сообщения в группу</b>\n\n"
        "Введите текст сообщения, которое будет отправлено в группу от имени группы:",
        reply_markup=keyboard,
        parse_mode="HTML",
    )
    await state.set_state(GroupManagementStates.waiting_message)


@dp.message(F.text == "Посмотреть участников")
async def admin_menu_group_list_members(message: Message, state: FSMContext):
    """Краткая информация о группе (через реплай-меню)."""
    if not is_admin(message.from_user.id):
        return

    if not GROUP_ID:
        await message.answer(
            "❌ GROUP_ID не настроен в конфигурации.",
            parse_mode="HTML",
        )
        return

    try:
        chat = await bot.get_chat(chat_id=GROUP_ID)
        member_count = (
            chat.members_count if hasattr(chat, "members_count") else "неизвестно"
        )

        await message.answer(
            "👥 <b>Информация о группе</b>\n\n"
            f"📛 Название: {chat.title}\n"
            f"🆔 ID: <code>{GROUP_ID}</code>\n"
            f"👥 Участников: {member_count}\n"
            f"🔗 Ссылка: {GROUP_LINK}\n\n"
            "<i>Для получения полного списка участников используйте сторонние боты или API.</i>",
            parse_mode="HTML",
        )
    except Exception as e:
        error_msg = str(e)
        logger.error(f"Ошибка получения информации о группе (реплай-меню): {e}")

        if "chat not found" in error_msg.lower():
            await message.answer(
                "❌ <b>Группа не найдена!</b>\n\n"
                "Проверьте, что:\n"
                "1. Бот добавлен в группу\n"
                "2. GROUP_ID указан правильно",
                parse_mode="HTML",
            )
        else:
            await message.answer(
                "❌ <b>Ошибка получения информации о группе.</b>\n\n"
                f"<code>{error_msg}</code>",
                parse_mode="HTML",
            )


@dp.message(F.text == "Добавить/Удалить")
async def admin_menu_group_add_remove(message: Message, state: FSMContext):
    """
    Переход к расширенному управлению группой (inline-меню),
    где уже есть кнопки Добавить/Удалить/Список/Сообщение.
    """
    if not is_admin(message.from_user.id):
        return

    if not GROUP_ID:
        await message.answer(
            "❌ GROUP_ID не настроен в конфигурации.",
            parse_mode="HTML",
        )
        return

    keyboard = get_group_management_keyboard()

    await message.answer(
        f"💬 <b>Управление группой</b>\n\n"
        f"🔗 Ссылка: {GROUP_LINK}\n"
        f"🆔 ID группы: <code>{GROUP_ID}</code>\n\n"
        "Выберите нужное действие ниже:",
        reply_markup=keyboard,
        parse_mode="HTML",
    )


@dp.message(F.text == "Бот")
async def admin_menu_bot(message: Message, state: FSMContext):
    """Переход в подменю 'Бот' админ-панели."""
    if not is_admin(message.from_user.id):
        return
    await state.set_state(AdminMenuStates.bot_menu)
    await message.answer(
        "🤖 <b>Админ → Бот</b>\n\nВыберите действие:",
        reply_markup=get_admin_bot_reply_keyboard(),
        parse_mode="HTML",
    )


async def _start_telegram_client_auth(message: Message, state: FSMContext, admin_user_id: int):
    """Общая логика авторизации Telegram Client для админа."""
    # Получаем данные админа
    admins_list = await get_admins_list()
    admin_data = None

    for admin in admins_list:
        if admin.get("user_id") == admin_user_id:
            admin_data = admin
            break

    if (
        not admin_data
        or not admin_data.get("api_id")
        or not admin_data.get("api_hash")
        or not admin_data.get("phone")
    ):
        await message.answer(
            "⚠️ <b>Telegram Client API не настроен</b>\n\n"
            "Добавьте в Google Sheets (вкладка 'Админ бота'):\n"
            "• API_ID (столбец D)\n"
            "• API_HASH (столбец E)\n"
            "• PHONE (столбец F)\n\n"
            "Получить API_ID и API_HASH можно на https://my.telegram.org/auth",
            parse_mode="HTML",
        )
        return

    # Пытаемся получить уже авторизованный клиент (без отправки кода)
    client = await get_or_init_client(
        admin_user_id,
        admin_data["api_id"],
        admin_data["api_hash"],
        admin_data["phone"],
    )

    if client:
        await message.answer(
            "✅ <b>Telegram Client уже авторизован!</b>\n\n"
            "Вы можете использовать поиск username по номеру телефона.",
            parse_mode="HTML",
        )
        return

    # Явно инициируем отправку кода подтверждения
    await message.answer("📱 Отправляю код подтверждения в ваш Telegram...")
    status, msg, code_type = await start_phone_login(
        admin_user_id,
        admin_data["api_id"],
        admin_data["api_hash"],
        admin_data["phone"],
    )

    if status == "authorized":
        await message.answer(
            "✅ <b>Telegram Client уже авторизован!</b>\n\n"
            "Вы можете использовать поиск username по номеру телефона.",
            parse_mode="HTML",
        )
        return

    if status in {"code_sent", "pending"}:
        # Есть действующий код, переводим в состояние ожидания ввода
        await state.set_state(TelegramClientAuthStates.waiting_code)
        await state.update_data(admin_user_id=admin_user_id)

        keyboard = InlineKeyboardMarkup(
            inline_keyboard=[
                [
                    InlineKeyboardButton(
                        text="📥 Считать код из таблицы",
                        callback_data="admin_read_code_from_sheet",
                    )
                ],
                [
                    InlineKeyboardButton(
                        text="🔄 Запросить новый код",
                        callback_data="resend_auth_code",
                    )
                ],
                [
                    InlineKeyboardButton(
                        text="❌ Отмена",
                        callback_data="admin_back",
                    )
                ],
            ]
        )

        await message.answer(
            "📱 <b>Код подтверждения отправлен в ваш Telegram</b>\n\n"
            f"{msg}\n\n"
            "⚡ <b>ВАЖНО: введите код как можно быстрее!</b>\n\n"
            "Коды подтверждения действительны ограниченное время (обычно 1–2 минуты).\n\n"
            "Варианты ввода кода:\n"
            "• Введите в боте команду: <code>/auth_code [код]</code>\n"
            "• Просто отправьте код в чат как сообщение\n"
            "• Или вставьте код в столбец G на листе «Админ бота» и нажмите кнопку «📥 Считать код из таблицы» ниже.\n\n"
            "💡 <b>Совет:</b>\n"
            "• Откройте Telegram заранее, чтобы быстро скопировать код\n"
            "• Код приходит в ваш Telegram (не в бота)\n"
            "• Если код не пришёл или устарел, воспользуйтесь «Запросить новый код».",
            parse_mode="HTML",
            reply_markup=keyboard,
        )
        return

    # Ошибка при старте логина
    await message.answer(msg, parse_mode="HTML")


@dp.message(F.text == "🆔 Найти user_id")
async def admin_menu_find_userid(message: Message, state: FSMContext):
    """Запрос username для получения user_id."""
    if not is_admin(message.from_user.id):
        return

    await state.set_state(AdminMenuStates.find_userid_waiting_username)
    await message.answer(
        "🆔 <b>Найти user_id по username</b>\n\n"
        "Пришлите @username или ссылку вида `https://t.me/username`.\n"
        "Важно: пользователь должен хотя бы раз написать боту или быть с ботом в одной группе.",
        parse_mode="HTML",
    )


@dp.message(AdminMenuStates.find_userid_waiting_username)
async def admin_menu_find_userid_username(message: Message, state: FSMContext):
    """Обработка username и ответ с user_id."""
    if not is_admin(message.from_user.id):
        await state.clear()
        return

    raw = (message.text or "").strip()
    if not raw:
        await message.answer("❌ Отправьте, пожалуйста, @username или ссылку на пользователя.")
        return

    username = raw
    # Поддерживаем форматы: @user, user, https://t.me/user, t.me/user
    username = username.replace("https://t.me/", "").replace("http://t.me/", "")
    username = username.replace("t.me/", "")
    if username.startswith("@"):
        username = username[1:]
    username = username.split()[0].strip()

    if not username:
        await message.answer(
            "❌ Не удалось распознать username.\n"
            "Попробуйте ещё раз в формате @username.",
            parse_mode="HTML",
        )
        return

    try:
        # Bot API: getChat поддерживает username (как @username, так и просто username)
        chat = await bot.get_chat(username)
        user_id = chat.id
        full_name = ""
        if getattr(chat, "first_name", None) or getattr(chat, "last_name", None):
            full_name = f"{getattr(chat, 'first_name', '')} {getattr(chat, 'last_name', '')}".strip()

        text_lines = [
            "🆔 <b>Информация о пользователе</b>",
            "",
            f"👤 Username: @{username}",
            f"🆔 user_id: <code>{user_id}</code>",
        ]
        if full_name:
            text_lines.insert(2, f"Имя: {full_name}")

        await message.answer("\n".join(text_lines), parse_mode="HTML")
    except Exception as e:
        logger.error(f"Не удалось получить user_id для @{username}: {e}")
        await message.answer(
            "❌ Не удалось получить user_id.\n\n"
            "Чаще всего это значит, что бот ещё не видел этого пользователя.\n"
            "Попросите его написать боту или добавить его в общую группу, и попробуйте снова.",
            parse_mode="HTML",
        )
    finally:
        # Возвращаемся в подменю 'Бот'
        await state.set_state(AdminMenuStates.bot_menu)
        await message.answer(
            "🤖 <b>Админ → Бот</b>\n\nВыберите следующее действие:",
            reply_markup=get_admin_bot_reply_keyboard(),
            parse_mode="HTML",
        )


@dp.message(F.text == "Статус бота")
async def admin_menu_bot_status(message: Message, state: FSMContext):
    """Показать статус бота из подменю 'Бот'."""
    if not is_admin(message.from_user.id):
        return
    await cmd_bot_status(message)


@dp.message(F.text == "🔐 Авторизовать клиент")
async def admin_menu_auth_client(message: Message, state: FSMContext):
    """Старт авторизации Telegram Client из реплай-меню 'Бот'."""
    if not is_admin(message.from_user.id):
        return
    await _start_telegram_client_auth(message, state, message.from_user.id)


@dp.callback_query(F.data == "admin_read_code_from_sheet")
async def admin_read_code_from_sheet_callback(callback: CallbackQuery, state: FSMContext):
    """
    Считать одноразовый код авторизации из вкладки «Админ бота» (столбец G)
    и сразу авторизовать Telegram Client.
    """
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    await callback.answer()

    admin_user_id = callback.from_user.id

    # Читаем код авторизации из Google Sheets и очищаем ячейку
    code = await get_admin_login_code_and_clear(admin_user_id)
    if not code:
        keyboard = InlineKeyboardMarkup(
            inline_keyboard=[
                [
                    InlineKeyboardButton(
                        text="🔄 Запросить новый код",
                        callback_data="resend_auth_code",
                    )
                ],
                [
                    InlineKeyboardButton(
                        text="📥 Считать код из таблицы",
                        callback_data="admin_read_code_from_sheet",
                    )
                ],
                [
                    InlineKeyboardButton(
                        text="⬅️ Вернуться",
                        callback_data="admin_back",
                    )
                ],
            ]
        )
        await callback.message.answer(
            "⚠️ В таблице 'Админ бота' в вашей строке (столбец G) нет кода.\n\n"
            "1️⃣ Вставьте код из сообщения Telegram в столбец G напротив своего username.\n"
            "2️⃣ Затем снова нажмите «📥 Считать код из таблицы».",
            reply_markup=keyboard,
            parse_mode="HTML",
        )
        return

    # Код есть — используем общий обработчик авторизации по коду
    await callback.message.answer("⏳ Нашёл код в таблице, авторизую Telegram Client...")
    await process_auth_code(callback.message, state, code.strip())


@dp.message(F.text == "Начать с нуля")
async def admin_menu_reset_me(message: Message, state: FSMContext):
    """Сбросить свою регистрацию для тестирования (подменю 'Бот')."""
    if not is_admin(message.from_user.id):
        return

    await cancel_guest_registration_by_user_id(message.from_user.id)
    await message.answer(
        "✅ <b>Данные сброшены!</b>\n\n"
        "Ваша регистрация удалена из базы данных.\n"
        "Теперь вы можете пройти весь путь заново, нажав /start",
        parse_mode="HTML",
    )


@dp.message(F.text == "Добавить админа")
async def admin_menu_add_admin(message: Message, state: FSMContext):
    """Старт процесса добавления нового администратора."""
    if not is_admin(message.from_user.id):
        return

    await state.set_state(AdminMenuStates.add_admin_waiting_username)
    await message.answer(
        "👤 <b>Добавление администратора</b>\n\n"
        "Пришлите @username человека, которого хотите сделать админом.\n"
        "Важно: этот пользователь должен хотя бы раз написать боту /start.",
        parse_mode="HTML",
    )


@dp.message(AdminMenuStates.add_admin_waiting_username)
async def admin_menu_add_admin_username(message: Message, state: FSMContext):
    """Приём @username нового администратора и сохранение его в Google Sheets."""
    if not is_admin(message.from_user.id):
        await state.clear()
        return

    text = (message.text or "").strip()
    if not text:
        await message.answer("❌ Отправьте, пожалуйста, @username администратора.")
        return

    username = text
    # Поддерживаем варианты: @user, https://t.me/user, user
    if username.startswith("@"):
        username = username[1:]
    if "t.me/" in username:
        username = username.split("t.me/")[-1]
    username = username.split()[0]
    username = username.strip().lstrip("@")

    if not username:
        await message.answer("❌ Не смог распарсить username. Попробуйте ещё раз в формате @username.")
        return

    try:
        # Пытаемся получить user_id по username
        chat = await bot.get_chat(username)
        new_admin_id = chat.id
    except Exception as e:
        logger.error(f"Не удалось получить user_id для нового админа @{username}: {e}")
        await message.answer(
            "❌ Не удалось найти этого пользователя по username.\n"
            "Убедитесь, что он уже написал боту /start, и попробуйте ещё раз.",
            parse_mode="HTML",
        )
        return

    # Сохраняем в таблицу админов
    success = await save_admin_user_id(username.lower(), new_admin_id)
    if not success:
        await message.answer(
            "❌ Не удалось сохранить администратора в Google Sheets.\n"
            "Проверьте логи сервера.",
            parse_mode="HTML",
        )
        await state.set_state(AdminMenuStates.bot_menu)
        return

    # Пытаемся отправить новому админу инструкцию
    try:
        await bot.send_message(
            chat_id=new_admin_id,
            text=(
                "👋 Вас назначили администратором свадебного бота.\n\n"
                "1. Убедитесь, что у вас установлен username в Telegram.\n"
                "2. При необходимости выполните /start ещё раз.\n"
                "3. Затем используйте /admin для входа в админ-панель.\n\n"
                "Вы будете получать служебные уведомления о регистрациях и событиях."
            ),
            parse_mode="HTML",
        )
    except Exception as e:
        logger.warning(f"Не удалось отправить инструкцию новому админу @{username}: {e}")

    await message.answer(
        f"✅ Администратор @{username} добавлен.\n"
        f"User ID: <code>{new_admin_id}</code>",
        parse_mode="HTML",
    )
    await state.set_state(AdminMenuStates.bot_menu)


@dp.message(F.text == "⬅️ Вернуться")
async def admin_menu_back(message: Message, state: FSMContext):
    """Кнопка 'Вернуться' для всех уровней админ-меню."""
    if not is_admin(message.from_user.id):
        return

    current_state = await state.get_state()

    # Из подменю возвращаемся в корень
    if current_state in {
        AdminMenuStates.guests.state,
        AdminMenuStates.table.state,
        AdminMenuStates.group.state,
        AdminMenuStates.bot_menu.state,
        AdminMenuStates.add_admin_waiting_username.state,
    }:
        await state.set_state(AdminMenuStates.root)
        await message.answer(
            "🔧 <b>Панель администратора</b>\n\nВыберите раздел:",
            reply_markup=get_admin_root_reply_keyboard(),
            parse_mode="HTML",
        )
        return

    # Из корня возвращаемся к обычной пользовательской клавиатуре
    if current_state == AdminMenuStates.root.state:
        await state.clear()
        is_admin_user = is_admin(message.from_user.id)
        await message.answer(
            "⬅️ Возвращаю обычное меню.",
            reply_markup=get_main_reply_keyboard(
                is_admin=is_admin_user,
                photo_mode_enabled=(message.from_user.id in PHOTO_MODE_USERS),
            ),
            parse_mode="HTML",
        )
        return

    # Если состояние неизвестно — просто показываем обычное меню
    await state.clear()
    is_admin_user = is_admin(message.from_user.id)
    await message.answer(
        "⬅️ Возвращаюсь в основное меню.",
        reply_markup=get_main_reply_keyboard(
            is_admin=is_admin_user,
            photo_mode_enabled=(message.from_user.id in PHOTO_MODE_USERS),
        ),
        parse_mode="HTML",
    )


@dp.message(F.text == "Список гостей")
async def admin_menu_guests_list(message: Message, state: FSMContext):
    """Показать список гостей (через реплай-меню)."""
    if not is_admin(message.from_user.id):
        return

    try:
        guests = await get_all_guests_from_sheets()

        if not guests:
            await message.answer(
                "📋 <b>Список гостей</b>\n\n"
                "Пока никто не подтвердил присутствие.",
                parse_mode="HTML",
            )
            return

        guests_text = "📋 <b>Список всех гостей:</b>\n\n"
        for i, guest in enumerate(guests, 1):
            first_name = guest.get("first_name", "")
            last_name = guest.get("last_name", "")
            category = guest.get("category", "")
            side = guest.get("side", "")
            user_id = guest.get("user_id", "")

            guest_line = f"{i}. <b>{first_name} {last_name}</b>"
            if category:
                guest_line += f" ({category})"
            if side:
                guest_line += f" - {side}"
            if user_id:
                guest_line += f" [ID: {user_id}]"

            guests_text += guest_line + "\n"

        guests_text += f"\n<b>Всего: {len(guests)} гостей</b>"

        await message.answer(guests_text, parse_mode="HTML")
    except Exception as e:
        logger.error(f"Ошибка получения списка гостей (реплай-меню): {e}")
        import traceback

        logger.error(traceback.format_exc())
        await message.answer(
            "❌ Ошибка при получении списка гостей. Попробуйте позже.",
            parse_mode="HTML",
        )


@dp.message(F.text == "Рассадка")
async def admin_menu_seating(message: Message, state: FSMContext):
    """Показать рассадку по столам (через реплай-меню)."""
    if not is_admin(message.from_user.id):
        return

    try:
        seating = await get_seating_from_sheets()

        if not seating:
            await message.answer(
                "🍽 <b>Рассадка</b>\n\n"
                "Пока нет данных по рассадке (лист 'Рассадка' пуст или без гостей).",
                parse_mode="HTML",
            )
            return

        lines = ["🍽 <b>Рассадка по столам</b>\n"]
        for table in seating:
            table_name = table.get("table") or "Без названия"
            guests = table.get("guests") or []
            lines.append(f"\n<b>{table_name}</b>")
            if not guests:
                lines.append("  (пока пусто)")
            else:
                for i, name in enumerate(guests, start=1):
                    lines.append(f"{i}. {name}")

        text = "\n".join(lines)
        await message.answer(text, parse_mode="HTML")
    except Exception as e:
        logger.error(f"Ошибка получения рассадки (реплай-меню): {e}")
        import traceback

        logger.error(traceback.format_exc())
        await message.answer(
            "❌ Ошибка при получении рассадки. Попробуйте позже.",
            parse_mode="HTML",
        )


@dp.message(F.text == "Отправить приглашение")
async def admin_menu_send_invite(message: Message, state: FSMContext):
    """Запуск режима отправки приглашений (через реплай-меню)."""
    if not is_admin(message.from_user.id):
        return

    await state.clear()

    invitations = await get_invitations_list()
    if not invitations:
        await message.answer(
            "❌ <b>Список приглашений пуст</b>\n\n"
            "Проверьте вкладку 'Пригласительные' в Google Sheets.\n"
            "Убедитесь, что:\n"
            "• Столбец A содержит имена гостей\n"
            "• Столбец B содержит телеграм ID (опционально, формат: @username, t.me/username или просто username)\n\n"
            "💡 <i>Все гости из таблицы будут показаны, даже если у них нет телеграм username.</i>",
            parse_mode="HTML",
        )
        return

    await state.update_data(invitations=invitations)

    sent_count = sum(1 for inv in invitations if inv.get("is_sent", False))
    guests_list = "📋 <b>Выберите гостя для отправки приглашения:</b>\n\n"
    guests_list += f"Всего гостей: <b>{len(invitations)}</b>\n"
    guests_list += f"✅ Отправлено: <b>{sent_count}</b>\n"
    guests_list += f"⏳ Осталось: <b>{len(invitations) - sent_count}</b>\n\n"
    guests_list += (
        "Нажмите на кнопку с именем гостя, чтобы открыть диалог с заготовленным текстом приглашения.\n\n"
        "💡 <i>Гости с галочкой ✅ уже получили приглашение</i>"
    )

    keyboard = get_guests_selection_keyboard(invitations)
    await message.answer(guests_list, reply_markup=keyboard, parse_mode="HTML")


@dp.message(F.text == "Исправить имя/фамилию")
async def admin_menu_fix_names(message: Message, state: FSMContext):
    """Запуск режима исправления Имя/Фамилия (через реплай-меню)."""
    if not is_admin(message.from_user.id):
        return

    guests = await list_confirmed_guests()
    if not guests:
        await message.answer(
            "📋 <b>Исправление Имя/Фамилия</b>\n\n"
            "Пока нет ни одного подтверждённого гостя.",
            parse_mode="HTML",
        )
        return

    keyboard = build_guest_swap_page(guests, page=0)
    await message.answer(
        "🔁 <b>Исправление Имя/Фамилия</b>\n\n"
        "Нажмите на гостя, чтобы поменять местами Имя и Фамилию в Google Sheets.\n"
        "Если нажать ещё раз — порядок вернётся обратно.\n\n"
        "Строка в списке соответствует строке в вкладке «Список гостей».",
        reply_markup=keyboard,
        parse_mode="HTML",
    )


@dp.message(F.text == "Рассылка в ЛС")
async def admin_menu_broadcast_dm(message: Message, state: FSMContext):
    """Запуск рассылки в личные сообщения (через реплай-меню)."""
    if not is_admin(message.from_user.id):
        return

    await state.clear()

    recipients = await get_broadcast_recipients()
    total = len(recipients)

    keyboard = InlineKeyboardMarkup(
        inline_keyboard=[
            [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_back")]
        ]
    )

    await message.answer(
        "📨 <b>Рассылка в личные сообщения</b>\n\n"
        f"Получателей (по базе гостей): <b>{total}</b>\n\n"
        "1️⃣ Отправьте текст сообщения, которое получат гости.",
        reply_markup=keyboard,
        parse_mode="HTML",
    )
    await state.set_state(BroadcastStates.waiting_text)


@dp.message(F.text == "Открыть таблицу")
async def admin_menu_open_table(message: Message, state: FSMContext):
    """Открыть Google Sheets с данными свадьбы."""
    if not is_admin(message.from_user.id):
        return

    sheets_url = f"https://docs.google.com/spreadsheets/d/{GOOGLE_SHEETS_ID}/edit"
    await message.answer(
        "📂 <b>Таблица гостей и настроек</b>\n\n"
        "Откроется в браузере по ссылке ниже:\n"
        f"{sheets_url}",
        parse_mode="HTML",
    )


async def _admin_ping_impl(target_message: Message):
    """Общая логика проверки связи бот → сервер → Google Sheets."""
    await target_message.answer("📶 Выполняю проверку связи с Google Sheets...")

    try:
        latency_ms = await ping_admin_sheet()
        status = "OK" if latency_ms >= 0 else "ERROR"

        if latency_ms < 0:
            latency_ms = -1

        await write_ping_to_admin_sheet(
            source="bot",
            latency_ms=latency_ms,
            status=status,
        )

        now_str = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        if status == "OK":
            text = (
                "📶 <b>Проверка связи: бот → сервер → Google Sheets</b>\n\n"
                f"⏰ Время: <code>{now_str}</code>\n"
                f"📄 Лист: <code>Админ бота</code>\n"
                f"⚙️ Строка: <code>5</code>\n"
                f"⏱ Задержка: <b>{latency_ms} мс</b>\n"
                f"✅ Статус: <b>OK</b>\n\n"
                "Запись о ping сохранена в Google Sheets (строка 5 вкладки 'Админ бота')."
            )
        else:
            text = (
                "📶 <b>Проверка связи: бот → сервер → Google Sheets</b>\n\n"
                f"⏰ Время: <code>{now_str}</code>\n"
                "❌ Не удалось получить корректный ответ от Google Sheets.\n"
                "Проверьте лог сервера и настройки доступа к таблице."
            )

        await target_message.answer(text, parse_mode="HTML")
    except Exception as e:
        logger.error(f"Ошибка в admin_ping_impl: {e}")
        import traceback

        logger.error(traceback.format_exc())
        await target_message.answer(
            "❌ Произошла ошибка при проверке связи с Google Sheets.\n"
            "Подробности смотри в логах сервера.",
            parse_mode="HTML",
        )


@dp.message(F.text == "Проверить связь")
async def admin_menu_ping(message: Message, state: FSMContext):
    """Проверка связи из реплай-меню (бот → сервер → Google Sheets)."""
    if not is_admin(message.from_user.id):
        return
    await _admin_ping_impl(message)

@dp.message(F.text == "🛠 Админ-панель")
async def open_admin_panel(message: Message, state: FSMContext):
    """Быстрый вход в админ-панель по кнопке."""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к админ-панели.")
        return
    # Просто переиспользуем существующий /admin
    await cmd_admin(message, state)

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

async def get_broadcast_recipients() -> list[int]:
    """
    Получить список получателей рассылки в ЛС.
    Используем всех гостей из Google Sheets, у которых есть user_id и подтверждение.
    """
    guests = await get_all_guests_from_sheets()
    user_ids: set[int] = set()
    for guest in guests:
        uid = guest.get('user_id')
        if not uid:
            continue
        try:
            user_ids.add(int(uid))
        except (TypeError, ValueError):
            continue
    return list(user_ids)

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
    user_id = message.from_user.id
    
    # Пытаемся обновить user_id в таблице приглашений, если пользователь там есть
    # Проверяем по полному имени (first_name + last_name)
    full_name = display_name
    if message.from_user.first_name and message.from_user.last_name:
        full_name = f"{message.from_user.first_name} {message.from_user.last_name}"
    elif message.from_user.first_name:
        full_name = message.from_user.first_name
    
    # Пытаемся обновить user_id в таблице приглашений
    try:
        updated = await update_invitation_user_id(full_name, user_id)
        if updated:
            logger.info(f"Обновлен user_id для {full_name} в таблице приглашений: {user_id}")
    except Exception as e:
        logger.warning(f"Не удалось обновить user_id в таблице приглашений: {e}")
        # Не блокируем выполнение, если обновление не удалось
    
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

@dp.message(Command("auth_code"))
async def cmd_auth_code(message: Message, state: FSMContext):
    """Авторизация Telegram Client с кодом подтверждения"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        return
    
    # Получаем код из сообщения
    command_parts = message.text.split(maxsplit=1)
    if len(command_parts) < 2:
        await message.answer(
            "📝 <b>Использование команды:</b>\n\n"
            "<code>/auth_code [код]</code>\n\n"
            "Пример: <code>/auth_code 12345</code>\n\n"
            "Код подтверждения приходит в ваш Telegram при первом использовании поиска username по номеру телефона.",
            parse_mode="HTML"
        )
        return
    
    code = command_parts[1].strip()
    await process_auth_code(message, state, code)

async def process_auth_code(message: Message, state: FSMContext, code: str):
    """Обработка кода подтверждения"""
    admin_user_id = message.from_user.id
    
    await message.answer("⏳ Авторизую Telegram Client...")
    
    # Пытаемся авторизоваться с кодом
    success, msg = await authorize_with_code(admin_user_id, code)
    
    if success:
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="⬅️ Вернуться в меню", callback_data="admin_back")]
        ])
        await message.answer(msg, reply_markup=keyboard)
        await state.clear()
    elif msg == "2FA_PASSWORD_REQUIRED":
        # Требуется пароль 2FA
        await state.set_state(TelegramClientAuthStates.waiting_password)
        await state.update_data(admin_user_id=admin_user_id)
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_back")]
        ])
        await message.answer(
            "🔐 <b>Требуется пароль двухфакторной аутентификации</b>\n\n"
            "Введите пароль 2FA:\n\n"
            "<code>/auth_password [пароль]</code>\n\n"
            "Пример: <code>/auth_password mypassword123</code>",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
    elif msg == "INVALID_CODE":
        # Неверный код
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="🔄 Запросить новый код", callback_data="resend_auth_code")],
            [InlineKeyboardButton(text="🔄 Попробовать снова", callback_data="admin_auth_telegram")],
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])
        await message.answer(
            "❌ <b>Неверный код подтверждения</b>\n\n"
            "Проверьте код и попробуйте снова.\n\n"
            "💡 Если код не приходит или устарел, запросите новый код.",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
    elif msg == "EXPIRED_CODE":
        # Код устарел - предлагаем запросить новый вручную
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="🔄 Запросить новый код", callback_data="resend_auth_code")],
            [InlineKeyboardButton(text="🔄 Попробовать использовать последний код", callback_data="try_last_code")],
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])
        await message.answer(
            "⏰ <b>Код подтверждения устарел</b>\n\n"
            "Коды подтверждения действительны ограниченное время (обычно 1-2 минуты).\n\n"
            "💡 <b>Что делать:</b>\n\n"
            "1. <b>Попробуйте использовать последний полученный код</b>\n"
            "   Если вы получили код ранее, он может быть еще действителен.\n\n"
            "2. <b>Запросите новый код</b>\n"
            "   Нажмите 'Запросить новый код' для получения нового кода.\n\n"
            "⚠️ <b>Внимание:</b> Telegram ограничивает количество попыток отправки кода.\n"
            "Если все варианты использованы, нужно подождать 24 часа.",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
    else:
        # Другие ошибки
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="🔄 Запросить новый код", callback_data="resend_auth_code")],
            [InlineKeyboardButton(text="🔄 Попробовать снова", callback_data="admin_auth_telegram")],
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])
        await message.answer(
            f"{msg}\n\n"
            "💡 Если проблема сохраняется, попробуйте запросить новый код.",
            reply_markup=keyboard
        )

# Обработка обычных сообщений в состоянии ожидания кода
@dp.message(TelegramClientAuthStates.waiting_code)
async def handle_auth_code_message(message: Message, state: FSMContext):
    """Обработка кода подтверждения из обычного сообщения (без команды)"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа.")
        await state.clear()
        return
    
    # Пробуем использовать текст сообщения как код
    code = message.text.strip()
    
    # Проверяем, что это похоже на код (цифры, возможно с дефисами)
    if not code.replace("-", "").replace(" ", "").isdigit():
        await message.answer(
            "❌ Неверный формат кода. Код должен содержать только цифры.\n\n"
            "Попробуйте снова или используйте команду:\n"
            "<code>/auth_code [код]</code>",
            parse_mode="HTML"
        )
        return
    
    # Убираем дефисы и пробелы
    code = code.replace("-", "").replace(" ", "")
    
    await process_auth_code(message, state, code)

@dp.message(Command("auth_password"))
async def cmd_auth_password(message: Message, state: FSMContext):
    """Авторизация Telegram Client с паролем 2FA"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        return
    
    # Получаем пароль из сообщения
    command_parts = message.text.split(maxsplit=1)
    if len(command_parts) < 2:
        await message.answer(
            "📝 <b>Использование команды:</b>\n\n"
            "<code>/auth_password [пароль]</code>\n\n"
            "Пример: <code>/auth_password mypassword123</code>",
            parse_mode="HTML"
        )
        return
    
    password = command_parts[1].strip()
    await process_auth_password(message, state, password)

async def process_auth_password(message: Message, state: FSMContext, password: str):
    """Обработка пароля 2FA"""
    admin_user_id = message.from_user.id
    
    await message.answer("⏳ Авторизую Telegram Client с паролем 2FA...")
    
    # Пытаемся авторизоваться с паролем
    success, msg = await authorize_with_password(admin_user_id, password)
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="⬅️ Вернуться в меню", callback_data="admin_back")]
    ])
    await message.answer(msg, reply_markup=keyboard)
    await state.clear()

# Обработка обычных сообщений в состоянии ожидания пароля
@dp.message(TelegramClientAuthStates.waiting_password)
async def handle_auth_password_message(message: Message, state: FSMContext):
    """Обработка пароля 2FA из обычного сообщения (без команды)"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа.")
        await state.clear()
        return
    
    # Используем текст сообщения как пароль
    password = message.text.strip()
    
    await process_auth_password(message, state, password)

@dp.callback_query(F.data == "auth_telegram_client")
async def auth_telegram_client_callback(callback: CallbackQuery, state: FSMContext):
    """Обработчик инлайн-кнопки авторизации Telegram Client."""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    await callback.answer()
    # Переиспользуем общую логику авторизации
    await _start_telegram_client_auth(callback.message, state, callback.from_user.id)

@dp.message(Command("admin"))
async def cmd_admin(message: Message, state: FSMContext):
    """Панель администратора"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        return
    await state.set_state(AdminMenuStates.root)
    
    admin_text = f"""
🔧 <b>Панель администратора</b>

💒 Свадьба: {GROOM_NAME} и {BRIDE_NAME}
📅 Дата: {format_wedding_date()}
🌐 Mini App: {WEBAPP_URL}

Выберите раздел в админ-меню снизу:
"""
    await message.answer(
        admin_text,
        reply_markup=get_admin_root_reply_keyboard(),
        parse_mode="HTML",
    )


# ========== РАССЫЛКА В ЛИЧНЫЕ СООБЩЕНИЯ ==========

@dp.callback_query(F.data == "admin_broadcast_dm")
async def admin_broadcast_start(callback: CallbackQuery, state: FSMContext):
    """Старт сценария рассылки в личные сообщения"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    await state.clear()
    await callback.answer()

    recipients = await get_broadcast_recipients()
    total = len(recipients)

    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_back")]
    ])

    await callback.message.answer(
        "📨 <b>Рассылка в личные сообщения</b>\n\n"
        f"Получателей (по базе гостей): <b>{total}</b>\n\n"
        "1️⃣ Отправьте текст сообщения, которое получат гости.",
        reply_markup=keyboard,
        parse_mode="HTML"
    )
    await state.set_state(BroadcastStates.waiting_text)


@dp.message(BroadcastStates.waiting_text)
async def broadcast_set_text(message: Message, state: FSMContext):
    """Получаем текст рассылки"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой функции.")
        await state.clear()
        return

    text = (message.text or "").strip()
    if not text:
        await message.answer("❌ Текст пустой. Пожалуйста, отправьте текст сообщения.")
        return

    await state.update_data(broadcast_text=text)

    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="➡️ Без картинки", callback_data="broadcast_no_photo")],
        [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_back")]
    ])

    await message.answer(
        "🖼 <b>Шаг 2.</b> Теперь отправьте <b>фото</b> для рассылки "
        "или нажмите «Без картинки».",
        reply_markup=keyboard,
        parse_mode="HTML"
    )
    await state.set_state(BroadcastStates.waiting_photo)


@dp.message(BroadcastStates.waiting_photo)
async def broadcast_set_photo(message: Message, state: FSMContext):
    """Получаем фото для рассылки"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой функции.")
        await state.clear()
        return

    if not message.photo:
        await message.answer("❌ Это не фото. Пожалуйста, отправьте изображение или нажмите «Без картинки».")
        return

    photo = message.photo[-1]  # самое большое
    await state.update_data(broadcast_photo_id=photo.file_id)

    await ask_broadcast_button_choice(message, state)


@dp.callback_query(F.data == "broadcast_no_photo")
async def broadcast_no_photo(callback: CallbackQuery, state: FSMContext):
    """Админ выбрал вариант без картинки"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    await callback.answer()
    # Явно очищаем фото
    data = await state.get_data()
    data.pop("broadcast_photo_id", None)
    await state.update_data(**data)

    await ask_broadcast_button_choice(callback.message, state)


async def ask_broadcast_button_choice(target_message: Message, state: FSMContext):
    """Попросить выбрать кнопку для сообщения"""
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="🔘 Без кнопки", callback_data="broadcast_btn_none")],
        [InlineKeyboardButton(text="💒 Открыть мини-эпп", callback_data="broadcast_btn_miniapp")],
        [InlineKeyboardButton(text="💬 Открыть общий чат", callback_data="broadcast_btn_group")],
        [InlineKeyboardButton(text="➕ Добавить свою кнопку", callback_data="broadcast_btn_custom")],
        [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_back")]
    ])

    await target_message.answer(
        "🔗 <b>Шаг 3.</b> Добавить к сообщению кнопку?\n\n"
        "• «Открыть мини-эпп» — кнопка запуска Mini App\n"
        "• «Открыть общий чат» — кнопка с ссылкой на свадебный чат\n"
        "• «Добавить свою кнопку» — задать любой текст и ссылку\n"
        "• «Без кнопки» — отправить только текст (и фото, если выбрали)",
        reply_markup=keyboard,
        parse_mode="HTML"
    )
    await state.set_state(BroadcastStates.waiting_button_choice)


@dp.callback_query(F.data.in_(["broadcast_btn_none", "broadcast_btn_miniapp", "broadcast_btn_group", "broadcast_btn_custom"]))
async def broadcast_button_choice(callback: CallbackQuery, state: FSMContext):
    """Обработка выбора варианта кнопки"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    choice = callback.data
    await callback.answer()

    if choice == "broadcast_btn_none":
        await state.update_data(button_type="none")
        await show_broadcast_preview(callback.message, state)
        return

    if choice == "broadcast_btn_miniapp":
        await state.update_data(button_type="miniapp")
        await show_broadcast_preview(callback.message, state)
        return

    if choice == "broadcast_btn_group":
        await state.update_data(button_type="group")
        await show_broadcast_preview(callback.message, state)
        return

    if choice == "broadcast_btn_custom":
        await state.update_data(button_type="custom")
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_back")]
        ])
        await callback.message.answer(
            "✏️ Введите <b>текст кнопки</b> (например: «Открыть сайт»):",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
        await state.set_state(BroadcastStates.waiting_custom_button_text)


@dp.message(BroadcastStates.waiting_custom_button_text)
async def broadcast_custom_button_text(message: Message, state: FSMContext):
    """Получаем текст пользовательской кнопки"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой функции.")
        await state.clear()
        return

    text = (message.text or "").strip()
    if not text:
        await message.answer("❌ Текст кнопки пустой. Пожалуйста, отправьте текст.")
        return

    await state.update_data(custom_button_text=text)

    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_back")]
    ])
    await message.answer(
        "🔗 Теперь отправьте <b>ссылку для кнопки</b> (начинается с http:// или https:// или tg://):",
        reply_markup=keyboard,
        parse_mode="HTML"
    )
    await state.set_state(BroadcastStates.waiting_custom_button_url)


@dp.message(BroadcastStates.waiting_custom_button_url)
async def broadcast_custom_button_url(message: Message, state: FSMContext):
    """Получаем URL для пользовательской кнопки"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой функции.")
        await state.clear()
        return

    url = (message.text or "").strip()
    if not (url.startswith("http://") or url.startswith("https://") or url.startswith("tg://")):
        await message.answer("❌ Неверный формат URL. Ссылка должна начинаться с http://, https:// или tg://")
        return

    await state.update_data(custom_button_url=url)
    await show_broadcast_preview(message, state)


async def build_broadcast_reply_markup(data: dict) -> InlineKeyboardMarkup | None:
    """Построить InlineKeyboardMarkup для рассылки по данным state"""
    button_type = data.get("button_type", "none")

    if button_type == "miniapp":
        return InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(
                text="💒 Открыть приглашение",
                web_app=WebAppInfo(url=WEBAPP_URL)
            )]
        ])

    if button_type == "group" and GROUP_LINK:
        return InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(
                text="💬 Открыть свадебный чат",
                url=GROUP_LINK
            )]
        ])

    if button_type == "custom":
        text = data.get("custom_button_text")
        url = data.get("custom_button_url")
        if text and url:
            return InlineKeyboardMarkup(inline_keyboard=[
                [InlineKeyboardButton(
                    text=text,
                    url=url
                )]
            ])

    return None


async def show_broadcast_preview(target_message: Message, state: FSMContext):
    """Показать админу превью рассылки и спросить подтверждение"""
    data = await state.get_data()
    text = data.get("broadcast_text", "")
    photo_id = data.get("broadcast_photo_id")

    recipients = await get_broadcast_recipients()
    total = len(recipients)

    markup = await build_broadcast_reply_markup(data)

    # Превью сообщения
    try:
        if photo_id:
            await target_message.answer_photo(
                photo=photo_id,
                caption=text,
                reply_markup=markup
            )
        else:
            await target_message.answer(
                text,
                reply_markup=markup
            )
    except Exception as e:
        logger.error(f"Ошибка отправки превью рассылки админу: {e}")

    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="✅ Отправить всем гостям", callback_data="broadcast_send_confirm")],
        [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_back")]
    ])

    await target_message.answer(
        "📨 <b>Проверьте сообщение выше.</b>\n\n"
        f"Оно будет отправлено в ЛС всем гостям из базы, у кого есть user_id.\n"
        f"Планируется отправка: <b>{total}</b> получателям.\n\n"
        "Если всё верно — нажмите «Отправить всем гостям».",
        reply_markup=keyboard,
        parse_mode="HTML"
    )
    await state.set_state(BroadcastStates.waiting_confirm)


@dp.callback_query(F.data == "broadcast_send_confirm")
async def broadcast_send_confirm(callback: CallbackQuery, state: FSMContext):
    """Фактическая отправка рассылки всем гостям"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    data = await state.get_data()
    text = data.get("broadcast_text", "")
    photo_id = data.get("broadcast_photo_id")
    markup = await build_broadcast_reply_markup(data)

    if not text:
        await callback.answer("❌ Нет текста сообщения", show_alert=True)
        await state.clear()
        return

    recipients = await get_broadcast_recipients()
    total = len(recipients)

    if total == 0:
        await callback.message.answer(
            "⚠️ В базе гостей пока нет ни одного user_id, рассылать некому.",
            parse_mode="HTML"
        )
        await state.clear()
        await callback.answer()
        return

    await callback.answer()
    await callback.message.answer(
        f"🚀 Начинаю рассылку для <b>{total}</b> получателей…",
        parse_mode="HTML"
    )

    sent = 0
    failed = 0

    for uid in recipients:
        try:
            if photo_id:
                await bot.send_photo(
                    chat_id=uid,
                    photo=photo_id,
                    caption=text,
                    reply_markup=markup
                )
            else:
                await bot.send_message(
                    chat_id=uid,
                    text=text,
                    reply_markup=markup
                )
            sent += 1
        except Exception as e:
            failed += 1
            logger.error(f"Ошибка отправки рассылки пользователю {uid}: {e}")
        # Небольшая пауза, чтобы не упереться в лимиты
        await asyncio.sleep(0.05)

    await state.clear()

    await callback.message.answer(
        "✅ <b>Рассылка завершена.</b>\n\n"
        f"Успешно отправлено: <b>{sent}</b>\n"
        f"С ошибкой: <b>{failed}</b>",
        parse_mode="HTML"
    )

@dp.message(Command("bot_status"))
async def cmd_bot_status(message: Message):
    """Проверка статуса бота - проверяет, запущен ли только один экземпляр"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ У вас нет доступа к этой команде.")
        return
    
    import os
    from datetime import datetime
    
    status_text = "🤖 <b>Статус бота</b>\n\n"
    
    try:
        # 1. Проверяем через getMe API
        try:
            bot_info = await bot.get_me()
            status_text += f"✅ <b>Бот активен</b>\n"
            status_text += f"👤 Имя: {bot_info.first_name}\n"
            status_text += f"🆔 ID: <code>{bot_info.id}</code>\n"
            status_text += f"📝 Username: @{bot_info.username}\n\n"
        except Exception as e:
            status_text += f"❌ <b>Ошибка получения информации о боте:</b>\n"
            status_text += f"<code>{str(e)}</code>\n\n"
            if 'Conflict' in str(e) or 'TelegramConflictError' in str(e):
                status_text += f"🚨 <b>ОБНАРУЖЕН КОНФЛИКТ!</b>\n"
                status_text += f"Запущено несколько экземпляров бота!\n\n"
        
        # 2. Информация о процессе
        status_text += f"📊 <b>Информация о процессе:</b>\n"
        status_text += f"🆔 Process ID: <code>{os.getpid()}</code>\n"
        try:
            import psutil
            process = psutil.Process(os.getpid())
            status_text += f"⏰ Время запуска: {datetime.fromtimestamp(process.create_time()).strftime('%Y-%m-%d %H:%M:%S')}\n"
            status_text += f"💾 Память: {process.memory_info().rss / 1024 / 1024:.2f} MB\n\n"
        except ImportError:
            status_text += f"⚠️ psutil не установлен, дополнительная информация недоступна\n\n"
        except Exception as e:
            status_text += f"⚠️ Ошибка получения информации: {str(e)}\n\n"
    
        # 3. Проверка на Render (если доступно)
        render_service_id = os.getenv('RENDER_SERVICE_ID', '')
        if render_service_id:
            status_text += f"🌐 <b>Render Service ID:</b> <code>{render_service_id}</code>\n\n"
        
        # 4. Рекомендации
        status_text += f"💡 <b>Как проверить дубликаты:</b>\n"
        status_text += f"1. Проверьте логи на наличие 'TelegramConflictError'\n"
        status_text += f"2. На Render проверьте, нет ли нескольких сервисов с одним токеном\n"
        status_text += f"3. Убедитесь, что не используется webhook одновременно с polling\n"
        status_text += f"4. Проверьте, что старый экземпляр полностью остановлен\n"
        
    except Exception as e:
        logger.error(f"Ошибка проверки статуса бота: {e}")
        import traceback
        logger.error(traceback.format_exc())
        status_text += f"❌ <b>Ошибка проверки:</b>\n<code>{str(e)}</code>"
    
    await message.answer(status_text, parse_mode="HTML")

@dp.callback_query(F.data == "admin_guests")
async def admin_guests_list(callback: CallbackQuery):
    """Список гостей для администратора из Google Sheets"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    try:
        guests = await get_all_guests_from_sheets()

        if not guests:
            keyboard = InlineKeyboardMarkup(
                inline_keyboard=[
                    [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
                ]
            )
            await callback.message.answer(
                "📋 <b>Список гостей</b>\n\n"
                "Пока никто не подтвердил присутствие.",
                reply_markup=keyboard,
                parse_mode="HTML",
            )
            await callback.answer()
            return

        guests_text = "📋 <b>Список всех гостей:</b>\n\n"
        for i, guest in enumerate(guests, 1):
            first_name = guest.get("first_name", "")
            last_name = guest.get("last_name", "")
            category = guest.get("category", "")
            side = guest.get("side", "")
            user_id = guest.get("user_id", "")

            guest_line = f"{i}. <b>{first_name} {last_name}</b>"
            if category:
                guest_line += f" ({category})"
            if side:
                guest_line += f" - {side}"
            if user_id:
                guest_line += f" [ID: {user_id}]"

            guests_text += guest_line + "\n"

        guests_text += f"\n<b>Всего: {len(guests)} гостей</b>"

        keyboard = InlineKeyboardMarkup(
            inline_keyboard=[
                [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
            ]
        )

        await callback.message.answer(
            guests_text, reply_markup=keyboard, parse_mode="HTML"
        )
        await callback.answer()
    except Exception as e:
        logger.error(f"Ошибка получения списка гостей: {e}")
        import traceback

        logger.error(traceback.format_exc())
        keyboard = InlineKeyboardMarkup(
            inline_keyboard=[
                [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
            ]
        )
        await callback.message.answer(
            "❌ Ошибка при получении списка гостей. Попробуйте позже.",
            reply_markup=keyboard,
        )
        await callback.answer()


@dp.callback_query(F.data == "admin_seating")
async def admin_seating(callback: CallbackQuery):
    """Показать рассадку по столам из листа 'Рассадка'."""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    await callback.answer()

    try:
        seating = await get_seating_from_sheets()

        if not seating:
            keyboard = InlineKeyboardMarkup(inline_keyboard=[
                [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
            ])
            await callback.message.answer(
                "🍽 <b>Рассадка</b>\n\n"
                "Пока нет данных по рассадке (лист 'Рассадка' пуст или без гостей).",
                reply_markup=keyboard,
                parse_mode="HTML",
            )
            return

        # Формируем текст с разбивкой по столам
        lines = ["🍽 <b>Рассадка по столам</b>\n"]
        for table in seating:
            table_name = table.get("table") or "Без названия"
            guests = table.get("guests") or []
            lines.append(f"\n<b>{table_name}</b>")
            if not guests:
                lines.append("  (пока пусто)")
            else:
                for i, name in enumerate(guests, start=1):
                    lines.append(f"{i}. {name}")

        text = "\n".join(lines)

        # Если текст слишком длинный — режем на несколько сообщений
        MAX_LEN = 3800
        chunks = []
        while len(text) > MAX_LEN:
            # ищем последний перевод строки перед пределом
            split_pos = text.rfind("\n\n", 0, MAX_LEN)
            if split_pos == -1:
                split_pos = MAX_LEN
            chunks.append(text[:split_pos])
            text = text[split_pos:].lstrip()
        if text:
            chunks.append(text)

        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])

        for i, chunk in enumerate(chunks):
            # клавиатуру добавляем только к последнему сообщению
            if i == len(chunks) - 1:
                await callback.message.answer(chunk, parse_mode="HTML", reply_markup=keyboard)
            else:
                await callback.message.answer(chunk, parse_mode="HTML")

    except Exception as e:
        logger.error(f"Ошибка при получении рассадки: {e}")
        import traceback
        logger.error(traceback.format_exc())

        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])
        await callback.message.answer(
            "❌ Ошибка при получении рассадки. Попробуйте позже.",
            reply_markup=keyboard,
            parse_mode="HTML",
        )


@dp.callback_query(F.data == "admin_ping")
async def admin_ping(callback: CallbackQuery, state: FSMContext):
    """Проверка связи: бот → сервер → Google Sheets."""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    await callback.answer()
    await _admin_ping_impl(callback.message)


@dp.callback_query(F.data == "admin_lock_seating")
async def admin_lock_seating(callback: CallbackQuery, state: FSMContext):
    """Одноразовое закрепление рассадки."""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    await callback.answer()

    # Проверяем дату: кнопка логически доступна только после 2026-05-01
    now = datetime.now()
    lock_available_from = datetime(2026, 5, 1)
    if now < lock_available_from:
        await callback.message.answer(
            "🔒 Закрепить рассадку можно только после 01.05.2026.\n"
            "Сейчас изменения ещё возможны.",
            parse_mode="HTML",
        )
        return

    # Проверяем, не закреплена ли уже рассадка
    status = await get_seating_lock_status()
    if status.get("locked"):
        locked_at = status.get("locked_at") or "неизвестно"
        await callback.message.answer(
            "🔒 <b>Рассадка уже закреплена.</b>\n\n"
            f"🕐 Время фиксации: <code>{locked_at}</code>",
            parse_mode="HTML",
        )
        return

    await callback.message.answer(
        "⏳ Закрепляю текущую рассадку...\n"
        "После этого любые изменения мест в таблице не будут учитываться.",
        parse_mode="HTML",
    )

    try:
        result = await lock_seating()
        if result.get("locked"):
            locked_at = result.get("locked_at") or datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            text = (
                "✅ <b>Рассадка закреплена.</b>\n\n"
                f"🕐 Время фиксации: <code>{locked_at}</code>\n\n"
                "Теперь:\n"
                "• Все onEdit-события для 'Список гостей' и 'Рассадка' игнорируются модулем рассадки.\n"
                "• Mini App и бот могут использовать зафиксированную рассадку для показа столов гостям."
            )
        else:
            reason = result.get("reason") or "unknown"
            text = (
                "❌ <b>Не удалось закрепить рассадку.</b>\n\n"
                f"Причина: <code>{reason}</code>\n"
                "Проверьте логи сервера и содержимое листов 'Рассадка' и 'Config'."
            )

        await callback.message.answer(text, parse_mode="HTML")
    except Exception as e:
        logger.error(f"Ошибка в admin_lock_seating: {e}")
        import traceback
        logger.error(traceback.format_exc())

        await callback.message.answer(
            "❌ Произошла внутренняя ошибка при закреплении рассадки.\n"
            "Подробности смотри в логах сервера.",
            parse_mode="HTML",
        )


@dp.message(F.text == "Закрепить рассадку")
async def admin_menu_lock_seating(message: Message, state: FSMContext):
    """Одноразовое закрепление рассадки из реплай-меню."""
    if not is_admin(message.from_user.id):
        return

    # Проверяем дату: кнопка логически доступна только после 2026-05-01
    now = datetime.now()
    lock_available_from = datetime(2026, 5, 1)
    if now < lock_available_from:
        await message.answer(
            "🔒 Закрепить рассадку можно только после 01.05.2026.\n"
            "Сейчас изменения ещё возможны.",
            parse_mode="HTML",
        )
        return

    try:
        locked, locked_at = await get_seating_lock_status()
        if locked:
            await message.answer(
                "🔒 Рассадка уже была закреплена ранее.\n"
                f"Время фиксации: <b>{locked_at}</b>",
                parse_mode="HTML",
            )
            return

        await message.answer(
            "⏳ Выполняю фиксацию рассадки… Это может занять несколько секунд.",
            parse_mode="HTML",
        )

        success, locked_at = await lock_seating()
        if not success:
            await message.answer(
                "❌ Не удалось закрепить рассадку. Проверьте логи сервера.",
                parse_mode="HTML",
            )
            return

        await message.answer(
            "✅ <b>Рассадка закреплена!</b>\n\n"
            "Создан снимок текущего листа 'Рассадка' в лист 'Рассадка_фикс'.\n"
            "Все дальнейшие изменения рассадки игнорируются синхронизацией.\n\n"
            f"Время фиксации: <b>{locked_at}</b>",
            parse_mode="HTML",
        )
    except Exception as e:
        logger.error(f"Ошибка при закреплении рассадки (реплай-меню): {e}")
        import traceback

        logger.error(traceback.format_exc())
        await message.answer(
            "❌ Произошла внутренняя ошибка при закреплении рассадки.\n"
            "Подробности смотри в логах сервера.",
            parse_mode="HTML",
        )


@dp.callback_query(F.data == "admin_fix_names")
async def admin_fix_names(callback: CallbackQuery):
    """Режим: исправление порядка Имя/Фамилия для гостей"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    await callback.answer()

    guests = await list_confirmed_guests()
    if not guests:
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])
        await callback.message.answer(
            "📋 <b>Исправление Имя/Фамилия</b>\n\n"
            "Пока нет ни одного подтверждённого гостя.",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
        return

    keyboard = build_guest_swap_page(guests, page=0)
    await callback.message.answer(
        "🔁 <b>Исправление Имя/Фамилия</b>\n\n"
        "Нажмите на гостя, чтобы поменять местами Имя и Фамилию в Google Sheets.\n"
        "Если нажать ещё раз — порядок вернётся обратно.\n\n"
        "Строка в списке соответствует строке в вкладке «Список гостей».",
        reply_markup=keyboard,
        parse_mode="HTML"
    )


@dp.callback_query(F.data.startswith("fixnames_page:"))
async def admin_fix_names_page(callback: CallbackQuery):
    """Пагинация списка гостей для исправления Имя/Фамилия"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    await callback.answer()

    try:
        _, page_str = callback.data.split(":", 1)
        page = int(page_str)
    except Exception:
        page = 0

    guests = await list_confirmed_guests()
    if not guests:
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])
        await callback.message.edit_text(
            "📋 <b>Исправление Имя/Фамилия</b>\n\n"
            "Пока нет ни одного подтверждённого гостя.",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
        return

    keyboard = build_guest_swap_page(guests, page=page)
    try:
        await callback.message.edit_reply_markup(reply_markup=keyboard)
    except Exception:
        # Если не получилось обновить только клавиатуру — перерисуем целиком
        await callback.message.edit_text(
            "🔁 <b>Исправление Имя/Фамилия</b>\n\n"
            "Нажмите на гостя, чтобы поменять местами Имя и Фамилию в Google Sheets.\n"
            "Если нажать ещё раз — порядок вернётся обратно.\n\n"
            "Строка в списке соответствует строке в вкладке «Список гостей».",
            reply_markup=keyboard,
            parse_mode="HTML"
        )


@dp.callback_query(F.data.startswith("swapname:"))
async def admin_swap_guest_name(callback: CallbackQuery):
    """Перестановка Имя/Фамилия для конкретного гостя"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return

    await callback.answer()

    try:
        _, row_str, page_str = callback.data.split(":")
        row_index = int(row_str)
        page = int(page_str)
    except Exception:
        await callback.message.answer(
            "❌ Неверные данные для изменения имени. Попробуйте ещё раз.",
            parse_mode="HTML"
        )
        return

    old_name, new_name = await swap_guest_name_order(row_index)

    if not old_name and not new_name:
        await callback.message.answer(
            "❌ Не удалось изменить имя гостя. Проверьте Google Sheets и попробуйте ещё раз.",
            parse_mode="HTML"
        )
        return

    guests = await list_confirmed_guests()
    keyboard = build_guest_swap_page(guests, page=page)

    text = (
        "✅ <b>Имя гостя обновлено в Google Sheets:</b>\n"
        f"<code>{old_name}</code> → <code>{new_name}</code>\n\n"
        "Можно продолжать исправлять имена."
    )

    await callback.message.edit_text(
        text,
        reply_markup=keyboard,
        parse_mode="HTML"
    )

# Команды name_mapping удалены - все данные теперь в Google Sheets

@dp.callback_query(F.data == "resend_auth_code")
async def resend_auth_code_callback(callback: CallbackQuery, state: FSMContext):
    """Повторный запрос кода подтверждения"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    admin_user_id = callback.from_user.id
    
    await callback.message.answer("📱 Отправляю новый код подтверждения...")
    
    # Пытаемся отправить новый код
    success, msg = await resend_code(admin_user_id)
    
    if success:
        # Обновляем состояние на ожидание кода
        await state.set_state(TelegramClientAuthStates.waiting_code)
        await state.update_data(admin_user_id=admin_user_id)
        
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_back")]
        ])
        await callback.message.answer(
            f"{msg}\n\n"
            "⚡ <b>Введите новый код как можно быстрее!</b>\n\n"
            "Коды подтверждения действительны ограниченное время (обычно 1-2 минуты).\n\n"
            "Введите код:\n"
            "<code>/auth_code [код]</code>\n\n"
            "Или просто отправьте код как обычное сообщение.\n\n"
            "💡 <b>Совет:</b> Откройте Telegram заранее, чтобы быстро скопировать код.",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
    elif msg == "ALL_OPTIONS_USED":
        # Все варианты отправки кода использованы
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="🔄 Попробовать использовать последний код", callback_data="try_last_code")],
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])
        await callback.message.answer(
            "⚠️ <b>Все варианты отправки кода использованы</b>\n\n"
            "Telegram ограничивает количество попыток отправки кода для безопасности.\n\n"
            "💡 <b>Что делать:</b>\n\n"
            "1. <b>Попробуйте использовать последний полученный код</b>\n"
            "   Если вы получили код ранее, он может быть еще действителен.\n\n"
            "2. <b>Подождите 24 часа</b>\n"
            "   Лимит на отправку кодов сбрасывается через 24 часа.\n\n"
            "3. <b>Используйте другой номер телефона</b>\n"
            "   Если у вас есть доступ к другому номеру с Telegram аккаунтом.\n\n"
            "4. <b>Авторизуйтесь через официальное приложение Telegram</b>\n"
            "   После авторизации в приложении, сессия может синхронизироваться.\n\n"
            "🔒 <i>Это ограничение безопасности от Telegram, мы не можем его обойти.</i>",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
    elif msg.startswith("RATE_LIMIT:"):
        # Ограничение частоты запросов
        wait_seconds = int(msg.split(":")[1])
        wait_minutes = wait_seconds // 60
        wait_seconds_remainder = wait_seconds % 60
        
        if wait_minutes > 0:
            wait_time = f"{wait_minutes} мин. {wait_seconds_remainder} сек."
        else:
            wait_time = f"{wait_seconds} сек."
        
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="🔄 Попробовать использовать последний код", callback_data="try_last_code")],
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])
        await callback.message.answer(
            f"⏳ <b>Ограничение частоты запросов</b>\n\n"
            f"Подождите <b>{wait_time}</b> перед повторным запросом кода.\n\n"
            "💡 <b>Попробуйте использовать последний полученный код</b>\n"
            "Если вы получили код ранее, он может быть еще действителен.",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
    else:
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="🔄 Начать заново", callback_data="admin_auth_telegram")],
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])
        await callback.message.answer(
            f"{msg}\n\n"
            "Попробуйте начать процесс авторизации заново.",
            reply_markup=keyboard
        )

@dp.callback_query(F.data == "try_last_code")
async def try_last_code_callback(callback: CallbackQuery, state: FSMContext):
    """Попытка использовать последний полученный код"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    # Проверяем, есть ли ожидающий авторизации клиент
    from telegram_client import _pending_clients
    admin_user_id = callback.from_user.id
    
    if admin_user_id not in _pending_clients:
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="🔄 Начать авторизацию заново", callback_data="admin_auth_telegram")],
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])
        await callback.message.answer(
            "⚠️ Нет активного процесса авторизации.\n\n"
            "Начните процесс авторизации заново.",
            reply_markup=keyboard
        )
        return
    
    # Переводим в состояние ожидания кода
    await state.set_state(TelegramClientAuthStates.waiting_code)
    await state.update_data(admin_user_id=admin_user_id)
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_back")]
    ])
    await callback.message.answer(
        "💡 <b>Попробуйте использовать последний полученный код</b>\n\n"
        "Если вы получили код подтверждения ранее, он может быть еще действителен.\n\n"
        "Введите код:\n"
        "<code>/auth_code [код]</code>\n\n"
        "Или просто отправьте код как обычное сообщение.\n\n"
        "⚠️ <b>Если код не подходит:</b>\n"
        "• Подождите 24 часа для сброса лимита\n"
        "• Или используйте другой номер телефона",
        reply_markup=keyboard,
        parse_mode="HTML"
    )

@dp.callback_query(F.data == "check_qr_auth")
async def check_qr_auth_callback(callback: CallbackQuery, state: FSMContext):
    """Проверка QR-код авторизации"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    admin_user_id = callback.from_user.id
    
    await callback.message.answer("⏳ Проверяю авторизацию...")
    
    success, msg = await check_qr_authorization(admin_user_id)
    
    if success:
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="⬅️ Вернуться в меню", callback_data="admin_back")]
        ])
        await callback.message.answer(msg, reply_markup=keyboard)
        await state.clear()
    elif msg == "2FA_PASSWORD_REQUIRED":
        # Требуется пароль 2FA после QR-кода
        await state.set_state(TelegramClientAuthStates.waiting_password)
        await state.update_data(admin_user_id=admin_user_id)
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_back")]
        ])
        await callback.message.answer(
            "🔐 <b>Требуется пароль двухфакторной аутентификации</b>\n\n"
            "QR-код успешно отсканирован, но требуется ввести пароль 2FA.\n\n"
            "Введите пароль 2FA:\n\n"
            "<code>/auth_password [пароль]</code>\n\n"
            "Пример: <code>/auth_password mypassword123</code>",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
    else:
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="🔄 Проверить снова", callback_data="check_qr_auth")],
            [InlineKeyboardButton(text="📱 Использовать код подтверждения", callback_data="use_code_auth")],
            [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
        ])
        await callback.message.answer(
            f"{msg}\n\n"
            "💡 Если QR-код уже отсканирован, попробуйте проверить снова.\n"
            "Или используйте код подтверждения.",
            reply_markup=keyboard
        )

@dp.callback_query(F.data == "use_code_auth")
async def use_code_auth_callback(callback: CallbackQuery, state: FSMContext):
    """Переключение на авторизацию через код подтверждения"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    admin_user_id = callback.from_user.id
    
    # Получаем данные админа
    admins_list = await get_admins_list()
    admin_data = None
    
    for admin in admins_list:
        if admin.get('user_id') == admin_user_id:
            admin_data = admin
            break
    
    if not admin_data:
        await callback.message.answer("❌ Ошибка: данные админа не найдены")
        return
    
    # Закрываем текущий клиент и создаем новый с кодом подтверждения
    from telegram_client import _pending_clients, close_client
    if admin_user_id in _pending_clients:
        try:
            await close_client(admin_user_id)
        except:
            pass
    
    await callback.message.answer("📱 Запрашиваю код подтверждения...")
    
    # Создаем новый клиент с кодом подтверждения
    from telegram_client import get_or_init_client
    client = await get_or_init_client(
        admin_user_id,
        admin_data['api_id'],
        admin_data['api_hash'],
        admin_data['phone']
    )
    
    if not client:
        # Код отправлен
        await state.set_state(TelegramClientAuthStates.waiting_code)
        await state.update_data(admin_user_id=admin_user_id, auth_method='code')
        
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="🔄 Запросить новый код", callback_data="resend_auth_code")],
            [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_back")]
        ])
        await callback.message.answer(
            "📱 <b>Код подтверждения отправлен в ваш Telegram</b>\n\n"
            "⚡ <b>ВАЖНО: Введите код как можно быстрее!</b>\n\n"
            "Коды подтверждения действительны ограниченное время (обычно 1-2 минуты).\n\n"
            "Введите код:\n"
            "<code>/auth_code [код]</code>\n\n"
            "Или просто отправьте код как обычное сообщение.",
            reply_markup=keyboard,
            parse_mode="HTML"
        )

@dp.callback_query(F.data == "admin_back")
async def admin_back(callback: CallbackQuery, state: FSMContext):
    """Возврат в главное меню админа"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    # Очищаем состояние при возврате в меню
    await state.clear()
    
    await callback.message.answer(
        "👋 <b>Главное меню</b>\n\n"
        "Выберите действие:",
        reply_markup=get_admin_keyboard(),
        parse_mode="HTML",
    )
    await callback.answer()

# ========== ОБРАБОТЧИКИ ДЛЯ УПРАВЛЕНИЯ ИГРАМИ ==========

@dp.callback_query(F.data == "admin_games")
async def admin_games(callback: CallbackQuery, state: FSMContext):
    """Меню управления играми"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    await state.clear()
    
    await callback.message.answer(
        "🎮 <b>Управление играми</b>\n\n"
        "Выберите игру для управления:",
        reply_markup=get_admin_games_keyboard(),
        parse_mode="HTML"
    )

@dp.callback_query(F.data == "admin_wordle")
async def admin_wordle(callback: CallbackQuery, state: FSMContext):
    """Меню управления Wordle"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    await state.clear()
    
    await callback.message.answer(
        "🔤 <b>Управление Wordle</b>\n\n"
        "Выберите действие:",
        reply_markup=get_admin_wordle_keyboard(),
        parse_mode="HTML"
    )

@dp.callback_query(F.data == "admin_wordle_switch")
async def admin_wordle_switch(callback: CallbackQuery, state: FSMContext):
    """Переключить слово Wordle для всех пользователей"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    await callback.message.answer("⏳ Переключаю слово для всех пользователей...")
    
    success = await switch_wordle_word_for_all()
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="⬅️ Назад", callback_data="admin_wordle")]
    ])
    
    if success:
        await callback.message.answer(
            "✅ <b>Слово переключено!</b>\n\n"
            "Все пользователи получат новое слово при следующем открытии Wordle.",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
    else:
        await callback.message.answer(
            "❌ <b>Ошибка</b>\n\n"
            "Не удалось переключить слово. Проверьте логи.",
            reply_markup=keyboard,
            parse_mode="HTML"
        )

@dp.callback_query(F.data == "admin_wordle_add")
async def admin_wordle_add(callback: CallbackQuery, state: FSMContext):
    """Добавить новое слово в Wordle"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    await state.set_state(GamesAdminStates.waiting_wordle_word)
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_wordle")]
    ])
    
    await callback.message.answer(
        "➕ <b>Добавить слово в Wordle</b>\n\n"
        "Отправьте слово (существительное в именительном падеже единственного числа):",
        reply_markup=keyboard,
        parse_mode="HTML"
    )

@dp.message(GamesAdminStates.waiting_wordle_word)
async def process_wordle_word(message: Message, state: FSMContext):
    """Обработка добавления слова в Wordle"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ Нет доступа")
        return
    
    word = message.text.strip().upper()
    
    if not word or len(word) < 2:
        await message.answer("❌ Слово слишком короткое. Попробуйте еще раз:")
        return
    
    # Проверяем слово через API
    from api import validate_word
    word_valid, validation_error = await validate_word(word)
    
    if not word_valid:
        await message.answer(
            f"❌ <b>Слово не прошло проверку:</b> {validation_error}\n\n"
            "Попробуйте еще раз:",
            parse_mode="HTML"
        )
        return
    
    await message.answer("⏳ Добавляю слово...")
    
    success = await add_wordle_word(word)
    
    await state.clear()
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="⬅️ Назад", callback_data="admin_wordle")]
    ])
    
    if success:
        await message.answer(
            f"✅ <b>Слово добавлено!</b>\n\n"
            f"Слово <b>{word}</b> успешно добавлено в список Wordle.",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
    else:
        await message.answer(
            f"❌ <b>Ошибка</b>\n\n"
            f"Не удалось добавить слово. Возможно, оно уже есть в списке.",
            reply_markup=keyboard,
            parse_mode="HTML"
        )

@dp.callback_query(F.data == "admin_crossword")
async def admin_crossword(callback: CallbackQuery, state: FSMContext):
    """Меню управления кроссвордом"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    await state.clear()
    
    await callback.message.answer(
        "📝 <b>Управление кроссвордом</b>\n\n"
        "Выберите действие:",
        reply_markup=get_admin_crossword_keyboard(),
        parse_mode="HTML"
    )

@dp.callback_query(F.data == "admin_crossword_refresh")
async def admin_crossword_refresh(callback: CallbackQuery, state: FSMContext):
    """Обновить кроссворд (пересобрать из текущих слов)"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="⬅️ Назад", callback_data="admin_crossword")]
    ])
    
    await callback.message.answer(
        "ℹ️ <b>Обновление кроссворда</b>\n\n"
        "Кроссворд автоматически пересобирается при каждом открытии игры из слов в таблице.\n\n"
        "Если вы изменили слова в таблице 'Кроссвод', кроссворд обновится автоматически при следующем открытии.",
        reply_markup=keyboard,
        parse_mode="HTML"
    )

@dp.callback_query(F.data == "admin_crossword_add")
async def admin_crossword_add(callback: CallbackQuery, state: FSMContext):
    """Добавить новый кроссворд"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    await state.set_state(GamesAdminStates.waiting_crossword_words)
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="❌ Отмена", callback_data="admin_crossword")]
    ])
    
    await callback.message.answer(
        "➕ <b>Добавить новый кроссворд</b>\n\n"
        "Отправьте слова через запятую в формате:\n"
        "<code>СЛОВО1:описание1, СЛОВО2:описание2, СЛОВО3:описание3</code>\n\n"
        "Пример:\n"
        "<code>СВАДЬБА:Главное событие дня, ТАНЕЦ:Развлечение, БУКЕТ:Цветы</code>\n\n"
        "⚠️ Система проверит, можно ли собрать кроссворд из этих слов.",
        reply_markup=keyboard,
        parse_mode="HTML"
    )

@dp.message(GamesAdminStates.waiting_crossword_words)
async def process_crossword_words(message: Message, state: FSMContext):
    """Обработка добавления кроссворда"""
    if not is_admin(message.from_user.id):
        await message.answer("❌ Нет доступа")
        return
    
    text = message.text.strip()
    
    # Парсим слова в формате "СЛОВО:описание, СЛОВО2:описание2"
    words_data = []
    try:
        parts = [p.strip() for p in text.split(',')]
        for part in parts:
            if ':' in part:
                word_part, desc_part = part.split(':', 1)
                word = word_part.strip().upper()
                description = desc_part.strip()
                if word and description:
                    words_data.append({'word': word, 'description': description})
        
        if not words_data:
            await message.answer(
                "❌ <b>Неверный формат</b>\n\n"
                "Используйте формат: <code>СЛОВО:описание, СЛОВО2:описание2</code>\n\n"
                "Попробуйте еще раз:",
                parse_mode="HTML"
            )
            return
        
        await message.answer("⏳ Проверяю возможность сборки кроссворда...")
        
        # Проверяем возможность сборки
        can_generate, error_msg = await can_generate_crossword(words_data)
        
        if not can_generate:
            await message.answer(
                f"❌ <b>Невозможно собрать кроссворд</b>\n\n"
                f"{error_msg}\n\n"
                "Попробуйте добавить слова с общими буквами.",
                parse_mode="HTML"
            )
            return
        
        await message.answer("⏳ Добавляю кроссворд...")
        
        # Добавляем кроссворд
        success, result_msg = await add_crossword(words_data)
        
        await state.clear()
        
        keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="⬅️ Назад", callback_data="admin_crossword")]
        ])
        
        if success:
            await message.answer(
                f"✅ <b>Кроссворд добавлен!</b>\n\n"
                f"{result_msg}\n\n"
                f"Добавлено слов: {len(words_data)}",
                reply_markup=keyboard,
                parse_mode="HTML"
            )
        else:
            await message.answer(
                f"❌ <b>Ошибка</b>\n\n"
                f"{result_msg}",
                reply_markup=keyboard,
                parse_mode="HTML"
            )
    except Exception as e:
        logger.error(f"Ошибка обработки кроссворда: {e}")
        await message.answer(
            "❌ <b>Ошибка</b>\n\n"
            f"Произошла ошибка: {str(e)}\n\n"
            "Попробуйте еще раз:",
            parse_mode="HTML"
        )

@dp.callback_query(F.data == "admin_send_invite")
async def admin_send_invite(callback: CallbackQuery, state: FSMContext):
    """Начать рассылку приглашений"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    # Очищаем state при возврате к списку
    await state.clear()
    
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
            "• Столбец B содержит телеграм ID (опционально, формат: @username, t.me/username или просто username)\n\n"
            "💡 <i>Все гости из таблицы будут показаны, даже если у них нет телеграм username.</i>",
            reply_markup=keyboard,
            parse_mode="HTML"
        )
        return
    
    # Сохраняем список приглашений в state для использования в callback
    await state.update_data(invitations=invitations)
    
    # Формируем сообщение со списком гостей
    sent_count = sum(1 for inv in invitations if inv.get('is_sent', False))
    guests_list = f"📋 <b>Выберите гостя для отправки приглашения:</b>\n\n"
    guests_list += f"Всего гостей: <b>{len(invitations)}</b>\n"
    guests_list += f"✅ Отправлено: <b>{sent_count}</b>\n"
    guests_list += f"⏳ Осталось: <b>{len(invitations) - sent_count}</b>\n\n"
    guests_list += "Нажмите на кнопку с именем гостя, чтобы открыть диалог с заготовленным текстом приглашения.\n\n"
    guests_list += "💡 <i>Гости с галочкой ✅ уже получили приглашение</i>"
    
    # Создаем клавиатуру с кнопками для каждого гостя
    keyboard = get_guests_selection_keyboard(invitations)
    
    await callback.message.answer(guests_list, reply_markup=keyboard, parse_mode="HTML")

@dp.callback_query(F.data.startswith("invite_guest_"))
async def process_guest_selection_callback(callback: CallbackQuery, state: FSMContext):
    """Обработка выбора гостя для отправки приглашения через callback"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    # Получаем индекс гостя из callback_data
    try:
        guest_index = int(callback.data.split("_")[-1])
    except (ValueError, IndexError):
        await callback.message.answer("❌ Ошибка: неверный формат данных.")
        return
    
    # Получаем список приглашений из state
    data = await state.get_data()
    invitations = data.get('invitations', [])
    
    if not invitations or guest_index >= len(invitations):
        await callback.message.answer("❌ Ошибка: гость не найден. Попробуйте выбрать снова.")
        return
    
    # Получаем данные выбранного гостя
    guest = invitations[guest_index]
    guest_name = guest['name']
    telegram_id = guest['telegram_id']
    guest_user_id_from_table = guest.get('user_id')  # User ID из столбца C
    
    # Проверяем, является ли telegram_id номером телефона
    is_phone = is_phone_number(telegram_id) if telegram_id else False
    
    # Получаем username бота для ссылки
    bot_username = "нашбот"  # По умолчанию
    bot_link = WEBAPP_URL  # Fallback на приложение
    try:
        if bot:
            bot_info = await bot.get_me()
            if bot_info and bot_info.username:
                bot_username = bot_info.username
                bot_link = f"https://t.me/{bot_username}"
    except:
        pass
    
    # Создаем текст приглашения согласно требованиям
    invitation_text = (
        f"{guest_name}, мы - {GROOM_NAME} и {BRIDE_NAME} - женимся и хотим разделить "
        f"этот знаменательный день с родными и близкими, прикрепляем ниже открытку - просим подтвердить, "
        f"хотя бы предварительно, свое присутствие"
    )
    
    # Создаем клавиатуру с кнопкой Mini App для гостя
    bot_invite_keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(
            text="💒 Открыть приглашение",
            web_app=WebAppInfo(url=WEBAPP_URL)
        )]
    ])
    
    # Пытаемся найти user_id гостя
    guest_user_id = None
    guest_username = None
    
    # Сначала проверяем, есть ли user_id в столбце C таблицы приглашений
    if guest_user_id_from_table:
        try:
            guest_user_id = int(guest_user_id_from_table)
            logger.info(f"Найден user_id для {guest_name} в таблице приглашений: {guest_user_id}")
        except (ValueError, TypeError):
            guest_user_id = None
    
    # Если указан номер телефона - пытаемся найти username
    if is_phone:
        found_username = None
        
        # Сначала ищем в списке зарегистрированных гостей
        registered_guests = await get_all_guests_from_sheets()
        for reg_guest in registered_guests:
            reg_full_name = f"{reg_guest.get('first_name', '')} {reg_guest.get('last_name', '')}".strip()
            if reg_full_name.lower() == guest_name.lower():
                found_username = reg_guest.get('username', '')
                if found_username:
                    found_user_id = reg_guest.get('user_id')
                    break
        
        # Если не нашли в списке гостей - ищем через Telegram Client API текущего админа
        if not found_username:
            # Получаем данные Telegram Client для текущего админа
            admin_user_id = callback.from_user.id
            admins_list = await get_admins_list()
            admin_data = None
            
            for admin in admins_list:
                if admin.get('user_id') == admin_user_id:
                    admin_data = admin
                    break
            
            if admin_data and admin_data.get('api_id') and admin_data.get('api_hash') and admin_data.get('phone'):
                await callback.message.answer("🔍 Ищу username по номеру телефона через ваш Telegram аккаунт...")
                try:
                    # Получаем или инициализируем клиент для этого админа
                    client = await get_or_init_client(
                        admin_user_id,
                        admin_data['api_id'],
                        admin_data['api_hash'],
                        admin_data['phone']
                    )
                    
                    if client:
                        found_username = await get_username_by_phone(telegram_id, admin_user_id, client)
                        if found_username:
                            logger.info(f"Найден username для {guest_name} по номеру {telegram_id}: @{found_username}")
                    else:
                        # Клиент не авторизован - нужно ввести код
                        keyboard = InlineKeyboardMarkup(inline_keyboard=[
                            [InlineKeyboardButton(
                                text="🔐 Авторизовать Telegram Client",
                                callback_data="auth_telegram_client"
                            )]
                        ])
                        await callback.message.answer(
                            "⚠️ <b>Telegram Client не авторизован</b>\n\n"
                            "Для поиска username по номеру телефона нужно авторизовать ваш Telegram аккаунт.\n\n"
                            "📱 <b>Код подтверждения был отправлен в ваш Telegram</b>\n\n"
                            "Используйте команду:\n"
                            "<code>/auth_code [код]</code>\n\n"
                            "Например: <code>/auth_code 12345</code>\n\n"
                            "Или нажмите кнопку ниже для авторизации.",
                            reply_markup=keyboard,
                            parse_mode="HTML"
                        )
                except Exception as e:
                    logger.error(f"Ошибка поиска username по номеру телефона: {e}")
                    await callback.message.answer(f"❌ Ошибка поиска username: {str(e)}")
            else:
                await callback.message.answer(
                    "⚠️ <b>Telegram Client API не настроен</b>\n\n"
                    "Для поиска username по номеру телефона нужно добавить в Google Sheets (вкладка 'Админ бота'):\n"
                    "• API_ID (столбец D)\n"
                    "• API_HASH (столбец E)\n"
                    "• PHONE (столбец F)\n\n"
                    "Получить API_ID и API_HASH можно на https://my.telegram.org/auth",
                    parse_mode="HTML"
                )
        
        # Если нашли username - обновляем таблицу
        if found_username:
            try:
                found_user_id = None
                # Ищем user_id в списке зарегистрированных гостей
                for reg_guest in registered_guests:
                    reg_full_name = f"{reg_guest.get('first_name', '')} {reg_guest.get('last_name', '')}".strip()
                    if reg_full_name.lower() == guest_name.lower():
                        found_user_id = reg_guest.get('user_id')
                        if found_user_id:
                            try:
                                found_user_id = int(found_user_id)
                            except (ValueError, TypeError):
                                found_user_id = None
                        break
                
                updated = await update_invitation_user_id(guest_name, found_user_id, found_username)
                if updated:
                    logger.info(f"Обновлена таблица приглашений для {guest_name}: номер {telegram_id} заменен на @{found_username}")
                    # Обновляем данные для дальнейшей обработки
                    telegram_id = found_username
                    is_phone = False
                    guest_username = found_username
                    if found_user_id:
                        guest_user_id = found_user_id
            except Exception as e:
                logger.error(f"Ошибка обновления таблицы приглашений: {e}")
    
    # Если user_id не найден в таблице приглашений, ищем в списке зарегистрированных гостей
    if not guest_user_id:
        registered_guests = await get_all_guests_from_sheets()
        
        # Ищем по имени (сравниваем полное имя)
        for reg_guest in registered_guests:
            reg_full_name = f"{reg_guest.get('first_name', '')} {reg_guest.get('last_name', '')}".strip()
            if reg_full_name.lower() == guest_name.lower():
                if not guest_user_id:  # Если еще не нашли
                    guest_user_id = reg_guest.get('user_id')
                    if not guest_username:  # Если еще не нашли username
                        guest_username = reg_guest.get('username', '')
                    if guest_user_id:
                        try:
                            guest_user_id = int(guest_user_id)
                        except (ValueError, TypeError):
                            guest_user_id = None
                break
    
    # Получаем username бота для ссылки
    bot_username = "нашбот"  # По умолчанию
    bot_link = WEBAPP_URL  # Fallback на приложение
    try:
        if bot:
            bot_info = await bot.get_me()
            if bot_info and bot_info.username:
                bot_username = bot_info.username
                bot_link = f"https://t.me/{bot_info.username}"
    except:
        pass
    
    # Объединяем текст приглашения с инструкцией и ссылкой для копирования
    full_text_for_copy = f"{invitation_text}\n\n"
    full_text_for_copy += f"Перейдите в бота {bot_username} и нажмите старт: {bot_link}"
    
    # ЛОГИКА 1: Если поле username пусто (telegram_id пусто или None)
    if not telegram_id or telegram_id == "":
        # Информация для админа
        info_text = f"💌 <b>Готовое сообщение для {guest_name}</b>\n\n"
        info_text += "📱 <b>Телеграм:</b> не указан\n\n"
        info_text += "💡 <b>Инструкция:</b>\n"
        info_text += "1. Скопируйте текст ниже и отправьте гостю вручную\n"
        info_text += "2. После отправки нажмите 'Отправлено' или 'Не отправлено'\n\n"
        info_text += "⚠️ <i>Username не указан, отправка вручную</i>"
        
        # Кнопки: Отправлено, Не отправлено, Вернуться
        buttons = InlineKeyboardMarkup(inline_keyboard=[
            [
                InlineKeyboardButton(
                    text="✅ Отправлено",
                    callback_data="invite_sent_yes"
                ),
                InlineKeyboardButton(
                    text="❌ Не отправлено",
                    callback_data="invite_sent_no"
                )
            ],
            [InlineKeyboardButton(
                text="⬅️ Вернуться к списку",
                callback_data="admin_send_invite"
            )]
        ])
        
        await callback.message.answer(info_text, reply_markup=buttons, parse_mode="HTML")
        
        # Отправляем текст сообщения для пересылки
        await callback.message.answer(
            f"📋 <b>Текст сообщения для пересылки:</b>\n\n"
            f"<code>{full_text_for_copy}</code>",
            parse_mode="HTML"
        )
        
        # Сохраняем имя гостя в state для подтверждения отправки
        await state.update_data(guest_name_for_confirmation=guest_name)
        await state.set_state(InvitationStates.waiting_sent_confirmation)
        return
    
    # ЛОГИКА 2: Если есть номер телефона - автоматически находим username (уже обработано выше)
    # После обработки номера телефона telegram_id либо стал username, либо остался номером (если не найден)
    
    # Если после обработки номера телефона username не найден - показываем как для пустого
    if is_phone:
        # Информация для админа
        info_text = f"💌 <b>Готовое сообщение для {guest_name}</b>\n\n"
        info_text += f"📱 <b>Телефон:</b> <code>{telegram_id}</code>\n\n"
        info_text += "⚠️ <b>Username не найден</b>\n\n"
        info_text += "💡 <b>Инструкция:</b>\n"
        info_text += "1. Скопируйте текст ниже и отправьте гостю вручную\n"
        info_text += "2. После отправки нажмите 'Отправлено' или 'Не отправлено'\n\n"
        info_text += "⚠️ <i>Username не найден в ваших контактах, отправка вручную</i>"
        
        # Кнопки: Отправлено, Не отправлено, Вернуться
        buttons = InlineKeyboardMarkup(inline_keyboard=[
            [
                InlineKeyboardButton(
                    text="✅ Отправлено",
                    callback_data="invite_sent_yes"
                ),
                InlineKeyboardButton(
                    text="❌ Не отправлено",
                    callback_data="invite_sent_no"
                )
            ],
            [InlineKeyboardButton(
                text="⬅️ Вернуться к списку",
                callback_data="admin_send_invite"
            )]
        ])
        
        await callback.message.answer(info_text, reply_markup=buttons, parse_mode="HTML")
        
        # Отправляем текст сообщения для пересылки
        await callback.message.answer(
            f"📋 <b>Текст сообщения для пересылки:</b>\n\n"
            f"<code>{full_text_for_copy}</code>",
            parse_mode="HTML"
        )
        
        # Сохраняем имя гостя в state для подтверждения отправки
        await state.update_data(guest_name_for_confirmation=guest_name)
        await state.set_state(InvitationStates.waiting_sent_confirmation)
        return
    
    # ЛОГИКА 3: Если есть telegram username - присылаем deep link кнопку "Отправить автоматически"
    # (telegram_id теперь точно username)
    from urllib.parse import quote
    
    # Получаем username бота для ссылки
    bot_username = "нашбот"  # По умолчанию
    bot_link = WEBAPP_URL  # Fallback на приложение
    try:
        if bot:
            bot_info = await bot.get_me()
            if bot_info and bot_info.username:
                bot_username = bot_info.username
                bot_link = f"https://t.me/{bot_username}"
    except:
        pass
    
    # Создаем deep link с текстом приглашения + ссылкой на бота (не на приложение)
    invitation_with_link = f"{invitation_text}\n\n🔗 Перейдите в бота: {bot_link}"
    encoded_text = quote(invitation_with_link)
    if len(encoded_text) > 2000:
        # Используем короткую версию для deep link
        short_text = f"{guest_name}, мы - {GROOM_NAME} и {BRIDE_NAME} - женимся! Перейдите в бота: {bot_link}"
        encoded_text = quote(short_text)
    
    username_clean = telegram_id.lstrip('@')
    
    # Используем формат https://t.me/{username}?text={text} - более надежный и работает везде
    # Этот формат работает в веб-версии Telegram, мобильных приложениях и десктопе
    deep_link = f"https://t.me/{username_clean}?text={encoded_text}"
    
    # Если deep link слишком длинный, используем короткую версию
    if len(deep_link) > 2000:
        # Используем короткую версию
        short_text = f"{guest_name}, мы - {GROOM_NAME} и {BRIDE_NAME} - женимся! Перейдите в бота: {bot_link}"
        encoded_short = quote(short_text)
        deep_link = f"https://t.me/{username_clean}?text={encoded_short}"
    
    # Информация для админа
    display_telegram = telegram_id if not telegram_id.startswith("@") else telegram_id
    if not display_telegram.startswith("@"):
        display_telegram = f"@{display_telegram}"
    
    # Объединяем текст приглашения с инструкцией и ссылкой для копирования
    full_text_for_copy = f"{invitation_text}\n\n"
    full_text_for_copy += f"Перейдите в бота {bot_username} и нажмите старт: {bot_link}"
    
    # Информация для админа (одно сообщение со всей информацией)
    info_text = f"💌 <b>Готовое сообщение для {guest_name}</b>\n\n"
    info_text += f"📱 <b>Телеграм:</b> {display_telegram}\n\n"
    info_text += "💡 <b>Инструкция:</b>\n"
    info_text += "1. Нажмите кнопку 'Отправить автоматически' для открытия диалога с предзаполненным текстом\n"
    info_text += "2. Или скопируйте текст ниже и отправьте вручную\n"
    info_text += "3. После отправки нажмите 'Отправлено' или 'Не отправлено'\n\n"
    
    # Кнопки: Отправить автоматически, Отправлено, Не отправлено, Вернуться
    send_button = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(
            text="✅ Отправить автоматически",
            url=deep_link
        )],
        [
            InlineKeyboardButton(
                text="✅ Отправлено",
                callback_data="invite_sent_yes"
            ),
            InlineKeyboardButton(
                text="❌ Не отправлено",
                callback_data="invite_sent_no"
            )
        ],
        [InlineKeyboardButton(
            text="⬅️ Вернуться к списку",
            callback_data="admin_send_invite"
        )]
    ])
    
    await callback.message.answer(info_text, reply_markup=send_button, parse_mode="HTML")
    
    # Отправляем текст приглашения для пересылки
    await callback.message.answer(
        f"📋 <b>Текст сообщения для пересылки:</b>\n\n"
        f"<code>{full_text_for_copy}</code>",
        parse_mode="HTML"
    )
    
    # Сохраняем имя гостя в state для подтверждения отправки
    await state.update_data(guest_name_for_confirmation=guest_name)
    await state.set_state(InvitationStates.waiting_sent_confirmation)
    
    # Сохраняем данные гостя в state для использования в других callback
    await state.update_data(
        current_guest_index=guest_index,
        current_guest_name=guest_name,
        current_guest_telegram_id=telegram_id,
        current_guest_user_id=guest_user_id,
        current_invitation_text=invitation_text
    )

@dp.callback_query(F.data.startswith("send_invite_auto_"))
async def send_invite_automatically(callback: CallbackQuery, state: FSMContext):
    """Автоматическая отправка приглашения гостю"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    # Получаем индекс гостя из callback_data
    try:
        guest_index = int(callback.data.split("_")[-1])
    except (ValueError, IndexError):
        await callback.message.answer("❌ Ошибка: неверный формат данных.")
        return
    
    # Получаем данные из state
    data = await state.get_data()
    guest_name = data.get('current_guest_name')
    guest_user_id = data.get('current_guest_user_id')
    invitation_text = data.get('current_invitation_text')
    
    if not guest_user_id or not invitation_text:
        await callback.message.answer("❌ Ошибка: данные гостя не найдены. Попробуйте выбрать снова.")
        return
    
    # Создаем клавиатуру с кнопкой Mini App
    bot_invite_keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(
            text="💒 Открыть приглашение",
            web_app=WebAppInfo(url=WEBAPP_URL)
        )]
    ])
    
    try:
        # Отправляем приглашение гостю
        await bot.send_message(
            chat_id=guest_user_id,
            text=invitation_text,
            reply_markup=bot_invite_keyboard
        )
        
        # Уведомляем админа об успешной отправке
        back_keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(
                text="⬅️ Вернуться к списку",
                callback_data="admin_send_invite"
            )]
        ])
        
        await callback.message.answer(
            f"✅ <b>Приглашение отправлено!</b>\n\n"
            f"👤 <b>Гость:</b> {guest_name}\n"
            f"🆔 <b>User ID:</b> <code>{guest_user_id}</code>\n\n"
            f"Приглашение доставлено гостю автоматически.",
            reply_markup=back_keyboard,
            parse_mode="HTML"
        )
        
        logger.info(f"Админ {callback.from_user.id} автоматически отправил приглашение гостю {guest_name} (user_id: {guest_user_id})")
        
    except Exception as e:
        error_msg = str(e)
        logger.error(f"Ошибка автоматической отправки приглашения: {e}")
        
        back_keyboard = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(
                text="📤 Получить для пересылки",
                callback_data=f"send_invite_forward_{guest_index}"
            )],
            [InlineKeyboardButton(
                text="⬅️ Вернуться к списку",
                callback_data="admin_send_invite"
            )]
        ])
        
        if "chat not found" in error_msg.lower() or "user not found" in error_msg.lower():
            error_text = (
                f"❌ <b>Не удалось отправить автоматически</b>\n\n"
                f"👤 <b>Гость:</b> {guest_name}\n"
                f"🆔 <b>User ID:</b> <code>{guest_user_id}</code>\n\n"
                f"⚠️ Гость не найден или не начал диалог с ботом.\n\n"
                f"💡 <b>Решение:</b> Используйте кнопку ниже, чтобы получить готовое сообщение для пересылки."
            )
        else:
            error_text = (
                f"❌ <b>Ошибка отправки</b>\n\n"
                f"<code>{error_msg}</code>\n\n"
                f"💡 Попробуйте использовать пересылку сообщения."
            )
        
        await callback.message.answer(error_text, reply_markup=back_keyboard, parse_mode="HTML")

@dp.callback_query(F.data.startswith("send_invite_forward_"))
async def get_invite_for_forwarding(callback: CallbackQuery, state: FSMContext):
    """Получение готового сообщения с приглашением для пересылки"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    # Получаем индекс гостя из callback_data
    try:
        guest_index = int(callback.data.split("_")[-1])
    except (ValueError, IndexError):
        await callback.message.answer("❌ Ошибка: неверный формат данных.")
        return
    
    # Получаем данные из state
    data = await state.get_data()
    guest_name = data.get('current_guest_name')
    telegram_id = data.get('current_guest_telegram_id')
    invitation_text = data.get('current_invitation_text')
    
    if not invitation_text:
        # Если данных нет в state, получаем из списка приглашений
        invitations = data.get('invitations', [])
        if not invitations or guest_index >= len(invitations):
            await callback.message.answer("❌ Ошибка: гость не найден. Попробуйте выбрать снова.")
            return
        
        guest = invitations[guest_index]
        guest_name = guest['name']
        telegram_id = guest['telegram_id']
        invitation_text = (
            f"{guest_name}, мы - {GROOM_NAME} и {BRIDE_NAME} - женимся и хотим разделить "
            f"этот знаменательный день с родными и близкими, прикрепляем ниже открытку - просим подтвердить, "
            f"хотя бы предварительно, свое присутствие"
        )
    
    # Создаем клавиатуру с кнопкой Mini App
    bot_invite_keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(
            text="💒 Открыть приглашение",
            web_app=WebAppInfo(url=WEBAPP_URL)
        )]
    ])
    
    # Проверяем, является ли telegram_id номером телефона
    is_phone = is_phone_number(telegram_id) if telegram_id else False
    
    # Формируем инструкцию в зависимости от типа контакта
    if is_phone:
        instruction_text = (
            f"📤 <b>Готовое сообщение для отправки</b>\n\n"
            f"👤 <b>Гость:</b> {guest_name}\n"
            f"📱 <b>Телефон:</b> <code>{telegram_id}</code>\n\n"
            f"💡 <b>Варианты отправки:</b>\n"
            f"1. <b>Через Telegram:</b> Найдите контакт по номеру {telegram_id} и перешлите сообщение\n"
            f"2. <b>Через другие мессенджеры:</b> Скопируйте текст и отправьте через WhatsApp/SMS\n"
            f"3. <b>Через бота:</b> Попросите гостя написать боту /start\n\n"
            f"✅ Кнопка приглашения включена в сообщение ниже (работает только в Telegram)!"
        )
    else:
        # Обычный username - показываем с @ если его нет
        display_telegram = telegram_id if telegram_id else 'не указан'
        if display_telegram != 'не указан' and not display_telegram.startswith("@"):
            display_telegram = f"@{display_telegram}"
        instruction_text = (
            f"📤 <b>Готовое сообщение для пересылки</b>\n\n"
            f"👤 <b>Гость:</b> {guest_name}\n"
            f"📱 <b>Телеграм:</b> {display_telegram}\n\n"
            f"💡 <b>Инструкция:</b>\n"
            f"1. Нажмите и удерживайте сообщение ниже\n"
            f"2. Выберите 'Переслать'\n"
            f"3. Выберите получателя ({display_telegram})\n"
            f"4. Отправьте\n\n"
            f"✅ Кнопка приглашения уже включена в сообщение!"
        )
    
    # Удалено: больше не отправляем сообщение с приглашением и кнопкой для пересылки
    # Админ использует текст для пересылки, который уже был отправлен выше

@dp.callback_query(F.data == "invite_sent_yes")
async def confirm_invite_sent(callback: CallbackQuery, state: FSMContext):
    """Подтверждение отправки приглашения - отмечаем в таблице"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    # Получаем имя гостя из state
    data = await state.get_data()
    guest_name = data.get('guest_name_for_confirmation')
    
    if not guest_name:
        await callback.message.answer("❌ Ошибка: не найдено имя гостя")
        await state.clear()
        return
    
    # Обновляем статус в таблице (столбец C = "ДА")
    success = await mark_invitation_as_sent(guest_name)
    
    if success:
        await callback.message.answer(
            f"✅ <b>Приглашение отмечено как отправленное!</b>\n\n"
            f"Гость: <b>{guest_name}</b>\n\n"
            f"В таблице установлена галочка ✅",
            parse_mode="HTML"
        )
        logger.info(f"Админ {callback.from_user.id} подтвердил отправку приглашения для {guest_name}")
    else:
        await callback.message.answer(
            f"⚠️ <b>Ошибка обновления таблицы</b>\n\n"
            f"Не удалось отметить приглашение как отправленное для <b>{guest_name}</b>.\n"
            f"Попробуйте позже или обновите вручную.",
            parse_mode="HTML"
        )
    
    # Очищаем state
    await state.clear()
    
    # Предлагаем вернуться к списку
    back_keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(
            text="⬅️ Вернуться к списку приглашений",
            callback_data="admin_send_invite"
        )]
    ])
    await callback.message.answer("Выберите действие:", reply_markup=back_keyboard)

@dp.callback_query(F.data == "invite_sent_no")
async def cancel_invite_sent(callback: CallbackQuery, state: FSMContext):
    """Отмена подтверждения отправки приглашения"""
    if not is_admin(callback.from_user.id):
        await callback.answer("❌ Нет доступа", show_alert=True)
        return
    
    await callback.answer()
    
    # Получаем имя гостя из state
    data = await state.get_data()
    guest_name = data.get('guest_name_for_confirmation')
    
    # Очищаем state
    await state.clear()
    
    if guest_name:
        await callback.message.answer(
            f"ℹ️ <b>Приглашение не отмечено как отправленное</b>\n\n"
            f"Гость: <b>{guest_name}</b>\n\n"
            f"Когда отправите приглашение, вернитесь и отметьте его.",
            parse_mode="HTML"
        )
    else:
        await callback.message.answer("ℹ️ Приглашение не отмечено как отправленное")
    
    # Предлагаем вернуться к списку
    back_keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(
            text="⬅️ Вернуться к списку приглашений",
            callback_data="admin_send_invite"
        )]
    ])
    await callback.message.answer("Выберите действие:", reply_markup=back_keyboard)

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
    
    # Telegram Client API теперь инициализируется индивидуально для каждого админа
    # Данные берутся из Google Sheets (столбцы D, E, F вкладки "Админ бота")
    logger.info("ℹ️ Telegram Client API будет инициализироваться индивидуально для каждого админа")
    logger.info("Добавьте в Google Sheets (вкладка 'Админ бота'): API_ID (D), API_HASH (E), PHONE (F)")
    
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

