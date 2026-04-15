package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"wedding-bot/internal/cache"
	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
)

const (
	authSessionCookieName = "wedding_auth_session"
	authSessionTTL        = 30 * 24 * time.Hour
)

type authSessionPayload struct {
	UserID   int    `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	Expires  int64  `json:"exp"`
}

type registerGuestCompanionRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Telegram  string `json:"telegram"`
}

func normalizeRegistrationNamePart(value string) string {
	return strings.TrimSpace(value)
}

func buildNormalizedRegistrationFullName(firstName, lastName string) string {
	return strings.TrimSpace(strings.Join([]string{
		normalizeRegistrationNamePart(firstName),
		normalizeRegistrationNamePart(lastName),
	}, " "))
}

func sanitizeAdditionalGuests(rawGuests []registerGuestCompanionRequest, ownerFirstName, ownerLastName string) []google_sheets.CompanionGuest {
	if len(rawGuests) == 0 {
		return nil
	}

	ownerFullName := strings.ToLower(buildNormalizedRegistrationFullName(ownerFirstName, ownerLastName))
	seen := make(map[string]struct{}, len(rawGuests))
	guests := make([]google_sheets.CompanionGuest, 0, len(rawGuests))

	for _, rawGuest := range rawGuests {
		firstName := normalizeRegistrationNamePart(rawGuest.FirstName)
		lastName := normalizeRegistrationNamePart(rawGuest.LastName)
		fullName := buildNormalizedRegistrationFullName(firstName, lastName)
		if fullName == "" {
			continue
		}

		normalizedKey := strings.ToLower(fullName)
		if normalizedKey == ownerFullName {
			continue
		}
		if _, ok := seen[normalizedKey]; ok {
			continue
		}

		seen[normalizedKey] = struct{}{}
		guests = append(guests, google_sheets.CompanionGuest{
			FirstName: firstName,
			LastName:  lastName,
		})
	}

	return guests
}

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

func authSessionSecret() []byte {
	secret := strings.TrimSpace(config.BotToken)
	if secret == "" {
		secret = "wedding-bot-session-secret"
	}
	return []byte(secret)
}

func signAuthSession(body string) string {
	mac := hmac.New(sha256.New, authSessionSecret())
	mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encodeAuthSession(userID int, username string) (string, error) {
	normalizedUsername := google_sheets.NormalizeTelegramUsername(username)
	if userID <= 0 && normalizedUsername == "" {
		return "", fmt.Errorf("empty auth identity")
	}

	payload := authSessionPayload{
		UserID:   userID,
		Username: normalizedUsername,
		Expires:  time.Now().Add(authSessionTTL).Unix(),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	body := base64.RawURLEncoding.EncodeToString(raw)
	signature := signAuthSession(body)
	return body + "." + signature, nil
}

func decodeAuthSession(raw string) (int, string, bool) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return 0, "", false
	}

	body := parts[0]
	signature := parts[1]
	expectedSignature := signAuthSession(body)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return 0, "", false
	}

	decodedBody, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return 0, "", false
	}

	var payload authSessionPayload
	if err := json.Unmarshal(decodedBody, &payload); err != nil {
		return 0, "", false
	}

	if payload.Expires > 0 && time.Now().Unix() > payload.Expires {
		return 0, "", false
	}

	normalizedUsername := google_sheets.NormalizeTelegramUsername(payload.Username)
	if payload.UserID <= 0 && normalizedUsername == "" {
		return 0, "", false
	}

	return payload.UserID, normalizedUsername, true
}

func isHTTPSRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func setAuthSessionCookie(w http.ResponseWriter, r *http.Request, userID int, username string) {
	sessionValue, err := encodeAuthSession(userID, username)
	if err != nil {
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authSessionCookieName,
		Value:    sessionValue,
		Path:     "/",
		MaxAge:   int(authSessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   isHTTPSRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func readAuthSessionCookie(r *http.Request) (int, string, bool) {
	if r == nil {
		return 0, "", false
	}

	cookie, err := r.Cookie(authSessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return 0, "", false
	}

	return decodeAuthSession(cookie.Value)
}

func usernameToUserIDCacheKey(username string) string {
	return fmt.Sprintf("tg:user-id-by-username:%s", google_sheets.NormalizeTelegramUsername(username))
}

func cacheUsernameToUserID(username string, userID int) {
	normalizedUsername := google_sheets.NormalizeTelegramUsername(username)
	if normalizedUsername == "" || userID <= 0 {
		return
	}
	cache.SetMemoryCache(usernameToUserIDCacheKey(normalizedUsername), userID, 24*time.Hour)
}

func getCachedUserIDByUsername(username string) (int, bool) {
	normalizedUsername := google_sheets.NormalizeTelegramUsername(username)
	if normalizedUsername == "" {
		return 0, false
	}

	cached, ok := cache.GetMemoryCacheValue(usernameToUserIDCacheKey(normalizedUsername))
	if !ok {
		return 0, false
	}

	switch v := cached.(type) {
	case int:
		if v > 0 {
			return v, true
		}
	case int64:
		if v > 0 {
			return int(v), true
		}
	case float64:
		if v > 0 {
			return int(v), true
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && parsed > 0 {
			return parsed, true
		}
	}

	return 0, false
}

func resolveAuthIdentity(userID int, username, initData string) (int, string) {
	normalizedUsername := google_sheets.NormalizeTelegramUsername(username)
	resolvedUserID := userID

	if initData != "" {
		if result, err := ParseInitData(initData); err == nil {
			if resolvedUserID == 0 {
				resolvedUserID = extractUserIDFromParsedInitData(result)
			}
			if normalizedUsername == "" {
				normalizedUsername = extractUsernameFromParsedInitData(result)
			}
		}
	}

	if resolvedUserID > 0 && normalizedUsername != "" {
		cacheUsernameToUserID(normalizedUsername, resolvedUserID)
		return resolvedUserID, normalizedUsername
	}

	if resolvedUserID == 0 && normalizedUsername != "" {
		if cachedID, ok := getCachedUserIDByUsername(normalizedUsername); ok {
			return cachedID, normalizedUsername
		}
		if resolvedID, err := ResolveUserIDByUsername(normalizedUsername); err == nil && resolvedID > 0 {
			cacheUsernameToUserID(normalizedUsername, resolvedID)
			return resolvedID, normalizedUsername
		}
	}

	return resolvedUserID, normalizedUsername
}

func resolveAuthIdentityFromRequest(r *http.Request, userID int, username, initData string) (int, string) {
	resolvedUserID, resolvedUsername := resolveAuthIdentity(userID, username, initData)
	if resolvedUserID > 0 || resolvedUsername != "" {
		return resolvedUserID, resolvedUsername
	}

	cookieUserID, cookieUsername, ok := readAuthSessionCookie(r)
	if !ok {
		return 0, ""
	}

	return resolveAuthIdentity(cookieUserID, cookieUsername, "")
}

func parseOptionalUserID(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}

	return strconv.Atoi(raw)
}

func resolveAuthIdentityAndRefreshSession(w http.ResponseWriter, r *http.Request, userID int, username, initData string) (int, string) {
	resolvedUserID, resolvedUsername := resolveAuthIdentityFromRequest(r, userID, username, initData)
	if resolvedUserID > 0 || resolvedUsername != "" {
		setAuthSessionCookie(w, r, resolvedUserID, resolvedUsername)
	}

	return resolvedUserID, resolvedUsername
}

func requireResolvedAuthUserID(w http.ResponseWriter, r *http.Request, userID int, username, initData string) (int, string, bool) {
	resolvedUserID, resolvedUsername := resolveAuthIdentityAndRefreshSession(w, r, userID, username, initData)
	if resolvedUserID == 0 {
		JSONError(w, http.StatusBadRequest, "user_id required")
		return 0, "", false
	}

	return resolvedUserID, resolvedUsername, true
}

func requireResolvedAuthIdentity(w http.ResponseWriter, r *http.Request, userID int, username, initData string) (int, string, bool) {
	resolvedUserID, resolvedUsername := resolveAuthIdentityAndRefreshSession(w, r, userID, username, initData)
	if resolvedUserID == 0 && resolvedUsername == "" {
		JSONError(w, http.StatusBadRequest, "user_id_or_username_required")
		return 0, "", false
	}

	return resolvedUserID, resolvedUsername, true
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
		username = req.Username
	}

	userID := req.UserID
	if parsedUserID, err := parseOptionalUserID(userIDStr); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid_user_id")
		return
	} else if parsedUserID > 0 {
		userID = parsedUserID
	}

	userID, username = resolveAuthIdentityFromRequest(r, userID, username, req.InitData)

	if userID == 0 && username == "" {
		JSONError(w, http.StatusBadRequest, "user_id_or_username_required")
		return
	}

	setAuthSessionCookie(w, r, userID, username)

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
		FirstName string                          `json:"firstName"`
		LastName  string                          `json:"lastName"`
		Age       *int                            `json:"age"`
		Category  string                          `json:"category"`
		Side      string                          `json:"side"`
		UserID    int                             `json:"userId"`
		InitData  string                          `json:"initData"`
		Username  string                          `json:"username"`
		Guests    []registerGuestCompanionRequest `json:"guests"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Получаем user_id из initData если не передан напрямую
	userID := req.UserID
	manualUsername := req.Username
	// #region agent log
	log.Printf("[DEBUG registerGuest] Initial userID from request: %d, hasInitData: %v, initData length: %d", userID, req.InitData != "", len(req.InitData))
	// #endregion
	userID, manualUsername = resolveAuthIdentityFromRequest(r, userID, manualUsername, req.InitData)
	// #region agent log
	log.Printf("[DEBUG registerGuest] Resolved auth identity: user_id=%d, username=%s", userID, manualUsername)
	// #endregion

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

	additionalGuests := sanitizeAdditionalGuests(req.Guests, req.FirstName, req.LastName)
	hasPrimaryGuestData := buildNormalizedRegistrationFullName(req.FirstName, req.LastName) != ""
	hasAdditionalGuests := len(additionalGuests) > 0

	if !hasPrimaryGuestData && !hasAdditionalGuests {
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

		JSONError(w, http.StatusBadRequest, "invalid request")
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

	if err := google_sheets.AddGuestGroupToSheets(ctx, req.FirstName, req.LastName, age, category, side, userIDPtr, usernamePtr, additionalGuests); err != nil {
		log.Printf("Error adding guest: %v", err)
		JSONError(w, http.StatusInternalServerError, "failed to register")
		return
	}

	for _, key := range registrationCacheKeys(userID, manualUsername) {
		cache.SetMemoryCache(key, true, 30*time.Second)
	}

	setAuthSessionCookie(w, r, userID, manualUsername)

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
	username := req.Username
	userID, username = resolveAuthIdentityFromRequest(r, userID, username, req.InitData)

	if userID == 0 && username == "" {
		JSONError(w, http.StatusBadRequest, "user_id_or_username_required")
		return
	}

	setAuthSessionCookie(w, r, userID, username)

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

// getSeatingInfo возвращает опубликованную рассадку для Mini App
func getSeatingInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status, err := google_sheets.GetSeatingLockStatus(ctx)
	if err != nil {
		log.Printf("Error getting seating info: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	tables, err := google_sheets.GetPublishedSeatingFromSheets(ctx)
	if err != nil {
		log.Printf("Error getting published seating: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	if len(tables) == 0 {
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"visible":      false,
			"published_at": status.LockedAt,
			"tables":       []google_sheets.SeatingTable{},
		})
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"visible":      true,
		"published_at": status.LockedAt,
		"tables":       tables,
	})
}

// getPersonalSeatingInfo возвращает персональную рассадку гостя из опубликованной версии.
func getPersonalSeatingInfo(w http.ResponseWriter, r *http.Request) {
	userID, err := parseOptionalUserID(r.URL.Query().Get("userId"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid_user_id")
		return
	}

	username := google_sheets.NormalizeTelegramUsername(r.URL.Query().Get("username"))
	initData := r.URL.Query().Get("initData")

	resolvedUserID, resolvedUsername, ok := requireResolvedAuthIdentity(w, r, userID, username, initData)
	if !ok {
		return
	}

	ctx := r.Context()

	status, err := google_sheets.GetSeatingLockStatus(ctx)
	if err != nil {
		log.Printf("Error getting personal seating status: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	info, err := google_sheets.GetGuestTableAndNeighborsByIdentifier(ctx, resolvedUserID, resolvedUsername)
	if err != nil {
		log.Printf("Error getting personal seating info: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	if info == nil {
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"visible":      false,
			"published_at": status.LockedAt,
		})
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"visible":      true,
		"published_at": status.LockedAt,
		"table":        info.Table,
		"neighbors":    info.Neighbors,
		"full_name":    info.FullName,
	})
}
