package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"wedding-bot/internal/cache"
	"wedding-bot/internal/google_sheets"
)

// getGameStatsEndpoint возвращает статистику игрока
func getGameStatsEndpoint(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("userId")
	if userIDStr == "" {
		JSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	ctx := r.Context()

	cacheKey := fmt.Sprintf("game_stats:%d", userID)
	if cached, ok := cache.GetMemoryCacheValue(cacheKey); ok {
		if s, ok := cached.(*google_sheets.GameStats); ok {
			respondGameStats(w, userID, s)
			return
		}
	}

	stats, err := google_sheets.GetGameStats(ctx, userID)
	if err != nil {
		log.Printf("Error getting game stats: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	if stats != nil {
		cache.SetMemoryCache(cacheKey, stats, 30*time.Second)
	}

	respondGameStats(w, userID, stats)
}

func respondGameStats(w http.ResponseWriter, userID int, stats *google_sheets.GameStats) {
	if stats == nil {
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"user_id":         userID,
			"first_name":      "",
			"last_name":       "",
			"total_score":     0,
			"dragon_score":    0,
			"flappy_score":    0,
			"crossword_score": 0,
			"wordle_score":    0,
			"rank":            "Незнакомец",
			"last_updated":    "",
		})
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"user_id":         stats.UserID,
		"first_name":      stats.FirstName,
		"last_name":       stats.LastName,
		"total_score":     stats.TotalScore,
		"dragon_score":    stats.DragonScore,
		"flappy_score":    stats.FlappyScore,
		"crossword_score": stats.CrosswordScore,
		"wordle_score":    stats.WordleScore,
		"rank":            stats.Rank,
		"last_updated":    stats.LastUpdated,
	})
}

