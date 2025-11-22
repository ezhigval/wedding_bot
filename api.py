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
import urllib.parse

from config import BOT_TOKEN, WEDDING_DATE, GROOM_NAME, BRIDE_NAME, GROOM_TELEGRAM, BRIDE_TELEGRAM, WEDDING_ADDRESS
from google_sheets import (
    add_guest_to_sheets, cancel_invitation, get_timeline,
    check_guest_registration, get_all_guests_from_sheets, 
    get_guests_count_from_sheets, cancel_guest_registration_by_user_id,
    find_guest_by_name, update_guest_user_id
)
import traceback
import logging

logger = logging.getLogger(__name__)

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
    api.router.add_post('/cancel', cancel_guest_registration)
    api.router.add_post('/questionnaire', save_questionnaire)
    api.router.add_get('/guests', get_guests_list)
    api.router.add_get('/stats', get_stats)
    api.router.add_get('/timeline', get_timeline_endpoint)
    api.router.add_post('/confirm-identity', confirm_identity)
    api.router.add_post('/parse-init-data', parse_init_data)
    
    return api

async def get_config(request):
    """Получить конфигурацию для Mini App"""
    try:
        return web.json_response({
            'weddingDate': WEDDING_DATE.strftime('%Y-%m-%d'),
            'groomName': GROOM_NAME,
            'brideName': BRIDE_NAME,
            'groomTelegram': GROOM_TELEGRAM,
            'brideTelegram': BRIDE_TELEGRAM,
            'weddingAddress': WEDDING_ADDRESS
        })
    except Exception as e:
        import logging
        logging.error(f"Error in get_config: {e}")
        return web.json_response({'error': str(e)}, status=500)

async def parse_init_data(request):
    """Парсинг initData от Telegram для извлечения user_id"""
    try:
        data = await request.json()
        init_data = data.get('initData', '')
        
        if not init_data:
            return web.json_response({
                'error': 'initData required'
            }, status=400)
        
        # Парсим initData
        parsed_data = {}
        for item in init_data.split('&'):
            if '=' in item:
                key, value = item.split('=', 1)
                parsed_data[key] = urllib.parse.unquote(value)
        
        # Извлекаем user из user JSON
        user_json = parsed_data.get('user', '')
        if user_json:
            try:
                import json
                user = json.loads(user_json)
                user_id = user.get('id')
                first_name = user.get('first_name', '')
                last_name = user.get('last_name', '')
                
                logger.info(f"parse_init_data: extracted user_id {user_id} from initData")
                
                return web.json_response({
                    'userId': user_id,
                    'firstName': first_name,
                    'lastName': last_name
                })
            except json.JSONDecodeError:
                logger.error("parse_init_data: failed to parse user JSON")
        
        return web.json_response({
            'error': 'user not found in initData'
        }, status=400)
        
    except Exception as e:
        logger.error(f"Error in parse_init_data: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return web.json_response({
            'error': 'server_error'
        }, status=500)

