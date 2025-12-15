package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"gopkg.in/telebot.v3"

	"wedding-bot/internal/config"
	"wedding-bot/internal/keyboards"
)

// handleAdminBroadcastDM запускает рассылку в ЛС
func handleAdminBroadcastDM(c telebot.Context) error {
	userID := c.Sender().ID

	// Очищаем предыдущее состояние
	ClearBroadcastState(userID)

	// Получаем список получателей
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	recipients, err := GetBroadcastRecipients(ctx)
	if err != nil {
		log.Printf("Ошибка получения получателей: %v", err)
		return c.Send("❌ Ошибка получения списка получателей")
	}

	total := len(recipients)

	keyboard := &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "❌ Отмена",
					Data: "broadcast:cancel",
				},
			},
		},
	}

	message := fmt.Sprintf(
		"📨 <b>Рассылка в личные сообщения</b>\n\n"+
			"Получателей (по базе гостей): <b>%d</b>\n\n"+
			"1️⃣ Отправьте текст сообщения, которое получат гости.",
		total,
	)

	// Устанавливаем начальное состояние
	SetBroadcastState(userID, &BroadcastState{
		Text: "",
	})

	return c.Send(message, keyboard, telebot.ModeHTML)
}

// handleBroadcastText обрабатывает текст для рассылки
func handleBroadcastText(c telebot.Context, text string) error {
	userID := c.Sender().ID

	state := GetBroadcastState(userID)
	if state == nil {
		// Если состояния нет, создаем новое
		state = &BroadcastState{}
		SetBroadcastState(userID, state)
	}

	state.Text = text

	keyboard := &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "➡️ Без картинки",
					Data: "broadcast:no_photo",
				},
			},
			{
				telebot.InlineButton{
					Text: "❌ Отмена",
					Data: "admin:back",
				},
			},
		},
	}

	message := "📨 <b>Текст сообщения сохранен</b>\n\n" +
		"Хотите добавить картинку? Или продолжить без картинки?"

	return c.Send(message, keyboard, telebot.ModeHTML)
}

// handleBroadcastPhoto обрабатывает фото для рассылки
func handleBroadcastPhoto(c telebot.Context, photoID string) error {
	userID := c.Sender().ID

	state := GetBroadcastState(userID)
	if state == nil {
		state = &BroadcastState{}
		SetBroadcastState(userID, state)
	}

	state.PhotoID = photoID

	keyboard := &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "🔘 Без кнопки",
					Data: "broadcast:btn:none",
				},
			},
			{
				telebot.InlineButton{
					Text: "💒 Открыть мини-эпп",
					Data: "broadcast:btn:miniapp",
				},
			},
			{
				telebot.InlineButton{
					Text: "💬 Открыть общий чат",
					Data: "broadcast:btn:group",
				},
			},
			{
				telebot.InlineButton{
					Text: "➕ Добавить свою кнопку",
					Data: "broadcast:btn:custom",
				},
			},
			{
				telebot.InlineButton{
					Text: "❌ Отмена",
					Data: "admin:back",
				},
			},
		},
	}

	message := "📨 <b>Картинка добавлена</b>\n\n" +
		"Хотите добавить кнопку к сообщению?"

	return c.Send(message, keyboard, telebot.ModeHTML)
}

// handleBroadcastButton обрабатывает выбор кнопки для рассылки
func handleBroadcastButton(c telebot.Context, buttonType string) error {
	userID := c.Sender().ID

	state := GetBroadcastState(userID)
	if state == nil {
		return c.Send("❌ Ошибка: состояние рассылки потеряно")
	}

	switch buttonType {
	case "none":
		state.ButtonText = ""
		state.ButtonURL = ""
		return showBroadcastPreview(c, state)
	case "miniapp":
		state.ButtonText = "💒 Открыть приглашение"
		state.ButtonURL = config.WebappURL // WebApp URL
		return showBroadcastPreview(c, state)
	case "group":
		state.ButtonText = "💬 Общий чат"
		state.ButtonURL = config.GroupLink // URL из config
		return showBroadcastPreview(c, state)
	case "custom":
		message := "📨 <b>Добавить свою кнопку</b>\n\n" +
			"Отправьте текст кнопки в формате:\n" +
			"<code>Текст кнопки|URL</code>\n\n" +
			"Пример: <code>Открыть сайт|https://example.com</code>"
		return c.Send(message, telebot.ModeHTML)
	default:
		return c.Send("❌ Неизвестный тип кнопки")
	}
}

