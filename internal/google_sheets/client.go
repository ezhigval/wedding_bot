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

	// Собираем варианты: сырой, без кавычек, unquote
	var variants []string
	for _, cand := range candidates {
		cand = strings.TrimSpace(cand)
		if cand == "" {
			continue
		}
		variants = append(variants, cand)

		trimmed := strings.Trim(cand, `"'`)
		if trimmed != cand {
			variants = append(variants, trimmed)
		}

		if unquoted, err := strconv.Unquote(cand); err == nil {
			variants = append(variants, unquoted)
		}
	}

	// Пробуем как JSON или base64 (std/raw/url)
	for _, val := range variants {
		if json.Valid([]byte(val)) {
			return []byte(val), nil
		}

		decoders := []*base64.Encoding{
			base64.StdEncoding,
			base64.RawStdEncoding,
			base64.URLEncoding,
			base64.RawURLEncoding,
		}
		for _, enc := range decoders {
			decoded, err := enc.DecodeString(val)
			if err != nil {
				continue
			}
			if json.Valid(decoded) {
				return decoded, nil
			}
		}
	}

	return nil, fmt.Errorf("credentials не установлены или имеют неверный формат (ожидается JSON или base64 с JSON)")
}

// GetGoogleSheetsClient получает клиент Google Sheets
func GetGoogleSheetsClient() (*sheets.Service, error) {
	if !GspreadAvailable {
		return nil, fmt.Errorf("Google Sheets API недоступен")
	}

	// Если сервис уже создан, возвращаем его
	if sheetsService != nil {
		return sheetsService, nil
	}

	credsJSON, err := getCredentialsJSON()
	if err != nil {
		log.Printf("GOOGLE_SHEETS_CREDENTIALS не установлен или испорчен: %v", err)
		return nil, fmt.Errorf("credentials не установлены: %w", err)
	}

	creds, err := google.CredentialsFromJSON(
		context.Background(),
		credsJSON,
		"https://www.googleapis.com/auth/spreadsheets",
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания credentials: %w", err)
	}

	// Создаем сервис
	service, err := sheets.NewService(
		context.Background(),
		option.WithCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сервиса Google Sheets: %w", err)
	}

	sheetsService = service
	return service, nil
}

// GetSpreadsheet получает таблицу по ID
func GetSpreadsheet(spreadsheetID string) (*sheets.Spreadsheet, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheet, err := service.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения таблицы: %w", err)
	}

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
