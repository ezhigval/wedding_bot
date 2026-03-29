package keyboards

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// GetAdminRootInlineKeyboard возвращает inline клавиатуру для корневого админ меню
func GetAdminRootInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Гости", "admin:guests"),
			tgbotapi.NewInlineKeyboardButtonData("🪑 Столы", "admin:seating"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Группа", "admin:group"),
			tgbotapi.NewInlineKeyboardButtonData("🤖 Бот", "admin:stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📨 Рассылка", "admin:broadcast"),
		),
	)
}

