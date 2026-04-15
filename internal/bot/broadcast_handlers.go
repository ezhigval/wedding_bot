package bot

import (
	"context"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"wedding-bot/internal/config"
	"wedding-bot/internal/keyboards"
)

func broadcastCancelInlineKeyboard() *tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "broadcast:cancel"),
		),
	)

	return &keyboard
}

// handleAdminBroadcastDM запускает рассылку в ЛС
func handleAdminBroadcastDM(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	userID := message.From.ID

	// Очищаем предыдущее состояние
	ClearBroadcastState(userID)

	// Получаем список получателей
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	recipients, err := GetBroadcastRecipients(ctx)
	if err != nil {
		log.Printf("Ошибка получения получателей: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка получения списка получателей")
		bot.Send(msg)
		return
	}

	total := len(recipients)

	// Проверяем, включен ли админ в список получателей
	adminInRecipients := false
	for _, recipientID := range recipients {
		if recipientID == userID {
			adminInRecipients = true
			break
		}
	}

	var adminInfo string
	if adminInRecipients {
		adminInfo = "\n✅ Вы также получите это сообщение"
	} else {
		adminInfo = "\nℹ️ Вы не получите это сообщение (нет в базе гостей)"
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧪 Тест (только себе)", "broadcast:test_self"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "broadcast:cancel"),
		),
	)

	msgText := fmt.Sprintf(
		"📨 <b>Создание рассылки</b>\n\n"+
			"Получателей (по базе гостей): <b>%d</b>"+
			adminInfo+
			"\n\n"+
			"📝 <b>Шаг 1/5: Текст сообщения</b>\n\n"+
			"Отправьте текст сообщения, которое получат гости.\n"+
			"Можно сразу прислать фото или видео с подписью.",
		total,
	)

	// Устанавливаем начальное состояние
	SetBroadcastState(userID, &BroadcastState{
		Text:       "",
		Step:       "text",
		Recipients: []int64{}, // пустой = всем
	})

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &keyboard
	bot.Send(msg)
}

// handleBroadcastText обрабатывает текст для рассылки
func handleBroadcastText(bot *tgbotapi.BotAPI, message *tgbotapi.Message, text string) {
	userID := message.From.ID

	state := GetBroadcastState(userID)
	if state == nil || state.Step != "text" {
		// Если состояния нет или не тот шаг, игнорируем
		return
	}

	state.Text = text

	// Проверяем, есть ли медиа в этом сообщении
	if len(message.Photo) > 0 {
		// Автоматически сохраняем фото
		photoID := message.Photo[len(message.Photo)-1].FileID
		state.PhotoID = photoID
		state.VideoID = "" // очищаем видео если было

		// Сразу переходим к шагу кнопок
		state.Step = "button"
		showBroadcastButtonSelection(bot, message, state)
		return
	}

	if message.Video != nil {
		// Автоматически сохраняем видео
		state.VideoID = message.Video.FileID
		state.PhotoID = "" // очищаем фото если было

		// Сразу переходим к шагу кнопок
		state.Step = "button"
		showBroadcastButtonSelection(bot, message, state)
		return
	}

	// Если нет медиа, показываем выбор медиа
	state.Step = "media"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🖼️ Добавить фото", "broadcast:media:photo"),
			tgbotapi.NewInlineKeyboardButtonData("🎥 Добавить видео", "broadcast:media:video"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➡️ Пропустить", "broadcast:media:skip"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "broadcast:cancel"),
		),
	)

	msgText := "📨 <b>Шаг 2/5: Медиа-файлы</b>\n\n" +
		"Текст сообщения сохранен!\n\n" +
		"Хотите добавить фото или видео к сообщению?"

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &keyboard
	bot.Send(msg)
}

// showBroadcastButtonSelection показывает выбор кнопок для рассылки
func showBroadcastButtonSelection(bot *tgbotapi.BotAPI, message *tgbotapi.Message, state *BroadcastState) {
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("💒 Добавить мини-эпп", "broadcast:btn:miniapp"),
			tgbotapi.NewInlineKeyboardButtonData("💬 Добавить общий чат", "broadcast:btn:group"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить свою кнопку", "broadcast:btn:custom"),
		},
	}

	if len(state.Buttons) > 0 {
		rows = append(rows,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➡️ К получателям", "broadcast:btn:done"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🗑 Очистить кнопки", "broadcast:btn:clear"),
			),
		)
	} else {
		rows = append(rows,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔘 Продолжить без кнопок", "broadcast:btn:none"),
			),
		)
	}

	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "broadcast:cancel"),
		),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	var statusText string
	if state.PhotoID != "" {
		statusText = "Фото добавлено."
	} else if state.VideoID != "" {
		statusText = "Видео добавлено."
	} else {
		statusText = "Текст сообщения сохранён."
	}

	buttonsSummary := "Пока без кнопок."
	if len(state.Buttons) > 0 {
		lines := make([]string, 0, len(state.Buttons))
		for idx, button := range state.Buttons {
			lines = append(lines, fmt.Sprintf("%d. <b>%s</b>", idx+1, html.EscapeString(button.Text)))
		}
		buttonsSummary = "Текущие кнопки:\n" + strings.Join(lines, "\n")
	}

	msgText := "📨 <b>Шаг 3/5: Кнопки и ссылки</b>\n\n" +
		statusText + "\n\n" +
		"Здесь можно собрать одну или несколько кнопок для сообщения.\n\n" +
		buttonsSummary

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &keyboard
	bot.Send(msg)
}

