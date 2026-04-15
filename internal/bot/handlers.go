package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
	"wedding-bot/internal/keyboards"
)

// handleStartCommand обрабатывает команду /start
func handleStartCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	user := message.From

	// Получаем имя пользователя
	displayName := getUserDisplayName(user)

	// Пытаемся обновить user_id в таблице приглашений
	fullName := displayName
	if user.FirstName != "" && user.LastName != "" {
		fullName = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	} else if user.FirstName != "" {
		fullName = user.FirstName
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Обновляем user_id в приглашениях (если есть)
	username := ""
	if user.UserName != "" {
		username = user.UserName
	}
	var usernamePtr *string
	if username != "" {
		usernamePtr = &username
	}
	_ = google_sheets.UpdateInvitationUserID(ctx, fullName, int(user.ID), usernamePtr)

	// Проверяем, является ли пользователь админом
	isAdmin := isAdminUser(int(user.ID))

	// Отправляем приветственное сообщение
	msgText := fmt.Sprintf("👋 Привет, %s!", displayName)

	keyboard := keyboards.GetMainReplyKeyboard(isAdmin)

	// Пытаемся отправить фото
	if config.PhotoPath != "" {
		photo := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FilePath(config.PhotoPath))
		photo.Caption = msgText
		photo.ReplyMarkup = keyboard
		if _, err := bot.Send(photo); err != nil {
			log.Printf("⚠️ Не удалось отправить фото: %v", err)
			// Отправляем только текст
			msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
			msg.ReplyMarkup = keyboard
			bot.Send(msg)
			return
		}
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// handleHelpCommand обрабатывает команду /help
func handleHelpCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msgText := "Помощь по использованию бота:\n\n" +
		"/start - Начать работу с ботом\n" +
		"/menu - Открыть меню\n" +
		"/admin - Админ панель (только для админов)"

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	bot.Send(msg)
}

// handleMenuCommand обрабатывает команду /menu
func handleMenuCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	keyboard := keyboards.GetInvitationKeyboard()

	msg := tgbotapi.NewMessage(message.Chat.ID, "Меню бота:")
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// handleAdminCommand обрабатывает команду /admin
func handleAdminCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	handleAdminPanel(bot, message)
}

// handleAdminPanel показывает админ панель
func handleAdminPanel(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msgText := "🛠 <b>Админ панель</b>\n\nВыберите раздел:"
	keyboard := keyboards.GetAdminRootReplyKeyboard()

	SetAdminNav(message.From.ID, AdminNavRoot)

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// getUserDisplayName получает имя пользователя
func getUserDisplayName(user *tgbotapi.User) string {
	if user.FirstName != "" {
		if user.LastName != "" {
			return fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		}
		return user.FirstName
	}
	return "друг"
}

// isAdminUser проверяет, является ли пользователь админом
func isAdminUser(userID int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	admins, err := google_sheets.GetAdminsList(ctx)
	if err != nil {
		log.Printf("Ошибка получения списка админов: %v", err)
		return false
	}

	for _, admin := range admins {
		if admin.UserID != nil && *admin.UserID == userID {
			return true
		}
	}

	return false
}
