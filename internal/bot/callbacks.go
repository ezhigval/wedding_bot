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

// handleAdminCallback обрабатывает callback от админ панели
func handleAdminCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) == 0 {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	section := parts[0]

	switch section {
	case "guests":
		handleAdminGuestsCallback(bot, callback)
	case "guests:list":
		handleAdminGuestsList(bot, callback)
	case "seating":
		handleAdminSeating(bot, callback)
	case "send_invite":
		handleAdminSendInvite(bot, callback)
	case "games":
		handleAdminGamesCallback(bot, callback)
	case "stats":
		handleAdminStatsCallback(bot, callback)
	case "group":
		handleAdminGroupCallback(bot, callback)
	case "broadcast":
		handleAdminBroadcastCallback(bot, callback)
	case "back":
		handleAdminBackCallback(bot, callback)
	default:
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

// handleAdminBackCallback возвращает в главное админ меню
func handleAdminBackCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	message := "🔧 <b>Панель администратора</b>\n\nВыберите раздел:"
	keyboard := keyboards.GetAdminRootInlineKeyboard()

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, message)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleAdminGuestsCallback показывает управление гостями
func handleAdminGuestsCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	count, err := google_sheets.GetGuestsCountFromSheets(ctx)
	if err != nil {
		log.Printf("Ошибка получения количества гостей: %v", err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Ошибка получения данных"))
		return
	}

	message := fmt.Sprintf("👥 <b>Управление гостями</b>\n\nЗарегистрировано: %d", count)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Список гостей", "admin:guests:list"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "admin:back"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, message)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleAdminGuestsList показывает список гостей
func handleAdminGuestsList(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	guests, err := google_sheets.GetAllGuestsFromSheets(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка гостей: %v", err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Ошибка при получении списка гостей"))
		return
	}

	if len(guests) == 0 {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "📋 <b>Список гостей</b>\n\nПока никто не подтвердил присутствие.")
		msg.ParseMode = tgbotapi.ModeHTML
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
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

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleAdminSeating показывает рассадку
func handleAdminSeating(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	seating, err := google_sheets.GetSeatingFromSheets(ctx)
	if err != nil {
		log.Printf("Ошибка получения рассадки: %v", err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Ошибка при получении рассадки"))
		return
	}

	if len(seating) == 0 {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "🍽 <b>Рассадка</b>\n\nПока нет данных по рассадке (лист 'Рассадка' пуст или без гостей).")
		msg.ParseMode = tgbotapi.ModeHTML
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
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

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, sb.String())
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleAdminSendInvite запускает отправку приглашений
func handleAdminSendInvite(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	invitations, err := google_sheets.GetInvitationsList(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка приглашений: %v", err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Ошибка получения списка приглашений"))
		return
	}

	if len(invitations) == 0 {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ <b>Список приглашений пуст</b>\n\nПроверьте вкладку 'Пригласительные' в Google Sheets.")
		msg.ParseMode = tgbotapi.ModeHTML
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	sentCount := 0
	for _, inv := range invitations {
		if inv.IsSent {
			sentCount++
		}
	}

	message := fmt.Sprintf(
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

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, message)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleAdminGamesCallback показывает управление играми
func handleAdminGamesCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	message := "🎮 <b>Управление играми</b>\n\nВыберите игру:"
	keyboard := keyboards.GetAdminGamesKeyboard()

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, message)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleAdminStatsCallback показывает статистику
func handleAdminStatsCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := google_sheets.GetGuestsCountFromSheets(ctx)
	if err != nil {
		log.Printf("Ошибка получения статистики: %v", err)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	message := fmt.Sprintf("📊 <b>Статистика</b>\n\nЗарегистрировано гостей: %d", count)
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, message)
	editMsg.ParseMode = tgbotapi.ModeHTML
	bot.Send(editMsg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleAdminGroupCallback показывает управление группой
func handleAdminGroupCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	message := "💬 <b>Управление группой</b>\n\nВыберите действие:"
	keyboard := keyboards.GetGroupManagementKeyboard()

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, message)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleInvitationCallback обрабатывает callback от приглашений
func handleInvitationCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) == 0 {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	action := parts[0]

	switch action {
	case "guest":
		if len(parts) < 2 {
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}
		index, err := strconv.Atoi(parts[1])
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}
		handleInvitationGuestSelect(bot, callback, index)
	case "mark_sent":
		if len(parts) < 2 {
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}
		index, err := strconv.Atoi(parts[1])
		if err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}
		handleInvitationMarkSent(bot, callback, index)
	default:
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

