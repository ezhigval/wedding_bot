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

# Импортируем бот только один раз
# ВАЖНО: bot.py не должен запускаться напрямую в продакшене
# Используется только server.py

# Флаги для отслеживания состояния (защита от дублирования)
_polling_started = False
_server_initialized = False
_bot_instance = None

logger_import = logging.getLogger(__name__)
logger_import.info("=" * 60)
logger_import.info("📦 ИМПОРТ МОДУЛЕЙ")
logger_import.info(f"🆔 Process ID: {os.getpid()}")
logger_import.info(f"🕐 Время: {__import__('datetime').datetime.now().isoformat()}")
logger_import.info("=" * 60)

from bot import dp, init_bot, notify_admins
from api import init_api, set_notify_function
from config import WEBAPP_PATH, WEBAPP_PHOTO_PATH

logger_import.info("✅ Модули bot, api и config импортированы успешно")

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
        
        # Специальная обработка для Lottie файла из res/ (старый формат .lottie)
        if path == 'ring_animation.lottie' or path == 'res/ring_animation.lottie' or path.endswith('/ring_animation.lottie'):
            lottie_path = Path('res/ring_animation.lottie')
            if lottie_path.exists():
                file_path = lottie_path
            else:
                logger.warning(f"Lottie file not found: {lottie_path}")
                # Не возвращаем 404, продолжаем поиск в webapp/

        # Специальная обработка для видео ring_v2.mp4 из res/
        if path == 'ring_v2.mp4' or path == 'res/ring_v2.mp4' or path.endswith('/ring_v2.mp4'):
            video_path = Path('res') / 'ring_v2.mp4'
            if video_path.exists():
                file_path = video_path
            else:
                logger.warning(f"Ring video not found: {video_path}")
                return Response(text='Video not found', status=404)

        # Специальная обработка для Lottie JSON rings.json из res/
        if path == 'rings.json' or path == 'res/rings.json' or path.endswith('/rings.json'):
            rings_json_path = Path('res') / 'rings.json'
            if rings_json_path.exists():
                file_path = rings_json_path
            else:
                logger.warning(f"Lottie JSON not found: {rings_json_path}")
                return Response(text='Lottie JSON not found', status=404)
        
        # Если это директория или файл не существует, возвращаем index.html
        if file_path.is_dir() or (not file_path.exists()
                                  and path != 'welcome_photo.jpeg'
                                  and path != 'ring_animation.lottie'
                                  and path != 'res/ring_animation.lottie'
                                  and path != 'ring_animation.json'
                                  and path != 'rings.json'
                                  and path != 'res/rings.json'):
            file_path = Path(WEBAPP_PATH) / 'index.html'
        
        # Если index.html не существует, это критическая ошибка
        if not file_path.exists():
            logger.error(f"File not found: {file_path}")
            # Для index.html возвращаем базовую HTML страницу вместо 404
            if path == '' or path == 'index.html' or file_path.name == 'index.html':
                return Response(
                    text='<html><body><h1>Application Error</h1><p>Main page not found. Please contact administrator.</p></body></html>',
                    content_type='text/html',
                    status=500
                )
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
        elif path.endswith('.mp4'):
            content_type = 'video/mp4'
        elif path.endswith('.webm'):
            content_type = 'video/webm'
        elif path.endswith('.lottie') or path.endswith('.json'):
            # Lottie файлы могут быть в формате .lottie (бинарный) или .json
            # Для .lottie используем application/octet-stream, для .json - application/json
            if path.endswith('.lottie'):
                content_type = 'application/octet-stream'
            else:
                content_type = 'application/json'
        
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
    
    # Возвращаем runner для корректного завершения
    return runner