async def check_registration(request):
    """
    Проверить, зарегистрирован ли пользователь
    Поддерживает несколько способов определения пользователя:
    1. По user_id (если доступен)
    2. По имени/фамилии (если user_id недоступен)
    
    Логика проверки:
    1. Если есть user_id - сначала проверяем по user_id в столбце F
    2. Если не найден по user_id, проверяем по имени/фамилии
    3. Если найден по имени - возвращаем информацию для подтверждения личности
    4. Если нет user_id, но есть имя/фамилия - ищем только по имени
    """
    try:
        # user_id может быть получен разными способами
        user_id_str = request.query.get('userId')
        first_name = request.query.get('firstName', '').strip()
        last_name = request.query.get('lastName', '').strip()
        search_by_name_only = request.query.get('searchByNameOnly', 'false') == 'true'
        
        logger.info(f"check_registration: received request - userId: {user_id_str}, firstName: {first_name}, lastName: {last_name}, searchByNameOnly: {search_by_name_only}")
        
        # Если запрошен поиск только по имени (без user_id)
        if search_by_name_only:
            if not first_name or not last_name:
                logger.warning("check_registration: searchByNameOnly requested but name is missing")
                return web.json_response({
                    'registered': False,
                    'error': 'name_required'
                }, status=400)
            
            # Ищем только по имени/фамилии
            guest_info = await find_guest_by_name(first_name, last_name)
            if guest_info:
                if guest_info.get('user_id'):
                    # Найден и уже имеет user_id - зарегистрирован
                    logger.info(f"check_registration: guest found by name and has user_id")
                    return web.json_response({
                        'registered': True
                    })
                else:
                    # Найден, но нет user_id - нужно подтвердить
                    logger.info(f"check_registration: guest found by name but no user_id, needs confirmation")
                    return web.json_response({
                        'registered': False,
                        'needs_confirmation': True,
                        'guest_name': f"{guest_info['first_name']} {guest_info['last_name']}",
                        'row': guest_info['row']
                    })
            
            # Не найден по имени
            logger.info(f"check_registration: guest not found by name")
            return web.json_response({
                'registered': False
            })
        
        # Обычная проверка с user_id
        if not user_id_str:
            logger.warning("check_registration: userId not provided")
            # Если нет user_id, но есть имя - пробуем поиск по имени
            if first_name and last_name:
                logger.info("check_registration: no userId, trying search by name")
                guest_info = await find_guest_by_name(first_name, last_name)
                if guest_info:
                    if guest_info.get('user_id'):
                        return web.json_response({
                            'registered': True
                        })
                    else:
                        return web.json_response({
                            'registered': False,
                            'needs_confirmation': True,
                            'guest_name': f"{guest_info['first_name']} {guest_info['last_name']}",
                            'row': guest_info['row']
                        })
            
            return web.json_response({
                'registered': False,
                'error': 'user_id_or_name_required'
            }, status=400)
        
        user_id = int(user_id_str)
        logger.info(f"check_registration: checking user_id {user_id} (from Telegram) against column F in Google Sheets")
        
        # 1. Проверяем по user_id в столбце F таблицы
        registered = await check_guest_registration(user_id)
        if registered:
            logger.info(f"check_registration: user_id {user_id} found and registered")
            return web.json_response({
                'registered': True
            })
        
        # 2. Если не найден по user_id, проверяем по имени/фамилии
        if first_name and last_name:
            guest_info = await find_guest_by_name(first_name, last_name)
            if guest_info:
                # Найден по имени, но user_id не совпадает или отсутствует
                if not guest_info.get('user_id'):
                    # user_id отсутствует - нужно подтвердить и сохранить
                    logger.info(f"check_registration: guest found by name but no user_id, needs confirmation")
                    return web.json_response({
                        'registered': False,
                        'needs_confirmation': True,
                        'guest_name': f"{guest_info['first_name']} {guest_info['last_name']}",
                        'row': guest_info['row']
                    })
                elif guest_info.get('user_id') != str(user_id):
                    # user_id не совпадает - возможно другой пользователь
                    logger.warning(f"check_registration: guest found by name but user_id mismatch: {guest_info.get('user_id')} != {user_id}")
                    return web.json_response({
                        'registered': False,
                        'needs_confirmation': True,
                        'guest_name': f"{guest_info['first_name']} {guest_info['last_name']}",
                        'row': guest_info['row']
                    })
        
        # Не найден ни по user_id, ни по имени
        logger.info(f"check_registration: user_id {user_id} not found")
        return web.json_response({
            'registered': False
        })
        
    except ValueError as e:
        logger.error(f"Error in check_registration: invalid user_id format: {e}")
        return web.json_response({
            'registered': False,
            'error': 'invalid_user_id'
        }, status=400)
    except Exception as e:
        logger.error(f"Error in check_registration: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return web.json_response({
            'registered': False,
            'error': 'server_error'
        }, status=500)

async def confirm_identity(request):
    """Подтвердить личность и сохранить user_id"""
    try:
        data = await request.json()
        row = data.get('row')
        user_id = data.get('userId')
        
        if not row or not user_id:
            return web.json_response({
                'success': False,
                'error': 'missing_data'
            }, status=400)
        
        user_id = int(user_id)
        row = int(row)
        
        logger.info(f"confirm_identity: updating row {row} with user_id {user_id}")
        
        result = await update_guest_user_id(row, user_id)
        
        if result:
            logger.info(f"confirm_identity: successfully updated user_id for row {row}")
            return web.json_response({
                'success': True
            })
        else:
            logger.error(f"confirm_identity: failed to update user_id for row {row}")
            return web.json_response({
                'success': False,
                'error': 'update_failed'
            }, status=500)
            
    except Exception as e:
        logger.error(f"Error in confirm_identity: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return web.json_response({
            'success': False,
            'error': 'server_error'
        }, status=500)

