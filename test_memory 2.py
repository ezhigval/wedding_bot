"""
Тестовый скрипт для проверки работы памяти LLM
"""
import asyncio
import sys
from llm_memory import (
    init_memory_db, save_fact, get_fact, get_all_facts,
    save_conversation, get_recent_conversations, get_context_for_llm
)
from llm_chat import get_llm_response, check_ollama_available

async def test_memory():
    """Тестирует работу памяти"""
    print("=" * 60)
    print("Тест системы памяти для LLM")
    print("=" * 60)
    print()
    
    # Инициализация БД
    print("🔧 Инициализация базы данных...")
    await init_memory_db()
    print("✅ База данных инициализирована\n")
    
    # Тест 1: Сохранение и получение фактов
    print("📝 Тест 1: Сохранение и получение фактов")
    await save_fact("test_time", "Свадьба начинается в 15:00", "Пользователь спросил о времени")
    fact = await get_fact("test_time")
    print(f"✅ Сохранен факт: test_time = {fact}")
    
    await save_fact("test_place", "Панорама Холл, Токсово", "Пользователь спросил о месте")
    fact = await get_fact("test_place")
    print(f"✅ Сохранен факт: test_place = {fact}\n")
    
    # Тест 2: Сохранение диалогов
    print("💬 Тест 2: Сохранение диалогов")
    test_chat_id = "test_chat_123"
    await save_conversation(
        chat_id=test_chat_id,
        user_id="user_1",
        user_name="Тестовый пользователь",
        message="Когда начинается свадьба?",
        response="Свадьба начинается в 15:00! 🎉"
    )
    print("✅ Диалог сохранен")
    
    await save_conversation(
        chat_id=test_chat_id,
        user_id="user_2",
        user_name="Другой пользователь",
        message="Где будет свадьба?",
        response="Свадьба пройдет в Панорама Холл, Токсово."
    )
    print("✅ Второй диалог сохранен\n")
    
    # Тест 3: Получение контекста
    print("📚 Тест 3: Получение контекста для LLM")
    context = await get_context_for_llm(test_chat_id, max_facts=5)
    print("Контекст:")
    print(context)
    print()
    
    # Тест 4: Получение всех фактов
    print("📋 Тест 4: Все сохраненные факты")
    all_facts = await get_all_facts()
    for fact in all_facts:
        print(f"  - {fact['key']}: {fact['value'][:50]}...")
    print()
    
    # Тест 5: Интеграция с LLM (если доступен)
    print("🤖 Тест 5: Интеграция с LLM")
    ollama_available = await check_ollama_available()
    if ollama_available:
        print("✅ Ollama доступен, тестируем ответ с памятью...")
        response = await get_llm_response(
            user_message="Когда начинается свадьба?",
            user_name="Тестовый пользователь",
            chat_id=test_chat_id,
            user_id="test_user"
        )
        if response:
            print(f"✅ LLM ответил: {response[:100]}...")
        else:
            print("❌ LLM не вернул ответ")
    else:
        print("⚠️ Ollama недоступен, пропускаем тест LLM")
    
    print("\n✅ Все тесты завершены!")

if __name__ == "__main__":
    asyncio.run(test_memory())

