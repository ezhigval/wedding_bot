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

from config import DB_PATH, BOT_TOKEN, WEDDING_DATE, GROOM_NAME, BRIDE_NAME
from database import init_db, add_guest, get_guest, get_all_guests, get_guests_count

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
            'brideName': BRIDE_NAME
        })
    except Exception as e:
        import logging
        logging.error(f"Error in get_config: {e}")
        return web.json_response({'error': str(e)}, status=500)

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
        persons_count = data.get('personsCount', 1)
        init_data = data.get('initData', '')
        
        if not user_id or not first_name or not last_name:
            return web.json_response({'error': 'Недостаточно данных'}, status=400)
        
        if len(first_name) < 2 or len(last_name) < 2:
            return web.json_response({'error': 'Имя и фамилия должны быть не менее 2 символов'}, status=400)
        
        # Проверка подлинности (опционально)
        if init_data and not verify_telegram_webapp_data(init_data):
            return web.json_response({'error': 'Неверные данные'}, status=403)
        
        # Сохранение в базу данных
        await add_guest(
            user_id=user_id,
            first_name=first_name,
            last_name=last_name,
            username=username
        )
        
        guests_count = await get_guests_count()
        
        return web.json_response({
            'success': True,
            'guestsCount': guests_count,
            'firstName': first_name,
            'lastName': last_name
        })
    except Exception as e:
        return web.json_response({'error': str(e)}, status=500)

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

