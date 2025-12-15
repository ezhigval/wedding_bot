package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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

// checkRegistration проверяет регистрацию пользователя
func checkRegistration(w http.ResponseWriter, r *http.Request) {
	// Создаем контекст с таймаутом для защиты от зависаний
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	userIDStr := r.URL.Query().Get("userId")
	firstName := r.URL.Query().Get("firstName")
	lastName := r.URL.Query().Get("lastName")
	searchByNameOnly := r.URL.Query().Get("searchByNameOnly") == "true"

	if searchByNameOnly {
		if firstName == "" || lastName == "" {
			JSONError(w, http.StatusBadRequest, "name_required")
			return
		}

		guest, err := google_sheets.FindGuestByName(ctx, firstName, lastName)
		if err != nil {
			log.Printf("Error finding guest: %v", err)
			JSONError(w, http.StatusInternalServerError, "server_error")
			return
		}

		if guest != nil {
			if guest.UserID != "" {
				JSONResponse(w, http.StatusOK, map[string]interface{}{
					"registered": true,
				})
				return
			}

			JSONResponse(w, http.StatusOK, map[string]interface{}{
				"registered":      false,
				"needs_confirmation": true,
				"guest_name":      fmt.Sprintf("%s %s", guest.FirstName, guest.LastName),
			})
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"registered": false,
		})
		return
	}

	if userIDStr == "" {
		if firstName != "" && lastName != "" {
			guest, err := google_sheets.FindGuestByName(ctx, firstName, lastName)
			if err != nil {
				log.Printf("Error finding guest: %v", err)
				JSONError(w, http.StatusInternalServerError, "server_error")
				return
			}

			if guest != nil {
				if guest.UserID != "" {
					JSONResponse(w, http.StatusOK, map[string]interface{}{
						"registered": true,
					})
					return
				}

				JSONResponse(w, http.StatusOK, map[string]interface{}{
					"registered":      false,
					"needs_confirmation": true,
					"guest_name":      fmt.Sprintf("%s %s", guest.FirstName, guest.LastName),
				})
				return
			}
		}

		JSONError(w, http.StatusBadRequest, "user_id_or_name_required")
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid_user_id")
		return
	}

	// Проверяем регистрацию
	registered, err := google_sheets.CheckGuestRegistration(ctx, userID)
	if err != nil {
		log.Printf("Error checking registration: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	if registered {
		inGroupChat, _ := IsUserInGroupChat(userID)
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"registered":   true,
			"in_group_chat": inGroupChat,
		})
		return
	}

	// Проверяем по имени
	if firstName != "" && lastName != "" {
		guest, err := google_sheets.FindGuestByName(ctx, firstName, lastName)
		if err != nil {
			log.Printf("Error finding guest: %v", err)
		} else if guest != nil {
			// Найден по имени - считаем зарегистрированным
			inGroupChat, _ := IsUserInGroupChat(userID)
			JSONResponse(w, http.StatusOK, map[string]interface{}{
				"registered":   true,
				"in_group_chat": inGroupChat,
			})
			return
		}
	}

	inGroupChat, _ := IsUserInGroupChat(userID)
	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"registered":   false,
		"in_group_chat": inGroupChat,
	})
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
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Получаем user_id из initData если не передан напрямую
	userID := req.UserID
	if userID == 0 && req.InitData != "" {
		result, err := ParseInitData(req.InitData)
		if err == nil {
			// Пробуем разные типы для userId
			if uid, ok := result["userId"].(int); ok {
				userID = uid
			} else if uidFloat, ok := result["userId"].(float64); ok {
				userID = int(uidFloat)
			} else if uidInt64, ok := result["userId"].(int64); ok {
				userID = int(uidInt64)
			}
			log.Printf("Parsed user_id from initData: %d", userID)
		} else {
			log.Printf("Error parsing initData: %v", err)
		}
	}

	if userID == 0 {
		log.Printf("Registration failed: user_id is 0. InitData provided: %v, UserID in request: %d", req.InitData != "", req.UserID)
		JSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	// Таймаут для регистрации (может быть долгой операцией)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

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

	if err := google_sheets.AddGuestToSheets(ctx, req.FirstName, req.LastName, age, category, side, &userID); err != nil {
		log.Printf("Error adding guest: %v", err)
		JSONError(w, http.StatusInternalServerError, "failed to register")
		return
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

	if err := google_sheets.CancelGuestRegistrationByUserID(ctx, userID); err != nil {
		log.Printf("Error canceling registration: %v", err)
		JSONError(w, http.StatusInternalServerError, "failed to cancel")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// confirmIdentity подтверждает личность
func confirmIdentity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Row    int `json:"row"`
		UserID int `json:"userId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if req.Row == 0 || req.UserID == 0 {
		JSONError(w, http.StatusBadRequest, "missing_data")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	err := google_sheets.UpdateGuestUserID(ctx, req.Row, req.UserID)
	if err != nil {
		log.Printf("Ошибка обновления user_id гостя: %v", err)
		JSONError(w, http.StatusInternalServerError, "update_failed")
		return
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

