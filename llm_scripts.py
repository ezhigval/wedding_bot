"""
Модуль для управления скриптами ответов для LLM
"""
import os
import logging
from typing import List, Optional
from datetime import datetime

logger = logging.getLogger(__name__)

SCRIPTS_FILE = "response_scripts.txt"

def load_scripts() -> str:
    """Загружает скрипты ответов из файла"""
    if os.path.exists(SCRIPTS_FILE):
        try:
            with open(SCRIPTS_FILE, 'r', encoding='utf-8') as f:
                content = f.read().strip()
                return content if content else ""
        except Exception as e:
            logger.error(f"Ошибка загрузки скриптов: {e}")
            return ""
    return ""

def save_script(script_text: str, admin_name: Optional[str] = None) -> bool:
    """Сохраняет новый скрипт в файл"""
    try:
        # Загружаем существующие скрипты
        existing_scripts = load_scripts()
        
        # Формируем новый скрипт с метаданными
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        admin_info = f" (добавлено админом: {admin_name})" if admin_name else ""
        
        new_script = f"\n\n--- Скрипт от {timestamp}{admin_info} ---\n{script_text}\n"
        
        # Добавляем к существующим скриптам
        all_scripts = existing_scripts + new_script if existing_scripts else new_script.strip()
        
        # Сохраняем в файл
        with open(SCRIPTS_FILE, 'w', encoding='utf-8') as f:
            f.write(all_scripts)
        
        logger.info(f"Скрипт сохранен: {script_text[:50]}...")
        return True
    except Exception as e:
        logger.error(f"Ошибка сохранения скрипта: {e}")
        return False

async def format_script_with_llm(raw_text: str) -> Optional[str]:
    """
    Форматирует текст правила с помощью LLM для лучшей структуры
    Возвращает отформатированный текст или None в случае ошибки
    """
    try:
        # Импортируем внутри функции, чтобы избежать циклического импорта
        from llm_chat import get_llm_response
        
        prompt = f"""Ты помогаешь форматировать правила для свадебного бота.

Админ написал следующее правило/инструкцию:
{raw_text}

Задача: переформулируй это правило в четкую, структурированную инструкцию для бота, которая будет добавлена в скрипты ответов.

Требования:
1. Сохрани смысл и намерение админа
2. Сделай текст четким и понятным для бота
3. Используй структурированный формат (можно с пунктами)
4. Убери лишние слова, но сохрани важную информацию
5. Ответ должен быть на русском языке
6. Не добавляй лишних комментариев, только само правило

Верни только отформатированное правило, без дополнительных объяснений."""

        formatted = await get_llm_response(
            user_message=prompt,
            user_name="Система форматирования"
        )
        
        if formatted:
            # Очищаем от возможных артефактов LLM
            formatted = formatted.strip()
            # Убираем кавычки, если LLM их добавил
            if formatted.startswith('"') and formatted.endswith('"'):
                formatted = formatted[1:-1]
            if formatted.startswith("'") and formatted.endswith("'"):
                formatted = formatted[1:-1]
            
            return formatted
        else:
            # Если LLM не ответил, возвращаем исходный текст
            return raw_text
            
    except Exception as e:
        logger.warning(f"Ошибка форматирования скрипта через LLM: {e}")
        # В случае ошибки возвращаем исходный текст
        return raw_text

def get_all_scripts() -> List[dict]:
    """Получает все скрипты с метаданными"""
    scripts = []
    if os.path.exists(SCRIPTS_FILE):
        try:
            with open(SCRIPTS_FILE, 'r', encoding='utf-8') as f:
                content = f.read()
                
            # Парсим скрипты по разделителям
            parts = content.split("--- Скрипт от")
            for i, part in enumerate(parts):
                if i == 0 and not part.strip():
                    continue
                
                lines = part.strip().split('\n')
                if not lines:
                    continue
                
                # Первая строка содержит метаданные
                meta_line = lines[0] if lines else ""
                script_text = '\n'.join(lines[1:]).strip() if len(lines) > 1 else ""
                
                if script_text:
                    scripts.append({
                        "meta": meta_line,
                        "text": script_text
                    })
        except Exception as e:
            logger.error(f"Ошибка чтения скриптов: {e}")
    
    return scripts

