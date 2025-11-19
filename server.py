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
from config import WEBAPP_PATH, WEBAPP_PHOTO_PATH

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
    try:
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
                logger.warning(f"Photo not found: {WEBAPP_PHOTO_PATH}")
                return Response(text='Photo not found', status=404)
        
        # Если это директория или файл не существует, возвращаем index.html
        if file_path.is_dir() or (not file_path.exists() and path != 'welcome_photo.jpeg' and path != 'wedding_photo.jpg'):
            file_path = Path(WEBAPP_PATH) / 'index.html'
        
        if not file_path.exists():
            logger.error(f"File not found: {file_path}")
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
        
        async with aiofiles.open(file_path, 'rb') as f:
            content = await f.read()
        return Response(body=content, content_type=content_type)
    except Exception as e:
        logger.error(f"Error serving file {path}: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return Response(text=f'Internal server error: {str(e)}', status=500)

async def root_handler(request):
    """Обработчик корневого пути"""
    return await serve_static(request)

async def init_app():
    """Инициализация приложения"""
    app = web.Application()
    
    # API routes (должны быть первыми)
    try:
        api = await init_api()
        app.add_subapp('/api', api)
    except Exception as e:
        logger.error(f"Ошибка инициализации API: {e}")
        import traceback
        traceback.print_exc()
        raise
    
    # Static files для Mini App (все остальные пути)
    app.router.add_get('/', root_handler)
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
            logger.info("🤖 Подготовка к запуску бота (polling)...")
            
            # Проверяем и отменяем webhook, если он установлен
            try:
                webhook_info = await bot.get_webhook_info()
                logger.info(f"📡 Информация о webhook: URL={webhook_info.url}, pending_updates={webhook_info.pending_update_count}")
                
                if webhook_info.url:
                    logger.warning(f"⚠️ Обнаружен активный webhook: {webhook_info.url}")
                    logger.info("🔄 Отменяю webhook для использования polling...")
                    await bot.delete_webhook(drop_pending_updates=True)
                    logger.info("✅ Webhook отменен")
                else:
                    logger.info("✅ Webhook не установлен, можно использовать polling")
            except Exception as e:
                logger.error(f"❌ Ошибка при проверке webhook: {e}")
                import traceback
                logger.error(traceback.format_exc())
            
            # Дополнительная информация для диагностики
            try:
                bot_info = await bot.get_me()
                logger.info(f"🤖 Информация о боте: @{bot_info.username} (ID: {bot_info.id})")
            except Exception as e:
                logger.error(f"❌ Ошибка при получении информации о боте: {e}")
            
            logger.info(f"🌍 Окружение: PORT={os.getenv('PORT')}, RENDER={os.getenv('RENDER')}")
            logger.info(f"🆔 Process ID: {os.getpid()}")
            logger.info(f"🕐 Время запуска: {__import__('datetime').datetime.now().isoformat()}")
            
            # Настраиваем детальное логирование для aiogram
            aiogram_logger = logging.getLogger('aiogram')
            aiogram_logger.setLevel(logging.INFO)
            
            # Добавляем обработчик для конфликтов
            def log_conflict_error(record):
                if 'TelegramConflictError' in str(record.msg) or 'Conflict' in str(record.msg):
                    logger.error(f"🚨 КОНФЛИКТ БОТОВ ОБНАРУЖЕН!")
                    logger.error(f"   Сообщение: {record.msg}")
                    logger.error(f"   Process ID: {os.getpid()}")
                    logger.error(f"   Время: {__import__('datetime').datetime.now().isoformat()}")
                    logger.error(f"   Возможные причины:")
                    logger.error(f"   1. На Render запущено несколько экземпляров сервиса")
                    logger.error(f"   2. Используется webhook вместо polling")
                    logger.error(f"   3. Старый экземпляр все еще работает")
                    logger.error(f"   Решение: Проверьте на Render, нет ли дублирующихся сервисов")
                return True
            
            # Создаем фильтр для логирования конфликтов
            class ConflictFilter(logging.Filter):
                def filter(self, record):
                    return log_conflict_error(record)
            
            conflict_filter = ConflictFilter()
            aiogram_logger.addFilter(conflict_filter)
            
            logger.info("🤖 Запуск бота (polling)...")
            
            try:
                await dp.start_polling(
                    bot, 
                    allowed_updates=["message", "callback_query"],
                    handle_as_tasks=False  # Обрабатываем последовательно
                )
            except Exception as e:
                logger.error(f"❌ Критическая ошибка при запуске polling: {e}")
                import traceback
                logger.error(traceback.format_exc())
                raise
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

