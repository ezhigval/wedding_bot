// Инициализация Telegram Web App
const tg = window.Telegram.WebApp;
tg.ready();
tg.expand();

// Анимация открытия конверта
function initEnvelopeAnimation() {
    const envelopeContainer = document.getElementById('envelopeAnimation');
    const appContainer = document.querySelector('.app-container');
    const silkBackground = document.querySelector('.silk-background');
    
    if (!envelopeContainer) {
        // Если по какой-то причине конверта нет, просто показываем содержимое
        if (appContainer) appContainer.classList.add('visible');
        if (silkBackground) silkBackground.style.opacity = '1';
        initScrollReveal();
        return;
    }

    // Стартуем анимацию конверта
    setTimeout(() => {
        envelopeContainer.classList.add('start');
    }, 300);

    // После завершения анимации конверта плавно показываем основное содержимое
    const totalDuration = 4500; // даём письму время полностью выехать и немного повисеть
    setTimeout(() => {
        envelopeContainer.classList.add('hidden');
        if (appContainer) appContainer.classList.add('visible');
        if (silkBackground) silkBackground.style.opacity = '1';
        initScrollReveal();
    }, totalDuration);
}

// Запускаем анимацию конверта при загрузке DOM
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initEnvelopeAnimation);
} else {
    initEnvelopeAnimation();
}

    // Загружаем конфигурацию с сервера
    let CONFIG = {
        weddingDate: '2026-06-05',
        groomName: 'Валентин',
        brideName: 'Мария',
        groomTelegram: 'ezhigval',
        brideTelegram: '',
        weddingAddress: 'Санкт-Петербург',
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
        telegram: ''
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
                    <label>Telegram (username)</label>
                    <input type="text" placeholder="@username или username" value="${guest.telegram || ''}" 
                           onchange="updateGuest(${guest.id}, 'telegram', this.value)">
                    <small style="color: #666; font-size: 14px; margin-top: 5px; display: block;">Опционально</small>
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
        } else if (response.status === 404) {
            // Гость не найден (возможно, был удален из таблицы)
            return false;
        }
    } catch (error) {
        console.error('Error checking registration:', error);
        // При ошибке считаем, что гость не найден
        return false;
    }
    
    return false;
}

// Функция для показа сообщения об успешной регистрации
function showSuccessMessage() {
    const successMessage = document.getElementById('successMessage');
    const registrationForm = document.getElementById('registrationForm');
    const errorMessage = document.getElementById('errorMessage');
    
    if (successMessage) {
        successMessage.style.display = 'block';
    }
    if (registrationForm) {
        registrationForm.style.display = 'none';
    }
    if (errorMessage) {
        errorMessage.style.display = 'none';
    }
}

// Функция для скрытия сообщения об успехе (если нужно будет снова показать форму)
function hideSuccessMessage() {
    const successMessage = document.getElementById('successMessage');
    const registrationForm = document.getElementById('registrationForm');
    
    if (successMessage) {
        successMessage.style.display = 'none';
    }
    if (registrationForm) {
        registrationForm.style.display = 'block';
    }
}

// Загрузка данных для основной страницы
async function loadMainPageData() {
    // Загружаем конфигурацию если еще не загружена
    await loadConfig();
    
    // Обновляем данные на странице
    updateMainPageUI();
    
    // Загружаем тайминг
    try {
        const response = await fetch(`${CONFIG.apiUrl}/timeline`);
        if (response.ok) {
            const data = await response.json();
            renderTimeline(data.timeline || []);
        }
    } catch (error) {
        console.error('Error loading timeline:', error);
    }
    
    // Загружаем изображения дресс-кода
    loadDresscodeImages();
    
    // Загружаем изображение места проведения
    loadVenueImage();
    
    // Инициализируем карту
    initYandexMap();
    
    // Обновляем контакты
    updateMainContacts();
}

