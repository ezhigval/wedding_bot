package keyboards

import (
	"fmt"
	"strings"

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

	// Временно отключаем Web App для локального тестирования (HTTP не поддерживается Telegram)
	// Используем обычную кнопку вместо Web App
	var row1 telebot.Row
	if strings.HasPrefix(config.WebappURL, "https://") {
		// Только для HTTPS используем Web App
		row1 = markup.Row(
			telebot.Btn{
				Text: "📱 Приглашение",
				WebApp: &telebot.WebApp{
					URL: config.WebappURL,
				},
			},
			telebot.Btn{Text: photoLabel},
		)
	} else {
		// Для HTTP используем обычную кнопку с URL
		row1 = markup.Row(
			telebot.Btn{
				Text: "📱 Приглашение",
				URL: config.WebappURL,
			},
			telebot.Btn{Text: photoLabel},
		)
	}

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

// GetAdminRootReplyKeyboard возвращает корневое меню администратора
func GetAdminRootReplyKeyboard() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}

	row1 := markup.Row(
		telebot.Btn{Text: "Гости"},
		telebot.Btn{Text: "Таблица"},
	)

	row2 := markup.Row(
		telebot.Btn{Text: "Группа"},
		telebot.Btn{Text: "Бот"},
	)

	row3 := markup.Row(
		telebot.Btn{Text: "⬅️ Вернуться"},
	)

	markup.Reply(row1, row2, row3)
	return markup
}

// GetAdminGuestsReplyKeyboard возвращает подменю администратора: гости
func GetAdminGuestsReplyKeyboard() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}

	row1 := markup.Row(
		telebot.Btn{Text: "Список гостей"},
		telebot.Btn{Text: "Рассадка"},
	)

	row2 := markup.Row(
		telebot.Btn{Text: "Отправить приглашение"},
		telebot.Btn{Text: "Исправить имя/фамилию"},
	)

	row3 := markup.Row(
		telebot.Btn{Text: "Рассылка в ЛС"},
	)

	row4 := markup.Row(
		telebot.Btn{Text: "⬅️ Вернуться"},
	)

	markup.Reply(row1, row2, row3, row4)
	return markup
}

// GetAdminTableReplyKeyboard возвращает подменю администратора: таблица
func GetAdminTableReplyKeyboard() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}

	row1 := markup.Row(
		telebot.Btn{Text: "Открыть таблицу"},
	)

	row2 := markup.Row(
		telebot.Btn{Text: "Проверить связь"},
	)

	row3 := markup.Row(
		telebot.Btn{Text: "Закрепить рассадку"},
	)

	row4 := markup.Row(
		telebot.Btn{Text: "⬅️ Вернуться"},
	)

	markup.Reply(row1, row2, row3, row4)
	return markup
}

// GetAdminGroupReplyKeyboard возвращает подменю администратора: группа
func GetAdminGroupReplyKeyboard() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}

	row1 := markup.Row(
		telebot.Btn{Text: "Написать сообщение"},
	)

	row2 := markup.Row(
		telebot.Btn{Text: "Посмотреть участников"},
	)

	row3 := markup.Row(
		telebot.Btn{Text: "Добавить/Удалить"},
	)

	row4 := markup.Row(
		telebot.Btn{Text: "⬅️ Вернуться"},
	)

	markup.Reply(row1, row2, row3, row4)
	return markup
}

// GetAdminBotReplyKeyboard возвращает подменю администратора: бот
func GetAdminBotReplyKeyboard() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}

	row1 := markup.Row(
		telebot.Btn{Text: "Статус бота"},
	)

	row2 := markup.Row(
		telebot.Btn{Text: "🔐 Авторизовать клиент"},
	)

	row3 := markup.Row(
		telebot.Btn{Text: "Начать с нуля"},
	)

	row4 := markup.Row(
		telebot.Btn{Text: "Добавить админа"},
	)

	row5 := markup.Row(
		telebot.Btn{Text: "🆔 Найти user_id"},
	)

	row6 := markup.Row(
		telebot.Btn{Text: "⬅️ Вернуться"},
	)

	markup.Reply(row1, row2, row3, row4, row5, row6)
	return markup
}

