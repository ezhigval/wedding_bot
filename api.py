"""
API для Mini App и управления
"""
from aiohttp import web
from aiohttp.web import Response
import json
import sqlite3
import os
from datetime import datetime
import hashlib
import hmac

from config import DB_PATH, BOT_TOKEN, WEDDING_DATE, GROOM_NAME, BRIDE_NAME, GROOM_TELEGRAM, BRIDE_TELEGRAM
from database import init_db, add_guest, get_guest, get_all_guests, get_guests_count
from google_sheets import add_guest_to_sheets
import traceback

# Импортируем функцию уведомлений (будет доступна после инициализации бота)
_notify_admins_func = None

def set_notify_function(func):
    """Установка функции уведомлений из bot.py"""
    global _notify_admins_func
    _notify_admins_func = func

async def notify_admins(message_text):
    """Отправка уведомления админам"""
    if _notify_admins_func:
        await _notify_admins_func(message_text)

async def init_api():
    """Инициализация API"""
    api = web.Application()
    
    # CORS middleware
    @web.middleware
    async def cors_middleware(request, handler):
        if request.method == 'OPTIONS':
            return web.Response(
                headers={
                    'Access-Control-Allow-Origin': '*',
                    'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
                    'Access-Control-Allow-Headers': 'Content-Type',
                }
            )
        
        try:
            response = await handler(request)
            if isinstance(response, web.Response):
                response.headers['Access-Control-Allow-Origin'] = '*'
            return response
        except Exception as e:
            import logging
            logging.error(f"API error: {e}")
            return web.json_response({'error': str(e)}, status=500)
    
    api.middlewares.append(cors_middleware)
    
    # Routes
    api.router.add_get('/config', get_config)
    api.router.add_get('/check', check_registration)
    api.router.add_post('/register', register_guest)
    api.router.add_post('/questionnaire', save_questionnaire)
    api.router.add_get('/guests', get_guests_list)
    api.router.add_get('/stats', get_stats)
    
    return api

async def get_config(request):
    """Получить конфигурацию для Mini App"""
    try:
        return web.json_response({
            'weddingDate': WEDDING_DATE.strftime('%Y-%m-%d'),
            'groomName': GROOM_NAME,
            'brideName': BRIDE_NAME,
            'groomTelegram': GROOM_TELEGRAM,
            'brideTelegram': BRIDE_TELEGRAM
        })
    except Exception as e:
        import logging
        logging.error(f"Error in get_config: {e}")
        return web.json_response({'error': str(e)}, status=500)

async def check_registration(request):
    """Проверить, зарегистрирован ли пользователь"""
    try:
        user_id = request.query.get('userId')
        if not user_id:
            return web.json_response({'registered': False})
        
        user_id = int(user_id)
        guest = await get_guest(user_id)
        
        return web.json_response({
            'registered': guest is not None
        })
    except Exception as e:
        import logging
        logging.error(f"Error in check_registration: {e}")
        return web.json_response({'registered': False})

def verify_telegram_webapp_data(init_data):
    """Проверка подлинности данных от Telegram"""
    if not BOT_TOKEN:
        return True
    
    try:
        parsed_data = {}
        for item in init_data.split('&'):
            if '=' in item:
                key, value = item.split('=', 1)
                parsed_data[key] = value
        
        received_hash = parsed_data.pop('hash', '')
        data_check_string = '\n'.join(f"{k}={v}" for k, v in sorted(parsed_data.items()))
        
        secret_key = hmac.new(
            key=b"WebAppData",
            msg=BOT_TOKEN.encode(),
            digestmod=hashlib.sha256
        ).digest()
        
        calculated_hash = hmac.new(
            key=secret_key,
            msg=data_check_string.encode(),
            digestmod=hashlib.sha256
        ).hexdigest()
        
        return calculated_hash == received_hash
    except:
        return False