// Обновление UI основной страницы
function updateMainPageUI() {
    const mainCoupleNames = document.getElementById('mainCoupleNames');
    const mainWeddingDate = document.getElementById('mainWeddingDate');
    const venueName = document.getElementById('venueName');
    const venueAddress = document.getElementById('venueAddress');
    
    if (mainCoupleNames) {
        mainCoupleNames.textContent = `${CONFIG.groomName} и ${CONFIG.brideName}`;
    }
    
    if (mainWeddingDate) {
        const date = new Date(CONFIG.weddingDate);
        const months = ['января', 'февраля', 'марта', 'апреля', 'мая', 'июня', 
                       'июля', 'августа', 'сентября', 'октября', 'ноября', 'декабря'];
        const day = date.getDate();
        const month = months[date.getMonth()];
        const year = date.getFullYear();
        mainWeddingDate.textContent = `${day} ${month} ${year}`;
    }
    
    if (venueName) {
        venueName.textContent = CONFIG.weddingAddress || 'Санкт-Петербург';
    }
    
    if (venueAddress) {
        venueAddress.textContent = CONFIG.weddingAddress || 'Санкт-Петербург';
    }
}

// Отрисовка тайминга
function renderTimeline(timeline) {
    const container = document.getElementById('timelineContainer');
    if (!container) return;
    
    if (!timeline || timeline.length === 0) {
        container.innerHTML = '<p style="text-align: center; color: #666;">План дня будет добавлен позже</p>';
        return;
    }
    
    container.innerHTML = timeline.map(item => `
        <div class="timeline-item scroll-reveal-item">
            <div class="timeline-time">${item.time || ''}</div>
            <div class="timeline-event">${item.event || ''}</div>
        </div>
    `).join('');
    
    // Инициализируем анимации появления
    initScrollReveal();
}

// Загрузка изображений дресс-кода
function loadDresscodeImages() {
    const slider = document.getElementById('dresscodeSlider');
    if (!slider) return;
    
    // Список изображений (будет загружаться из папки res/dresscode)
    const images = [
        'res/dresscode/1.jpg',
        'res/dresscode/2.jpg',
        'res/dresscode/3.jpg'
    ];
    
    // Пока используем заглушку, пока пользователь не добавит фотографии
    slider.innerHTML = `
        <div class="dresscode-slider-container">
            <div class="dresscode-slide active">
                <img src="res/dresscode/1.jpg" alt="Дресс-код" onerror="this.style.display='none'; this.nextElementSibling.style.display='block';">
                <div style="display: none; padding: 40px; text-align: center; color: #666;">
                    <p>Фотографии дресс-кода будут добавлены позже</p>
                </div>
            </div>
        </div>
    `;
    
    // Если есть несколько изображений, создаем слайдер
    if (images.length > 1) {
        initDresscodeSlider(images);
    }
}

// Инициализация слайдера дресс-кода
function initDresscodeSlider(images) {
    const slider = document.getElementById('dresscodeSlider');
    if (!slider) return;
    
    let currentIndex = 0;
    
    slider.innerHTML = images.map((img, index) => `
        <div class="dresscode-slide ${index === 0 ? 'active' : ''}">
            <img src="${img}" alt="Дресс-код ${index + 1}" onerror="this.style.display='none';">
        </div>
    `).join('');
    
    // Автоматическая смена слайдов каждые 1500мс (1.5 секунды)
    setInterval(() => {
        const slides = slider.querySelectorAll('.dresscode-slide');
        if (slides.length > 1) {
            slides[currentIndex].classList.remove('active');
            currentIndex = (currentIndex + 1) % slides.length;
            slides[currentIndex].classList.add('active');
        }
    }, 1500);
    
    // Свайп для мобильных
    let touchStartX = 0;
    let touchEndX = 0;
    
    slider.addEventListener('touchstart', (e) => {
        touchStartX = e.changedTouches[0].screenX;
    });
    
    slider.addEventListener('touchend', (e) => {
        touchEndX = e.changedTouches[0].screenX;
        handleSwipe();
    });
    
    function handleSwipe() {
        const slides = slider.querySelectorAll('.dresscode-slide');
        if (slides.length <= 1) return;
        
        if (touchEndX < touchStartX - 50) {
            // Свайп влево - следующий слайд
            slides[currentIndex].classList.remove('active');
            currentIndex = (currentIndex + 1) % slides.length;
            slides[currentIndex].classList.add('active');
        }
        if (touchEndX > touchStartX + 50) {
            // Свайп вправо - предыдущий слайд
            slides[currentIndex].classList.remove('active');
            currentIndex = (currentIndex - 1 + slides.length) % slides.length;
            slides[currentIndex].classList.add('active');
        }
    }
}

