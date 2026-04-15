package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"

	"wedding-bot/internal/api"
	"wedding-bot/internal/bot"
	"wedding-bot/internal/cache"
	"wedding-bot/internal/config"
	"wedding-bot/internal/daily_reset"
	"wedding-bot/internal/google_sheets"
)

var (
	server      *http.Server
	telegramBot *tgbotapi.BotAPI
	wg          sync.WaitGroup
)

func main() {
	// Инициализируем структурированное логирование
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if config.IsDebug() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Обработка паник
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🚨 КРИТИЧЕСКАЯ ОШИБКА (panic): %v", r)
			os.Exit(1)
		}
	}()

	log.Println("=" + strings.Repeat("=", 59))
	log.Println("🚀 ЗАПУСК СВАДЕБНОГО БОТА (GO)")
	log.Println("=" + strings.Repeat("=", 59))
	log.Printf("🆔 Process ID: %d", os.Getpid())
	log.Printf("🕐 Время: %s", time.Now().Format(time.RFC3339))
	log.Printf("🌍 PORT: %s", os.Getenv("PORT"))
	log.Println("=" + strings.Repeat("=", 59))

	// Создаем контекст с отменой для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Загружаем конфигурацию
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("❌ Ошибка загрузки конфигурации: %v", err)
	}

	log.Println("✅ Переменные окружения проверены")
	log.Printf("🌐 Порт: %s", os.Getenv("PORT"))
	log.Println("=" + strings.Repeat("=", 59))

	// Инициализируем кэш
	cache.InitMemoryCache()
	if err := cache.InitGameStatsCache(); err != nil {
		log.Printf("⚠️ Ошибка инициализации кэша: %v (продолжаем без кэша)", err)
	} else {
		log.Println("✅ Кэш игровой статистики инициализирован")
	}

	// Инициализируем Google Sheets
	log.Println("🔧 Инициализация Google Sheets...")
	if err := google_sheets.EnsureRequiredSheets(ctx); err != nil {
		log.Printf("❌ Ошибка инициализации Google Sheets: %v", err)
		log.Printf("💡 Возможные причины:")
		log.Printf("   1. Неправильные credentials (проверь GOOGLE_SHEETS_CREDENTIALS)")
		log.Printf("   2. Сервисный аккаунт не имеет доступа к таблице")
		log.Printf("   3. Google Sheets API не включен в проекте")
		log.Printf("   4. Неправильный GOOGLE_SHEETS_ID")
	} else {
		log.Println("✅ Google Sheets инициализирован")
	}

	// Проверяем структуру листа гостей
	log.Println("🔍 Проверка структуры листа гостей...")
	if err := google_sheets.ValidateGuestSheetStructure(ctx); err != nil {
		log.Printf("⚠️ Ошибка проверки структуры листа гостей: %v", err)
	} else {
		log.Println("✅ Структура листа гостей проверена")
	}

	// Проверяем служебные листы, которыми управляет само приложение
	log.Println("🔍 Проверка служебных листов...")
	if err := google_sheets.ValidateCoreSheetsStructure(ctx); err != nil {
		log.Printf("⚠️ Ошибка проверки служебных листов: %v", err)
	} else {
		log.Println("✅ Служебные листы проверены")
	}

	// Инициализируем API
	apiRouter, err := api.InitAPI(ctx)
	if err != nil {
		log.Fatalf("❌ Ошибка инициализации API: %v", err)
	}
	log.Println("✅ API инициализирован")

	// Инициализируем бота
	var botErr error
	telegramBot, botErr = bot.InitBot(ctx)
	if botErr != nil {
		log.Printf("⚠️ Ошибка инициализации бота: %v", botErr)
		log.Println("Бот не будет работать, но API продолжит функционировать")
	} else {
		log.Println("✅ Telegram бот инициализирован")
		// Устанавливаем функцию уведомлений
		api.SetNotifyFunction(func(message string) error {
			return bot.NotifyAdmins(message)
		})

		// Бот уже запущен в InitBot через startUpdateHandler
		log.Println("🤖 Telegram бот запущен")
	}

	// Планируем ежедневный сброс в отдельной горутине
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Паника в daily_reset: %v", r)
			}
		}()

		log.Println("⏰ Запуск планировщика ежедневного сброса...")
		daily_reset.ScheduleDailyReset(ctx)
	}()

	// Настраиваем веб-сервер
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","time":"%s"}`, time.Now().Format(time.RFC3339))
	}).Methods("GET")

	// API routes
	router.PathPrefix("/api").Handler(apiRouter)

	// Статические файлы для Mini App
	router.PathPrefix("/").Handler(serveStaticFiles())

	// Получаем порт
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	server = &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Запускаем сервер в отдельной горутине
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Паника в HTTP сервере: %v", r)
			}
		}()

		log.Printf("🌐 Веб-сервер запущен на порту %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ Ошибка запуска сервера: %v", err)
			cancel() // Отменяем контекст при ошибке
		}
	}()

	// Ожидаем сигнал завершения
	sig := <-sigChan
	log.Printf("\n🛑 Получен сигнал завершения: %v", sig)
	log.Println("Начинаем graceful shutdown...")

	// Отменяем контекст для остановки всех горутин
	cancel()

	// Останавливаем HTTP сервер
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	log.Println("⏳ Остановка HTTP сервера...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️ Ошибка остановки сервера: %v", err)
	} else {
		log.Println("✅ HTTP сервер остановлен")
	}

	// Останавливаем бота
	if telegramBot != nil {
		log.Println("⏳ Остановка Telegram бота...")
		// Bot stops automatically when context is cancelled
		log.Println("✅ Telegram бот остановлен")
	}

	// Ждем завершения всех горутин (с таймаутом)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("✅ Все горутины завершены")
	case <-time.After(10 * time.Second):
		log.Println("⚠️ Таймаут ожидания завершения горутин")
	}

	log.Println("=" + strings.Repeat("=", 59))
	log.Println("✅ Сервер полностью остановлен")
	log.Println("=" + strings.Repeat("=", 59))
}

