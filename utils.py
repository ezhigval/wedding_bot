from datetime import datetime
from config import WEDDING_DATE

def get_time_until_wedding():
    """Возвращает строку с обратным отсчетом до свадьбы"""
    now = datetime.now()
    delta = WEDDING_DATE - now
    
    if delta.total_seconds() <= 0:
        return "🎉 Свадьба уже прошла!"
    
    days = delta.days
    hours, remainder = divmod(delta.seconds, 3600)
    minutes, seconds = divmod(remainder, 60)
    
    if days > 0:
        return f"⏰ Осталось {days} дней, {hours} часов, {minutes} минут"
    else:
        return f"⏰ Осталось {hours} часов, {minutes} минут"

def format_wedding_date():
    """Форматирует дату свадьбы для отображения"""
    return WEDDING_DATE.strftime("%d.%m.%Y")