// updateGameScoreEndpoint обновляет счет игрока
func updateGameScoreEndpoint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   int    `json:"userId"`
		GameType string `json:"gameType"`
		Score    int    `json:"score"`
		InitData string `json:"initData"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	userID := req.UserID
	if userID == 0 && req.InitData != "" {
		result, err := ParseInitData(req.InitData)
		if err == nil {
			if uid, ok := result["userId"].(int); ok {
				userID = uid
			}
		}
	}

	if userID == 0 {
		JSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	if req.GameType == "" {
		JSONError(w, http.StatusBadRequest, "game_type required")
		return
	}

	ctx := r.Context()

	if err := google_sheets.UpdateGameScore(ctx, userID, req.GameType, req.Score); err != nil {
		log.Printf("Error updating game score: %v", err)
		JSONError(w, http.StatusInternalServerError, "failed to update score")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// getWordleWordEndpoint возвращает актуальное слово для Wordle
func getWordleWordEndpoint(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("userId")
	initData := r.URL.Query().Get("initData")

	ctx := r.Context()

	var userID int
	var err error

	if initData != "" {
		result, err := ParseInitData(initData)
		if err == nil {
			if uid, ok := result["userId"].(int); ok {
				userID = uid
			}
		}
	}

	if userID == 0 && userIDStr != "" {
		userID, err = strconv.Atoi(userIDStr)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
	}

	if userID == 0 {
		JSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	word, err := google_sheets.GetWordleWordForUser(ctx, userID)
	if err != nil {
		log.Printf("Error getting Wordle word: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	if word == "" {
		JSONError(w, http.StatusNotFound, "word not found")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"word": word,
	})
}

// getWordleProgressEndpoint возвращает прогресс пользователя в Wordle
func getWordleProgressEndpoint(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("userId")
	initData := r.URL.Query().Get("initData")

	ctx := r.Context()

	var userID int
	var err error

	if initData != "" {
		result, err := ParseInitData(initData)
		if err == nil {
			if uid, ok := result["userId"].(int); ok {
				userID = uid
			}
		}
	}

	if userID == 0 && userIDStr != "" {
		userID, err = strconv.Atoi(userIDStr)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
	}

	if userID == 0 {
		JSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	guessedWords, err := google_sheets.GetWordleGuessedWords(ctx, userID)
	if err != nil {
		log.Printf("Error getting Wordle progress: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"guessed_words": guessedWords,
	})
}

// submitWordleGuessEndpoint обрабатывает отгаданное слово в Wordle
func submitWordleGuessEndpoint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Word     string `json:"word"`
		UserID   int    `json:"userId"`
		InitData string `json:"initData"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	word := strings.TrimSpace(strings.ToUpper(req.Word))
	if word == "" {
		JSONError(w, http.StatusBadRequest, "word required")
		return
	}

	ctx := r.Context()

	userID := req.UserID
	if userID == 0 && req.InitData != "" {
		result, err := ParseInitData(req.InitData)
		if err == nil {
			if uid, ok := result["userId"].(int); ok {
				userID = uid
			}
		}
	}

	if userID == 0 {
		JSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	// Получаем текущее слово
	currentWord, err := google_sheets.GetWordleWordForUser(ctx, userID)
	if err != nil || currentWord == "" {
		JSONError(w, http.StatusNotFound, "word not found")
		return
	}

	// Проверяем совпадение
	if word != currentWord {
		JSONError(w, http.StatusBadRequest, "incorrect word")
		return
	}

	// Получаем отгаданные слова
	guessedWords, err := google_sheets.GetWordleGuessedWords(ctx, userID)
	if err != nil {
		log.Printf("Error getting guessed words: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	// Проверяем, не отгадано ли уже
	for _, gw := range guessedWords {
		if gw == word {
			JSONResponse(w, http.StatusOK, map[string]interface{}{
				"success":         false,
				"message":         "Это слово уже было отгадано",
				"already_guessed": true,
			})
			return
		}
	}

	// Добавляем слово
	guessedWords = append(guessedWords, word)
	if err := google_sheets.SaveWordleProgress(ctx, userID, guessedWords); err != nil {
		log.Printf("Error saving progress: %v", err)
		JSONError(w, http.StatusInternalServerError, "failed to save")
		return
	}

	// Начисляем очки
	if err := google_sheets.UpdateGameScore(ctx, userID, "wordle", 1); err != nil {
		log.Printf("Error updating score: %v", err)
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Слово отгадано! +5 очков",
		"points":  5,
	})
}

// getWordleStateEndpoint возвращает состояние игры Wordle
func getWordleStateEndpoint(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("userId")
	initData := r.URL.Query().Get("initData")

	ctx := r.Context()

	var userID int
	var err error

	if initData != "" {
		result, err := ParseInitData(initData)
		if err == nil {
			if uid, ok := result["userId"].(int); ok {
				userID = uid
			}
		}
	}

	if userID == 0 && userIDStr != "" {
		userID, err = strconv.Atoi(userIDStr)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
	}

	if userID == 0 {
		JSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	state, err := google_sheets.GetWordleState(ctx, userID)
	if err != nil {
		log.Printf("Error getting Wordle state: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	if state == nil {
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"current_word":   nil,
			"attempts":       []interface{}{},
			"current_guess":  "",
			"last_word_date": nil,
		})
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"current_word":   state.CurrentWord,
		"attempts":       state.Attempts,
		"current_guess":  state.CurrentGuess,
		"last_word_date": state.LastWordDate,
	})
}

