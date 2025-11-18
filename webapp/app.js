// Инициализация Telegram Web App
const tg = window.Telegram.WebApp;
tg.ready();
tg.expand();

// Загружаем конфигурацию с сервера
let CONFIG = {
    weddingDate: '2026-06-06',
    groomName: 'Валентин',
    brideName: 'Мария',
    groomTelegram: 'ezhigval',
    brideTelegram: '',
    apiUrl: window.location.origin + '/api'
};

// Состояние гостей
let guests = [];
let maxGuests = 9;

// Загружаем конфигурацию
async function loadConfig() {
    try {
        const response = await fetch(`${CONFIG.apiUrl}/config`);
        if (response.ok) {
            const data = await response.json();
            CONFIG = { ...CONFIG, ...data };
            updateUI();
            updateContacts();
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
    
    const monthNames = ['Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь', 
                       'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь'];
    const weddingDate = new Date(CONFIG.weddingDate);
    const day = String(weddingDate.getDate()).padStart(2, '0');
    const year = weddingDate.getFullYear();
    document.getElementById('calendarDateFull').textContent = `${monthNames[weddingDate.getMonth()]} ${day} ${year}`;
}

// Обновляем контакты
function updateContacts() {
    document.getElementById('groomTelegram').textContent = `@${CONFIG.groomTelegram}`;
    document.getElementById('groomContact').href = `https://t.me/${CONFIG.groomTelegram}`;
    
    if (CONFIG.brideTelegram) {
        document.getElementById('brideTelegram').textContent = `@${CONFIG.brideTelegram}`;
        document.getElementById('brideContact').href = `https://t.me/${CONFIG.brideTelegram}`;
    } else {
        document.getElementById('brideContact').style.display = 'none';
    }
}

// Обратный отсчет с анимацией перелистывания
let previousValues = {
    months: null,
    days: null,
    hours: null,
    minutes: null
};

function updateCountdown() {
    const weddingDate = new Date(CONFIG.weddingDate);
    const now = new Date();
    const diff = weddingDate - now;
    
    if (diff <= 0) {
        setClockValue('months', 0);
        setClockValue('days', 0);
        setClockValue('hours', 0);
        setClockValue('minutes', 0);
        return;
    }
    
    const months = Math.floor(diff / (1000 * 60 * 60 * 24 * 30));
    const days = Math.floor((diff % (1000 * 60 * 60 * 24 * 30)) / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    
    setClockValue('months', months);
    setClockValue('days', days);
    setClockValue('hours', hours);
    setClockValue('minutes', minutes);
}

function setClockValue(type, value) {
    const topEl = document.getElementById(`${type}Top`);
    const bottomEl = document.getElementById(`${type}Bottom`);
    
    if (previousValues[type] !== null && previousValues[type] !== value) {
        // Анимация перелистывания
        topEl.classList.add('flip');
        bottomEl.classList.add('flip');
        
        setTimeout(() => {
            topEl.textContent = String(value).padStart(2, '0');
            bottomEl.textContent = String(value).padStart(2, '0');
            topEl.classList.remove('flip');
            bottomEl.classList.remove('flip');
        }, 300);
    } else {
        const paddedValue = String(value).padStart(2, '0');
        topEl.textContent = paddedValue;
        bottomEl.textContent = paddedValue;
    }
    
    previousValues[type] = value;
}

function formatDate(dateString) {
    const date = new Date(dateString);
    const day = String(date.getDate()).padStart(2, '0');
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const year = date.getFullYear();
    return `${day}.${month}.${year}`;
}

// Добавление гостя
function addGuest() {
    if (guests.length >= maxGuests) {
        tg.showAlert(`Можно добавить максимум ${maxGuests} гостей`);
        return;
    }
    
    const guestId = Date.now();
    guests.push({ id: guestId, firstName: '', lastName: '' });
    renderGuests();
}

// Удаление гостя
function removeGuest(guestId) {
    guests = guests.filter(g => g.id !== guestId);
    renderGuests();
}

// Отрисовка списка гостей
function renderGuests() {
    const guestsList = document.getElementById('guestsList');
    guestsList.innerHTML = '';
    
    guests.forEach(guest => {
        const guestItem = document.createElement('div');
        guestItem.className = 'guest-item';
        guestItem.innerHTML = `
            <input type="text" placeholder="Имя" value="${guest.firstName}" 
                   onchange="updateGuest(${guest.id}, 'firstName', this.value)">
            <input type="text" placeholder="Фамилия" value="${guest.lastName}" 
                   onchange="updateGuest(${guest.id}, 'lastName', this.value)">
            <button type="button" class="btn-remove" onclick="removeGuest(${guest.id})">Удалить</button>
        `;
        guestsList.appendChild(guestItem);
    });
    
    const addBtn = document.getElementById('addGuestBtn');
    addBtn.disabled = guests.length >= maxGuests;
}

// Обновление данных гостя
function updateGuest(guestId, field, value) {
    const guest = guests.find(g => g.id === guestId);
    if (guest) {
        guest[field] = value;
    }
}

// Проверка, зарегистрирован ли пользователь
async function checkRegistration() {
    const user = tg.initDataUnsafe?.user;
    const userId = user?.id;
    
    if (!userId) return false;
    
    try {
        const response = await fetch(`${CONFIG.apiUrl}/check?userId=${userId}`);
        if (response.ok) {
            const data = await response.json();
            return data.registered || false;
        }
    } catch (error) {
        console.error('Error checking registration:', error);
    }
    
    return false;
}

// Инициализация
loadConfig();
updateCountdown();
setInterval(updateCountdown, 60000); // Обновляем каждую минуту

// Проверяем регистрацию при загрузке
checkRegistration().then(registered => {
    if (registered) {
        document.getElementById('rsvpSection').style.display = 'none';
        document.getElementById('confirmationSection').style.display = 'block';
        // Загружаем количество гостей
        fetch(`${CONFIG.apiUrl}/stats`)
            .then(r => r.json())
            .then(data => {
                document.getElementById('guestsCount').textContent = data.guestsCount || 0;
            });
    }
});

// Обработчик формы RSVP
document.getElementById('guestForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const firstName = document.getElementById('firstName').value.trim();
    const lastName = document.getElementById('lastName').value.trim();
    
    if (firstName.length < 2 || lastName.length < 2) {
        tg.showAlert('Пожалуйста, введите корректные имя и фамилию');
        return;
    }
    
    // Валидация дополнительных гостей
    const invalidGuests = guests.filter(g => 
        !g.firstName.trim() || g.firstName.trim().length < 2 ||
        !g.lastName.trim() || g.lastName.trim().length < 2
    );
    
    if (invalidGuests.length > 0) {
        tg.showAlert('Пожалуйста, заполните данные всех гостей');
        return;
    }
    
    // Получаем данные пользователя из Telegram
    const user = tg.initDataUnsafe?.user;
    const userId = user?.id;
    const username = user?.username;
    
    // Подготавливаем список всех гостей
    const allGuests = [
        { firstName, lastName },
        ...guests.map(g => ({ firstName: g.firstName.trim(), lastName: g.lastName.trim() }))
    ];
    
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
                guests: allGuests,
                initData: tg.initData
            })
        });
        
        if (response.ok) {
            const data = await response.json();
            // Скрываем форму и показываем подтверждение
            document.getElementById('rsvpSection').style.display = 'none';
            document.getElementById('confirmationSection').style.display = 'block';
            document.getElementById('guestName').textContent = `${firstName} ${lastName}`;
            document.getElementById('guestsCount').textContent = data.guestsCount || 0;
            document.getElementById('confirmationSection').scrollIntoView({ behavior: 'smooth' });
            
            // Вибрация
            if (tg.HapticFeedback) {
                tg.HapticFeedback.notificationOccurred('success');
            }
        } else {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Ошибка при регистрации');
        }
    } catch (error) {
        console.error('Error:', error);
        tg.showAlert('Ошибка при отправке данных. Попробуйте позже.');
    }
});

// Кнопка добавления гостя
document.getElementById('addGuestBtn').addEventListener('click', addGuest);

// Настройка темы Telegram
if (tg.colorScheme === 'dark') {
    document.body.style.background = '#1a1a1a';
    document.body.style.color = '#fff';
}

// Плавная прокрутка при загрузке
window.addEventListener('load', () => {
    window.scrollTo(0, 0);
});
