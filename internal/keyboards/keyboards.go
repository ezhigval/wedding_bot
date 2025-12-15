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

	var webAppButton telebot.Btn
	if strings.HasPrefix(config.WebappURL, "https://") {
		webAppButton = telebot.Btn{
			Text: "💒 Открыть приглашение",
			WebApp: &telebot.WebApp{
				URL: config.WebappURL,
			},
		}
	} else {
		// Если URL не HTTPS, используем обычную кнопку
		webAppButton = telebot.Btn{
			Text: "📱 Приглашение (локально)",
		}
	}

	row1 := markup.Row(
		webAppButton,
		telebot.Btn{Text: photoLabel},
	)

	if isAdmin {
		row2 := markup.Row(
			telebot.Btn{Text: "🛠 Админ-панель"},
		)
		markup.Reply(row1, row2)
	} else {
		markup.Reply(row1)
	}

	return markup
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
		telebot.Btn{Text: "Проверить связь"},
	)

	row3 := markup.Row(
		telebot.Btn{Text: "Закрепить рассадку"},
	)

	row4 := markup.Row(
		telebot.Btn{Text: "⬅️ Вернуться"},
	)

	markup.Reply(row1, row3, row4)
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

// InvitationInfoForKeyboard структура для передачи информации о приглашении в клавиатуру
type InvitationInfoForKeyboard struct {
	Name   string
	IsSent bool
}

// GetGuestsSelectionKeyboard возвращает клавиатуру с кнопками для выбора гостя из списка приглашений
func GetGuestsSelectionKeyboard(invitations []InvitationInfoForKeyboard) *telebot.ReplyMarkup {
	var rows []telebot.Row
	for i := 0; i < len(invitations); i += 2 {
		var row []telebot.InlineButton
		// Первая кнопка в ряду
		inv1 := invitations[i]
		buttonText1 := fmt.Sprintf("👤 %s", inv1.Name)
		if inv1.IsSent {
			buttonText1 = fmt.Sprintf("✅ %s", inv1.Name)
		}
		row = append(row, telebot.InlineButton{
			Text: buttonText1,
			Data: fmt.Sprintf("admin:invite_guest:%d", i),
		})

		// Вторая кнопка в ряду (если есть)
		if i+1 < len(invitations) {
			inv2 := invitations[i+1]
			buttonText2 := fmt.Sprintf("👤 %s", inv2.Name)
			if inv2.IsSent {
				buttonText2 = fmt.Sprintf("✅ %s", inv2.Name)
			}
			row = append(row, telebot.InlineButton{
				Text: buttonText2,
				Data: fmt.Sprintf("admin:invite_guest:%d", i+1),
			})
		}
		rows = append(rows, telebot.Row(row))
	}

	// Кнопка возврата
	rows = append(rows, telebot.Row{
		telebot.InlineButton{
			Text: "⬅️ Вернуться",
			Data: "admin:back",
		},
	})

	return &telebot.ReplyMarkup{InlineKeyboard: rows}
}

// GetGuestsSwapKeyboard создает клавиатуру с кнопками для выбора гостя для обмена имени/фамилии
func GetGuestsSwapKeyboard(guests []map[string]interface{}, page int) *telebot.ReplyMarkup {
	const itemsPerPage = 10
	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > len(guests) {
		end = len(guests)
	}

	var keyboard [][]telebot.InlineButton
	for i := start; i < end; i++ {
		guest := guests[i]
		rowNum, _ := guest["row"].(int)
		fullName, _ := guest["full_name"].(string)
		if fullName == "" {
			fullName = "Без имени"
		}

		buttonText := fmt.Sprintf("👤 %s", fullName)
		keyboard = append(keyboard, []telebot.InlineButton{
			{
				Text: buttonText,
				Data: fmt.Sprintf("swapname:%d", rowNum),
			},
		})
	}

	// Навигация по страницам
	var navRow []telebot.InlineButton
	if page > 0 {
		navRow = append(navRow, telebot.InlineButton{
			Text: "⬅️ Назад",
			Data: fmt.Sprintf("fixnames_page:%d", page-1),
		})
	}
	if end < len(guests) {
		navRow = append(navRow, telebot.InlineButton{
			Text: "Вперед ➡️",
			Data: fmt.Sprintf("fixnames_page:%d", page+1),
		})
	}
	if len(navRow) > 0 {
		keyboard = append(keyboard, navRow)
	}

	// Кнопка возврата
	keyboard = append(keyboard, []telebot.InlineButton{
		{
			Text: "⬅️ Вернуться",
			Data: "admin:back",
		},
	})

	return &telebot.ReplyMarkup{InlineKeyboard: keyboard}
}
