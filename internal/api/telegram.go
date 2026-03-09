package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"wedding-bot/internal/config"
)

func parseInitDataParams(initData string) (map[string]string, map[string]string) {
	decoded := make(map[string]string)
	raw := make(map[string]string)

	pairs := strings.Split(initData, "&")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}

		keyRaw := parts[0]
		valRaw := parts[1]
		raw[keyRaw] = valRaw

		key, err1 := url.QueryUnescape(keyRaw)
		if err1 != nil {
			key = keyRaw
		}
		value, err2 := url.QueryUnescape(valRaw)
		if err2 != nil {
			value = valRaw
		}
		decoded[key] = value
	}

	return decoded, raw
}

func verifyInitDataSignature(decodedParams, rawParams map[string]string, isDebug bool) error {
	if err := verifyTelegramSignature(decodedParams); err != nil {
		// В редких случаях данные могут прийти уже закодированными — пробуем сырые значения
		if errRaw := verifyTelegramSignature(rawParams); errRaw == nil {
			if isDebug {
				log.Printf("⚠️ Подпись initData прошла только с сырыми значениями, проверьте экранирование")
			}
			return nil
		}

		if isDebug {
			log.Printf("⚠️ Ошибка подписи initData: %v", err)
			return err
		}

		return err
	}

	return nil
}

// buildDataCheckString формирует строку для проверки подписи согласно требованиям Telegram
func buildDataCheckString(params map[string]string) string {
	// Ключи сортируются лексикографически, hash исключаем
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "hash" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}

	return b.String()
}

// verifyTelegramSignature валидирует initData согласно документации Telegram WebApp
func verifyTelegramSignature(params map[string]string) error {
	if config.BotToken == "" {
		// В dev среде допускаем отсутствие токена, но явно сигнализируем
		if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1" {
			log.Printf("⚠️ BOT_TOKEN не задан, пропускаем проверку подписи initData (DEBUG)")
			return nil
		}
		return errors.New("bot token not configured")
	}

	hash := params["hash"]
	if hash == "" {
		return errors.New("hash not found")
	}

	dataCheckString := buildDataCheckString(params)

	// secret key = HMAC_SHA256("WebAppData", bot_token)
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(config.BotToken))

	h := hmac.New(sha256.New, secret.Sum(nil))
	h.Write([]byte(dataCheckString))

	calculated := hex.EncodeToString(h.Sum(nil))
	if calculated != hash {
		return fmt.Errorf("invalid hash")
	}

	return nil
}

// ParseInitData парсит initData от Telegram для извлечения user_id
func ParseInitData(initData string) (map[string]interface{}, error) {
	// #region agent log
	log.Printf("[DEBUG ParseInitData] Called with initData length: %d, empty: %v", len(initData), initData == "")
	// #endregion
	if initData == "" {
		return nil, fmt.Errorf("initData required")
	}

	// Парсим query string: собираем сырые пары для подписи и декодированные для данных
	params, rawParams := parseInitDataParams(initData)
	isDebug := os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1"
	// #region agent log
	log.Printf("[DEBUG ParseInitData] Parsed params: has_user=%v, user_value_length=%d, isDebug=%v", params["user"] != "", len(params["user"]), isDebug)
	// #endregion

	// Проверяем подпись
	if err := verifyInitDataSignature(params, rawParams, isDebug); err != nil {
		// #region agent log
		log.Printf("[DEBUG ParseInitData] Signature verification failed: %v", err)
		// #endregion
		if isDebug {
			log.Printf("⚠️ Ошибка подписи initData, продолжаем в DEBUG: %v", err)
		} else {
			return nil, fmt.Errorf("invalid initData signature: %w", err)
		}
	}

	// Извлекаем user из user JSON
	userJSON := params["user"]
	// #region agent log
	log.Printf("[DEBUG ParseInitData] userJSON extracted: length=%d, empty=%v", len(userJSON), userJSON == "")
	// #endregion
	if userJSON == "" {
		return nil, fmt.Errorf("user not found in initData")
	}

	// Парсим user JSON
	var userData map[string]interface{}
	if err := json.Unmarshal([]byte(userJSON), &userData); err != nil {
		log.Printf("Ошибка парсинга user JSON: %v, raw: %s", err, userJSON)
		// Если не удалось распарсить, пробуем упрощенный способ
		userIDStr := extractUserIDFromJSON(userJSON)
		if userIDStr == "" {
			return nil, fmt.Errorf("user_id not found in user: %v", err)
		}
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid user_id: %v", err)
		}
		return map[string]interface{}{
			"userId":    userID,
			"firstName": "",
			"lastName":  "",
			"username":  "",
			"user":      userJSON,
		}, nil
	}

	// Извлекаем данные из распарсенного JSON
	var userID int
	switch v := userData["id"].(type) {
	case float64:
		userID = int(v)
		// #region agent log
		log.Printf("[DEBUG ParseInitData] Extracted userID from float64: %d", userID)
		// #endregion
	case int:
		userID = v
		// #region agent log
		log.Printf("[DEBUG ParseInitData] Extracted userID from int: %d", userID)
		// #endregion
	case int64:
		userID = int(v)
		// #region agent log
		log.Printf("[DEBUG ParseInitData] Extracted userID from int64: %d", userID)
		// #endregion
	default:
		// #region agent log
		keys := make([]string, 0, len(userData))
		for k := range userData {
			keys = append(keys, k)
		}
		log.Printf("[DEBUG ParseInitData] user_id not found or invalid type: %T, value: %v, userData keys: %v", v, v, keys)
		// #endregion
		return nil, fmt.Errorf("user_id not found or invalid type in user: %T", v)
	}

	firstName := ""
	if fn, ok := userData["first_name"].(string); ok {
		firstName = fn
	}

	lastName := ""
	if ln, ok := userData["last_name"].(string); ok {
		lastName = ln
	}

	username := ""
	if un, ok := userData["username"].(string); ok {
		username = strings.TrimSpace(un)
	}

	return map[string]interface{}{
		"userId":    userID,
		"firstName": firstName,
		"lastName":  lastName,
		"username":  username,
		"user":      userJSON,
	}, nil
}

