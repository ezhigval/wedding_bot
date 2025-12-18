package google_sheets

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"wedding-bot/internal/config"
)

var (
	// GspreadAvailable указывает, доступен ли Google Sheets API
	GspreadAvailable = true
	sheetsService    *sheets.Service
)

// getCredentialsJSON пытается получить bytes для Google credentials.
// Поддерживает два формата: сырой JSON и base64 с JSON.
func getCredentialsJSON() ([]byte, error) {
	candidates := []string{
		config.GoogleSheetsCredentials,
		config.GoogleSheetsCredentialsBase64,
	}

	// Логируем наличие переменных (без значений)
	hasCredentials := config.GoogleSheetsCredentials != ""
	hasBase64 := config.GoogleSheetsCredentialsBase64 != ""
	log.Printf("🔍 Проверка credentials: GOOGLE_SHEETS_CREDENTIALS=%v, GOOGLE_SHEETS_CREDENTIALS_BASE64=%v", hasCredentials, hasBase64)

	// Собираем варианты: сырой, без кавычек, unquote
	var variants []string
	for i, cand := range candidates {
		cand = strings.TrimSpace(cand)
		if cand == "" {
			log.Printf("⚠️ Кандидат %d пустой", i)
			continue
		}
		
		// Логируем длину (первые 50 символов для безопасности)
		preview := cand
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		log.Printf("📝 Кандидат %d: длина=%d, начало=%s", i, len(cand), preview)
		
		variants = append(variants, cand)

		trimmed := strings.Trim(cand, `"'`)
		if trimmed != cand {
			log.Printf("✂️ Обрезаны кавычки у кандидата %d", i)
			variants = append(variants, trimmed)
		}

		if unquoted, err := strconv.Unquote(cand); err == nil {
			log.Printf("🔓 Unquoted кандидат %d", i)
			variants = append(variants, unquoted)
		}
	}

	if len(variants) == 0 {
		return nil, fmt.Errorf("credentials не установлены: обе переменные пустые")
	}

	// Пробуем как JSON или base64 (std/raw/url)
	for i, val := range variants {
		log.Printf("🔄 Попытка %d: проверка как JSON (длина=%d)", i+1, len(val))
		
		if json.Valid([]byte(val)) {
			log.Printf("✅ Найден валидный JSON (попытка %d)", i+1)
			return []byte(val), nil
		}

		decoders := []struct {
			name string
			enc  *base64.Encoding
		}{
			{"StdEncoding", base64.StdEncoding},
			{"RawStdEncoding", base64.RawStdEncoding},
			{"URLEncoding", base64.URLEncoding},
			{"RawURLEncoding", base64.RawURLEncoding},
		}
		
		for _, decoder := range decoders {
			decoded, err := decoder.enc.DecodeString(val)
			if err != nil {
				continue
			}
			log.Printf("🔓 Декодировано через %s (длина=%d)", decoder.name, len(decoded))
			if json.Valid(decoded) {
				log.Printf("✅ Найден валидный JSON после base64 декодирования (%s)", decoder.name)
				return decoded, nil
			}
		}
	}

	return nil, fmt.Errorf("credentials не установлены или имеют неверный формат (ожидается JSON или base64 с JSON). Проверено вариантов: %d", len(variants))
}

// GetGoogleSheetsClient получает клиент Google Sheets
func GetGoogleSheetsClient() (*sheets.Service, error) {
	if !GspreadAvailable {
		return nil, fmt.Errorf("Google Sheets API недоступен")
	}

	// Если сервис уже создан, возвращаем его
	if sheetsService != nil {
		log.Printf("♻️ Используется существующий клиент Google Sheets")
		return sheetsService, nil
	}

	log.Printf("🔧 Создание нового клиента Google Sheets...")
	
	credsJSON, err := getCredentialsJSON()
	if err != nil {
		log.Printf("❌ GOOGLE_SHEETS_CREDENTIALS не установлен или испорчен: %v", err)
		return nil, fmt.Errorf("credentials не установлены: %w", err)
	}

	log.Printf("✅ Credentials JSON получен (длина=%d байт)", len(credsJSON))
	
	// Парсим JSON для логирования email (безопасно)
	var credsMap map[string]interface{}
	if err := json.Unmarshal(credsJSON, &credsMap); err == nil {
		if email, ok := credsMap["client_email"].(string); ok {
			log.Printf("📧 Service account email: %s", email)
		}
		if projectID, ok := credsMap["project_id"].(string); ok {
			log.Printf("🆔 Project ID: %s", projectID)
		}
	}

	creds, err := google.CredentialsFromJSON(
		context.Background(),
		credsJSON,
		"https://www.googleapis.com/auth/spreadsheets",
	)
	if err != nil {
		log.Printf("❌ Ошибка создания credentials из JSON: %v", err)
		return nil, fmt.Errorf("ошибка создания credentials: %w", err)
	}

	log.Printf("✅ Credentials созданы успешно")

	// Создаем сервис
	service, err := sheets.NewService(
		context.Background(),
		option.WithCredentials(creds),
	)
	if err != nil {
		log.Printf("❌ Ошибка создания сервиса Google Sheets: %v", err)
		return nil, fmt.Errorf("ошибка создания сервиса Google Sheets: %w", err)
	}

	log.Printf("✅ Сервис Google Sheets создан успешно")
	sheetsService = service
	return service, nil
}

