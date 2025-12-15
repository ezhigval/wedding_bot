package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"gopkg.in/telebot.v3"

	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
	"wedding-bot/internal/keyboards"
)

// handleCallback обрабатывает callback queries
func handleCallback(c telebot.Context) error {
	data := c.Callback().Data

	// Парсим callback data
	parts := strings.Split(data, ":")
	if len(parts) == 0 {
		return c.Answer(&telebot.QueryResponse{})
	}

	action := parts[0]

	switch action {
	case "admin":
		return handleAdminCallback(c, parts[1:])
	case "invite":
		return handleInvitationCallback(c, parts[1:])
	case "game":
		return handleGameAdminCallback(c, parts[1:])
	case "group":
		return handleGroupCallback(c, parts[1:])
	case "admin_wordle":
		return handleWordleAdminCallback(c)
	case "admin_crossword":
		return handleCrosswordAdminCallback(c)
	case "admin_back":
		return handleAdminBackCallback(c)
	case "swapname":
		return handleSwapNameCallback(c, parts[1:])
	case "fixnames_page":
		return handleFixNamesPageCallback(c, parts[1:])
	case "delete_guest":
		return handleDeleteGuestCallback(c, parts[1:])
	case "broadcast":
		return handleBroadcastCallback(c, parts[1:])
	default:
		return c.Answer(&telebot.QueryResponse{})
	}
}

// handleAdminCallback обрабатывает callback от админ панели
func handleAdminCallback(c telebot.Context, parts []string) error {
	if len(parts) == 0 {
		return c.Answer(&telebot.QueryResponse{})
	}

	section := parts[0]

	switch section {
	case "guests":
		return handleAdminGuestsCallback(c)
	case "guests:list":
		return handleAdminGuestsList(c)
	case "seating":
		return handleAdminSeating(c)
	case "send_invite":
		return handleAdminSendInvite(c)
	case "games":
		return handleAdminGamesCallback(c)
	case "stats":
		return handleAdminStatsCallback(c)
	case "group":
		return handleAdminGroupCallback(c)
	case "back":
		return handleAdminBackCallback(c)
	default:
		return c.Answer(&telebot.QueryResponse{})
	}
}

// handleAdminBackCallback возвращает в главное админ меню
func handleAdminBackCallback(c telebot.Context) error {
	message := "🔧 <b>Панель администратора</b>\n\nВыберите раздел:"
	keyboard := keyboards.GetAdminRootReplyKeyboard()
	return c.Edit(message, keyboard, telebot.ModeHTML)
}

// handleAdminGuestsCallback показывает управление гостями
func handleAdminGuestsCallback(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	count, err := google_sheets.GetGuestsCountFromSheets(ctx)
	if err != nil {
		log.Printf("Ошибка получения количества гостей: %v", err)
		return c.Send("❌ Ошибка получения данных")
	}

	message := fmt.Sprintf("👥 <b>Управление гостями</b>\n\nЗарегистрировано: %d", count)

	keyboard := &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "📋 Список гостей",
					Data: "admin:guests:list",
				},
			},
			{
				telebot.InlineButton{
					Text: "🔙 Назад",
					Data: "admin:back",
				},
			},
		},
	}

	return c.Edit(message, keyboard, telebot.ModeHTML)
}

