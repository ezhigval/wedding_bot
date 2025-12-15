# Бэкап Python версии Wedding Bot

## Дата создания
$(date)

## Описание
Это полный бэкап Python версии свадебного бота перед миграцией на Go.

## Содержимое
- `server.py` - основной сервер (aiohttp)
- `bot.py` - Telegram бот (aiogram)
- `api.py` - API endpoints
- `config.py` - конфигурация
- `daily_reset.py` - ежедневный сброс игр
- `google_sheets.py` - работа с Google Sheets
- `keyboards.py` - клавиатуры бота
- `game_stats_cache.py` - кэш статистики игр
- `requirements.txt` - зависимости Python
- `start.sh` - скрипт запуска

## Восстановление
Для восстановления Python версии:
1. Скопируйте файлы обратно в проект
2. Установите зависимости: `pip install -r requirements.txt`
3. Запустите: `python server.py`

## Git ветка
Python версия также сохранена в ветке `python-dev` в GitHub:
```bash
git checkout python-dev
```

## Примечание
Этот бэкап создан перед полной миграцией на Go. 
Go версия находится в ветке `go-dev` и будет влита в `main`.

