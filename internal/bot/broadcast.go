package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"wedding-bot/internal/google_sheets"
)

// BroadcastState хранит состояние рассылки для пользователя
type BroadcastState struct {
	Text       string
	PhotoID    string
	VideoID    string
	ButtonURL  string
	ButtonText string
	Recipients []int64 // пустой = всем, иначе список конкретных user_id
	Step       string  // текущий шаг: "text", "media", "button", "recipients", "preview"
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
	guests, err := google_sheets.GetAllGuestsFromSheets(ctx)
	if err != nil {
		return nil, err
	}

	var recipients []int64
	for _, guest := range guests {
		if guest.UserID != "" {
			// Парсим user_id из строки
			var userID int64
			if _, err := fmt.Sscanf(guest.UserID, "%d", &userID); err == nil {
				recipients = append(recipients, userID)
			}
		}
	}

	return recipients, nil
}

// SendBroadcast отправляет рассылку всем получателям
func SendBroadcast(bot *tgbotapi.BotAPI, state *BroadcastState, recipients []int64) (sent, failed int) {
	// Если указаны конкретные получатели, используем их, иначе всех
	var targetRecipients []int64
	if len(state.Recipients) > 0 {
		targetRecipients = state.Recipients
	} else {
		targetRecipients = recipients
	}

	for _, userID := range targetRecipients {
		// Создаем клавиатуру если есть кнопка
		var keyboard *tgbotapi.InlineKeyboardMarkup
		if state.ButtonText != "" && state.ButtonURL != "" {
			keyboard = &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{
						tgbotapi.NewInlineKeyboardButtonURL(state.ButtonText, state.ButtonURL),
					},
				},
			}
		}

		// Отправляем сообщение
		var err error
		if state.VideoID != "" {
			// Приоритет видео над фото
			video := tgbotapi.NewVideo(userID, tgbotapi.FileID(state.VideoID))
			if state.Text != "" {
				video.Caption = state.Text
			}
			if keyboard != nil {
				video.ReplyMarkup = keyboard
			}
			_, err = bot.Send(video)
		} else if state.PhotoID != "" {
			photo := tgbotapi.NewPhoto(userID, tgbotapi.FileID(state.PhotoID))
			if state.Text != "" {
				photo.Caption = state.Text
			}
			if keyboard != nil {
				photo.ReplyMarkup = keyboard
			}
			_, err = bot.Send(photo)
		} else {
			msg := tgbotapi.NewMessage(userID, state.Text)
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
