package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
	"wedding-bot/internal/keyboards"
)

var (
	botInstance *tgbotapi.BotAPI
	mu          sync.RWMutex
)

// InitBot инициализирует Telegram бота
func InitBot(ctx context.Context) (*tgbotapi.BotAPI, error) {
	if config.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN не установлен")
	}

	// Проверяем, не инициализирован ли уже бот
	mu.RLock()
	if botInstance != nil {
		mu.RUnlock()
		return botInstance, nil
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()

	// Двойная проверка
	if botInstance != nil {
		return botInstance, nil
	}

	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания бота: %w", err)
	}

	botInstance = bot

	// Настраиваем бота
	bot.Debug = false

	log.Printf("✅ Бот инициализирован: @%s", bot.Self.UserName)

	// Запускаем обработку обновлений в отдельной горутине
	go startUpdateHandler(ctx, bot)

	log.Println("✅ Бот инициализирован успешно")
	return bot, nil
}

// startUpdateHandler обрабатывает обновления от Telegram
func startUpdateHandler(ctx context.Context, bot *tgbotapi.BotAPI) {
	log.Println("🔄 Запуск обработчика обновлений Telegram...")

	// Сбрасываем вебхук, если он был установлен (чтобы избежать конфликтов)
	log.Println("🔧 Проверка и сброс вебхука...")
	deleteWebhookConfig := tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false}
	if _, err := bot.Request(deleteWebhookConfig); err != nil {
		log.Printf("⚠️ Ошибка сброса вебхука (может быть не установлен): %v", err)
	} else {
		log.Println("✅ Вебхук сброшен (если был установлен)")
	}

	// Задержка перед началом polling, чтобы старый процесс успел завершиться
	// Это особенно важно при деплое на Render, где старый контейнер может еще работать
	// Telegram API может держать соединение открытым до 60 секунд (timeout)
	// Увеличиваем до 20 секунд для большей надежности
	log.Println("⏳ Ожидание 20 секунд перед началом polling (для завершения старого процесса)...")
	select {
	case <-ctx.Done():
		return
	case <-time.After(20 * time.Second):
	}
	log.Println("✅ Начинаем polling обновлений...")

	// Используем явный polling вместо GetUpdatesChan для полного контроля
	// GetUpdatesChan может создавать множественные соединения при ошибках
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Отслеживаем последний обработанный update ID
	lastUpdateID := 0

	retryDelay := 3 * time.Second
	maxRetryDelay := 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Остановка обработки обновлений бота")
			bot.StopReceivingUpdates()
			return
		default:
			// Устанавливаем offset для получения только новых обновлений
			u.Offset = lastUpdateID

			// Получаем обновления явно через Request
			updates, err := bot.GetUpdates(u)
			if err != nil {
				// Обрабатываем ошибки
				if strings.Contains(err.Error(), "Conflict") {
					// При конфликте увеличиваем задержку более агрессивно
					// Это означает, что другой процесс все еще активен
					log.Printf("⚠️ Conflict detected: %v. Ожидание %v перед повтором...", err, retryDelay)
					if retryDelay < maxRetryDelay {
						retryDelay = time.Duration(float64(retryDelay) * 1.5)
					}
				} else {
					log.Printf("⚠️ Ошибка получения обновлений: %v. Повтор через %v...", err, retryDelay)
					// Для других ошибок увеличиваем задержку медленнее
					if retryDelay < maxRetryDelay {
						retryDelay = time.Duration(float64(retryDelay) * 1.2)
					}
				}

				// Exponential backoff
				select {
				case <-ctx.Done():
					return
				case <-time.After(retryDelay):
				}
				continue
			}

			// Сбрасываем задержку при успешном получении
			retryDelay = 3 * time.Second

			// Обрабатываем все полученные обновления
			for _, update := range updates {
				// Обновляем lastUpdateID
				if update.UpdateID >= lastUpdateID {
					lastUpdateID = update.UpdateID + 1
				}

				// Обрабатываем обновление в отдельной горутине для параллелизма
				go handleUpdate(bot, update)
			}

			// Небольшая задержка между запросами, если обновлений нет
			if len(updates) == 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(100 * time.Millisecond):
				}
			}
		}
	}
}

