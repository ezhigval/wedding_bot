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
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Остановка обработки обновлений бота")
			bot.StopReceivingUpdates()
			return
		case update := <-updates:
			// Обрабатываем обновление в отдельной горутине для параллелизма
			go handleUpdate(bot, update)
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

	// Проверяем, является ли пользователь админом для обработки admin команд
	if isAdminUser(int(userID)) {
		// Проверяем, есть ли активная рассылка
		state := GetBroadcastState(userID)
		if state != nil {
			// Обрабатываем текст/фото/кнопку для рассылки
			if state.Text == "" {
				// Ожидаем текст
				handleBroadcastText(bot, message, text)
				return
			} else if state.PhotoID == "" && len(message.Photo) > 0 {
				// Обрабатываем фото
				photoID := message.Photo[len(message.Photo)-1].FileID
				handleBroadcastPhoto(bot, message, photoID)
				return
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
						// Отправляем сообщение о добавлении кнопки
						msg := tgbotapi.NewMessage(message.Chat.ID, "✅ Кнопка добавлена. Нажмите 'Без картинки' или отправьте фото для завершения настройки.")
						bot.Send(msg)
						return
					}
				}
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

	// Проверяем, есть ли активная рассылка
	if isAdminUser(int(userID)) {
		state := GetBroadcastState(userID)
		if state != nil && state.Text != "" && state.PhotoID == "" {
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

		registered, err := google_sheets.CheckGuestRegistration(ctx, int(userID))
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
