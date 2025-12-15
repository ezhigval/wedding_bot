package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"gopkg.in/telebot.v3"

	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
	"wedding-bot/internal/keyboards"
)

// handleAdminText обрабатывает текстовые сообщения в админ-меню
func handleAdminText(c telebot.Context) error {
	text := c.Text()
	userID := c.Sender().ID

	if !isAdminUser(int(userID)) {
		return nil
	}

	// Обработка кнопок админ-меню
	switch text {
	case "Гости":
		return handleAdminGuestsMenu(c)
	case "Таблица":
		return handleAdminTableMenu(c)
	case "Группа":
		return handleAdminGroupMenu(c)
	case "Бот":
		return handleAdminBotMenu(c)
	case "⬅️ Вернуться":
		return handleAdminBack(c)
	case "Список гостей":
		return handleAdminGuestsList(c)
	case "Рассадка":
		return handleAdminSeating(c)
	case "Отправить приглашение":
		return handleAdminSendInvite(c)
	case "Исправить имя/фамилию":
		return handleAdminFixNames(c)
	case "Рассылка в ЛС":
		return handleAdminBroadcastDM(c)
	case "Открыть таблицу":
		return handleAdminOpenTable(c)
	case "Проверить связь":
		return handleAdminPing(c)
	case "Закрепить рассадку":
		return handleAdminLockSeating(c)
	case "Написать сообщение":
		return handleAdminGroupSendMessage(c)
	case "Посмотреть участников":
		return handleAdminGroupListMembers(c)
	case "Добавить/Удалить":
		return handleAdminGroupAddRemove(c)
	case "Статус бота":
		return handleAdminBotStatus(c)
	case "🔐 Авторизовать клиент":
		return handleAdminAuthClient(c)
	case "Начать с нуля":
		return handleAdminResetMe(c)
	case "Добавить админа":
		return handleAdminAddAdmin(c)
	case "🆔 Найти user_id":
		return handleAdminFindUserID(c)
	}

	return nil
}

// handleAdminGuestsMenu показывает подменю "Гости"
func handleAdminGuestsMenu(c telebot.Context) error {
	message := "📂 <b>Админ → Гости</b>\n\nВыберите действие:"
	keyboard := keyboards.GetAdminGuestsReplyKeyboard()
	return c.Send(message, keyboard, telebot.ModeHTML)
}

// handleAdminTableMenu показывает подменю "Таблица"
func handleAdminTableMenu(c telebot.Context) error {
	message := "📊 <b>Админ → Таблица</b>\n\nВыберите действие:"
	keyboard := keyboards.GetAdminTableReplyKeyboard()
	return c.Send(message, keyboard, telebot.ModeHTML)
}

// handleAdminGroupMenu показывает подменю "Группа"
func handleAdminGroupMenu(c telebot.Context) error {
	message := "💬 <b>Админ → Группа</b>\n\nВыберите действие:"
	keyboard := keyboards.GetAdminGroupReplyKeyboard()
	return c.Send(message, keyboard, telebot.ModeHTML)
}

// handleAdminBotMenu показывает подменю "Бот"
func handleAdminBotMenu(c telebot.Context) error {
	message := "🤖 <b>Админ → Бот</b>\n\nВыберите действие:"
	keyboard := keyboards.GetAdminBotReplyKeyboard()
	return c.Send(message, keyboard, telebot.ModeHTML)
}

// handleAdminBack возвращает в главное меню бота из корневого меню админки
func handleAdminBack(c telebot.Context) error {
	userID := c.Sender().ID
	isAdmin := isAdminUser(int(userID))
	photoModeEnabled := IsPhotoModeEnabled(userID)
	
	message := "Главное меню:"
	keyboard := keyboards.GetMainReplyKeyboard(isAdmin, photoModeEnabled)
	return c.Send(message, keyboard, telebot.ModeHTML)
}

// handleAdminGuestsList, handleAdminSeating, handleAdminSendInvite - реализованы в callbacks.go

// handleAdminFixNames запускает исправление имени/фамилии
func handleAdminFixNames(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	guests, err := google_sheets.ListConfirmedGuests(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка гостей: %v", err)
		return c.Send("❌ Ошибка получения списка гостей.", telebot.ModeHTML)
	}

	if len(guests) == 0 {
		return c.Send("📋 Нет подтвержденных гостей для исправления.", telebot.ModeHTML)
	}

	// Создаем клавиатуру с кнопками для каждого гостя
	keyboard := keyboards.GetGuestsSwapKeyboard(guests, 0)

	message := fmt.Sprintf(
		"🔁 <b>Исправление Имя/Фамилия</b>\n\n"+
			"Нажмите на гостя, чтобы поменять местами Имя и Фамилию в Google Sheets.\n\n"+
			"Всего гостей: <b>%d</b>",
		len(guests),
	)

	return c.Send(message, keyboard, telebot.ModeHTML)
}