// handleUpdate обрабатывает одно обновление
func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🚨 Паника в handler бота: %v", r)
		}
	}()

	// Обработка команд
	if update.Message != nil && update.Message.IsCommand() {
		handleCommand(bot, update.Message)
		return
	}

	// Обработка текстовых сообщений
	if update.Message != nil && update.Message.Text != "" {
		handleMessage(bot, update.Message)
		return
	}

	// Обработка фото
	if update.Message != nil && update.Message.Photo != nil && len(update.Message.Photo) > 0 {
		handlePhotoMessage(bot, update.Message)
		return
	}

	// Обработка видео
	if update.Message != nil && update.Message.Video != nil {
		handleVideoMessage(bot, update.Message)
		return
	}

	// Обработка документов
	if update.Message != nil && update.Message.Document != nil {
		handleDocumentMessage(bot, update.Message)
		return
	}

	// Обработка callback queries
	if update.CallbackQuery != nil {
		handleCallbackQuery(bot, update.CallbackQuery)
		return
	}
}

// handleCommand обрабатывает команды
func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	command := message.Command()
	userID := message.From.ID

	switch command {
	case "start":
		handleStartCommand(bot, message)
	case "help":
		handleHelpCommand(bot, message)
	case "menu":
		handleMenuCommand(bot, message)
	case "admin":
		if isAdminUser(int(userID)) {
			handleAdminCommand(bot, message)
		} else {
			msg := tgbotapi.NewMessage(message.Chat.ID, "❌ У вас нет прав администратора")
			bot.Send(msg)
		}
	}
}

// handleMessage обрабатывает текстовые сообщения
func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	text := message.Text
	userID := message.From.ID

	// Обработка режимов ввода администратора (Wordle/Crossword/группа)
	if isAdminUser(int(userID)) {
		switch GetAdminInputMode(userID) {
		case AdminInputModeWordleAdd:
			handleWordleAddInput(bot, message)
			return
		case AdminInputModeCrosswordAdd:
			handleCrosswordAddInput(bot, message)
			return
		case AdminInputModeGroupBroadcast:
			handleGroupBroadcastInput(bot, message)
			return
		}
	}

	// Проверяем фоторежим
	if text == "📸 Фоторежим ❌" || text == "📸 Фоторежим ✅" {
		handleTogglePhotoMode(bot, message)
		return
	}

	// Открыть приглашение (fallback для кнопки в reply keyboard)
	if text == "💒 Открыть приглашение" {
		keyboard := keyboards.GetInvitationKeyboard()
		msg := tgbotapi.NewMessage(message.Chat.ID, "Открыть приглашение:")
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
		return
	}

	// Проверяем другие кнопки
	if text == "💬 Общий чат" {
		keyboard := keyboards.GetGroupLinkKeyboard()
		msg := tgbotapi.NewMessage(message.Chat.ID, "Перейдите в общий чат:")
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
		return
	}

	if text == "📞 Связаться с нами" {
		keyboard := keyboards.GetContactsInlineKeyboard()
		msg := tgbotapi.NewMessage(message.Chat.ID, "Свяжитесь с организаторами:")
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
		return
	}

	if text == "💒 Открыть приглашение" {
		keyboard := keyboards.GetInvitationKeyboard()
		msg := tgbotapi.NewMessage(message.Chat.ID, "Открыть приглашение:")
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
		return
	}

	if text == "⚙️ Админ-панель" {
		if !isAdminUser(int(userID)) {
			msg := tgbotapi.NewMessage(message.Chat.ID, "❌ У вас нет прав администратора")
			bot.Send(msg)
			return
		}
		handleAdminPanel(bot, message)
		return
	}

	// Если это не админ и не команда админа, пересылаем сообщение админам
	if !isAdminUser(int(userID)) && !IsAdminCommand(text) {
		ForwardMessageToAdmins(bot, message)
		return
	}

	// Проверяем, является ли пользователь админом для обработки admin команд
	if isAdminUser(int(userID)) {
		// Проверяем, есть ли активная рассылка
		state := GetBroadcastState(userID)
		if state != nil {
			switch state.Step {
			case "text":
				// Ожидаем текст
				handleBroadcastText(bot, message, text)
				return
			case "media":
				// Ожидаем фото или видео
				if len(message.Photo) > 0 {
					photoID := message.Photo[len(message.Photo)-1].FileID
					handleBroadcastPhoto(bot, message, photoID)
					return
				} else if message.Video != nil {
					handleBroadcastVideo(bot, message, message.Video.FileID)
					return
				}
				// Игнорируем текст на этом шаге
				return
			case "custom_button":
				// Ожидаем текст кастомной кнопки
				if strings.Contains(text, "|") {
					parts := strings.SplitN(text, "|", 2)
					if len(parts) == 2 {
						buttonText := strings.TrimSpace(parts[0])
						buttonURL := strings.TrimSpace(parts[1])

						// Базовая валидация URL
						if !strings.HasPrefix(buttonURL, "http://") && !strings.HasPrefix(buttonURL, "https://") {
							msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Неверный URL. Используйте формат: <code>Текст кнопки|https://example.com</code>")
							msg.ParseMode = tgbotapi.ModeHTML
							bot.Send(msg)
							return
						}

						if buttonText == "" {
							msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Текст кнопки не может быть пустым")
							bot.Send(msg)
							return
						}

						state.ButtonText = buttonText
						state.ButtonURL = buttonURL
						state.Step = "recipients"
						showRecipientsSelectionFromMessage(bot, message, state)
						return
					}
				}
				// Неверный формат
				msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Неверный формат. Используйте: <code>Текст кнопки|URL</code>")
				msg.ParseMode = tgbotapi.ModeHTML
				bot.Send(msg)
				return
			}
		}

		handleAdminText(bot, message)
		return
	}
}