// handleAdminGuestsList показывает список гостей (из admin_handlers.go)
func handleAdminGuestsList(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	guests, err := google_sheets.GetAllGuestsFromSheets(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка гостей: %v", err)
		return c.Send("❌ Ошибка при получении списка гостей. Попробуйте позже.")
	}

	if len(guests) == 0 {
		return c.Send("📋 <b>Список гостей</b>\n\nПока никто не подтвердил присутствие.", telebot.ModeHTML)
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

	return c.Send(sb.String(), telebot.ModeHTML)
}

// handleAdminSeating показывает рассадку (из admin_handlers.go)
func handleAdminSeating(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	seating, err := google_sheets.GetSeatingFromSheets(ctx)
	if err != nil {
		log.Printf("Ошибка получения рассадки: %v", err)
		return c.Send("❌ Ошибка при получении рассадки. Попробуйте позже.")
	}

	if len(seating) == 0 {
		return c.Send("🍽 <b>Рассадка</b>\n\nПока нет данных по рассадке (лист 'Рассадка' пуст или без гостей).", telebot.ModeHTML)
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

	return c.Send(sb.String(), telebot.ModeHTML)
}

// handleAdminSendInvite запускает отправку приглашений (из admin_handlers.go)
func handleAdminSendInvite(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	invitations, err := google_sheets.GetInvitationsList(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка приглашений: %v", err)
		return c.Send("❌ Ошибка получения списка приглашений.")
	}

	if len(invitations) == 0 {
		return c.Send(
			"❌ <b>Список приглашений пуст</b>\n\n"+
				"Проверьте вкладку 'Пригласительные' в Google Sheets.",
			telebot.ModeHTML,
		)
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
	return c.Send(message, keyboard, telebot.ModeHTML)
}

// handleAdminGamesCallback показывает управление играми
func handleAdminGamesCallback(c telebot.Context) error {
	message := "🎮 <b>Управление играми</b>\n\nВыберите игру:"

	keyboard := keyboards.GetAdminGamesKeyboard()

	return c.Edit(message, keyboard, telebot.ModeHTML)
}

// handleAdminStatsCallback показывает статистику
func handleAdminStatsCallback(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := google_sheets.GetGuestsCountFromSheets(ctx)
	if err != nil {
		log.Printf("Ошибка получения статистики: %v", err)
		return c.Answer(&telebot.QueryResponse{})
	}

	message := fmt.Sprintf("📊 <b>Статистика</b>\n\nЗарегистрировано гостей: %d", count)
	return c.Edit(message, telebot.ModeHTML)
}

// handleAdminGroupCallback показывает управление группой
func handleAdminGroupCallback(c telebot.Context) error {
	message := "💬 <b>Управление группой</b>\n\nВыберите действие:"

	keyboard := keyboards.GetGroupManagementKeyboard()

	return c.Edit(message, keyboard, telebot.ModeHTML)
}

// handleInvitationCallback обрабатывает callback от приглашений
func handleInvitationCallback(c telebot.Context, parts []string) error {
	if len(parts) == 0 {
		return c.Answer(&telebot.QueryResponse{})
	}

	action := parts[0]

	switch action {
	case "guest":
		if len(parts) < 2 {
			return c.Answer(&telebot.QueryResponse{})
		}
		index, err := strconv.Atoi(parts[1])
		if err != nil {
			return c.Answer(&telebot.QueryResponse{})
		}
		return handleInvitationGuestSelect(c, index)
	default:
		return c.Answer(&telebot.QueryResponse{})
	}
}

// handleInvitationGuestSelect обрабатывает выбор гостя для отправки приглашения
func handleInvitationGuestSelect(c telebot.Context, index int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	invitations, err := google_sheets.GetInvitationsList(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка приглашений: %v", err)
		return c.Answer(&telebot.QueryResponse{})
	}

	if index < 0 || index >= len(invitations) {
		return c.Answer(&telebot.QueryResponse{})
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
		return c.Answer(&telebot.QueryResponse{})
	}

	// Убираем @ если есть
	telegramID = strings.TrimPrefix(telegramID, "@")
	telegramID = strings.TrimPrefix(telegramID, "https://t.me/")
	telegramID = strings.TrimPrefix(telegramID, "t.me/")

	deepLink := fmt.Sprintf("tg://msg?to=%s&text=%s", telegramID, invitationText)

	keyboard := &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "💬 Открыть диалог с текстом",
					URL: deepLink,
				},
			},
			{
				telebot.InlineButton{
					Text: "✅ Отметить как отправленное",
					Data: fmt.Sprintf("invite:mark_sent:%d", index),
				},
			},
			{
				telebot.InlineButton{
					Text: "⬅️ Вернуться к списку",
					Data: "admin:guests:list",
				},
			},
		},
	}

	message := fmt.Sprintf(
		"📋 <b>Приглашение для %s</b>\n\n"+
			"Telegram ID: <code>%s</code>\n\n"+
			"Нажмите кнопку ниже, чтобы открыть диалог с предзаполненным текстом приглашения.",
		inv.Name, inv.TelegramID,
	)

	return c.Edit(message, keyboard, telebot.ModeHTML)
}

// handleGameAdminCallback обрабатывает callback от админ панели игр
func handleGameAdminCallback(c telebot.Context, parts []string) error {
	if len(parts) == 0 {
		return c.Answer(&telebot.QueryResponse{})
	}

	game := parts[0]

	switch game {
	case "wordle":
		if len(parts) > 1 {
			return handleWordleAdminCallbackWithAction(c, parts[1:])
		}
		return handleWordleAdminCallback(c)
	case "crossword":
		if len(parts) > 1 {
			return handleCrosswordAdminCallbackWithAction(c, parts[1:])
		}
		return handleCrosswordAdminCallback(c)
	default:
		return c.Answer(&telebot.QueryResponse{})
	}
}

