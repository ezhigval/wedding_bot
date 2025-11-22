// Инициализация Telegram Web App
const tg = window.Telegram.WebApp;
tg.ready();
tg.expand();

// Анимация загрузки с обручальным кольцом (Lottie)
function initRingLoader() {
    const ringLoader = document.getElementById('ringLoader');
    const lottieContainer = document.getElementById('lottieContainer');
    const appContainer = document.querySelector('.app-container');
    const silkBackground = document.querySelector('.silk-background');
    
    if (!ringLoader || !lottieContainer) {
        // Если по какой-то причине загрузчика нет, просто показываем содержимое
        if (appContainer) appContainer.classList.add('visible');
        if (silkBackground) silkBackground.style.opacity = '1';
        initScrollReveal();
        return;
    }

    // Изначально скрываем сайт
    if (appContainer) {
        appContainer.style.opacity = '0';
    }
    if (silkBackground) {
        silkBackground.style.opacity = '0';
    }

    // Проверяем наличие Lottie
    if (typeof lottie === 'undefined') {
        console.error('Lottie library not loaded');
        // Fallback: показываем сайт через 3 секунды
    setTimeout(() => {
            ringLoader.classList.add('hidden');
            if (appContainer) {
                appContainer.classList.add('visible');
                setTimeout(() => {
                    appContainer.style.transition = 'opacity 0.8s ease-in';
                    appContainer.style.opacity = '1';
                }, 50);
            }
            if (silkBackground) {
                setTimeout(() => {
                    silkBackground.style.transition = 'opacity 0.8s ease-in';
                    silkBackground.style.opacity = '1';
                }, 50);
            }
            initScrollReveal();
        }, 3000);
        return;
    }

    // Загружаем Lottie анимацию из ring_animation.json
    const animationPath = 'ring_animation.json';
    
    const anim = lottie.loadAnimation({
        container: lottieContainer,
        renderer: 'svg', // или 'canvas' для лучшей производительности
        loop: false,
        autoplay: true,
        path: animationPath
    });

    // Обработка событий анимации
    anim.addEventListener('data_ready', () => {
        console.log('Lottie animation loaded');
    });

    anim.addEventListener('complete', () => {
        // Анимация завершена, скрываем загрузчик и показываем сайт
        ringLoader.classList.add('hidden');
        if (appContainer) {
            appContainer.classList.add('visible');
    setTimeout(() => {
                appContainer.style.transition = 'opacity 0.8s ease-in';
                appContainer.style.opacity = '1';
            }, 50);
        }
        if (silkBackground) {
            setTimeout(() => {
                silkBackground.style.transition = 'opacity 0.8s ease-in';
                silkBackground.style.opacity = '1';
            }, 50);
        }
        initScrollReveal();
    });

    // Fallback: если анимация не загрузилась за 5 секунд, показываем сайт
    setTimeout(() => {
        if (!ringLoader.classList.contains('hidden')) {
            console.warn('Lottie animation timeout, showing site');
            ringLoader.classList.add('hidden');
            if (appContainer) {
                appContainer.classList.add('visible');
                setTimeout(() => {
                    appContainer.style.transition = 'opacity 0.8s ease-in';
                    appContainer.style.opacity = '1';
                }, 50);
            }
            if (silkBackground) {
                setTimeout(() => {
                    silkBackground.style.transition = 'opacity 0.8s ease-in';
                    silkBackground.style.opacity = '1';
                }, 50);
            }
            initScrollReveal();
        }
    }, 5000);
}

// Запускаем анимацию конверта при загрузке DOM
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initRingLoader);
} else {
    initRingLoader();
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
    const dateText = `${monthNames[weddingDate.getMonth()]} ${day} ${year}`;
    
    const calendarDateFull = document.getElementById('calendarDateFull');
    if (calendarDateFull) {
        calendarDateFull.textContent = dateText;
    }
    
    const mainCalendarDateFull = document.getElementById('mainCalendarDateFull');
    if (mainCalendarDateFull) {
        mainCalendarDateFull.textContent = dateText;
    }
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
                    <small style="color: var(--color-text-secondary); font-size: 14px; margin-top: 5px; display: block;">Опционально</small>
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
    const firstName = user?.first_name || '';
    const lastName = user?.last_name || '';
    
    if (!userId) {
        console.log('checkRegistration: userId not found');
        return { registered: false, error: 'no_user_id' };
    }
    
    try {
        // Передаем userId, firstName и lastName для поиска
        const params = new URLSearchParams({
            userId: userId,
            firstName: firstName,
            lastName: lastName
        });
        const url = `${CONFIG.apiUrl}/check?${params}`;
        console.log('checkRegistration: fetching', url);
        const response = await fetch(url);
        
        if (response.ok) {
            const data = await response.json();
            console.log('checkRegistration: response data', data);
            return data;
        } else {
            console.error('checkRegistration: response error', response.status, response.statusText);
            // При ошибке сервера показываем страницу ошибки
            return { registered: false, error: 'server_error' };
        }
    } catch (error) {
        console.error('Error checking registration:', error);
        // При ошибке сети показываем страницу ошибки
        return { registered: false, error: 'network_error' };
    }
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
    
    // Точный адрес места проведения
    const venueFullAddress = 'Разъезжая улица, 15, городской посёлок Токсово, Токсовское городское поселение, Всеволожский район, Ленинградская область';
    
    if (venueName) {
        venueName.textContent = 'Токсово';
    }
    
    if (venueAddress) {
        venueAddress.textContent = venueFullAddress;
    }
}