// handleBroadcastPhoto обрабатывает фото для рассылки
func handleBroadcastPhoto(bot *tgbotapi.BotAPI, message *tgbotapi.Message, photoID string) {
	userID := message.From.ID

	state := GetBroadcastState(userID)
	if state == nil || state.Step != "media" {
		return
	}

	state.PhotoID = photoID
	state.VideoID = "" // очищаем видео если было
	state.Step = "button"

	showBroadcastButtonSelection(bot, message, state)
}

// handleBroadcastVideo обрабатывает видео для рассылки
func handleBroadcastVideo(bot *tgbotapi.BotAPI, message *tgbotapi.Message, videoID string) {
	userID := message.From.ID

	state := GetBroadcastState(userID)
	if state == nil || state.Step != "media" {
		return
	}

	state.VideoID = videoID
	state.PhotoID = "" // очищаем фото если было
	state.Step = "button"

	showBroadcastButtonSelection(bot, message, state)
}

// handleBroadcastButton обрабатывает выбор кнопки для рассылки
func handleBroadcastButton(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, buttonType string) {
	userID := callback.From.ID

	state := GetBroadcastState(userID)
	if state == nil || state.Step != "button" {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Ошибка: состояние рассылки потеряно")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	switch buttonType {
	case "none":
		clearBroadcastButtons(state)
		state.Step = "recipients"
		showRecipientsSelection(bot, callback, state)
	case "miniapp":
		if err := validateBroadcastButtonURL(config.WebappURL); err != nil {
			msg := tgbotapi.NewMessage(callback.Message.Chat.ID, fmt.Sprintf("❌ %s", err.Error()))
			bot.Send(msg)
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}
		addBroadcastButton(state, "💒 Открыть приглашение", config.WebappURL)
		state.Step = "button"
		showBroadcastButtonSelection(bot, callback.Message, state)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	case "group":
		if err := validateBroadcastButtonURL(config.GroupLink); err != nil {
			msg := tgbotapi.NewMessage(callback.Message.Chat.ID, fmt.Sprintf("❌ %s", err.Error()))
			bot.Send(msg)
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}
		addBroadcastButton(state, "💬 Общий чат", config.GroupLink)
		state.Step = "button"
		showBroadcastButtonSelection(bot, callback.Message, state)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	case "custom":
		state.Step = "custom_button"
		promptCustomBroadcastButton(bot, callback.Message.Chat.ID)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	case "done":
		state.Step = "recipients"
		showRecipientsSelection(bot, callback, state)
	case "clear":
		clearBroadcastButtons(state)
		state.Step = "button"
		showBroadcastButtonSelection(bot, callback.Message, state)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	default:
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Неизвестный тип кнопки")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

// showRecipientsSelection показывает выбор получателей
func showRecipientsSelection(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, state *BroadcastState) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌐 Всем гостям", "broadcast:recipients:all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧪 Только себе", "broadcast:recipients:self"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Выбрать вручную", "broadcast:recipients:select"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "broadcast:cancel"),
		),
	)

	msgText := "📨 <b>Шаг 4/5: Выбор получателей</b>\n\n" +
		"Кому отправить сообщение?"

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &keyboard
	bot.Send(msg)

	// Если это реальный callback, отвечаем на него
	if callback.ID != "" {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

// showRecipientsSelectionFromMessage показывает выбор получателей из сообщения (без callback)
func showRecipientsSelectionFromMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, state *BroadcastState) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌐 Всем гостям", "broadcast:recipients:all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧪 Только себе", "broadcast:recipients:self"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Выбрать вручную", "broadcast:recipients:select"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "broadcast:cancel"),
		),
	)

	msgText := "📨 <b>Шаг 4/5: Выбор получателей</b>\n\n" +
		"Кому отправить сообщение?"

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &keyboard
	bot.Send(msg)
}

