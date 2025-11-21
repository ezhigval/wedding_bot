"""
Тестовый скрипт для проверки работы бота в групповом чате
"""
import asyncio
from group_context import (
    add_message, find_question_in_context, should_respond_to_message,
    get_recent_messages, cleanup_old_messages
)

def print_separator():
    print("=" * 70)

def test_scenario_1():
    """Тест 1: Прямое упоминание через @username"""
    print_separator()
    print("ТЕСТ 1: Прямое упоминание через @username")
    print_separator()
    
    messages = [
        ("Гость1", "@weddingbot когда свадьба?"),
        ("Гость2", "@weddingbot подскажи где будет?"),
        ("Гость3", "Привет всем!"),
    ]
    
    for user_name, text in messages:
        should = should_respond_to_message(text, bot_username="weddingbot")
        print(f"👤 {user_name}: {text}")
        print(f"   {'✅ Бот ОТВЕТИТ' if should else '❌ Бот НЕ ответит'}")
        print()

def test_scenario_2():
    """Тест 2: Слово 'бот' в сообщении"""
    print_separator()
    print("ТЕСТ 2: Слово 'бот' в сообщении")
    print_separator()
    
    messages = [
        ("Гость1", "бот, когда свадьба?"),
        ("Гость2", "Подскажи бот, где будет?"),
        ("Гость3", "Бот, какой дресс-код?"),
        ("Гость4", "Привет ботик!"),
        ("Гость5", "Обычное сообщение без обращения"),
    ]
    
    for user_name, text in messages:
        should = should_respond_to_message(text)
        print(f"👤 {user_name}: {text}")
        print(f"   {'✅ Бот ОТВЕТИТ' if should else '❌ Бот НЕ ответит'}")
        print()

def test_scenario_3():
    """Тест 3: Вопрос от одного, 'бот' от другого (контекст)"""
    print_separator()
    print("ТЕСТ 3: Вопрос от одного, 'бот' от другого (контекст)")
    print_separator()
    
    # Очищаем контекст перед тестом
    cleanup_old_messages("test_chat")
    
    print("📝 Симуляция диалога в группе:")
    print()
    
    # Шаг 1: Гость задает вопрос
    print("1️⃣ Гость1: 'Когда начинается свадьба?'")
    add_message("test_chat", 1, "Гость1", "Когда начинается свадьба?")
    should1 = should_respond_to_message("Когда начинается свадьба?")
    print(f"   {'✅ Бот ОТВЕТИТ' if should1 else '❌ Бот НЕ ответит (правильно - нет обращения)'}")
    print()
    
    # Шаг 2: Админ пишет "бот"
    print("2️⃣ Админ: 'бот'")
    add_message("test_chat", 2, "Админ", "бот")
    should2 = should_respond_to_message("бот")
    question = find_question_in_context("test_chat", 2)
    print(f"   should_respond: {should2} ✅")
    if question:
        print(f"   ✅ Найден вопрос в контексте: {question['user_name']}: {question['text']}")
        print(f"   ✅ Бот ответит на вопрос от {question['user_name']}")
    else:
        print("   ❌ Вопрос не найден")
    print()
    
    # Шаг 3: Другой сценарий
    print("3️⃣ Гость2: 'Где будет свадьба?'")
    add_message("test_chat", 3, "Гость2", "Где будет свадьба?")
    print()
    
    print("4️⃣ Гость3: 'бот'")
    add_message("test_chat", 4, "Гость3", "бот")
    question2 = find_question_in_context("test_chat", 4)
    if question2:
        print(f"   ✅ Найден вопрос: {question2['user_name']}: {question2['text']}")
    else:
        print("   ❌ Вопрос не найден")
    print()

def test_scenario_4():
    """Тест 4: Разные варианты обращений"""
    print_separator()
    print("ТЕСТ 4: Разные варианты обращений")
    print_separator()
    
    test_cases = [
        ("@weddingbot привет", "weddingbot", True, "Упоминание через @"),
        ("бот, помоги", None, True, "Слово 'бот' в начале"),
        ("Подскажи бот", None, True, "Слово 'бот' в середине"),
        ("Спасибо ботик!", None, True, "Слово 'ботик'"),
        ("Помощник, когда?", None, True, "Слово 'помощник'"),
        ("Обычное сообщение", None, False, "Без обращения"),
        ("Когда свадьба?", None, False, "Вопрос без обращения"),
        ("Привет всем!", None, False, "Обычное сообщение"),
    ]
    
    for text, bot_username, expected, description in test_cases:
        result = should_respond_to_message(text, bot_username=bot_username)
        status = "✅" if result == expected else "❌"
        print(f"{status} {description}")
        print(f"   Текст: '{text}'")
        print(f"   Ожидается: {expected}, Получено: {result}")
        print()

