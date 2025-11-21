# Dockerfile для деплоя бота с Ollama на Render.com
FROM python:3.11-slim

# Установка системных зависимостей
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Установка Ollama
RUN curl -fsSL https://ollama.com/install.sh | sh

# Создание рабочей директории
WORKDIR /app

# Копирование requirements и установка зависимостей Python
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Копирование всех файлов проекта
COPY . .

# Создание директорий для данных
RUN mkdir -p data res webapp

# Переменные окружения для Ollama
ENV OLLAMA_HOST=0.0.0.0:11434
ENV OLLAMA_MODELS=/root/.ollama/models

# Скрипт запуска
COPY start.sh /start.sh
RUN chmod +x /start.sh

# Открытие портов
EXPOSE 10000 11434

# Запуск через скрипт
CMD ["/start.sh"]

