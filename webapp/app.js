// Инициализация Telegram Web App
const tg = window.Telegram.WebApp;
tg.ready();
tg.expand();

// Загружаем конфигурацию с сервера
let CONFIG = {
    weddingDate: '2026-06-06',
    groomName: 'Валентин',
    brideName: 'Мария',
    apiUrl: window.location.origin + '/api'
};

// Загружаем конфигурацию
async function loadConfig() {
    try {
        const response = await fetch(`${CONFIG.apiUrl}/config`);
        if (response.ok) {
            const data = await response.json();
            CONFIG = { ...CONFIG, ...data };
            updateUI();
        }
    } catch (error) {
        console.log('Используем конфигурацию по умолчанию');
    }
}

// Обновляем UI с конфигурацией
function updateUI() {
    const coupleNames = `${CONFIG.groomName} и ${CONFIG.brideName}`;
    document.getElementById('coupleNames').textContent = coupleNames;
    document.getElementById('coupleNamesFinal').textContent = coupleNames;
    
    const date = formatDate(CONFIG.weddingDate);
    document.getElementById('weddingDateHero').textContent = date;
    document.getElementById('calendarDate').textContent = date.split('.')[0];
    
    const monthNames = ['Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь', 
                       'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь'];
    const weddingDate = new Date(CONFIG.weddingDate);
    document.getElementById('monthName').textContent = monthNames[weddingDate.getMonth()];
}

// Обратный отсчет
function updateCountdown() {
    const weddingDate = new Date(CONFIG.weddingDate);
    const now = new Date();
    const diff = weddingDate - now;
    
    if (diff <= 0) {
        document.getElementById('months').textContent = '0';
        document.getElementById('days').textContent = '0';
        document.getElementById('hours').textContent = '0';
        document.getElementById('minutes').textContent = '0';
        return;
    }
    
    const months = Math.floor(diff / (1000 * 60 * 60 * 24 * 30));
    const days = Math.floor((diff % (1000 * 60 * 60 * 24 * 30)) / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    
    document.getElementById('months').textContent = months;
    document.getElementById('days').textContent = days;
    document.getElementById('hours').textContent = hours;
    document.getElementById('minutes').textContent = minutes;
}

function formatDate(dateString) {
    const date = new Date(dateString);
    const day = String(date.getDate()).padStart(2, '0');
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const year = date.getFullYear();
    return `${day}.${month}.${year}`;
}

// Инициализация
loadConfig();
updateCountdown();
setInterval(updateCountdown, 60000);

// Обработчик формы RSVP
document.getElementById('guestForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const firstName = document.getElementById('firstName').value.trim();
    const lastName = document.getElementById('lastName').value.trim();
    const personsCount = parseInt(document.getElementById('personsCount').value) || 1;
    
    if (firstName.length < 2 || lastName.length < 2) {
        tg.showAlert('Пожалуйста, введите корректные имя и фамилию');
        return;
    }
    
    // Получаем данные пользователя из Telegram
    const user = tg.initDataUnsafe?.user;
    const userId = user?.id;
    const username = user?.username;
    
    // Отправляем данные на сервер
    try {
        const response = await fetch(`${CONFIG.apiUrl}/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                userId: userId,
                firstName: firstName,
                lastName: lastName,
                username: username,
                personsCount: personsCount,
                initData: tg.initData
            })
        });
        
        if (response.ok) {
            const data = await response.json();
            // Показываем анкету
            document.getElementById('rsvpSection').style.display = 'none';
            document.getElementById('questionnaireSection').style.display = 'block';
            document.getElementById('questionnaireSection').scrollIntoView({ behavior: 'smooth' });
        } else {
            throw new Error('Ошибка при регистрации');
        }
    } catch (error) {
        console.error('Error:', error);
        tg.showAlert('Ошибка при отправке данных. Попробуйте позже.');
    }
});

// Обработчик анкеты
document.getElementById('questionnaireForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const formData = new FormData(e.target);
    const transfer = formData.get('transfer');
    const food = formData.getAll('food');
    const alcohol = document.getElementById('alcohol').value;
    
    const user = tg.initDataUnsafe?.user;
    const userId = user?.id;
    
    try {
        const response = await fetch(`${CONFIG.apiUrl}/questionnaire`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                userId: userId,
                transfer: transfer,
                food: food,
                alcohol: alcohol,
                initData: tg.initData
            })
        });
        
        if (response.ok) {
            const data = await response.json();
            showConfirmation(data.firstName, data.lastName, data.guestsCount || 0);
        } else {
            throw new Error('Ошибка при отправке анкеты');
        }
    } catch (error) {
        console.error('Error:', error);
        tg.showAlert('Ошибка при отправке данных. Попробуйте позже.');
    }
});

// Кнопка "Отклонить"
document.getElementById('declineBtn').addEventListener('click', () => {
    tg.showConfirm('Вы уверены, что не сможете присутствовать?', (confirmed) => {
        if (confirmed) {
            tg.showAlert('Мы будем скучать без вас! 💔');
        }
    });
});

function showConfirmation(firstName, lastName, guestsCount) {
    document.getElementById('questionnaireSection').style.display = 'none';
    document.getElementById('confirmationSection').style.display = 'block';
    document.getElementById('guestName').textContent = `${firstName} ${lastName}`;
    document.getElementById('guestsCount').textContent = guestsCount;
    document.getElementById('confirmationSection').scrollIntoView({ behavior: 'smooth' });
    
    // Вибрация
    if (tg.HapticFeedback) {
        tg.HapticFeedback.notificationOccurred('success');
    }
}

// Настройка темы Telegram
if (tg.colorScheme === 'dark') {
    document.body.style.background = '#1a1a1a';
    document.body.style.color = '#fff';
}

// Плавная прокрутка при загрузке
window.addEventListener('load', () => {
    window.scrollTo(0, 0);
});