// Загрузка изображения места проведения
function loadVenueImage() {
    const container = document.getElementById('venueImageContainer');
    if (!container) return;
    
    container.innerHTML = `
        <img src="res/venue/venue.jpg" alt="Место проведения" class="venue-image" 
             onerror="this.style.display='none';">
    `;
}

// Инициализация Яндекс карты
function initYandexMap() {
    const mapContainer = document.getElementById('venueMap');
    if (!mapContainer) return;
    
    const address = CONFIG.weddingAddress || 'Санкт-Петербург';
    
    // Используем iframe для встраивания Яндекс карты
    mapContainer.innerHTML = `
        <iframe 
            src="https://yandex.ru/map-widget/v1/?text=${encodeURIComponent(address)}&z=15"
            width="100%" 
            height="300" 
            frameborder="0" 
            style="border-radius: 10px; margin-top: 20px; border: 2px solid rgba(90, 124, 82, 0.3);">
        </iframe>
    `;
}

// Обновление контактов на основной странице
function updateMainContacts() {
    const groomTelegram = document.getElementById('mainGroomTelegram');
    const brideTelegram = document.getElementById('mainBrideTelegram');
    
    if (groomTelegram) {
        groomTelegram.textContent = `@${CONFIG.groomTelegram || 'ezhigval'}`;
    }
    if (brideTelegram) {
        brideTelegram.textContent = `@${CONFIG.brideTelegram || 'mrfilmpro'}`;
    }
    
    // Обработчики для открытия Telegram
    const groomContact = document.getElementById('mainGroomContact');
    const brideContact = document.getElementById('mainBrideContact');
    
    if (groomContact) {
        groomContact.href = `https://t.me/${CONFIG.groomTelegram || 'ezhigval'}`;
    }
    if (brideContact) {
        brideContact.href = `https://t.me/${CONFIG.brideTelegram || 'mrfilmpro'}`;
    }
}

// Инициализация
loadConfig();

// Функция для проверки регистрации и отображения правильной страницы
async function checkAndShowPage() {
    try {
        const user = tg.initDataUnsafe?.user;
        const userId = user?.id;
        
        // ВРЕМЕННО: Всегда показываем страницу регистрации
        // Основная страница закрыта для всех
        document.querySelector('.hero-section').style.display = 'block';
        document.querySelector('.greeting-section').style.display = 'block';
        document.querySelector('.calendar-section').style.display = 'block';
        document.getElementById('rsvpSection').style.display = 'block';
        document.getElementById('registrationContactSection').style.display = 'block';
        document.querySelector('.closing-section').style.display = 'block';
        
        // Скрываем основную страницу
        document.getElementById('mainPage').style.display = 'none';
        
        if (userId) {
            // Проверяем, был ли пользователь ранее зарегистрирован (из localStorage)
            const wasRegistered = localStorage.getItem(`registered_${userId}`) === 'true';
            
            // Проверяем текущий статус регистрации
            const registered = await checkRegistration();
            
            // Если пользователь был зарегистрирован, но теперь не найден - показываем ошибку
            if (wasRegistered && !registered) {
                showErrorMessage();
            } else {
                hideErrorMessage();
            }
        } else {
            hideErrorMessage();
        }
    } catch (error) {
        console.error('Error checking registration:', error);
        // При ошибке показываем страницу регистрации
        showRegistrationPage();
    }
}

// Функция для показа сообщения об ошибке
function showErrorMessage() {
    const errorMessage = document.getElementById('errorMessage');
    const registrationForm = document.getElementById('registrationForm');
    const successMessage = document.getElementById('successMessage');
    
    if (errorMessage) {
        errorMessage.style.display = 'block';
    }
    if (registrationForm) {
        registrationForm.style.display = 'block';
    }
    if (successMessage) {
        successMessage.style.display = 'none';
    }
}

// Функция для скрытия сообщения об ошибке
function hideErrorMessage() {
    const errorMessage = document.getElementById('errorMessage');
    if (errorMessage) {
        errorMessage.style.display = 'none';
    }
}

