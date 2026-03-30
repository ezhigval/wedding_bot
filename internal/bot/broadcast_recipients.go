package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"wedding-bot/internal/google_sheets"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BroadcastRecipientInfo хранит информацию о получателе
type BroadcastRecipientInfo struct {
	UserID      int64
	DisplayName string
	Username    string
	Selected    bool
}

// GetBroadcastRecipientsWithInfo получает список получателей с информацией
func GetBroadcastRecipientsWithInfo(ctx context.Context) ([]BroadcastRecipientInfo, error) {
	guests, err := google_sheets.GetAllGuestsFromSheets(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения гостей: %w", err)
	}

	var recipients []BroadcastRecipientInfo
	for _, guest := range guests {
		if guest.UserID != "" {
			userID, err := strconv.ParseInt(guest.UserID, 10, 64)
			if err == nil {
				recipients = append(recipients, BroadcastRecipientInfo{
					UserID:      userID,
					DisplayName: fmt.Sprintf("%s %s", guest.FirstName, guest.LastName),
					Username:    guest.Username,
					Selected:    false,
				})
			}
		}
	}

	return recipients, nil
}

// ShowRecipientsSelectionPage показывает страницу выбора получателей
func ShowRecipientsSelectionPage(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, recipients []BroadcastRecipientInfo, page int, selectedCount int) {
	const itemsPerPage = 8
	totalPages := (len(recipients) + itemsPerPage - 1) / itemsPerPage

	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > len(recipients) {
		end = len(recipients)
	}

	// Формируем текст с информацией о выбранных
	msgText := fmt.Sprintf(
		"📝 <b>Выбор получателей вручную</b>\n\n"+
			"Всего гостей с user_id: <b>%d</b>\n"+
			"Выбрано: <b>%d</b>\n\n",
		len(recipients), selectedCount,
	)

	// Добавляем список гостей на текущей странице
	for i := start; i < end; i++ {
		recipient := recipients[i]
		status := "⭕"
		if recipient.Selected {
			status = "✅"
		}
		msgText += fmt.Sprintf("%s %s", status, recipient.DisplayName)
		if recipient.Username != "" {
			msgText += fmt.Sprintf(" (@%s)", recipient.Username)
		}
		msgText += "\n"
	}

	// Создаем клавиатуру
	var keyboardRows [][]tgbotapi.InlineKeyboardButton

	// Кнопки выбора гостей
	for i := start; i < end; i++ {
		if i%2 == start%2 { // Новая строка каждые 2 кнопки
			keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{})
		}

		recipient := recipients[i]
		buttonText := recipient.DisplayName
		if len(buttonText) > 20 {
			buttonText = buttonText[:17] + "..."
		}

		callbackData := fmt.Sprintf("broadcast:select:%d", recipient.UserID)
		if recipient.Selected {
			callbackData = fmt.Sprintf("broadcast:deselect:%d", recipient.UserID)
		}

		rowIndex := len(keyboardRows) - 1
		keyboardRows[rowIndex] = append(keyboardRows[rowIndex],
			tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData))
	}

	// Кнопки навигации
	var navRow []tgbotapi.InlineKeyboardButton

	// Кнопка "Выбрать всех"
	if selectedCount < len(recipients) {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("✅ Выбрать всех", "broadcast:select_all"))
	} else {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("❌ Снять все", "broadcast:deselect_all"))
	}

	// Пагинация
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("broadcast:page:%d", page-1)))
	}
	if page < totalPages-1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("➡️ Вперед", fmt.Sprintf("broadcast:page:%d", page+1)))
	}

	keyboardRows = append(keyboardRows, navRow)

	// Кнопки действий
	if selectedCount > 0 {
		keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📨 Отправить выбранным", "broadcast:send_selected"),
		})
	}

	keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "broadcast:cancel"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)

	// Отправляем или редактируем сообщение
	if callback.Message != nil {
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, msgText)
		editMsg.ParseMode = tgbotapi.ModeHTML
		editMsg.ReplyMarkup = &keyboard
		bot.Send(editMsg)
	} else {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
		msg.ParseMode = tgbotapi.ModeHTML
		msg.ReplyMarkup = &keyboard
		bot.Send(msg)
	}

	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

// HandleRecipientsSelection обрабатывает выбор получателей
func HandleRecipientsSelection(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, action string, userID int64) {
	state := GetBroadcastState(callback.From.ID)
	if state == nil {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

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

	switch action {
	case "select":
		// Выбор конкретного пользователя
		for i := range recipients {
			if recipients[i].UserID == userID {
				recipients[i].Selected = true
				break
			}
		}
		ShowRecipientsSelectionPage(bot, callback, recipients, 0, countSelected(recipients))

	case "deselect":
		// Снятие выбора с пользователя
		for i := range recipients {
			if recipients[i].UserID == userID {
				recipients[i].Selected = false
				break
			}
		}
		ShowRecipientsSelectionPage(bot, callback, recipients, 0, countSelected(recipients))

	case "select_all":
		// Выбрать всех
		for i := range recipients {
			recipients[i].Selected = true
		}
		ShowRecipientsSelectionPage(bot, callback, recipients, 0, len(recipients))

	case "deselect_all":
		// Снять выбор со всех
		for i := range recipients {
			recipients[i].Selected = false
		}
		ShowRecipientsSelectionPage(bot, callback, recipients, 0, 0)

	case "send_selected":
		// Отправить выбранным
		var selectedIDs []int64
		for _, recipient := range recipients {
			if recipient.Selected {
				selectedIDs = append(selectedIDs, recipient.UserID)
			}
		}

		if len(selectedIDs) == 0 {
			msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Никто не выбран")
			bot.Send(msg)
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}

		// Сохраняем выбранных получателей в состояние
		state.Recipients = selectedIDs
		state.Step = "preview"
		showBroadcastPreview(bot, callback, state)
	}
}

func countSelected(recipients []BroadcastRecipientInfo) int {
	count := 0
	for _, recipient := range recipients {
		if recipient.Selected {
			count++
		}
	}
	return count
}
