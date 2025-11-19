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

// Добавление гостя
function addGuest() {
    if (guests.length >= maxGuests) {
        tg.showAlert(`Можно добавить максимум ${maxGuests} гостей`);
        return;
    }
    
    const guestId = Date.now();
    guests.push({ 
        id: guestId, 
        firstName: '', 
        lastName: '',
        category: '',
        side: ''
    });
    renderGuests();
}

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
    minutes: null,
    seconds: null
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
        setClockValue('seconds', 0);
        return;
    }
    
    const months = Math.floor(diff / (1000 * 60 * 60 * 24 * 30));
    const days = Math.floor((diff % (1000 * 60 * 60 * 24 * 30)) / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((diff % (1000 * 60)) / 1000);
    
    setClockValue('months', months);
    setClockValue('days', days);
    setClockValue('hours', hours);
    setClockValue('minutes', minutes);
    setClockValue('seconds', seconds);
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
            <div class="guest-item-header">
                <h4 class="guest-item-title">Дополнительный гость</h4>
                <button type="button" class="btn-remove" onclick="removeGuest(${guest.id})">Удалить</button>
            </div>
            <div class="guest-item-fields">
                <div class="form-group">
                    <label>Имя, Фамилия</label>
                    <input type="text" placeholder="Имя" value="${guest.firstName}" 
                           onchange="updateGuest(${guest.id}, 'firstName', this.value)">
                    <input type="text" placeholder="Фамилия" value="${guest.lastName}" 
                           onchange="updateGuest(${guest.id}, 'lastName', this.value)">
                </div>
                <div class="form-group">
                    <label>Родство</label>
                    <select class="form-select" onchange="updateGuest(${guest.id}, 'category', this.value)">
                        <option value="">Выберите...</option>
                        <option value="Семья" ${guest.category === 'Семья' ? 'selected' : ''}>Семья</option>
                        <option value="Друзья" ${guest.category === 'Друзья' ? 'selected' : ''}>Друзья</option>
                        <option value="Родственники" ${guest.category === 'Родственники' ? 'selected' : ''}>Родственники</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Сторона</label>
                    <select class="form-select" onchange="updateGuest(${guest.id}, 'side', this.value)">
                        <option value="">Выберите...</option>
                        <option value="Жених" ${guest.side === 'Жених' ? 'selected' : ''}>Жених</option>
                        <option value="Невеста" ${guest.side === 'Невеста' ? 'selected' : ''}>Невеста</option>
                        <option value="Общие" ${guest.side === 'Общие' ? 'selected' : ''}>Общие</option>
                    </select>
                </div>
            </div>
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
setInterval(updateCountdown, 1000); // Обновляем каждую секунду

// Проверяем регистрацию при загрузке
checkRegistration().then(registered => {
    if (registered) {
        document.getElementById('rsvpSection').style.display = 'none';
        document.getElementById('confirmationSection').style.display = 'block';
        document.getElementById('cancelSection').style.display = 'block';
        // Загружаем количество гостей
        fetch(`${CONFIG.apiUrl}/stats`)
            .then(r => r.json())
            .then(data => {
                const guestsCountEl = document.getElementById('guestsCount');
                if (guestsCountEl) {
                    guestsCountEl.textContent = data.guestsCount || 0;
                }
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
    
    // Получаем данные основного гостя
    const category = document.getElementById('category').value;
    const side = document.getElementById('side').value;
    
    if (!category || !side) {
        tg.showAlert('Пожалуйста, выберите Родство и Сторону для основного гостя');
        return;
    }
    
    // Валидация дополнительных гостей
    const invalidGuests = guests.filter(g => 
        !g.firstName.trim() || g.firstName.trim().length < 2 ||
        !g.lastName.trim() || g.lastName.trim().length < 2 ||
        !g.category || !g.side
    );
    
    if (invalidGuests.length > 0) {
        tg.showAlert('Пожалуйста, заполните все данные для всех гостей (имя, фамилия, родство, сторона)');
        return;
    }
    
    // Получаем данные пользователя из Telegram
    const user = tg.initDataUnsafe?.user;
    const userId = user?.id;
    const username = user?.username;
    
    if (!userId) {
        tg.showAlert('Ошибка: не удалось получить данные пользователя. Пожалуйста, откройте приложение через Telegram.');
        console.error('User ID not found in Telegram data');
        return;
    }
    
    // Подготавливаем список всех гостей
    const allGuests = [
        { firstName, lastName, category, side },
        ...guests.map(g => ({ 
            firstName: g.firstName.trim(), 
            lastName: g.lastName.trim(),
            category: g.category,
            side: g.side
        }))
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
            document.getElementById('confirmationSection').scrollIntoView({ behavior: 'smooth' });
            
            // Вибрация
            if (tg.HapticFeedback) {
                tg.HapticFeedback.notificationOccurred('success');
            }
        } else {
            // Пытаемся получить детали ошибки
            let errorMessage = 'Ошибка при регистрации';
            try {
                const errorData = await response.json();
                errorMessage = errorData.error || errorMessage;
                console.error('Server error:', errorData);
            } catch (e) {
                console.error('Response status:', response.status, response.statusText);
            }
            throw new Error(errorMessage);
        }
    } catch (error) {
        console.error('Error details:', error);
        console.error('Error stack:', error.stack);
        const errorMessage = error.message || 'Ошибка при отправке данных. Попробуйте позже.';
        tg.showAlert(errorMessage);
    }
});

// Кнопка добавления гостя
document.getElementById('addGuestBtn').addEventListener('click', addGuest);

// Обработчик отмены приглашения
document.getElementById('cancelInvitationBtn').addEventListener('click', async () => {
    if (!confirm('Вы уверены, что не сможете прийти?')) {
        return;
    }
    
    const user = tg.initDataUnsafe?.user;
    const userId = user?.id;
    
    if (!userId) {
        tg.showAlert('Ошибка: не удалось получить данные пользователя.');
        return;
    }
    
    try {
        const response = await fetch(`${CONFIG.apiUrl}/cancel`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                userId: userId,
                initData: tg.initData
            })
        });
        
        if (response.ok) {
            // Показываем форму снова
            document.getElementById('confirmationSection').style.display = 'none';
            document.getElementById('cancelSection').style.display = 'none';
            document.getElementById('rsvpSection').style.display = 'block';
            document.getElementById('rsvpSection').scrollIntoView({ behavior: 'smooth' });
            
            // Очищаем форму
            document.getElementById('firstName').value = '';
            document.getElementById('lastName').value = '';
            document.getElementById('category').value = '';
            document.getElementById('side').value = '';
            guests = [];
            renderGuests();
            
            tg.showAlert('Приглашение отменено. Вы можете заполнить форму заново.');
            
            // Вибрация
            if (tg.HapticFeedback) {
                tg.HapticFeedback.notificationOccurred('warning');
            }
        } else {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Ошибка при отмене приглашения');
        }
    } catch (error) {
        console.error('Error canceling invitation:', error);
        tg.showAlert(error.message || 'Ошибка при отмене приглашения. Попробуйте позже.');
    }
});

// Настройка темы Telegram
if (tg.colorScheme === 'dark') {
    document.body.style.background = '#1a1a1a';
    document.body.style.color = '#fff';
}

// Плавная прокрутка при загрузке
window.addEventListener('load', () => {
    window.scrollTo(0, 0);
    initScrollReveal();
});

// Инициализация анимаций появления при скролле
function initScrollReveal() {
    const revealItems = document.querySelectorAll('.scroll-reveal-item');
    
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('revealed');
                // Опционально: отключаем наблюдение после появления
                observer.unobserve(entry.target);
            }
        });
    }, {
        threshold: 0.1, // Элемент должен быть виден на 10%
        rootMargin: '0px 0px -50px 0px' // Запускаем анимацию немного раньше
    });
    
    revealItems.forEach(item => {
        observer.observe(item);
    });
}