// handlePhotoMessage обрабатывает фото
func handlePhotoMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if len(message.Photo) == 0 {
		return
	}

	userID := message.From.ID

	// Если это не админ, пересылаем фото админам
	if !isAdminUser(int(userID)) {
		ForwardMessageToAdmins(bot, message)
		return
	}

	// Проверяем, есть ли активная рассылка
	if isAdminUser(int(userID)) {
		state := GetBroadcastState(userID)
		if state != nil && state.Step == "media" {
			// Обрабатываем фото для рассылки
			photoID := message.Photo[len(message.Photo)-1].FileID
			handleBroadcastPhoto(bot, message, photoID)
			return
		}
	}

	// Проверяем, включен ли фоторежим
	if !IsPhotoModeEnabled(userID) {
		// Проверяем, зарегистрирован ли пользователь
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		username := ""
		if message.From != nil {
			username = message.From.UserName
		}

		registered, err := google_sheets.CheckGuestRegistrationByIdentifier(ctx, int(userID), username)
		if err != nil {
			log.Printf("Ошибка проверки регистрации: %v", err)
		}

		if registered {
			// Пользователь зарегистрирован, но фоторежим не включен
			isAdmin := isAdminUser(int(userID))
			keyboard := keyboards.GetMainReplyKeyboard(isAdmin, false)
			msg := tgbotapi.NewMessage(message.Chat.ID, "📸 Чтобы сохранить фото в свадебный альбом, включите фоторежим.\nНажмите кнопку «📸 Фоторежим ❌» в меню.")
			msg.ReplyMarkup = keyboard
			bot.Send(msg)
			return
		} else {
			// Пользователь не зарегистрирован
			msg := tgbotapi.NewMessage(message.Chat.ID, "📸 Для сохранения фото в свадебный альбом необходимо подтвердить ваше присутствие.\nИспользуйте Mini App для регистрации.")
			bot.Send(msg)
			return
		}
	}

	// Сохраняем фото
	displayName := getUserDisplayName(message.From)

	username := ""
	if message.From.UserName != "" {
		username = "@" + message.From.UserName
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Берем самое большое фото
	photo := message.Photo[len(message.Photo)-1]
	fileID := photo.FileID

	if err := google_sheets.SavePhotoFromUser(ctx, int(userID), &username, displayName, fileID); err != nil {
		log.Printf("Ошибка сохранения фото: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка сохранения фото")
		bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "✅ Фото сохранено! 📸")
	bot.Send(msg)
}