// Функция для показа страницы регистрации
function showRegistrationPage() {
    document.querySelector('.hero-section').style.display = 'block';
    document.querySelector('.greeting-section').style.display = 'block';
    document.querySelector('.calendar-section').style.display = 'block';
    document.getElementById('rsvpSection').style.display = 'block';
    document.getElementById('registrationContactSection').style.display = 'block';
    document.querySelector('.closing-section').style.display = 'block';
    document.getElementById('mainPage').style.display = 'none';
}

// Проверка регистрации при загрузке страницы
checkAndShowPage();

// Также проверяем при каждом открытии Mini App (когда пользователь возвращается)
// Telegram Web App может быть открыт в фоне, поэтому проверяем при видимости страницы
document.addEventListener('visibilitychange', () => {
    if (!document.hidden) {
        // Страница стала видимой - проверяем регистрацию снова
        // Если гость был удален из таблицы, покажем ошибку
        checkAndShowPage();
    }
});

// Проверяем при фокусе окна (если Mini App открыт в браузере)
window.addEventListener('focus', () => {
    checkAndShowPage();
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
        !g.lastName.trim() || g.lastName.trim().length < 2
    );
    
    if (invalidGuests.length > 0) {
        tg.showAlert('Пожалуйста, заполните имя и фамилию для всех дополнительных гостей');
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
    // Для дополнительных гостей используем category и side основного гостя
    const allGuests = [
        { firstName, lastName, category, side },
        ...guests.map(g => ({ 
            firstName: g.firstName.trim(), 
            lastName: g.lastName.trim(),
            category: category,  // Используем category основного гостя
            side: side,  // Используем side основного гостя
            telegram: g.telegram ? g.telegram.trim().replace('@', '') : ''  // Telegram username (опционально)
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
                
                // Сохраняем информацию о регистрации в localStorage
                const user = tg.initDataUnsafe?.user;
                const userId = user?.id;
                if (userId) {
                    localStorage.setItem(`registered_${userId}`, 'true');
                }
                
                // Показываем сообщение об успешной регистрации
                showSuccessMessage();
                
                // Скрываем сообщение об ошибке, если оно было показано
                hideErrorMessage();
                
                // Прокручиваем к сообщению
                document.getElementById('rsvpSection').scrollIntoView({ behavior: 'smooth' });
                
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

// Обработчик отмены приглашения (на регистрационной странице)
const cancelInvitationBtn = document.getElementById('cancelInvitationBtn');
if (cancelInvitationBtn) {
    cancelInvitationBtn.addEventListener('click', async () => {
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
                // Удаляем информацию о регистрации из localStorage
                const user = tg.initDataUnsafe?.user;
                const userId = user?.id;
                if (userId) {
                    localStorage.removeItem(`registered_${userId}`);
                }
                
                // После отмены приглашения показываем страницу регистрации
                await checkAndShowPage();
                
                // Скрываем сообщения об успехе/ошибке и показываем форму
                hideSuccessMessage();
                hideErrorMessage();
                
                // Очищаем форму
                document.getElementById('firstName').value = '';
                document.getElementById('lastName').value = '';
                document.getElementById('category').value = '';
                document.getElementById('side').value = '';
                guests = [];
                renderGuests();
                
                // Прокручиваем к форме регистрации
                document.getElementById('rsvpSection').scrollIntoView({ behavior: 'smooth' });
                
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
}

// Обработчик отмены приглашения (на основной странице)
const mainCancelInvitationBtn = document.getElementById('mainCancelInvitationBtn');
if (mainCancelInvitationBtn) {
    mainCancelInvitationBtn.addEventListener('click', async () => {
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
                // Удаляем информацию о регистрации из localStorage
                const user = tg.initDataUnsafe?.user;
                const userId = user?.id;
                if (userId) {
                    localStorage.removeItem(`registered_${userId}`);
                }
                
                // После отмены приглашения показываем страницу регистрации
                await checkAndShowPage();
                
                // Скрываем сообщения об успехе/ошибке и показываем форму
                hideSuccessMessage();
                hideErrorMessage();
                
                // Очищаем форму
                document.getElementById('firstName').value = '';
                document.getElementById('lastName').value = '';
                document.getElementById('category').value = '';
                document.getElementById('side').value = '';
                guests = [];
                renderGuests();
                
                // Прокручиваем к форме регистрации
                document.getElementById('rsvpSection').scrollIntoView({ behavior: 'smooth' });
                
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
}

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
