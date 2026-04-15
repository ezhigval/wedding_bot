package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"wedding-bot/internal/google_sheets"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BroadcastRecipientInfo хранит информацию о получателе
type BroadcastRecipientInfo struct {
	UserID      int64
	DisplayName string
	Username    string
}

// GetBroadcastRecipientsWithInfo получает список получателей с информацией
func GetBroadcastRecipientsWithInfo(ctx context.Context) ([]BroadcastRecipientInfo, error) {
	guests, err := google_sheets.GetAllGuestsFromSheets(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения гостей: %w", err)
	}

	return buildBroadcastRecipientsInfo(guests), nil
}

func buildBroadcastRecipientsInfo(guests []google_sheets.GuestInfo) []BroadcastRecipientInfo {
	recipientsByID := make(map[int64]BroadcastRecipientInfo, len(guests))
	order := make([]int64, 0, len(guests))

	for _, guest := range guests {
		if guest.UserID == "" {
			continue
		}

		userID, err := strconv.ParseInt(guest.UserID, 10, 64)
		if err != nil || userID <= 0 {
			continue
		}

		recipient, exists := recipientsByID[userID]
		if !exists {
			order = append(order, userID)
			recipient = BroadcastRecipientInfo{UserID: userID}
		}

		displayName := strings.TrimSpace(fmt.Sprintf("%s %s", guest.FirstName, guest.LastName))
		if displayName == "" {
			displayName = guest.Username
		}
		if displayName == "" {
			displayName = fmt.Sprintf("user_id %d", userID)
		}

		if recipient.DisplayName == "" {
			recipient.DisplayName = displayName
		}
		if recipient.Username == "" {
			recipient.Username = strings.TrimSpace(guest.Username)
		}

		recipientsByID[userID] = recipient
	}

	recipients := make([]BroadcastRecipientInfo, 0, len(order))
	for _, userID := range order {
		recipients = append(recipients, recipientsByID[userID])
	}

	return recipients
}

func recipientIDsFromInfo(recipients []BroadcastRecipientInfo) []int64 {
	ids := make([]int64, 0, len(recipients))
	for _, recipient := range recipients {
		if recipient.UserID > 0 {
			ids = append(ids, recipient.UserID)
		}
	}

	return uniqueRecipientIDs(ids)
}

func uniqueRecipientIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func containsRecipientID(ids []int64, target int64) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}

	return false
}

func addRecipientID(ids []int64, target int64) []int64 {
	if target <= 0 || containsRecipientID(ids, target) {
		return uniqueRecipientIDs(ids)
	}

	return append(uniqueRecipientIDs(ids), target)
}

func removeRecipientID(ids []int64, target int64) []int64 {
	if len(ids) == 0 {
		return nil
	}

	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id != target {
			result = append(result, id)
		}
	}

	return uniqueRecipientIDs(result)
}

func filterRecipientIDsByAvailable(ids []int64, recipients []BroadcastRecipientInfo) []int64 {
	if len(ids) == 0 || len(recipients) == 0 {
		return nil
	}

	allowed := make(map[int64]struct{}, len(recipients))
	for _, recipient := range recipients {
		allowed[recipient.UserID] = struct{}{}
	}

	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := allowed[id]; ok {
			result = append(result, id)
		}
	}

	return uniqueRecipientIDs(result)
}

func ensureBroadcastRecipientsLoaded(ctx context.Context, state *BroadcastState) error {
	if state == nil {
		return fmt.Errorf("состояние рассылки не найдено")
	}
	if len(state.AvailableRecipients) > 0 {
		return nil
	}

	recipients, err := GetBroadcastRecipientsWithInfo(ctx)
	if err != nil {
		return err
	}

	state.AvailableRecipients = recipients
	state.Recipients = filterRecipientIDsByAvailable(state.Recipients, recipients)
	return nil
}

// ShowRecipientsSelectionPage показывает страницу выбора получателей
func ShowRecipientsSelectionPage(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, state *BroadcastState, page int) {
	recipients := state.AvailableRecipients
	const itemsPerPage = 8
	totalPages := (len(recipients) + itemsPerPage - 1) / itemsPerPage
	if totalPages == 0 {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "⚠️ В базе гостей пока нет ни одного получателя с user_id.")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

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
		len(recipients), len(uniqueRecipientIDs(state.Recipients)),
	)

	// Добавляем список гостей на текущей странице
	for i := start; i < end; i++ {
		recipient := recipients[i]
		status := "⭕"
		if containsRecipientID(state.Recipients, recipient.UserID) {
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

		callbackData := fmt.Sprintf("broadcast:select:%d:%d", recipient.UserID, page)
		if containsRecipientID(state.Recipients, recipient.UserID) {
			callbackData = fmt.Sprintf("broadcast:deselect:%d:%d", recipient.UserID, page)
		}

		rowIndex := len(keyboardRows) - 1
		keyboardRows[rowIndex] = append(keyboardRows[rowIndex],
			tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData))
	}

	// Кнопки навигации
	var navRow []tgbotapi.InlineKeyboardButton

	// Кнопка "Выбрать всех"
	if len(uniqueRecipientIDs(state.Recipients)) < len(recipients) {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("✅ Выбрать всех", fmt.Sprintf("broadcast:select_all:%d", page)))
	} else {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("❌ Снять все", fmt.Sprintf("broadcast:deselect_all:%d", page)))
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
	if len(uniqueRecipientIDs(state.Recipients)) > 0 {
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
func HandleRecipientsSelection(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, action string, userID int64, page int) {
	state := GetBroadcastState(callback.From.ID)
	if state == nil {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ensureBroadcastRecipientsLoaded(ctx, state); err != nil {
		log.Printf("Ошибка получения получателей: %v", err)
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Ошибка получения списка получателей")
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	switch action {
	case "select":
		state.TestOnly = false
		state.Recipients = addRecipientID(state.Recipients, userID)
		ShowRecipientsSelectionPage(bot, callback, state, page)

	case "deselect":
		state.TestOnly = false
		state.Recipients = removeRecipientID(state.Recipients, userID)
		ShowRecipientsSelectionPage(bot, callback, state, page)

	case "select_all":
		state.TestOnly = false
		state.Recipients = recipientIDsFromInfo(state.AvailableRecipients)
		ShowRecipientsSelectionPage(bot, callback, state, page)

	case "deselect_all":
		state.TestOnly = false
		state.Recipients = nil
		ShowRecipientsSelectionPage(bot, callback, state, page)

	case "send_selected":
		state.TestOnly = false
		state.Recipients = filterRecipientIDsByAvailable(state.Recipients, state.AvailableRecipients)
		if len(state.Recipients) == 0 {
			msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Никто не выбран")
			bot.Send(msg)
			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			return
		}

		state.Step = "preview"
		showBroadcastPreview(bot, callback, state)
	}
}