// GetAdminGamesKeyboard возвращает клавиатуру для управления играми
func GetAdminGamesKeyboard() *telebot.ReplyMarkup {
	return &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "🔤 Wordle",
					Data: "admin:games:wordle",
				},
				telebot.InlineButton{
					Text: "📝 Кроссворд",
					Data: "admin:games:crossword",
				},
			},
			{
				telebot.InlineButton{
					Text: "⬅️ Назад",
					Data: "admin:back",
				},
			},
		},
	}
}

// GetAdminWordleKeyboard возвращает клавиатуру для управления Wordle
func GetAdminWordleKeyboard() *telebot.ReplyMarkup {
	return &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "🔄 Переключить слово для всех",
					Data: "admin:games:wordle:switch",
				},
			},
			{
				telebot.InlineButton{
					Text: "➕ Добавить слово",
					Data: "admin:games:wordle:add",
				},
			},
			{
				telebot.InlineButton{
					Text: "⬅️ Назад",
					Data: "admin:games",
				},
			},
		},
	}
}

// GetAdminCrosswordKeyboard возвращает клавиатуру для управления кроссвордом
func GetAdminCrosswordKeyboard() *telebot.ReplyMarkup {
	return &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "🔄 Обновить кроссворд",
					Data: "admin:games:crossword:update",
				},
			},
			{
				telebot.InlineButton{
					Text: "➕ Добавить кроссворд",
					Data: "admin:games:crossword:add",
				},
			},
			{
				telebot.InlineButton{
					Text: "⬅️ Назад",
					Data: "admin:games",
				},
			},
		},
	}
}

// GetGroupManagementKeyboard возвращает клавиатуру для управления группой
func GetGroupManagementKeyboard() *telebot.ReplyMarkup {
	return &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "📢 Отправить сообщение в группу",
					Data: "group:send_message",
				},
			},
			{
				telebot.InlineButton{
					Text: "➕ Добавить участника",
					Data: "group:add_member",
				},
				telebot.InlineButton{
					Text: "➖ Удалить участника",
					Data: "group:remove_member",
				},
			},
			{
				telebot.InlineButton{
					Text: "👥 Список участников",
					Data: "group:list_members",
				},
			},
			{
				telebot.InlineButton{
					Text: "⬅️ Вернуться",
					Data: "admin:back",
				},
			},
		},
	}
}

// InvitationInfoForKeyboard представляет информацию о приглашении для клавиатуры
type InvitationInfoForKeyboard struct {
	Name   string
	IsSent bool
}

// GetGuestsSelectionKeyboard возвращает клавиатуру с кнопками для выбора гостя из списка приглашений
func GetGuestsSelectionKeyboard(invitations []InvitationInfoForKeyboard) *telebot.ReplyMarkup {
	var keyboard [][]telebot.InlineButton

	// Создаем кнопки для каждого гостя (максимум 2 кнопки в ряд)
	for i := 0; i < len(invitations); i += 2 {
		var row []telebot.InlineButton

		// Первая кнопка в ряду
		if i < len(invitations) {
			inv := invitations[i]
			var buttonText string
			if inv.IsSent {
				buttonText = fmt.Sprintf("✅ %s", inv.Name)
			} else {
				buttonText = fmt.Sprintf("👤 %s", inv.Name)
			}
			row = append(row, telebot.InlineButton{
				Text: buttonText,
				Data: fmt.Sprintf("invite_guest_%d", i),
			})
		}

		// Вторая кнопка в ряду (если есть)
		if i+1 < len(invitations) {
			inv := invitations[i+1]
			var buttonText string
			if inv.IsSent {
				buttonText = fmt.Sprintf("✅ %s", inv.Name)
			} else {
				buttonText = fmt.Sprintf("👤 %s", inv.Name)
			}
			row = append(row, telebot.InlineButton{
				Text: buttonText,
				Data: fmt.Sprintf("invite_guest_%d", i+1),
			})
		}

		keyboard = append(keyboard, row)
	}

	// Кнопка возврата
	keyboard = append(keyboard, []telebot.InlineButton{
		{
			Text: "⬅️ Вернуться",
			Data: "admin:back",
		},
	})

	return &telebot.ReplyMarkup{
		InlineKeyboard: keyboard,
	}
}