// extractUserIDFromJSON извлекает user_id из JSON строки (упрощенная версия)
func extractUserIDFromJSON(jsonStr string) string {
	// Ищем "id":число в JSON
	idx := strings.Index(jsonStr, `"id":`)
	if idx == -1 {
		return ""
	}

	start := idx + 5 // после "id":
	end := start
	for end < len(jsonStr) && (jsonStr[end] >= '0' && jsonStr[end] <= '9') {
		end++
	}

	if end > start {
		return jsonStr[start:end]
	}

	return ""
}

// VerifyTelegramWebappData проверяет подпись Telegram WebApp данных
func VerifyTelegramWebappData(initData string) bool {
	params, rawParams := parseInitDataParams(initData)

	if verifyTelegramSignature(params) == nil {
		return true
	}

	return verifyTelegramSignature(rawParams) == nil
}

// ResolveUserIDByUsername пытается получить Telegram user_id по username через Bot API.
// Работает, если бот может резолвить этот username в чате.
func ResolveUserIDByUsername(username string) (int, error) {
	u := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(username), "@"))
	if u == "" {
		return 0, fmt.Errorf("username is empty")
	}
	if config.BotToken == "" {
		return 0, fmt.Errorf("bot token not configured")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getChat", config.BotToken)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return 0, err
	}
	q := req.URL.Query()
	q.Set("chat_id", "@"+u)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description"`
		Result      struct {
			ID int64 `json:"id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	if !data.Ok {
		return 0, fmt.Errorf("getChat failed: %s", data.Description)
	}
	if data.Result.ID <= 0 {
		return 0, fmt.Errorf("invalid resolved id")
	}

	return int(data.Result.ID), nil
}

// IsUserInGroupChat проверяет, состоит ли пользователь в общем чате гостей
func IsUserInGroupChat(userID int) (bool, error) {
	if config.BotToken == "" || config.GroupID == "" {
		return false, nil
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/getChatMember", config.BotToken)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	q := req.URL.Query()
	q.Add("chat_id", config.GroupID)
	q.Add("user_id", strconv.Itoa(userID))
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Сетевые ошибки считаем временными: логируем, но не тревожим админов
		log.Printf("is_user_in_group_chat: error %v", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("is_user_in_group_chat: getChatMember HTTP %d", resp.StatusCode)
		// Уведомляем только если проблема требует вмешательства (например, неверный токен/группа или Telegram недоступен)
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode >= 500 {
			notifyAdminsThrottled("group_check_status", fmt.Sprintf("🚨 getChatMember HTTP %d для user %d (проверьте BOT_TOKEN/GROUP_ID)", resp.StatusCode, userID), 15*time.Minute)
		}
		return false, fmt.Errorf("getChatMember status %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		notifyAdminsThrottled("group_check_decode", fmt.Sprintf("🚨 Ошибка парсинга ответа getChatMember для user %d: %v", userID, err), 15*time.Minute)
		return false, err
	}

	ok, _ := data["ok"].(bool)
	if !ok {
		notifyAdminsThrottled("group_check_not_ok", fmt.Sprintf("🚨 Telegram вернул ok=false в getChatMember для user %d (проверьте права бота в группе)", userID), 15*time.Minute)
		return false, fmt.Errorf("telegram ok=false")
	}

	result, _ := data["result"].(map[string]interface{})
	status, _ := result["status"].(string)

	// статусы: creator, administrator, member, restricted, left, kicked
	return status == "creator" || status == "administrator" || status == "member", nil
}