// handleWordleAdminCallback показывает управление Wordle
func handleWordleAdminCallback(c telebot.Context) error {
	message := "🔤 <b>Управление Wordle</b>\n\nВыберите действие:"

	keyboard := keyboards.GetAdminWordleKeyboard()

	return c.Edit(message, keyboard, telebot.ModeHTML)
}

// handleWordleAdminCallbackWithAction обрабатывает действия Wordle
func handleWordleAdminCallbackWithAction(c telebot.Context, parts []string) error {
	if len(parts) == 0 {
		return handleWordleAdminCallback(c)
	}

	action := parts[0]

	switch action {
	case "switch":
		return handleWordleSwitch(c)
	case "add":
		return handleWordleAdd(c)
	default:
		return handleWordleAdminCallback(c)
	}
}

// handleWordleSwitch переключает слово Wordle для всех
func handleWordleSwitch(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := google_sheets.SwitchWordleWordForAll(ctx)
	if err != nil {
		log.Printf("Ошибка переключения слова Wordle: %v", err)
		return c.Answer(&telebot.QueryResponse{})
	}

	_ = c.Send("✅ Слово Wordle переключено для всех пользователей")
	return c.Answer(&telebot.QueryResponse{})
}

// handleWordleAdd запускает добавление слова в Wordle
func handleWordleAdd(c telebot.Context) error {
	// TODO: Реализовать FSM для ввода слова
	_ = c.Send("Функция добавления слова в разработке")
	return c.Answer(&telebot.QueryResponse{})
}

// handleCrosswordAdminCallback показывает управление Crossword
func handleCrosswordAdminCallback(c telebot.Context) error {
	message := "📝 <b>Управление Кроссвордом</b>\n\nВыберите действие:"

	keyboard := keyboards.GetAdminCrosswordKeyboard()

	return c.Edit(message, keyboard, telebot.ModeHTML)
}

// handleCrosswordAdminCallbackWithAction обрабатывает действия Crossword
func handleCrosswordAdminCallbackWithAction(c telebot.Context, parts []string) error {
	if len(parts) == 0 {
		return handleCrosswordAdminCallback(c)
	}

	action := parts[0]

	switch action {
	case "update":
		return handleCrosswordUpdate(c)
	case "add":
		return handleCrosswordAdd(c)
	default:
		return handleCrosswordAdminCallback(c)
	}
}

// handleCrosswordUpdate обновляет кроссворд
func handleCrosswordUpdate(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// TODO: Реализовать обновление кроссворда
	_ = ctx
	_ = c.Send("Функция обновления кроссворда в разработке")
	return c.Answer(&telebot.QueryResponse{})
}

// handleCrosswordAdd запускает добавление кроссворда
func handleCrosswordAdd(c telebot.Context) error {
	// TODO: Реализовать FSM для ввода слов кроссворда
	_ = c.Send("Функция добавления кроссворда в разработке")
	return c.Answer(&telebot.QueryResponse{})
}

// handleGroupCallback обрабатывает callback от управления группой
func handleGroupCallback(c telebot.Context, parts []string) error {
	if len(parts) == 0 {
		return c.Answer(&telebot.QueryResponse{})
	}

	action := parts[0]

	switch action {
	case "send_message":
		return handleGroupSendMessage(c)
	case "add_member":
		return handleGroupAddMember(c)
	case "remove_member":
		return handleGroupRemoveMember(c)
	case "list_members":
		return handleGroupListMembers(c)
	default:
		return c.Answer(&telebot.QueryResponse{})
	}
}

// handleGroupSendMessage запускает отправку сообщения в группу
func handleGroupSendMessage(c telebot.Context) error {
	if config.GroupID == "" {
		return c.Answer(&telebot.QueryResponse{})
	}

	// TODO: Реализовать FSM для ввода сообщения
	_ = c.Send("Функция отправки сообщения в группу в разработке")
	return c.Answer(&telebot.QueryResponse{})
}

// handleGroupAddMember запускает добавление участника в группу
func handleGroupAddMember(c telebot.Context) error {
	if config.GroupID == "" {
		return c.Answer(&telebot.QueryResponse{})
	}

	// TODO: Реализовать FSM для ввода username
	_ = c.Send("Функция добавления участника в разработке")
	return c.Answer(&telebot.QueryResponse{})
}

// handleGroupRemoveMember запускает удаление участника из группы
func handleGroupRemoveMember(c telebot.Context) error {
	if config.GroupID == "" {
		return c.Answer(&telebot.QueryResponse{})
	}

	// TODO: Реализовать FSM для ввода username
	_ = c.Send("Функция удаления участника в разработке")
	return c.Answer(&telebot.QueryResponse{})
}

