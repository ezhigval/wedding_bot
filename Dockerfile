# Dockerfile для деплоя бота на Render.com
FROM python:3.11-slim

# Установка системных зависимостей
RUN apt-get update && apt-get install -y \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Создание рабочей директории
WORKDIR /app

# Копирование requirements и установка зависимостей Python
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Копирование всех файлов проекта
COPY . .

# Создание директорий для данных
RUN mkdir -p data res webapp

# Скрипт запуска
COPY start.sh /start.sh
RUN chmod +x /start.sh

# Открытие порта
EXPOSE 10000

# Запуск через скрипт
CMD ["/start.sh"]

