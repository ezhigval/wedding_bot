package keyboards

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"wedding-bot/internal/config"
)

// GetInvitationKeyboard возвращает клавиатуру для приглашения с Mini App
func GetInvitationKeyboard() tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton
	if strings.HasPrefix(config.WebappURL, "https://") {
		keyboard = [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonURL("💒 Открыть приглашение", config.WebappURL),
			},
		}
	} else {
		keyboard = [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonURL("📱 Приглашение", config.WebappURL),
			},
		}
	}
	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

// GetMainReplyKeyboard возвращает основную пользовательскую клавиатуру
func GetMainReplyKeyboard(isAdmin bool, photoModeEnabled bool) tgbotapi.ReplyKeyboardMarkup {
	photoLabel := "📸 Фоторежим ❌"
	if photoModeEnabled {
		photoLabel = "📸 Фоторежим ✅"
	}

	var keyboard [][]tgbotapi.KeyboardButton

	// Первая строка
	var row1 []tgbotapi.KeyboardButton
	row1 = append(row1, tgbotapi.NewKeyboardButton("💒 Открыть приглашение"))
	row1 = append(row1, tgbotapi.NewKeyboardButton(photoLabel))
	keyboard = append(keyboard, row1)

	// Вторая строка: контакты и общий чат
	row2 := []tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButton("💬 Общий чат"),
		tgbotapi.NewKeyboardButton("📞 Связаться с нами"),
	}
	keyboard = append(keyboard, row2)

	// Вторая строка для админов
	if isAdmin {
		row3 := []tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("⚙️ Админ-панель"),
		}
		keyboard = append(keyboard, row3)
	}

	return tgbotapi.NewReplyKeyboard(keyboard...)
}

// GetAdminRootReplyKeyboard возвращает корневое меню администратора
func GetAdminRootReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := [][]tgbotapi.KeyboardButton{
		{
			tgbotapi.NewKeyboardButton("👥 Гости"),
			tgbotapi.NewKeyboardButton("🪑 Столы"),
		},
		{
			tgbotapi.NewKeyboardButton("💬 Группа"),
			tgbotapi.NewKeyboardButton("🤖 Бот"),
		},
		{
			tgbotapi.NewKeyboardButton("🎮 Игры"),
		},
		{
			tgbotapi.NewKeyboardButton("⬅️ Вернуться"),
		},
	}
	return tgbotapi.NewReplyKeyboard(keyboard...)
}

// GetAdminGuestsReplyKeyboard возвращает подменю администратора: гости
func GetAdminGuestsReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := [][]tgbotapi.KeyboardButton{
		{
			tgbotapi.NewKeyboardButton("📋 Список гостей"),
			tgbotapi.NewKeyboardButton("📤 Отправить приглашение"),
		},
		{
			tgbotapi.NewKeyboardButton("🔁 Исправление Имя/Фамилия"),
		},
		{
			tgbotapi.NewKeyboardButton("⬅️ Вернуться"),
		},
	}
	return tgbotapi.NewReplyKeyboard(keyboard...)
}

// GetAdminTableReplyKeyboard возвращает подменю администратора: таблица
func GetAdminTableReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := [][]tgbotapi.KeyboardButton{
		{
			tgbotapi.NewKeyboardButton("📊 Посмотреть рассадку"),
			tgbotapi.NewKeyboardButton("🔄 Обновить рассадку"),
		},
		{
			tgbotapi.NewKeyboardButton("⬅️ Вернуться"),
		},
	}
	return tgbotapi.NewReplyKeyboard(keyboard...)
}

// GetAdminGroupReplyKeyboard возвращает подменю администратора: группа
func GetAdminGroupReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := [][]tgbotapi.KeyboardButton{
		{
			tgbotapi.NewKeyboardButton("Написать сообщение"),
		},
		{
			tgbotapi.NewKeyboardButton("Посмотреть участников"),
		},
		{
			tgbotapi.NewKeyboardButton("Добавить/Удалить"),
		},
		{
			tgbotapi.NewKeyboardButton("⬅️ Вернуться"),
		},
	}
	return tgbotapi.NewReplyKeyboard(keyboard...)
}

// GetAdminBotReplyKeyboard возвращает подменю администратора: бот
func GetAdminBotReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := [][]tgbotapi.KeyboardButton{
		{
			tgbotapi.NewKeyboardButton("📊 Статус бота"),
		},
		{
			tgbotapi.NewKeyboardButton("Начать с нуля"),
		},
		{
			tgbotapi.NewKeyboardButton("Добавить админа"),
		},
		{
			tgbotapi.NewKeyboardButton("🆔 Найти user_id"),
		},
		{
			tgbotapi.NewKeyboardButton("⬅️ Вернуться"),
		},
	}
	return tgbotapi.NewReplyKeyboard(keyboard...)
}

