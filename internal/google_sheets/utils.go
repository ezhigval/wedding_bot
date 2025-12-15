package google_sheets

import (
	"context"
	"fmt"
	"log"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

// EnsureRequiredSheets проверяет и создает все необходимые вкладки в Google Sheets
func EnsureRequiredSheets(ctx context.Context) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	spreadsheet, err := GetSpreadsheet(spreadsheetID)
	if err != nil {
		return err
	}

	existingSheets := make(map[string]bool)
	for _, sheet := range spreadsheet.Sheets {
		existingSheets[sheet.Properties.Title] = true
	}

	requiredSheets := map[string]SheetConfig{
		"Кроссвод": {
			Headers: []string{"слово", "описание"},
			DefaultData: [][]interface{}{
				{"СВАДЬБА", "Главное событие дня"},
				{"ТАМАДА", "Ведущий праздника"},
				{"ФАТА", "Головной убор невесты"},
				{"БУКЕТ", "Цветы в руках невесты"},
				{"КОЛЬЦО", "Символ брака"},
				{"ТОРТ", "Сладкое угощение"},
				{"ТОСТ", "Поздравление гостей"},
				{"ТАНЕЦ", "Развлечение на празднике"},
			},
		},
		"Кроссвод_Прогресс": {
			Headers: []string{"user_id", "current_crossword_index", "guessed_words_json", "crossword_start_date"},
		},
		"Wordle": {
			Headers: []string{"слово"},
			DefaultData: [][]interface{}{
				{"ГОСТИ"},
				{"ТАНЕЦ"},
				{"БУКЕТ"},
			},
		},
		"Wordle_Прогресс": {
			Headers: []string{"user_id", "отгаданные_слова"},
		},
		"Wordle_Состояние": {
			Headers: []string{"user_id", "current_word", "attempts", "current_guess", "last_word_date"},
		},
		"Игры": {
			Headers: []string{"user_id", "first_name", "last_name", "total_score", "dragon_score", "flappy_score", "crossword_score", "wordle_score", "rank", "last_updated"},
		},
		"Фото": {
			Headers: []string{"timestamp", "user_id", "username", "full_name", "photo_data"},
		},
	}

	for sheetName, sheetConfig := range requiredSheets {
		if existingSheets[sheetName] {
			continue
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

		// Добавляем заголовки
		headers := make([][]interface{}, 1)
		headers[0] = make([]interface{}, len(sheetConfig.Headers))
		for i, h := range sheetConfig.Headers {
			headers[0][i] = h
		}

		if err := UpdateSheetValues(spreadsheetID, sheetName, "A1", headers); err != nil {
			return fmt.Errorf("ошибка добавления заголовков в %s: %w", sheetName, err)
		}

		// Добавляем дефолтные данные, если они есть
		if len(sheetConfig.DefaultData) > 0 {
			if err := AppendSheetValues(spreadsheetID, sheetName, sheetConfig.DefaultData); err != nil {
				return fmt.Errorf("ошибка добавления дефолтных данных в %s: %w", sheetName, err)
			}
			log.Printf("Создана вкладка '%s' с заголовками и %d дефолтными строками", sheetName, len(sheetConfig.DefaultData))
		} else {
			log.Printf("Создана вкладка '%s' с заголовками", sheetName)
		}
	}

	return nil
}

// SheetConfig представляет конфигурацию листа
type SheetConfig struct {
	Headers     []string
	DefaultData [][]interface{}
}