// saveWordleStateEndpoint сохраняет состояние игры Wordle
func saveWordleStateEndpoint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID       int                        `json:"userId"`
		CurrentWord  string                     `json:"current_word"`
		Attempts     [][]map[string]interface{} `json:"attempts"`
		CurrentGuess string                     `json:"current_guess"`
		LastWordDate string                     `json:"last_word_date"`
		InitData     string                     `json:"initData"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	ctx := r.Context()

	userID := req.UserID
	if userID == 0 && req.InitData != "" {
		result, err := ParseInitData(req.InitData)
		if err == nil {
			if uid, ok := result["userId"].(int); ok {
				userID = uid
			}
		}
	}

	if userID == 0 {
		JSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	if err := google_sheets.SaveWordleState(ctx, userID, req.CurrentWord, req.Attempts, req.LastWordDate, req.CurrentGuess); err != nil {
		log.Printf("Error saving Wordle state: %v", err)
		JSONError(w, http.StatusInternalServerError, "failed to save")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// getCrosswordDataEndpoint возвращает данные кроссворда
func getCrosswordDataEndpoint(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("userId")
	crosswordIndexStr := r.URL.Query().Get("crosswordIndex")

	ctx := r.Context()

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	crosswordIndex := 0
	if crosswordIndexStr != "" {
		crosswordIndex, _ = strconv.Atoi(crosswordIndexStr)
	} else {
		crosswordIndex = google_sheets.GetCurrentCrosswordIndex(ctx)
	}

	// Получаем слова
	words, err := google_sheets.GetCrosswordWords(ctx, crosswordIndex)
	if err != nil {
		log.Printf("Error getting crossword words: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	// Получаем прогресс
	progress, err := google_sheets.GetCrosswordProgress(ctx, userID, crosswordIndex)
	if err != nil {
		log.Printf("Error getting crossword progress: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	// Конвертируем слова в нужный формат
	wordsData := make([]map[string]string, len(words))
	for i, w := range words {
		wordsData[i] = map[string]string{
			"word":        w.Word,
			"description": w.Description,
		}
	}

	response := map[string]interface{}{
		"words":                wordsData,
		"crossword_index":      crosswordIndex,
		"guessed_words":        progress.GuessedWords,
		"cell_letters":         progress.CellLetters,
		"wrong_attempts":       progress.WrongAttempts,
		"crossword_start_date": progress.StartDate,
	}

	JSONResponse(w, http.StatusOK, response)
}

// saveCrosswordProgressEndpoint сохраняет прогресс кроссворда
func saveCrosswordProgressEndpoint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID             int               `json:"userId"`
		GuessedWords       []string          `json:"guessed_words"`
		CrosswordIndex     int               `json:"crossword_index"`
		CellLetters        map[string]string `json:"cell_letters"`
		WrongAttempts      []string          `json:"wrong_attempts"`
		CrosswordStartDate string            `json:"crossword_start_date"`
		InitData           string            `json:"initData"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	ctx := r.Context()

	userID := req.UserID
	if userID == 0 && req.InitData != "" {
		result, err := ParseInitData(req.InitData)
		if err == nil {
			if uid, ok := result["userId"].(int); ok {
				userID = uid
			}
		}
	}

	if userID == 0 {
		JSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	if err := google_sheets.SaveCrosswordProgress(ctx, userID, req.GuessedWords, req.CrosswordIndex, req.CellLetters, req.WrongAttempts, req.CrosswordStartDate); err != nil {
		log.Printf("Error saving crossword progress: %v", err)
		JSONError(w, http.StatusInternalServerError, "failed to save")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// getCrosswordStateEndpoint возвращает состояние кроссворда
func getCrosswordStateEndpoint(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("userId")
	if userIDStr == "" {
		JSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	ctx := r.Context()

	index, err := google_sheets.GetCrosswordState(ctx, userID)
	if err != nil {
		log.Printf("Error getting crossword state: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"current_crossword_index": index,
	})
}

// setCrosswordIndexEndpoint устанавливает индекс кроссворда
func setCrosswordIndexEndpoint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID         int `json:"userId"`
		CrosswordIndex int `json:"crossword_index"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if req.UserID == 0 {
		JSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	ctx := r.Context()

	if err := google_sheets.SetCrosswordIndex(ctx, req.UserID, req.CrosswordIndex); err != nil {
		log.Printf("Error setting crossword index: %v", err)
		JSONError(w, http.StatusInternalServerError, "failed to set")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}
