// Инициализация Telegram Web App
const tg = window.Telegram.WebApp;
tg.ready();
tg.expand();

// Анимация загрузки с обручальным кольцом (Lottie rings.json)
function initRingLoader() {
    const ringLoader = document.getElementById('ringLoader');
    const lottieContainer = document.getElementById('lottieContainer');
    const appContainer = document.querySelector('.app-container');
    const silkBackground = document.querySelector('.silk-background');
    const htmlEl = document.documentElement;
    const bodyEl = document.body;
    
    // Функция показа основного содержимого
    function showApp() {
        if (!ringLoader) {
            if (appContainer) {
                appContainer.classList.add('visible');
                appContainer.style.opacity = '1';
            }
            if (silkBackground) {
                silkBackground.style.opacity = '1';
            }
            // Разрешаем вертикальный скролл после появления сайта
            if (htmlEl) {
                htmlEl.style.overflowY = 'auto';
                htmlEl.style.overflowX = 'hidden';
            }
            if (bodyEl) {
                bodyEl.style.overflowY = 'auto';
                bodyEl.style.overflowX = 'hidden';
            }
            initScrollReveal();
            return;
        }

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
        // Разрешаем вертикальный скролл после появления сайта
        if (htmlEl) {
            htmlEl.style.overflowY = 'auto';
            htmlEl.style.overflowX = 'hidden';
        }
        if (bodyEl) {
            bodyEl.style.overflowY = 'auto';
            bodyEl.style.overflowX = 'hidden';
        }
        initScrollReveal();
    }

    // Если по какой-то причине контейнера загрузчика нет — просто показываем сайт
    if (!ringLoader) {
        showApp();
        return;
    }

    // Изначально скрываем сайт
    if (appContainer) {
        appContainer.style.opacity = '0';
    }
    if (silkBackground) {
        silkBackground.style.opacity = '0';
    }
    // Во время анимации блокируем скролл полностью
    if (htmlEl) {
        htmlEl.style.overflow = 'hidden';
    }
    if (bodyEl) {
        bodyEl.style.overflow = 'hidden';
    }

    // Основная логика: сначала пытаемся использовать Lottie, потом простой таймер
    function initLottieOrFallback() {
        // Проверяем наличие Lottie
        if (typeof lottie === 'undefined') {
            console.error('Lottie library not loaded');
            // Fallback: показываем сайт через 5 секунд
            setTimeout(showApp, 5000);
            return;
        }

        // Если нет контейнера для Lottie — просто показываем сайт
        if (!lottieContainer) {
            setTimeout(showApp, 300);
            return;
        }

        // Загружаем Lottie анимацию из локального файла ring_animation.json (лежит в webapp/)
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

        // Если данные не загрузились — сразу показываем сайт
        anim.addEventListener('data_failed', () => {
            console.error('Lottie animation data_failed, showing site');
            showApp();
        });

    anim.addEventListener('complete', () => {
        // Анимация завершена, скрываем загрузчик и показываем сайт
            showApp();
        });

        // Жёсткий таймаут: через ~7 секунд обязательно показываем сайт,
        // даже если анимация зависла или не успела загрузиться
    setTimeout(() => {
        if (!ringLoader.classList.contains('hidden')) {
            console.warn('Lottie animation timeout, showing site');
                showApp();
            }
        }, 7000);
    }

    // Стартуем Lottie/фоллбек
    initLottieOrFallback();
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

// Флаг: пользователь уже зарегистрирован (по данным localStorage)
let isUserRegistered = false;

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
                    <label>Фамилия, Имя</label>
                    <input type="text" placeholder="Фамилия" value="${guest.lastName}" 
                           onchange="updateGuest(${guest.id}, 'lastName', this.value)">
                    <input type="text" placeholder="Имя" value="${guest.firstName}" 
                           onchange="updateGuest(${guest.id}, 'firstName', this.value)">
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

// Функция для получения user_id из Telegram Mini App (несколько способов)
function getTelegramUserId() {
    let userId = null;
    let firstName = '';
    let lastName = '';
    
    // Способ 1: Через initDataUnsafe (самый простой)
    const user = tg.initDataUnsafe?.user;
    if (user?.id) {
        userId = user.id;
        firstName = user.first_name || '';
        lastName = user.last_name || '';
        console.log('getTelegramUserId: found via initDataUnsafe', { userId, firstName, lastName });
        return { userId, firstName, lastName, method: 'initDataUnsafe' };
    }
    
    // Способ 2: Через initData (нужно отправить на сервер для парсинга)
    const initData = tg.initData;
    if (initData) {
        console.log('getTelegramUserId: initData available, will parse on server');
        // Вернем initData для парсинга на сервере
        return { userId: null, firstName: '', lastName: '', initData, method: 'initData' };
    }
    
    // Способ 3: Из localStorage (если был сохранен ранее)
    const savedUserId = localStorage.getItem('telegram_user_id');
    if (savedUserId) {
        console.log('getTelegramUserId: found in localStorage', savedUserId);
        return { userId: parseInt(savedUserId), firstName: '', lastName: '', method: 'localStorage' };
    }
    
    console.warn('getTelegramUserId: no user_id found by any method');
    return { userId: null, firstName: '', lastName: '', method: 'none' };
}

// Проверка, зарегистрирован ли пользователь через API
// Используем несколько способов получения user_id из Telegram Mini App
async function checkRegistration() {
    // Получаем данные пользователя из Telegram Web App (несколько способов)
    const userData = getTelegramUserId();
    const { userId, firstName, lastName, initData, method } = userData;
    
    console.log('checkRegistration: user data method:', method, { userId, firstName, lastName, hasInitData: !!initData });
    
    // Если есть initData, отправляем его на сервер для парсинга
    if (initData && !userId) {
        try {
            const response = await fetch(`${CONFIG.apiUrl}/parse-init-data`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ initData })
            });
            
            if (response.ok) {
                const parsed = await response.json();
                if (parsed.userId) {
                    // Сохраняем в localStorage для будущего использования
                    localStorage.setItem('telegram_user_id', parsed.userId.toString());
                    return await checkRegistrationWithUserId(parsed.userId, parsed.firstName || '', parsed.lastName || '');
                }
            }
        } catch (error) {
            console.error('Error parsing initData:', error);
        }
    }
    
    // Если нет userId, но есть имя/фамилия - пробуем поиск только по имени
    if (!userId && firstName && lastName) {
        console.log('checkRegistration: no userId, trying search by name only');
        try {
            const params = new URLSearchParams({
                firstName: firstName,
                lastName: lastName,
                searchByNameOnly: 'true'
            });
            const url = `${CONFIG.apiUrl}/check?${params}`;
            const response = await fetch(url);
            
            if (response.ok) {
                const data = await response.json();
                console.log('checkRegistration: response by name only', data);
                return data;
            }
        } catch (error) {
            console.error('Error checking by name only:', error);
        }
    }
    
    // Если нет userId вообще - возвращаем ошибку
    if (!userId) {
        console.log('checkRegistration: userId not found by any method');
        return { registered: false, error: 'no_user_id' };
    }
    
    // Если userId есть - проверяем регистрацию
    return await checkRegistrationWithUserId(userId, firstName, lastName);
}