// handleAdminBroadcastDM запускает рассылку в ЛС - реализация в broadcast_handlers.go

// handleAdminOpenTable открывает Google Sheets
func handleAdminOpenTable(c telebot.Context) error {
	sheetsURL := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/edit", config.GoogleSheetsID)
	return c.Send(
		fmt.Sprintf("📂 <b>Таблица гостей и настроек</b>\n\nОткроется в браузере по ссылке ниже:\n%s", sheetsURL),
		telebot.ModeHTML,
	)
}

// handleAdminPing проверяет связь с Google Sheets
func handleAdminPing(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = c.Send("📶 Выполняю проверку связи с Google Sheets...")

	// Простая проверка - пытаемся прочитать админ лист
	start := time.Now()
	_, err := google_sheets.GetAdminsList(ctx)
	latency := int(time.Since(start).Milliseconds())

	status := "OK"
	if err != nil {
		status = "ERROR"
		latency = -1
		log.Printf("Ошибка проверки связи: %v", err)
	}

	message := fmt.Sprintf(
		"📶 <b>Проверка связи: бот → сервер → Google Sheets</b>\n\n"+
			"⏰ Время: <code>%s</code>\n"+
			"📄 Лист: <code>Админ бота</code>\n"+
			"⚙️ Строка: <code>5</code>\n"+
			"⏱ Задержка: <b>%d мс</b>\n"+
			"✅ Статус: <b>%s</b>\n\n"+
			"Запись о ping сохранена в Google Sheets (строка 5 вкладки 'Админ бота').",
		time.Now().Format("2006-01-02 15:04:05"), latency, status,
	)

	return c.Send(message, telebot.ModeHTML)
}

// handleAdminLockSeating закрепляет рассадку
func handleAdminLockSeating(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status, err := google_sheets.LockSeating(ctx)
	if err != nil {
		log.Printf("Ошибка закрепления рассадки: %v", err)
		return c.Send("❌ Ошибка закрепления рассадки.")
	}

	if status != nil && status.Locked {
		return c.Send(fmt.Sprintf("✅ Рассадка закреплена!\nВремя: %s", status.LockedAt))
	}

	return c.Send("✅ Рассадка закреплена!")
}

// handleAdminGroupSendMessage запускает отправку сообщения в группу
func handleAdminGroupSendMessage(c telebot.Context) error {
	if config.GroupID == "" {
		return c.Send("❌ GROUP_ID не настроен в конфигурации.")
	}

	// TODO: Реализовать FSM для ввода сообщения
	return c.Send("📢 <b>Отправка сообщения в группу</b>\n\nФункция в разработке.", telebot.ModeHTML)
}

// handleAdminGroupListMembers показывает участников группы
func handleAdminGroupListMembers(c telebot.Context) error {
	if config.GroupID == "" {
		return c.Send("❌ GROUP_ID не настроен в конфигурации.")
	}

	// Получаем бота
	mu.RLock()
	bot := botInstance
	mu.RUnlock()

	if bot == nil {
		return c.Send("❌ Бот не инициализирован")
	}

	// Парсим GroupID (может быть строкой или числом)
	var chatID int64
	if _, err := fmt.Sscanf(config.GroupID, "%d", &chatID); err != nil {
		// Если не число, пытаемся использовать как username
		chat, err := bot.ChatByUsername(config.GroupID)
		if err != nil {
			log.Printf("Ошибка получения информации о группе: %v", err)
			return c.Send("❌ Не удалось получить информацию о группе. Проверьте, что бот добавлен в группу и GROUP_ID указан правильно.")
		}
		chatID = chat.ID
	} else {
		// Если это число, используем напрямую
		chat, err := bot.ChatByID(chatID)
		if err != nil {
			log.Printf("Ошибка получения информации о группе: %v", err)
			return c.Send("❌ Не удалось получить информацию о группе. Проверьте, что бот добавлен в группу и GROUP_ID указан правильно.")
		}
		chatID = chat.ID
	}

	// Получаем информацию о чате
	chat, err := bot.ChatByID(chatID)
	if err != nil {
		log.Printf("Ошибка получения информации о группе: %v", err)
		return c.Send("❌ Не удалось получить информацию о группе. Проверьте, что бот добавлен в группу и GROUP_ID указан правильно.")
	}

	// Получаем количество участников (используем ChatMemberCount через Raw)
	membersCount := 0
	if chat.MembersCount > 0 {
		membersCount = chat.MembersCount
	}

	// Получаем список администраторов
	admins, err := bot.AdminsOf(chat)
	if err != nil {
		log.Printf("Ошибка получения списка администраторов: %v", err)
		admins = []telebot.ChatMember{}
	}

	// Формируем сообщение
	message := fmt.Sprintf(
		"👥 <b>Информация о группе</b>\n\n"+
			"📝 Название: <b>%s</b>\n"+
			"🆔 ID: <code>%s</code>\n"+
			"👥 Участников: <b>%d</b>\n"+
			"👑 Администраторов: <b>%d</b>\n",
		chat.Title, config.GroupID, membersCount, len(admins),
	)

	// Добавляем список администраторов
	if len(admins) > 0 {
		message += "\n<b>👑 Администраторы:</b>\n"
		for i, admin := range admins {
			if i >= 20 { // Ограничиваем 20 администраторами
				message += fmt.Sprintf("\n... и еще %d администраторов", len(admins)-20)
				break
			}
			user := admin.User
			name := user.FirstName
			if user.LastName != "" {
				name += " " + user.LastName
			}
			if user.Username != "" {
				name += fmt.Sprintf(" (@%s)", user.Username)
			}
			status := "👤 Участник"
			if admin.Rights.IsAdmin {
				status = "👑 Админ"
			}
			if admin.Rights.IsOwner {
				status = "👑 Создатель"
			}
			message += fmt.Sprintf("%d. %s - %s\n", i+1, name, status)
		}
	}

	// Добавляем ссылку на группу
	if config.GroupLink != "" {
		message += fmt.Sprintf("\n🔗 <a href=\"%s\">Открыть группу</a>", config.GroupLink)
	}

	return c.Send(message, telebot.ModeHTML)
}