// serveStaticFiles возвращает handler для статических файлов
func serveStaticFiles() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "" || path == "/" {
			path = "/index.html"
		}

		// Безопасность: защита от path traversal
		if strings.Contains(path, "..") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Специальная обработка для файлов из res/
		if strings.HasPrefix(path, "/res/") {
			cleanPath := strings.TrimPrefix(path, "/")
			if _, err := os.Stat(cleanPath); err == nil {
				filePath := cleanPath
				contentType := getContentType(path)
				w.Header().Set("Content-Type", contentType)
				http.ServeFile(w, r, filePath)
				return
			} else {
				http.NotFound(w, r)
				return
			}
		}

		// Специальная обработка для фотографии
		if path == "/welcome_photo.jpeg" || path == "/wedding_photo.jpg" {
			photoPath := config.WebappPhotoPath
			if _, err := os.Stat(photoPath); err == nil {
				contentType := getContentType(path)
				w.Header().Set("Content-Type", contentType)
				http.ServeFile(w, r, photoPath)
				return
			} else {
				http.NotFound(w, r)
				return
			}
		}

		// Убираем ведущий слэш для построения пути к файлу
		cleanPath := strings.TrimPrefix(path, "/")
		filePath := config.WebappPath + "/" + cleanPath

		// Если файл не существует, возвращаем index.html (SPA fallback)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if path != "/index.html" && path != "/" {
				filePath = config.WebappPath + "/index.html"
			}
		}

		contentType := getContentType(path)
		w.Header().Set("Content-Type", contentType)

		// Отключаем кэширование для HTML и JS файлов (чтобы обновления применялись сразу)
		if strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".mjs") {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		} else if strings.HasSuffix(path, ".css") {
			// CSS можно кэшировать, но с коротким временем
			w.Header().Set("Cache-Control", "public, max-age=300") // 5 минут
		} else {
			// Для остальных файлов (изображения и т.д.) - кэшируем дольше
			w.Header().Set("Cache-Control", "public, max-age=3600") // 1 час
		}

		http.ServeFile(w, r, filePath)
	})
}

// getContentType определяет content-type по расширению файла
func getContentType(path string) string {
	if strings.HasSuffix(path, ".css") {
		return "text/css"
	} else if strings.HasSuffix(path, ".js") {
		return "application/javascript"
	} else if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") {
		return "image/jpeg"
	} else if strings.HasSuffix(path, ".png") {
		return "image/png"
	} else if strings.HasSuffix(path, ".gif") {
		return "image/gif"
	} else if strings.HasSuffix(path, ".svg") {
		return "image/svg+xml"
	} else if strings.HasSuffix(path, ".json") {
		return "application/json"
	} else if strings.HasSuffix(path, ".woff") || strings.HasSuffix(path, ".woff2") {
		return "font/woff"
	} else if strings.HasSuffix(path, ".ttf") {
		return "font/ttf"
	} else if strings.HasSuffix(path, ".ico") {
		return "image/x-icon"
	}
	return "text/html"
}
