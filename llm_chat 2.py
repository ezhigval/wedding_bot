"""
Модуль для работы с локальной языковой моделью через Ollama
"""
import logging
import aiohttp
import os
from typing import Optional, List, Dict
from config import GROOM_NAME, BRIDE_NAME, WEDDING_DATE, WEDDING_ADDRESS
from llm_memory import (
    init_memory_db, save_conversation, get_recent_conversations,
    extract_and_save_info, get_context_for_llm, save_fact, get_fact
)

logger = logging.getLogger(__name__)

# URL для локального Ollama сервера
OLLAMA_BASE_URL = os.getenv("OLLAMA_BASE_URL", "http://localhost:11434")
# Qwen2.5 - лучшая бесплатная модель для русского языка (отличная поддержка русского, хорошее качество)
# Альтернативы: llama3.2 (быстрая), mistral (хорошая), phi-3 (легкая)
OLLAMA_MODEL = os.getenv("OLLAMA_MODEL", "qwen2.5:7b")  # Qwen2.5 7B - оптимальный баланс качества и скорости

# Загружаем инструкции для бота
async def load_bot_instructions() -> str:
    """Загружает инструкции для бота из файла, скрипты ответов и правила из Google Sheets"""
    instructions_path = "bot_instructions.txt"
    base_instructions = ""
    
    # Загружаем базовые инструкции
    if os.path.exists(instructions_path):
        try:
            with open(instructions_path, 'r', encoding='utf-8') as f:
                base_instructions = f.read()
        except Exception as e:
            logger.error(f"Ошибка загрузки инструкций: {e}")
    
    # Если базовых инструкций нет, используем дефолтные
    if not base_instructions:
        base_instructions = get_default_instructions()
    
    # Загружаем скрипты ответов
    from llm_scripts import load_scripts
    scripts = load_scripts()
    
    # Объединяем инструкции и скрипты
    if scripts:
        base_instructions += f"\n\n=== ДОПОЛНИТЕЛЬНЫЕ СКРИПТЫ ДЛЯ ОТВЕТОВ ===\n{scripts}\n\nИспользуй эти скрипты при ответах на соответствующие вопросы гостей.\n"
    
    # Загружаем правила из Google Sheets
    try:
        from google_sheets import get_ai_rules_from_sheets
        rules = await get_ai_rules_from_sheets()
        if rules:
            rules_text = "\n".join([f"- {rule}" for rule in rules])
            base_instructions += f"\n\n=== ПРАВИЛА ИЗ GOOGLE SHEETS ===\n{rules_text}\n\nЭти правила добавлены админами и должны использоваться при ответах.\n"
    except Exception as e:
        logger.warning(f"Ошибка загрузки правил из Google Sheets: {e}")
    
    return base_instructions

def get_default_instructions() -> str:
    """Возвращает инструкции по умолчанию"""
    wedding_date_str = WEDDING_DATE.strftime("%d.%m.%Y")
    
    return f"""Ты - дружелюбный помощник на свадьбе {GROOM_NAME} и {BRIDE_NAME}.

О СВАДЬБЕ:
- Дата свадьбы: {wedding_date_str}
- Место проведения: {WEDDING_ADDRESS}
- Жених: {GROOM_NAME}
- Невеста: {BRIDE_NAME}

ТВОЯ РОЛЬ:
Ты помогаешь гостям свадьбы в групповом чате. Отвечай дружелюбно, вежливо и по делу.

ПРАВИЛА ОБЩЕНИЯ:
1. Отвечай кратко и по существу
2. Используй дружелюбный, но не слишком фамильярный тон
3. Если не знаешь ответа, честно скажи об этом
4. Не выдумывай информацию о свадьбе
5. Помогай гостям с вопросами о дате, месте, дресс-коде
6. Можешь делиться полезной информацией о мероприятии

ЧТО ТЫ МОЖЕШЬ:
- Отвечать на вопросы о свадьбе
- Помогать с информацией о месте и времени
- Делиться полезными советами для гостей
- Поддерживать дружелюбную атмосферу в чате

ЧТО ТЫ НЕ ДЕЛАЕШЬ:
- Не даешь личные контакты жениха и невесты без разрешения
- Не меняешь информацию о свадьбе
- Не участвуешь в спорах или конфликтах
- Не даешь медицинские или юридические советы

Отвечай на русском языке, будь полезным и дружелюбным помощником!"""