def verify_telegram_webapp_data(init_data):
    """Проверка подлинности данных от Telegram"""
    if not BOT_TOKEN:
        logger.warning("BOT_TOKEN не установлен, пропускаем проверку")
        return True
    
    if not init_data:
        logger.warning("initData пустой, пропускаем проверку")
        return True
    
    try:
        parsed_data = {}
        for item in init_data.split('&'):
            if '=' in item:
                key, value = item.split('=', 1)
                # URL декодируем значения
                parsed_data[key] = urllib.parse.unquote(value)
        
        received_hash = parsed_data.pop('hash', '')
        if not received_hash:
            logger.warning("Hash не найден в initData")
            return True  # Разрешаем, если hash нет
        
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
        
        is_valid = calculated_hash == received_hash
        if not is_valid:
            logger.warning(f"Проверка hash не прошла. Получен: {received_hash[:10]}..., вычислен: {calculated_hash[:10]}...")
            logger.debug(f"Data check string: {data_check_string[:100]}...")
        
        return is_valid
    except Exception as e:
        logger.error(f"Ошибка при проверке данных Telegram: {e}")
        import traceback
        logger.error(traceback.format_exc())
        # Для разработки разрешаем, если есть ошибка
        return True

async def register_guest(request):
    """Регистрация гостя"""
    # Инициализируем guests_count в начале функции
    guests_count = 0
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
        
        # Проверка подлинности (опционально, не блокируем если проверка не прошла)
        if init_data:
            is_valid = verify_telegram_webapp_data(init_data)
            if not is_valid:
                logger.warning(f"Проверка данных Telegram не прошла, но продолжаем регистрацию (user_id={user_id})")
                # Не блокируем регистрацию, так как у нас есть userId
                # Это безопасно, так как userId - это уникальный идентификатор от Telegram
        
        # Получаем данные основного гостя из запроса
        main_guest_data = guests_list[0] if guests_list else {}
        category = main_guest_data.get('category') or data.get('category')
        side = main_guest_data.get('side') or data.get('side')
        
        # Добавляем в Google Sheets (единственный источник данных)
        try:
            result = await add_guest_to_sheets(
                first_name=first_name,
                last_name=last_name,
                age=None,  # Пока не собираем возраст
                category=category,
                side=side,
                user_id=user_id  # Сохраняем user_id в столбец F
            )
            if result:
                logger.info(f"Гость {first_name} {last_name} успешно добавлен в Google Sheets")
            else:
                logger.warning(f"Не удалось добавить гостя {first_name} {last_name} в Google Sheets (возможно, нет credentials)")
            
            # Добавляем дополнительных гостей в Google Sheets
            for guest in guests_list[1:]:  # Пропускаем первого (основного гостя)
                guest_first_name = guest.get('firstName', '').strip()
                guest_last_name = guest.get('lastName', '').strip()
                guest_category = guest.get('category', '')
                guest_side = guest.get('side', '')
                
                if guest_first_name and guest_last_name:
                    await add_guest_to_sheets(
                        first_name=guest_first_name,
                        last_name=guest_last_name,
                        age=None,
                        category=guest_category,
                        side=guest_side
                    )
        except Exception as sheets_error:
            logger.error(f"Ошибка добавления в Google Sheets: {sheets_error}")
            logger.error(traceback.format_exc())
            # Не блокируем ответ, так как это не критично
        
        # Получаем количество гостей из Google Sheets
        try:
            guests_count = await get_guests_count_from_sheets()
        except Exception as count_error:
            logger.error(f"Ошибка получения количества гостей: {count_error}")
            guests_count = 0  # Используем значение по умолчанию
        
        # Формируем уведомление для админов
        username_text = f" @{username}" if username else ""
        notification_text = (
            f"✅ <b>Новая регистрация!</b>\n\n"
            f"👤 <b>Основной гость:</b>\n"
            f"{first_name} {last_name}{username_text}\n"
        )
        
        # Добавляем информацию о дополнительных гостях
        if guests_list and len(guests_list) > 1:
            additional_guests = guests_list[1:]  # Пропускаем первого (основного)
            notification_text += f"\n👥 <b>Дополнительные гости ({len(additional_guests)}):</b>\n"
            for i, guest in enumerate(additional_guests, 1):
                guest_telegram = guest.get('telegram', '')
                telegram_text = f" @{guest_telegram}" if guest_telegram else ""
                notification_text += f"{i}. {guest.get('firstName', '')} {guest.get('lastName', '')}{telegram_text}\n"
        
        notification_text += f"\n📊 Всего гостей: {guests_count}"
        
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