// Вспомогательная функция для проверки регистрации с известным userId
async function checkRegistrationWithUserId(userId, firstName, lastName) {
    try {
        // Передаем userId из Telegram для сравнения с таблицей (столбец F)
        // Также передаем firstName и lastName для поиска по имени, если user_id не найден
        const params = new URLSearchParams({
            userId: userId,  // Сравниваем с user_id в столбце F таблицы
            firstName: firstName,
            lastName: lastName
        });
        const url = `${CONFIG.apiUrl}/check?${params}`;
        console.log('checkRegistrationWithUserId: fetching', url);
        const response = await fetch(url);
        
        if (response.ok) {
            const data = await response.json();
            console.log('checkRegistrationWithUserId: response data', data);
            
            // Если регистрация успешна, сохраняем userId в localStorage
            if (data.registered && userId) {
                localStorage.setItem('telegram_user_id', userId.toString());
            }
            
            return data;
        } else {
            console.error('checkRegistrationWithUserId: response error', response.status, response.statusText);
            // При ошибке сервера показываем страницу ошибки
            return { registered: false, error: 'server_error' };
        }
    } catch (error) {
        console.error('Error checking registration:', error);
        // При ошибке сети показываем страницу ошибки
        return { registered: false, error: 'network_error' };
    }
}

