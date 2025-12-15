package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"gopkg.in/telebot.v3"

	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
	"wedding-bot/internal/keyboards"
)

// handleStart обрабатывает команду /start
func handleStart(c telebot.Context) error {
	user := c.Sender()

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
	if user.Username != "" {
		username = user.Username
	}
	var usernamePtr *string
	if username != "" {
		usernamePtr = &username
	}
	_ = google_sheets.UpdateInvitationUserID(ctx, fullName, int(user.ID), usernamePtr)

	// Проверяем, является ли пользователь админом
	isAdmin := isAdminUser(int(user.ID))

	// Отправляем приветственное сообщение
	message := fmt.Sprintf("👋 Привет, %s!", displayName)

	keyboard := keyboards.GetMainReplyKeyboard(isAdmin, IsPhotoModeEnabled(user.ID))

	// Пытаемся отправить фото
	if config.PhotoPath != "" {
		photo := &telebot.Photo{File: telebot.FromDisk(config.PhotoPath)}
		if err := c.Send(photo, message, keyboard); err != nil {
			log.Printf("⚠️ Не удалось отправить фото: %v", err)
			// Отправляем только текст
			return c.Send(message, keyboard)
		}
		return nil
	}

	return c.Send(message, keyboard)
}

// handleText обрабатывает текстовые сообщения
func handleText(c telebot.Context) error {
	text := c.Text()
	userID := c.Sender().ID

	// Проверяем фоторежим
	if text == "📸 Фоторежим ❌" || text == "📸 Фоторежим ✅" {
		return handleTogglePhotoMode(c)
	}

	// Проверяем другие кнопки
	if text == "💬 Общий чат" {
		keyboard := keyboards.GetContactsInlineKeyboard()
		return c.Send("Перейдите в общий чат:", keyboard)
	}

	if text == "📞 Связаться с нами" {
		keyboard := keyboards.GetContactsInlineKeyboard()
		return c.Send("Свяжитесь с организаторами:", keyboard)
	}

	if text == "🛠 Админ-панель" {
		if !isAdminUser(int(userID)) {
			return c.Send("❌ У вас нет прав администратора")
		}
		return handleAdminPanel(c)
	}

	// Проверяем, является ли пользователь админом для обработки admin команд
	if isAdminUser(int(userID)) {
		// Проверяем, есть ли активная рассылка
		state := GetBroadcastState(userID)
		if state != nil {
			// Обрабатываем текст/фото/кнопку для рассылки
			if state.Text == "" {
				// Ожидаем текст
				return handleBroadcastText(c, text)
			} else if state.PhotoID == "" {
				// Можем обработать фото или пропустить
				// Продолжаем к обычной обработке
			} else if state.ButtonText == "" {
				// Ожидаем выбор кнопки или текст кнопки
				// Проверяем, не является ли это текстом кнопки
				if strings.Contains(text, "|") {
					// Формат: "Текст|URL"
					parts := strings.SplitN(text, "|", 2)
					if len(parts) == 2 {
						state.ButtonText = strings.TrimSpace(parts[0])
						state.ButtonURL = strings.TrimSpace(parts[1])
						return showBroadcastPreview(c, state)
					}
				}
			}
		}

		return handleAdminText(c)
	}

	return nil
}

// handleTogglePhotoMode обрабатывает переключение фоторежима
func handleTogglePhotoMode(c telebot.Context) error {
	userID := c.Sender().ID
	isAdminUser := isAdminUser(int(userID))

	enabled := IsPhotoModeEnabled(userID)

	if enabled {
		// Выключаем фоторежим
		SetPhotoModeEnabled(userID, false)
		keyboard := keyboards.GetMainReplyKeyboard(isAdminUser, false)
		return c.Send(
			"📸 Фоторежим <b>выключен</b>.\nФото больше не собираются автоматически.",
			keyboard,
			telebot.ModeHTML,
		)
	}

	// Включаем фоторежим - проверяем регистрацию
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	registered, err := google_sheets.CheckGuestRegistration(ctx, int(userID))
	if err != nil {
		log.Printf("Ошибка проверки регистрации: %v", err)
		// В случае ошибки разрешаем включить
	}

	if !registered {
		keyboard := keyboards.GetMainReplyKeyboard(isAdminUser, false)
		return c.Send(
			"⚠️ Для использования фоторежима необходимо подтвердить ваше присутствие.\nИспользуйте Mini App для регистрации.",
			keyboard,
		)
	}

	SetPhotoModeEnabled(userID, true)
	keyboard := keyboards.GetMainReplyKeyboard(isAdminUser, true)
	return c.Send(
		"📸 Фоторежим <b>включен</b>.\nПросто отправляйте фото в этот чат — я всё соберу в свадебный альбом! 🙌",
		keyboard,
		telebot.ModeHTML,
	)
}

// handlePhoto обрабатывает фото
func handlePhoto(c telebot.Context) error {
	photo := c.Message().Photo
	if photo == nil {
		return nil
	}

	userID := c.Sender().ID

	// Проверяем, включен ли фоторежим
	if !IsPhotoModeEnabled(userID) {
		// Проверяем, зарегистрирован ли пользователь
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		registered, err := google_sheets.CheckGuestRegistration(ctx, int(userID))
		if err != nil {
			log.Printf("Ошибка проверки регистрации: %v", err)
		}

		if registered {
			// Пользователь зарегистрирован, но фоторежим не включен
			isAdmin := isAdminUser(int(userID))
			keyboard := keyboards.GetMainReplyKeyboard(isAdmin, false)
			return c.Send(
				"📸 Чтобы сохранить фото в свадебный альбом, включите фоторежим.\nНажмите кнопку «📸 Фоторежим ❌» в меню.",
				keyboard,
			)
		} else {
			// Пользователь не зарегистрирован
			return c.Send(
				"📸 Для сохранения фото в свадебный альбом необходимо подтвердить ваше присутствие.\nИспользуйте Mini App для регистрации.",
			)
		}
	}

	// Сохраняем фото
	user := c.Sender()
	displayName := getUserDisplayName(user)

	username := ""
	if user.Username != "" {
		username = "@" + user.Username
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fileID := photo.FileID
	if err := google_sheets.SavePhotoFromUser(ctx, int(userID), &username, displayName, fileID); err != nil {
		log.Printf("Ошибка сохранения фото: %v", err)
		return c.Send("❌ Ошибка сохранения фото")
	}

	return c.Send("✅ Фото сохранено! 📸")
}

// handleAdminPanel показывает админ панель
func handleAdminPanel(c telebot.Context) error {
	message := "🛠 <b>Админ панель</b>\n\n" +
		"Выберите раздел:"

	keyboard := keyboards.GetAdminRootReplyKeyboard()

	return c.Send(message, keyboard, telebot.ModeHTML)
}

// getUserDisplayName получает имя пользователя
func getUserDisplayName(user *telebot.User) string {
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

