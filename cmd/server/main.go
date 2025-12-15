package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"wedding-bot/internal/api"
	"wedding-bot/internal/bot"
	"wedding-bot/internal/cache"
	"wedding-bot/internal/config"
	"wedding-bot/internal/daily_reset"
	"wedding-bot/internal/google_sheets"
)

func main() {
	log.Println("=" + strings.Repeat("=", 59))
	log.Println("🚀 ЗАПУСК СВАДЕБНОГО БОТА")
	log.Println("=" + strings.Repeat("=", 59))
	log.Printf("🆔 Process ID: %d", os.Getpid())
	log.Printf("🕐 Время: %s", time.Now().Format(time.RFC3339))
	log.Println("=" + strings.Repeat("=", 59))

	// Загружаем конфигурацию
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	log.Println("✅ Переменные окружения проверены")
	log.Printf("🌐 Порт: %s", os.Getenv("PORT"))
	log.Println("=" + strings.Repeat("=", 59))

	ctx := context.Background()

	// Инициализируем кэш
	if err := cache.InitGameStatsCache(); err != nil {
		log.Printf("⚠️ Ошибка инициализации кэша: %v", err)
	}

	// Инициализируем Google Sheets
	if err := google_sheets.EnsureRequiredSheets(ctx); err != nil {
		log.Printf("⚠️ Ошибка инициализации Google Sheets: %v", err)
	}

	// Инициализируем API
	apiRouter, err := api.InitAPI(ctx)
	if err != nil {
		log.Fatalf("Ошибка инициализации API: %v", err)
	}

	// Инициализируем бота
	telegramBot, err := bot.InitBot(ctx)
	if err != nil {
		log.Printf("⚠️ Ошибка инициализации бота: %v", err)
		log.Println("Бот не будет работать, но API продолжит функционировать")
	} else {
		// Устанавливаем функцию уведомлений
		api.SetNotifyFunction(func(message string) error {
			return bot.NotifyAdmins(message)
		})

		// Запускаем бота в отдельной горутине
		go func() {
			log.Println("🤖 Запуск Telegram бота...")
			telegramBot.Start()
		}()
	}

	// Планируем ежедневный сброс
	daily_reset.ScheduleDailyReset(ctx)

	// Настраиваем веб-сервер
	router := mux.NewRouter()

	// API routes
	router.PathPrefix("/api").Handler(apiRouter)

	// Статические файлы для Mini App
	router.PathPrefix("/").Handler(serveStaticFiles())

	// Получаем порт
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🌐 Веб-сервер запущен на порту %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	<-sigChan
	log.Println("\n🛑 Получен сигнал завершения, останавливаем сервер...")

	// Останавливаем сервер
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка остановки сервера: %v", err)
	}

	if telegramBot != nil {
		telegramBot.Stop()
	}

	log.Println("✅ Сервер остановлен")
}

// serveStaticFiles возвращает handler для статических файлов
func serveStaticFiles() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "" || path == "/" {
			path = "/index.html"
		}

		// Безопасность: только файлы из webapp директории
		if strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		filePath := config.WebappPath + path

		// Специальная обработка для фотографии
		if path == "/welcome_photo.jpeg" || path == "/wedding_photo.jpg" {
			photoPath := config.WebappPhotoPath
			if _, err := os.Stat(photoPath); err == nil {
				filePath = photoPath
			} else {
				http.NotFound(w, r)
				return
			}
		}

		// Специальная обработка для файлов из res/
		if strings.HasPrefix(path, "/res/") {
			cleanPath := strings.TrimPrefix(path, "/")
			if _, err := os.Stat(cleanPath); err == nil {
				filePath = cleanPath
			} else {
				http.NotFound(w, r)
				return
			}
		}

		// Если файл не существует, возвращаем index.html
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if path != "/index.html" {
				filePath = config.WebappPath + "/index.html"
			}
		}

		// Определяем content-type
		contentType := "text/html"
		if strings.HasSuffix(path, ".css") {
			contentType = "text/css"
		} else if strings.HasSuffix(path, ".js") {
			contentType = "application/javascript"
		} else if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") {
			contentType = "image/jpeg"
		} else if strings.HasSuffix(path, ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(path, ".gif") {
			contentType = "image/gif"
		} else if strings.HasSuffix(path, ".svg") {
			contentType = "image/svg+xml"
		} else if strings.HasSuffix(path, ".json") {
			contentType = "application/json"
		}

		w.Header().Set("Content-Type", contentType)
		http.ServeFile(w, r, filePath)
	})
}

