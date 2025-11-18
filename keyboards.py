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
        [InlineKeyboardButton(text="👤 Управление именами", callback_data="admin_names")]
    ])
    return keyboard

