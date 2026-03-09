package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"wedding-bot/internal/cache"
	"wedding-bot/internal/google_sheets"
)

// parseInitData парсит initData и возвращает user_id
func parseInitData(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InitData string `json:"initData"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if req.InitData == "" {
		JSONError(w, http.StatusBadRequest, "initData required")
		return
	}

	result, err := ParseInitData(req.InitData)
	if err != nil {
		log.Printf("Error parsing initData: %v", err)
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	JSONResponse(w, http.StatusOK, result)
}

func extractUserIDFromParsedInitData(result map[string]interface{}) int {
	if result == nil {
		return 0
	}

	switch uid := result["userId"].(type) {
	case int:
		return uid
	case float64:
		return int(uid)
	case int64:
		return int(uid)
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(uid)); err == nil {
			return parsed
		}
	}

	return 0
}

func extractUsernameFromParsedInitData(result map[string]interface{}) string {
	if result == nil {
		return ""
	}

	if username, ok := result["username"].(string); ok {
		return google_sheets.NormalizeTelegramUsername(username)
	}

	return ""
}

func registrationCacheKeys(userID int, username string) []string {
	keys := make([]string, 0, 2)
	if userID > 0 {
		keys = append(keys, fmt.Sprintf("registration:id:%d", userID))
	}

	normalizedUsername := google_sheets.NormalizeTelegramUsername(username)
	if normalizedUsername != "" {
		keys = append(keys, fmt.Sprintf("registration:username:%s", normalizedUsername))
	}

	return keys
}

// checkRegistration проверяет регистрацию пользователя
func checkRegistration(w http.ResponseWriter, r *http.Request) {
	// Создаем контекст с таймаутом для защиты от зависаний
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var req struct {
		UserID   int    `json:"userId"`
		Username string `json:"username"`
		InitData string `json:"initData"`
	}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
	}

	userIDStr := strings.TrimSpace(r.URL.Query().Get("userId"))
	username := google_sheets.NormalizeTelegramUsername(r.URL.Query().Get("username"))
	if username == "" {
		username = google_sheets.NormalizeTelegramUsername(req.Username)
	}

	userID := req.UserID
	if userIDStr != "" {
		parsedUserID, err := strconv.Atoi(userIDStr)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid_user_id")
			return
		}
		userID = parsedUserID
	}

	if req.InitData != "" {
		result, err := ParseInitData(req.InitData)
		if err == nil {
			if userID == 0 {
				userID = extractUserIDFromParsedInitData(result)
			}
			if username == "" {
				username = extractUsernameFromParsedInitData(result)
			}
		}
	}

	if userID == 0 && username == "" {
		JSONError(w, http.StatusBadRequest, "user_id_or_username_required")
		return
	}

	cacheKeys := registrationCacheKeys(userID, username)

	// Используем кэш только для положительного результата.
	// Негативный кэш может быстро устаревать и давать ложный "не зарегистрирован".
	for _, cacheKey := range cacheKeys {
		if cached, ok := cache.GetMemoryCacheValue(cacheKey); ok {
			if val, ok := cached.(bool); ok && val {
				resp := map[string]interface{}{
					"registered": true,
					"cached":     true,
				}
				if userID > 0 {
					inGroupChat, _ := IsUserInGroupChat(userID)
					resp["in_group_chat"] = inGroupChat
				}
				if username != "" && userID == 0 {
					resp["auth_mode"] = "username"
				}
				JSONResponse(w, http.StatusOK, resp)
				return
			}
		}
	}

	registered, err := google_sheets.CheckGuestRegistrationByIdentifier(ctx, userID, username)
	if err != nil {
		log.Printf("Error checking registration: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	if registered {
		for _, cacheKey := range cacheKeys {
			cache.SetMemoryCache(cacheKey, true, 30*time.Second)
		}
	}

	resp := map[string]interface{}{
		"registered": registered,
	}
	if userID > 0 {
		inGroupChat, _ := IsUserInGroupChat(userID)
		resp["in_group_chat"] = inGroupChat
	}
	if username != "" && userID == 0 {
		resp["auth_mode"] = "username"
	}

	JSONResponse(w, http.StatusOK, resp)
}

// registerGuest регистрирует гостя
func registerGuest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Age       *int   `json:"age"`
		Category  string `json:"category"`
		Side      string `json:"side"`
		UserID    int    `json:"userId"`
		InitData  string `json:"initData"`
		Username  string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Получаем user_id из initData если не передан напрямую
	userID := req.UserID
	manualUsername := google_sheets.NormalizeTelegramUsername(req.Username)
	// #region agent log
	log.Printf("[DEBUG registerGuest] Initial userID from request: %d, hasInitData: %v, initData length: %d", userID, req.InitData != "", len(req.InitData))
	// #endregion
	if req.InitData != "" {
		result, err := ParseInitData(req.InitData)
		// #region agent log
		log.Printf("[DEBUG registerGuest] ParseInitData result: err=%v, result=%+v", err, result)
		// #endregion
		if err == nil {
			if userID == 0 {
				userID = extractUserIDFromParsedInitData(result)
			}
			if manualUsername == "" {
				manualUsername = extractUsernameFromParsedInitData(result)
			}
			// #region agent log
			log.Printf("[DEBUG registerGuest] Parsed auth from initData: user_id=%d, username=%s", userID, manualUsername)
			// #endregion
			log.Printf("Parsed user_id from initData: %d", userID)
		} else {
			log.Printf("Error parsing initData: %v", err)
		}
	}

	// Разрешаем регистрацию без user_id только при наличии username (режим браузера вне Telegram)
	if userID == 0 && manualUsername == "" {
		// #region agent log
		log.Printf("[DEBUG registerGuest] user_id is 0 after all attempts. InitData provided: %v (len=%d), UserID in request: %d, FirstName: %s, LastName: %s", req.InitData != "", len(req.InitData), req.UserID, req.FirstName, req.LastName)
		// #endregion
		log.Printf("Registration failed: user_id is 0. InitData provided: %v, UserID in request: %d", req.InitData != "", req.UserID)
		JSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	// Таймаут для регистрации (может быть долгой операцией)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Идемпотентность: если уже зарегистрирован по user_id/username, возвращаем успех
	if registered, err := google_sheets.CheckGuestRegistrationByIdentifier(ctx, userID, manualUsername); err == nil && registered {
		for _, key := range registrationCacheKeys(userID, manualUsername) {
			cache.SetMemoryCache(key, true, 30*time.Second)
		}
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success":            true,
			"already_registered": true,
		})
		return
	}

	var age *int
	if req.Age != nil {
		age = req.Age
	}

	var category *string
	if req.Category != "" {
		category = &req.Category
	}

	var side *string
	if req.Side != "" {
		side = &req.Side
	}

	// Если userID == 0 (браузерный режим), пишем запись без user_id
	var userIDPtr *int
	if userID != 0 {
		userIDPtr = &userID
	}

	var usernamePtr *string
	if manualUsername != "" {
		usernamePtr = &manualUsername
	}

	if err := google_sheets.AddGuestToSheets(ctx, req.FirstName, req.LastName, age, category, side, userIDPtr, usernamePtr); err != nil {
		log.Printf("Error adding guest: %v", err)
		JSONError(w, http.StatusInternalServerError, "failed to register")
		return
	}

	for _, key := range registrationCacheKeys(userID, manualUsername) {
		cache.SetMemoryCache(key, true, 30*time.Second)
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// cancelGuestRegistration отменяет регистрацию гостя
func cancelGuestRegistration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   int    `json:"userId"`
		InitData string `json:"initData"`
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	userID := req.UserID
	username := google_sheets.NormalizeTelegramUsername(req.Username)
	if req.InitData != "" {
		result, err := ParseInitData(req.InitData)
		if err == nil {
			if userID == 0 {
				userID = extractUserIDFromParsedInitData(result)
			}
			if username == "" {
				username = extractUsernameFromParsedInitData(result)
			}
		}
	}

	if userID == 0 && username == "" {
		JSONError(w, http.StatusBadRequest, "user_id_or_username_required")
		return
	}

	ctx := r.Context()

	if err := google_sheets.CancelGuestRegistrationByIdentifier(ctx, userID, username); err != nil {
		log.Printf("Error canceling registration: %v", err)
		if strings.Contains(strings.ToLower(err.Error()), "не найден") {
			JSONError(w, http.StatusBadRequest, "guest_not_found")
			return
		}
		JSONError(w, http.StatusInternalServerError, "failed_to_cancel")
		return
	}

	for _, key := range registrationCacheKeys(userID, username) {
		cache.SetMemoryCache(key, false, 30*time.Second)
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// getGuestsList возвращает список гостей
func getGuestsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	guests, err := google_sheets.GetAllGuestsFromSheets(ctx)
	if err != nil {
		log.Printf("Error getting guests: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"guests": guests,
	})
}

// getStats возвращает статистику
func getStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	count, err := google_sheets.GetGuestsCountFromSheets(ctx)
	if err != nil {
		log.Printf("Error getting stats: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"count": count,
	})
}

// getTimelineEndpoint возвращает тайминг мероприятия
func getTimelineEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	timeline, err := google_sheets.GetTimeline(ctx)
	if err != nil {
		log.Printf("Error getting timeline: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"timeline": timeline,
	})
}

// uploadPhoto загружает фото
func uploadPhoto(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    int    `json:"userId"`
		Username  string `json:"username"`
		FullName  string `json:"fullName"`
		PhotoData string `json:"photoData"`
		InitData  string `json:"initData"`
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

	ctx := r.Context()

	var username *string
	if req.Username != "" {
		username = &req.Username
	}

	if err := google_sheets.SavePhotoFromWebapp(ctx, userID, username, req.FullName, req.PhotoData); err != nil {
		log.Printf("Error saving photo: %v", err)
		JSONError(w, http.StatusInternalServerError, "failed to save photo")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// getSeatingInfo возвращает информацию о рассадке
func getSeatingInfo(w http.ResponseWriter, r *http.Request) {
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

	info, err := google_sheets.GetGuestTableAndNeighbors(ctx, userID)
	if err != nil {
		log.Printf("Error getting seating info: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	if info == nil {
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"visible": false,
		})
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"visible":   true,
		"table":     info.Table,
		"neighbors": info.Neighbors,
		"full_name": info.FullName,
	})
}
