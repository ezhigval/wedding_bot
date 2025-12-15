package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"

	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
)

var (
	botInstance *telebot.Bot
)

// InitBot инициализирует Telegram бота
func InitBot(ctx context.Context) (*telebot.Bot, error) {
	if config.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN не установлен")
	}

	pref := telebot.Settings{
		Token:  config.BotToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания бота: %w", err)
	}

	botInstance = bot

	// Middleware для логирования
	bot.Use(middleware.Logger())

	// Регистрируем handlers
	registerHandlers(bot)

	log.Println("Бот инициализирован успешно")
	return bot, nil
}

// registerHandlers регистрирует все handlers бота
func registerHandlers(bot *telebot.Bot) {
	// Команды
	bot.Handle("/start", handleStart)
	bot.Handle("/help", handleHelp)
	bot.Handle("/menu", handleMenu)

	// Админ команды
	bot.Handle("/admin", handleAdmin)
	bot.Handle(telebot.OnText, handleText)
	bot.Handle(telebot.OnPhoto, handlePhoto)
	bot.Handle(telebot.OnCallback, handleCallback)
}

// handleStart обрабатывает команду /start
func handleStart(c telebot.Context) error {
	user := c.Sender()
	
	message := fmt.Sprintf(
		"Привет, %s! 👋\n\n"+
			"Добро пожаловать на свадьбу %s и %s! 💒\n\n"+
			"Используй кнопку ниже, чтобы открыть Mini App и зарегистрироваться.",
		user.FirstName,
		config.GroomName,
		config.BrideName,
	)

	keyboard := &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "Открыть Mini App",
					WebApp: &telebot.WebApp{
						URL: config.WebappURL,
					},
				},
			},
		},
	}

	return c.Send(message, keyboard)
}

// handleHelp обрабатывает команду /help
func handleHelp(c telebot.Context) error {
	message := "Помощь по использованию бота:\n\n" +
		"/start - Начать работу с ботом\n" +
		"/menu - Открыть меню\n" +
		"/admin - Админ панель (только для админов)"

	return c.Send(message)
}

// handleMenu обрабатывает команду /menu
func handleMenu(c telebot.Context) error {
	keyboard := &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "Открыть Mini App",
					WebApp: &telebot.WebApp{
						URL: config.WebappURL,
					},
				},
			},
			{
				telebot.InlineButton{
					Text: "Ссылка на группу",
					URL:   config.GroupLink,
				},
			},
		},
	}

	message := "Меню бота:"
	return c.Send(message, keyboard)
}

// handleAdmin обрабатывает команду /admin
func handleAdmin(c telebot.Context) error {
	userID := c.Sender().ID

	// Проверяем, является ли пользователь админом
	ctx := context.Background()
	admins, err := google_sheets.GetAdminsList(ctx)
	if err != nil {
		log.Printf("Error getting admins: %v", err)
		return c.Send("Ошибка проверки прав доступа")
	}

	isAdmin := false
	for _, admin := range admins {
		if admin.UserID != nil && *admin.UserID == int(userID) {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		return c.Send("У вас нет прав администратора")
	}

	// TODO: Показать админ панель
	message := "Админ панель:\n\n" +
		"Доступные команды:\n" +
		"/admin_guests - Управление гостями\n" +
		"/admin_games - Управление играми"

	return c.Send(message)
}

// handleText обрабатывает текстовые сообщения
func handleText(c telebot.Context) error {
	// TODO: Реализовать обработку текстовых сообщений
	return nil
}

// handlePhoto обрабатывает фото
func handlePhoto(c telebot.Context) error {
	photo := c.Message().Photo
	if photo == nil {
		return nil
	}

	user := c.Sender()
	ctx := context.Background()

	// Сохраняем фото в Google Sheets
	fullName := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	username := user.Username
	if username != "" {
		username = "@" + username
	}

	// Получаем file_id
	fileID := photo.FileID

	// TODO: Получить file_id и сохранить
	// Пока просто логируем
	log.Printf("Photo received from user_id=%d, username=%s, file_id=%s", user.ID, username, fileID)

	// Сохраняем в Google Sheets
	if err := google_sheets.SavePhotoFromUser(ctx, int(user.ID), &username, fullName, fileID); err != nil {
		log.Printf("Error saving photo: %v", err)
		return c.Send("Ошибка сохранения фото")
	}

	return c.Send("Фото сохранено! 📸")
}

// handleCallback обрабатывает callback queries
func handleCallback(c telebot.Context) error {
	// TODO: Реализовать обработку callback queries
	return c.Answer(&telebot.QueryResponse{})
}

// NotifyAdmins отправляет уведомление всем админам
func NotifyAdmins(message string) error {
	if botInstance == nil {
		return fmt.Errorf("бот не инициализирован")
	}

	ctx := context.Background()
	admins, err := google_sheets.GetAdminsList(ctx)
	if err != nil {
		return fmt.Errorf("ошибка получения списка админов: %w", err)
	}

	for _, admin := range admins {
		if admin.UserID != nil {
			userID := int64(*admin.UserID)
			if _, err := botInstance.Send(&telebot.User{ID: userID}, message); err != nil {
				log.Printf("Ошибка отправки уведомления админу %d: %v", *admin.UserID, err)
			}
		}
	}

	return nil
}

