"""
Объединенный сервер: Telegram бот + API + Веб-сервер для Mini App
"""
import asyncio
import logging
import sys
from aiohttp import web
from aiohttp.web import Response
import aiofiles
import os
from pathlib import Path

from bot import dp, init_bot, notify_admins
from api import init_api, set_notify_function
from config import WEBAPP_PATH

# Настройка логирования с выводом в stdout для Render
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout)
    ]
)

logger = logging.getLogger(__name__)

async def serve_static(request):
    """Сервинг статических файлов для Mini App"""
    path = request.match_info.get('path', '')
    
    # Если путь пустой или это корень, возвращаем index.html
    if not path or path == '':
        path = 'index.html'
    
    # Безопасность: только файлы из webapp директории
    if '..' in path or path.startswith('/'):
        return Response(text='Forbidden', status=403)
    
    file_path = Path(WEBAPP_PATH) / path
    
    # Специальная обработка для фотографии - проверяем в res/
    if path == 'welcome_photo.jpeg' or path == 'wedding_photo.jpg' or path.endswith('/welcome_photo.jpeg') or path.endswith('/wedding_photo.jpg'):
        photo_path = Path(WEBAPP_PHOTO_PATH)
        if photo_path.exists():
            file_path = photo_path
        else:
            # Если фото нет, возвращаем 404
            return Response(text='Photo not found', status=404)
    
    # Если это директория или файл не существует, возвращаем index.html
    if file_path.is_dir() or (not file_path.exists() and path != 'welcome_photo.jpeg' and path != 'wedding_photo.jpg'):
        file_path = Path(WEBAPP_PATH) / 'index.html'
    
    if not file_path.exists():
        return Response(text='File not found', status=404)
    
    # Определяем content-type
    content_type = 'text/html'
    if path.endswith('.css'):
        content_type = 'text/css'
    elif path.endswith('.js'):
        content_type = 'application/javascript'
    elif path.endswith('.jpg') or path.endswith('.jpeg'):
        content_type = 'image/jpeg'
    elif path.endswith('.png'):
        content_type = 'image/png'
    elif path.endswith('.svg'):
        content_type = 'image/svg+xml'
    
    # Для фотографии из res/ всегда image/jpeg
    if file_path == Path(WEBAPP_PHOTO_PATH):
        content_type = 'image/jpeg'
    
    try:
        async with aiofiles.open(file_path, 'rb') as f:
            content = await f.read()
        return Response(body=content, content_type=content_type)
    except Exception as e:
        logger.error(f"Error serving file {path}: {e}")
        return Response(text='Internal server error', status=500)

async def init_app():
    """Инициализация приложения"""
    app = web.Application()
    
    # API routes (должны быть первыми)
    api = await init_api()
    app.add_subapp('/api', api)
    
    # Static files для Mini App (все остальные пути)
    app.router.add_get('/', lambda r: serve_static(r))
    app.router.add_get('/{path:.*}', serve_static)
    
    return app

async def start_web_server():
    """Запуск веб-сервера"""
    app = await init_app()
    runner = web.AppRunner(app)
    await runner.setup()
    
    port = int(os.getenv("PORT", 10000))  # Render использует порт 10000
    site = web.TCPSite(runner, '0.0.0.0', port)
    await site.start()
    
    logger.info(f"🌐 Веб-сервер запущен на порту {port}")
    logger.info(f"📱 Mini App доступен по адресу: http://localhost:{port}")

async def main():
    """Главная функция"""
    try:
        # Инициализация бота
        bot = await init_bot()
        if bot is None:
            logger.error("❌ Не удалось инициализировать бота")
            logger.error("Проверьте переменные окружения на Render")
            return
        
        # Устанавливаем функцию уведомлений в API
        set_notify_function(notify_admins)
        
        # Запуск веб-сервера
        await start_web_server()
        
        # Запуск бота только если есть переменная PORT (значит на сервере)
        # Это предотвращает конфликт с локальным запуском
        if os.getenv("PORT"):
            logger.info("🚀 Все сервисы запущены!")
            logger.info("🤖 Запуск бота (polling)...")
            await dp.start_polling(bot, allowed_updates=["message", "callback_query"])
        else:
            logger.warning("⚠️ PORT не установлен - бот не запущен (вероятно локальный запуск)")
            logger.info("🌐 Только веб-сервер запущен")
            # Держим сервер запущенным
            import asyncio
            while True:
                await asyncio.sleep(3600)  # Спим час за раз
    except Exception as e:
        logger.error(f"❌ Критическая ошибка: {e}")
        import traceback
        traceback.print_exc()
        raise

if __name__ == "__main__":
    asyncio.run(main())

