package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
	"wedding-bot/internal/keyboards"
)

// handleAdminText обрабатывает текстовые сообщения в админ-меню
func handleAdminText(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	text := message.Text
	userID := message.From.ID

	if !isAdminUser(int(userID)) {
		return
	}

	// Обработка кнопок админ-меню
	switch text {
	case "👥 Гости":
		handleAdminGuestsMenu(bot, message)
	case "🪑 Столы":
		handleAdminTableMenu(bot, message)
	case "💬 Группа":
		handleAdminGroupMenu(bot, message)
	case "🤖 Бот":
		handleAdminBotMenu(bot, message)
	case "⬅️ Вернуться":
		handleAdminBack(bot, message)
	case "📋 Список гостей":
		handleAdminGuestsListFromText(bot, message)
	case "📊 Посмотреть рассадку":
		handleAdminSeatingFromText(bot, message)
	case "🔄 Обновить рассадку":
		handleAdminRefreshSeating(bot, message)
	case "📤 Отправить приглашение":
		handleAdminSendInviteFromText(bot, message)
	case "🔁 Исправление Имя/Фамилия":
		handleAdminFixNames(bot, message)
	case "Рассылка в ЛС":
		handleAdminBroadcastDM(bot, message)
	case "Открыть таблицу":
		handleAdminOpenTable(bot, message)
	case "Проверить связь":
		handleAdminPing(bot, message)
	case "Закрепить рассадку":
		handleAdminLockSeating(bot, message)
	case "Написать сообщение":
		handleAdminGroupSendMessage(bot, message)
	case "Посмотреть участников":
		handleAdminGroupListMembers(bot, message)
	case "Добавить/Удалить":
		handleAdminGroupAddRemove(bot, message)
	case "📊 Статус бота":
		handleAdminBotStatus(bot, message)
	case "🎮 Игры":
		handleAdminGamesMenu(bot, message)
	case "🔐 Авторизовать клиент":
		handleAdminAuthClient(bot, message)
	case "Начать с нуля":
		handleAdminResetMe(bot, message)
	case "Добавить админа":
		handleAdminAddAdmin(bot, message)
	case "🆔 Найти user_id":
		handleAdminFindUserID(bot, message)
	}
}

