package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"

	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
)

var (
	botInstance *telebot.Bot
	mu          sync.RWMutex
)

// InitBot инициализирует Telegram бота
func InitBot(ctx context.Context) (*telebot.Bot, error) {
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

	// Middleware для обработки паник
	bot.Use(func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("🚨 Паника в handler бота: %v", r)
					// Отправляем сообщение об ошибке пользователю
					c.Send("Произошла ошибка. Попробуйте позже.")
				}
			}()
			return next(c)
		}
	})

	// Регистрируем handlers
	registerHandlers(bot)

	log.Println("✅ Бот инициализирован успешно")
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

	// Обработчики событий (должны быть после команд)
	bot.Handle(telebot.OnText, handleText)
	bot.Handle(telebot.OnPhoto, handlePhoto)
	bot.Handle(telebot.OnCallback, handleCallback)
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

	if !isAdminUser(int(userID)) {
		return c.Send("❌ У вас нет прав администратора")
	}

	return handleAdminPanel(c)
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

				userID := int64(adminID)
				if _, err := bot.Send(&telebot.User{ID: userID}, message); err != nil {
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