def test_scenario_5():
    """Тест 5: Поиск вопросов в контексте"""
    print_separator()
    print("ТЕСТ 5: Поиск вопросов в контексте")
    print_separator()
    
    cleanup_old_messages("test_chat_2")
    
    # Добавляем разные сообщения
    messages = [
        (1, "Гость1", "Привет всем!"),
        (2, "Гость2", "Когда начинается свадьба?"),
        (3, "Гость1", "Отлично!"),
        (4, "Гость3", "Где будет?"),
        (5, "Админ", "бот"),
    ]
    
    print("📝 История сообщений:")
    for user_id, user_name, text in messages:
        add_message("test_chat_2", user_id, user_name, text)
        print(f"   {user_name}: {text}")
    print()
    
    # Ищем вопрос для последнего сообщения (от Админа)
    question = find_question_in_context("test_chat_2", 5)
    print("🔍 Поиск вопроса для последнего сообщения (Админ: 'бот'):")
    if question:
        print(f"   ✅ Найден: {question['user_name']}: {question['text']}")
    else:
        print("   ❌ Вопрос не найден")
    print()
    
    # Показываем последние сообщения
    recent = get_recent_messages("test_chat_2", limit=5)
    print("📚 Последние сообщения в контексте:")
    for msg in recent:
        print(f"   {msg['user_name']}: {msg['text']}")
    print()

async def test_scenario_6():
    """Тест 6: Полная симуляция с LLM (если доступен)"""
    print_separator()
    print("ТЕСТ 6: Полная симуляция с LLM")
    print_separator()
    
    try:
        from llm_chat import check_ollama_available, get_llm_response
        
        ollama_available = await check_ollama_available()
        if not ollama_available:
            print("⚠️ Ollama недоступен, пропускаем тест с LLM")
            print("   Для полного теста запустите: ollama serve")
            return
        
        print("✅ Ollama доступен, тестируем...")
        print()
        
        # Симулируем диалог
        cleanup_old_messages("test_chat_3")
        
        print("1️⃣ Гость1: 'Когда начинается свадьба?'")
        add_message("test_chat_3", 1, "Гость1", "Когда начинается свадьба?")
        print()
        
        print("2️⃣ Админ: 'бот'")
        add_message("test_chat_3", 2, "Админ", "бот")
        
        # Находим вопрос
        question = find_question_in_context("test_chat_3", 2)
        if question:
            print(f"   ✅ Найден вопрос: {question['user_name']}: {question['text']}")
            
            # Формируем сообщение для LLM
            user_message = f"{question['user_name']}: {question['text']}\nАдмин: бот"
            print()
            print("3️⃣ Отправляем в LLM:")
            print(f"   {user_message}")
            print()
            print("⏳ Ожидание ответа от LLM...")
            
            response = await get_llm_response(
                user_message=user_message,
                user_name="Админ",
                chat_id="test_chat_3",
                user_id="2"
            )
            
            if response:
                print(f"✅ Ответ LLM: {response[:200]}...")
            else:
                print("❌ LLM не вернул ответ")
        else:
            print("   ❌ Вопрос не найден в контексте")
        
    except Exception as e:
        print(f"❌ Ошибка при тестировании LLM: {e}")
        import traceback
        traceback.print_exc()

async def main():
    """Главная функция для запуска всех тестов"""
    print("\n" + "=" * 70)
    print("ТЕСТИРОВАНИЕ РАБОТЫ БОТА В ГРУППОВОМ ЧАТЕ")
    print("=" * 70)
    print()
    
    # Запускаем синхронные тесты
    test_scenario_1()
    test_scenario_2()
    test_scenario_3()
    test_scenario_4()
    test_scenario_5()
    
    # Запускаем асинхронный тест с LLM
    await test_scenario_6()
    
    print()
    print_separator()
    print("✅ ВСЕ ТЕСТЫ ЗАВЕРШЕНЫ")
    print_separator()
    print()
    print("💡 Для тестирования в реальной группе:")
    print("   1. Убедитесь, что Ollama запущен: ollama serve")
    print("   2. Запустите бота: python server.py")
    print("   3. Добавьте бота в группу")
    print("   4. Попробуйте разные сценарии:")
    print("      - @бот когда свадьба?")
    print("      - бот, подскажи где?")
    print("      - (кто-то задает вопрос, другой пишет 'бот')")
    print()

if __name__ == "__main__":
    asyncio.run(main())