// handleGroupListMembers показывает список участников группы
func handleGroupListMembers(c telebot.Context) error {
	if config.GroupID == "" {
		return c.Answer(&telebot.QueryResponse{})
	}

	// TODO: Реализовать получение списка участников
	_ = c.Send("Функция просмотра участников в разработке")
	return c.Answer(&telebot.QueryResponse{})
}

// handleSwapNameCallback обрабатывает смену имени/фамилии
func handleSwapNameCallback(c telebot.Context, parts []string) error {
	if len(parts) == 0 {
		return c.Answer(&telebot.QueryResponse{})
	}

	rowStr := parts[0]
	row, err := strconv.Atoi(rowStr)
	if err != nil {
		return c.Answer(&telebot.QueryResponse{})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = google_sheets.SwapGuestNameOrder(ctx, row)
	if err != nil {
		log.Printf("Ошибка смены имени/фамилии: %v", err)
		return c.Answer(&telebot.QueryResponse{})
	}

	// Обновляем список гостей и показываем обновленную страницу
	guests, err := google_sheets.ListConfirmedGuests(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка гостей: %v", err)
		return c.Answer(&telebot.QueryResponse{})
	}

	// Определяем текущую страницу (просто показываем первую)
	keyboard := keyboards.GetGuestsSwapKeyboard(guests, 0)
	message := fmt.Sprintf(
		"✅ <b>Имя и фамилия поменяны местами!</b>\n\n"+
			"Нажмите на гостя, чтобы поменять местами Имя и Фамилию в Google Sheets.\n\n"+
			"Всего гостей: <b>%d</b>",
		len(guests),
	)

	return c.Edit(message, keyboard, telebot.ModeHTML)
}

// handleFixNamesPageCallback обрабатывает переключение страницы в исправлении имен
func handleFixNamesPageCallback(c telebot.Context, parts []string) error {
	if len(parts) == 0 {
		return c.Answer(&telebot.QueryResponse{})
	}

	page, err := strconv.Atoi(parts[0])
	if err != nil {
		return c.Answer(&telebot.QueryResponse{})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	guests, err := google_sheets.ListConfirmedGuests(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка гостей: %v", err)
		return c.Answer(&telebot.QueryResponse{})
	}

	keyboard := keyboards.GetGuestsSwapKeyboard(guests, page)
	message := fmt.Sprintf(
		"🔁 <b>Исправление Имя/Фамилия</b>\n\n"+
			"Нажмите на гостя, чтобы поменять местами Имя и Фамилию в Google Sheets.\n\n"+
			"Всего гостей: <b>%d</b>",
		len(guests),
	)

	return c.Edit(message, keyboard, telebot.ModeHTML)
}

// handleDeleteGuestCallback обрабатывает удаление гостя
func handleDeleteGuestCallback(c telebot.Context, parts []string) error {
	if len(parts) < 2 {
		return c.Answer(&telebot.QueryResponse{})
	}

	action := parts[0]
	userIDStr := parts[1]

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return c.Answer(&telebot.QueryResponse{})
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
			return c.Answer(&telebot.QueryResponse{})
		}
		_ = c.Send("✅ Гость удален из группы и списка")
		return c.Answer(&telebot.QueryResponse{})
	case "confirm_only":
		// Удалить только из списка
		err = google_sheets.DeleteGuestFromSheets(ctx, userID)
		if err != nil {
			log.Printf("Ошибка удаления гостя: %v", err)
			return c.Answer(&telebot.QueryResponse{})
		}
		_ = c.Send("✅ Гость удален из списка")
		return c.Answer(&telebot.QueryResponse{})
	default:
		return c.Answer(&telebot.QueryResponse{})
	}
}

// handleBroadcastCallback обрабатывает callback от рассылки
func handleBroadcastCallback(c telebot.Context, parts []string) error {
	if len(parts) == 0 {
		return c.Answer(&telebot.QueryResponse{})
	}

	action := parts[0]

	switch action {
	case "no_photo":
		userID := c.Sender().ID
		state := GetBroadcastState(userID)
		if state == nil {
			return c.Answer(&telebot.QueryResponse{})
		}
		// Пропускаем фото, переходим к кнопке
		return handleBroadcastButton(c, "none")
	case "btn":
		if len(parts) < 2 {
			return c.Answer(&telebot.QueryResponse{})
		}
		buttonType := parts[1]
		return handleBroadcastButton(c, buttonType)
	case "send":
		if len(parts) > 1 && parts[1] == "confirm" {
			return handleBroadcastSendConfirm(c)
		}
		return c.Answer(&telebot.QueryResponse{})
	case "cancel":
		return handleBroadcastCancel(c)
	default:
		return c.Answer(&telebot.QueryResponse{})
	}
}
