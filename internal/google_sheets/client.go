package google_sheets

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"golang.org/x/oauth2/google"

	"wedding-bot/internal/config"
)

var (
	// GspreadAvailable указывает, доступен ли Google Sheets API
	GspreadAvailable = true
	sheetsService    *sheets.Service
)

// GetGoogleSheetsClient получает клиент Google Sheets
func GetGoogleSheetsClient() (*sheets.Service, error) {
	if !GspreadAvailable {
		return nil, fmt.Errorf("Google Sheets API недоступен")
	}

	if config.GoogleSheetsCredentials == "" {
		log.Printf("GOOGLE_SHEETS_CREDENTIALS не установлен")
		return nil, fmt.Errorf("credentials не установлены")
	}

	// Если сервис уже создан, возвращаем его
	if sheetsService != nil {
		return sheetsService, nil
	}

	// Парсим JSON credentials из переменной окружения
	credsJSON := []byte(config.GoogleSheetsCredentials)
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

