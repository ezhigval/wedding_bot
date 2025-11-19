from aiogram.types import InlineKeyboardMarkup, InlineKeyboardButton, WebAppInfo
from config import WEBAPP_URL, GROOM_NAME, BRIDE_NAME

def get_invitation_keyboard():
    """Клавиатура для приглашения с Mini App"""
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(
            text="💒 Открыть приглашение",
            web_app=WebAppInfo(url=WEBAPP_URL)
        )]
    ])
    return keyboard

def get_registration_keyboard():
    """Клавиатура для отмены регистрации"""
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="❌ Отмена", callback_data="cancel_registration")]
    ])
    return keyboard

def get_admin_keyboard():
    """Клавиатура для администратора"""
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="📊 Статистика", callback_data="admin_stats")],
        [InlineKeyboardButton(text="🔄 Обновить Mini App", callback_data="admin_reload")],
        [InlineKeyboardButton(text="📋 Список гостей", callback_data="admin_guests")],
        [InlineKeyboardButton(text="👤 Управление именами", callback_data="admin_names")],
        [InlineKeyboardButton(text="💌 Отправить приглашение", callback_data="admin_send_invite")],
        [InlineKeyboardButton(text="🔄 Начать с нуля", callback_data="admin_reset_me")],
        [InlineKeyboardButton(text="⬅️ Вернуться в меню", callback_data="admin_back")]
    ])
    return keyboard

def get_send_invitation_keyboard(guest_name: str, telegram_id: str):
    """Клавиатура для отправки приглашения конкретному гостю"""
    # Создаем текст приглашения
    invitation_text = (
        f"Дорогой(ая) {guest_name}, с большой радостью сообщаю - мы, {GROOM_NAME} и {BRIDE_NAME}, "
        f"женимся и приглашаем тебя на наш прекрасный праздник."
    )
    
    # URL для открытия диалога с пользователем
    # Telegram не поддерживает прямой deep link с текстом для конкретного пользователя
    # Используем tg://resolve?domain=username для открытия профиля
    deep_link = f"tg://resolve?domain={telegram_id}"
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(
            text="💬 Открыть диалог",
            url=deep_link
        )],
        [InlineKeyboardButton(
            text="⬅️ Вернуться",
            callback_data="admin_send_invite"
        )]
    ])
    return keyboard, invitation_text

