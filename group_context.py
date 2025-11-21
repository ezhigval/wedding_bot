"""
Модуль для отслеживания контекста сообщений в групповом чате
"""
from typing import Dict, List, Optional
from datetime import datetime, timedelta
import logging

logger = logging.getLogger(__name__)

# Хранилище последних сообщений по чатам
# Структура: {chat_id: [{"user_id": ..., "user_name": ..., "text": ..., "timestamp": ...}, ...]}
chat_messages: Dict[str, List[Dict]] = {}

# Время хранения сообщений (5 минут)
MESSAGE_TTL = timedelta(minutes=5)

# Максимальное количество сообщений для хранения на чат
MAX_MESSAGES_PER_CHAT = 20

def add_message(chat_id: str, user_id: int, user_name: str, text: str):
    """Добавляет сообщение в контекст чата"""
    chat_id_str = str(chat_id)
    
    if chat_id_str not in chat_messages:
        chat_messages[chat_id_str] = []
    
    chat_messages[chat_id_str].append({
        "user_id": user_id,
        "user_name": user_name,
        "text": text,
        "timestamp": datetime.now()
    })
    
    # Ограничиваем количество сообщений
    if len(chat_messages[chat_id_str]) > MAX_MESSAGES_PER_CHAT:
        chat_messages[chat_id_str] = chat_messages[chat_id_str][-MAX_MESSAGES_PER_CHAT:]
    
    # Очищаем старые сообщения
    cleanup_old_messages(chat_id_str)

def cleanup_old_messages(chat_id: str):
    """Удаляет старые сообщения из контекста"""
    if chat_id not in chat_messages:
        return
    
    now = datetime.now()
    chat_messages[chat_id] = [
        msg for msg in chat_messages[chat_id]
        if now - msg["timestamp"] < MESSAGE_TTL
    ]

def get_recent_messages(chat_id: str, limit: int = 10) -> List[Dict]:
    """Получает последние сообщения из чата"""
    chat_id_str = str(chat_id)
    cleanup_old_messages(chat_id_str)
    
    if chat_id_str not in chat_messages:
        return []
    
    return chat_messages[chat_id_str][-limit:]

def find_question_in_context(chat_id: str, current_user_id: int) -> Optional[Dict]:
    """
    Ищет вопрос в предыдущих сообщениях, если текущее сообщение содержит "бот"
    Возвращает последнее сообщение с вопросом от другого пользователя
    """
    chat_id_str = str(chat_id)
    cleanup_old_messages(chat_id_str)
    
    if chat_id_str not in chat_messages:
        return None
    
    # Ищем в обратном порядке (от новых к старым)
    messages = chat_messages[chat_id_str][::-1]
    
    # Пропускаем самое последнее сообщение (текущее, которое только что добавили)
    # и ищем вопрос от другого пользователя
    for msg in messages[1:]:  # Пропускаем первое (текущее) сообщение
        # Игнорируем сообщения от текущего пользователя
        if msg["user_id"] == current_user_id:
            continue
        
        # Ищем сообщения, которые выглядят как вопросы
        text = msg["text"].lower().strip()
        
        # Проверяем, что это похоже на вопрос
        question_indicators = [
            "?", "когда", "где", "как", "что", "кто", "почему", "зачем", "сколько",
            "во сколько", "какой", "какая", "какое", "какие", "можно", "можно ли",
            "подскажи", "скажи", "расскажи", "помоги", "помогите"
        ]
        
        # Проверяем наличие индикаторов вопроса
        has_question_indicator = any(indicator in text for indicator in question_indicators)
        ends_with_question = text.endswith("?")
        
        if has_question_indicator or ends_with_question:
            # Проверяем, что это не слишком старое сообщение (не более 2 минут назад)
            if datetime.now() - msg["timestamp"] < timedelta(minutes=2):
                return msg
    
    return None

def should_respond_to_message(message_text: str, bot_username: Optional[str] = None, 
                            reply_to_message: Optional[Dict] = None) -> bool:
    """
    Проверяет, должен ли бот отвечать на сообщение
    
    Args:
        message_text: Текст сообщения
        bot_username: Username бота (без @)
        reply_to_message: Сообщение, на которое отвечают (если есть)
    
    Returns:
        True, если бот должен ответить
    """
    if not message_text:
        return False
    
    text_lower = message_text.lower()
    
    # Проверяем упоминание через @username
    if bot_username:
        if f"@{bot_username}" in text_lower or f"@{bot_username.lower()}" in text_lower:
            return True
    
    # Проверяем наличие слова "бот"
    bot_keywords = ["бот", "bot", "ботик", "помощник"]
    if any(keyword in text_lower for keyword in bot_keywords):
        return True
    
    # Проверяем, если это ответ на сообщение бота
    if reply_to_message and reply_to_message.get("from_user", {}).get("is_bot"):
        return True
    
    return False

