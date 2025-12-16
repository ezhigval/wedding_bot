package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"wedding-bot/internal/config"
	"wedding-bot/internal/keyboards"
)

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

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "broadcast:cancel"),
		),
	)

	msgText := fmt.Sprintf(
		"📨 <b>Рассылка в личные сообщения</b>\n\n"+
			"Получателей (по базе гостей): <b>%d</b>\n\n"+
			"1️⃣ Отправьте текст сообщения, которое получат гости.",
		total,
	)

	// Устанавливаем начальное состояние
	SetBroadcastState(userID, &BroadcastState{
		Text: "",
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
	if state == nil {
		// Если состояния нет, создаем новое
		state = &BroadcastState{}
		SetBroadcastState(userID, state)
	}

	state.Text = text

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➡️ Без картинки", "broadcast:no_photo"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "broadcast:cancel"),
		),
	)

	msgText := "📨 <b>Текст сообщения сохранен</b>\n\n" +
		"Хотите добавить картинку? Или продолжить без картинки?"

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &keyboard
	bot.Send(msg)
}

// handleBroadcastPhoto обрабатывает фото для рассылки
func handleBroadcastPhoto(bot *tgbotapi.BotAPI, message *tgbotapi.Message, photoID string) {
	userID := message.From.ID

	state := GetBroadcastState(userID)
	if state == nil {
		state = &BroadcastState{}
		SetBroadcastState(userID, state)
	}

	state.PhotoID = photoID

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔘 Без кнопки", "broadcast:btn:none"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💒 Открыть мини-эпп", "broadcast:btn:miniapp"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Открыть общий чат", "broadcast:btn:group"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить свою кнопку", "broadcast:btn:custom"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "broadcast:cancel"),
		),
	)

	msgText := "📨 <b>Картинка добавлена</b>\n\n" +
		"Хотите добавить кнопку к сообщению?"

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = &keyboard
	bot.Send(msg)
}

// handleBroadcastButton обрабатывает выбор кнопки для рассылки
func handleBroadcastButton(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, buttonType string) {
	userID := callback.From.ID

	state := GetBroadcastState(userID)
	if state == nil {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Ошибка: состояние рассылки потеряно")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	switch buttonType {
	case "none":
		state.ButtonText = ""
		state.ButtonURL = ""
		showBroadcastPreview(bot, callback, state)
	case "miniapp":
		state.ButtonText = "💒 Открыть приглашение"
		state.ButtonURL = config.WebappURL // WebApp URL
		showBroadcastPreview(bot, callback, state)
	case "group":
		state.ButtonText = "💬 Общий чат"
		state.ButtonURL = config.GroupLink // URL из config
		showBroadcastPreview(bot, callback, state)
	case "custom":
		msgText := "📨 <b>Добавить свою кнопку</b>\n\n" +
			"Отправьте текст кнопки в формате:\n" +
			"<code>Текст кнопки|URL</code>\n\n" +
			"Пример: <code>Открыть сайт|https://example.com</code>"
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
		msg.ParseMode = tgbotapi.ModeHTML
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	default:
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Неизвестный тип кнопки")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

// showBroadcastPreview показывает превью рассылки
func showBroadcastPreview(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, state *BroadcastState) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	recipients, err := GetBroadcastRecipients(ctx)
	if err != nil {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Ошибка получения получателей")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	total := len(recipients)

	// Создаем клавиатуру для превью
	var previewKeyboard *tgbotapi.InlineKeyboardMarkup
	if state.ButtonText != "" {
		previewKeyboard = &tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
				{
					tgbotapi.NewInlineKeyboardButtonURL(state.ButtonText, state.ButtonURL),
				},
			},
		}
	}

	// Отправляем превью
	if state.PhotoID != "" {
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
			tgbotapi.NewInlineKeyboardButtonData("✅ Отправить всем гостям", "broadcast:send:confirm"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "broadcast:cancel"),
		),
	)

	msgText := fmt.Sprintf(
		"📨 <b>Проверьте сообщение выше.</b>\n\n"+
			"Оно будет отправлено в ЛС всем гостям из базы, у кого есть user_id.\n"+
			"Планируется отправка: <b>%d</b> получателям.\n\n"+
			"Если всё верно — нажмите «Отправить всем гостям».",
		total,
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

	recipients, err := GetBroadcastRecipients(ctx)
	if err != nil {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Ошибка получения получателей")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	total := len(recipients)
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
	sent, failed := SendBroadcast(botInstance, state, recipients)

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