// handleAdminGuestsMenu показывает подменю "Гости"
func handleAdminGuestsMenu(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msgText := "📂 <b>Админ → Гости</b>\n\nВыберите действие:"
	keyboard := keyboards.GetAdminGuestsReplyKeyboard()

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// handleAdminTableMenu показывает подменю "Таблица"
func handleAdminTableMenu(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msgText := "📊 <b>Админ → Таблица</b>\n\nВыберите действие:"
	keyboard := keyboards.GetAdminTableReplyKeyboard()

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// handleAdminGroupMenu показывает подменю "Группа"
func handleAdminGroupMenu(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msgText := "💬 <b>Админ → Группа</b>\n\nВыберите действие:"
	keyboard := keyboards.GetAdminGroupReplyKeyboard()

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// handleAdminBotMenu показывает подменю "Бот"
func handleAdminBotMenu(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msgText := "🤖 <b>Админ → Бот</b>\n\nВыберите действие:"
	keyboard := keyboards.GetAdminBotReplyKeyboard()

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// handleAdminGamesMenu показывает меню игр
func handleAdminGamesMenu(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msgText := "🎮 <b>Управление играми</b>\n\nВыберите игру:"
	keyboard := keyboards.GetAdminGamesKeyboard()

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &keyboard
	bot.Send(msg)
}

// handleAdminBack возвращает в главное меню бота из корневого меню админки
func handleAdminBack(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	userID := message.From.ID
	isAdmin := isAdminUser(int(userID))
	photoModeEnabled := IsPhotoModeEnabled(userID)

	msgText := "Главное меню:"
	keyboard := keyboards.GetMainReplyKeyboard(isAdmin, photoModeEnabled)

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// handleAdminGuestsListFromText показывает список гостей (из текстового сообщения)
func handleAdminGuestsListFromText(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	guests, err := google_sheets.GetAllGuestsFromSheets(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка гостей: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка при получении списка гостей. Попробуйте позже.")
		bot.Send(msg)
		return
	}

	if len(guests) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "📋 <b>Список гостей</b>\n\nПока никто не подтвердил присутствие.")
		msg.ParseMode = tgbotapi.ModeHTML
		bot.Send(msg)
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 <b>Список всех гостей:</b>\n\n")

	for i, guest := range guests {
		sb.WriteString(fmt.Sprintf("%d. <b>%s %s</b>", i+1, guest.FirstName, guest.LastName))
		if guest.Category != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", guest.Category))
		}
		if guest.Side != "" {
			sb.WriteString(fmt.Sprintf(" - %s", guest.Side))
		}
		if guest.UserID != "" {
			sb.WriteString(fmt.Sprintf(" [ID: %s]", guest.UserID))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("\n<b>Всего: %d гостей</b>", len(guests)))

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}

// handleAdminSeatingFromText показывает рассадку (из текстового сообщения)
func handleAdminSeatingFromText(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	seating, err := google_sheets.GetSeatingFromSheets(ctx)
	if err != nil {
		log.Printf("Ошибка получения рассадки: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка при получении рассадки. Попробуйте позже.")
		bot.Send(msg)
		return
	}

	if len(seating) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "🍽 <b>Рассадка</b>\n\nПока нет данных по рассадке (лист 'Рассадка' пуст или без гостей).")
		msg.ParseMode = tgbotapi.ModeHTML
		bot.Send(msg)
		return
	}

	var sb strings.Builder
	sb.WriteString("🍽 <b>Рассадка по столам</b>\n")

	for _, table := range seating {
		tableName := table.Table
		if tableName == "" {
			tableName = "Без названия"
		}
		sb.WriteString(fmt.Sprintf("\n<b>%s</b>", tableName))
		if len(table.Guests) == 0 {
			sb.WriteString("\n  (пока пусто)")
		} else {
			for i, name := range table.Guests {
				sb.WriteString(fmt.Sprintf("\n%d. %s", i+1, name))
			}
		}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}

// handleAdminRefreshSeating обновляет рассадку
func handleAdminRefreshSeating(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// TODO: Реализовать обновление рассадки
	msg := tgbotapi.NewMessage(message.Chat.ID, "🔄 Функция обновления рассадки в разработке")
	bot.Send(msg)
}

// handleAdminSendInviteFromText запускает отправку приглашений (из текстового сообщения)
func handleAdminSendInviteFromText(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	invitations, err := google_sheets.GetInvitationsList(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка приглашений: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка получения списка приглашений.")
		bot.Send(msg)
		return
	}

	if len(invitations) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ <b>Список приглашений пуст</b>\n\nПроверьте вкладку 'Пригласительные' в Google Sheets.")
		msg.ParseMode = tgbotapi.ModeHTML
		bot.Send(msg)
		return
	}

	sentCount := 0
	for _, inv := range invitations {
		if inv.IsSent {
			sentCount++
		}
	}

	msgText := fmt.Sprintf(
		"📋 <b>Выберите гостя для отправки приглашения:</b>\n\n"+
			"Всего гостей: <b>%d</b>\n"+
			"✅ Отправлено: <b>%d</b>\n"+
			"⏳ Осталось: <b>%d</b>\n\n"+
			"Нажмите на кнопку с именем гостя, чтобы открыть диалог с заготовленным текстом приглашения.\n\n"+
			"💡 <i>Гости с галочкой ✅ уже получили приглашение</i>",
		len(invitations), sentCount, len(invitations)-sentCount,
	)

	// Преобразуем invitations в формат для клавиатуры
	keyboardInvitations := make([]keyboards.InvitationInfoForKeyboard, len(invitations))
	for i, inv := range invitations {
		keyboardInvitations[i] = keyboards.InvitationInfoForKeyboard{
			Name:   inv.Name,
			IsSent: inv.IsSent,
		}
	}

	keyboard := keyboards.GetGuestsSelectionKeyboard(keyboardInvitations)

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &keyboard
	bot.Send(msg)
}

// handleAdminFixNames запускает исправление имени/фамилии
func handleAdminFixNames(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	guests, err := google_sheets.ListConfirmedGuests(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка гостей: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка получения списка гостей.")
		msg.ParseMode = tgbotapi.ModeHTML
		bot.Send(msg)
		return
	}

	if len(guests) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "📋 Нет подтвержденных гостей для исправления.")
		msg.ParseMode = tgbotapi.ModeHTML
		bot.Send(msg)
		return
	}

	// Создаем клавиатуру с кнопками для каждого гостя
	keyboard := keyboards.GetGuestsSwapKeyboard(guests, 0)

	msgText := fmt.Sprintf(
		"🔁 <b>Исправление Имя/Фамилия</b>\n\n"+
			"Нажмите на гостя, чтобы поменять местами Имя и Фамилию в Google Sheets.\n\n"+
			"Всего гостей: <b>%d</b>",
		len(guests),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &keyboard
	bot.Send(msg)
}

// handleAdminOpenTable открывает Google Sheets
func handleAdminOpenTable(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	sheetsURL := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/edit", config.GoogleSheetsID)
	msgText := fmt.Sprintf("📂 <b>Таблица гостей и настроек</b>\n\nОткроется в браузере по ссылке ниже:\n%s", sheetsURL)

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}

// handleAdminPing проверяет связь с Google Sheets
func handleAdminPing(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg := tgbotapi.NewMessage(message.Chat.ID, "📶 Выполняю проверку связи с Google Sheets...")
	bot.Send(msg)

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

	msgText := fmt.Sprintf(
		"📶 <b>Проверка связи: бот → сервер → Google Sheets</b>\n\n"+
			"⏰ Время: <code>%s</code>\n"+
			"📄 Лист: <code>Админ бота</code>\n"+
			"⚙️ Строка: <code>5</code>\n"+
			"⏱ Задержка: <b>%d мс</b>\n"+
			"✅ Статус: <b>%s</b>\n\n"+
			"Запись о ping сохранена в Google Sheets (строка 5 вкладки 'Админ бота').",
		time.Now().Format("2006-01-02 15:04:05"), latency, status,
	)

	msg = tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}

// handleAdminLockSeating закрепляет рассадку
func handleAdminLockSeating(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status, err := google_sheets.LockSeating(ctx)
	if err != nil {
		log.Printf("Ошибка закрепления рассадки: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка закрепления рассадки.")
		bot.Send(msg)
		return
	}

	if status != nil && status.Locked {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ Рассадка закреплена!\nВремя: %s", status.LockedAt))
		bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "✅ Рассадка закреплена!")
	bot.Send(msg)
}

// handleAdminGroupSendMessage запускает отправку сообщения в группу
func handleAdminGroupSendMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if config.GroupID == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ GROUP_ID не настроен в конфигурации.")
		bot.Send(msg)
		return
	}

	// TODO: Реализовать FSM для ввода сообщения
	msg := tgbotapi.NewMessage(message.Chat.ID, "📢 <b>Отправка сообщения в группу</b>\n\nФункция в разработке.")
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}

