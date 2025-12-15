# План миграции на go-telegram-bot-api/v5

## Основные различия API

### telebot.v3 → go-telegram-bot-api/v5

1. **Инициализация бота**
   ```go
   // Старое (telebot.v3)
   bot, err := telebot.NewBot(telebot.Settings{...})
   
   // Новое (v5)
   bot, err := tgbotapi.NewBotAPI(token)
   ```

2. **Обработка обновлений**
   ```go
   // Старое
   bot.Handle("/start", handleStart)
   bot.Handle(telebot.OnText, handleText)
   
   // Новое
   u := tgbotapi.NewUpdate(0)
   updates := bot.GetUpdatesChan(u)
   for update := range updates {
       if update.Message != nil {
           // обработка
       }
   }
   ```

3. **Отправка сообщений**
   ```go
   // Старое
   c.Send("Hello")
   
   // Новое
   msg := tgbotapi.NewMessage(chatID, "Hello")
   bot.Send(msg)
   ```

4. **Клавиатуры**
   ```go
   // Старое
   keyboard := &telebot.ReplyMarkup{...}
   
   // Новое
   keyboard := tgbotapi.NewReplyKeyboard(...)
   ```

## Порядок миграции

1. ✅ Установка библиотеки
2. ⏳ Создание новой структуры bot.go
3. ⏳ Миграция handlers (по одному)
4. ⏳ Миграция keyboards
5. ⏳ Миграция callbacks
6. ⏳ Тестирование
7. ⏳ Удаление старой библиотеки

## Файлы для миграции

- `internal/bot/bot.go` - основная структура
- `internal/bot/handlers.go` - обработчики команд
- `internal/bot/admin_handlers.go` - админ обработчики
- `internal/bot/callbacks.go` - callback обработчики
- `internal/bot/broadcast_handlers.go` - рассылка
- `internal/keyboards/keyboards.go` - клавиатуры
- `cmd/server/main.go` - инициализация

