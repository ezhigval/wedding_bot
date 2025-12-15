package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"wedding-bot/internal/config"
)

// ParseInitData парсит initData от Telegram для извлечения user_id
func ParseInitData(initData string) (map[string]interface{}, error) {
	if initData == "" {
		return nil, fmt.Errorf("initData required")
	}

	// Парсим query string с правильным URL decode
	params := make(map[string]string)
	pairs := strings.Split(initData, "&")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key, err1 := url.QueryUnescape(parts[0])
			value, err2 := url.QueryUnescape(parts[1])
			if err1 != nil {
				key = parts[0]
			}
			if err2 != nil {
				value = parts[1]
			}
			params[key] = value
		}
	}

	// Проверяем подпись
	hash := params["hash"]
	if hash == "" {
		return nil, fmt.Errorf("hash not found")
	}

	// Проверяем подпись (упрощенная версия, для полной проверки нужен BOT_TOKEN)
	// Здесь мы просто извлекаем данные

	// Извлекаем user из user JSON
	userJSON := params["user"]
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
			"user":      userJSON,
		}, nil
	}

	// Извлекаем данные из распарсенного JSON
	var userID int
	switch v := userData["id"].(type) {
	case float64:
		userID = int(v)
	case int:
		userID = v
	case int64:
		userID = int(v)
	default:
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

	return map[string]interface{}{
		"userId":    userID,
		"firstName": firstName,
		"lastName":  lastName,
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
	if config.BotToken == "" {
		return false
	}

	// Парсим query string
	pairs := strings.Split(initData, "&")
	var dataCheckString strings.Builder
	hash := ""

	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			if key == "hash" {
				hash = value
			} else {
				if dataCheckString.Len() > 0 {
					dataCheckString.WriteString("\n")
				}
				dataCheckString.WriteString(key)
				dataCheckString.WriteString("=")
				dataCheckString.WriteString(value)
			}
		}
	}

	if hash == "" {
		return false
	}

	// Создаем секретный ключ
	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(config.BotToken))

	// Вычисляем HMAC
	h := hmac.New(sha256.New, secretKey.Sum(nil))
	h.Write([]byte(dataCheckString.String()))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	return calculatedHash == hash
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

	client := &http.Client{Timeout: 5}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("is_user_in_group_chat: error %v", err)
		return false, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("is_user_in_group_chat: getChatMember HTTP %d", resp.StatusCode)
		return false, nil
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, err
	}

	ok, _ := data["ok"].(bool)
	if !ok {
		return false, nil
	}

	result, _ := data["result"].(map[string]interface{})
	status, _ := result["status"].(string)

	// статусы: creator, administrator, member, restricted, left, kicked
	return status == "creator" || status == "administrator" || status == "member", nil
}