// showBroadcastPreview показывает превью рассылки
func showBroadcastPreview(c telebot.Context, state *BroadcastState) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	recipients, err := GetBroadcastRecipients(ctx)
	if err != nil {
		return c.Send("❌ Ошибка получения получателей")
	}

	total := len(recipients)

	// Создаем клавиатуру для превью
	var previewKeyboard *telebot.ReplyMarkup
	if state.ButtonText != "" {
		previewKeyboard = &telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.InlineButton{
				{
					telebot.InlineButton{
						Text: state.ButtonText,
						URL:  state.ButtonURL,
					},
				},
			},
		}
	}

	// Отправляем превью
	if state.PhotoID != "" {
		photo := &telebot.Photo{File: telebot.File{FileID: state.PhotoID}}
		if state.Text != "" {
			_ = c.Send(photo, state.Text, previewKeyboard)
		} else {
			_ = c.Send(photo, previewKeyboard)
		}
	} else {
		_ = c.Send(state.Text, previewKeyboard)
	}

	// Отправляем подтверждение
	confirmKeyboard := &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "✅ Отправить всем гостям",
					Data: "broadcast:send:confirm",
				},
			},
			{
				telebot.InlineButton{
					Text: "❌ Отмена",
					Data: "admin:back",
				},
			},
		},
	}

	message := fmt.Sprintf(
		"📨 <b>Проверьте сообщение выше.</b>\n\n"+
			"Оно будет отправлено в ЛС всем гостям из базы, у кого есть user_id.\n"+
			"Планируется отправка: <b>%d</b> получателям.\n\n"+
			"Если всё верно — нажмите «Отправить всем гостям».",
		total,
	)

	return c.Send(message, confirmKeyboard, telebot.ModeHTML)
}

// handleBroadcastSendConfirm отправляет рассылку
func handleBroadcastSendConfirm(c telebot.Context) error {
	userID := c.Sender().ID

	state := GetBroadcastState(userID)
	if state == nil {
		return c.Send("❌ Ошибка: состояние рассылки потеряно")
	}

	if state.Text == "" {
		return c.Send("❌ Нет текста сообщения")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	recipients, err := GetBroadcastRecipients(ctx)
	if err != nil {
		return c.Send("❌ Ошибка получения получателей")
	}

	total := len(recipients)
	if total == 0 {
		return c.Send("⚠️ В базе гостей пока нет ни одного user_id, рассылать некому.")
	}

	_ = c.Send(fmt.Sprintf("🚀 Начинаю рассылку для <b>%d</b> получателей…", total), telebot.ModeHTML)

	// Получаем бота
	mu.RLock()
	bot := botInstance
	mu.RUnlock()

	if bot == nil {
		return c.Send("❌ Бот не инициализирован")
	}

	// Отправляем рассылку
	sent, failed := SendBroadcast(bot, state, recipients)

	// Очищаем состояние
	ClearBroadcastState(userID)

	message := fmt.Sprintf(
		"✅ <b>Рассылка завершена.</b>\n\n"+
			"Успешно отправлено: <b>%d</b>\n"+
			"С ошибкой: <b>%d</b>",
		sent, failed,
	)

	return c.Send(message, telebot.ModeHTML)
}

// handleBroadcastCancel отменяет рассылку и возвращает в админ-меню
func handleBroadcastCancel(c telebot.Context) error {
	userID := c.Sender().ID

	// Очищаем состояние рассылки
	ClearBroadcastState(userID)

	// Возвращаемся в админ-меню
	message := "🔧 <b>Панель администратора</b>\n\nВыберите раздел:"
	keyboard := keyboards.GetAdminRootReplyKeyboard()
	return c.Edit(message, keyboard, telebot.ModeHTML)
}

