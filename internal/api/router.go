package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
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
	router := mux.NewRouter()

	// CORS middleware
	router.Use(corsMiddleware)

	// Инициализируем необходимые листы в Google Sheets
	if err := google_sheets.EnsureRequiredSheets(ctx); err != nil {
		log.Printf("Ошибка инициализации Google Sheets: %v", err)
	}

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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
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

// JSONError отправляет JSON ошибку
func JSONError(w http.ResponseWriter, status int, message string) {
	JSONResponse(w, status, map[string]string{
		"error": message,
	})
}

// getConfig возвращает конфигурацию для фронтенда
func getConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"wedding_date":      config.WeddingDate.Format("2006-01-02"),
		"groom_name":        config.GroomName,
		"bride_name":        config.BrideName,
		"wedding_address":   config.WeddingAddress,
		"groom_telegram":    config.GroomTelegram,
		"bride_telegram":    config.BrideTelegram,
		"group_link":        config.GroupLink,
	}

	JSONResponse(w, http.StatusOK, config)
}

