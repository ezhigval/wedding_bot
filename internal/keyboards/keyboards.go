package keyboards

import (
	"fmt"

	"gopkg.in/telebot.v3"

	"wedding-bot/internal/config"
)

// GetInvitationKeyboard возвращает клавиатуру для приглашения с Mini App
func GetInvitationKeyboard() *telebot.ReplyMarkup {
	return &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "💒 Открыть приглашение",
					WebApp: &telebot.WebApp{
						URL: config.WebappURL,
					},
				},
			},
		},
	}
}

// GetMainReplyKeyboard возвращает основную пользовательскую клавиатуру
func GetMainReplyKeyboard(isAdmin bool, photoModeEnabled bool) *telebot.ReplyMarkup {
	photoLabel := "📸 Фоторежим ❌"
	if photoModeEnabled {
		photoLabel = "📸 Фоторежим ✅"
	}

	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}

	row1 := markup.Row(
		telebot.Btn{
			Text: "📱 Приглашение",
			WebApp: &telebot.WebApp{
				URL: config.WebappURL,
			},
		},
		telebot.Btn{Text: photoLabel},
	)

	row2 := markup.Row(
		telebot.Btn{Text: "💬 Общий чат"},
		telebot.Btn{Text: "📞 Связаться с нами"},
	)

	rows := []telebot.Row{row1, row2}

	if isAdmin {
		rows = append(rows, markup.Row(
			telebot.Btn{Text: "🛠 Админ-панель"},
		))
	}

	markup.Reply(rows...)
	return markup
}

// GetContactsInlineKeyboard возвращает кнопки для быстрого перехода в диалог с организаторами
func GetContactsInlineKeyboard() *telebot.ReplyMarkup {
	var buttons []telebot.InlineButton

	if config.GroomTelegram != "" {
		buttons = append(buttons, telebot.InlineButton{
			Text: fmt.Sprintf("Валентин (@%s)", config.GroomTelegram),
			URL:  fmt.Sprintf("https://t.me/%s", config.GroomTelegram),
		})
	}

	if config.BrideTelegram != "" {
		buttons = append(buttons, telebot.InlineButton{
			Text: fmt.Sprintf("Мария (@%s)", config.BrideTelegram),
			URL:  fmt.Sprintf("https://t.me/%s", config.BrideTelegram),
		})
	}

	if len(buttons) == 0 {
		// Fallback
		telegram := config.GroomTelegram
		if telegram == "" {
			telegram = config.BrideTelegram
		}
		if telegram != "" {
			buttons = append(buttons, telebot.InlineButton{
				Text: "Организатор",
				URL:  fmt.Sprintf("https://t.me/%s", telegram),
			})
		}
	}

	keyboard := make([][]telebot.InlineButton, len(buttons))
	for i, btn := range buttons {
		keyboard[i] = []telebot.InlineButton{btn}
	}

	return &telebot.ReplyMarkup{
		InlineKeyboard: keyboard,
	}
}

// GetGroupLinkKeyboard возвращает кнопку для перехода в общий чат
func GetGroupLinkKeyboard() *telebot.ReplyMarkup {
	return &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "Перейти в общий чат",
					URL:  config.GroupLink,
				},
			},
		},
	}
}

