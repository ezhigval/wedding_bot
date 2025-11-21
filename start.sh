#!/bin/bash
set -e

echo "🚀 Запуск свадебного бота с Ollama..."

# Получаем модель из переменной окружения или используем по умолчанию
OLLAMA_MODEL=${OLLAMA_MODEL:-qwen2.5:7b}

# Запуск Ollama в фоне
echo "📦 Запуск Ollama сервера..."
ollama serve &
OLLAMA_PID=$!

# Ждем пока Ollama запустится
echo "⏳ Ожидание запуска Ollama..."
sleep 5

# Проверяем доступность Ollama
for i in {1..30}; do
    if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        echo "✅ Ollama запущен!"
        break
    fi
    echo "⏳ Попытка $i/30..."
    sleep 2
done

# Проверяем наличие модели, если нет - скачиваем
echo "🔍 Проверка модели $OLLAMA_MODEL..."
MODEL_EXISTS=$(ollama list 2>/dev/null | grep -q "$OLLAMA_MODEL" && echo "yes" || echo "no")

if [ "$MODEL_EXISTS" = "no" ]; then
    echo "📥 Скачивание модели $OLLAMA_MODEL..."
    ollama pull "$OLLAMA_MODEL" || {
        echo "⚠️ Ошибка загрузки модели, пробуем более легкую..."
        ollama pull llama3.2:3b
        export OLLAMA_MODEL=llama3.2:3b
    }
    echo "✅ Модель загружена!"
else
    echo "✅ Модель уже установлена!"
fi

# Запуск основного приложения
echo "🤖 Запуск бота..."
python server.py

# Обработка завершения
trap "kill $OLLAMA_PID 2>/dev/null || true" EXIT