// Управление видимостью блока "СВАДЕБНЫЙ ЧАТ"
function updateGroupSectionVisibility(shouldShow) {
    const groupSection = document.getElementById('groupSection');
    if (!groupSection) return;
    groupSection.style.display = shouldShow ? 'block' : 'none';
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

// Инициализация состояния блока RSVP:
// при каждом открытии обращаемся к серверу и сверяем user_id с Google Sheets
async function initRsvpForCurrentUser() {
    const rsvpSection = document.getElementById('rsvpSection');
    if (!rsvpSection) return;

    try {
        const status = await checkRegistration();
        // Серверная истина: зарегистрирован ли пользователь в таблице гостей
        isUserRegistered = !!(status && status.registered);
        const inGroupChat = !!(status && status.in_group_chat);

        // Блок "СВАДЕБНЫЙ ЧАТ" показываем только зарегистрированным гостям,
        // которые ЕЩЁ НЕ состоят в общем чате Telegram
        updateGroupSectionVisibility(isUserRegistered && !inGroupChat);

        if (isUserRegistered) {
            setupAddGuestOnlyView();
        } else {
            setupFullRsvpView();
        }
    } catch (e) {
        console.error('initRsvpForCurrentUser: error while checking registration status:', e);
        // На всякий случай показываем полную анкету, если проверка не удалась
        isUserRegistered = false;
        setupFullRsvpView();

        // При ошибке проверки скрываем блок "СВАДЕБНЫЙ ЧАТ"
        updateGroupSectionVisibility(false);
    }
}

// Полная анкета "Присутствие" (для тех, кто еще не регистрировался)
function setupFullRsvpView() {
    const rsvpSection = document.getElementById('rsvpSection');
    if (!rsvpSection) return;

    rsvpSection.style.display = 'block';

    const titleEl = rsvpSection.querySelector('.section-title');
    if (titleEl) {
        titleEl.textContent = 'ПРИСУТСТВИЕ';
    }

    const textEl = rsvpSection.querySelector('.rsvp-text');
    if (textEl) {
        textEl.textContent = 'Пожалуйста, подтвердите ваше присутствие на нашем празднике. Заполните форму ниже:';
    }

    const mainGuestGroup = document.getElementById('mainGuestGroup');
    if (mainGuestGroup) {
        mainGuestGroup.style.display = 'block';
    }

    const successMessage = document.getElementById('successMessage');
    const registrationForm = document.getElementById('registrationForm');
    if (successMessage) {
        successMessage.style.display = 'none';
    }
    if (registrationForm) {
        registrationForm.style.display = 'block';
    }

    // Сбрасываем поля основной анкеты
    const firstNameInput = document.getElementById('firstName');
    const lastNameInput = document.getElementById('lastName');
    const categorySelect = document.getElementById('category');
    const sideSelect = document.getElementById('side');
    if (firstNameInput) firstNameInput.value = '';
    if (lastNameInput) lastNameInput.value = '';
    if (categorySelect) categorySelect.value = '';
    if (sideSelect) sideSelect.value = '';

    // Сбрасываем дополнительных гостей
    guests = [];
    renderGuests();
}

// Режим "Добавить дополнительного гостя" для уже зарегистрированных
function setupAddGuestOnlyView() {
    const rsvpSection = document.getElementById('rsvpSection');
    if (!rsvpSection) return;

    rsvpSection.style.display = 'block';

    const titleEl = rsvpSection.querySelector('.section-title');
    if (titleEl) {
        titleEl.textContent = 'ДОБАВИТЬ ДОПОЛНИТЕЛЬНОГО ГОСТЯ';
    }

    const textEl = rsvpSection.querySelector('.rsvp-text');
    if (textEl) {
        textEl.textContent = 'Вы уже подтвердили своё присутствие. Если вы приходите не один, добавьте дополнительных гостей ниже.';
    }

    const mainGuestGroup = document.getElementById('mainGuestGroup');
    const firstNameInput = document.getElementById('firstName');
    const lastNameInput = document.getElementById('lastName');
    const categorySelect = document.getElementById('category');
    const sideSelect = document.getElementById('side');

    // Восстанавливаем данные основного гостя из localStorage (если есть)
    const storedFirst = localStorage.getItem('main_guest_first_name');
    const storedLast = localStorage.getItem('main_guest_last_name');
    const storedCat = localStorage.getItem('main_guest_category');
    const storedSide = localStorage.getItem('main_guest_side');

    if (storedFirst && firstNameInput) firstNameInput.value = storedFirst;
    if (storedLast && lastNameInput) lastNameInput.value = storedLast;
    if (storedCat && categorySelect) categorySelect.value = storedCat;
    if (storedSide && sideSelect) sideSelect.value = storedSide;

    // Если у нас есть все данные по основному гостю — скрываем блок его полей,
    // но сами инпуты остаются в DOM и будут отправлены на сервер
    if (storedFirst && storedLast && storedCat && storedSide && mainGuestGroup) {
        mainGuestGroup.style.display = 'none';
    }

    const successMessage = document.getElementById('successMessage');
    const registrationForm = document.getElementById('registrationForm');
    if (successMessage) {
        successMessage.style.display = 'none';
    }
    if (registrationForm) {
        registrationForm.style.display = 'block';
    }

    // Очищаем список дополнительных гостей в интерфейсе
    guests = [];
    renderGuests();
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

    // Пытаемся загрузить информацию о столе и соседях (если доступна)
    loadSeatingInfoForCurrentUser().catch(err => {
        console.error('Error loading seating info:', err);
    });
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
    if (!mapContainer) {
        console.warn('venueMap container not found');
        return;
    }
    
    // Проверяем, что элемент видим (не скрыт через display: none)
    const tabContent = mapContainer.closest('.tab-content');
    if (tabContent && tabContent.style.display === 'none') {
        console.warn('Map container is in hidden tab, will retry later');
        return;
    }
    
    // Если карта уже загружена, не перезагружаем
    if (mapContainer.querySelector('iframe')) {
        return;
    }
    
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

// Загрузка информации о столе и соседях для текущего пользователя
async function loadSeatingInfoForCurrentUser() {
    try {
        const userData = getTelegramUserId();
        const userId = userData.userId;

        if (!userId) {
            console.log('loadSeatingInfoForCurrentUser: no userId, skipping');
            return;
        }

        const url = `${CONFIG.apiUrl}/seating-info?userId=${encodeURIComponent(userId)}`;
        console.log('loadSeatingInfoForCurrentUser: fetching', url);

        const response = await fetch(url);
        if (!response.ok) {
            console.log('loadSeatingInfoForCurrentUser: response not ok', response.status);
            return;
        }

        const data = await response.json();
        console.log('loadSeatingInfoForCurrentUser: data', data);

        if (!data.visible) {
            return;
        }

        const tableName = data.table || '';
        const neighbors = Array.isArray(data.neighbors) ? data.neighbors : [];

        const greetingSection = document.querySelector('.greeting-section');
        if (!greetingSection) return;

        const titleEl = greetingSection.querySelector('.section-title');
        const textEl = greetingSection.querySelector('.greeting-text');

        if (titleEl) {
            const tableText = tableName ? `ВАШ СТОЛ ${tableName}` : 'ВАШ СТОЛ';
            titleEl.textContent = tableText;
        }

        if (textEl) {
            if (!neighbors.length) {
                textEl.textContent = 'Ваш стол готов. Список соседей будет доступен позже.';
            } else {
                const lines = ['Ваши соседи:'];
                for (const name of neighbors) {
                    lines.push(`• ${name}`);
                }
                textEl.textContent = lines.join('\n');
            }
        }
    } catch (error) {
        console.error('loadSeatingInfoForCurrentUser error:', error);
    }
}

// Инициализация
loadConfig();

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

// Функция для показа основной страницы
function showMainPage() {
    // Основная страница теперь общая для всех гостей
    const mainPage = document.getElementById('mainPage');
    if (mainPage) {
        mainPage.style.display = 'block';
        // Загружаем данные для основной страницы
        loadMainPageData();
    }
    // Видимость формы RSVP управляется отдельно (логикой регистрации/отправки формы)
}

// При загрузке показываем основную страницу всем пользователям
showMainPage();
// Инициализируем состояние блока RSVP (полная анкета или "добавить гостя")
initRsvpForCurrentUser();

// Обработчик формы RSVP
document.getElementById('guestForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const firstNameInput = document.getElementById('firstName');
    const lastNameInput = document.getElementById('lastName');
    const categorySelect = document.getElementById('category');
    const sideSelect = document.getElementById('side');

    let firstName = firstNameInput ? firstNameInput.value.trim() : '';
    let lastName = lastNameInput ? lastNameInput.value.trim() : '';
    let category = categorySelect ? categorySelect.value : '';
    let side = sideSelect ? sideSelect.value : '';

    // Основные данные гостя в запросе (для уже зарегистрированных берем из localStorage)
    let mainFirstName = firstName;
    let mainLastName = lastName;
    let mainCategory = category;
    let mainSide = side;

    if (!isUserRegistered) {
        // Полная регистрация: проверяем основные поля
        if (firstName.length < 2 || lastName.length < 2) {
            tg.showAlert('Пожалуйста, введите корректные имя и фамилию');
            return;
        }
        
        if (!category || !side) {
            tg.showAlert('Пожалуйста, выберите Родство и Сторону для основного гостя');
            return;
            }
        } else {
        // Пользователь уже зарегистрирован: основные данные берем из localStorage,
        // форму используем только для добавления дополнительных гостей
        const storedFirst = localStorage.getItem('main_guest_first_name');
        const storedLast = localStorage.getItem('main_guest_last_name');
        const storedCat = localStorage.getItem('main_guest_category');
        const storedSide = localStorage.getItem('main_guest_side');

        if (storedFirst && storedLast && storedCat && storedSide) {
            mainFirstName = storedFirst;
            mainLastName = storedLast;
            mainCategory = storedCat;
            mainSide = storedSide;
        }

        // В режиме "добавить гостя" дополнительные гости опциональны:
        // пользователь может добавить их сразу или вернуться позже.
    }

    // Валидация дополнительных гостей (в обоих режимах)
    const invalidGuests = guests.filter(g => 
        !g.firstName.trim() || g.firstName.trim().length < 2 ||
        !g.lastName.trim() || g.lastName.trim().length < 2
    );
    
    if (invalidGuests.length > 0) {
        tg.showAlert('Пожалуйста, заполните имя и фамилию для всех дополнительных гостей');
        return;
    }
    const categoryValue = mainCategory;
    const sideValue = mainSide;
    
    // Получаем данные пользователя из Telegram (несколько способов)
    const userData = getTelegramUserId();
    let userId = userData.userId;
    let firstNameFromTelegram = userData.firstName;
    let lastNameFromTelegram = userData.lastName;
    
    // Если userId не найден, пробуем получить из initData через сервер
    if (!userId && userData.initData) {
        try {
            const response = await fetch(`${CONFIG.apiUrl}/parse-init-data`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ initData: userData.initData })
            });
            
            if (response.ok) {
                const parsed = await response.json();
                if (parsed.userId) {
                    userId = parsed.userId;
                    firstNameFromTelegram = parsed.firstName || firstNameFromTelegram;
                    lastNameFromTelegram = parsed.lastName || lastNameFromTelegram;
                    localStorage.setItem('telegram_user_id', userId.toString());
                }
            }
        } catch (error) {
            console.error('Error parsing initData:', error);
        }
    }
    
    // Если все еще нет userId, пробуем из localStorage
    if (!userId) {
        const savedUserId = localStorage.getItem('telegram_user_id');
        if (savedUserId) {
            userId = parseInt(savedUserId);
            console.log('Using userId from localStorage:', userId);
        }
    }
    
    if (!userId) {
        tg.showAlert('Ошибка: не удалось получить данные пользователя. Пожалуйста, откройте приложение через Telegram.');
        console.error('User ID not found in Telegram data');
        return;
    }
    
    const user = tg.initDataUnsafe?.user;
    const username = user?.username || '';
    
    // Подготавливаем список всех гостей
    // Для дополнительных гостей используем category и side основного гостя
    const allGuests = [
        { firstName: mainFirstName, lastName: mainLastName, category: categoryValue, side: sideValue },
        ...guests.map(g => ({ 
            firstName: g.firstName.trim(), 
            lastName: g.lastName.trim(),
            category: categoryValue,  // Используем category основного гостя
            side: sideValue,  // Используем side основного гостя
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
                const userDataAfter = getTelegramUserId();
                const effectiveUserId = userDataAfter.userId || userId; // Используем userId из формы, если есть
                if (effectiveUserId) {
                    localStorage.setItem(`registered_${effectiveUserId}`, 'true');
                    localStorage.setItem('telegram_user_id', effectiveUserId.toString());
                    // Сохраняем данные основного гостя для будущего режима "добавить гостя"
                    localStorage.setItem('main_guest_first_name', mainFirstName);
                    localStorage.setItem('main_guest_last_name', mainLastName);
                    localStorage.setItem('main_guest_category', categoryValue || '');
                    localStorage.setItem('main_guest_side', sideValue || '');
                }
                
                // Скрываем сообщение об ошибке, если оно было показано
                hideErrorMessage();
                
                // Переключаемся на основную страницу
                showMainPage();
                
                // После успешной регистрации помечаем пользователя как зарегистрированного
                isUserRegistered = true;
                // И переключаем блок на режим "добавить дополнительного гостя"
                setupAddGuestOnlyView();

                // После регистрации открываем блок "СВАДЕБНЫЙ ЧАТ"
                updateGroupSectionVisibility(true);
                
                // Прокручиваем к началу страницы
                window.scrollTo({ top: 0, behavior: 'smooth' });
                
                // Вибрация
                if (tg.HapticFeedback) {
                    tg.HapticFeedback.notificationOccurred('success');
                }

                // После успешной отправки формы перезагружаем страницу,
                // чтобы все состояния (включая блок "СВАДЕБНЫЙ ЧАТ") подтянулись с сервера
                setTimeout(() => {
                    window.location.reload();
                }, 300);
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
            const baseMessage = error.message && typeof error.message === 'string'
                ? error.message
                : 'Ошибка при отправке данных.';
            const finalMessage = `${baseMessage}\n\nФорма не отправилась.\nПожалуйста, обновите или перезагрузите приложение и попробуйте ещё раз.`;
            tg.showAlert(finalMessage);
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
                    localStorage.removeItem('main_guest_first_name');
                    localStorage.removeItem('main_guest_last_name');
                    localStorage.removeItem('main_guest_category');
                    localStorage.removeItem('main_guest_side');
                }

                // После отмены приглашения скрываем блок "СВАДЕБНЫЙ ЧАТ"
                updateGroupSectionVisibility(false);
                
                // После отмены приглашения очищаем форму и снова показываем блок регистрации
                hideSuccessMessage();
                hideErrorMessage();
                
                document.getElementById('firstName').value = '';
                document.getElementById('lastName').value = '';
                document.getElementById('category').value = '';
                document.getElementById('side').value = '';
                guests = [];
                renderGuests();

                const rsvpSection = document.getElementById('rsvpSection');
                if (rsvpSection) {
                    rsvpSection.style.display = 'block';
                }
                
                // Прокручиваем к форме регистрации
                document.getElementById('rsvpSection').scrollIntoView({ behavior: 'smooth' });
                
                tg.showAlert('Приглашение отменено. Вы можете заполнить форму заново.');
                
                // Вибрация
                if (tg.HapticFeedback) {
                    tg.HapticFeedback.notificationOccurred('warning');
                }

                // Перезагружаем страницу, чтобы состояние "незарегистрирован"
                // и стандартная форма регистрации подтянулись с сервера
                setTimeout(() => {
                    window.location.reload();
                }, 300);
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
                
                // После отмены приглашения очищаем форму и снова показываем блок регистрации
                hideSuccessMessage();
                hideErrorMessage();
                
                document.getElementById('firstName').value = '';
                document.getElementById('lastName').value = '';
                document.getElementById('category').value = '';
                document.getElementById('side').value = '';
                guests = [];
                renderGuests();

                const rsvpSection = document.getElementById('rsvpSection');
                if (rsvpSection) {
                    rsvpSection.style.display = 'block';
                }
                
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
    
    // Инициализация анимации дресс-кода
    initDresscodeAnimation();
}

// Инициализация изображения дресс-кода
function initDresscodeAnimation() {
    const container = document.getElementById('dresscodeAnimationContainer');
    const finalElement = document.getElementById('dresscodeAnimationFinal');
    
    if (!container || !finalElement) {
        return;
    }
    
    // Просто показываем картинку при появлении в viewport
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                // Плавное появление картинки
                finalElement.style.transition = 'opacity 0.6s ease-in';
                finalElement.style.opacity = '1';
                observer.unobserve(container);
            }
        });
    }, {
        threshold: 0.3,
        rootMargin: '0px'
    });
    
    // Изначально картинка невидима
    finalElement.style.opacity = '0';
    observer.observe(container);
}