// GetSpreadsheet получает таблицу по ID
func GetSpreadsheet(spreadsheetID string) (*sheets.Spreadsheet, error) {
	log.Printf("📊 Получение таблицы: %s", spreadsheetID)
	
	service, err := GetGoogleSheetsClient()
	if err != nil {
		log.Printf("❌ Ошибка получения клиента: %v", err)
		return nil, err
	}

	spreadsheet, err := service.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		log.Printf("❌ Ошибка получения таблицы %s: %v", spreadsheetID, err)
		log.Printf("💡 Проверь:")
		log.Printf("   1. Правильность ID таблицы")
		log.Printf("   2. Доступ сервисного аккаунта к таблице (поделись таблицей с email из credentials)")
		log.Printf("   3. Включен ли Google Sheets API в проекте")
		return nil, fmt.Errorf("ошибка получения таблицы: %w", err)
	}

	log.Printf("✅ Таблица получена: %s (листов: %d)", spreadsheet.Properties.Title, len(spreadsheet.Sheets))
	return spreadsheet, nil
}

// EnsureSheetExists проверяет существование листа и создает его если нужно
func EnsureSheetExists(spreadsheetID string, sheetName string) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheet, err := GetSpreadsheet(spreadsheetID)
	if err != nil {
		return err
	}

	// Проверяем, существует ли лист
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == sheetName {
			return nil // Лист уже существует
		}
	}

	// Создаем новый лист
	addSheetRequest := &sheets.AddSheetRequest{
		Properties: &sheets.SheetProperties{
			Title: sheetName,
		},
	}

	batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddSheet: addSheetRequest,
			},
		},
	}

	_, err = service.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Do()
	if err != nil {
		return fmt.Errorf("ошибка создания листа %s: %w", sheetName, err)
	}

	return nil
}

// GetSheetValues получает значения из листа
func GetSheetValues(spreadsheetID string, sheetName string, range_ string) ([][]interface{}, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	readRange := fmt.Sprintf("%s!%s", sheetName, range_)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	return resp.Values, nil
}

// UpdateSheetValues обновляет значения в листе
func UpdateSheetValues(spreadsheetID string, sheetName string, range_ string, values [][]interface{}) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	readRange := fmt.Sprintf("%s!%s", sheetName, range_)
	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err = service.Spreadsheets.Values.Update(
		spreadsheetID,
		readRange,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("ошибка обновления значений: %w", err)
	}

	return nil
}

// AppendSheetValues добавляет значения в конец листа
func AppendSheetValues(spreadsheetID string, sheetName string, values [][]interface{}) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	readRange := fmt.Sprintf("%s!A:Z", sheetName)
	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err = service.Spreadsheets.Values.Append(
		spreadsheetID,
		readRange,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("ошибка добавления значений: %w", err)
	}

	return nil
}

// UpdateCell обновляет одну ячейку
func UpdateCell(spreadsheetID string, sheetName string, row, col int, value interface{}) error {
	// Конвертируем в формат A1 notation
	colLetter := ""
	for col > 0 {
		col--
		colLetter = string(rune('A'+col%26)) + colLetter
		col /= 26
	}
	range_ := fmt.Sprintf("%s%d", colLetter, row)

	values := [][]interface{}{{value}}
	return UpdateSheetValues(spreadsheetID, sheetName, range_, values)
}