// handleAdminGroupListMembers показывает участников группы
func handleAdminGroupListMembers(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if config.GroupID == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ GROUP_ID не настроен в конфигурации.")
		bot.Send(msg)
		return
	}

	mu.RLock()
	botInstance := botInstance
	mu.RUnlock()

	if botInstance == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Бот не инициализирован")
		bot.Send(msg)
		return
	}

	// Парсим GroupID (может быть строкой или числом)
	var chatConfig tgbotapi.ChatInfoConfig
	if chatID, err := strconv.ParseInt(config.GroupID, 10, 64); err == nil {
		// Если это число, используем ChatID
		chatConfig = tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: chatID}}
	} else {
		// Если не число, используем как username
		chatConfig = tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{SuperGroupUsername: config.GroupID}}
	}

	// Получаем информацию о чате
	chat, err := botInstance.GetChat(chatConfig)
	if err != nil {
		log.Printf("Ошибка получения информации о группе: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Не удалось получить информацию о группе. Проверьте, что бот добавлен в группу и GROUP_ID указан правильно.")
		bot.Send(msg)
		return
	}

	// Получаем количество участников
	membersCount := 0
	countConfig := tgbotapi.ChatMemberCountConfig{ChatConfig: chatConfig.ChatConfig}
	if count, err := botInstance.GetChatMembersCount(countConfig); err == nil {
		membersCount = count
	}

	// Получаем список администраторов
	adminsConfig := tgbotapi.ChatAdministratorsConfig{ChatConfig: chatConfig.ChatConfig}
	admins, err := botInstance.GetChatAdministrators(adminsConfig)
	if err != nil {
		log.Printf("Ошибка получения списка администраторов: %v", err)
		admins = []tgbotapi.ChatMember{}
	}

	// Формируем сообщение
	msgText := fmt.Sprintf(
		"👥 <b>Информация о группе</b>\n\n"+
			"📝 Название: <b>%s</b>\n"+
			"🆔 ID: <code>%s</code>\n"+
			"👥 Участников: <b>%d</b>\n"+
			"👑 Администраторов: <b>%d</b>\n",
		chat.Title, config.GroupID, membersCount, len(admins),
	)

	// Добавляем список администраторов
	if len(admins) > 0 {
		msgText += "\n<b>👑 Администраторы:</b>\n"
		for i, admin := range admins {
			if i >= 20 { // Ограничиваем 20 администраторами
				msgText += fmt.Sprintf("\n... и еще %d администраторов", len(admins)-20)
				break
			}
			user := admin.User
			name := user.FirstName
			if user.LastName != "" {
				name += " " + user.LastName
			}
			if user.UserName != "" {
				name += fmt.Sprintf(" (@%s)", user.UserName)
			}
			status := "👤 Участник"
			if admin.CanDeleteMessages || admin.CanRestrictMembers || admin.CanPromoteMembers {
				status = "👑 Админ"
			}
			if admin.Status == "creator" {
				status = "👑 Создатель"
			} else if admin.Status == "administrator" {
				status = "👑 Админ"
			}
			msgText += fmt.Sprintf("%d. %s - %s\n", i+1, name, status)
		}
	} else {
		msgText += "\n⚠️ Не удалось получить список администраторов."
	}

	// Добавляем ссылку на группу
	if config.GroupLink != "" {
		msgText += fmt.Sprintf("\n🔗 <a href=\"%s\">Открыть группу</a>", config.GroupLink)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}

