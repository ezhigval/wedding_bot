"""
Простой API сервер для Mini App
Можно разместить на Railway, Render или другом хостинге
"""
from flask import Flask, request, jsonify
from flask_cors import CORS
import sqlite3
import os
from datetime import datetime
import hashlib
import hmac
import json

app = Flask(__name__)
CORS(app)  # Разрешаем запросы с любого домена

# Токен бота для проверки подлинности запросов
BOT_TOKEN = os.getenv("BOT_TOKEN", "")

# Путь к базе данных
DB_PATH = "data/wedding.db"

def init_db():
    """Инициализация базы данных"""
    os.makedirs("data", exist_ok=True)
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS guests (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER,
            first_name TEXT NOT NULL,
            last_name TEXT NOT NULL,
            username TEXT,
            confirmed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            UNIQUE(user_id)
        )
    """)
    conn.commit()
    conn.close()

def verify_telegram_webapp_data(init_data):
    """Проверка подлинности данных от Telegram"""
    if not BOT_TOKEN:
        return True  # Если токен не установлен, пропускаем проверку
    
    try:
        # Парсим данные
        parsed_data = {}
        for item in init_data.split('&'):
            key, value = item.split('=', 1)
            parsed_data[key] = value
        
        # Получаем hash и проверяем
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

@app.route('/api/register', methods=['POST'])
def register_guest():
    """Регистрация гостя"""
    try:
        data = request.json
        user_id = data.get('userId')
        first_name = data.get('firstName', '').strip()
        last_name = data.get('lastName', '').strip()
        username = data.get('username')
        init_data = data.get('initData', '')
        
        # Проверка данных
        if not user_id or not first_name or not last_name:
            return jsonify({'error': 'Недостаточно данных'}), 400
        
        if len(first_name) < 2 or len(last_name) < 2:
            return jsonify({'error': 'Имя и фамилия должны быть не менее 2 символов'}), 400
        
        # Проверка подлинности (опционально)
        if init_data and not verify_telegram_webapp_data(init_data):
            return jsonify({'error': 'Неверные данные'}), 403
        
        # Сохранение в базу данных
        conn = sqlite3.connect(DB_PATH)
        cursor = conn.cursor()
        cursor.execute("""
            INSERT OR REPLACE INTO guests (user_id, first_name, last_name, username)
            VALUES (?, ?, ?, ?)
        """, (user_id, first_name, last_name, username))
        conn.commit()
        
        # Получаем количество гостей
        cursor.execute("SELECT COUNT(*) FROM guests")
        guests_count = cursor.fetchone()[0]
        conn.close()
        
        return jsonify({
            'success': True,
            'guestsCount': guests_count
        })
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/guests', methods=['GET'])
def get_guests():
    """Получить список гостей"""
    try:
        conn = sqlite3.connect(DB_PATH)
        cursor = conn.cursor()
        cursor.execute("""
            SELECT first_name, last_name, username, confirmed_at 
            FROM guests 
            ORDER BY confirmed_at DESC
        """)
        guests = cursor.fetchall()
        conn.close()
        
        return jsonify({
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
        return jsonify({'error': str(e)}), 500

@app.route('/api/config', methods=['GET'])
def get_config():
    """Получить конфигурацию для Mini App"""
    return jsonify({
        'weddingDate': os.getenv("WEDDING_DATE", "2026-06-05"),
        'groomName': os.getenv("GROOM_NAME", "Валентин"),
        'brideName': os.getenv("BRIDE_NAME", "Мария")
    })

@app.route('/health', methods=['GET'])
def health():
    """Проверка работоспособности"""
    return jsonify({'status': 'ok'})

if __name__ == '__main__':
    init_db()
    port = int(os.getenv("PORT", 5000))
    app.run(host='0.0.0.0', port=port, debug=False)