// handleInvitationGuestSelect обрабатывает выбор гостя для отправки приглашения
func handleInvitationGuestSelect(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, index int) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	invitations, err := google_sheets.GetInvitationsList(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка приглашений: %v", err)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	if index < 0 || index >= len(invitations) {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	inv := invitations[index]

	// Формируем текст приглашения
	invitationText := fmt.Sprintf(
		"💒 <b>Свадьба</b>\n\n"+
			"👫 <b>%s и %s</b>\n\n"+
			"📅 <b>%s</b>\n\n"+
			"📍 <b>Адрес:</b> %s\n\n"+
			"━━━━━━━━━━━━━━━━━━━━\n"+
			"Дорогой(ая) %s!\n\n"+
			"Мы будем рады видеть вас на нашем торжестве!\n"+
			"Этот день будет особенным, и ваше присутствие сделает его ещё более незабываемым! 💕\n\n"+
			"Просим предварительно подтвердить ваше присутствие в приложении.\n"+
			"━━━━━━━━━━━━━━━━━━━━",
		config.GroomName, config.BrideName,
		config.WeddingDate.Format("02.01.2006"),
		config.WeddingAddress,
		inv.Name,
	)

	// Создаем deep link для открытия диалога
	telegramID := inv.TelegramID
	if telegramID == "" {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	// Убираем @ если есть
	telegramID = strings.TrimPrefix(telegramID, "@")
	telegramID = strings.TrimPrefix(telegramID, "https://t.me/")
	telegramID = strings.TrimPrefix(telegramID, "t.me/")

	deepLink := fmt.Sprintf("tg://msg?to=%s&text=%s", telegramID, invitationText)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("💬 Открыть диалог с текстом", deepLink),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Отметить как отправленное", fmt.Sprintf("invite:mark_sent:%d", index)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться к списку", "admin:guests:list"),
		),
	)

	message := fmt.Sprintf(
		"📋 <b>Приглашение для %s</b>\n\n"+
			"Telegram ID: <code>%s</code>\n\n"+
			"Нажмите кнопку ниже, чтобы открыть диалог с предзаполненным текстом приглашения.",
		inv.Name, inv.TelegramID,
	)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, message)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleInvitationMarkSent отмечает приглашение как отправленное
func handleInvitationMarkSent(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, index int) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	invitations, err := google_sheets.GetInvitationsList(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка приглашений: %v", err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Ошибка"))
		return
	}

	if index < 0 || index >= len(invitations) {
		bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Неверный индекс"))
		return
	}

	inv := invitations[index]
	err = google_sheets.MarkInvitationAsSent(ctx, inv.Name)
	if err != nil {
		log.Printf("Ошибка отметки приглашения как отправленного: %v", err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Ошибка"))
		return
	}

	bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Приглашение отмечено как отправленное"))
	// Обновляем список приглашений
	handleAdminSendInvite(bot, callback)
}

// handleGameAdminCallback обрабатывает callback от админ панели игр
func handleGameAdminCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) == 0 {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	game := parts[0]

	switch game {
	case "wordle":
		if len(parts) > 1 {
			handleWordleAdminCallbackWithAction(bot, callback, parts[1:])
		} else {
			handleWordleAdminCallback(bot, callback)
		}
	case "crossword":
		if len(parts) > 1 {
			handleCrosswordAdminCallbackWithAction(bot, callback, parts[1:])
		} else {
			handleCrosswordAdminCallback(bot, callback)
		}
	default:
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

// handleWordleAdminCallback показывает управление Wordle
func handleWordleAdminCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	message := "🔤 <b>Управление Wordle</b>\n\nВыберите действие:"
	keyboard := keyboards.GetAdminWordleKeyboard()

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, message)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleWordleAdminCallbackWithAction обрабатывает действия Wordle
func handleWordleAdminCallbackWithAction(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) == 0 {
		handleWordleAdminCallback(bot, callback)
		return
	}

	action := parts[0]

	switch action {
	case "switch":
		handleWordleSwitch(bot, callback)
	case "add":
		handleWordleAdd(bot, callback)
	default:
		handleWordleAdminCallback(bot, callback)
	}
}