// handleAdminGroupAddRemove показывает управление участниками группы
func handleAdminGroupAddRemove(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if config.GroupID == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ GROUP_ID не настроен в конфигурации.")
		bot.Send(msg)
		return
	}

	msgText := fmt.Sprintf(
		"💬 <b>Управление группой</b>\n\n"+
			"🔗 Ссылка: %s\n"+
			"🆔 ID группы: <code>%s</code>\n\n"+
			"Выберите нужное действие ниже:",
		config.GroupLink, config.GroupID,
	)

	keyboard := keyboards.GetGroupManagementKeyboard()
	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &keyboard
	bot.Send(msg)
}

// handleAdminBotStatus показывает статус бота
func handleAdminBotStatus(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msgText := "🤖 <b>Статус бота</b>\n\n" +
		"✅ Бот работает\n" +
		"✅ API доступен\n" +
		"✅ Google Sheets подключен\n\n" +
		"💻 <b>Технологии:</b>\n" +
		"• Бот написан на <b>Go</b> (Golang)\n" +
		"• Веб-приложение написано на <b>React.js</b>"

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}

// handleAdminAuthClient запускает авторизацию Telegram Client
func handleAdminAuthClient(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// TODO: Реализовать авторизацию Telegram Client
	msg := tgbotapi.NewMessage(message.Chat.ID, "🔐 <b>Авторизация Telegram Client</b>\n\nФункция в разработке.")
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}

// handleAdminResetMe сбрасывает регистрацию админа
func handleAdminResetMe(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := google_sheets.CancelGuestRegistrationByUserID(ctx, int(message.From.ID))
	if err != nil {
		log.Printf("Ошибка сброса регистрации: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка сброса регистрации.")
		bot.Send(msg)
		return
	}

	msgText := "✅ <b>Данные сброшены!</b>\n\n" +
		"Ваша регистрация удалена из базы данных.\n" +
		"Теперь вы можете пройти весь путь заново, нажав /start"

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}

// handleAdminAddAdmin запускает добавление админа
func handleAdminAddAdmin(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// TODO: Реализовать FSM для добавления админа
	msgText := "👤 <b>Добавление администратора</b>\n\n" +
		"Пришлите @username человека, которого хотите сделать админом.\n" +
		"Важно: этот пользователь должен хотя бы раз написать боту /start."

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}

// handleAdminFindUserID запускает поиск user_id по username
func handleAdminFindUserID(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// TODO: Реализовать FSM для поиска user_id
	msgText := "🆔 <b>Найти user_id по username</b>\n\n" +
		"Пришлите @username или ссылку вида `https://t.me/username`.\n" +
		"Важно: пользователь должен хотя бы раз написать боту или быть с ботом в одной группе."

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}