async def get_llm_response(
    user_message: str,
    context_messages: Optional[List[Dict[str, str]]] = None,
    user_name: Optional[str] = None,
    chat_id: Optional[str] = None,
    user_id: Optional[str] = None
) -> Optional[str]:
    """
    Получает ответ от локальной LLM через Ollama с использованием памяти
    
    Args:
        user_message: Сообщение пользователя
        context_messages: История сообщений для контекста (опционально)
        user_name: Имя пользователя (опционально)
        chat_id: ID чата для сохранения контекста (опционально)
        user_id: ID пользователя (опционально)
    
    Returns:
        Ответ от LLM или None в случае ошибки
    """
    try:
        # Загружаем инструкции
        system_prompt = await load_bot_instructions()
        
        # Получаем контекст из памяти, если есть chat_id
        memory_context = ""
        if chat_id:
            memory_context = await get_context_for_llm(chat_id, max_facts=5)
            if memory_context:
                system_prompt += f"\n\n=== ПАМЯТЬ И КОНТЕКСТ ===\n{memory_context}\n\nВАЖНО: Используй эту информацию из памяти при ответе. Если в памяти есть релевантная информация - используй её. Если информация в памяти противоречит базовым данным о свадьбе - приоритет у базовых данных.\n"
        
        # Формируем промпт
        messages = [
            {"role": "system", "content": system_prompt}
        ]
        
        # Добавляем контекст из памяти, если есть
        if context_messages:
            messages.extend(context_messages[-5:])  # Берем последние 5 сообщений для контекста
        elif chat_id:
            # Получаем последние диалоги из памяти
            recent_convs = await get_recent_conversations(chat_id, limit=3)
            for conv in reversed(recent_convs):  # В хронологическом порядке
                if conv["user_name"] and conv["message"]:
                    messages.append({
                        "role": "user",
                        "content": f"{conv['user_name']}: {conv['message']}"
                    })
                if conv["response"]:
                    messages.append({
                        "role": "assistant",
                        "content": conv["response"]
                    })
        
        # Добавляем текущее сообщение
        if user_name:
            messages.append({
                "role": "user",
                "content": f"{user_name}: {user_message}"
            })
        else:
            messages.append({
                "role": "user",
                "content": user_message
            })
        
        # Отправляем запрос в Ollama
        async with aiohttp.ClientSession() as session:
            url = f"{OLLAMA_BASE_URL}/api/chat"
            payload = {
                "model": OLLAMA_MODEL,
                "messages": messages,
                "stream": False,
                "options": {
                    "temperature": 0.7,
                    "top_p": 0.9,
                    "max_tokens": 500  # Ограничиваем длину ответа
                }
            }
            
            async with session.post(url, json=payload, timeout=aiohttp.ClientTimeout(total=30)) as response:
                if response.status == 200:
                    data = await response.json()
                    llm_response = data.get("message", {}).get("content", "").strip()
                    
                    # Сохраняем диалог в память
                    if chat_id:
                        await save_conversation(
                            chat_id=chat_id,
                            user_id=user_id,
                            user_name=user_name,
                            message=user_message,
                            response=llm_response,
                            importance=0.7  # Средняя важность
                        )
                        
                        # Извлекаем и сохраняем важную информацию
                        try:
                            saved_facts = await extract_and_save_info(user_message, llm_response)
                            if saved_facts:
                                logger.info(f"Сохранены факты: {saved_facts}")
                        except Exception as e:
                            logger.warning(f"Ошибка извлечения информации: {e}")
                    
                    return llm_response
                else:
                    error_text = await response.text()
                    logger.error(f"Ошибка Ollama API: {response.status} - {error_text}")
                    return None
                    
    except aiohttp.ClientError as e:
        logger.error(f"Ошибка подключения к Ollama: {e}")
        return None
    except Exception as e:
        logger.error(f"Неожиданная ошибка при работе с LLM: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return None

async def check_ollama_available() -> bool:
    """Проверяет доступность Ollama сервера"""
    try:
        async with aiohttp.ClientSession() as session:
            url = f"{OLLAMA_BASE_URL}/api/tags"
            async with session.get(url, timeout=aiohttp.ClientTimeout(total=5)) as response:
                return response.status == 200
    except:
        return False