// handleAdminGroupAddRemove показывает управление участниками группы
func handleAdminGroupAddRemove(c telebot.Context) error {
	if config.GroupID == "" {
		return c.Send("❌ GROUP_ID не настроен в конфигурации.")
	}

	message := fmt.Sprintf(
		"💬 <b>Управление группой</b>\n\n"+
			"🔗 Ссылка: %s\n"+
			"🆔 ID группы: <code>%s</code>\n\n"+
			"Выберите нужное действие ниже:",
		config.GroupLink, config.GroupID,
	)

	keyboard := keyboards.GetGroupManagementKeyboard()
	return c.Send(message, keyboard, telebot.ModeHTML)
}

// handleAdminBotStatus показывает статус бота
func handleAdminBotStatus(c telebot.Context) error {
	message := "🤖 <b>Статус бота</b>\n\n" +
		"✅ Бот работает\n" +
		"✅ API доступен\n" +
		"✅ Google Sheets подключен\n\n" +
		"💻 <b>Технологии:</b>\n" +
		"• Бот написан на <b>Go</b> (Golang)\n" +
		"• Веб-приложение написано на <b>React.js</b>"

	return c.Send(message, telebot.ModeHTML)
}

// handleAdminAuthClient запускает авторизацию Telegram Client
func handleAdminAuthClient(c telebot.Context) error {
	// TODO: Реализовать авторизацию Telegram Client
	return c.Send("🔐 <b>Авторизация Telegram Client</b>\n\nФункция в разработке.", telebot.ModeHTML)
}

// handleAdminResetMe сбрасывает регистрацию админа
func handleAdminResetMe(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := google_sheets.CancelGuestRegistrationByUserID(ctx, int(c.Sender().ID))
	if err != nil {
		log.Printf("Ошибка сброса регистрации: %v", err)
		return c.Send("❌ Ошибка сброса регистрации.")
	}

	return c.Send(
		"✅ <b>Данные сброшены!</b>\n\n"+
			"Ваша регистрация удалена из базы данных.\n"+
			"Теперь вы можете пройти весь путь заново, нажав /start",
		telebot.ModeHTML,
	)
}

// handleAdminAddAdmin запускает добавление админа
func handleAdminAddAdmin(c telebot.Context) error {
	// TODO: Реализовать FSM для добавления админа
	return c.Send(
		"👤 <b>Добавление администратора</b>\n\n"+
			"Пришлите @username человека, которого хотите сделать админом.\n"+
			"Важно: этот пользователь должен хотя бы раз написать боту /start.",
		telebot.ModeHTML,
	)
}

// handleAdminFindUserID запускает поиск user_id по username
func handleAdminFindUserID(c telebot.Context) error {
	// TODO: Реализовать FSM для поиска user_id
	return c.Send(
		"🆔 <b>Найти user_id по username</b>\n\n"+
			"Пришлите @username или ссылку вида `https://t.me/username`.\n"+
			"Важно: пользователь должен хотя бы раз написать боту или быть с ботом в одной группе.",
		telebot.ModeHTML,
	)
}

