"""
Тестовый скрипт для проверки команд-действий от админов
"""
import asyncio
from admin_commands import detect_admin_command, extract_message_text_manual, AVAILABLE_COMMANDS

async def test_command_detection():
    """Тест определения команд"""
    print("=" * 70)
    print("ТЕСТ ОПРЕДЕЛЕНИЯ КОМАНД ОТ АДМИНОВ")
    print("=" * 70)
    print()
    
    # Симулируем админа
    admin_id = 12345
    is_admin = True
    
    test_cases = [
        ("Сделай рассылку всем в группу 'Дорогие друзья, на свадьбе не будет салютов, но будет файр шоу'", "рассылка"),
        ("Пришли список гостей", "список_гостей"),
        ("Покажи статистику", "статистика"),
        ("Отправить сообщение: Привет всем!", "рассылка"),
        ("Рассылка: Спасибо за регистрацию!", "рассылка"),
        ("Написать в группу 'Важная информация'", "рассылка"),
        ("Обычное сообщение", None),
        ("Когда свадьба?", None),
    ]
    
    print("📝 Тест определения команд:")
    print()
    for message, expected_command in test_cases:
        result = await detect_admin_command(message, admin_id, is_admin)
        if result:
            detected_command = result.get("command")
            status = "✅" if detected_command == expected_command else "❌"
            print(f"{status} '{message}'")
            print(f"   Определено: {detected_command}, Ожидалось: {expected_command}")
            if detected_command == "рассылка":
                params = result.get("params", {})
                msg_text = params.get("message_text")
                if msg_text:
                    print(f"   Извлеченный текст: {msg_text[:50]}...")
        else:
            status = "✅" if expected_command is None else "❌"
            print(f"{status} '{message}'")
            print(f"   Команда не определена (ожидалось: {expected_command})")
        print()

def test_message_extraction():
    """Тест извлечения текста сообщения"""
    print("=" * 70)
    print("ТЕСТ ИЗВЛЕЧЕНИЯ ТЕКСТА СООБЩЕНИЯ")
    print("=" * 70)
    print()
    
    test_cases = [
        ("Сделай рассылку 'Дорогие друзья, на свадьбе не будет салютов'", "Дорогие друзья, на свадьбе не будет салютов"),
        ("Отправить сообщение: Привет всем!", "Привет всем!"),
        ("Рассылка: Спасибо за регистрацию!", "Спасибо за регистрацию!"),
        ("Написать в группу \"Важная информация\"", "Важная информация"),
    ]
    
    print("📝 Тест извлечения текста:")
    print()
    for message, expected in test_cases:
        result = extract_message_text_manual(message)
        status = "✅" if expected.lower() in result.lower() or result.lower() in expected.lower() else "❌"
        print(f"{status} '{message}'")
        print(f"   Извлечено: '{result}'")
        print(f"   Ожидалось: '{expected}'")
        print()

async def test_full_flow():
    """Тест полного потока выполнения команды"""
    print("=" * 70)
    print("ТЕСТ ПОЛНОГО ПОТОКА (требует LLM)")
    print("=" * 70)
    print()
    
    try:
        from llm_chat import check_ollama_available
        
        ollama_available = await check_ollama_available()
        if not ollama_available:
            print("⚠️ Ollama недоступен, пропускаем тест с LLM")
            print("   Для полного теста запустите: ollama serve")
            return
        
        print("✅ Ollama доступен, тестируем...")
        print()
        
        admin_id = 12345
        is_admin = True
        
        # Тест команды рассылки
        message = "Сделай рассылку всем в группу 'Дорогие друзья, на свадьбе не будет салютов, но будет файр шоу'"
        print(f"📤 Команда: {message}")
        
        command_info = await detect_admin_command(message, admin_id, is_admin)
        if command_info:
            print(f"✅ Команда определена: {command_info.get('command')}")
            params = command_info.get("params", {})
            msg_text = params.get("message_text")
            if msg_text:
                print(f"✅ Текст извлечен: {msg_text}")
            else:
                print("⚠️ Текст не извлечен")
        else:
            print("❌ Команда не определена")
        print()
        
    except Exception as e:
        print(f"❌ Ошибка: {e}")
        import traceback
        traceback.print_exc()

async def main():
    """Главная функция"""
    print("\n" + "=" * 70)
    print("ТЕСТИРОВАНИЕ КОМАНД-ДЕЙСТВИЙ ОТ АДМИНОВ")
    print("=" * 70)
    print()
    
    print(f"📋 Доступные команды: {list(AVAILABLE_COMMANDS.keys())}")
    print()
    
    # Запускаем тесты
    await test_command_detection()
    test_message_extraction()
    await test_full_flow()
    
    print()
    print("=" * 70)
    print("✅ ВСЕ ТЕСТЫ ЗАВЕРШЕНЫ")
    print("=" * 70)
    print()
    print("💡 Для тестирования в реальной группе:")
    print("   1. Убедитесь, что вы админ")
    print("   2. Напишите в группу команду, например:")
    print("      - 'Сделай рассылку всем в группу \"Ваше сообщение\"'")
    print("      - 'Пришли список гостей'")
    print("      - 'Покажи статистику'")
    print()

if __name__ == "__main__":
    asyncio.run(main())