// showBroadcastPreview показывает превью рассылки
func showBroadcastPreview(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, state *BroadcastState) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	allRecipients, err := GetBroadcastRecipients(ctx)
	if err != nil {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Ошибка получения получателей")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	// Определяем количество получателей
	var total int
	var recipientText string
	if state.TestOnly {
		total = 1
		recipientText = "только вам (тестовый режим)"
	} else if len(state.Recipients) > 0 {
		total = len(uniqueRecipientIDs(state.Recipients))
		recipientText = fmt.Sprintf("%d выбранных получателей", total)
	} else {
		total = len(allRecipients)
		recipientText = fmt.Sprintf("%d гостей из базы", total)
	}

	// Создаем клавиатуру для превью
	previewKeyboard := broadcastReplyMarkup(state)

	// Отправляем превью
	if state.VideoID != "" {
		video := tgbotapi.NewVideo(callback.Message.Chat.ID, tgbotapi.FileID(state.VideoID))
		if state.Text != "" {
			video.Caption = state.Text
		}
		if previewKeyboard != nil {
			video.ReplyMarkup = previewKeyboard
		}
		bot.Send(video)
	} else if state.PhotoID != "" {
		photo := tgbotapi.NewPhoto(callback.Message.Chat.ID, tgbotapi.FileID(state.PhotoID))
		if state.Text != "" {
			photo.Caption = state.Text
		}
		if previewKeyboard != nil {
			photo.ReplyMarkup = previewKeyboard
		}
		bot.Send(photo)
	} else {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, state.Text)
		if previewKeyboard != nil {
			msg.ReplyMarkup = previewKeyboard
		}
		bot.Send(msg)
	}

	// Отправляем подтверждение
	confirmKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Отправить", "broadcast:send:confirm"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "broadcast:cancel"),
		),
	)

	msgText := fmt.Sprintf(
		"📨 <b>Шаг 5/5: Проверка сообщения</b>\n\n"+
			"Сообщение выше будет отправлено %s.\n\n"+
			"Если всё верно — нажмите «Отправить».",
		recipientText,
	)

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &confirmKeyboard
	bot.Send(msg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleBroadcastSendConfirm отправляет рассылку
func handleBroadcastSendConfirm(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID

	state := GetBroadcastState(userID)
	if state == nil {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Ошибка: состояние рассылки потеряно")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	if state.Text == "" {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Нет текста сообщения")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	allRecipients, err := GetBroadcastRecipients(ctx)
	if err != nil {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Ошибка получения получателей")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	// Определяем количество получателей
	var total int
	if state.TestOnly {
		state.Recipients = []int64{userID}
		total = 1
	} else if len(state.Recipients) > 0 {
		total = len(uniqueRecipientIDs(state.Recipients))
	} else {
		total = len(allRecipients)
	}

	if total == 0 {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "⚠️ В базе гостей пока нет ни одного user_id, рассылать некому.")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, fmt.Sprintf("🚀 Начинаю рассылку для <b>%d</b> получателей…", total))
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)

	// Получаем бота
	mu.RLock()
	botInstance := botInstance
	mu.RUnlock()

	if botInstance == nil {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Бот не инициализирован")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	// Отправляем рассылку
	sent, failed := SendBroadcast(botInstance, state, allRecipients)

	// Очищаем состояние
	ClearBroadcastState(userID)

	msgText := fmt.Sprintf(
		"✅ <b>Рассылка завершена.</b>\n\n"+
			"Успешно отправлено: <b>%d</b>\n"+
			"С ошибкой: <b>%d</b>",
		sent, failed,
	)

	msg = tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// handleBroadcastCancel отменяет рассылку и возвращает в админ-меню
func handleBroadcastCancel(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID

	// Очищаем состояние рассылки
	ClearBroadcastState(userID)

	// Возвращаемся в админ-меню
	msgText := "🔧 <b>Панель администратора</b>\n\nВыберите раздел:"
	keyboard := keyboards.GetAdminRootInlineKeyboard()

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, msgText)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

func promptCustomBroadcastButton(bot *tgbotapi.BotAPI, chatID int64) {
	msgText := "📨 <b>Добавить свою кнопку</b>\n\n" +
		"Отправьте одну кнопку в формате:\n" +
		"<code>Текст кнопки|URL</code>\n\n" +
		"Пример: <code>Открыть сайт|https://example.com</code>\n\n" +
		"После добавления можно будет прикрепить ещё кнопки или перейти к выбору получателей."

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = broadcastCancelInlineKeyboard()
	bot.Send(msg)
}

func validateBroadcastButtonURL(buttonURL string) error {
	cleanURL := strings.TrimSpace(buttonURL)
	if cleanURL == "" {
		return fmt.Errorf("ссылка для кнопки не настроена")
	}
	if !strings.HasPrefix(cleanURL, "http://") && !strings.HasPrefix(cleanURL, "https://") {
		return fmt.Errorf("ссылка для кнопки должна начинаться с http:// или https://")
	}

	return nil
}