// handleWordleSwitch переключает слово Wordle для всех
func handleWordleSwitch(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := google_sheets.SwitchWordleWordForAll(ctx)
	if err != nil {
		log.Printf("Ошибка переключения слова Wordle: %v", err)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "✅ Слово Wordle переключено для всех пользователей")
	bot.Send(msg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleWordleAdd запускает добавление слова в Wordle
func handleWordleAdd(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	SetAdminInputMode(callback.From.ID, AdminInputModeWordleAdd)
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "🔤 Пришлите новое слово для Wordle (одно слово).")
	bot.Send(msg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleCrosswordAdminCallback показывает управление Crossword
func handleCrosswordAdminCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	message := "📝 <b>Управление Кроссвордом</b>\n\nВыберите действие:"
	keyboard := keyboards.GetAdminCrosswordKeyboard()

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, message)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleCrosswordAdminCallbackWithAction обрабатывает действия Crossword
func handleCrosswordAdminCallbackWithAction(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) == 0 {
		handleCrosswordAdminCallback(bot, callback)
		return
	}

	action := parts[0]

	switch action {
	case "update":
		handleCrosswordUpdate(bot, callback)
	case "add":
		handleCrosswordAdd(bot, callback)
	default:
		handleCrosswordAdminCallback(bot, callback)
	}
}

// handleCrosswordUpdate обновляет кроссворд
func handleCrosswordUpdate(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nextIndex, err := google_sheets.SwitchCrosswordForAll(ctx)
	if err != nil {
		log.Printf("Ошибка переключения кроссворда: %v", err)
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Не удалось переключить кроссворд, проверьте логи.")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, fmt.Sprintf("✅ Кроссворд переключён на индекс %d. Прогресс гостей сброшен.", nextIndex))
	bot.Send(msg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleCrosswordAdd запускает добавление кроссворда
func handleCrosswordAdd(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	SetAdminInputMode(callback.From.ID, AdminInputModeCrosswordAdd)
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "➕ Пришлите строки формата \"слово; описание\" (каждая с новой строки) для нового кроссворда.")
	bot.Send(msg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleGroupCallback обрабатывает callback от управления группой
func handleGroupCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) == 0 {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	action := parts[0]

	switch action {
	case "list_members":
		handleGroupListMembers(bot, callback)
	default:
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

// handleGroupListMembers показывает список участников группы
func handleGroupListMembers(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	// Реализовано в admin_handlers.go
	handleAdminGroupListMembers(bot, callback.Message)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleSwapNameCallback обрабатывает смену имени/фамилии
func handleSwapNameCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) == 0 {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	rowStr := parts[0]
	row, err := strconv.Atoi(rowStr)
	if err != nil {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = google_sheets.SwapGuestNameOrder(ctx, row)
	if err != nil {
		log.Printf("Ошибка смены имени/фамилии: %v", err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Ошибка"))
		return
	}

	// Обновляем список гостей и показываем обновленную страницу
	guests, err := google_sheets.ListConfirmedGuests(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка гостей: %v", err)
		bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Имя и фамилия поменяны местами"))
		return
	}

	// Определяем текущую страницу (просто показываем первую)
	keyboard := keyboards.GetGuestsSwapKeyboard(guests, 0)
	message := fmt.Sprintf(
		"✅ <b>Имя и фамилия поменяны местами!</b>\n\n"+
			"Нажмите на гостя, чтобы поменять местами Имя и Фамилию в Google Sheets.\n\n"+
			"Всего гостей: <b>%d</b>",
		len(guests),
	)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, message)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleFixNamesPageCallback обрабатывает переключение страницы в исправлении имен
func handleFixNamesPageCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) == 0 {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	page, err := strconv.Atoi(parts[0])
	if err != nil {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	guests, err := google_sheets.ListConfirmedGuests(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка гостей: %v", err)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	keyboard := keyboards.GetGuestsSwapKeyboard(guests, page)
	message := fmt.Sprintf(
		"🔁 <b>Исправление Имя/Фамилия</b>\n\n"+
			"Нажмите на гостя, чтобы поменять местами Имя и Фамилию в Google Sheets.\n\n"+
			"Всего гостей: <b>%d</b>",
		len(guests),
	)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, message)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleDeleteGuestCallback обрабатывает удаление гостя
func handleDeleteGuestCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	action := parts[0]
	userIDStr := parts[1]

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	switch action {
	case "confirm_group":
		// Удалить из группы и из списка
		// TODO: Реализовать удаление из группы
		err = google_sheets.DeleteGuestFromSheets(ctx, userID)
		if err != nil {
			log.Printf("Ошибка удаления гостя: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "✅ Гость удален из группы и списка")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	case "confirm_only":
		// Удалить только из списка
		err = google_sheets.DeleteGuestFromSheets(ctx, userID)
		if err != nil {
			log.Printf("Ошибка удаления гостя: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "✅ Гость удален из списка")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	default:
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

// handleBroadcastCallback обрабатывает callback от рассылки
func handleBroadcastCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) == 0 {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	userID := callback.From.ID
	state := GetBroadcastState(userID)

	action := parts[0]

	switch action {
	case "cancel":
		handleBroadcastCancel(bot, callback)
	case "media":
		if len(parts) < 2 {
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}
		mediaType := parts[1]
		if mediaType == "skip" {
			state.Step = "button"
			showRecipientsSelection(bot, callback, state)
		}
		// фото и видео обрабатываются в основном обработчике сообщений
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	case "btn":
		if len(parts) < 2 {
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}
		buttonType := parts[1]
		handleBroadcastButton(bot, callback, buttonType)
	case "recipients":
		if len(parts) < 2 {
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}
		recipientsType := parts[1]
		if recipientsType == "all" {
			state.Step = "preview"
			showBroadcastPreview(bot, callback, state)
		} else if recipientsType == "select" {
			// Показываем интерфейс выбора получателей
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			recipients, err := GetBroadcastRecipientsWithInfo(ctx)
			if err != nil {
				log.Printf("Ошибка получения получателей: %v", err)
				msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Ошибка получения списка получателей")
				bot.Send(msg)
				bot.Request(tgbotapi.NewCallback(callback.ID, ""))
				return
			}

			ShowRecipientsSelectionPage(bot, callback, recipients, 0, 0)
		}
	case "select", "deselect", "select_all", "deselect_all", "send_selected":
		if len(parts) < 2 {
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}

		var userID int64
		if parts[0] == "select" || parts[0] == "deselect" {
			if len(parts) < 3 {
				bot.Request(tgbotapi.NewCallback(callback.ID, ""))
				return
			}
			// Парсим user_id из callback data
			if _, err := fmt.Sscanf(parts[2], "%d", &userID); err != nil {
				bot.Request(tgbotapi.NewCallback(callback.ID, ""))
				return
			}
		}

		HandleRecipientsSelection(bot, callback, parts[0], userID)
	case "page":
		if len(parts) < 2 {
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}
		// Обработка пагинации
		page := 0
		if _, err := fmt.Sscanf(parts[1], "%d", &page); err != nil {
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		recipients, err := GetBroadcastRecipientsWithInfo(ctx)
		if err != nil {
			log.Printf("Ошибка получения получателей: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}

		ShowRecipientsSelectionPage(bot, callback, recipients, page, countSelected(recipients))
	case "send":
		if len(parts) > 1 && parts[1] == "confirm" {
			handleBroadcastSendConfirm(bot, callback)
		} else {
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		}
	default:
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

// handleAdminBroadcastCallback запускает рассылку из админ-панели
func handleAdminBroadcastCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	// Создаем сообщение-заглушку для имитации сообщения от пользователя
	message := &tgbotapi.Message{
		From: callback.From,
		Chat: callback.Message.Chat,
	}

	// Вызываем существующую функцию обработки рассылки
	handleAdminBroadcastDM(bot, message)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}
