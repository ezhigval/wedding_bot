from aiogram.types import InlineKeyboardMarkup, InlineKeyboardButton, WebAppInfo
from config import WEBAPP_URL, GROOM_NAME, BRIDE_NAME, WEDDING_DATE
from datetime import datetime

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
        [InlineKeyboardButton(text="📱 Обновить номера → username", callback_data="admin_update_phones")],
        [InlineKeyboardButton(text="🗑️ Удалить гостя", callback_data="admin_delete_guest")],
        [InlineKeyboardButton(text="💬 Управление группой", callback_data="admin_group")],
        [InlineKeyboardButton(text="🔄 Начать с нуля", callback_data="admin_reset_me")],
        [InlineKeyboardButton(text="⬅️ Вернуться в меню", callback_data="admin_back")]
    ])
    return keyboard

def get_delete_guest_confirmation_keyboard(guest_user_id: int):
    """Клавиатура для подтверждения удаления гостя из группы"""
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="✅ Да, удалить из беседы", callback_data=f"delete_guest_confirm_group_{guest_user_id}")],
        [InlineKeyboardButton(text="❌ Нет, только из списка", callback_data=f"delete_guest_confirm_only_{guest_user_id}")],
        [InlineKeyboardButton(text="⬅️ Отмена", callback_data="admin_back")]
    ])
    return keyboard

def get_group_management_keyboard():
    """Клавиатура для управления группой"""
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="📢 Отправить сообщение в группу", callback_data="group_send_message")],
        [InlineKeyboardButton(text="➕ Добавить участника", callback_data="group_add_member")],
        [InlineKeyboardButton(text="➖ Удалить участника", callback_data="group_remove_member")],
        [InlineKeyboardButton(text="👥 Список участников", callback_data="group_list_members")],
        [InlineKeyboardButton(text="⬅️ Вернуться", callback_data="admin_back")]
    ])
    return keyboard

def get_guests_selection_keyboard(invitations: list):
    """Клавиатура с кнопками для выбора гостя из списка приглашений"""
    keyboard_buttons = []
    
    # Создаем кнопки для каждого гостя (максимум 2 кнопки в ряд)
    for i in range(0, len(invitations), 2):
        row = []
        # Первая кнопка в ряду
        inv = invitations[i]
        is_sent = inv.get('is_sent', False)
        # Если приглашение отправлено - показываем с галочкой
        if is_sent:
            button_text = f"✅ {inv['name']}"
        else:
            button_text = f"👤 {inv['name']}"
        row.append(InlineKeyboardButton(
            text=button_text,
            callback_data=f"invite_guest_{i}"
        ))
        # Вторая кнопка в ряду (если есть)
        if i + 1 < len(invitations):
            inv2 = invitations[i + 1]
            is_sent2 = inv2.get('is_sent', False)
            if is_sent2:
                button_text2 = f"✅ {inv2['name']}"
            else:
                button_text2 = f"👤 {inv2['name']}"
            row.append(InlineKeyboardButton(
                text=button_text2,
                callback_data=f"invite_guest_{i + 1}"
            ))
        keyboard_buttons.append(row)
    
    # Кнопка возврата
    keyboard_buttons.append([InlineKeyboardButton(
        text="⬅️ Вернуться",
        callback_data="admin_back"
    )])
    
    return InlineKeyboardMarkup(inline_keyboard=keyboard_buttons)

def get_invitation_dialog_keyboard(telegram_id: str, invitation_text: str = ""):
    """Клавиатура для открытия диалога с гостем"""
    # URL для открытия диалога с предзаполненным текстом
    # Используем tg://msg?to=username&text=... для предзаполнения текста
    # Если текст слишком длинный, используем более короткую версию
    if invitation_text:
        # Кодируем текст для URL
        from urllib.parse import quote
        encoded_text = quote(invitation_text)
        # Ограничиваем длину текста (Telegram имеет ограничения)
        if len(encoded_text) > 2000:
            # Используем короткую версию
            short_text = f"Дорогой(ая), мы - {GROOM_NAME} и {BRIDE_NAME} - женимся! Открой приглашение ниже 💒"
            encoded_text = quote(short_text)
        deep_link = f"tg://msg?to={telegram_id}&text={encoded_text}"
    else:
        # Fallback: просто открываем диалог
        deep_link = f"tg://resolve?domain={telegram_id}"
    
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(
            text="💬 Открыть диалог с текстом",
            url=deep_link
        )],
        [InlineKeyboardButton(
            text="⬅️ Вернуться к списку",
            callback_data="admin_send_invite"
        )]
    ])
    return keyboard