// handleVideoMessage обрабатывает видео
func handleVideoMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if message.Video == nil {
		return
	}

	userID := message.From.ID

	// Если это не админ, пересылаем видео админам
	if !isAdminUser(int(userID)) {
		ForwardMessageToAdmins(bot, message)
		return
	}

	// Проверяем, есть ли активная рассылка
	if isAdminUser(int(userID)) {
		state := GetBroadcastState(userID)
		if state != nil && state.Step == "media" {
			// Обрабатываем видео для рассылки
			handleBroadcastVideo(bot, message, message.Video.FileID)
			return
		}
	}

	// Для обычных пользователей можно добавить обработку видео в будущем
	msg := tgbotapi.NewMessage(message.Chat.ID, "📹 Видео получено!")
	bot.Send(msg)
}

// handleDocumentMessage обрабатывает документы
func handleDocumentMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if message.Document == nil {
		return
	}

	userID := message.From.ID

	// Если это не админ, пересылаем документ админам
	if !isAdminUser(int(userID)) {
		ForwardMessageToAdmins(bot, message)
		return
	}

	// Для админов можно добавить обработку документов в будущем
	msg := tgbotapi.NewMessage(message.Chat.ID, "📄 Документ получен!")
	bot.Send(msg)
}

// handleCallbackQuery обрабатывает callback queries
func handleCallbackQuery(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	// Парсим callback data
	parts := strings.Split(data, ":")
	if len(parts) == 0 {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	action := parts[0]

	switch action {
	case "admin":
		handleAdminCallback(bot, callback, parts[1:])
	case "invite":
		handleInvitationCallback(bot, callback, parts[1:])
	case "game":
		handleGameAdminCallback(bot, callback, parts[1:])
	case "group":
		handleGroupCallback(bot, callback, parts[1:])
	case "admin_wordle":
		handleWordleAdminCallback(bot, callback)
	case "admin_crossword":
		handleCrosswordAdminCallback(bot, callback)
	case "admin_back":
		handleAdminBackCallback(bot, callback)
	case "swapname":
		handleSwapNameCallback(bot, callback, parts[1:])
	case "fixnames_page":
		handleFixNamesPageCallback(bot, callback, parts[1:])
	case "delete_guest":
		handleDeleteGuestCallback(bot, callback, parts[1:])
	case "broadcast":
		handleBroadcastCallback(bot, callback, parts[1:])
	default:
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

// NotifyAdmins отправляет уведомление всем админам
func NotifyAdmins(message string) error {
	mu.RLock()
	bot := botInstance
	mu.RUnlock()

	if bot == nil {
		return fmt.Errorf("бот не инициализирован")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admins, err := google_sheets.GetAdminsList(ctx)
	if err != nil {
		return fmt.Errorf("ошибка получения списка админов: %w", err)
	}

	var wg sync.WaitGroup
	errorChan := make(chan error, len(admins))

	for _, admin := range admins {
		if admin.UserID != nil {
			wg.Add(1)
			go func(adminID int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						log.Printf("🚨 Паника при отправке уведомления админу %d: %v", adminID, r)
					}
				}()

				msg := tgbotapi.NewMessage(int64(adminID), message)
				msg.ParseMode = tgbotapi.ModeHTML
				if _, err := bot.Send(msg); err != nil {
					log.Printf("⚠️ Ошибка отправки уведомления админу %d: %v", adminID, err)
					errorChan <- err
				}
			}(*admin.UserID)
		}
	}

	wg.Wait()
	close(errorChan)

	// Проверяем, были ли ошибки
	hasErrors := false
	for err := range errorChan {
		if err != nil {
			hasErrors = true
		}
	}

	if hasErrors {
		return fmt.Errorf("некоторые уведомления не были отправлены")
	}

	return nil
}
