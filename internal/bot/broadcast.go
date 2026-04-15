package bot

import (
	"context"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BroadcastState хранит состояние рассылки для пользователя
type BroadcastState struct {
	Text                  string
	PhotoID               string
	VideoID               string
	ButtonURL             string
	ButtonText            string
	Buttons               []tgbotapi.InlineKeyboardButton
	SelectedPresetButtons map[string]bool
	Recipients            []int64 // пустой = всем, иначе список конкретных user_id
	AvailableRecipients   []BroadcastRecipientInfo
	Step                  string // текущий шаг: "text", "media", "button", "recipients", "preview"
	TestOnly              bool
}

var (
	broadcastStates = make(map[int64]*BroadcastState)
	broadcastMu     sync.RWMutex
)

// GetBroadcastState получает состояние рассылки для пользователя
func GetBroadcastState(userID int64) *BroadcastState {
	broadcastMu.RLock()
	defer broadcastMu.RUnlock()
	return broadcastStates[userID]
}

// SetBroadcastState устанавливает состояние рассылки для пользователя
func SetBroadcastState(userID int64, state *BroadcastState) {
	broadcastMu.Lock()
	defer broadcastMu.Unlock()
	broadcastStates[userID] = state
}

// ClearBroadcastState очищает состояние рассылки для пользователя
func ClearBroadcastState(userID int64) {
	broadcastMu.Lock()
	defer broadcastMu.Unlock()
	delete(broadcastStates, userID)
}

// GetBroadcastRecipients получает список получателей для рассылки
func GetBroadcastRecipients(ctx context.Context) ([]int64, error) {
	recipients, err := GetBroadcastRecipientsWithInfo(ctx)
	if err != nil {
		return nil, err
	}

	return recipientIDsFromInfo(recipients), nil
}

// SendBroadcast отправляет рассылку всем получателям
func SendBroadcast(bot *tgbotapi.BotAPI, state *BroadcastState, recipients []int64) (sent, failed int) {
	// Если указаны конкретные получатели, используем их, иначе всех
	var targetRecipients []int64
	if len(state.Recipients) > 0 {
		targetRecipients = uniqueRecipientIDs(state.Recipients)
	} else {
		targetRecipients = uniqueRecipientIDs(recipients)
	}

	for _, userID := range targetRecipients {
		// Создаем клавиатуру если есть кнопка
		keyboard := broadcastReplyMarkup(state)
		renderedText := broadcastRenderedText(state)

		// Отправляем сообщение
		var err error
		if state.VideoID != "" {
			// Приоритет видео над фото
			video := tgbotapi.NewVideo(userID, tgbotapi.FileID(state.VideoID))
			if renderedText != "" {
				video.Caption = renderedText
			}
			if keyboard != nil {
				video.ReplyMarkup = keyboard
			}
			_, err = bot.Send(video)
		} else if state.PhotoID != "" {
			photo := tgbotapi.NewPhoto(userID, tgbotapi.FileID(state.PhotoID))
			if renderedText != "" {
				photo.Caption = renderedText
			}
			if keyboard != nil {
				photo.ReplyMarkup = keyboard
			}
			_, err = bot.Send(photo)
		} else {
			msg := tgbotapi.NewMessage(userID, renderedText)
			if keyboard != nil {
				msg.ReplyMarkup = keyboard
			}
			_, err = bot.Send(msg)
		}

		if err != nil {
			log.Printf("Ошибка отправки рассылки пользователю %d: %v", userID, err)
			failed++
		} else {
			sent++
		}

		// Небольшая пауза, чтобы не упереться в лимиты
		time.Sleep(50 * time.Millisecond)
	}

	return sent, failed
}

func broadcastReplyMarkup(state *BroadcastState) *tgbotapi.InlineKeyboardMarkup {
	if state == nil {
		return nil
	}

	buttons := state.Buttons
	if len(buttons) == 0 && state.ButtonText != "" && state.ButtonURL != "" {
		buttons = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonURL(state.ButtonText, state.ButtonURL),
		}
	}
	if len(buttons) == 0 {
		return nil
	}

	replyMarkup := tgbotapi.NewInlineKeyboardMarkup(broadcastButtonRows(buttons)...)
	return &replyMarkup
}

func broadcastButtonRows(buttons []tgbotapi.InlineKeyboardButton) [][]tgbotapi.InlineKeyboardButton {
	if len(buttons) == 0 {
		return nil
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, 0, (len(buttons)+1)/2)
	for i := 0; i < len(buttons); i += 2 {
		end := i + 2
		if end > len(buttons) {
			end = len(buttons)
		}

		row := make([]tgbotapi.InlineKeyboardButton, 0, end-i)
		row = append(row, buttons[i:end]...)
		rows = append(rows, row)
	}

	return rows
}

func clearBroadcastButtons(state *BroadcastState) {
	if state == nil {
		return
	}

	state.ButtonText = ""
	state.ButtonURL = ""
	state.Buttons = nil
	state.SelectedPresetButtons = nil
}

func addBroadcastButton(state *BroadcastState, buttonText string, buttonURL string) {
	if state == nil || buttonText == "" || buttonURL == "" {
		return
	}

	state.Buttons = addBroadcastInlineButton(state.Buttons, tgbotapi.NewInlineKeyboardButtonURL(buttonText, buttonURL))
	state.ButtonText = ""
	state.ButtonURL = ""
}

func broadcastButtonURL(button tgbotapi.InlineKeyboardButton) string {
	if button.URL == nil {
		return ""
	}

	return *button.URL
}