async def register_guest(request):
    """Регистрация гостя"""
    try:
        data = await request.json()
        user_id = data.get('userId')
        first_name = data.get('firstName', '').strip()
        last_name = data.get('lastName', '').strip()
        username = data.get('username')
        guests_list = data.get('guests', [])  # Список всех гостей
        init_data = data.get('initData', '')
        
        if not user_id or not first_name or not last_name:
            logger.error(f"Недостаточно данных: user_id={user_id}, first_name={first_name}, last_name={last_name}")
            return web.json_response({'error': 'Недостаточно данных'}, status=400)
        
        if len(first_name) < 2 or len(last_name) < 2:
            logger.error(f"Слишком короткие имена: first_name={first_name}, last_name={last_name}")
            return web.json_response({'error': 'Имя и фамилия должны быть не менее 2 символов'}, status=400)
        
        # Проверка подлинности (опционально)
        if init_data and not verify_telegram_webapp_data(init_data):
            logger.error("Неверные данные Telegram")
            return web.json_response({'error': 'Неверные данные'}, status=403)
        
        # Сохранение основного гостя в базу данных
        try:
            await add_guest(
                user_id=user_id,
                first_name=first_name,
                last_name=last_name,
                username=username
            )
        except Exception as db_error:
            logger.error(f"Ошибка сохранения в БД: {db_error}")
            logger.error(traceback.format_exc())
            return web.json_response({'error': f'Ошибка сохранения: {str(db_error)}'}, status=500)
        
        # Добавляем в Google Sheets (асинхронно, не блокируем ответ)
        try:
            await add_guest_to_sheets(
                first_name=first_name,
                last_name=last_name,
                age=None,  # Пока не собираем возраст
                category=None,  # Пока не собираем категорию
                side=None  # Пока не собираем сторону
            )
        except Exception as sheets_error:
            logger.warning(f"Ошибка добавления в Google Sheets (не критично): {sheets_error}")
        
        guests_count = await get_guests_count()
        
        # Отправляем уведомление админам
        username_text = f" @{username}" if username else ""
        notification_text = (
            f"✅ <b>Новая регистрация!</b>\n\n"
            f"👤 {first_name} {last_name}{username_text}\n"
            f"подтвердил(а) присутствие на свадьбе\n\n"
            f"📊 Всего гостей: {guests_count}"
        )
        await notify_admins(notification_text)
        
        return web.json_response({
            'success': True,
            'guestsCount': guests_count,
            'firstName': first_name,
            'lastName': last_name
        })
    except Exception as e:
        logger.error(f"Критическая ошибка в register_guest: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': f'Внутренняя ошибка сервера: {str(e)}'}, status=500)

async def save_questionnaire(request):
    """Сохранение анкеты"""
    try:
        data = await request.json()
        user_id = data.get('userId')
        transfer = data.get('transfer')
        food = data.get('food', [])
        alcohol = data.get('alcohol', '')
        
        # Здесь можно добавить сохранение в отдельную таблицу
        # Пока просто возвращаем успех
        
        guest = await get_guest(user_id)
        if guest:
            first_name, last_name, _ = guest
            guests_count = await get_guests_count()
            
            return web.json_response({
                'success': True,
                'firstName': first_name,
                'lastName': last_name,
                'guestsCount': guests_count
            })
        else:
            return web.json_response({'error': 'Гость не найден'}, status=404)
    except Exception as e:
        return web.json_response({'error': str(e)}, status=500)

async def get_guests_list(request):
    """Получить список гостей"""
    try:
        guests = await get_all_guests()
        return web.json_response({
            'guests': [
                {
                    'firstName': g[0],
                    'lastName': g[1],
                    'username': g[2],
                    'confirmedAt': g[3]
                }
                for g in guests
            ],
            'count': len(guests)
        })
    except Exception as e:
        return web.json_response({'error': str(e)}, status=500)

async def get_stats(request):
    """Получить статистику"""
    try:
        guests_count = await get_guests_count()
        return web.json_response({
            'guestsCount': guests_count,
            'weddingDate': WEDDING_DATE.strftime('%Y-%m-%d')
        })
    except Exception as e:
        return web.json_response({'error': str(e)}, status=500)