// Отрисовка тайминга
function renderTimeline(timeline) {
    const container = document.getElementById('timelineContainer');
    if (!container) return;
    
    if (!timeline || timeline.length === 0) {
        container.innerHTML = '<p style="text-align: center; color: var(--color-text-secondary);">План дня будет добавлен позже</p>';
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
    
    // Координаты места проведения: 60.136143, 30.525849
    const lat = 60.136143;
    const lon = 30.525849;
    const zoom = 15;
    
    // Используем iframe для встраивания Яндекс карты с координатами
    mapContainer.innerHTML = `
        <iframe 
            src="https://yandex.ru/map-widget/v1/?ll=${lon},${lat}&z=${zoom}&pt=${lon},${lat}"
            width="100%" 
            height="100%" 
            frameborder="0" 
            style="border: none; display: block;">
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
        
        if (!userId) {
            // Если нет userId, показываем страницу регистрации
            showRegistrationPage();
            return;
        }
        
        // Проверяем текущий статус регистрации
        const result = await checkRegistration();
        
        if (result.registered) {
            // Пользователь зарегистрирован - показываем основную страницу
            showMainPage();
        } else if (result.needs_confirmation) {
            // Найден по имени, нужно подтвердить личность
            showConfirmationPage(result.guest_name, result.row);
        } else if (result.error) {
            // Ошибка - показываем страницу ошибки
            showErrorPage();
        } else {
            // Пользователь не найден - показываем страницу регистрации
            showRegistrationPage();
        }
    } catch (error) {
        console.error('Error checking registration:', error);
        // При ошибке показываем страницу ошибки
        showErrorPage();
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
    // Показываем секции регистрационной страницы
    const registrationSections = [
        document.querySelector('.hero-section'),
        document.querySelector('.greeting-section'),
        document.querySelector('.calendar-section'),
        document.getElementById('rsvpSection'),
        document.getElementById('registrationContactSection'),
        document.querySelector('.closing-section')
    ];
    
    registrationSections.forEach(section => {
        if (section) section.style.display = 'block';
    });
    
    // Скрываем основную страницу
    const mainPage = document.getElementById('mainPage');
    if (mainPage) mainPage.style.display = 'none';
}

// Функция для показа страницы подтверждения личности
function showConfirmationPage(guestName, row) {
    // Скрываем все секции
    document.querySelectorAll('section').forEach(section => {
        section.style.display = 'none';
    });
    
    // Показываем страницу подтверждения
    const confirmationPage = document.getElementById('confirmationPage');
    if (confirmationPage) {
        confirmationPage.style.display = 'block';
        const guestNameElement = document.getElementById('confirmationGuestName');
        if (guestNameElement) {
            guestNameElement.textContent = guestName;
        }
        
        // Сохраняем данные для подтверждения
        const confirmYesBtn = document.getElementById('confirmYesBtn');
        const confirmNoBtn = document.getElementById('confirmNoBtn');
        
        if (confirmYesBtn) {
            confirmYesBtn.onclick = async () => {
                await confirmIdentity(row);
            };
        }
        
        if (confirmNoBtn) {
            confirmNoBtn.onclick = () => {
                showRegistrationPage();
            };
        }
    } else {
        // Если страницы нет, показываем регистрацию
        showRegistrationPage();
    }
}

// Функция для подтверждения личности
async function confirmIdentity(row) {
    try {
        const user = tg.initDataUnsafe?.user;
        const userId = user?.id;
        
        if (!userId) {
            showErrorPage();
            return;
        }
        
        const response = await fetch(`${CONFIG.apiUrl}/confirm-identity`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                row: row,
                userId: userId
            })
        });
        
        if (response.ok) {
            const data = await response.json();
            if (data.success) {
                // Успешно подтверждено - показываем основную страницу
                showMainPage();
            } else {
                showErrorPage();
            }
        } else {
            showErrorPage();
        }
    } catch (error) {
        console.error('Error confirming identity:', error);
        showErrorPage();
    }
}

// Функция для показа страницы ошибки
function showErrorPage() {
    // Скрываем все секции
    document.querySelectorAll('section').forEach(section => {
        section.style.display = 'none';
    });
    
    // Показываем страницу ошибки
    const errorPage = document.getElementById('errorPage');
    if (errorPage) {
        errorPage.style.display = 'block';
    } else {
        // Если страницы нет, показываем регистрацию с сообщением
        showRegistrationPage();
        alert('Ой, что-то пошло не так. Пожалуйста, напишите нам.');
    }
}

// Функция для показа основной страницы
function showMainPage() {
    // Скрываем секции регистрационной страницы
    const registrationSections = [
        document.querySelector('.hero-section'),
        document.querySelector('.greeting-section'),
        document.querySelector('.calendar-section'),
        document.getElementById('rsvpSection'),
        document.getElementById('registrationContactSection'),
        document.querySelector('.closing-section')
    ];
    
    registrationSections.forEach(section => {
        if (section) section.style.display = 'none';
    });
    
    // Показываем основную страницу
    const mainPage = document.getElementById('mainPage');
    if (mainPage) {
        mainPage.style.display = 'block';
        // Загружаем данные для основной страницы
        loadMainPageData();
    }
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
                
                // Скрываем сообщение об ошибке, если оно было показано
                hideErrorMessage();
                
                // Переключаемся на основную страницу
                showMainPage();
                
                // Прокручиваем к началу страницы
                window.scrollTo({ top: 0, behavior: 'smooth' });
                
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
    document.body.style.background = 'var(--color-text-dark)';
    document.body.style.color = 'var(--color-bg-white)';
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