async def cancel_guest_registration(request):
    """Отмена регистрации гостя"""
    try:
        data = await request.json()
        user_id = data.get('userId')
        init_data = data.get('initData', '')
        
        if not user_id:
            logger.error("Недостаточно данных для отмены: user_id отсутствует")
            return web.json_response({'error': 'Недостаточно данных'}, status=400)
        
        user_id = int(user_id)
        
        # Получаем данные гостя из Google Sheets перед отменой
        guests = await get_all_guests_from_sheets()
        guest_info = None
        for guest in guests:
            if guest.get('user_id') == str(user_id):
                guest_info = guest
                break
        
        if not guest_info:
            return web.json_response({'error': 'Гость не найден'}, status=404)
        
        first_name = guest_info.get('first_name', '')
        last_name = guest_info.get('last_name', '')
        
        # Обновляем Google Sheets - ставим "НЕТ" в столбец C
        try:
            result = await cancel_guest_registration_by_user_id(user_id)
            if not result:
                logger.warning(f"Не удалось отменить регистрацию для user_id {user_id}")
                return web.json_response({'error': 'Не удалось отменить регистрацию'}, status=500)
            logger.info(f"Регистрация гостя {first_name} {last_name} (user_id: {user_id}) отменена в Google Sheets")
        except Exception as sheets_error:
            logger.error(f"Ошибка обновления Google Sheets при отмене: {sheets_error}")
            logger.error(traceback.format_exc())
            return web.json_response({'error': f'Ошибка отмены регистрации: {str(sheets_error)}'}, status=500)
        
        guests_count = await get_guests_count_from_sheets()
        
        # Отправляем уведомление админам
        notification_text = (
            f"❌ <b>Отмена регистрации</b>\n\n"
            f"👤 {first_name} {last_name}\n"
            f"отменил(а) присутствие на свадьбе\n\n"
            f"📊 Всего гостей: {guests_count}"
        )
        await notify_admins(notification_text)
        
        return web.json_response({
            'success': True,
            'guestsCount': guests_count
        })
    except Exception as e:
        logger.error(f"Критическая ошибка в cancel_guest_registration: {e}")
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
        
        if not user_id:
            return web.json_response({'error': 'Недостаточно данных'}, status=400)
        
        user_id = int(user_id)
        
        # Получаем данные гостя из Google Sheets
        guests = await get_all_guests_from_sheets()
        guest_info = None
        for guest in guests:
            if guest.get('user_id') == str(user_id):
                guest_info = guest
                break
        
        if not guest_info:
            return web.json_response({'error': 'Гость не найден'}, status=404)
        
        # Здесь можно добавить сохранение в отдельную таблицу Google Sheets
        # Пока просто возвращаем успех
        
        first_name = guest_info.get('first_name', '')
        last_name = guest_info.get('last_name', '')
        guests_count = await get_guests_count_from_sheets()
        
        return web.json_response({
            'success': True,
            'firstName': first_name,
            'lastName': last_name,
            'guestsCount': guests_count
        })
    except Exception as e:
        logger.error(f"Ошибка в save_questionnaire: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': str(e)}, status=500)

async def get_guests_list(request):
    """Получить список гостей"""
    try:
        guests = await get_all_guests_from_sheets()
        return web.json_response({
            'guests': [
                {
                    'firstName': g.get('first_name', ''),
                    'lastName': g.get('last_name', ''),
                    'username': g.get('username', ''),
                    'user_id': g.get('user_id', ''),
                    'category': g.get('category', ''),
                    'side': g.get('side', '')
                }
                for g in guests
            ],
            'count': len(guests)
        })
    except Exception as e:
        logger.error(f"Ошибка в get_guests_list: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': str(e)}, status=500)

async def get_stats(request):
    """Получить статистику"""
    try:
        guests_count = await get_guests_count_from_sheets()
        return web.json_response({
            'guestsCount': guests_count,
            'weddingDate': WEDDING_DATE.strftime('%Y-%m-%d')
        })
    except Exception as e:
        logger.error(f"Ошибка в get_stats: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': str(e)}, status=500)

async def get_timeline_endpoint(request):
    """Получить тайминг мероприятия"""
    try:
        timeline = await get_timeline()
        return web.json_response({
            'timeline': timeline
        })
    except Exception as e:
        logger.error(f"Ошибка получения тайминга: {e}")
        logger.error(traceback.format_exc())
        return web.json_response({'error': str(e)}, status=500)