// GetAdminGamesKeyboard возвращает клавиатуру для управления играми
func GetAdminGamesKeyboard() tgbotapi.InlineKeyboardMarkup {
	keyboard := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🔤 Wordle", "admin:games:wordle"),
			tgbotapi.NewInlineKeyboardButtonData("📝 Кроссворд", "admin:games:crossword"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "admin:back"),
		},
	}
	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

// GetAdminWordleKeyboard возвращает клавиатуру для управления Wordle
func GetAdminWordleKeyboard() tgbotapi.InlineKeyboardMarkup {
	keyboard := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🔄 Переключить слово для всех", "admin:games:wordle:switch"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить слово", "admin:games:wordle:add"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "admin:games"),
		},
	}
	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

// GetAdminCrosswordKeyboard возвращает клавиатуру для управления кроссвордом
func GetAdminCrosswordKeyboard() tgbotapi.InlineKeyboardMarkup {
	keyboard := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить кроссворд", "admin:games:crossword:update"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить кроссворд", "admin:games:crossword:add"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "admin:games"),
		},
	}
	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

// InvitationInfoForKeyboard структура для передачи информации о приглашении в клавиатуру
type InvitationInfoForKeyboard struct {
	Name   string
	IsSent bool
}

// GetGuestsSelectionKeyboard создает клавиатуру с кнопками для выбора гостя
func GetGuestsSelectionKeyboard(invitations []InvitationInfoForKeyboard) tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(invitations); i += 2 {
		var row []tgbotapi.InlineKeyboardButton
		// Первая кнопка в ряду
		inv1 := invitations[i]
		buttonText1 := fmt.Sprintf("👤 %s", inv1.Name)
		if inv1.IsSent {
			buttonText1 = fmt.Sprintf("✅ %s", inv1.Name)
		}
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(buttonText1, fmt.Sprintf("admin:invite_guest:%d", i)))

		// Вторая кнопка в ряду (если есть)
		if i+1 < len(invitations) {
			inv2 := invitations[i+1]
			buttonText2 := fmt.Sprintf("👤 %s", inv2.Name)
			if inv2.IsSent {
				buttonText2 = fmt.Sprintf("✅ %s", inv2.Name)
			}
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(buttonText2, fmt.Sprintf("admin:invite_guest:%d", i+1)))
		}
		keyboard = append(keyboard, row)
	}

	// Кнопка возврата
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "admin:back"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

// GetGuestsSwapKeyboard создает клавиатуру с кнопками для выбора гостя для обмена имени/фамилии
func GetGuestsSwapKeyboard(guests []map[string]interface{}, page int) tgbotapi.InlineKeyboardMarkup {
	const itemsPerPage = 10
	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > len(guests) {
		end = len(guests)
	}

	var keyboard [][]tgbotapi.InlineKeyboardButton
	for i := start; i < end; i++ {
		guest := guests[i]
		rowNum, _ := guest["row"].(int)
		fullName, _ := guest["full_name"].(string)
		if fullName == "" {
			fullName = "Без имени"
		}

		buttonText := fmt.Sprintf("👤 %s", fullName)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("swapname:%d:%d", rowNum, page)),
		})
	}

	// Навигация по страницам
	var navRow []tgbotapi.InlineKeyboardButton
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("fixnames_page:%d", page-1)))
	}
	if end < len(guests) {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", fmt.Sprintf("fixnames_page:%d", page+1)))
	}
	if len(navRow) > 0 {
		keyboard = append(keyboard, navRow)
	}

	// Кнопка возврата
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "admin:back"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

// GetGroupManagementKeyboard возвращает inline клавиатуру для управления группой
func GetGroupManagementKeyboard() tgbotapi.InlineKeyboardMarkup {
	keyboard := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("👥 Посмотреть участников", "admin:group:list_members"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "admin:back"),
		},
	}
	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

// GetContactsInlineKeyboard возвращает inline клавиатуру для контактов
func GetContactsInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton
	if strings.HasPrefix(config.WebappURL, "https://") {
		webAppURL := config.WebappURL
		keyboard = [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonURL("💒 Открыть приглашение", webAppURL),
			},
		}
	} else {
		keyboard = [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonData("📱 Приглашение", "open_invitation"),
			},
		}
	}
	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}
