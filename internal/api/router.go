package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"wedding-bot/internal/config"
)

var (
	notifyAdminsFunc func(message string) error
)

// SetNotifyFunction устанавливает функцию уведомлений из bot.go
func SetNotifyFunction(fn func(message string) error) {
	notifyAdminsFunc = fn
}

// NotifyAdmins отправляет уведомление админам
func NotifyAdmins(message string) error {
	if notifyAdminsFunc != nil {
		return notifyAdminsFunc(message)
	}
	return nil
}

// InitAPI инициализирует API роутер
func InitAPI(ctx context.Context) (*mux.Router, error) {
	// Инициализируем структурированное логирование
	initLogger()

	router := mux.NewRouter()

	// Middleware (порядок важен!)
	router.Use(recoveryMiddleware)
	router.Use(requestIDMiddleware)
	router.Use(structuredLoggingMiddleware)
	router.Use(securityMiddleware)
	router.Use(rateLimitMiddleware)
	router.Use(corsMiddleware)

	// API endpoints
	api := router.PathPrefix("/api").Subrouter()

	// Config
	api.HandleFunc("/config", getConfig).Methods("GET")

	// Registration
	api.HandleFunc("/check-registration", checkRegistration).Methods("POST")
	api.HandleFunc("/register", registerGuest).Methods("POST")
	api.HandleFunc("/cancel-registration", cancelGuestRegistration).Methods("POST")
	api.HandleFunc("/confirm-identity", confirmIdentity).Methods("POST")

	// Guests
	api.HandleFunc("/guests", getGuestsList).Methods("GET")
	api.HandleFunc("/stats", getStats).Methods("GET")

	// Timeline
	api.HandleFunc("/timeline", getTimelineEndpoint).Methods("GET")

	// Photo
	api.HandleFunc("/upload-photo", uploadPhoto).Methods("POST")

	// Games
	api.HandleFunc("/game-stats", getGameStatsEndpoint).Methods("GET")
	api.HandleFunc("/update-game-score", updateGameScoreEndpoint).Methods("POST")

	// Wordle
	api.HandleFunc("/wordle/word", getWordleWordEndpoint).Methods("GET")
	api.HandleFunc("/wordle/progress", getWordleProgressEndpoint).Methods("GET")
	api.HandleFunc("/wordle/guess", submitWordleGuessEndpoint).Methods("POST")
	api.HandleFunc("/wordle/state", getWordleStateEndpoint).Methods("GET")
	api.HandleFunc("/wordle/state", saveWordleStateEndpoint).Methods("POST")

	// Crossword
	api.HandleFunc("/crossword/data", getCrosswordDataEndpoint).Methods("GET")
	api.HandleFunc("/crossword/progress", saveCrosswordProgressEndpoint).Methods("POST")
	api.HandleFunc("/crossword/state", getCrosswordStateEndpoint).Methods("GET")
	api.HandleFunc("/crossword/index", setCrosswordIndexEndpoint).Methods("POST")

	// Seating
	api.HandleFunc("/seating/info", getSeatingInfo).Methods("GET")

	// Parse init data
	api.HandleFunc("/parse-init-data", parseInitData).Methods("POST")

	// Health check
	router.HandleFunc("/health", healthCheck).Methods("GET")

	return router, nil
}

// corsMiddleware добавляет CORS заголовки
func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := buildAllowedOrigins()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" && isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		} else if origin == "" {
			// Запросы без Origin (например, внутри Telegram или с того же домена)
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware обрабатывает паники в handlers
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("🚨 Паника в API handler %s %s: %v", r.Method, r.URL.Path, rec)
				JSONError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware логирует запросы (legacy, используется structuredLoggingMiddleware)
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		log.Printf("📡 %s %s - %v", r.Method, r.URL.Path, duration)
	})
}

// healthCheck проверка здоровья сервиса
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// JSONResponse отправляет JSON ответ
func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// JSONError отправляет JSON ошибку в едином формате {error, message}
func JSONError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]string{
		"error":   code,
		"message": humanizeError(code),
	}
	json.NewEncoder(w).Encode(resp)
}

func humanizeError(code string) string {
	switch code {
	case "invalid request":
		return "Некорректный запрос"
	case "invalid_user_id":
		return "Неверный user_id"
	case "user_id required":
		return "Требуется user_id"
	case "server_error":
		return "Внутренняя ошибка сервера"
	default:
		return code
	}
}

func buildAllowedOrigins() []string {
	origins := []string{}

	if config.WebappURL != "" {
		if normalized := normalizeOrigin(config.WebappURL); normalized != "" {
			origins = append(origins, normalized)
		}
	}

	// Локальная разработка
	if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1" {
		origins = append(origins,
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		)
	}

	return origins
}

func isOriginAllowed(origin string, allowed []string) bool {
	normalized := normalizeOrigin(origin)
	for _, o := range allowed {
		if o == normalized {
			return true
		}
	}
	return false
}

func normalizeOrigin(raw string) string {
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	if u.Scheme == "" || u.Host == "" {
		return ""
	}

	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
}

// getConfig возвращает конфигурацию для фронтенда
func getConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"wedding_date":    config.WeddingDate.Format("2006-01-02"),
		"groom_name":      config.GroomName,
		"bride_name":      config.BrideName,
		"wedding_address": config.WeddingAddress,
		"groom_telegram":  config.GroomTelegram,
		"bride_telegram":  config.BrideTelegram,
		"group_link":      config.GroupLink,
	}

	JSONResponse(w, http.StatusOK, config)
}