async def main():
    """Главная функция"""
    global _polling_started, _server_initialized, _bot_instance
    
    try:
        # Проверка: main() не должен вызываться дважды
        if _server_initialized:
            logger.error("🚨 КРИТИЧЕСКАЯ ОШИБКА: main() уже был вызван!")
            logger.error(f"   Process ID: {os.getpid()}")
            logger.error("   Это означает двойной запуск сервера")
            return
        
        logger.info("=" * 60)
        logger.info("🚀 НАЧАЛО ИНИЦИАЛИЗАЦИИ СЕРВЕРА")
        logger.info(f"🆔 Process ID: {os.getpid()}")
        logger.info(f"🕐 Время: {__import__('datetime').datetime.now().isoformat()}")
        logger.info(f"🌍 PORT: {os.getenv('PORT')}")
        logger.info(f"🌍 RENDER: {os.getenv('RENDER')}")
        logger.info(f"📦 Dispatcher ID: {id(dp)}")
        logger.info("=" * 60)
        
        # Устанавливаем флаг инициализации
        _server_initialized = True
        
        # Проверяем, не запущен ли уже polling (на всякий случай)
        if _polling_started:
            logger.error("🚨 КРИТИЧЕСКАЯ ОШИБКА: Polling уже был запущен!")
            logger.error("   Это не должно происходить. Проверьте код.")
            return
        
        # Проверяем состояние Dispatcher
        logger.info("🔍 Проверка состояния Dispatcher...")
        try:
            # Проверяем различные атрибуты, которые могут указывать на активный polling
            polling_active = False
            if hasattr(dp, '_polling'):
                polling_active = bool(dp._polling)
                if polling_active:
                    logger.warning(f"⚠️ dp._polling = {dp._polling}")
            
            # Проверяем другие возможные индикаторы
            if hasattr(dp, '_running') and dp._running:
                polling_active = True
                logger.warning(f"⚠️ dp._running = {dp._running}")
            
            if polling_active:
                logger.warning("⚠️ Обнаружен активный polling в Dispatcher!")
                logger.warning("   Пытаемся остановить...")
                try:
                    await dp.stop_polling()
                    await asyncio.sleep(1)
                    logger.info("✅ Polling остановлен")
                except Exception as stop_error:
                    logger.warning(f"⚠️ Не удалось остановить polling: {stop_error}")
                    logger.warning("   Возможно, polling не был запущен")
        except Exception as check_error:
            logger.debug(f"Проверка Dispatcher: {check_error}")
        logger.info("✅ Dispatcher проверен")
        
        # Инициализация бота
        logger.info("🤖 Инициализация бота...")
        if _bot_instance is not None:
            logger.warning("⚠️ Экземпляр бота уже существует! Используем его")
            bot = _bot_instance
        else:
            bot = await init_bot()
            if bot is None:
                logger.error("❌ Не удалось инициализировать бота")
                logger.error("Проверьте переменные окружения на Render")
                return
            _bot_instance = bot
            logger.info(f"✅ Бот инициализирован (ID: {id(bot)})")
        
        # Устанавливаем функцию уведомлений в API
        logger.info("📡 Настройка API...")
        set_notify_function(notify_admins)
        
        # Запуск веб-сервера
        logger.info("🌐 Запуск веб-сервера...")
        runner = await start_web_server()
        
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
                    logger.error(f"   4. Бот запускается несколько раз в одном процессе")
                    logger.error(f"   Решение: Проверьте на Render, нет ли дублирующихся сервисов")
                return True
            
            # Создаем фильтр для логирования конфликтов
            class ConflictFilter(logging.Filter):
                def filter(self, record):
                    return log_conflict_error(record)
            
            conflict_filter = ConflictFilter()
            aiogram_logger.addFilter(conflict_filter)
            
            # Проверяем глобальный флаг
            if _polling_started:
                logger.error("🚨 КРИТИЧЕСКАЯ ОШИБКА: Попытка запустить polling второй раз!")
                logger.error("   Это не должно происходить. Проверьте код на наличие двойных вызовов.")
                logger.error(f"   Process ID: {os.getpid()}")
                logger.error(f"   Dispatcher ID: {id(dp)}")
                logger.error(f"   Bot ID: {id(bot)}")
                return
            
            # Финальная проверка состояния Dispatcher перед запуском
            logger.info("🔍 Финальная проверка Dispatcher перед запуском polling...")
            polling_detected = False
            try:
                # Проверяем все возможные индикаторы активного polling
                if hasattr(dp, '_polling') and dp._polling:
                    polling_detected = True
                    logger.warning(f"⚠️ dp._polling = {dp._polling}")
                
                if hasattr(dp, '_running') and dp._running:
                    polling_detected = True
                    logger.warning(f"⚠️ dp._running = {dp._running}")
                
                if polling_detected:
                    logger.warning("⚠️ Обнаружен возможный активный polling")
                    logger.warning("   Пытаемся безопасно остановить...")
                    try:
                        # Пытаемся остановить только если действительно запущен
                        await dp.stop_polling()
                        await asyncio.sleep(2)
                        logger.info("✅ Polling остановлен (если был запущен)")
                    except Exception as stop_error:
                        # Если ошибка "Polling is not started" - это нормально
                        if "not started" in str(stop_error).lower():
                            logger.info("ℹ️ Polling не был запущен (это нормально)")
                        else:
                            logger.warning(f"⚠️ Ошибка при остановке: {stop_error}")
            except Exception as check_error:
                logger.debug(f"Проверка Dispatcher: {check_error}")
            
            # Проверяем, что бот существует
            if bot is None:
                logger.error("🚨 КРИТИЧЕСКАЯ ОШИБКА: Бот не инициализирован!")
                return
            
            # Добавляем задержку перед запуском, чтобы избежать конфликтов
            logger.info("⏳ Задержка 1 секунда перед запуском polling...")
            await asyncio.sleep(1)
            
            logger.info("=" * 60)
            logger.info("🤖 ЗАПУСК БОТА (POLLING)")
            logger.info(f"🆔 Process ID: {os.getpid()}")
            logger.info(f"🕐 Время: {__import__('datetime').datetime.now().isoformat()}")
            logger.info(f"📦 Dispatcher ID: {id(dp)}")
            logger.info(f"🤖 Bot ID: {id(bot)}")
            logger.info("=" * 60)
            
            # Устанавливаем флаг
            _polling_started = True
            logger.info("✅ Флаг _polling_started установлен")
            
            try:
                # Запускаем polling (это блокирующая операция)
                # Важно: start_polling должен быть вызван только один раз
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
            while True:
                await asyncio.sleep(3600)  # Спим час за раз
    except Exception as e:
        logger.error(f"❌ Критическая ошибка: {e}")
        import traceback
        traceback.print_exc()
        raise

if __name__ == "__main__":
    # Проверяем, не запущен ли уже event loop
    try:
        loop = asyncio.get_running_loop()
        logger.error("🚨 КРИТИЧЕСКАЯ ОШИБКА: Event loop уже запущен!")
        logger.error("   Это может означать, что server.py импортируется в другом контексте")
        logger.error("   Process ID: %s", os.getpid())
    except RuntimeError:
        # Event loop не запущен - это нормально, запускаем
        logger.info("✅ Event loop не запущен, запускаем main()...")
    asyncio.run(main())

