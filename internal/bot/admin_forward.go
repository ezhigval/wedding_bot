package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"wedding-bot/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// GetAdminUserIDs возвращает список user_id администраторов
func GetAdminUserIDs() []int64 {
	// Получаем из конфига или можно хранить в БД
	// Пока используем hardcoded список, можно вынести в конфиг
	return []int64{
		// TODO: вынести в конфиг или БД
	}
}

// ForwardMessageToAdmins пересылает сообщение всем администраторам
func ForwardMessageToAdmins(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	userID := message.From.ID

	// Проверяем, что это не админ
	if isAdminUser(int(userID)) {
		return
	}

	// Получаем список админов
	adminIDs := GetAdminUserIDs()
	if len(adminIDs) == 0 {
		// Если список админов пустой, берем из конфига
		if adminIDStr := config.AdminUserID; adminIDStr != "" {
			if adminID, err := strconv.ParseInt(adminIDStr, 10, 64); err == nil {
				adminIDs = []int64{adminID}
			}
		}
	}

	// Формируем информацию о пользователе
	userInfo := fmt.Sprintf(
		"📩 <b>Сообщение от пользователя</b>\n\n"+
			"👤 <b>Имя:</b> %s %s\n"+
			"🆔 <b>User ID:</b> <code>%d</code>\n"+
			"👥 <b>Username:</b> @%s\n\n"+
			"💬 <b>Сообщение:</b>\n",
		message.From.FirstName,
		message.From.LastName,
		message.From.ID,
		message.From.UserName,
	)

	// Добавляем текст сообщения
	if message.Text != "" {
		userInfo += fmt.Sprintf("<code>%s</code>", message.Text)
	}

	// Если есть фото, добавляем информацию о нем
	if len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1]
		userInfo += fmt.Sprintf("\n\n📷 <b>Фото:</b> %dx%d (ID: %s)",
			photo.Width, photo.Height, photo.FileID)
	}

	// Если есть видео, добавляем информацию о нем
	if message.Video != nil {
		userInfo += fmt.Sprintf("\n\n🎥 <b>Видео:</b> %s (ID: %s)",
			message.Video.MimeType, message.Video.FileID)
	}

	// Если есть документ, добавляем информацию о нем
	if message.Document != nil {
		userInfo += fmt.Sprintf("\n\n📄 <b>Документ:</b> %s (ID: %s)",
			message.Document.FileName, message.Document.FileID)
	}

	// Отправляем сообщение каждому админу
	for _, adminID := range adminIDs {
		if adminID == userID {
			continue // Не отправляем самому себе
		}

		// Если есть фото, пересылаем его
		if len(message.Photo) > 0 {
			photo := message.Photo[len(message.Photo)-1]
			forwardedPhoto := tgbotapi.NewPhoto(adminID, tgbotapi.FileID(photo.FileID))
			forwardedPhoto.Caption = userInfo
			forwardedPhoto.ParseMode = tgbotapi.ModeHTML
			bot.Send(forwardedPhoto)
			continue
		}

		// Если есть видео, пересылаем его
		if message.Video != nil {
			forwardedVideo := tgbotapi.NewVideo(adminID, tgbotapi.FileID(message.Video.FileID))
			forwardedVideo.Caption = userInfo
			forwardedVideo.ParseMode = tgbotapi.ModeHTML
			bot.Send(forwardedVideo)
			continue
		}

		// Если есть документ, пересылаем его
		if message.Document != nil {
			forwardedDoc := tgbotapi.NewDocument(adminID, tgbotapi.FileID(message.Document.FileID))
			forwardedDoc.Caption = userInfo
			forwardedDoc.ParseMode = tgbotapi.ModeHTML
			bot.Send(forwardedDoc)
			continue
		}

		// Для текстовых сообщений
		msg := tgbotapi.NewMessage(adminID, userInfo)
		msg.ParseMode = tgbotapi.ModeHTML
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Ошибка пересылки сообщения админу %d: %v", adminID, err)
		}
	}

	// Отправляем подтверждение пользователю
	confirmMsg := tgbotapi.NewMessage(message.Chat.ID,
		"✅ Ваше сообщение отправлено администраторам")
	bot.Send(confirmMsg)
}

// IsAdminCommand проверяет, является ли сообщение командой админа
func IsAdminCommand(text string) bool {
	adminCommands := []string{
		"⚙️ Админ-панель",
		"👥 Гости",
		"🪑 Столы",
		"💬 Группа",
		"🤖 Бот",
		"🎮 Игры",
		"📨 Рассылка",
		"⬅️ Вернуться",
		"📋 Список гостей",
		"📊 Посмотреть рассадку",
		"🔄 Обновить рассадку",
		"📤 Отправить приглашение",
		"🔁 Исправление Имя/Фамилия",
		"Рассылка в ЛС",
	}

	for _, cmd := range adminCommands {
		if strings.TrimSpace(text) == cmd {
			return true
		}
	}
	return false
}