// Инициализация навигации между вкладками
function initTabNavigation() {
    const navItems = document.querySelectorAll('.nav-item');
    const tabContents = document.querySelectorAll('.tab-content');
    
    // Флаг для отслеживания загруженных данных
    let homeDataLoaded = false;
    let timelineDataLoaded = false;
    
    function switchTab(tabName) {
        // Скрываем все вкладки
        tabContents.forEach(tab => {
            tab.style.display = 'none';
        });
        
        // Показываем выбранную вкладку
        const activeTab = document.getElementById(`tab-${tabName}`);
        if (activeTab) {
            activeTab.style.display = 'block';
            // Прокручиваем наверх при переключении
            window.scrollTo({ top: 0, behavior: 'smooth' });
            
            // Загружаем данные для соответствующей вкладки
            if (tabName === 'home') {
                if (!homeDataLoaded) {
                    // Загружаем данные для главной страницы
                    loadHomeTabData();
                    homeDataLoaded = true;
                } else {
                    // Если данные уже загружены, просто инициализируем карту заново
                    setTimeout(() => {
                        initYandexMap();
                    }, 200);
                }
            }
            
            if (tabName === 'timeline') {
                if (!timelineDataLoaded) {
                    // Загружаем тайминг для вкладки план-сетка
                    loadTimelineData();
                    timelineDataLoaded = true;
                } else {
                    // Если данные уже загружены, просто обновляем scroll reveal
                    setTimeout(() => {
                        initScrollReveal();
                    }, 100);
                }
            }
            
            // Инициализируем scroll reveal для новой вкладки
            setTimeout(() => {
                initScrollReveal();
            }, 100);
        }
        
        // Обновляем активное состояние кнопок
        navItems.forEach(item => {
            if (item.dataset.tab === tabName) {
                item.classList.add('active');
            } else {
                item.classList.remove('active');
            }
        });
        
        // Вибрация при переключении
        if (tg.HapticFeedback) {
            tg.HapticFeedback.impactOccurred('light');
        }
    }
    
    // Функция загрузки данных для главной вкладки
    async function loadHomeTabData() {
        // Загружаем конфигурацию если еще не загружена
        await loadConfig();
        
        // Обновляем данные на странице
        updateMainPageUI();
        
        // Загружаем изображение места проведения
        loadVenueImage();
        
        // Инициализируем карту (с небольшой задержкой, чтобы элемент был видим)
        setTimeout(() => {
            initYandexMap();
        }, 200);
        
        // Обновляем контакты
        updateMainContacts();
    }
    
    // Функция загрузки данных для вкладки план-сетка
    async function loadTimelineData() {
        try {
            console.log('Loading timeline from:', `${CONFIG.apiUrl}/timeline`);
            const response = await fetch(`${CONFIG.apiUrl}/timeline`);
            if (response.ok) {
                const data = await response.json();
                console.log('Timeline data received:', data);
                renderTimeline(data.timeline || []);
            } else {
                console.error('Failed to load timeline:', response.status, await response.text());
                const container = document.getElementById('timelineContainer');
                if (container) {
                    container.innerHTML = '<p style="text-align: center; color: var(--color-text-secondary);">Ошибка загрузки плана дня. Попробуйте позже.</p>';
                }
            }
        } catch (error) {
            console.error('Error loading timeline:', error);
            const container = document.getElementById('timelineContainer');
            if (container) {
                container.innerHTML = '<p style="text-align: center; color: var(--color-text-secondary);">Ошибка загрузки плана дня. Попробуйте позже.</p>';
            }
        }
    }
    
    // Обработчики кликов на кнопки навбара
    navItems.forEach(item => {
        item.addEventListener('click', () => {
            const tabName = item.dataset.tab;
            if (tabName) {
                switchTab(tabName);
            }
        });
    });
    
    // По умолчанию показываем главную вкладку и загружаем данные
    switchTab('home');
    
    // Также загружаем данные при первой загрузке страницы (на случай, если пользователь уже на главной)
    if (document.getElementById('tab-home') && document.getElementById('tab-home').style.display !== 'none') {
        loadHomeTabData();
        homeDataLoaded = true;
    }
}

// Инициализация навигации после загрузки DOM
document.addEventListener('DOMContentLoaded', () => {
    initTabNavigation();
});
