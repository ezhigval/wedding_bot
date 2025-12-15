package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gopkg.in/telebot.v3"

	"wedding-bot/internal/google_sheets"
)

// BroadcastState хранит состояние рассылки для пользователя
type BroadcastState struct {
	Text      string
	PhotoID   string
	ButtonURL string
	ButtonText string
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
func SendBroadcast(bot *telebot.Bot, state *BroadcastState, recipients []int64) (sent, failed int) {
	for _, userID := range recipients {
		// Создаем клавиатуру если есть кнопка
		var keyboard *telebot.ReplyMarkup
		if state.ButtonText != "" && state.ButtonURL != "" {
			keyboard = &telebot.ReplyMarkup{
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

		// Отправляем сообщение
		var err error
		if state.PhotoID != "" {
			photo := &telebot.Photo{File: telebot.File{FileID: state.PhotoID}}
			if state.Text != "" {
				// Отправляем фото с подписью
				_, err = bot.Send(&telebot.User{ID: userID}, photo, state.Text, keyboard)
			} else {
				_, err = bot.Send(&telebot.User{ID: userID}, photo, keyboard)
			}
		} else {
			_, err = bot.Send(&telebot.User{ID: userID}, state.Text, keyboard)
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

